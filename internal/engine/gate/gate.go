// Package gate defines the unified gate interface and implementations for
// exec, script, and LLM gate types.
package gate

import (
	"context"

	"github.com/irahardianto/gatekeeper/internal/engine/formatter"
)

// Gate represents a single validation gate that can be executed.
type Gate interface {
	// Execute runs the gate and returns the result.
	Execute(ctx context.Context) (*formatter.GateResult, error)
}
