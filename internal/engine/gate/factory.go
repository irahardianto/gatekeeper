package gate

import (
	"fmt"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
	"github.com/irahardianto/gatekeeper/internal/engine/git"
	"github.com/irahardianto/gatekeeper/internal/engine/llm"
	"github.com/irahardianto/gatekeeper/internal/engine/parser"
)

// Factory creates Gate instances from configuration.
type Factory struct {
	pool        PoolManager
	executor    CommandExecutor
	registry    *parser.Registry
	llmClient   llm.Client
	gitService  git.Service
	projectPath string
}

// NewFactory creates a new Factory with the given dependencies.
// llmClient may be nil if no LLM gates are configured.
func NewFactory(
	p PoolManager,
	exec CommandExecutor,
	reg *parser.Registry,
	llmClient llm.Client,
	gitSvc git.Service,
	projectPath string,
) *Factory {
	return &Factory{
		pool:        p,
		executor:    exec,
		registry:    reg,
		llmClient:   llmClient,
		gitService:  gitSvc,
		projectPath: projectPath,
	}
}

// Create builds a Gate from a gate config entry.
// Returns an error if the gate type is unknown or dependencies are missing.
func (f *Factory) Create(cfg config.Gate) (Gate, error) {
	switch cfg.Type {
	case config.GateTypeExec, config.GateTypeScript:
		return f.createContainerGate(cfg), nil
	case config.GateTypeLLM:
		return f.createLLMGate(cfg)
	default:
		return nil, fmt.Errorf("unknown gate type %q for gate %q", cfg.Type, cfg.Name)
	}
}

// createContainerGate builds a ContainerGate with the appropriate parser.
// Handles both "exec" and "script" gate types.
func (f *Factory) createContainerGate(cfg config.Gate) Gate {
	prs := f.registry.GetOrDefault(cfg.Parser)
	return NewContainerGate(cfg, f.pool, f.executor, prs, f.projectPath)
}

// createLLMGate builds an LLMGate, returning an error if no LLM client is configured.
func (f *Factory) createLLMGate(cfg config.Gate) (Gate, error) {
	if f.llmClient == nil {
		return nil, fmt.Errorf("gate %q requires an LLM client but none is configured — set GATEKEEPER_GEMINI_KEY or add to ~/.config/gatekeeper/config.yaml", cfg.Name)
	}
	return NewLLMGate(cfg, f.llmClient, f.gitService), nil
}

// CreateAll builds Gates from a list of gate configs.
// Returns the created gates and any errors encountered.
func (f *Factory) CreateAll(gates []config.Gate) ([]Gate, error) {
	result := make([]Gate, 0, len(gates))
	for _, cfg := range gates {
		g, err := f.Create(cfg)
		if err != nil {
			return nil, fmt.Errorf("creating gate %q: %w", cfg.Name, err)
		}
		result = append(result, g)
	}
	return result, nil
}
