package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// mockWriteCloser wraps a bytes.Buffer and tracks whether it was closed.
type mockWriteCloser struct {
	buf    *bytes.Buffer
	closed bool
	mu     sync.Mutex
}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, fmt.Errorf("write on closed file")
	}
	return m.buf.Write(p)
}

func (m *mockWriteCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// mockReadCloser wraps a bytes.Reader and tracks whether it was closed.
type mockReadCloser struct {
	reader *bytes.Reader
	closed bool
	mu     sync.Mutex
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, fmt.Errorf("read on closed file")
	}
	return m.reader.Read(p)
}

func (m *mockReadCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// MockFileSystem is an in-memory implementation of FileSystem for testing.
type MockFileSystem struct {
	// Files stores created files as path -> buffer
	Files map[string]*bytes.Buffer
	// Dirs tracks created directories
	Dirs map[string]bool
	// FileContent stores preset file content for reading
	FileContent map[string]string
	// GlobMatches stores preset glob results
	GlobMatches map[string][]string
	// Errors stores errors to return for specific operations
	// Keys are like "create:/path" or "open:/path" or "mkdir:/path" or "glob:pattern"
	Errors map[string]error

	mu sync.Mutex
}

// NewMockFileSystem creates a new MockFileSystem with initialized maps.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files:       make(map[string]*bytes.Buffer),
		Dirs:        make(map[string]bool),
		FileContent: make(map[string]string),
		GlobMatches: make(map[string][]string),
		Errors:      make(map[string]error),
	}
}

func (m *MockFileSystem) Create(name string) (io.WriteCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, ok := m.Errors["create:"+name]; ok {
		return nil, err
	}

	buf := &bytes.Buffer{}
	m.Files[name] = buf
	return &mockWriteCloser{buf: buf}, nil
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, ok := m.Errors["mkdir:"+path]; ok {
		return err
	}

	m.Dirs[path] = true
	return nil
}

func (m *MockFileSystem) Open(name string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, ok := m.Errors["open:"+name]; ok {
		return nil, err
	}

	content, ok := m.FileContent[name]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}

	return &mockReadCloser{reader: bytes.NewReader([]byte(content))}, nil
}

func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, ok := m.Errors["read:"+name]; ok {
		return nil, err
	}

	content, ok := m.FileContent[name]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}

	return []byte(content), nil
}

func (m *MockFileSystem) Glob(pattern string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, ok := m.Errors["glob:"+pattern]; ok {
		return nil, err
	}

	if matches, ok := m.GlobMatches[pattern]; ok {
		return matches, nil
	}

	// If no preset matches, check if any FileContent paths match the pattern
	var matches []string
	for path := range m.FileContent {
		matched, err := filepath.Match(pattern, path)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, path)
		}
	}

	return matches, nil
}

// GetFileContent returns the content written to a file, or empty string if not found.
func (m *MockFileSystem) GetFileContent(name string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if buf, ok := m.Files[name]; ok {
		return buf.String()
	}
	return ""
}

// HasDir returns true if MkdirAll was called for the given path.
func (m *MockFileSystem) HasDir(path string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Dirs[path]
}

// SetFileContent sets the content that will be returned when reading a file.
func (m *MockFileSystem) SetFileContent(name, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.FileContent[name] = content
}

// SetGlobMatches sets the matches that will be returned for a glob pattern.
func (m *MockFileSystem) SetGlobMatches(pattern string, matches []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GlobMatches[pattern] = matches
}

// SetError sets an error to be returned for a specific operation.
// Key format: "create:/path", "open:/path", "mkdir:/path", "read:/path", "glob:pattern"
func (m *MockFileSystem) SetError(key string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Errors[key] = err
}
