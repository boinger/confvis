package cli

import (
	"io"
	"os"
	"path/filepath"
)

// FileSystem abstracts file operations for testability.
type FileSystem interface {
	// Create creates or truncates the named file.
	Create(name string) (io.WriteCloser, error)
	// MkdirAll creates a directory named path, along with any necessary parents.
	MkdirAll(path string, perm os.FileMode) error
	// Open opens the named file for reading.
	Open(name string) (io.ReadCloser, error)
	// ReadFile reads the named file and returns the contents.
	ReadFile(name string) ([]byte, error)
	// Glob returns the names of all files matching pattern.
	Glob(pattern string) ([]string, error)
}

// osFS implements FileSystem using the real os and filepath packages.
type osFS struct{}

// DefaultFileSystem is the default FileSystem implementation using real OS calls.
var DefaultFileSystem FileSystem = osFS{}

func (osFS) Create(name string) (io.WriteCloser, error) {
	return os.Create(name)
}

func (osFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (osFS) Open(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

func (osFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (osFS) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
