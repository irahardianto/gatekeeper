// Package formatter handles formatting gate results for CLI and JSON output.
package formatter

import (
	"github.com/irahardianto/gatekeeper/internal/engine/parser"
)

// GateResult holds the result of executing a single gate.
type GateResult struct {
	Name        string                   `json:"name"`
	Type        string                   `json:"type"`
	Passed      bool                     `json:"passed"`
	Blocking    bool                     `json:"blocking"`
	Skipped     bool                     `json:"skipped,omitempty"`
	DurationMs  int64                    `json:"duration_ms"`
	Errors      []parser.StructuredError `json:"errors,omitempty"`
	SystemError string                   `json:"system_error,omitempty"`
	RawOutput   string                   `json:"raw_output,omitempty"`
}

// RunResult holds the aggregated result of all gates in a run.
type RunResult struct {
	Passed     bool         `json:"passed"`
	DurationMs int64        `json:"duration_ms"`
	Gates      []GateResult `json:"gates"`
}

// Formatter formats a RunResult into a human-readable or machine-readable string.
type Formatter interface {
	Format(result RunResult) string
}
