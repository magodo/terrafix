package filesystem_test

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/magodo/terrafix/internal/filesystem"
	"github.com/stretchr/testify/require"
)

func TestMemFS(t *testing.T) {
	memfs, err := filesystem.NewMemFS("testdata", nil)
	require.NoError(t, err)

	///////////////////////
	// Test memfs methods

	// .ReadFile
	eb := []byte("locals {}\n")
	b2, err := memfs.ReadFile("testdata/a.tf")
	require.NoError(t, err)
	require.Equal(t, eb, b2)

	b3, err := memfs.ReadFile("./testdata/a.tf")
	require.NoError(t, err)
	require.Equal(t, eb, b3)

	b4, err := memfs.ReadFile("./testdata/../testdata/a.tf")
	require.NoError(t, err)
	require.Equal(t, eb, b4)

	// .ReadDir
	es, err := memfs.ReadDir("testdata")
	require.NoError(t, err)
	require.Len(t, es, 2)
	require.Equal(t, "a.tf", es[0].Name())
	require.Equal(t, "module", es[1].Name())

	// .State
	info, err := memfs.Stat("testdata")
	require.NoError(t, err)
	require.Equal(t, "testdata", info.Name())

	info, err = memfs.Stat("testdata/a.tf")
	require.NoError(t, err)
	require.Equal(t, "a.tf", info.Name())

	// .Open
	f, err := memfs.Open("testdata/a.tf")
	require.NoError(t, err)
	b, err := io.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, eb, b)
	require.NoError(t, f.Close())

	///////////////////////
	// Change & write
	newContent := bytes.Repeat([]byte("a"), 100)
	require.NoError(t, memfs.WriteFile("testdata/a.tf", newContent, 0))
	b, err = memfs.ReadFile("testdata/a.tf")
	require.NoError(t, err)
	require.Equal(t, newContent, b)

	// Write to a tempdir on OS
	tmpdir := t.TempDir()
	require.NoError(t, memfs.Write(&tmpdir))

	// Check the memfs created from this new tempdir is the same as the prior one,
	// except the baseDir and the modtime
	newMemfs, err := filesystem.NewMemFS(filepath.Join(tmpdir, "testdata"), nil)
	require.NoError(t, err)

	info, err = newMemfs.Stat(filepath.Join(tmpdir, "testdata"))
	require.NoError(t, err)
	info, err = newMemfs.Stat(filepath.Join(tmpdir, "testdata/a.tf"))
	require.NoError(t, err)
	info, err = newMemfs.Stat(filepath.Join(tmpdir, "testdata/module"))
	require.NoError(t, err)
	info, err = newMemfs.Stat(filepath.Join(tmpdir, "testdata/module/b.tf"))
	require.NoError(t, err)

	b, err = newMemfs.ReadFile(filepath.Join(tmpdir, "testdata/a.tf"))
	require.NoError(t, err)
	require.Equal(t, newContent, b)

	b, err = newMemfs.ReadFile(filepath.Join(tmpdir, "testdata/module/b.tf"))
	require.NoError(t, err)
	require.Equal(t, eb, b)
}
