package parser

import (
	"context"
	"testing"
)

type mockParser struct{}

func (m *mockParser) Parse(ctx context.Context, stdout, stderr []byte, exitCode int) (*ParseResult, error) {
	return &ParseResult{Passed: true}, nil
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	mock := &mockParser{}

	r.Register("mock", mock)

	if r.Get("mock") != mock {
		t.Error("expected to retrieve registered parser")
	}

	if r.Get("unknown") != nil {
		t.Error("expected nil for unknown parser")
	}

	// Test GetOrDefault
	if r.GetOrDefault("mock") != mock {
		t.Error("expected GetOrDefault to return specific parser")
	}

	def := r.GetOrDefault("unknown")
	if _, ok := def.(*GenericParser); !ok {
		t.Errorf("expected GenericParser fallback, got %T", def)
	}
}

func TestGenericParser_Pass(t *testing.T) {
	p := NewGenericParser()
	res, err := p.Parse(context.Background(), []byte("out"), []byte("err"), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Passed {
		t.Error("expected passed")
	}
	if len(res.Errors) > 0 {
		t.Error("expected no errors")
	}
}

func TestGenericParser_Fail(t *testing.T) {
	p := NewGenericParser()
	res, err := p.Parse(context.Background(), []byte("stdout info"), []byte("stderr error"), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Passed {
		t.Error("expected failed")
	}
	if len(res.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(res.Errors))
	}

	expectedMsg := "stderr error"
	if res.Errors[0].Message != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, res.Errors[0].Message)
	}
}

func TestGenericParser_Fail_FallbackStdout(t *testing.T) {
	p := NewGenericParser()
	res, err := p.Parse(context.Background(), []byte("stdout error"), []byte(""), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Errors[0].Message != "stdout error" {
		t.Errorf("expected fallback to stdout, got %q", res.Errors[0].Message)
	}
}
