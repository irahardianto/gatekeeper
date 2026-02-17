package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSarifParser_Valid(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "valid.sarif"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	p := NewSarifParser()
	res, err := p.Parse(context.Background(), data, nil, 1) // Exit code 1 common for lint errors
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not pass because there is an error level result
	if res.Passed {
		t.Error("expected failed")
	}

	if len(res.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(res.Errors))
	}

	// Check error details
	err1 := res.Errors[0]
	if err1.Rule != "no-unused-vars" {
		t.Errorf("expected rule no-unused-vars, got %s", err1.Rule)
	}
	if err1.Severity != "error" {
		t.Errorf("expected severity error, got %s", err1.Severity)
	}
	if err1.File != "src/app.js" {
		t.Errorf("expected file src/app.js, got %s", err1.File)
	}
	if err1.Line != 10 {
		t.Errorf("expected line 10, got %d", err1.Line)
	}

	// Check warning details
	warn := res.Errors[1]
	if warn.Severity != "warning" {
		t.Errorf("expected severity warning, got %s", warn.Severity)
	}
}

func TestSarifParser_InvalidJSON(t *testing.T) {
	data := []byte("{ invalid json }")
	p := NewSarifParser()
	_, err := p.Parse(context.Background(), data, nil, 1)
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestSarifParser_EmptyStdout_NonZeroExit(t *testing.T) {
	p := NewSarifParser()
	res, err := p.Parse(context.Background(), []byte("   "), []byte("stderr output"), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Passed {
		t.Error("expected failed")
	}
	if len(res.Errors) != 1 {
		t.Fatal("expected 1 error")
	}
	if res.Errors[0].Message != "stderr output" {
		t.Errorf("expected stderr message, got %q", res.Errors[0].Message)
	}
}

func TestSarifParser_EmptyStdout_ZeroExit(t *testing.T) {
	p := NewSarifParser()
	res, err := p.Parse(context.Background(), []byte(""), nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Passed {
		t.Error("expected passed")
	}
}
