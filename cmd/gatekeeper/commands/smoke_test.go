package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSmoke_InitAndTeardown verifies the init → teardown lifecycle in a real git repo.
// This is a lightweight E2E test that doesn't require Docker.
func TestSmoke_InitAndTeardown(t *testing.T) {
	// Create a temporary directory with a go.mod to simulate a Go project.
	tmpDir := t.TempDir()

	// Initialize a git repo so hook installation works.
	run(t, tmpDir, "git", "init")
	run(t, tmpDir, "git", "config", "user.email", "test@test.com")
	run(t, tmpDir, "git", "config", "user.name", "test")

	// Create a marker file for stack detection.
	gomod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(gomod, []byte("module testproject\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	// Save cwd and change to tmpDir for the commands.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to tmpDir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// 1. Run init.
	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify gates.yaml was created.
	configPath := filepath.Join(tmpDir, ".gatekeeper", "gates.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading gates.yaml: %v", err)
	}
	content := string(data)

	// Should contain Go-specific gates since we have go.mod.
	if !strings.Contains(content, "go vet") {
		t.Error("expected gates.yaml to contain 'go vet' for Go project")
	}
	if !strings.Contains(content, "version: 1") {
		t.Error("expected gates.yaml to contain 'version: 1'")
	}

	// Verify hook was installed.
	hookPath := filepath.Join(tmpDir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Error("expected pre-commit hook to be installed")
	}

	// 2. Run teardown.
	rootCmd.SetArgs([]string{"teardown"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("teardown command failed: %v", err)
	}

	// Verify hook was removed.
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("expected pre-commit hook to be removed after teardown")
	}
}

// TestSmoke_InitExistingConfig verifies init skips generation when config exists.
func TestSmoke_InitExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()

	run(t, tmpDir, "git", "init")
	run(t, tmpDir, "git", "config", "user.email", "test@test.com")
	run(t, tmpDir, "git", "config", "user.name", "test")

	// Pre-create config.
	gkDir := filepath.Join(tmpDir, ".gatekeeper")
	if err := os.MkdirAll(gkDir, 0o750); err != nil {
		t.Fatalf("creating .gatekeeper dir: %v", err)
	}
	existingConfig := "version: 1\ngates: []\n"
	if err := os.WriteFile(filepath.Join(gkDir, "gates.yaml"), []byte(existingConfig), 0o644); err != nil {
		t.Fatalf("writing existing config: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify config was NOT overwritten.
	data, err := os.ReadFile(filepath.Join(gkDir, "gates.yaml"))
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if string(data) != existingConfig {
		t.Error("expected existing config to be preserved")
	}
}

// TestSmoke_InitNodeProject verifies stack detection works for Node.js projects.
func TestSmoke_InitNodeProject(t *testing.T) {
	tmpDir := t.TempDir()

	run(t, tmpDir, "git", "init")
	run(t, tmpDir, "git", "config", "user.email", "test@test.com")
	run(t, tmpDir, "git", "config", "user.name", "test")

	// Create package.json for Node detection.
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"test"}`), 0o644); err != nil {
		t.Fatalf("writing package.json: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".gatekeeper", "gates.yaml"))
	if err != nil {
		t.Fatalf("reading gates.yaml: %v", err)
	}
	if !strings.Contains(string(data), "eslint") {
		t.Error("expected Node.js gates.yaml to contain 'eslint'")
	}
}

// run is a test helper that executes a command and fails the test on error.
func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %q failed: %v\nOutput: %s", name+" "+strings.Join(args, " "), err, out)
	}
}
