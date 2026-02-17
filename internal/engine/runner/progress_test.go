package runner

import (
	"bytes"
	"testing"
	"time"
)

func TestProgress_Suppressed(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, true, 3)

	p.OnStart("lint")
	p.OnComplete("lint", true, false, "", 800*time.Millisecond)
	p.Finish()

	if buf.Len() != 0 {
		t.Errorf("expected no output in suppressed mode, got: %q", buf.String())
	}
}

func TestProgress_GateTransitions(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, false, 2)

	p.OnStart("lint")
	p.OnComplete("lint", true, false, "", 800*time.Millisecond)

	output := buf.String()
	if output == "" {
		t.Error("expected progress output, got empty string")
	}
}

func TestProgress_FailedGate(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, false, 2)

	p.OnStart("security")
	p.OnComplete("security", false, false, "", 500*time.Millisecond)
	p.Finish()

	output := buf.String()
	if output == "" {
		t.Error("expected progress output for failed gate")
	}
}

func TestProgress_SystemError(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, false, 1)

	p.OnStart("lint")
	p.OnComplete("lint", false, true, "container crashed", 500*time.Millisecond)
	p.Finish()

	output := buf.String()
	if output == "" {
		t.Error("expected progress output for system error")
	}
}

func TestProgress_Header(t *testing.T) {
	var buf bytes.Buffer
	_ = NewProgress(&buf, false, 3)

	// The header should include the gate count.
	output := buf.String()
	if len(output) == 0 {
		t.Error("expected header output on creation")
	}
}

func TestProgress_MultipleGates(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, false, 3)

	p.OnStart("lint")
	p.OnStart("test")
	p.OnStart("security")

	p.OnComplete("lint", true, false, "", 800*time.Millisecond)
	p.OnComplete("test", true, false, "", 1200*time.Millisecond)
	p.OnComplete("security", false, false, "", 500*time.Millisecond)
	p.Finish()

	output := buf.String()
	if output == "" {
		t.Error("expected progress output for multiple gates")
	}
}
