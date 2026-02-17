package commands

import (
	"testing"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
)

func TestFilterSkippedGates_NoFilters(t *testing.T) {
	gates := []config.Gate{
		{Name: "lint", Type: config.GateTypeExec},
		{Name: "test", Type: config.GateTypeExec},
		{Name: "review", Type: config.GateTypeLLM},
	}

	result := filterSkippedGates(gates, nil, false)
	if len(result) != 3 {
		t.Errorf("expected 3 gates (no filter), got %d", len(result))
	}
}

func TestFilterSkippedGates_SkipByName(t *testing.T) {
	gates := []config.Gate{
		{Name: "lint", Type: config.GateTypeExec},
		{Name: "test", Type: config.GateTypeExec},
		{Name: "review", Type: config.GateTypeLLM},
	}

	result := filterSkippedGates(gates, []string{"lint"}, false)
	if len(result) != 2 {
		t.Fatalf("expected 2 gates, got %d", len(result))
	}
	for _, g := range result {
		if g.Name == "lint" {
			t.Error("expected 'lint' to be skipped")
		}
	}
}

func TestFilterSkippedGates_SkipMultipleByName(t *testing.T) {
	gates := []config.Gate{
		{Name: "lint", Type: config.GateTypeExec},
		{Name: "test", Type: config.GateTypeExec},
		{Name: "review", Type: config.GateTypeLLM},
	}

	result := filterSkippedGates(gates, []string{"lint", "test"}, false)
	if len(result) != 1 {
		t.Fatalf("expected 1 gate, got %d", len(result))
	}
	if result[0].Name != "review" {
		t.Errorf("expected 'review', got %q", result[0].Name)
	}
}

func TestFilterSkippedGates_SkipLLM(t *testing.T) {
	gates := []config.Gate{
		{Name: "lint", Type: config.GateTypeExec},
		{Name: "test", Type: config.GateTypeExec},
		{Name: "review", Type: config.GateTypeLLM},
	}

	result := filterSkippedGates(gates, nil, true)
	if len(result) != 2 {
		t.Fatalf("expected 2 gates, got %d", len(result))
	}
	for _, g := range result {
		if g.Type == config.GateTypeLLM {
			t.Error("expected LLM gates to be skipped")
		}
	}
}

func TestFilterSkippedGates_SkipBoth(t *testing.T) {
	gates := []config.Gate{
		{Name: "lint", Type: config.GateTypeExec},
		{Name: "test", Type: config.GateTypeExec},
		{Name: "review", Type: config.GateTypeLLM},
	}

	result := filterSkippedGates(gates, []string{"lint"}, true)
	if len(result) != 1 {
		t.Fatalf("expected 1 gate, got %d", len(result))
	}
	if result[0].Name != "test" {
		t.Errorf("expected 'test', got %q", result[0].Name)
	}
}

func TestFilterSkippedGates_Empty(t *testing.T) {
	result := filterSkippedGates(nil, nil, false)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestFilterSkippedGates_SkipAll(t *testing.T) {
	gates := []config.Gate{
		{Name: "lint", Type: config.GateTypeExec},
	}

	result := filterSkippedGates(gates, []string{"lint"}, false)
	if len(result) != 0 {
		t.Errorf("expected 0 gates after skipping all, got %d", len(result))
	}
}

func TestFilterSkippedGates_NonexistentSkip(t *testing.T) {
	gates := []config.Gate{
		{Name: "lint", Type: config.GateTypeExec},
		{Name: "test", Type: config.GateTypeExec},
	}

	// Skipping a gate that doesn't exist should have no effect.
	result := filterSkippedGates(gates, []string{"nonexistent"}, false)
	if len(result) != 2 {
		t.Errorf("expected 2 gates (skip nonexistent has no effect), got %d", len(result))
	}
}

// --- formatStacks tests ---

func TestFormatStacks_Single(t *testing.T) {
	result := formatStacks([]config.Stack{config.StackGo})
	if result != "go" {
		t.Errorf("expected 'go', got %q", result)
	}
}

func TestFormatStacks_Multiple(t *testing.T) {
	result := formatStacks([]config.Stack{config.StackGo, config.StackNode})
	if result != "go + node" {
		t.Errorf("expected 'go + node', got %q", result)
	}
}
