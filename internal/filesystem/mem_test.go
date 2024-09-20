package filesystem_test

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/magodo/terrafix/internal/filesystem"
	"github.com/stretchr/testify/require"
)

func TestMemFS(t *testing.T) {
	memfs, err := filesystem.NewMemFS("testdata", func(d fs.DirEntry) bool {
		// Skip any hidden file/dir
		if strings.HasPrefix(d.Name(), ".") {
			return false
		}
		if d.IsDir() {
			return true
		}
		return strings.HasSuffix(d.Name(), ".tf")
	})
	require.NoError(t, err)

	children := memfs.GetChildren()
	require.Len(t, children, 2)
}
