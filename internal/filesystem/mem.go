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

// AddEntry adds an entry (file/dir) to the memfs at path.
// Note that the parent directory must already exist. So it is
// only expected to be used in NewMemFS().
func (m *MemFS) addEntry(path string, entry MemEntry) error {
	dir := m.memDir

	path, err := filepath.Rel(m.basePath, path)
	if err != nil {
		return err
	}

	opaths := []string{m.basePath}
	paths := strings.Split(path, string(filepath.Separator))
	for _, seg := range paths[:len(paths)-1] {
		opaths = append(opaths, seg)
		found := false
		for _, entry := range dir.getChildren() {
			if entry.Name() == seg {
				found = true
				var ok bool
				dir, ok = entry.(*memDir)
				if !ok {
					return fmt.Errorf("%s is not a dir", filepath.Join(opaths...))
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("%s: %w", filepath.Join(opaths...), fs.ErrNotExist)
		}
	}
	dir.mu.Lock()
	defer dir.mu.Unlock()

	dir.children = append(dir.children, entry)
	return nil
}
