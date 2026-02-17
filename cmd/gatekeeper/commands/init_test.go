package commands

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/irahardianto/gatekeeper/internal/engine/git"
)

// --- mockInitFS implements InitFS for unit tests ---

type mockInitFS struct {
	statErr        error
	statNotExist   bool
	mkdirErr       error
	readDirEntries []fs.DirEntry
	readDirErr     error
	writeErr       error
	writtenData    []byte
	writtenPath    string
}

func (m *mockInitFS) Stat(_ string) (fs.FileInfo, error) {
	if m.statNotExist {
		return nil, m.statErr
	}
	// Return non-nil info to simulate file exists.
	return &mockFileInfo{}, m.statErr
}

func (m *mockInitFS) IsNotExist(_ error) bool {
	return m.statNotExist
}

func (m *mockInitFS) MkdirAll(_ string, _ fs.FileMode) error {
	return m.mkdirErr
}

func (m *mockInitFS) ReadDir(_ string) ([]fs.DirEntry, error) {
	return m.readDirEntries, m.readDirErr
}

func (m *mockInitFS) WriteFile(name string, data []byte, _ fs.FileMode) error {
	m.writtenPath = name
	m.writtenData = data
	return m.writeErr
}

// mockFileInfo satisfies fs.FileInfo for testing.
type mockFileInfo struct{}

func (m *mockFileInfo) Name() string       { return "gates.yaml" }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() fs.FileMode  { return 0 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// mockDirEntry implements fs.DirEntry for testing.
type mockDirEntry struct {
	name string
}

func (d *mockDirEntry) Name() string               { return d.name }
func (d *mockDirEntry) IsDir() bool                { return false }
func (d *mockDirEntry) Type() fs.FileMode          { return 0 }
func (d *mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

// --- Tests ---

func TestInitProject_HappyPath_GoProject(t *testing.T) {
	fsys := &mockInitFS{
		statNotExist: true,
		statErr:      errors.New("file does not exist"),
		readDirEntries: []fs.DirEntry{
			&mockDirEntry{name: "go.mod"},
			&mockDirEntry{name: "main.go"},
		},
	}
	gitSvc := &git.MockService{}
	out := &bytes.Buffer{}

	err := initProject(context.Background(), "/project", fsys, gitSvc, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect Go and generate config.
	if !strings.Contains(out.String(), "Detected") {
		t.Errorf("expected 'Detected' in output, got %q", out.String())
	}
	if fsys.writtenData == nil {
		t.Error("expected gates.yaml to be written")
	}
}

func TestInitProject_ConfigAlreadyExists(t *testing.T) {
	fsys := &mockInitFS{
		statNotExist: false, // Config exists
	}
	gitSvc := &git.MockService{}
	out := &bytes.Buffer{}

	err := initProject(context.Background(), "/project", fsys, gitSvc, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "already exists") {
		t.Errorf("expected 'already exists' in output, got %q", out.String())
	}
}

func TestInitProject_MkdirError(t *testing.T) {
	fsys := &mockInitFS{
		mkdirErr: errors.New("permission denied"),
	}
	gitSvc := &git.MockService{}
	out := &bytes.Buffer{}

	err := initProject(context.Background(), "/project", fsys, gitSvc, out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "creating .gatekeeper directory") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestInitProject_WriteError(t *testing.T) {
	fsys := &mockInitFS{
		statNotExist: true,
		statErr:      errors.New("file does not exist"),
		readDirEntries: []fs.DirEntry{
			&mockDirEntry{name: "go.mod"},
		},
		writeErr: errors.New("disk full"),
	}
	gitSvc := &git.MockService{}
	out := &bytes.Buffer{}

	err := initProject(context.Background(), "/project", fsys, gitSvc, out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "writing gates.yaml") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestInitProject_HookInstallError(t *testing.T) {
	fsys := &mockInitFS{
		statNotExist: false, // Config exists, skip writing
	}
	gitSvc := &git.MockService{HookInstErr: errors.New("not a git repo")}
	out := &bytes.Buffer{}

	err := initProject(context.Background(), "/project", fsys, gitSvc, out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "installing hook") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}
