package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRealFileSystem(t *testing.T) {
	tmpDir := t.TempDir()
	fs := &RealFileSystem{}

	path := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello")

	// Test IsNotExist before create
	if !fs.IsNotExist(os.ErrNotExist) {
		t.Error("expected IsNotExist to return true for os.ErrNotExist")
	}

	// Test Write (setup)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Test Stat
	info, err := fs.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Size() != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), info.Size())
	}

	// Test ReadFile
	got, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("expected content %q, got %q", content, got)
	}

	// Test UserHomeDir
	home, err := fs.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir failed: %v", err)
	}
	if home == "" {
		t.Error("expected non-empty UserHomeDir")
	}
}
