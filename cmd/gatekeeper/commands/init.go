package commands

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
	"github.com/irahardianto/gatekeeper/internal/engine/git"
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
	"github.com/spf13/cobra"
)

// InitFS abstracts file system operations needed by the init command.
type InitFS interface {
	Stat(name string) (fs.FileInfo, error)
	IsNotExist(err error) bool
	MkdirAll(path string, perm fs.FileMode) error
	ReadDir(name string) ([]fs.DirEntry, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gatekeeper in the current project",
	Long: `Detect the project's technology stack, generate a default .gatekeeper/gates.yaml,
and install the git pre-commit hook.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		log := logger.FromContext(ctx)
		log.Info("init started")

		projectDir, err := getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}

		gitSvc := git.NewExecService(projectDir)
		if err := initProject(ctx, projectDir, &osInitFS{}, gitSvc, cmd.OutOrStdout()); err != nil {
			return err
		}

		log.Info("init completed")
		return nil
	},
}

// initProject performs the init workflow with injected dependencies for testability.
func initProject(ctx context.Context, projectDir string, fsys InitFS, gitSvc git.Service, out io.Writer) error {
	// 1. Create .gatekeeper directory if it doesn't exist.
	gkDir := filepath.Join(projectDir, ".gatekeeper")
	if err := fsys.MkdirAll(gkDir, 0o750); err != nil {
		return fmt.Errorf("creating .gatekeeper directory: %w", err)
	}

	// 2. Generate default gates.yaml if it doesn't exist.
	configPath := filepath.Join(gkDir, "gates.yaml")
	if _, err := fsys.Stat(configPath); fsys.IsNotExist(err) {
		// Detect stacks from project root files.
		entries, readErr := fsys.ReadDir(projectDir)
		if readErr != nil {
			return fmt.Errorf("reading project directory: %w", readErr)
		}
		var files []string
		for _, e := range entries {
			files = append(files, e.Name())
		}

		stacks := config.DetectStacks(files)
		yamlContent := config.GenerateGatesYAML(stacks)

		if writeErr := fsys.WriteFile(configPath, []byte(yamlContent), 0o644); writeErr != nil { // #nosec G306 -- config file, not sensitive
			return fmt.Errorf("writing gates.yaml: %w", writeErr)
		}

		if len(stacks) > 0 {
			fmt.Fprintf(out, "✅ Detected %s project. Generated %s with gates.\n", formatStacks(stacks), configPath)
		} else {
			fmt.Fprintf(out, "📝 No stack detected. Created minimal %s — customize it.\n", configPath)
		}
	} else {
		fmt.Fprintf(out, "⚡ Config already exists at %s. Skipping generation.\n", configPath)
	}

	// 3. Install git pre-commit hook.
	if err := gitSvc.InstallHook(ctx); err != nil {
		return fmt.Errorf("installing hook: %w", err)
	}

	fmt.Fprintln(out, "🔒 Gatekeeper initialized successfully!")
	return nil
}

// getwd is a variable for testability (defaults to os.Getwd).
var getwd = os.Getwd

// formatStacks returns a human-readable string of detected stacks.
func formatStacks(stacks []config.Stack) string {
	if len(stacks) == 1 {
		return string(stacks[0])
	}
	var stackStrings []string
	for _, s := range stacks {
		stackStrings = append(stackStrings, string(s))
	}
	return join(stackStrings, " + ")
}

// join concatenates string slices with a separator (avoids importing strings for one call).
func join(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	result := elems[0]
	for _, e := range elems[1:] {
		result += sep + e
	}
	return result
}

func init() {
	rootCmd.AddCommand(initCmd)
}
