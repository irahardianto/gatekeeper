package config

import (
	"io/fs"
	"os"
)

// FileSystem abstracts file system operations for testing.
type FileSystem interface {
	ReadFile(name string) ([]byte, error)
	UserHomeDir() (string, error)
	Stat(name string) (fs.FileInfo, error)
	IsNotExist(err error) bool
}

// RealFileSystem implements FileSystem using the os package.
type RealFileSystem struct{}

// ReadFile reads the named file and returns the contents.
func (r *RealFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) // #nosec G304 -- Accepted risk: variable path is intentional; all callers validate/clean paths. CLI runs as current user.
}

// UserHomeDir returns the current user's home directory.
func (r *RealFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

// Stat returns a FileInfo describing the named file.
func (r *RealFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// IsNotExist returns a boolean indicating whether the error is known to report that a file or directory does not exist.
func (r *RealFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}
