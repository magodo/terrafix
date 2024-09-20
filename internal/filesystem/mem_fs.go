package filesystem

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var _ FS = &MemFS{}

type MemFS struct {
	basePath string
	*memDir
}

func (m *MemFS) getEntry(name string) (MemEntry, error) {
	name, err := filepath.Rel(m.basePath, name)
	if err != nil {
		return nil, err
	}

	if name == "." {
		return m.memDir, nil
	}

	opaths := []string{m.basePath}
	paths := strings.Split(name, string(filepath.Separator))
	var entry MemEntry = m.memDir
	for _, path := range paths {
		opaths = append(opaths, path)
		dir, ok := entry.(*memDir)
		if !ok {
			return nil, fmt.Errorf("%s is not a dir", filepath.Join(opaths...))
		}
		children := dir.getChildren()
		ok = false
		for _, child := range children {
			if child.Name() == path {
				entry = child
				ok = true
				break
			}
		}
		if !ok {
			return nil, fmt.Errorf("%s: %w", filepath.Join(opaths...), fs.ErrNotExist)
		}
	}
	return entry, nil
}

func (m *MemFS) Open(name string) (fs.File, error) {
	entry, err := m.getEntry(name)
	if err != nil {
		return nil, err
	}
	file, ok := entry.(*memFile)
	if !ok {
		return nil, fs.ErrNotExist
	}
	file.resetPtr()
	return file, nil
}

func (m *MemFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entry, err := m.getEntry(name)
	if err != nil {
		return nil, err
	}
	dir, ok := entry.(*memDir)
	if !ok {
		return nil, fs.ErrNotExist
	}
	var out []fs.DirEntry
	for _, child := range dir.getChildren() {
		out = append(out, child)
	}
	return out, nil
}

func (m *MemFS) ReadFile(name string) ([]byte, error) {
	entry, err := m.getEntry(name)
	if err != nil {
		return nil, err
	}
	f, ok := entry.(*memFile)
	if !ok {
		return nil, fs.ErrNotExist
	}

	f.resetPtr()
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

type memDir struct {
	fileinfo FileInfo
	children []MemEntry
	mu       sync.RWMutex
}

func (*memDir) isMemDirEntry() {}

func (m *memDir) Name() string {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.name
}

func (m *memDir) IsDir() bool {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.isDir
}

func (m *memDir) Type() fs.FileMode {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.mode
}

func (m *memDir) Info() (fs.FileInfo, error) {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	// return the info when this entry is read
	return m.fileinfo, nil
}

func (m *memDir) getChildren() []MemEntry {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()

	var out []MemEntry
	for _, v := range m.children {
		out = append(out, v)
	}
	return out
}

type memFile struct {
	fileinfo FileInfo
	content  []byte
	mu       sync.RWMutex

	ptr   int
	ptrMu sync.Mutex
}

func (*memFile) isMemDirEntry() {}

func (m *memFile) Name() string {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.name
}

func (m *memFile) IsDir() bool {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.isDir
}

func (m *memFile) Type() fs.FileMode {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	return m.fileinfo.mode
}

func (m *memFile) Info() (fs.FileInfo, error) {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()
	// return the info when this entry is read
	return m.fileinfo, nil
}

func (m *memFile) Stat() (fs.FileInfo, error) {
	return m.Info()
}

func (m *memFile) resetPtr() {
	m.ptrMu.Lock()
	defer m.ptrMu.Unlock()
	m.ptr = 0
}

func (m *memFile) Read(b []byte) (int, error) {
	m.mu.RLocker().Lock()
	defer m.mu.RLocker().Unlock()

	m.ptrMu.Lock()
	defer m.ptrMu.Unlock()

	n := copy(b, m.content[m.ptr:])
	m.ptr += n
	if m.ptr == len(m.content) {
		return n, io.EOF
	}
	return n, nil
}

func (m *memFile) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

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
	return FileInfo{
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
