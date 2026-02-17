package commands

import (
	"fmt"
	"os"

	"github.com/irahardianto/gatekeeper/internal/engine/git"
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
	"github.com/spf13/cobra"
)

var teardownCmd = &cobra.Command{
	Use:   "teardown",
	Short: "Remove the git pre-commit hook",
	Long: `Remove the gatekeeper git pre-commit hook.
The .gatekeeper/ directory and configuration are preserved.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		log := logger.FromContext(ctx)
		log.Info("teardown started")

		projectDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}

		gitSvc := git.NewExecService(projectDir)
		if err := gitSvc.RemoveHook(ctx); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "🔓 Gatekeeper pre-commit hook removed")
		log.Info("teardown completed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(teardownCmd)
}
