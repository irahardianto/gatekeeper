package parser

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

// SarifParser parses SARIF v2.1.0 JSON output.
type SarifParser struct{}

// NewSarifParser creates a new SarifParser.
func NewSarifParser() *SarifParser {
	return &SarifParser{}
}

// Parse implements the Parser interface.
func (p *SarifParser) Parse(ctx context.Context, stdout, stderr []byte, exitCode int) (*ParseResult, error) {
	// 1. Fail-closed on empty stdout with non-zero exit code
	if len(bytes.TrimSpace(stdout)) == 0 {
		if exitCode != 0 {
			msg := string(stderr)
			if msg == "" {
				msg = "Tool failed with non-zero exit code and empty stdout"
			}
			return &ParseResult{
				Passed: false,
				Errors: []StructuredError{
					{
						Severity: "error",
						Message:  msg,
						Tool:     "sarif",
					},
				},
			}, nil
		}
		// Empty stdout and exit code 0 usually means success
		return &ParseResult{Passed: true}, nil
	}

	// 2. Parse SARIF
	report, err := sarif.FromBytes(stdout)
	if err != nil {
		return nil, fmt.Errorf("parsing SARIF JSON: %w", err)
	}

	var errors []StructuredError
	failed := false

	// Iterate over runs and results
	for _, run := range report.Runs {
		toolName := run.Tool.Driver.Name

		for _, outcome := range run.Results {
			// Determine severity
			algoLevel := "info"
			if outcome.Level != nil {
				algoLevel = *outcome.Level
			}

			// Map SARIF level to our severity
			severity := "info"
			switch strings.ToLower(algoLevel) {
			case "error":
				severity = "error"
				failed = true
			case "warning":
				severity = "warning"
			case "note", "none":
				severity = "info"
			}

			// Extract rule ID
			ruleID := ""
			if outcome.RuleID != nil {
				ruleID = *outcome.RuleID
			}

			// Extract message
			msg := ""
			if outcome.Message.Text != nil {
				msg = *outcome.Message.Text
			}

			// Extract location
			file := ""
			line := 0
			col := 0
			if len(outcome.Locations) > 0 {
				loc := outcome.Locations[0]
				if loc.PhysicalLocation != nil {
					if loc.PhysicalLocation.ArtifactLocation != nil && loc.PhysicalLocation.ArtifactLocation.URI != nil {
						file = *loc.PhysicalLocation.ArtifactLocation.URI
					}
					if loc.PhysicalLocation.Region != nil {
						if loc.PhysicalLocation.Region.StartLine != nil {
							line = *loc.PhysicalLocation.Region.StartLine
						}
						if loc.PhysicalLocation.Region.StartColumn != nil {
							col = *loc.PhysicalLocation.Region.StartColumn
						}
					}
				}
			}

			errors = append(errors, StructuredError{
				File:     file,
				Line:     line,
				Column:   col,
				Severity: severity,
				Rule:     ruleID,
				Message:  msg,
				Tool:     toolName,
			})
		}
	}

	return &ParseResult{
		Passed: !failed,
		Errors: errors,
	}, nil
}
