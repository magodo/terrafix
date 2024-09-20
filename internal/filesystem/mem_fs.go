package filesystem

import (
	"io"
	"io/fs"
	"path/filepath"
	"sync"
	"time"
)

var _ FS = &MemFS{}

type MemFS struct {
	basePath string
	*MemDir
}

func (m *MemFS) getEntry(name string) (MemEntry, error) {
	name, err := filepath.Rel(m.basePath, name)
	if err != nil {
		return nil, err
	}
	segs := filepath.SplitList(name)
	var entry MemEntry = m.MemDir
	for _, seg := range segs {
		dir, ok := entry.(*MemDir)
		if !ok {
			return nil, fs.ErrNotExist
		}
		children := dir.GetChildren()
		ok = false
		for _, child := range children {
			if child.Name() == seg {
				entry = child
				ok = true
				break
			}
		}
		if !ok {
			return nil, fs.ErrNotExist
		}
	}
	return entry, nil
}

func (m *MemFS) Open(name string) (fs.File, error) {
	entry, err := m.getEntry(name)
	if err != nil {
		return nil, err
	}
	file, ok := entry.(*MemFile)
	if !ok {
		return nil, fs.ErrNotExist
	}
	return file, nil
}

func (m *MemFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entry, err := m.getEntry(name)
	if err != nil {
		return nil, err
	}
	dir, ok := entry.(*MemDir)
	if !ok {
		return nil, fs.ErrNotExist
	}
	var out []fs.DirEntry
	for _, child := range dir.GetChildren() {
		out = append(out, child)
	}
	return out, nil
}

func (m *MemFS) ReadFile(name string) ([]byte, error) {
	entry, err := m.getEntry(name)
	if err != nil {
		return nil, err
	}
	f, ok := entry.(*MemFile)
	if !ok {
		return nil, fs.ErrNotExist
	}
	return io.ReadAll(f)
}

func (m *MemFS) Stat(name string) (fs.FileInfo, error) {
	entry, err := m.getEntry(name)
	if err != nil {
		return nil, err
	}
	return entry.Info()
}

type MemEntry interface {
	isMemDirEntry()
	fs.DirEntry
}

type MemDir struct {
	fileinfo FileInfo
	children []MemEntry
	mu       sync.RWMutex
}

func (*MemDir) isMemDirEntry() {}

func (m *MemDir) Name() string {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.name
}

func (m *MemDir) IsDir() bool {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.isDir
}

func (m *MemDir) Type() fs.FileMode {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.mode
}

func (m *MemDir) Info() (fs.FileInfo, error) {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	// return the info when this entry is read
	return m.fileinfo, nil
}

func (m *MemDir) GetChildren() []MemEntry {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()

	var out []MemEntry
	for _, v := range m.children {
		out = append(out, v)
	}
	return out
}

type MemFile struct {
	fileinfo FileInfo
	content  []byte
	mu       sync.RWMutex
}

func (*MemFile) isMemDirEntry() {}

func (m *MemFile) Name() string {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.name
}

func (m *MemFile) IsDir() bool {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.isDir
}

func (m *MemFile) Type() fs.FileMode {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.mode
}

func (m *MemFile) Info() (fs.FileInfo, error) {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	// return the info when this entry is read
	return m.fileinfo, nil
}

func (m *MemFile) Stat() (fs.FileInfo, error) {
	return m.Info()
}

func (m *MemFile) Read(b []byte) (int, error) {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()

	return copy(b, m.content), nil
}

func (m *MemFile) Close() error {
	return nil
}

type FileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func NewFileInfo(info fs.FileInfo) FileInfo {
	return FileInfo {
		name:    info.Name(),
		size:    info.Size(),
		mode:    info.Mode(),
		modTime: info.ModTime(),
		isDir:   info.IsDir(),
	}
}

func (f FileInfo) Name() string {
	return f.name
}

func (f FileInfo) Size() int64 {
	return f.size
}

func (f FileInfo) Mode() fs.FileMode {
	return f.mode
}

func (f FileInfo) ModTime() time.Time {
	return f.modTime
}

func (f FileInfo) IsDir() bool {
	return f.isDir
}

func (f FileInfo) Sys() any {
	return nil
}
