package ctrl_test

import (
	"context"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/magodo/terrafix/internal/ctrl"
	"github.com/magodo/terrafix/internal/fixer"
	"github.com/magodo/terrafix/internal/terraform/find"
	"github.com/stretchr/testify/require"
)

type FixDefinitionChecker func(t *testing.T, req fixer.FixDefinitionRequest)
type FixReferenceOriginsChecker func(t *testing.T, req fixer.FixReferenceOriginsRequest)

type TestFixer struct {
	t *testing.T
	FixDefinitionChecker
	FixReferenceOriginsChecker
}

var _ fixer.Fixer = &TestFixer{}

func (d *TestFixer) FixDefinition(_ context.Context, req fixer.FixDefinitionRequest) (*fixer.FixDefinitionResponse, error) {
	d.FixDefinitionChecker(d.t, req)
	return &fixer.FixDefinitionResponse{
		RawContent: req.RawContent,
	}, nil
}

func (d *TestFixer) FixReferenceOrigins(_ context.Context, req fixer.FixReferenceOriginsRequest) (*fixer.FixReferenceOriginsResponse, error) {
	d.FixReferenceOriginsChecker(d.t, req)
	return &fixer.FixReferenceOriginsResponse{RawContents: req.RawContents}, nil
}

func TestCtrl(t *testing.T) {
	rootModPath := "testdata/module"
	tfpath, err := find.FindTF(context.Background(), version.MustConstraints(version.NewConstraint(">=1.0.0")))
	require.NoError(t, err)
	tf, err := tfexec.NewTerraform(rootModPath, tfpath)
	require.NoError(t, err)

	var defN, refN int
	fx := &TestFixer{
		t: t,
		FixDefinitionChecker: func(t *testing.T, req fixer.FixDefinitionRequest) {
			if req.BlockType == "resource" && req.BlockName == "azurerm_container_registry" {
				require.True(t, req.Version > 0)
			}
			defN += 1
		},
		FixReferenceOriginsChecker: func(t *testing.T, req fixer.FixReferenceOriginsRequest) {
			if req.BlockType == "resource" && req.BlockName == "azurerm_container_registry" {
				require.True(t, req.Version > 0)
			}
			for range req.RawContents {
				refN += 1
			}
		},
	}

	ctrl, err := ctrl.NewController(ctrl.Option{
		Path:         rootModPath,
		ProviderAddr: tfaddr.MustParseProviderSource("registry.terraform.io/hashicorp/azurerm"),
		TF:           tf,
		Fixer:        fx,
	})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, ctrl.FixReferenceOrigins(ctx))
	require.NoError(t, ctrl.UpdateRootState())
	require.NoError(t, ctrl.FixDefinition(ctx))
	require.Equal(t, 6, defN)
	require.Equal(t, 12, refN)
}
