package state_test

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/magodo/terrafix/internal/filesystem"
	"github.com/magodo/terrafix/internal/state"
	"github.com/magodo/terrafix/internal/terraform/find"
	"github.com/stretchr/testify/require"
)

func TestRootStateDecoder(t *testing.T) {
	rootModPath := "testdata/simple_module"
	tfpath, err := find.FindTF(context.Background(), version.MustConstraints(version.NewConstraint(">=1.0.0")))
	require.NoError(t, err)
	tf, err := tfexec.NewTerraform(rootModPath, tfpath)
	require.NoError(t, err)

	fs, err := filesystem.NewMemFS(rootModPath, nil)
	require.NoError(t, err)

	root, err := state.NewRootState(tf, fs, rootModPath)
	require.NoError(t, err)

	// Two module states are expected
	modRoot, ok := root.ModuleStates[rootModPath]
	require.True(t, ok)

	localModPath := filepath.Join(rootModPath, "local")
	modLocal, ok := root.ModuleStates[localModPath]
	require.True(t, ok)

	// Origin checks
	d := root.Decoder()

	type ReferenceRange struct {
		origin hcl.Range
		target hcl.Range
	}

	var verifyReference = func(t *testing.T, d *decoder.Decoder, path string, expectRef ReferenceRange, origin reference.Origin) {
		require.Equal(t, expectRef.origin, origin.OriginRange())
		targets, err := d.ReferenceTargetsForOriginAtPos(lang.Path{Path: path, LanguageID: "terraform"}, origin.OriginRange().Filename, origin.OriginRange().Start)
		require.NoError(t, err)
		require.Len(t, targets, 1)
		require.Equal(t, expectRef.target, targets[0].Range)
	}

	modRootReferenceRanges := []ReferenceRange{
		{
			origin: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   11,
					Column: 14,
					Byte:   185,
				},
				End: hcl.Pos{
					Line:   11,
					Column: 45,
					Byte:   216,
				},
			},
			target: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   6,
					Column: 3,
					Byte:   82,
				},
				End: hcl.Pos{
					Line:   6,
					Column: 21,
					Byte:   100,
				},
			},
		},
		{
			origin: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   12,
					Column: 14,
					Byte:   230,
				},
				End: hcl.Pos{
					Line:   12,
					Column: 49,
					Byte:   265,
				},
			},
			target: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   7,
					Column: 3,
					Byte:   103,
				},
				End: hcl.Pos{
					Line:   7,
					Column: 26,
					Byte:   126,
				},
			},
		},
		{
			origin: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   14,
					Column: 17,
					Byte:   293,
				},
				End: hcl.Pos{
					Line:   14,
					Column: 46,
					Byte:   322,
				},
			},
			target: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   5,
					Column: 41,
					Byte:   78,
				},
				End: hcl.Pos{
					Line:   5,
					Column: 41,
					Byte:   78,
				},
			},
		},
		{
			origin: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   19,
					Column: 14,
					Byte:   360,
				},
				End: hcl.Pos{
					Line:   19,
					Column: 23,
					Byte:   369,
				},
			},
			target: hcl.Range{
				Filename: "local.tf",
				Start: hcl.Pos{
					Line:   1,
					Column: 1,
					Byte:   0,
				},
				End: hcl.Pos{
					Line:   1,
					Column: 1,
					Byte:   0,
				},
			},
		},
		{
			origin: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   20,
					Column: 3,
					Byte:   372,
				},
				End: hcl.Pos{
					Line:   20,
					Column: 7,
					Byte:   376,
				},
			},
			target: hcl.Range{
				Filename: "local.tf",
				Start: hcl.Pos{
					Line:   1,
					Column: 1,
					Byte:   0,
				},
				End: hcl.Pos{
					Line:   3,
					Column: 2,
					Byte:   35,
				},
			},
		},
		{
			origin: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   20,
					Column: 14,
					Byte:   383,
				},
				End: hcl.Pos{
					Line:   20,
					Column: 45,
					Byte:   414,
				},
			},
			target: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   6,
					Column: 3,
					Byte:   82,
				},
				End: hcl.Pos{
					Line:   6,
					Column: 21,
					Byte:   100,
				},
			},
		},
		{
			origin: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   21,
					Column: 3,
					Byte:   417,
				},
				End: hcl.Pos{
					Line:   21,
					Column: 11,
					Byte:   425,
				},
			},
			target: hcl.Range{
				Filename: "local.tf",
				Start: hcl.Pos{
					Line:   5,
					Column: 1,
					Byte:   37,
				},
				End: hcl.Pos{
					Line:   7,
					Column: 2,
					Byte:   76,
				},
			},
		},
		{
			origin: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   21,
					Column: 14,
					Byte:   428,
				},
				End: hcl.Pos{
					Line:   21,
					Column: 49,
					Byte:   463,
				},
			},
			target: hcl.Range{
				Filename: "main.tf",
				Start: hcl.Pos{
					Line:   7,
					Column: 3,
					Byte:   103,
				},
				End: hcl.Pos{
					Line:   7,
					Column: 26,
					Byte:   126,
				},
			},
		},
	}

	require.Len(t, modRoot.OriginRefs, len(modRootReferenceRanges))
	for i, expectRef := range modRootReferenceRanges {
		t.Run("root-"+strconv.Itoa(i), func(t *testing.T) {
			verifyReference(t, d, rootModPath, expectRef, modRoot.OriginRefs[i])
		})
	}

	modLocalReferenceRanges := []ReferenceRange{
		{
			origin: hcl.Range{
				Filename: "local.tf",
				Start: hcl.Pos{
					Line:   10,
					Column: 14,
					Byte:   134,
				},
				End: hcl.Pos{
					Line:   10,
					Column: 22,
					Byte:   142,
				},
			},
			target: hcl.Range{
				Filename: "local.tf",
				Start: hcl.Pos{
					Line:   1,
					Column: 1,
					Byte:   0,
				},
				End: hcl.Pos{
					Line:   3,
					Column: 2,
					Byte:   35,
				},
			},
		},
		{
			origin: hcl.Range{
				Filename: "local.tf",
				Start: hcl.Pos{
					Line:   11,
					Column: 14,
					Byte:   156,
				},
				End: hcl.Pos{
					Line:   11,
					Column: 26,
					Byte:   168,
				},
			},
			target: hcl.Range{
				Filename: "local.tf",
				Start: hcl.Pos{
					Line:   5,
					Column: 1,
					Byte:   37,
				},
				End: hcl.Pos{
					Line:   7,
					Column: 2,
					Byte:   76,
				},
			},
		},
	}

	require.Len(t, modLocal.OriginRefs, len(modLocalReferenceRanges))
	for i, expectRef := range modLocalReferenceRanges {
		t.Run("local-"+strconv.Itoa(i), func(t *testing.T) {
			verifyReference(t, d, localModPath, expectRef, modLocal.OriginRefs[i])
		})
	}
}

