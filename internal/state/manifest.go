package state

import (
	"github.com/magodo/terrafix/internal/terraform/datadir"
)

func ParseModuleManifest(modPath string) (*datadir.ModuleManifest, error) {
	manifestPath, ok := datadir.ModuleManifestFilePath(modPath)
	if !ok {
		return nil, nil
	}

	mm, err := datadir.ParseModuleManifestFromFile(manifestPath)
	if err != nil {
		return nil, err
	}
	return mm, nil
}
