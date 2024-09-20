package ctrl

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"

	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	"github.com/magodo/terrafix/internal/filesystem"
	"github.com/magodo/terrafix/internal/fixer"
	"github.com/magodo/terrafix/internal/state"
	"github.com/magodo/terrafix/internal/writer"
)

type Controller struct {
	tf        *tfexec.Terraform
	fs        *filesystem.MemFS
	psch      *tfschema.ProviderSchema
	path      string
	rootState *state.RootState
	fixer     fixer.Fixer
}

func NewController(tf *tfexec.Terraform, path string, paddr string, fixer fixer.Fixer) (*Controller, error) {
	ctrl := Controller{
		tf:    tf,
		path:  path,
		fixer: fixer,
	}

	fs, err := filesystem.NewMemFS(path)
	if err != nil {
		return nil, fmt.Errorf("error new memory filesystem: %v", err)
	}
	ctrl.fs = fs

	if err := ctrl.UpdateRootState(); err != nil {
		return nil, err
	}

	psch, ok := ctrl.rootState.ProviderSchemas[tfaddr.MustParseProviderSource(paddr)]
	if !ok {
		possibles := []string{}
		for v := range maps.Keys(ctrl.rootState.ProviderSchemas) {
			possibles = append(possibles, v.String())
		}
		return nil, fmt.Errorf("no provider schema defined for %s, possible values include %v", paddr, possibles)
	}
	ctrl.psch = psch

	return &ctrl, nil
}

func (ctrl *Controller) UpdateRootState() error {
	rootState, err := state.NewRootState(ctrl.tf, ctrl.fs, ctrl.path)
	if err != nil {
		return err
	}
	ctrl.rootState = rootState
	return nil
}

func (ctrl *Controller) FixReferenceOrigins() error {
	for modPath, modState := range ctrl.rootState.ModuleStates {
		origins, err := ctrl.filterOriginRefsForMod(modPath, modState)
		if err != nil {
			return fmt.Errorf("finding reference targets from origins, for module %s: %v", modPath, err)
		}

		type ReqType struct {
			BlockType fixer.BlockType
			BlockName string
		}

		// Combine origins belong to the same targeting to the same resource/data source into one request
		reqs := map[ReqType]fixer.FixReferenceOriginsRequest{}
		for _, origin := range origins {
			reqType := ReqType{
				BlockType: fixer.BlockTypeResource,
			}
			if origin.Addr[0].String() == "data" {
				reqType.BlockType = fixer.BlockTypeDataSource
				reqType.BlockName = origin.Addr[1].String()
			} else {
				reqType.BlockName = origin.Addr[0].String()
			}

			req, ok := reqs[reqType]
			if !ok {
				req = fixer.FixReferenceOriginsRequest{
					BlockType:        reqType.BlockType,
					BlockName:        reqType.BlockName,
					Version:          0,
					ReferenceOrigins: []fixer.HCLContent{},
				}
			}
			req.ReferenceOrigins = append(req.ReferenceOrigins, fixer.HCLContent{
				Range:      origin.Range,
				RawContent: origin.Range.SliceBytes(modState.Files[origin.Range.Filename].Bytes),
			})
			reqs[reqType] = req
		}

		updatesMap := map[string][]writer.Update{}
		for _, req := range reqs {
			resp := ctrl.fixer.FixReferenceOrigins(req)
			if resp.Error != nil {
				return errors.New(*resp.Error)
			}
			for _, origin := range resp.ReferenceOrigins {
				updatesMap[origin.Range.Filename] = append(updatesMap[origin.Range.Filename], writer.Update{
					Range:   origin.Range,
					Content: origin.RawContent,
				})
			}
		}

		for filename, updates := range updatesMap {
			fpath := filepath.Join(modPath, filename)
			b, err := ctrl.fs.ReadFile(fpath)
			if err != nil {
				return fmt.Errorf("reading %s: %v", fpath, err)
			}
			nb, err := writer.UpdateContent(b, updates)
			if err != nil {
				return fmt.Errorf("failed to update content for %s: %v", fpath, err)
			}
			if err := ctrl.fs.WriteFile(fpath, nb, 0644); err != nil {
				return fmt.Errorf("writing back the new content: %v", err)
			}
			//fmt.Printf("Updated %s\n\n%s\n", fpath, string(nb))
		}
	}

	return nil
}

