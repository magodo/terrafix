package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// This file contains additional methods for the MemFS and its related types, other than the ones defined in FS.

func (m *MemFile) Write(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.content = b
	m.fileinfo.size = int64(len(b))
	return len(b), nil
}

// NOTE: This is not a general implementation, in that it only supports write to an
//
//		     existing file, otherwise, fs.ErrNotExist will return.
//	      Also, the perm is not used at all.
func (m *MemFS) WriteFile(name string, data []byte, perm fs.FileInfo) error {
	entry, err := m.getEntry(name)
	if err != nil {
		return err
	}
	f, ok := entry.(*MemFile)
	if !ok {
		return fs.ErrNotExist
	}
	_, err = f.Write(data)
	return err
}

// AddEntry adds an entry (file/dir) to the memfs at path.
// Note that the parent directory must already exist.
func (m *MemFS) addEntry(path string, entry MemEntry) error {
	dir := m.MemDir

	path, err := filepath.Rel(m.basePath, path)
	if err != nil {
		return err
	}

	opaths := []string{m.basePath}
	paths := strings.Split(path, string(filepath.Separator))
	for _, seg := range paths[:len(paths)-1] {
		opaths = append(opaths, seg)
		found := false
		for _, entry := range dir.GetChildren() {
			if entry.Name() == seg {
				found = true
				var ok bool
				dir, ok = entry.(*MemDir)
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

type FilterFunc func(fs.DirEntry) bool

func NewMemFS(path string, filter FilterFunc) (*MemFS, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("initial path can't be a file")
	}

	memfs := MemFS{
		basePath: path,
		MemDir: &MemDir{
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

		if !filter(d) {
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
			entry = &MemDir{
				fileinfo: NewFileInfo(info),
			}
		} else {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			entry = &MemFile{
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
