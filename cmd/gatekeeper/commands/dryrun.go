package commands

import (
	"github.com/spf13/cobra"
)

var dryRunCmd = &cobra.Command{
	Use:   "dry-run",
	Short: "Run all gates but always exit 0 (informational only)",
	Long: `Execute all configured gates identically to 'run', but always exit 0
regardless of gate results. Useful for testing configuration without blocking commits.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runPipeline(cmd.Context(), true)
	},
}

func init() {
	rootCmd.AddCommand(dryRunCmd)
}
