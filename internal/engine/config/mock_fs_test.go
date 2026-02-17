package config

import (
	"io/fs"
	"os"
	"time"
)

// MockFileSystem is an in-memory file system for testing.
type MockFileSystem struct {
	Files       map[string][]byte
	ReadErrors  map[string]error
	StatErrors  map[string]error
	UserHome    string
	UserHomeErr error
}

// NewMockFileSystem creates a new MockFileSystem.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files:      make(map[string][]byte),
		ReadErrors: make(map[string]error),
		StatErrors: make(map[string]error),
	}
}

// ReadFile returns the content of the file from memory.
func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	if err, ok := m.ReadErrors[name]; ok {
		return nil, err
	}
	content, ok := m.Files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return content, nil
}

// UserHomeDir returns the configured user home directory.
func (m *MockFileSystem) UserHomeDir() (string, error) {
	if m.UserHomeErr != nil {
		return "", m.UserHomeErr
	}
	return m.UserHome, nil
}

// Stat returns a mock FileInfo.
func (m *MockFileSystem) Stat(name string) (fs.FileInfo, error) {
	if err, ok := m.StatErrors[name]; ok {
		return nil, err
	}
	if _, ok := m.Files[name]; !ok {
		return nil, os.ErrNotExist
	}
	return &mockFileInfo{name: name}, nil
}

// IsNotExist checks if the error is os.ErrNotExist.
func (m *MockFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

// mockFileInfo implements fs.FileInfo.
type mockFileInfo struct {
	name string
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() fs.FileMode  { return 0o644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }
