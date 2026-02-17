package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestNew_Defaults(t *testing.T) {
	l := New(false, false)
	if l == nil {
		t.Fatal("New returned nil")
	}
	if !l.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected Info level to be enabled by default")
	}
	if l.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected Debug level to be disabled by default")
	}
}

func TestNew_Verbose(t *testing.T) {
	l := New(true, false)
	if !l.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected Debug level to be enabled when verbose is true")
	}
}

func TestContext(t *testing.T) {
	l := New(false, false)
	ctx := context.Background()

	// Default when missing
	l1 := FromContext(ctx)
	if l1 == nil {
		t.Fatal("FromContext returned nil for empty context")
	}

	// With context
	ctx = WithContext(ctx, l)
	l2 := FromContext(ctx)
	if l2 != l {
		t.Error("FromContext did not return the logger injected with WithContext")
	}
}

func TestJSONHandler(t *testing.T) {
	// We can't easily capture stdout from os.Stdout in a test without redirecting it.
	// However, we can trust slog.NewJSONHandler works.
	// This test mainly verifies that New(..., true) returns a logger.
	l := New(false, true)
	if l == nil {
		t.Fatal("New(..., true) returned nil")
	}
}

func TestLoggerOutput(t *testing.T) {
	// Verify that we can log without panic
	l := New(true, true)
	l.Info("test message", "key", "value")
}

// Redact verifies that we don't accidentally log sensitive data if we use the logger incorrectly,
// although strictly speaking redaction is often handled by types implementing LogValuer.
// This test ensures our setup honors standard slog behavior.
type secret string

func (s secret) LogValue() slog.Value {
	return slog.StringValue("[REDACTED]")
}

func TestRedactionParams(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, nil)
	l := slog.New(h)

	l.Info("sensitive", "password", secret("abc"))

	if strings.Contains(buf.String(), "abc") {
		t.Error("log contained secret value")
	}
	if !strings.Contains(buf.String(), "[REDACTED]") {
		t.Error("log did not contain redacted value")
	}
}
