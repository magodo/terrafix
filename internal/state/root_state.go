package state

import (
	"context"
	"fmt"
	"path/filepath"

	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmodule "github.com/hashicorp/terraform-schema/module"
	"github.com/hashicorp/terraform-schema/registry"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	"github.com/magodo/terrafix/internal/filesystem"
	"github.com/magodo/terrafix/internal/terraform/datadir"
)

const languageIDTF = "terraform"

type RootState struct {
	// RootPath is the root module's path
	RootPath string

	CoreVersion *version.Version
	CoreSchema  *schema.BodySchema

	ProviderSchemas map[tfaddr.Provider]*tfschema.ProviderSchema

	ModuleManifest *datadir.ModuleManifest

	// InstalledModules is a map of normalized source addresses from the
	// manifest to the path of the local directory where the module is installed
	InstalledModules InstalledModules

	// ModuleStates includes states of each module, keyed by module path.
	// Especially, the "." key represents the root module.
	// TODO: Shall we use tfmodule.ModuleSourceAddr instead?
	ModuleStates map[string]*ModuleState
}

func NewRootState(tf *tfexec.Terraform, fs *filesystem.FS, path string) (*RootState, error) {
	ctx := context.Background()
	var rootState RootState

	path = filepath.Clean(path)

	rootState.RootPath = path

	tfVersion, _, err := tf.Version(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("terraform version failed: %v", err)
	}
	rootState.CoreVersion = tfVersion

	// Core schema
	coreSchema, err := tfschema.CoreModuleSchemaForVersion(tfVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get core module schema: %v", err)
	}
	rootState.CoreSchema = coreSchema

	// Provider schemas
	providerSchemasJSON, err := tf.ProvidersSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("terraform providers schema failed: %v", err)
	}
	providerSchemas := map[tfaddr.Provider]*tfschema.ProviderSchema{}

	for paddr, providerSchemaJSON := range providerSchemasJSON.Schemas {
		paddr := tfaddr.MustParseProviderSource(paddr)
		providerSchema := tfschema.ProviderSchemaFromJson(providerSchemaJSON, paddr)

		providerSchemas[adjustProviderAddr(paddr)] = providerSchema
	}
	rootState.ProviderSchemas = providerSchemas

	// Module manifest
	mm, err := ParseModuleManifest(path)
	if err != nil {
		return nil, fmt.Errorf("parsing module manifest: %v", err)
	}
	rootState.ModuleManifest = mm
	rootState.InstalledModules = InstalledModulesFromManifest(mm)

	// Add module states
	rootState.ModuleStates = map[string]*ModuleState{}
	if err := rootState.AddModuleState(fs, path); err != nil {
		return nil, fmt.Errorf("add module state for %q: %v", path, err)
	}

	// Collect references
	d := rootState.Decoder()
	for modPath, modState := range rootState.ModuleStates {
		pd, err := d.Path(lang.Path{Path: modPath, LanguageID: languageIDTF})
		if err != nil {
			return nil, fmt.Errorf("failed to new path decoder for %q: %v", modPath, err)
		}

		origins, err := pd.CollectReferenceOrigins()
		if err != nil {
			return nil, fmt.Errorf("failed to collect reference origins for %q: %v", modPath, err)
		}
		modState.OriginRefs = origins

		targets, err := pd.CollectReferenceTargets()
		if err != nil {
			return nil, fmt.Errorf("failed to collect reference targets for %q: %v", modPath, err)
		}
		modState.TargetRefs = targets
	}

	return &rootState, nil
}

func (s *RootState) Decoder() *decoder.Decoder {
	return decoder.NewDecoder(s)
}

var _ tfschema.StateReader = &RootState{}

// DeclaredModuleCalls implements schema.StateReader.
func (s *RootState) DeclaredModuleCalls(modPath string) (map[string]tfmodule.DeclaredModuleCall, error) {
	ms, ok := s.ModuleStates[modPath]
	if !ok {
		return nil, fmt.Errorf("module path %q not found", modPath)
	}

	// Not sure why, but copied from: https://github.com/hashicorp/terraform-ls/blob/abe92f01988de5445556fe1576765cb8f1cb80d9/internal/features/modules/state/module_store.go#L171-L181
	declared := make(map[string]tfmodule.DeclaredModuleCall)
	for _, mc := range ms.Meta.ModuleCalls {
		declared[mc.LocalName] = tfmodule.DeclaredModuleCall{
			LocalName:     mc.LocalName,
			RawSourceAddr: mc.RawSourceAddr,
			SourceAddr:    mc.SourceAddr,
			Version:       mc.Version,
			InputNames:    mc.InputNames,
			RangePtr:      mc.RangePtr,
		}
	}
	return declared, nil
}

