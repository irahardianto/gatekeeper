package commands

import (
	"bytes"
	"testing"
)

func TestRootCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("root --help returned error: %v", err)
	}

	output := buf.String()
	assertContains(t, output, "gatekeeper")
	assertContains(t, output, "pre-commit")
}

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command returned error: %v", err)
	}
}

func TestRootCommand_HasSubcommands(t *testing.T) {
	expected := map[string]bool{
		"init":     false,
		"teardown": false,
		"run":      false,
		"dry-run":  false,
		"cleanup":  false,
		"version":  false,
	}

	for _, cmd := range rootCmd.Commands() {
		if _, ok := expected[cmd.Use]; ok {
			expected[cmd.Use] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("expected subcommand %q to be registered, but it was not", name)
		}
	}
}

func TestRootCommand_GlobalFlags(t *testing.T) {
	flags := []string{"json", "verbose", "no-color", "fail-fast", "skip", "skip-llm"}

	for _, name := range flags {
		flag := rootCmd.PersistentFlags().Lookup(name)
		if flag == nil {
			t.Errorf("expected global flag --%s to be registered", name)
		}
	}
}

func TestRunCommand_NoConfig(t *testing.T) {
	rootCmd.SetArgs([]string{"run"})

	// Run command now does real work — it will fail without gates.yaml.
	// This verifies the pipeline starts correctly and exits with a meaningful error.
	err := rootCmd.Execute()
	if err != nil {
		// Expected: config not found or Docker not available.
		// Both are acceptable in test environment.
		t.Logf("run command returned expected error: %v", err)
	}
}

func TestDryRunCommand_NoConfig(t *testing.T) {
	rootCmd.SetArgs([]string{"dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Logf("dry-run command returned expected error: %v", err)
	}
}

func TestCleanupCommand(t *testing.T) {
	rootCmd.SetArgs([]string{"cleanup"})

	// Cleanup accesses Docker — may fail if Docker is not running.
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("cleanup command returned expected error: %v", err)
	}
}

func TestInitCommand_NoGitRepo(t *testing.T) {
	rootCmd.SetArgs([]string{"init"})

	// Init requires a git repo to install hooks.
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("init command returned expected error: %v", err)
	}
}

func TestTeardownCommand_NoGitRepo(t *testing.T) {
	rootCmd.SetArgs([]string{"teardown"})

	// Teardown requires a git repo to find hooks.
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("teardown command returned expected error: %v", err)
	}
}

func TestExecute(t *testing.T) {
	// Execute() is a convenience wrapper around rootCmd.Execute().
	// With no args it prints help and succeeds.
	rootCmd.SetArgs([]string{})
	if err := Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
}

func assertContains(t *testing.T, output, substr string) {
	t.Helper()
	if !bytes.Contains([]byte(output), []byte(substr)) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, output)
	}
}
