package commands

import (
	"fmt"

	"github.com/irahardianto/gatekeeper/internal/engine/pool"
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Stop and remove all gatekeeper containers",
	Long: `Stop and remove all Docker containers with the gatekeeper.managed=true label.
This is useful for cleaning up resources when you're done with gatekeeper.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		log := logger.FromContext(ctx)
		log.Info("cleanup started")

		runtime, err := pool.NewDockerRuntime()
		if err != nil {
			return fmt.Errorf("connecting to Docker: %w", err)
		}

		p := pool.NewPool(runtime)
		count, err := p.CleanupAll(ctx)
		if err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "♻️  Removed %d gatekeeper container(s)\n", count)
		log.Info("cleanup completed", "removed", count)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}
