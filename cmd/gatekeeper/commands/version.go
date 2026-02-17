package commands

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags.
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and build information",
	Long:  "Print the gatekeeper version, Go version, and build information.",
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Printf("gatekeeper %s\n", version)
		fmt.Printf("  go:     %s\n", runtime.Version())
		fmt.Printf("  os:     %s/%s\n", runtime.GOOS, runtime.GOARCH)

		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					fmt.Printf("  commit: %s\n", setting.Value)
				}
				if setting.Key == "vcs.time" {
					fmt.Printf("  built:  %s\n", setting.Value)
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
