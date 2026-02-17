package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
	"github.com/irahardianto/gatekeeper/internal/engine/formatter"
	"github.com/irahardianto/gatekeeper/internal/engine/gate"
	"github.com/irahardianto/gatekeeper/internal/engine/git"
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
)

// PipelineOpts holds per-invocation options for the pipeline.
type PipelineOpts struct {
	DryRun   bool
	JSON     bool
	Verbose  bool
	NoColor  bool
	FailFast bool
	Skip     []string
	SkipLLM  bool
}

// Pipeline orchestrates the full gatekeeper pipeline with injected dependencies.
// This struct enables testing the orchestration logic without real infrastructure.
type Pipeline struct {
	// Git provides stash, staged files, hook, and cleanup operations.
	Git git.Service

	// Docker checks Docker availability before running gates.
	Docker DockerChecker

	// Gates creates gate instances from configuration.
	Gates GateCreator

	// Runner executes gates in parallel.
	Runner GateRunner

	// LoadConfig loads the project-level gates.yaml.
	LoadConfig func(ctx context.Context, path string) (*config.GatekeeperConfig, error)

	// GlobalConfig holds the pre-loaded global configuration (~/.config/gatekeeper/).
	GlobalConfig *config.GlobalConfig

	// ConfigPath is the path to the gates.yaml file.
	ConfigPath string

	// Stdout is the output writer for formatted results.
	Stdout io.Writer

	// Stderr is the output writer for progress/status messages.
	Stderr io.Writer

	// StagedFiles is the list of staged file paths (injected for testability).
	// If nil, Pipeline calls Git.StagedFiles.
	stagedFiles []string
}

// Execute runs the full pipeline orchestration.
func (p *Pipeline) Execute(ctx context.Context, opts PipelineOpts) error {
	log := logger.FromContext(ctx)
	operation := "run"
	if opts.DryRun {
		operation = "dry-run"
	}
	log.Info("gatekeeper pipeline started", "operation", operation)

	// 1. Load project configuration.
	cfg, err := p.LoadConfig(ctx, p.ConfigPath)
	if err != nil {
		return err
	}

	// 2. Validate global configuration is available.
	if p.GlobalConfig == nil {
		return fmt.Errorf("global config not loaded")
	}

	// 3. Docker pre-flight check (before stash).
	if err := p.Docker.CheckDocker(ctx); err != nil {
		return err
	}

	// 4. Stash unstaged changes.
	stashed, err := p.Git.Stash(ctx)
	if err != nil {
		return fmt.Errorf("stashing changes: %w", err)
	}

	// Set up signal handler and defer stash pop.
	if stashed {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			log.Info("signal received, restoring stash")
			_ = p.Git.StashPop(context.Background())
			os.Exit(1)
		}()

		defer func() {
			signal.Stop(sigCh)
			if popErr := p.Git.StashPop(ctx); popErr != nil {
				log.Error("failed to restore stash", "error", popErr)
			}
		}()
	}

	// 5. Get staged files for filtering.
	stagedFiles := p.stagedFiles
	if stagedFiles == nil {
		stagedFiles, err = p.Git.StagedFiles(ctx)
		if err != nil {
			return fmt.Errorf("getting staged files: %w", err)
		}
	}

	// 6. Apply --skip and --skip-llm filters.
	gates := filterSkippedGates(cfg.Gates, opts.Skip, opts.SkipLLM)

	// 7. Apply file filters (only/except).
	gates = gate.FilterGates(gates, stagedFiles)

	if len(gates) == 0 {
		fmt.Fprintln(p.Stderr, "✅ No gates to run")
		return nil
	}

	// 8. Create gate instances.
	gateInstances, err := p.Gates.CreateAll(gates)
	if err != nil {
		return err
	}

	// 9. Build gate names for progress.
	var gateNames []string
	for _, g := range gates {
		gateNames = append(gateNames, g.Name)
	}

	// 10. Execute gates in parallel.
	result, err := p.Runner.RunAll(ctx, gateInstances, opts.FailFast, gateNames)
	if err != nil {
		return err
	}

	// 11. Clean up writable file modifications.
	for _, g := range gates {
		if g.Writable {
			if cleanErr := p.Git.CleanWritableFiles(ctx); cleanErr != nil {
				log.Error("failed to clean writable files", "error", cleanErr)
			}
			break
		}
	}

	// 12. Format and print results.
	var fmtr formatter.Formatter
	if opts.JSON {
		fmtr = formatter.NewJSONFormatter()
	} else {
		fmtr = formatter.NewCLIFormatter(!opts.NoColor, opts.Verbose)
	}
	fmt.Fprint(p.Stdout, fmtr.Format(*result))

	// 13. Determine exit code.
	if opts.DryRun {
		return nil
	}
	if !result.Passed {
		return ErrGatesFailed
	}
	return nil
}
