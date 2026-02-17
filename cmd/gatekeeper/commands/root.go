// Package commands implements the CLI commands for gatekeeper.
package commands

import (
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
	"github.com/spf13/cobra"
)

// Global flag values accessible to all commands.
var (
	flagJSON     bool
	flagVerbose  bool
	flagNoColor  bool
	flagFailFast bool
	flagSkip     []string
	flagSkipLLM  bool
)

// rootCmd is the base command for the gatekeeper CLI.
var rootCmd = &cobra.Command{
	Use:   "gatekeeper",
	Short: "Git pre-commit hook gatekeeper",
	Long: `Gatekeeper is an open-source CLI tool that acts as a git pre-commit hook gatekeeper.
It reads a declarative configuration file (.gatekeeper/gates.yaml), executes validation
gates inside Docker containers in parallel, and blocks commits that fail any gate.

Built for AI-assisted development — structured JSON output gives agents precise
file/line locations and fix hints for fast automated remediation.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		l := logger.New(flagVerbose, flagJSON)
		ctx := logger.WithContext(cmd.Context(), l)
		cmd.SetContext(ctx)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output results as JSON to stdout")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Include raw tool stdout/stderr in output")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVar(&flagFailFast, "fail-fast", false, "Cancel remaining gates on first blocking failure")
	rootCmd.PersistentFlags().StringSliceVar(&flagSkip, "skip", nil, "Skip specific gates by name")
	rootCmd.PersistentFlags().BoolVar(&flagSkipLLM, "skip-llm", false, "Skip all LLM gates")
}

// Execute runs the root command. Returns an error if the command fails.
func Execute() error {
	return rootCmd.Execute()
}
