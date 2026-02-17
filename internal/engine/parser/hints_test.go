package parser

import (
	"testing"
)

func TestEnrichHints_KnownRule(t *testing.T) {
	errors := []StructuredError{
		{Rule: "G101", Message: "hardcoded credential"},
	}
	result := EnrichHints(errors)
	if result[0].Hint == "" {
		t.Error("expected hint to be populated for known rule G101")
	}
	if result[0].Hint != hintDatabase["G101"] {
		t.Errorf("expected hint %q, got %q", hintDatabase["G101"], result[0].Hint)
	}
}

func TestEnrichHints_UnknownRule(t *testing.T) {
	errors := []StructuredError{
		{Rule: "UNKNOWN_RULE_XYZ", Message: "something"},
	}
	result := EnrichHints(errors)
	if result[0].Hint != "" {
		t.Errorf("expected empty hint for unknown rule, got %q", result[0].Hint)
	}
}

func TestEnrichHints_PreservesExistingHint(t *testing.T) {
	existing := "Custom hint from tool"
	errors := []StructuredError{
		{Rule: "G101", Hint: existing},
	}
	result := EnrichHints(errors)
	if result[0].Hint != existing {
		t.Errorf("expected existing hint %q to be preserved, got %q", existing, result[0].Hint)
	}
}

func TestEnrichHints_EmptySlice(t *testing.T) {
	result := EnrichHints(nil)
	if result != nil {
		t.Errorf("expected nil result for nil input, got %v", result)
	}
}

func TestEnrichHints_MixedRules(t *testing.T) {
	errors := []StructuredError{
		{Rule: "G201", Message: "sql injection"},
		{Rule: "no-unused-vars", Message: "unused var"},
		{Rule: "NONEXISTENT", Message: "unknown"},
		{Rule: "B608", Hint: "already has hint"},
	}
	result := EnrichHints(errors)

	if result[0].Hint != hintDatabase["G201"] {
		t.Errorf("G201: expected hint, got %q", result[0].Hint)
	}
	if result[1].Hint != hintDatabase["no-unused-vars"] {
		t.Errorf("no-unused-vars: expected hint, got %q", result[1].Hint)
	}
	if result[2].Hint != "" {
		t.Errorf("NONEXISTENT: expected empty hint, got %q", result[2].Hint)
	}
	if result[3].Hint != "already has hint" {
		t.Errorf("B608: expected preserved hint, got %q", result[3].Hint)
	}
}
