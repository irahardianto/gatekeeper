package llm

import (
	"context"

	"github.com/irahardianto/gatekeeper/internal/engine/parser"
)

// MockClient is a test double for llm.Client.
type MockClient struct {
	Result []parser.StructuredError
	Err    error
}

// Review returns the configured result and error.
func (m *MockClient) Review(_ context.Context, _ string) ([]parser.StructuredError, error) {
	return m.Result, m.Err
}
