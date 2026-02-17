package gate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
	"github.com/irahardianto/gatekeeper/internal/engine/parser"
	"github.com/irahardianto/gatekeeper/internal/engine/pool"
)

// TestContainerGate_ExecSuccess verifies successful execution of an exec gate.
func TestContainerGate_ExecSuccess(t *testing.T) {
	mockPool := &pool.MockPool{
		ContainerID: "test-container",
	}
	mockExecutor := &pool.MockExecutor{
		Result: &pool.ExecResult{
			Stdout:   []byte("all checks passed"),
			Stderr:   []byte(""),
			ExitCode: 0,
		},
	}
	mockParser := &parser.MockParser{
		Result: &parser.ParseResult{
			Passed: true,
			Errors: nil,
		},
	}

	cfg := config.Gate{
		Name:    "lint",
		Type:    config.GateTypeExec,
		Command: "golangci-lint run ./...",
	}

	gate := NewContainerGate(cfg, mockPool, mockExecutor, mockParser, "/project")
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected gate to pass")
	}
	if result.SystemError != "" {
		t.Errorf("expected no system error, got %q", result.SystemError)
	}
	if result.DurationMs < 0 {
		t.Errorf("expected non-negative duration, got %d", result.DurationMs)
	}
}

// TestContainerGate_ScriptSuccess verifies successful execution of a script gate.
func TestContainerGate_ScriptSuccess(t *testing.T) {
	mockPool := &pool.MockPool{
		ContainerID: "test-container",
	}
	mockExecutor := &pool.MockExecutor{
		Result: &pool.ExecResult{
			Stdout:   []byte("validation passed"),
			Stderr:   []byte(""),
			ExitCode: 0,
		},
	}
	mockParser := &parser.MockParser{
		Result: &parser.ParseResult{
			Passed: true,
			Errors: nil,
		},
	}

	cfg := config.Gate{
		Name: "validate",
		Type: config.GateTypeScript,
		Path: "./scripts/validate.sh",
	}

	gate := NewContainerGate(cfg, mockPool, mockExecutor, mockParser, "/project")
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected gate to pass")
	}
}

// TestContainerGate_ContainerSetupFailure verifies error handling when container setup fails.
func TestContainerGate_ContainerSetupFailure(t *testing.T) {
	mockPool := &pool.MockPool{
		Err: errors.New("docker daemon not running"),
	}

	cfg := config.Gate{
		Name:    "lint",
		Type:    config.GateTypeExec,
		Command: "go vet ./...",
	}

	gate := NewContainerGate(cfg, mockPool, nil, nil, "/project")
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected gate to fail")
	}
	if result.SystemError == "" {
		t.Error("expected system error for container setup failure")
	}
	if !contains(result.SystemError, "container setup failed") {
		t.Errorf("expected 'container setup failed' in error, got %q", result.SystemError)
	}
}

// TestContainerGate_ExecutionFailure verifies error handling when command execution fails.
func TestContainerGate_ExecutionFailure(t *testing.T) {
	mockPool := &pool.MockPool{
		ContainerID: "test-container",
	}
	mockExecutor := &pool.MockExecutor{
		Err: errors.New("command timeout"),
	}

	cfg := config.Gate{
		Name:    "test",
		Type:    config.GateTypeExec,
		Command: "npm test",
		Timeout: 5 * time.Second,
	}

	gate := NewContainerGate(cfg, mockPool, mockExecutor, nil, "/project")
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected gate to fail")
	}
	if result.SystemError == "" {
		t.Error("expected system error for execution failure")
	}
	if !contains(result.SystemError, "execution failed") {
		t.Errorf("expected 'execution failed' in error, got %q", result.SystemError)
	}
}

