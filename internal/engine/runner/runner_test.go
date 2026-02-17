package runner

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/irahardianto/gatekeeper/internal/engine/formatter"
	"github.com/irahardianto/gatekeeper/internal/engine/gate"
)

// --- Mock Gate for testing ---

type mockGate struct {
	result   *formatter.GateResult
	err      error
	delay    time.Duration
	executed atomic.Bool
}

func (m *mockGate) Execute(ctx context.Context) (*formatter.GateResult, error) {
	m.executed.Store(true)

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return m.result, m.err
}

func newPassGate(name string) *mockGate {
	return &mockGate{
		result: &formatter.GateResult{
			Name:     name,
			Type:     "exec",
			Passed:   true,
			Blocking: true,
		},
	}
}

func newFailGate(name string, blocking bool) *mockGate {
	return &mockGate{
		result: &formatter.GateResult{
			Name:     name,
			Type:     "exec",
			Passed:   false,
			Blocking: blocking,
			Errors:   nil,
		},
	}
}

func newSlowGate(name string, delay time.Duration) *mockGate {
	return &mockGate{
		result: &formatter.GateResult{
			Name:     name,
			Type:     "exec",
			Passed:   true,
			Blocking: true,
		},
		delay: delay,
	}
}

func newErrorGate(name string) *mockGate {
	return &mockGate{
		err: errors.New("system error"),
	}
}

// --- Tests ---

func TestRunAll_Empty(t *testing.T) {
	engine := NewEngine()
	result, err := engine.RunAll(context.Background(), nil, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass for empty gates")
	}
}

func TestRunAll_AllPass(t *testing.T) {
	engine := NewEngine()
	gates := []gate.Gate{
		newPassGate("lint"),
		newPassGate("test"),
		newPassGate("vet"),
	}

	result, err := engine.RunAll(context.Background(), gates, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass when all gates pass")
	}
	if len(result.Gates) != 3 {
		t.Errorf("expected 3 gate results, got %d", len(result.Gates))
	}
}

func TestRunAll_BlockingFail(t *testing.T) {
	engine := NewEngine()
	gates := []gate.Gate{
		newPassGate("lint"),
		newFailGate("test", true), // blocking
		newPassGate("vet"),
	}

	result, err := engine.RunAll(context.Background(), gates, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail when a blocking gate fails")
	}
}

func TestRunAll_NonBlockingFail(t *testing.T) {
	engine := NewEngine()
	gates := []gate.Gate{
		newPassGate("lint"),
		newFailGate("review", false), // non-blocking
		newPassGate("vet"),
	}

	result, err := engine.RunAll(context.Background(), gates, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass when only non-blocking gate fails")
	}
}

func TestRunAll_FailFast(t *testing.T) {
	engine := NewEngine()

	failGate := newFailGate("lint", true)
	slowGate := newSlowGate("slow_test", 2*time.Second)

	gates := []gate.Gate{
		failGate,
		slowGate,
	}

	start := time.Now()
	result, err := engine.RunAll(context.Background(), gates, true, nil)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail with fail-fast")
	}

	// With fail-fast, should complete much faster than the slow gate's 2s delay.
	if duration > 1*time.Second {
		t.Errorf("expect fail-fast to complete quickly, took %v", duration)
	}
}

func TestRunAll_SystemError(t *testing.T) {
	engine := NewEngine()
	gates := []gate.Gate{
		newPassGate("lint"),
		newErrorGate("broken"),
	}

	result, err := engine.RunAll(context.Background(), gates, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// System errors should be captured in gate results.
	found := false
	for _, g := range result.Gates {
		if g.SystemError != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one gate with system error")
	}
}

func TestRunAll_ParallelExecution(t *testing.T) {
	engine := NewEngine()

	// Three gates each taking 100ms — should complete in ~100ms if parallel.
	gates := []gate.Gate{
		newSlowGate("gate1", 100*time.Millisecond),
		newSlowGate("gate2", 100*time.Millisecond),
		newSlowGate("gate3", 100*time.Millisecond),
	}

	start := time.Now()
	result, err := engine.RunAll(context.Background(), gates, false, nil)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass")
	}

	// Sequential would take ~300ms. Parallel should take ~100ms + overhead.
	if duration > 250*time.Millisecond {
		t.Errorf("expected parallel execution (~100ms), took %v", duration)
	}
}

func TestRunAll_DurationTracked(t *testing.T) {
	engine := NewEngine()
	gates := []gate.Gate{
		newPassGate("lint"),
	}

	result, err := engine.RunAll(context.Background(), gates, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DurationMs < 0 {
		t.Errorf("expected non-negative duration, got %d", result.DurationMs)
	}
}

func TestRunAll_ContextCancelled(t *testing.T) {
	engine := NewEngine()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	gates := []gate.Gate{
		newSlowGate("gate1", 1*time.Second),
	}

	_, err := engine.RunAll(ctx, gates, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunAll_SystemErrorBlocking(t *testing.T) {
	engine := NewEngine()

	// A system error on a blocking gate should mark the run as failed.
	errGate := &mockGate{
		result: &formatter.GateResult{
			Name:        "docker_check",
			Type:        "exec",
			Blocking:    true,
			SystemError: "container crashed",
		},
	}

	gates := []gate.Gate{errGate}
	result, err := engine.RunAll(context.Background(), gates, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail when blocking gate has system error")
	}
}

func TestRunAll_ProgressFinishCalled(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(&buf, false, 2)

	engine := NewEngineWithProgress(progress)
	gates := []gate.Gate{
		newPassGate("lint"),
		newPassGate("test"),
	}

	_, err := engine.RunAll(context.Background(), gates, false, []string{"lint", "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "All 2 gate(s) passed") {
		t.Errorf("expected Finish() summary in output, got %q", output)
	}
}
