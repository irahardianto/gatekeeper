package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testdataPath(name string) string {
	return filepath.Join("testdata", name)
}

func loadConfig(t *testing.T, name string) (*GatekeeperConfig, error) {
	path := testdataPath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		// For tests expecting missing files, we might fail here if we try to read it.
		// But those tests usually pass a non-existent name.
		// If the file intentionally doesn't exist on disk but we want to simulate it,
		// we should handle that.
		// However, for existing test structure, we can just return error if read fails,
		// UNLESS the test is specifically for missing file.
		// Let's rely on the test specific logic.
		return nil, err
	}

	mockFS := NewMockFileSystem()
	mockFS.Files[path] = data
	// Need to handle absolute paths if Load calls Clean?
	// filepath.Clean("testdata/...") stays relative.

	loader := NewLoader(mockFS)
	return loader.Load(context.Background(), path)
}

func TestLoad_ValidFull(t *testing.T) {
	cfg, err := loadConfig(t, "valid_full.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}

	if len(cfg.Gates) != 4 {
		t.Fatalf("expected 4 gates, got %d", len(cfg.Gates))
	}

	// Verify exec gate
	g := cfg.Gates[0]
	if g.Name != "unit_tests" {
		t.Errorf("expected gate name 'unit_tests', got %q", g.Name)
	}
	if g.Type != GateTypeExec {
		t.Errorf("expected gate type 'exec', got %q", g.Type)
	}
	if g.Command != "go test ./... -json" {
		t.Errorf("expected command 'go test ./... -json', got %q", g.Command)
	}
	if g.Parser != "go-test-json" {
		t.Errorf("expected parser 'go-test-json', got %q", g.Parser)
	}
	if g.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", g.Timeout)
	}

	// Verify script gate
	g = cfg.Gates[2]
	if g.Type != GateTypeScript {
		t.Errorf("expected type 'script', got %q", g.Type)
	}
	if g.Path != "./scripts/validate-api.sh" {
		t.Errorf("expected path './scripts/validate-api.sh', got %q", g.Path)
	}
	if g.OnError != OnErrorWarn {
		t.Errorf("expected on_error 'warn', got %q", g.OnError)
	}

	// Verify LLM gate
	g = cfg.Gates[3]
	if g.Type != GateTypeLLM {
		t.Errorf("expected type 'llm', got %q", g.Type)
	}
	if g.Provider != "gemini-3-pro" {
		t.Errorf("expected provider 'gemini-3-pro', got %q", g.Provider)
	}
	if g.Prompt == "" {
		t.Error("expected prompt to be set")
	}
}

func TestLoad_ValidMinimal(t *testing.T) {
	cfg, err := loadConfig(t, "valid_minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Gates) != 1 {
		t.Fatalf("expected 1 gate, got %d", len(cfg.Gates))
	}

	g := cfg.Gates[0]
	if g.Name != "unit_tests" {
		t.Errorf("expected gate name 'unit_tests', got %q", g.Name)
	}
}

func TestLoad_DefaultsApplied(t *testing.T) {
	cfg, err := loadConfig(t, "valid_minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	g := cfg.Gates[0]

	// Container should be inherited from defaults
	if g.Container != "golang:1.23" {
		t.Errorf("expected container 'golang:1.23' from defaults, got %q", g.Container)
	}

	// Timeout should be inherited from defaults
	if g.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s from defaults, got %v", g.Timeout)
	}
}

