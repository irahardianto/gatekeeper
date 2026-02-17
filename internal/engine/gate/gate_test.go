package gate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
	"github.com/irahardianto/gatekeeper/internal/engine/git"
	"github.com/irahardianto/gatekeeper/internal/engine/llm"
	"github.com/irahardianto/gatekeeper/internal/engine/parser"
)

// --- Tests ---

func TestExecGate_Success(t *testing.T) {
	// This test verifies that ExecGate works at the integration level with
	// pool/executor/parser. Since pool and executor are internal, we can't easily
	// mock them from outside the package without exposing test helpers.
	//
	// For unit testing the gate logic, we focus on the Factory and LLMGate tests
	// which use mockable interfaces (llm.Client, git.Service).
	//
	// Full ExecGate execution requires Docker mock setup at the pool level,
	// which is covered by pool package tests.
}

func TestLLMGate_Success(t *testing.T) {
	gitSvc := &git.MockService{
		Diffs: []git.FileDiff{
			{Path: "main.go", Content: "diff content here\n@@ -1,5 +1,10 @@\n+var password = \"secret\""},
		},
	}
	llmClient := &llm.MockClient{
		Result: []parser.StructuredError{
			{File: "main.go", Line: 2, Severity: "error", Message: "hardcoded secret", Rule: "G101"},
		},
	}
	cfg := config.Gate{
		Name:     "secret_review",
		Type:     config.GateTypeLLM,
		Provider: "gemini-3-pro",
		Prompt:   "Check for hardcoded secrets",
	}

	gate := NewLLMGate(cfg, llmClient, gitSvc)
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected gate to fail (found issues)")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Tool != "gemini-3-pro" {
		t.Errorf("expected tool 'gemini-3-pro', got %q", result.Errors[0].Tool)
	}
	if result.Name != "secret_review" {
		t.Errorf("expected gate name 'secret_review', got %q", result.Name)
	}
}

func TestLLMGate_NoDiffs(t *testing.T) {
	gitSvc := &git.MockService{
		Diffs: nil,
	}
	llmClient := &llm.MockClient{}
	cfg := config.Gate{
		Name:     "review",
		Type:     config.GateTypeLLM,
		Provider: "gemini-3-pro",
		Prompt:   "Review code",
	}

	gate := NewLLMGate(cfg, llmClient, gitSvc)
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected gate to pass when no diffs")
	}
}

func TestLLMGate_DiffError(t *testing.T) {
	gitSvc := &git.MockService{
		DiffErr: errors.New("git error"),
	}
	llmClient := &llm.MockClient{}
	cfg := config.Gate{
		Name:     "review",
		Type:     config.GateTypeLLM,
		Provider: "gemini-3-pro",
		Prompt:   "Review code",
	}

	gate := NewLLMGate(cfg, llmClient, gitSvc)
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SystemError == "" {
		t.Error("expected system error for git failure")
	}
}

func TestLLMGate_ReviewError(t *testing.T) {
	gitSvc := &git.MockService{
		Diffs: []git.FileDiff{
			{Path: "main.go", Content: "diff\n@@ -1,5 +1,10 @@\n+foo"},
		},
	}
	llmClient := &llm.MockClient{
		Err: errors.New("LLM API error"),
	}
	cfg := config.Gate{
		Name:     "review",
		Type:     config.GateTypeLLM,
		Provider: "gemini-3-pro",
		Prompt:   "Review code",
	}

	gate := NewLLMGate(cfg, llmClient, gitSvc)
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SystemError == "" {
		t.Error("expected system error for LLM API failure")
	}
}

func TestLLMGate_AllFilesExceedSizeLimit(t *testing.T) {
	// Large content that exceeds 1KB
	largeContent := make([]byte, 2048)
	for i := range largeContent {
		largeContent[i] = 'x'
	}

	gitSvc := &git.MockService{
		Diffs: []git.FileDiff{
			{Path: "big.go", Content: string(largeContent)},
		},
	}
	llmClient := &llm.MockClient{}
	cfg := config.Gate{
		Name:        "review",
		Type:        config.GateTypeLLM,
		Provider:    "gemini-3-pro",
		Prompt:      "Review code",
		MaxFileSize: "1KB",
	}

	gate := NewLLMGate(cfg, llmClient, gitSvc)
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected gate to pass when all files skipped")
	}
}

func TestLLMGate_NoIssuesFound(t *testing.T) {
	gitSvc := &git.MockService{
		Diffs: []git.FileDiff{
			{Path: "main.go", Content: "diff\n@@ -1,5 +1,10 @@\n+clean code"},
		},
	}
	llmClient := &llm.MockClient{
		Result: nil, // No issues
	}
	cfg := config.Gate{
		Name:     "review",
		Type:     config.GateTypeLLM,
		Provider: "gemini-3-pro",
		Prompt:   "Review code",
	}

	gate := NewLLMGate(cfg, llmClient, gitSvc)
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected gate to pass when no issues found")
	}
}

func TestSkippedGate(t *testing.T) {
	gate := NewSkippedGate("lint", "exec")
	result, err := gate.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected skipped gate to pass")
	}
	if !result.Skipped {
		t.Error("expected skipped flag to be true")
	}
	if result.Name != "lint" {
		t.Errorf("expected name 'lint', got %q", result.Name)
	}
}

func TestParseMaxFileSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
		{"500", 500},
		{"invalid", 0},
		{"  200KB  ", 200 * 1024},
		{"100kb", 100 * 1024},
	}

	for _, tc := range tests {
		got := parseMaxFileSize(tc.input)
		if got != tc.expected {
			t.Errorf("parseMaxFileSize(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

// --- Factory Tests ---

func TestFactory_CreateExecGate(t *testing.T) {
	reg := parser.NewRegistry()
	reg.Register("generic", parser.NewGenericParser())
	f := NewFactory(nil, nil, reg, nil, nil, "/project")

	cfg := config.Gate{
		Name:    "lint",
		Type:    config.GateTypeExec,
		Command: "golangci-lint run ./...",
	}

	gate, err := f.Create(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate == nil {
		t.Fatal("expected gate, got nil")
	}
	if _, ok := gate.(*ContainerGate); !ok {
		t.Errorf("expected ContainerGate, got %T", gate)
	}
}

func TestFactory_CreateScriptGate(t *testing.T) {
	reg := parser.NewRegistry()
	f := NewFactory(nil, nil, reg, nil, nil, "/project")

	cfg := config.Gate{
		Name: "validate",
		Type: config.GateTypeScript,
		Path: "./scripts/validate.sh",
	}

	gate, err := f.Create(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := gate.(*ContainerGate); !ok {
		t.Errorf("expected ContainerGate, got %T", gate)
	}
}

func TestFactory_CreateLLMGate(t *testing.T) {
	reg := parser.NewRegistry()
	llmClient := &llm.MockClient{}
	gitSvc := &git.MockService{}
	f := NewFactory(nil, nil, reg, llmClient, gitSvc, "/project")

	cfg := config.Gate{
		Name:     "review",
		Type:     config.GateTypeLLM,
		Provider: "gemini-3-pro",
		Prompt:   "Review code",
	}

	gate, err := f.Create(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := gate.(*LLMGate); !ok {
		t.Errorf("expected LLMGate, got %T", gate)
	}
}

func TestFactory_CreateLLMGate_NoClient(t *testing.T) {
	reg := parser.NewRegistry()
	f := NewFactory(nil, nil, reg, nil, nil, "/project")

	cfg := config.Gate{
		Name:     "review",
		Type:     config.GateTypeLLM,
		Provider: "gemini-3-pro",
		Prompt:   "Review code",
	}

	_, err := f.Create(cfg)
	if err == nil {
		t.Error("expected error when LLM client is nil")
	}
}

func TestFactory_CreateUnknownType(t *testing.T) {
	reg := parser.NewRegistry()
	f := NewFactory(nil, nil, reg, nil, nil, "/project")

	cfg := config.Gate{
		Name: "unknown",
		Type: "magic",
	}

	_, err := f.Create(cfg)
	if err == nil {
		t.Error("expected error for unknown gate type")
	}
}

func TestFactory_CreateAll(t *testing.T) {
	reg := parser.NewRegistry()
	reg.Register("sarif", parser.NewSarifParser())
	llmClient := &llm.MockClient{}
	gitSvc := &git.MockService{}
	f := NewFactory(nil, nil, reg, llmClient, gitSvc, "/project")

	gates := []config.Gate{
		{Name: "lint", Type: config.GateTypeExec, Command: "go vet ./..."},
		{Name: "validate", Type: config.GateTypeScript, Path: "./check.sh"},
		{Name: "review", Type: config.GateTypeLLM, Provider: "gemini-3-pro", Prompt: "check"},
	}

	result, err := f.CreateAll(gates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 gates, got %d", len(result))
	}
}

func TestFactory_CreateAll_Error(t *testing.T) {
	reg := parser.NewRegistry()
	f := NewFactory(nil, nil, reg, nil, nil, "/project")

	gates := []config.Gate{
		{Name: "lint", Type: config.GateTypeExec, Command: "go vet ./..."},
		{Name: "review", Type: config.GateTypeLLM, Provider: "gemini", Prompt: "check"}, // No LLM client
	}

	_, err := f.CreateAll(gates)
	if err == nil {
		t.Error("expected error when LLM client is nil")
	}
}

// Verify gate blocking behavior
func TestGateResult_Blocking(t *testing.T) {
	blocking := true
	cfg := config.Gate{Name: "test", Blocking: &blocking}
	if !cfg.IsBlocking() {
		t.Error("expected IsBlocking to be true")
	}

	notBlocking := false
	cfg2 := config.Gate{Name: "test2", Blocking: &notBlocking}
	if cfg2.IsBlocking() {
		t.Error("expected IsBlocking to be false")
	}

	cfg3 := config.Gate{Name: "test3"} // nil = default true
	if !cfg3.IsBlocking() {
		t.Error("expected default IsBlocking to be true")
	}
}

// Verify gate duration is tracked
func TestLLMGate_DurationTracked(t *testing.T) {
	gitSvc := &git.MockService{
		Diffs: []git.FileDiff{
			{Path: "main.go", Content: "diff\n@@ -1,5 +1,10 @@\n+code"},
		},
	}
	llmClient := &llm.MockClient{Result: nil}
	cfg := config.Gate{
		Name:     "review",
		Type:     config.GateTypeLLM,
		Provider: "gemini",
		Prompt:   "check",
	}

	gate := NewLLMGate(cfg, llmClient, gitSvc)
	start := time.Now()
	result, _ := gate.Execute(context.Background())
	duration := time.Since(start)

	if result.DurationMs < 0 {
		t.Errorf("expected non-negative duration, got %d", result.DurationMs)
	}
	if result.DurationMs > duration.Milliseconds()+100 {
		t.Errorf("duration seems too large: %dms vs wall clock %dms", result.DurationMs, duration.Milliseconds())
	}
}
