package state

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/earlydecoder"
	tfmodule "github.com/hashicorp/terraform-schema/module"
	"github.com/magodo/terrafix/internal/filesystem"
)

type ModuleState struct {
	SourceAddr tfmodule.ModuleSourceAddr

	Meta tfmodule.Meta

	Files map[string]*hcl.File

	Schema *schema.BodySchema

	OriginRefs reference.Origins
	TargetRefs reference.Targets
}

func (s *RootState) AddModuleState(fs filesystem.FS, modPath string) error {
	state := ModuleState{
		SourceAddr: tfmodule.LocalSourceAddr(modPath),
	}

	files := map[string]*hcl.File{}
	es, err := fs.ReadDir(modPath)
	if err != nil {
		return fmt.Errorf("reading dir %q: %v", modPath, err)
	}
	for _, e := range es {
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".tf") {
			fpath := filepath.Join(modPath, e.Name())
			b, err := fs.ReadFile(fpath)
			if err != nil {
				return fmt.Errorf("reading %q: %v", fpath, err)
			}
			f, diags := hclsyntax.ParseConfig(b, e.Name(), hcl.InitialPos)
			if diags.HasErrors() {
				return fmt.Errorf("HCL parse %q: %v", fpath, diags.Error())
			}
			files[e.Name()] = f
		}
	}
	state.Files = files

	meta, diags := earlydecoder.LoadModule(modPath, files)
	if diags.HasErrors() {
		return fmt.Errorf("earlydecoder load module %q: %v", modPath, diags.Error())
	}
	state.Meta = *meta

	// Add the the partially built module state into the root state.
	// The Origin/Target Refs will be updated once all the modules are added.
	s.ModuleStates[modPath] = &state

	// Recursively add module states.
	// Based on: https://github.com/hashicorp/terraform-ls/blob/abe92f01988de5445556fe1576765cb8f1cb80d9/internal/features/modules/events.go#L177
	declared, err := s.DeclaredModuleCalls(modPath)
	if err != nil {
		return fmt.Errorf("getting declared module calls for %q failed: %v", modPath, err)
	}
	var errs *multierror.Error
	for _, mc := range declared {
		var mcPath string
		switch source := mc.SourceAddr.(type) {
		// For local module sources, we can construct the path directly from the configuration
		case tfmodule.LocalSourceAddr:
			mcPath = filepath.Join(modPath, filepath.FromSlash(source.String()))

		// For registry modules, we need to find the local installation path (if installed)
		case tfaddr.Module:
			// installedDir, ok := s.InstalledModulePath(modPath, source.String())
			// if !ok {
			// 	continue
			// }
			// mcPath = filepath.Join(modPath, filepath.FromSlash(installedDir))

			// Only local module is taken into consideration as it is mutable
			continue

		// For other remote modules, we need to find the local installation path (if installed)
		case tfmodule.RemoteSourceAddr:
			// installedDir, ok := s.InstalledModulePath(modPath, source.String())
			// if !ok {
			// 	continue
			// }
			// mcPath = filepath.Join(modPath, filepath.FromSlash(installedDir))

			// Only local module is taken into consideration as it is mutable
			continue

		default:
			// Unknown source address, we can't resolve the path
			continue
		}

		fi, err := fs.Stat(mcPath)
		if err != nil || !fi.IsDir() {
			multierror.Append(errs, err)
			continue
		}

		if _, ok := s.ModuleStates[mcPath]; ok {
			continue
		}

		if err := s.AddModuleState(fs, mcPath); err != nil {
			multierror.Append(errs, fmt.Errorf("add module state for %q: %v", mcPath, err))
		}
	}

	return errs.ErrorOrNil()
}
