package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/irahardianto/gatekeeper/internal/platform/logger"
)

const (
	hookMarker = "# gatekeeper-managed"
	hookScript = `#!/bin/sh
# gatekeeper-managed
# This hook was installed by gatekeeper. Do not edit manually.
# Run 'gatekeeper teardown' to remove.
exec gatekeeper run "$@"
`
)

// InstallHook creates a pre-commit hook that invokes gatekeeper.
// If the hook already exists and is not managed by gatekeeper, it returns an error.
func (s *ExecService) InstallHook(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("installing pre-commit hook")

	gitDir, err := s.findGitDir(ctx)
	if err != nil {
		return fmt.Errorf("finding .git directory: %w", err)
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")

	// Check if hook already exists.
	if data, err := os.ReadFile(hookPath); err == nil { // #nosec G304 -- path is constructed from .git dir, not user input
		content := string(data)
		if strings.Contains(content, hookMarker) {
			log.Info("hook already installed, skipping")
			return nil // Already managed by gatekeeper.
		}
		return fmt.Errorf("pre-commit hook already exists at %s — remove it first or back it up", hookPath)
	}

	// Create hooks directory if it doesn't exist.
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	// Write hook script.
	if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil { // #nosec G306 -- hook must be executable
		return fmt.Errorf("writing hook script: %w", err)
	}

	log.Info("pre-commit hook installed", "path", hookPath)
	return nil
}

// RemoveHook removes the gatekeeper-managed pre-commit hook.
// Returns nil if no hook exists or the hook is not managed by gatekeeper.
func (s *ExecService) RemoveHook(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("removing pre-commit hook")

	gitDir, err := s.findGitDir(ctx)
	if err != nil {
		return fmt.Errorf("finding .git directory: %w", err)
	}

	hookPath := filepath.Join(gitDir, "hooks", "pre-commit")

	data, err := os.ReadFile(hookPath) // #nosec G304 -- path is constructed from .git dir, not user input
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Info("no pre-commit hook found, nothing to remove")
			return nil
		}
		return fmt.Errorf("reading hook: %w", err)
	}

	// Only remove if it's managed by gatekeeper.
	if !strings.Contains(string(data), hookMarker) {
		return fmt.Errorf("pre-commit hook exists but is not managed by gatekeeper — will not remove")
	}

	if err := os.Remove(hookPath); err != nil {
		return fmt.Errorf("removing hook: %w", err)
	}

	log.Info("pre-commit hook removed", "path", hookPath)
	return nil
}

// findGitDir locates the .git directory by running `git rev-parse --git-dir`.
func (s *ExecService) findGitDir(ctx context.Context) (string, error) {
	out, err := s.runGit(ctx, "rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}

	gitDir := strings.TrimSpace(out)
	if !filepath.IsAbs(gitDir) && s.WorkDir != "" {
		gitDir = filepath.Join(s.WorkDir, gitDir)
	}

	return gitDir, nil
}
