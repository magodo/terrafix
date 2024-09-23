package ctrl

import (
	"context"
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

func (ctrl *Controller) FixReferenceOrigins(ctx context.Context) error {
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
		originRangesMap := map[ReqType][]hcl.Range{}
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
					BlockType:   reqType.BlockType,
					BlockName:   reqType.BlockName,
					Version:     0,
					RawContents: [][]byte{},
				}
			}
			req.RawContents = append(req.RawContents, origin.Range.SliceBytes(modState.Files[origin.Range.Filename].Bytes))
			reqs[reqType] = req

			originRanges, ok := originRangesMap[reqType]
			if !ok {
				originRanges = []hcl.Range{}
			}
			originRanges = append(originRanges, origin.Range)
			originRangesMap[reqType] = originRanges

		}

		updatesMap := map[string][]writer.Update{}
		for reqtype, req := range reqs {
			resp, err := ctrl.fixer.FixReferenceOrigins(ctx, req)
			if err != nil {
				return fmt.Errorf("fixer fix reference origins: %v", err)
			}
			originRanges := originRangesMap[reqtype]

			for i, origin := range resp.RawContents {
				originRange := originRanges[i]
				updatesMap[originRange.Filename] = append(updatesMap[originRange.Filename], writer.Update{
					Range:   originRange,
					Content: origin,
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
			fmt.Printf("Updated %s\n\n%s\n", fpath, string(nb))
		}
	}

	return nil
}

func (ctrl *Controller) FixDefinition(ctx context.Context) error {
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
				BlockName:  blk.Labels[0],
				RawContent: blk.Range().SliceBytes(f.Bytes),
			}
			switch blk.Type {
			case "data":
				req.BlockType = fixer.BlockTypeDataSource
			case "resource":
				req.BlockType = fixer.BlockTypeResource
			default:
				panic("unreachable")
			}
			resp, err := ctrl.fixer.FixDefinition(ctx, req)
			if err != nil {
				return fmt.Errorf("fixer fix definition: %v", err)
			}
			updatesMap[filename] = append(updatesMap[filename], writer.Update{
				Range:   blk.Range(),
				Content: resp.RawContent,
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
			fmt.Printf("Updated %s\n\n%s\n", fpath, string(nb))
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
