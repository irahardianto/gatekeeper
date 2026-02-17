package gate

import (
	"path/filepath"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
)

// ShouldRun determines whether a gate should run based on its only/except patterns
// and the list of staged files. Returns true if the gate should run.
//
// Rules:
//   - If no only/except patterns are configured, the gate always runs.
//   - If only is set, the gate runs only if at least one staged file matches.
//   - If except is set, files matching except patterns are excluded first.
//   - Patterns use filepath.Match glob syntax (e.g., "*.go", "cmd/**").
func ShouldRun(cfg config.Gate, stagedFiles []string) bool {
	if len(cfg.Only) == 0 && len(cfg.Except) == 0 {
		return true
	}

	// Filter files: remove those matching except patterns.
	filtered := stagedFiles
	if len(cfg.Except) > 0 {
		filtered = excludeFiles(stagedFiles, cfg.Except)
	}

	// If only patterns are set, check if any remaining file matches.
	if len(cfg.Only) > 0 {
		return matchesAny(filtered, cfg.Only)
	}

	// Except-only: run if any files survive the exclusion.
	return len(filtered) > 0
}

// excludeFiles returns files that do NOT match any of the given patterns.
func excludeFiles(files, patterns []string) []string {
	var result []string
	for _, f := range files {
		if !matchesPattern(f, patterns) {
			result = append(result, f)
		}
	}
	return result
}

// matchesAny returns true if at least one file matches any of the patterns.
func matchesAny(files, patterns []string) bool {
	for _, f := range files {
		if matchesPattern(f, patterns) {
			return true
		}
	}
	return false
}

// matchesPattern returns true if the file matches any of the given glob patterns.
// Matches against both the full path and the base name.
func matchesPattern(file string, patterns []string) bool {
	base := filepath.Base(file)
	for _, p := range patterns {
		// Match against full path.
		if matched, _ := filepath.Match(p, file); matched {
			return true
		}
		// Match against base name (e.g., "*.go" should match "cmd/main.go").
		if matched, _ := filepath.Match(p, base); matched {
			return true
		}
	}
	return false
}

// FilterGates wraps each gate config with ShouldRun logic.
// Returns a filtered list — gates that don't match any staged file are excluded.
//
// When stagedFiles is empty, all gates are returned. This allows gates without
// file filters (only/except patterns) to run even when no files are staged.
// This is intentional behavior for gates that perform global checks (e.g., LLM review
// of all changes, or gates that don't depend on specific files).
func FilterGates(gates []config.Gate, stagedFiles []string) []config.Gate {
	if len(stagedFiles) == 0 {
		return gates
	}

	var result []config.Gate
	for _, g := range gates {
		if ShouldRun(g, stagedFiles) {
			result = append(result, g)
		}
	}
	return result
}
