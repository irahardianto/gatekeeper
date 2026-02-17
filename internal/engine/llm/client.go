// Package llm provides LLM-powered code review capabilities.
package llm

import (
	"context"

	"github.com/irahardianto/gatekeeper/internal/engine/parser"
)

// Client abstracts LLM API interaction for testability.
type Client interface {
	// Review sends a prompt to the LLM and returns structured errors.
	Review(ctx context.Context, prompt string) ([]parser.StructuredError, error)
}
