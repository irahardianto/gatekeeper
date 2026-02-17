// Package git abstracts git operations for testability.
package git

import (
	"context"
)

// FileDiff holds a per-file diff from staged changes.
type FileDiff struct {
	Path    string
	Content string
}

// Service abstracts git operations for testability.
type Service interface {
	// StagedDiff returns per-file diffs of staged changes.
	StagedDiff(ctx context.Context) ([]FileDiff, error)
	// StagedFiles returns the list of staged file paths.
	StagedFiles(ctx context.Context) ([]string, error)

	// InstallHook creates a pre-commit hook script in .git/hooks/.
	InstallHook(ctx context.Context) error
	// RemoveHook removes the gatekeeper pre-commit hook.
	RemoveHook(ctx context.Context) error

	// Stash saves unstaged/untracked changes so only staged changes remain.
	// Returns true if a stash was actually created (i.e., there was something to stash).
	Stash(ctx context.Context) (bool, error)
	// StashPop restores previously stashed changes.
	StashPop(ctx context.Context) error
	// CleanWritableFiles reverts writable gate modifications in the working tree.
	CleanWritableFiles(ctx context.Context) error
}
