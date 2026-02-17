package git

import (
	"strings"
)

// SplitDiffs splits a unified diff into per-file FileDiff entries.
// Each entry begins with "diff --git a/..." header.
func SplitDiffs(rawDiff string) []FileDiff {
	if strings.TrimSpace(rawDiff) == "" {
		return nil
	}

	const diffPrefix = "diff --git "
	var diffs []FileDiff

	// Split on "diff --git " boundaries
	parts := strings.Split(rawDiff, diffPrefix)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Extract file path from "a/<path> b/<path>" header
		path := extractFilePath(part)

		diffs = append(diffs, FileDiff{
			Path:    path,
			Content: diffPrefix + part,
		})
	}

	return diffs
}

// extractFilePath parses the file path from a diff header line.
// Format: "a/<path> b/<path>\n..."
func extractFilePath(diffBlock string) string {
	// First line: "a/path b/path"
	firstLine := diffBlock
	if idx := strings.IndexByte(diffBlock, '\n'); idx >= 0 {
		firstLine = diffBlock[:idx]
	}

	// Parse "a/<path> b/<path>" — take the b/ path (destination)
	parts := strings.SplitN(firstLine, " ", 2)
	if len(parts) == 2 {
		bPath := parts[1]
		if strings.HasPrefix(bPath, "b/") {
			return bPath[2:]
		}
		return bPath
	}

	// Fallback: take a/ path
	if strings.HasPrefix(parts[0], "a/") {
		return parts[0][2:]
	}
	return parts[0]
}

// FilterBySize separates diffs by a maximum content size in bytes.
// Returns included diffs (within limit) and skipped diffs (exceeding limit).
func FilterBySize(diffs []FileDiff, maxSize int) (included, skipped []FileDiff) {
	if maxSize <= 0 {
		// No limit — include all.
		return diffs, nil
	}

	for _, d := range diffs {
		if len(d.Content) > maxSize {
			skipped = append(skipped, d)
		} else {
			included = append(included, d)
		}
	}

	return included, skipped
}
