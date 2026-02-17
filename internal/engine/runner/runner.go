// Package runner provides the parallel execution engine for running gates.
package runner

import (
	"context"
	"sync"
	"time"

	"github.com/irahardianto/gatekeeper/internal/engine/formatter"
	"github.com/irahardianto/gatekeeper/internal/engine/gate"
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
)

// Engine orchestrates parallel gate execution.
type Engine struct {
	// Progress is an optional progress tracker. If nil, no progress output is produced.
	Progress *Progress
}

// NewEngine creates a new execution engine.
func NewEngine() *Engine {
	return &Engine{}
}

// NewEngineWithProgress creates a new execution engine with progress tracking.
func NewEngineWithProgress(p *Progress) *Engine {
	return &Engine{Progress: p}
}

// RunAll executes all gates in parallel and collects results.
// If failFast is true, remaining gates are cancelled when a blocking gate fails.
// gateNames provides human-readable names for progress tracking (must match gates length).
func (e *Engine) RunAll(ctx context.Context, gates []gate.Gate, failFast bool, gateNames []string) (*formatter.RunResult, error) {
	log := logger.FromContext(ctx)
	log.Info("Engine.RunAll started", "gates", len(gates), "fail_fast", failFast)
	start := time.Now()

	if len(gates) == 0 {
		return &formatter.RunResult{Passed: true, DurationMs: 0}, nil
	}

	// Create a cancellable context for fail-fast support.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Fan-out: run each gate in its own goroutine.
	type indexedResult struct {
		idx    int
		result *formatter.GateResult
		err    error
	}

	resultsCh := make(chan indexedResult, len(gates))
	var wg sync.WaitGroup

	for i, g := range gates {
		wg.Add(1)
		go func(idx int, g gate.Gate) {
			defer wg.Done()

			// Check if context is already cancelled before starting.
			select {
			case <-ctx.Done():
				return
			default:
			}

			if e.Progress != nil && idx < len(gateNames) {
				e.Progress.OnStart(gateNames[idx])
			}

			gateStart := time.Now()
			result, err := g.Execute(ctx)
			gateDur := time.Since(gateStart)
			resultsCh <- indexedResult{idx: idx, result: result, err: err}

			if e.Progress != nil && result != nil {
				e.Progress.OnComplete(result.Name, result.Passed, result.SystemError != "", result.SystemError, gateDur)
			}

			// Fail-fast: cancel remaining gates if a blocking gate failed.
			if failFast && result != nil && !result.Passed && result.Blocking {
				log.Info("fail-fast: cancelling remaining gates", "failed_gate", result.Name)
				cancel()
			}
		}(i, g)
	}

	// Close channel when all goroutines complete.
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Collect results in order.
	collected := make([]*formatter.GateResult, len(gates))
	for ir := range resultsCh {
		if ir.result != nil {
			collected[ir.idx] = ir.result
		} else if ir.err != nil {
			// System error — create a placeholder result.
			collected[ir.idx] = &formatter.GateResult{
				SystemError: ir.err.Error(),
			}
		}
	}

	// Build RunResult.
	runResult := &formatter.RunResult{
		Passed:     true,
		DurationMs: time.Since(start).Milliseconds(),
	}

	for _, r := range collected {
		if r == nil {
			// Gate was cancelled before completion (fail-fast).
			continue
		}
		runResult.Gates = append(runResult.Gates, *r)

		// A run fails if any blocking gate failed or had a system error with block policy.
		if r.Blocking && (!r.Passed || r.SystemError != "") {
			runResult.Passed = false
		}
	}

	// Print progress summary.
	if e.Progress != nil {
		e.Progress.Finish()
	}

	log.Info("Engine.RunAll completed", "passed", runResult.Passed, "duration_ms", runResult.DurationMs, "gates_run", len(runResult.Gates))
	return runResult, nil
}
