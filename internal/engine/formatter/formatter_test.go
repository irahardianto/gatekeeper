package formatter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/irahardianto/gatekeeper/internal/engine/parser"
)

func sampleResult() RunResult {
	return RunResult{
		Passed:     false,
		DurationMs: 1200,
		Gates: []GateResult{
			{
				Name:       "lint",
				Type:       "exec",
				Passed:     true,
				Blocking:   true,
				DurationMs: 800,
			},
			{
				Name:       "security",
				Type:       "exec",
				Passed:     false,
				Blocking:   true,
				DurationMs: 400,
				Errors: []parser.StructuredError{
					{
						File:     "main.go",
						Line:     42,
						Column:   10,
						Severity: "error",
						Rule:     "G101",
						Message:  "hardcoded credential",
						Hint:     "Use environment variables instead.",
						Tool:     "gosec",
					},
				},
			},
			{
				Name:        "format",
				Type:        "exec",
				Passed:      false,
				Blocking:    false,
				DurationMs:  100,
				SystemError: "container timeout",
			},
			{
				Name:       "style",
				Type:       "exec",
				Passed:     true,
				Blocking:   false,
				Skipped:    true,
				DurationMs: 0,
			},
		},
	}
}

// --- JSON Formatter Tests ---

func TestJSONFormatter_ValidJSON(t *testing.T) {
	f := NewJSONFormatter()
	output := f.Format(sampleResult())

	var parsed RunResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput:\n%s", err, output)
	}

	if parsed.Passed {
		t.Error("expected Passed=false")
	}
	if parsed.DurationMs != 1200 {
		t.Errorf("expected DurationMs=1200, got %d", parsed.DurationMs)
	}
	if len(parsed.Gates) != 4 {
		t.Errorf("expected 4 gates, got %d", len(parsed.Gates))
	}
}

func TestJSONFormatter_ErrorFields(t *testing.T) {
	f := NewJSONFormatter()
	output := f.Format(sampleResult())

	if !strings.Contains(output, `"file": "main.go"`) {
		t.Error("expected JSON to contain file field")
	}
	if !strings.Contains(output, `"rule": "G101"`) {
		t.Error("expected JSON to contain rule field")
	}
	if !strings.Contains(output, `"hint": "Use environment variables instead."`) {
		t.Error("expected JSON to contain hint field")
	}
}

func TestJSONFormatter_SystemError(t *testing.T) {
	f := NewJSONFormatter()
	output := f.Format(sampleResult())

	if !strings.Contains(output, `"system_error": "container timeout"`) {
		t.Error("expected JSON to contain system_error field")
	}
}

func TestJSONFormatter_EmptyGates(t *testing.T) {
	f := NewJSONFormatter()
	result := RunResult{Passed: true, DurationMs: 50}
	output := f.Format(result)

	if !strings.Contains(output, `"passed": true`) {
		t.Error("expected passed=true in output")
	}
}

// --- CLI Formatter Tests ---

func TestCLIFormatter_ContainsGateNames(t *testing.T) {
	f := NewCLIFormatter(false, false)
	output := f.Format(sampleResult())

	for _, name := range []string{"lint", "security", "format", "style"} {
		if !strings.Contains(output, name) {
			t.Errorf("expected output to contain gate name %q", name)
		}
	}
}

func TestCLIFormatter_PassFailIcons(t *testing.T) {
	f := NewCLIFormatter(false, false)
	output := f.Format(sampleResult())

	if !strings.Contains(output, "✅") {
		t.Error("expected output to contain ✅ icon for passing gate")
	}
	if !strings.Contains(output, "❌") {
		t.Error("expected output to contain ❌ icon for failing gate")
	}
	if !strings.Contains(output, "💥") {
		t.Error("expected output to contain 💥 icon for system error")
	}
	if !strings.Contains(output, "⏭️") {
		t.Error("expected output to contain ⏭️ icon for skipped gate")
	}
}

func TestCLIFormatter_ErrorDetails(t *testing.T) {
	f := NewCLIFormatter(false, false)
	output := f.Format(sampleResult())

	if !strings.Contains(output, "main.go:42:10") {
		t.Error("expected output to contain file:line:col location")
	}
	if !strings.Contains(output, "hardcoded credential") {
		t.Error("expected output to contain error message")
	}
	if !strings.Contains(output, "[G101]") {
		t.Error("expected output to contain rule ID")
	}
	if !strings.Contains(output, "💡") {
		t.Error("expected output to contain hint icon")
	}
	if !strings.Contains(output, "Use environment variables instead.") {
		t.Error("expected output to contain hint text")
	}
}

func TestCLIFormatter_NoColorMode(t *testing.T) {
	f := NewCLIFormatter(false, false)
	output := f.Format(sampleResult())

	if strings.Contains(output, "\033[") {
		t.Error("expected no ANSI escape codes in no-color mode")
	}
}

func TestCLIFormatter_ColorMode(t *testing.T) {
	f := NewCLIFormatter(true, false)
	output := f.Format(sampleResult())

	if !strings.Contains(output, "\033[") {
		t.Error("expected ANSI escape codes in color mode")
	}
}

func TestCLIFormatter_VerboseMode(t *testing.T) {
	result := RunResult{
		Passed:     true,
		DurationMs: 100,
		Gates: []GateResult{
			{
				Name:       "lint",
				Type:       "exec",
				Passed:     true,
				Blocking:   true,
				DurationMs: 100,
				RawOutput:  "all checks passed\nno issues found",
			},
		},
	}

	// Without verbose
	fQuiet := NewCLIFormatter(false, false)
	quiet := fQuiet.Format(result)
	if strings.Contains(quiet, "all checks passed") {
		t.Error("expected no raw output in non-verbose mode")
	}

	// With verbose
	fVerbose := NewCLIFormatter(false, true)
	verbose := fVerbose.Format(result)
	if !strings.Contains(verbose, "all checks passed") {
		t.Error("expected raw output in verbose mode")
	}
	if !strings.Contains(verbose, "raw output") {
		t.Error("expected raw output header in verbose mode")
	}
}

func TestCLIFormatter_WarningAndInfoSeverity(t *testing.T) {
	result := RunResult{
		Passed:     true,
		DurationMs: 100,
		Gates: []GateResult{
			{
				Name:       "check",
				Passed:     true,
				DurationMs: 50,
				Errors: []parser.StructuredError{
					{Severity: "warning", Message: "might be wrong"},
					{Severity: "info", Message: "just a note"},
				},
			},
		},
	}

	f := NewCLIFormatter(false, false)
	output := f.Format(result)

	if !strings.Contains(output, "⚠️") {
		t.Error("expected warning icon")
	}
	if !strings.Contains(output, "ℹ️") {
		t.Error("expected info icon")
	}
}
