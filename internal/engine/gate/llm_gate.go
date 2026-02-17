package gate

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
	"github.com/irahardianto/gatekeeper/internal/engine/formatter"
	"github.com/irahardianto/gatekeeper/internal/engine/git"
	"github.com/irahardianto/gatekeeper/internal/engine/llm"
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
)

// LLMGate sends staged diffs to an LLM for semantic code review.
type LLMGate struct {
	cfg    config.Gate
	client llm.Client
	gitSvc git.Service
}

// NewLLMGate creates a new LLMGate.
func NewLLMGate(cfg config.Gate, client llm.Client, gitSvc git.Service) *LLMGate {
	return &LLMGate{
		cfg:    cfg,
		client: client,
		gitSvc: gitSvc,
	}
}

// Execute extracts staged diffs, sends them to the LLM, and returns structured results.
func (g *LLMGate) Execute(ctx context.Context) (*formatter.GateResult, error) {
	log := logger.FromContext(ctx)
	log.Info("LLMGate.Execute started", "gate", g.cfg.Name, "provider", g.cfg.Provider)
	start := time.Now()

	result := &formatter.GateResult{
		Name:     g.cfg.Name,
		Type:     string(g.cfg.Type),
		Blocking: g.cfg.IsBlocking(),
	}

	// 1. Get staged diffs
	diffs, err := g.gitSvc.StagedDiff(ctx)
	if err != nil {
		result.SystemError = fmt.Sprintf("failed to get staged diffs: %v", err)
		result.DurationMs = time.Since(start).Milliseconds()
		return result, nil
	}

	if len(diffs) == 0 {
		result.Passed = true
		result.DurationMs = time.Since(start).Milliseconds()
		log.Info("LLMGate.Execute skipped — no staged diffs", "gate", g.cfg.Name)
		return result, nil
	}

	// 2. Filter by size
	maxSize := parseMaxFileSize(g.cfg.MaxFileSize)
	filtered, _ := git.FilterBySize(diffs, maxSize)

	if len(filtered) == 0 {
		result.Passed = true
		result.DurationMs = time.Since(start).Milliseconds()
		log.Info("LLMGate.Execute skipped — all files exceed size limit", "gate", g.cfg.Name)
		return result, nil
	}

	// 3. Build prompt and review
	prompt := llm.BuildPrompt(g.cfg.Prompt, "", filtered)
	errors, err := g.client.Review(ctx, prompt)
	if err != nil {
		result.SystemError = fmt.Sprintf("LLM review failed: %v", err)
		result.DurationMs = time.Since(start).Milliseconds()
		return result, nil
	}

	// 4. Validate line numbers against actual diffs (hallucination mitigation)
	validated := llm.ValidateLineNumbers(errors, filtered)

	// 5. Set tool field
	for i := range validated {
		validated[i].Tool = g.cfg.Provider
	}

	result.Errors = validated
	result.Passed = len(validated) == 0

	result.DurationMs = time.Since(start).Milliseconds()
	log.Info("LLMGate.Execute completed", "gate", g.cfg.Name, "passed", result.Passed, "issues", len(validated), "duration_ms", result.DurationMs)
	return result, nil
}

// parseMaxFileSize converts a size string like "100KB" to bytes.
// Supports KB, MB suffixes (case-insensitive). Returns 0 (no limit) on empty or invalid input.
func parseMaxFileSize(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	s = strings.ToUpper(s)

	var multiplier int
	var numStr string

	switch {
	case strings.HasSuffix(s, "MB"):
		multiplier = 1024 * 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(s, "KB"):
		multiplier = 1024
		numStr = s[:len(s)-2]
	default:
		// Assume bytes
		multiplier = 1
		numStr = s
	}

	n, err := strconv.Atoi(strings.TrimSpace(numStr))
	if err != nil || n <= 0 {
		return 0
	}

	return n * multiplier
}

// Ensure LLMGate and ContainerGate implement Gate at compile time.
var (
	_ Gate = (*LLMGate)(nil)
	_ Gate = (*ContainerGate)(nil)
)

// skippedGate is a no-op gate for gates that should be skipped (e.g., no matching files).
type skippedGate struct {
	name     string
	gateType string
}

// Ensure skippedGate implements Gate at compile time.
var _ Gate = (*skippedGate)(nil)

// NewSkippedGate creates a gate that immediately returns a passed, skipped result.
func NewSkippedGate(name, gateType string) Gate {
	return &skippedGate{name: name, gateType: gateType}
}

// Execute returns a passed, skipped result immediately.
func (g *skippedGate) Execute(_ context.Context) (*formatter.GateResult, error) {
	return &formatter.GateResult{
		Name:    g.name,
		Type:    g.gateType,
		Passed:  true,
		Skipped: true,
	}, nil
}
