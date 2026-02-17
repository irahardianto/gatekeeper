package parser

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoTestParser_AllPassing(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "gotest_pass.json"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	p := NewGoTestParser()
	res, err := p.Parse(context.Background(), data, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Passed {
		t.Error("expected passed")
	}
	if len(res.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(res.Errors))
	}
}

func TestGoTestParser_SomeFailures(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "gotest_fail.json"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	p := NewGoTestParser()
	res, err := p.Parse(context.Background(), data, nil, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Passed {
		t.Error("expected failed")
	}

	// Should have 2 failing tests: TestDivide and TestMultiply.
	if len(res.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(res.Errors))
	}

	// Verify first failure.
	e1 := res.Errors[0]
	if e1.Severity != "error" {
		t.Errorf("expected severity error, got %q", e1.Severity)
	}
	if e1.Tool != "go-test" {
		t.Errorf("expected tool go-test, got %q", e1.Tool)
	}
	if !strings.Contains(e1.Message, "expected 2, got 3") {
		t.Errorf("expected message to contain failure output, got %q", e1.Message)
	}

	// Verify second failure.
	e2 := res.Errors[1]
	if !strings.Contains(e2.Message, "wrong result") {
		t.Errorf("expected message to contain 'wrong result', got %q", e2.Message)
	}
}

func TestGoTestParser_EmptyStdout_NonZeroExit(t *testing.T) {
	p := NewGoTestParser()
	res, err := p.Parse(context.Background(), []byte("   "), []byte("build failed"), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Passed {
		t.Error("expected failed")
	}
	if len(res.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(res.Errors))
	}
	if res.Errors[0].Message != "build failed" {
		t.Errorf("expected stderr in message, got %q", res.Errors[0].Message)
	}
}

func TestGoTestParser_EmptyStdout_ZeroExit(t *testing.T) {
	p := NewGoTestParser()
	res, err := p.Parse(context.Background(), []byte(""), nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Passed {
		t.Error("expected passed")
	}
}

func TestGoTestParser_MalformedJSON(t *testing.T) {
	data := []byte("not json at all\n{invalid}\n")
	p := NewGoTestParser()
	_, err := p.Parse(context.Background(), data, nil, 1)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestGoTestParser_PartialMalformedJSON(t *testing.T) {
	// Valid event followed by garbage.
	data := []byte(`{"Action":"start","Package":"example.com/foo"}` + "\n" + "garbage\n")
	p := NewGoTestParser()
	_, err := p.Parse(context.Background(), data, nil, 1)
	if err == nil {
		t.Fatal("expected error for partially malformed JSON")
	}
}

func TestGoTestParser_BuildError(t *testing.T) {
	// Build errors produce a single-line output action followed by a fail action.
	data := []byte(`{"Action":"output","Package":"example.com/myapp","Output":"# example.com/myapp\n"}
{"Action":"output","Package":"example.com/myapp","Output":"./main.go:5:6: undefined: foo\n"}
{"Action":"fail","Package":"example.com/myapp","Elapsed":0}
`)
	p := NewGoTestParser()
	res, err := p.Parse(context.Background(), data, nil, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Passed {
		t.Error("expected failed")
	}
	if len(res.Errors) != 1 {
		t.Fatalf("expected 1 error (package-level failure), got %d", len(res.Errors))
	}
	if !strings.Contains(res.Errors[0].Message, "undefined: foo") {
		t.Errorf("expected build error message, got %q", res.Errors[0].Message)
	}
	if res.Errors[0].File != "example.com/myapp" {
		t.Errorf("expected package as file, got %q", res.Errors[0].File)
	}
}

func TestGoTestParser_SkippedTests(t *testing.T) {
	data := []byte(`{"Action":"start","Package":"example.com/myapp"}
{"Action":"run","Package":"example.com/myapp","Test":"TestSkipped"}
{"Action":"output","Package":"example.com/myapp","Test":"TestSkipped","Output":"=== RUN   TestSkipped\n"}
{"Action":"output","Package":"example.com/myapp","Test":"TestSkipped","Output":"--- SKIP: TestSkipped (0.00s)\n"}
{"Action":"skip","Package":"example.com/myapp","Test":"TestSkipped","Elapsed":0}
{"Action":"pass","Package":"example.com/myapp","Elapsed":0.001}
`)
	p := NewGoTestParser()
	res, err := p.Parse(context.Background(), data, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Passed {
		t.Error("expected passed (skipped tests are not failures)")
	}
	if len(res.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(res.Errors))
	}
}
