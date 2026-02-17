package commands

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run all gates and block commit on failure",
	Long: `Execute all configured gates in parallel. Exit 0 if all blocking gates pass,
exit 1 if any blocking gate fails. Non-blocking gate failures are reported but
do not affect the exit code.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		err := runPipeline(cmd.Context(), false)
		if errors.Is(err, ErrGatesFailed) {
			os.Exit(1)
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