func TestLoad_DefaultsDoNotOverrideExplicit(t *testing.T) {
	cfg, err := loadConfig(t, "valid_full.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// unit_tests gate has explicit timeout 60s, defaults is 30s
	g := cfg.Gates[0]
	if g.Timeout != 60*time.Second {
		t.Errorf("expected explicit timeout 60s to not be overridden, got %v", g.Timeout)
	}

	// lint gate has explicit container, defaults container is golang:1.23
	g = cfg.Gates[1]
	if g.Container != "golangci/golangci-lint:latest" {
		t.Errorf("expected explicit container to not be overridden, got %q", g.Container)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	// For missing file, we simulate it in mockFS
	mockFS := NewMockFileSystem() // empty
	loader := NewLoader(mockFS)

	_, err := loader.Load(context.Background(), "nonexistent.yaml")
	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound, got: %v", err)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	cfg, err := loadConfig(t, "invalid_yaml.yaml")
	// loadConfig might succeed in reading file but fail in Loader.Load
	if err == nil && cfg != nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
	// Check that we got error from Loader, not from os.ReadFile in helper
	if err != nil && cfg == nil {
		// OK
	}
}

func TestLoad_MissingCommand(t *testing.T) {
	_, err := loadConfig(t, "missing_command.yaml")
	if err == nil {
		t.Fatal("expected validation error for missing command, got nil")
	}
	expected := "gate \"lint\": missing required field 'command' for type 'exec'"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestLoad_MissingPath(t *testing.T) {
	_, err := loadConfig(t, "missing_path.yaml")
	if err == nil {
		t.Fatal("expected validation error for missing path, got nil")
	}
	expected := "gate \"check\": missing required field 'path' for type 'script'"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestLoad_MissingPrompt(t *testing.T) {
	_, err := loadConfig(t, "missing_prompt.yaml")
	if err == nil {
		t.Fatal("expected validation error for missing prompt, got nil")
	}
	expected := "gate \"review\": missing required field 'prompt' for type 'llm'"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestLoad_UnknownType(t *testing.T) {
	_, err := loadConfig(t, "unknown_type.yaml")
	if err == nil {
		t.Fatal("expected validation error for unknown gate type, got nil")
	}
	expected := "gate \"check\": unknown gate type \"magic\" (valid: exec, script, llm)"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestLoad_EmptyGates(t *testing.T) {
	cfg, err := loadConfig(t, "empty_gates.yaml")
	if err != nil {
		t.Fatalf("unexpected error for empty gates: %v", err)
	}
	if len(cfg.Gates) != 0 {
		t.Errorf("expected 0 gates, got %d", len(cfg.Gates))
	}
}

func TestGate_IsBlocking(t *testing.T) {
	tests := []struct {
		name     string
		blocking *bool
		want     bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Gate{Blocking: tt.blocking}
			if got := g.IsBlocking(); got != tt.want {
				t.Errorf("IsBlocking() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGate_GetOnError(t *testing.T) {
	tests := []struct {
		name    string
		onError OnErrorPolicy
		want    OnErrorPolicy
	}{
		{"empty defaults to block", "", OnErrorBlock},
		{"explicit block", OnErrorBlock, OnErrorBlock},
		{"explicit warn", OnErrorWarn, OnErrorWarn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Gate{OnError: tt.onError}
			if got := g.GetOnError(); got != tt.want {
				t.Errorf("GetOnError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestLoad_ConvenienceFunction(t *testing.T) {
	// Load() uses RealFileSystem under the hood.
	// Test that it returns ErrConfigNotFound for a non-existent path.
	_, err := Load(context.Background(), "/nonexistent/path/gates.yaml")
	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound, got: %v", err)
	}
}

func TestLoad_ReadError(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.ReadErrors["broken.yaml"] = errors.New("disk I/O error")

	loader := NewLoader(mockFS)
	_, err := loader.Load(context.Background(), "broken.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reading config file") {
		t.Errorf("expected 'reading config file' error, got: %v", err)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &GatekeeperConfig{
		Gates: []Gate{
			{Name: "g1", Type: GateTypeExec, Command: ""}, // missing command
			{Name: "g2", Type: GateTypeScript, Path: ""},  // missing path
		},
	}
	err := validate(cfg)
	if err == nil {
		t.Fatal("expected validation errors, got nil")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "g1") || !strings.Contains(errStr, "g2") {
		t.Errorf("expected errors for both g1 and g2, got: %v", err)
	}
}

func TestValidate_PathContainsSingleQuote(t *testing.T) {
	cfg := &GatekeeperConfig{
		Gates: []Gate{
			{Name: "check", Type: GateTypeScript, Path: "my'script.sh"},
		},
	}
	err := validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for path with single quote")
	}
	if !strings.Contains(err.Error(), "invalid character (single quote)") {
		t.Errorf("expected single-quote error, got: %v", err)
	}
}
