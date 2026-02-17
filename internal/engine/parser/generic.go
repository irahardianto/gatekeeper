package parser

import (
	"context"
	"strings"
)

// GenericParser is a fallback parser for tools without structured output.
// It relies solely on the exit code.
type GenericParser struct{}

// NewGenericParser creates a new GenericParser.
func NewGenericParser() *GenericParser {
	return &GenericParser{}
}

// Parse implements the Parser interface.
// If exitCode is 0, returns passed.
// If exitCode is non-zero, returns failed with stderr (or stdout) as message.
func (p *GenericParser) Parse(ctx context.Context, stdout, stderr []byte, exitCode int) (*ParseResult, error) {
	if exitCode == 0 {
		return &ParseResult{Passed: true}, nil
	}

	// Capture output for the error message
	msg := string(stderr)
	if strings.TrimSpace(msg) == "" {
		msg = string(stdout)
	}
	if strings.TrimSpace(msg) == "" {
		msg = "Tool failed with no output"
	}

	return &ParseResult{
		Passed: false,
		Errors: []StructuredError{
			{
				Severity: "error",
				Message:  strings.TrimSpace(msg),
				Tool:     "generic",
			},
		},
	}, nil
}
