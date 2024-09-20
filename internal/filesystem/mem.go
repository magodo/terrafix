package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func tfFilter(d fs.DirEntry) bool {
	// Skip any hidden file/dir
	if strings.HasPrefix(d.Name(), ".") {
		return false
	}
	// Allows directory
	if d.IsDir() {
		return true
	}
	// Allows .tf files
	return strings.HasSuffix(d.Name(), ".tf")
}

func NewMemFS(path string) (*MemFS, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("initial path can't be a file")
	}

	memfs := MemFS{
		basePath: path,
		memDir: &memDir{
			fileinfo: NewFileInfo(info),
		},
	}

	if err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == p {
			return nil
		}

		if !tfFilter(d) {
			if d.IsDir() {
				return fs.SkipDir
			} else {
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		var entry MemEntry
		if d.IsDir() {
			entry = &memDir{
				fileinfo: NewFileInfo(info),
			}
		} else {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			entry = &memFile{
				fileinfo: NewFileInfo(info),
				content:  b,
			}
		}
		return memfs.addEntry(p, entry)
	}); err != nil {
		return nil, err
	}
	return &memfs, nil
}
