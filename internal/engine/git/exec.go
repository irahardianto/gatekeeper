package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/irahardianto/gatekeeper/internal/platform/logger"
)

// ExecService implements Service by running git commands via os/exec.
type ExecService struct {
	// WorkDir is the working directory for git commands.
	// If empty, the current directory is used.
	WorkDir string
}

// NewExecService creates a new ExecService with the given working directory.
func NewExecService(workDir string) *ExecService {
	return &ExecService{WorkDir: workDir}
}

// StagedDiff returns per-file diffs of staged changes.
func (s *ExecService) StagedDiff(ctx context.Context) ([]FileDiff, error) {
	logger.FromContext(ctx).Debug("getting staged diffs")

	out, err := s.runGit(ctx, "diff", "--cached")
	if err != nil {
		return nil, fmt.Errorf("getting staged diff: %w", err)
	}

	return SplitDiffs(out), nil
}

// StagedFiles returns the list of staged file paths.
func (s *ExecService) StagedFiles(ctx context.Context) ([]string, error) {
	logger.FromContext(ctx).Debug("getting staged file list")

	out, err := s.runGit(ctx, "diff", "--cached", "--name-only")
	if err != nil {
		return nil, fmt.Errorf("getting staged files: %w", err)
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	return strings.Split(out, "\n"), nil
}

// runGit executes a git command and returns the combined stdout.
func (s *ExecService) runGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...) // #nosec G204 -- args are controlled by the application, not user input
	cmd.Dir = s.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w (stderr: %s)", strings.Join(args, " "), err, stderr.String())
	}

	return stdout.String(), nil
}