func TestNewRootState_PopulateTFState(t *testing.T) {
	rootModPath := "testdata/nested_modules"
	tfpath, err := find.FindTF(context.Background(), version.MustConstraints(version.NewConstraint(">=1.0.0")))
	require.NoError(t, err)
	tf, err := tfexec.NewTerraform(rootModPath, tfpath)
	require.NoError(t, err)

	fs, err := filesystem.NewMemFS(rootModPath, nil)
	require.NoError(t, err)

	root, err := state.NewRootState(tf, fs, rootModPath)
	require.NoError(t, err)

	mod0 := root.ModuleStates["testdata/nested_modules"]
	require.Len(t, mod0.TFStateResources, 1)
	require.NotNil(t, mod0.TFStateResources["data.azurerm_resource_group.test"])
	mod1 := root.ModuleStates["testdata/nested_modules/module"]
	require.Len(t, mod1.TFStateResources, 2)
	require.NotNil(t, mod1.TFStateResources["azurerm_resource_group.test"])
	require.NotNil(t, mod1.TFStateResources["data.azurerm_resource_group.test"])
	mod2 := root.ModuleStates["testdata/nested_modules/module/module"]
	require.Len(t, mod2.TFStateResources, 2)
	require.NotNil(t, mod2.TFStateResources["azurerm_resource_group.test"])
	require.NotNil(t, mod2.TFStateResources["data.azurerm_resource_group.test"])
}

func TestNewRootState_RemoteModule(t *testing.T) {
	rootModPath := "testdata/remote_module"
	tfpath, err := find.FindTF(context.Background(), version.MustConstraints(version.NewConstraint(">=1.0.0")))
	require.NoError(t, err)
	tf, err := tfexec.NewTerraform(rootModPath, tfpath)
	require.NoError(t, err)

	fs, err := filesystem.NewMemFS(rootModPath, nil)
	require.NoError(t, err)

	_, err = state.NewRootState(tf, fs, rootModPath)
	require.NoError(t, err)
}