func (ctrl *Controller) FixDefinition() error {
	for modPath, modState := range ctrl.rootState.ModuleStates {
		blks, err := ctrl.filterDefinitionForMod(modState)
		if err != nil {
			return fmt.Errorf("finding definition blocks, for module %s: %v", modPath, err)
		}

		updatesMap := map[string][]writer.Update{}
		for _, blk := range blks {
			filename := blk.Range().Filename
			f := modState.Files[filename]
			req := fixer.FixDefinitionRequest{
				BlockName: blk.Labels[0],
				Definition: fixer.HCLContent{
					RawContent: blk.Range().SliceBytes(f.Bytes),
					Range:      blk.Range(),
				},
			}
			switch blk.Type {
			case "data":
				req.BlockType = fixer.BlockTypeDataSource
			case "resource":
				req.BlockType = fixer.BlockTypeResource
			default:
				panic("unreachable")
			}
			resp := ctrl.fixer.FixDefinition(req)
			if resp.Error != nil {
				return errors.New(*resp.Error)
			}
			updatesMap[filename] = append(updatesMap[filename], writer.Update{
				Range:   resp.Definition.Range,
				Content: resp.Definition.RawContent,
			})
		}

		for filename, updates := range updatesMap {
			fpath := filepath.Join(modPath, filename)
			b, err := ctrl.fs.ReadFile(fpath)
			if err != nil {
				return fmt.Errorf("reading %s: %v", fpath, err)
			}
			nb, err := writer.UpdateContent(b, updates)
			if err != nil {
				return fmt.Errorf("failed to update content for %s: %v", fpath, err)
			}
			if err := ctrl.fs.WriteFile(fpath, nb, 0644); err != nil {
				return fmt.Errorf("writing back the new content: %v", err)
			}
			//fmt.Printf("Updated %s\n\n%s\n", fpath, string(nb))
		}
	}

	return nil
}

// filterDefinitionForMod filters the module's resource/data source definitions only if it belongs to the
// interested provider.
func (ctrl *Controller) filterDefinitionForMod(modState *state.ModuleState) ([]*hclsyntax.Block, error) {
	var blks []*hclsyntax.Block
	for _, f := range modState.Files {
		body := f.Body.(*hclsyntax.Body)
		for _, blk := range body.Blocks {
			ok, err := ctrl.filterBlock(blk.AsHCLBlock())
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			blks = append(blks, blk)
		}
	}
	return blks, nil
}

// filterOriginRefsForMod filters the module's reference origins only if its target belongs to a
// resource/datasource that is defined in the interested provider.
//
// Note that this only handles local reference origins, but omit direct/path origins,
// as we are only interested in the former.
func (ctrl *Controller) filterOriginRefsForMod(modPath string, modState *state.ModuleState) ([]reference.LocalOrigin, error) {
	d := ctrl.rootState.Decoder()
	var out []reference.LocalOrigin
	for _, origin := range modState.OriginRefs {
		origin, ok := origin.(reference.LocalOrigin)
		if !ok {
			continue
		}
		targets, err := d.ReferenceTargetsForOriginAtPos(lang.Path{Path: modPath, LanguageID: "terraform"}, origin.OriginRange().Filename, origin.OriginRange().Start)
		if err != nil {
			return nil, err
		}
		if len(targets) == 0 {
			continue
		}
		if len(targets) > 1 {
			return nil, fmt.Errorf("unexpected multiple targets for origin %s (%s)", origin.Addr, origin.Range)
		}
		tgt := targets[0]

		// Filter the origin only if its target belongs to a resource/data source that is defined by the interested provider
		if tgt.Path.Path != modPath {
			// This shouldn't happen as we are processing reference local origins only, whose target should be within the same module.
			return nil, fmt.Errorf("unexpected target path, expect=%s, got=%s", modPath, tgt.Path.Path)
		}
		f := modState.Files[tgt.Range.Filename]
		blk := f.OutermostBlockAtPos(tgt.Range.Start)
		ok, err = ctrl.filterBlock(blk)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		out = append(out, origin)
	}
	return out, nil
}

// filterBlock tells whether a (top-level) block is a resource/data source, belongs to the interested provider.
func (ctrl *Controller) filterBlock(blk *hcl.Block) (bool, error) {
	switch blk.Type {
	case "resource":
		if len(blk.Labels) != 2 {
			return false, fmt.Errorf("invalid resource definition at %s: label length is not 2", blk.DefRange)
		}
		// The target doesn't belong to the interested provider
		_, ok := ctrl.psch.Resources[blk.Labels[0]]
		return ok, nil
	case "data":
		if len(blk.Labels) != 2 {
			return false, fmt.Errorf("invalid data source definition at %s: label length is not 2", blk.DefRange)
		}
		// The target doesn't belong to the interested provider
		_, ok := ctrl.psch.DataSources[blk.Labels[0]]
		return ok, nil
	default:
		// Ignore reference origins targeting to non-resource/datasource
		return false, nil
	}
}
