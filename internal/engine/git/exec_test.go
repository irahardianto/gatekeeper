package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupGitRepo creates a temporary git repository and returns its path.
// It initializes the repo, configures a test user, and returns a cleanup function.
func setupGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Init repo
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")

	return dir
}

// run executes a command in the given directory and fails the test on error.
func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

func TestExecService_StagedDiff(t *testing.T) {
	dir := setupGitRepo(t)

	// Create and stage a file
	filePath := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "hello.go")

	svc := NewExecService(dir)
	diffs, err := svc.StagedDiff(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Path != "hello.go" {
		t.Errorf("expected path 'hello.go', got %q", diffs[0].Path)
	}
	if !strings.Contains(diffs[0].Content, "package main") {
		t.Errorf("expected diff to contain 'package main', got:\n%s", diffs[0].Content)
	}
}

func TestExecService_StagedDiff_MultipleStagedFiles(t *testing.T) {
	dir := setupGitRepo(t)

	// Create and stage two files
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.go"), []byte("package b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "a.go", "b.go")

	svc := NewExecService(dir)
	diffs, err := svc.StagedDiff(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}

	paths := map[string]bool{}
	for _, d := range diffs {
		paths[d.Path] = true
	}
	if !paths["a.go"] || !paths["b.go"] {
		t.Errorf("expected diffs for a.go and b.go, got paths: %v", paths)
	}
}

func TestExecService_StagedDiff_EmptyStaging(t *testing.T) {
	dir := setupGitRepo(t)

	svc := NewExecService(dir)
	diffs, err := svc.StagedDiff(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for empty staging, got %d", len(diffs))
	}
}

func TestExecService_StagedFiles(t *testing.T) {
	dir := setupGitRepo(t)

	// Create and stage files
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "utils.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "main.go", "utils.go")

	svc := NewExecService(dir)
	files, err := svc.StagedFiles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	fileSet := map[string]bool{}
	for _, f := range files {
		fileSet[f] = true
	}
	if !fileSet["main.go"] || !fileSet["utils.go"] {
		t.Errorf("expected main.go and utils.go, got: %v", files)
	}
}

func TestExecService_StagedFiles_EmptyStaging(t *testing.T) {
	dir := setupGitRepo(t)

	svc := NewExecService(dir)
	files, err := svc.StagedFiles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if files != nil {
		t.Errorf("expected nil for empty staging, got %v", files)
	}
}

func TestExecService_StagedDiff_InvalidWorkDir(t *testing.T) {
	svc := NewExecService("/nonexistent/path/that/does/not/exist")

	_, err := svc.StagedDiff(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid work dir, got nil")
	}
}

func TestExecService_StagedFiles_InvalidWorkDir(t *testing.T) {
	svc := NewExecService("/nonexistent/path/that/does/not/exist")

	_, err := svc.StagedFiles(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid work dir, got nil")
	}
}
