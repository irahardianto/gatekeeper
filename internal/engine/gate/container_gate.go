package gate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
	"github.com/irahardianto/gatekeeper/internal/engine/formatter"
	"github.com/irahardianto/gatekeeper/internal/engine/parser"
	"github.com/irahardianto/gatekeeper/internal/engine/pool"
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
)

// PoolManager abstracts container pool operations for testability.
type PoolManager interface {
	GetOrCreate(ctx context.Context, img, projectPath string, writable bool) (string, error)
}

// CommandExecutor abstracts command execution for testability.
type CommandExecutor interface {
	Run(ctx context.Context, containerID, command string, timeout time.Duration) (*pool.ExecResult, error)
}

// ContainerGate executes a command or script inside a Docker container and parses the output.
// It handles both "exec" gates (direct command execution) and "script" gates (shell script execution).
type ContainerGate struct {
	cfg      config.Gate
	pool     PoolManager
	executor CommandExecutor
	parser   parser.Parser
	project  string
}

// NewContainerGate creates a new ContainerGate.
func NewContainerGate(cfg config.Gate, p PoolManager, exec CommandExecutor, prs parser.Parser, projectPath string) *ContainerGate {
	return &ContainerGate{
		cfg:      cfg,
		pool:     p,
		executor: exec,
		parser:   prs,
		project:  projectPath,
	}
}

// Execute runs the command or script in a container, parses the output, and returns the result.
func (g *ContainerGate) Execute(ctx context.Context) (*formatter.GateResult, error) {
	log := logger.FromContext(ctx)
	log.Info("ContainerGate.Execute started", "gate", g.cfg.Name, "type", g.cfg.Type)
	start := time.Now()

	result := &formatter.GateResult{
		Name:     g.cfg.Name,
		Type:     string(g.cfg.Type),
		Blocking: g.cfg.IsBlocking(),
	}

	// 1. Get or create container
	containerID, err := g.pool.GetOrCreate(ctx, g.cfg.Container, g.project, g.cfg.Writable)
	if err != nil {
		result.SystemError = fmt.Sprintf("container setup failed: %v", err)
		result.DurationMs = time.Since(start).Milliseconds()
		return result, nil
	}

	// 2. Build command based on gate type
	command := g.buildCommand()

	// 3. Execute command
	timeout := g.cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	execResult, err := g.executor.Run(ctx, containerID, command, timeout)
	if err != nil {
		result.SystemError = fmt.Sprintf("execution failed: %v", err)
		result.DurationMs = time.Since(start).Milliseconds()
		return result, nil
	}

	result.RawOutput = string(execResult.Stdout)

	// 4. Parse output
	parsed, err := g.parser.Parse(ctx, execResult.Stdout, execResult.Stderr, execResult.ExitCode)
	if err != nil {
		result.SystemError = fmt.Sprintf("parser error: %v", err)
		result.DurationMs = time.Since(start).Milliseconds()
		return result, nil
	}

	result.Passed = parsed.Passed
	result.Errors = parsed.Errors

	// 5. Enrich hints
	parser.EnrichHints(result.Errors)

	result.DurationMs = time.Since(start).Milliseconds()
	log.Info("ContainerGate.Execute completed", "gate", g.cfg.Name, "passed", result.Passed, "duration_ms", result.DurationMs)
	return result, nil
}

// shellQuote wraps a string in single quotes with proper escaping.
// Single quotes within the string are escaped as '\” (end quote, escaped quote, start quote).
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// buildCommand constructs the command string based on gate type.
// For "exec" gates, uses cfg.Command directly.
// For "script" gates, constructs a shell invocation of cfg.Path.
// The project root is mounted at /workspace, so scripts are accessible at /workspace/<path>.
func (g *ContainerGate) buildCommand() string {
	switch g.cfg.Type {
	case config.GateTypeScript:
		// Security: Properly shell-quote the path to prevent injection.
		// Config validation also rejects paths with single quotes for defense-in-depth.
		return "sh " + shellQuote(g.cfg.Path)
	case config.GateTypeExec:
		return g.cfg.Command
	default:
		return g.cfg.Command
	}
}
