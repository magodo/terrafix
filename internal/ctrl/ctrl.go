package ctrl

import (
	"fmt"
	"maps"

	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	"github.com/magodo/terrafix/internal/fixer"
	"github.com/magodo/terrafix/internal/state"
)

type Controller struct {
	tf        *tfexec.Terraform
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
	rootState, err := state.NewRootState(ctrl.tf, ctrl.path)
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
			//fmt.Printf("Origin: %s (%s)\n", origin.Address(), origin.Range)
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
					ReferenceOrigins: []fixer.ReferenceOrigin{},
				}
			}
			req.ReferenceOrigins = append(req.ReferenceOrigins, fixer.ReferenceOrigin{
				Addr:    origin.Addr,
				Range:   origin.Range,
				Content: origin.Range.SliceBytes(modState.Files[origin.Range.Filename].Bytes),
			})
			reqs[reqType] = req
		}

		var updatedOrigins []fixer.ReferenceOrigin
		for _, req := range reqs {
			resp := ctrl.fixer.FixReferenceOrigins(req)
			for _, origin := range resp.ReferenceOrigins {
				updatedOrigins = append(updatedOrigins, origin)
			}
		}

		for _, origin := range updatedOrigins {
			fmt.Printf("Change range: %s, content: %s\n", origin.Range, string(origin.Content))
		}
	}
	return nil
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

		switch blk.Type {
		case "resource":
			if len(blk.Labels) != 2 {
				return nil, fmt.Errorf("invalid resource definition at %s: label length is not 2", blk.DefRange)
			}
			// The target doesn't belong to the interested provider
			if _, ok := ctrl.psch.Resources[blk.Labels[0]]; !ok {
				continue
			}
		case "data":
			if len(blk.Labels) != 2 {
				return nil, fmt.Errorf("invalid data source definition at %s: label length is not 2", blk.DefRange)
			}
			// The target doesn't belong to the interested provider
			if _, ok := ctrl.psch.DataSources[blk.Labels[0]]; !ok {
				continue
			}
		default:
			// Ignore reference origins targeting to non-resource/datasource
			continue
		}

		out = append(out, origin)
	}
	return out, nil
}
