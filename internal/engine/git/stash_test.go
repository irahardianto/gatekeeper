package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStash_WithUnstagedChanges(t *testing.T) {
	dir := setupGitRepo(t)

	// Create and commit an initial file.
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "main.go")
	run(t, dir, "git", "commit", "-m", "initial")

	// Stage a change.
	if err := os.WriteFile(filePath, []byte("package main\n// staged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "main.go")

	// Make an unstaged modification.
	if err := os.WriteFile(filePath, []byte("package main\n// staged\n// unstaged\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewExecService(dir)
	stashed, err := svc.Stash(context.Background())
	if err != nil {
		t.Fatalf("unexpected stash error: %v", err)
	}
	if !stashed {
		t.Error("expected stash to be created")
	}
}

func TestStash_NothingToStash(t *testing.T) {
	dir := setupGitRepo(t)

	// Create and commit a file (clean working tree).
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "main.go")
	run(t, dir, "git", "commit", "-m", "initial")

	svc := NewExecService(dir)
	stashed, err := svc.Stash(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stashed {
		t.Error("expected no stash for clean working tree")
	}
}

func TestStashPop_RestoresChanges(t *testing.T) {
	dir := setupGitRepo(t)

	// Create and commit a file.
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "main.go")
	run(t, dir, "git", "commit", "-m", "initial")

	// Stage a change to main.go.
	if err := os.WriteFile(filePath, []byte("package main\n// staged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "main.go")

	// Create an unstaged untracked file (no conflict on pop).
	untrackedPath := filepath.Join(dir, "unstaged.txt")
	if err := os.WriteFile(untrackedPath, []byte("unstaged content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewExecService(dir)
	stashed, err := svc.Stash(context.Background())
	if err != nil {
		t.Fatalf("stash: %v", err)
	}
	if !stashed {
		t.Fatal("expected stash to be created")
	}

	// After stash, the untracked file should be gone.
	if _, err := os.Stat(untrackedPath); !os.IsNotExist(err) {
		t.Error("expected untracked file to be stashed away")
	}

	// Pop stash.
	if err := svc.StashPop(context.Background()); err != nil {
		t.Fatalf("stash pop: %v", err)
	}

	// After pop, the untracked file should be back.
	if _, err := os.Stat(untrackedPath); os.IsNotExist(err) {
		t.Error("expected untracked file to be restored after pop")
	}
}

func TestStashPop_NoStash(t *testing.T) {
	dir := setupGitRepo(t)

	// Create initial commit so git stash list works.
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "main.go")
	run(t, dir, "git", "commit", "-m", "initial")

	svc := NewExecService(dir)
	// Should not error when no stash exists.
	if err := svc.StashPop(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanWritableFiles_RevertsModifications(t *testing.T) {
	dir := setupGitRepo(t)

	// Create, add, and commit a file.
	filePath := filepath.Join(dir, "app.go")
	original := "package app\n"
	if err := os.WriteFile(filePath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "app.go")
	run(t, dir, "git", "commit", "-m", "initial")

	// Modify the tracked file (simulating writable gate).
	if err := os.WriteFile(filePath, []byte("package app\n// modified by gate\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create an untracked file (simulating gate output).
	untrackedPath := filepath.Join(dir, "gate-output.txt")
	if err := os.WriteFile(untrackedPath, []byte("output"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewExecService(dir)
	if err := svc.CleanWritableFiles(context.Background()); err != nil {
		t.Fatalf("clean: %v", err)
	}

	// Tracked file should be restored.
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading restored file: %v", err)
	}
	if string(data) != original {
		t.Errorf("expected original content, got:\n%s", string(data))
	}

	// Untracked file should be removed.
	if _, err := os.Stat(untrackedPath); !os.IsNotExist(err) {
		t.Error("expected untracked file to be removed after clean")
	}
}