// TestContainerGate_ParserError verifies error handling when parser fails.
func TestContainerGate_ParserError(t *testing.T) {
	mockPool := &pool.MockPool{
		ContainerID: "test-container",
	}
	mockExecutor := &pool.MockExecutor{
		Result: &pool.ExecResult{
			Stdout:   []byte("invalid output"),
			Stderr:   []byte(""),
			ExitCode: 1,
		},
	}
	mockParser := &parser.MockParser{
		Err: errors.New("failed to parse SARIF output"),
	}

	cfg := config.Gate{
		Name:    "security",
		Type:    config.GateTypeExec,
		Command: "gosec -fmt=sarif ./...",
		Parser:  "sarif",
	}

	gate := NewContainerGate(cfg, mockPool, mockExecutor, mockParser, "/project")
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected gate to fail")
	}
	if result.SystemError == "" {
		t.Error("expected system error for parser failure")
	}
	if !contains(result.SystemError, "parser error") {
		t.Errorf("expected 'parser error' in error, got %q", result.SystemError)
	}
}

// TestContainerGate_DefaultTimeout verifies default timeout is applied when not specified.
func TestContainerGate_DefaultTimeout(t *testing.T) {
	mockPool := &pool.MockPool{
		ContainerID: "test-container",
	}
	mockExecutor := &pool.MockExecutor{
		Result: &pool.ExecResult{
			Stdout:   []byte("ok"),
			Stderr:   []byte(""),
			ExitCode: 0,
		},
	}
	mockParser := &parser.MockParser{
		Result: &parser.ParseResult{Passed: true},
	}

	cfg := config.Gate{
		Name:    "quick",
		Type:    config.GateTypeExec,
		Command: "echo ok",
		// No timeout specified
	}

	gate := NewContainerGate(cfg, mockPool, mockExecutor, mockParser, "/project")
	_, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify executor was called with default timeout (30s)
	if mockExecutor.LastTimeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", mockExecutor.LastTimeout)
	}
}

// TestContainerGate_BuildCommand verifies command construction for exec and script gates.
func TestContainerGate_BuildCommand(t *testing.T) {
	tests := []struct {
		name     string
		gateType config.GateType
		command  string
		path     string
		expected string
	}{
		{
			name:     "exec gate uses command directly",
			gateType: config.GateTypeExec,
			command:  "go test ./...",
			expected: "go test ./...",
		},
		{
			name:     "script gate quotes path",
			gateType: config.GateTypeScript,
			path:     "./scripts/check.sh",
			expected: "sh './scripts/check.sh'",
		},
		{
			name:     "script gate with special chars",
			gateType: config.GateTypeScript,
			path:     "./scripts/test file.sh",
			expected: "sh './scripts/test file.sh'",
		},
		{
			name:     "script gate escapes single quotes",
			gateType: config.GateTypeScript,
			path:     "./scripts/my'script.sh",
			expected: `sh './scripts/my'\''script.sh'`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Gate{
				Name:    "test",
				Type:    tc.gateType,
				Command: tc.command,
				Path:    tc.path,
			}

			gate := NewContainerGate(cfg, nil, nil, nil, "/project")
			got := gate.buildCommand()

			if got != tc.expected {
				t.Errorf("buildCommand() = %q, want %q", got, tc.expected)
			}
		})
	}
}

// TestContainerGate_BlockingFlag verifies blocking flag is set correctly.
func TestContainerGate_BlockingFlag(t *testing.T) {
	mockPool := &pool.MockPool{ContainerID: "test"}
	mockExecutor := &pool.MockExecutor{
		Result: &pool.ExecResult{Stdout: []byte("ok"), ExitCode: 0},
	}
	mockParser := &parser.MockParser{
		Result: &parser.ParseResult{Passed: false},
	}

	blocking := true
	cfg := config.Gate{
		Name:     "critical",
		Type:     config.GateTypeExec,
		Command:  "test",
		Blocking: &blocking,
	}

	gate := NewContainerGate(cfg, mockPool, mockExecutor, mockParser, "/project")
	result, _ := gate.Execute(context.Background())

	if !result.Blocking {
		t.Error("expected blocking flag to be true")
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
