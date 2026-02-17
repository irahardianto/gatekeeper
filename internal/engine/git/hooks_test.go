package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallHook_Fresh(t *testing.T) {
	dir := setupGitRepo(t)
	svc := NewExecService(dir)

	if err := svc.InstallHook(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("reading hook: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty hook file")
	}

	// Check hook is executable.
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("stat hook: %v", err)
	}
	if info.Mode()&0o100 == 0 {
		t.Errorf("expected executable hook, got mode %v", info.Mode())
	}
}

func TestInstallHook_Idempotent(t *testing.T) {
	dir := setupGitRepo(t)
	svc := NewExecService(dir)

	if err := svc.InstallHook(context.Background()); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Re-install should not error (idempotent).
	if err := svc.InstallHook(context.Background()); err != nil {
		t.Fatalf("second install should be idempotent: %v", err)
	}
}

func TestInstallHook_ConflictsWithExistingHook(t *testing.T) {
	dir := setupGitRepo(t)
	svc := NewExecService(dir)

	// Create a non-gatekeeper hook.
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho custom hook\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := svc.InstallHook(context.Background())
	if err == nil {
		t.Fatal("expected error when existing non-gatekeeper hook present")
	}
}

func TestRemoveHook_Existing(t *testing.T) {
	dir := setupGitRepo(t)
	svc := NewExecService(dir)

	// Install first.
	if err := svc.InstallHook(context.Background()); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Remove.
	if err := svc.RemoveHook(context.Background()); err != nil {
		t.Fatalf("remove: %v", err)
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("expected hook to be removed, but it still exists")
	}
}

func TestRemoveHook_NoHookPresent(t *testing.T) {
	dir := setupGitRepo(t)
	svc := NewExecService(dir)

	// Should not error when no hook exists.
	if err := svc.RemoveHook(context.Background()); err != nil {
		t.Fatalf("unexpected error removing non-existent hook: %v", err)
	}
}

func TestRemoveHook_NonGatekeeperHook(t *testing.T) {
	dir := setupGitRepo(t)
	svc := NewExecService(dir)

	// Create a non-gatekeeper hook.
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho custom\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := svc.RemoveHook(context.Background())
	if err == nil {
		t.Fatal("expected error when removing non-gatekeeper hook")
	}
}
