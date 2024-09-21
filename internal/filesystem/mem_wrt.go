package filesystem

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// This file contains additional write-related methods for the MemFS and its related types

func (m *memFile) Write(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	eN := len(b)
	m.content = make([]byte, eN)
	aN := copy(m.content, b)
	if aN != eN {
		return aN, io.ErrShortWrite
	}
	m.fileinfo.size = int64(aN)
	return aN, nil
}

// NOTE: This is not a general implementation, in that it only supports write to an
//
//		 existing file, otherwise, fs.ErrNotExist will return.
//	     Also, the perm is not used at all.
func (m *MemFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	entry, err := m.getEntry(name)
	if err != nil {
		return err
	}
	f, ok := entry.(*memFile)
	if !ok {
		return fs.ErrNotExist
	}
	_, err = f.Write(data)
	return err
}

// WriteToOS writes the whole FS to the target path.
// If the path is nil, it writes to the original path that used to build the MemFS.
func (m *MemFS) WriteToOS(path *string) error {
	var p string
	if path == nil {
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}
		p = pwd
	} else {
		p = *path
	}
	return fs.WalkDir(m, m.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		ep := filepath.Join(p, path)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(ep, info.Mode())
		} else {
			b, err := m.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(ep, b, info.Mode())
		}
	})
}
