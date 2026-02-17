package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/irahardianto/gatekeeper/internal/platform/logger"
)

const stashMessage = "gatekeeper-stash"

// Stash saves unstaged and untracked changes so only staged changes remain in the working tree.
// Returns true if a stash was actually created (there was something to stash).
// If there are no unstaged changes, no stash is created and false is returned.
func (s *ExecService) Stash(ctx context.Context) (bool, error) {
	log := logger.FromContext(ctx)
	log.Info("stashing unstaged changes")

	// Check if there are any unstaged changes to stash.
	_, err := s.runGit(ctx, "diff", "--quiet")
	if err == nil {
		// No unstaged changes — check untracked files.
		out, err := s.runGit(ctx, "ls-files", "--others", "--exclude-standard")
		if err != nil {
			return false, fmt.Errorf("checking untracked files: %w", err)
		}
		if strings.TrimSpace(out) == "" {
			log.Info("nothing to stash — working tree is clean relative to index")
			return false, nil
		}
	}

	// Stash everything except staged changes.
	_, err = s.runGit(ctx, "stash", "push", "--keep-index", "--include-untracked", "-m", stashMessage)
	if err != nil {
		return false, fmt.Errorf("stashing changes: %w", err)
	}

	log.Info("changes stashed successfully")
	return true, nil
}

// StashPop restores previously stashed changes.
// It looks for the gatekeeper stash by message and pops it.
func (s *ExecService) StashPop(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("restoring stashed changes")

	// Find the gatekeeper stash entry.
	out, err := s.runGit(ctx, "stash", "list")
	if err != nil {
		return fmt.Errorf("listing stashes: %w", err)
	}

	// Search for our stash.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, stashMessage) {
			// Extract stash reference (e.g., "stash@{0}")
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 0 {
				continue
			}
			ref := strings.TrimSpace(parts[0])

			_, err := s.runGit(ctx, "stash", "pop", ref)
			if err != nil {
				return fmt.Errorf("popping stash: %w", err)
			}

			log.Info("stash restored successfully")
			return nil
		}
	}

	log.Info("no gatekeeper stash found to restore")
	return nil
}

// CleanWritableFiles reverts modifications made by writable gates.
// This runs `git checkout -- .` and `git clean -fd` to restore the working tree.
func (s *ExecService) CleanWritableFiles(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("cleaning writable file modifications")

	// Restore tracked files.
	if _, err := s.runGit(ctx, "checkout", "--", "."); err != nil {
		return fmt.Errorf("reverting tracked files: %w", err)
	}

	// Remove untracked files created by writable gates.
	if _, err := s.runGit(ctx, "clean", "-fd"); err != nil {
		return fmt.Errorf("cleaning untracked files: %w", err)
	}

	log.Info("working tree cleaned")
	return nil
}
