package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
	"github.com/irahardianto/gatekeeper/internal/engine/gate"
	"github.com/irahardianto/gatekeeper/internal/engine/git"
	"github.com/irahardianto/gatekeeper/internal/engine/llm"
	"github.com/irahardianto/gatekeeper/internal/engine/parser"
	"github.com/irahardianto/gatekeeper/internal/engine/pool"
	"github.com/irahardianto/gatekeeper/internal/engine/runner"
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
)

// ErrGatesFailed is returned when one or more gates fail.
var ErrGatesFailed = errors.New("gates failed")

// runPipeline wires real infrastructure and delegates to Pipeline.Execute.
// This is a composition root — it instantiates production dependencies.
func runPipeline(ctx context.Context, dryRun bool) error {
	log := logger.FromContext(ctx)

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Create Docker runtime and checker.
	runtime, err := pool.NewDockerRuntime()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}

	// Load global config to determine LLM availability.
	globalCfg, err := config.LoadGlobalConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading global config: %w", err)
	}

	// Build gate factory dependencies.
	p := pool.NewPool(runtime)
	exec := pool.NewExecutor(runtime)

	reg := parser.NewRegistry()
	reg.Register("sarif", parser.NewSarifParser())
	reg.Register("go-test-json", parser.NewGoTestParser())

	var llmClient llm.Client
	if !globalCfg.GeminiAPIKey.IsEmpty() {
		llmClient = llm.NewGeminiClient(string(globalCfg.GeminiAPIKey), "", llm.DefaultClientFactory)
	}

	gitSvc := git.NewExecService(projectDir)
	factory := gate.NewFactory(p, exec, reg, llmClient, gitSvc, projectDir)

	// Build a progress-aware runner.
	progress := runner.NewProgress(os.Stderr, flagJSON, 0)
	engine := runner.NewEngineWithProgress(progress)

	// Assemble the pipeline with real infrastructure.
	pipeline := &Pipeline{
		Git:          gitSvc,
		Docker:       &dockerCheckerAdapter{runtime: runtime},
		Gates:        factory,
		Runner:       engine,
		LoadConfig:   config.Load,
		GlobalConfig: globalCfg,
		ConfigPath:   filepath.Join(projectDir, ".gatekeeper", "gates.yaml"),
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	}

	err = pipeline.Execute(ctx, PipelineOpts{
		DryRun:   dryRun,
		JSON:     flagJSON,
		Verbose:  flagVerbose,
		NoColor:  flagNoColor,
		FailFast: flagFailFast,
		Skip:     flagSkip,
		SkipLLM:  flagSkipLLM,
	})
	if err != nil {
		log.Error("pipeline failed", "error", err)
	}
	return err
}

// dockerCheckerAdapter wraps pool.ContainerRuntime to implement DockerChecker.
type dockerCheckerAdapter struct {
	runtime pool.ContainerRuntime
}

func (d *dockerCheckerAdapter) CheckDocker(ctx context.Context) error {
	return pool.CheckDocker(ctx, d.runtime)
}

// filterSkippedGates removes gates matching --skip names or --skip-llm flag.
func filterSkippedGates(gates []config.Gate, skipNames []string, skipLLM bool) []config.Gate {
	if len(skipNames) == 0 && !skipLLM {
		return gates
	}

	skipSet := make(map[string]bool, len(skipNames))
	for _, name := range skipNames {
		skipSet[name] = true
	}

	var result []config.Gate
	for _, g := range gates {
		if skipSet[g.Name] {
			continue
		}
		if skipLLM && g.Type == config.GateTypeLLM {
			continue
		}
		result = append(result, g)
	}
	return result
}