// InstalledModulePath implements schema.StateReader.
func (s *RootState) InstalledModulePath(rootPath string, normalizedSource string) (string, bool) {
	v, ok := s.InstalledModules[normalizedSource]
	return v, ok
}

// LocalModuleMeta implements schema.StateReader.
func (s *RootState) LocalModuleMeta(modPath string) (*tfmodule.Meta, error) {
	ms, ok := s.ModuleStates[modPath]
	if !ok {
		return nil, fmt.Errorf("module path %q not found", modPath)
	}
	return &ms.Meta, nil
}

// ProviderSchema implements schema.StateReader.
func (s *RootState) ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error) {
	// TODO: handling vc
	sch, ok := s.ProviderSchemas[adjustProviderAddr(addr)]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", addr)
	}
	return sch, nil
}

// RegistryModuleMeta implements schema.StateReader.
func (s *RootState) RegistryModuleMeta(addr tfaddr.Module, cons version.Constraints) (*registry.ModuleData, error) {
	panic("RegistryModuleMeta unimplemented")
}

var _ decoder.PathReader = &RootState{}

// PathContext implements decoder.PathReader.
// Per: https://github.com/hashicorp/terraform-ls/blob/abe92f01988de5445556fe1576765cb8f1cb80d9/internal/features/modules/decoder/path_reader.go#L67
func (s *RootState) PathContext(path lang.Path) (*decoder.PathContext, error) {
	if path.LanguageID != languageIDTF {
		return nil, fmt.Errorf("unsupported language id: %s", path.LanguageID)
	}
	modState, ok := s.ModuleStates[path.Path]
	if !ok {
		return nil, fmt.Errorf("path %q not found", path.Path)
	}

	schema, err := s.schemaForModule(modState)
	if err != nil {
		return nil, err
	}
	modState.Schema = schema

	pathCtx := &decoder.PathContext{
		Schema:           schema,
		ReferenceOrigins: modState.OriginRefs,
		ReferenceTargets: modState.TargetRefs,
		Files:            modState.Files,
		// Functions:
		// Validators:
	}

	// for _, origin := range modState.OriginRefs {
	// 	if IsModuleFilename(origin.OriginRange().Filename) {
	// 		pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
	// 	}
	// }
	// for _, target := range modState.TargetRefs {
	// 	if target.RangePtr != nil && IsModuleFilename(target.RangePtr.Filename) {
	// 		pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
	// 	} else if target.RangePtr == nil {
	// 		pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
	// 	}
	// }

	return pathCtx, nil

}

// Paths implements decoder.PathReader.
func (s *RootState) Paths(ctx context.Context) []lang.Path {
	var paths []lang.Path
	for path := range s.ModuleStates {
		paths = append(paths, lang.Path{
			Path:       path,
			LanguageID: languageIDTF,
		})
	}
	return paths
}

func (s *RootState) schemaForModule(ms *ModuleState) (*schema.BodySchema, error) {
	sm := tfschema.NewSchemaMerger(s.CoreSchema)
	sm.SetTerraformVersion(s.CoreVersion)
	sm.SetStateReader(s)

	meta := &tfmodule.Meta{
		Path:                 ms.Meta.Path,
		CoreRequirements:     ms.Meta.CoreRequirements,
		ProviderRequirements: ms.Meta.ProviderRequirements,
		ProviderReferences:   ms.Meta.ProviderReferences,
		Variables:            ms.Meta.Variables,
		Filenames:            ms.Meta.Filenames,
		ModuleCalls:          ms.Meta.ModuleCalls,
	}

	return sm.SchemaForModule(meta)
}

// adjustProviderAddr adjusts the provider address, per:
// https://github.com/hashicorp/terraform-ls/blob/abe92f01988de5445556fe1576765cb8f1cb80d9/internal/features/modules/jobs/schema.go#L85
// The reason is that the earlydecoder.LoadModule will generate a legacy provider address for 1st class provider,
// with no explict definition in required_providers.
func adjustProviderAddr(pAddr tfaddr.Provider) tfaddr.Provider {
	if pAddr.IsLegacy() && pAddr.Type == "terraform" {
		pAddr = tfaddr.NewProvider(tfaddr.BuiltInProviderHost, tfaddr.BuiltInProviderNamespace, "terraform")
	} else if pAddr.IsLegacy() {
		pAddr.Namespace = "hashicorp"
	}
	return pAddr
}

func IsModuleFilename(name string) bool {
	return strings.HasSuffix(name, ".tf") ||
		strings.HasSuffix(name, ".tf.json")
}
