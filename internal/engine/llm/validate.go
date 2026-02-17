package llm

import (
	"strconv"
	"strings"

	"github.com/irahardianto/gatekeeper/internal/engine/git"
	"github.com/irahardianto/gatekeeper/internal/engine/parser"
)

// ValidateLineNumbers filters out StructuredError entries with line numbers
// that fall outside the actual diff ranges. This mitigates LLM hallucinations
// where the model reports issues on lines outside the changed code.
func ValidateLineNumbers(errors []parser.StructuredError, diffs []git.FileDiff) []parser.StructuredError {
	diffLines := buildDiffLineMap(diffs)

	var validated []parser.StructuredError
	for _, e := range errors {
		maxLine, ok := diffLines[e.File]
		if !ok {
			// File not in diff — hallucination; discard.
			continue
		}
		if e.Line > 0 && e.Line > maxLine {
			// Line beyond diff range — hallucination; discard.
			continue
		}
		validated = append(validated, e)
	}

	return validated
}

// buildDiffLineMap extracts the maximum line number referenced in each
// file's diff hunk headers. This gives a rough upper bound for validation.
func buildDiffLineMap(diffs []git.FileDiff) map[string]int {
	result := make(map[string]int, len(diffs))

	for _, d := range diffs {
		maxLine := 0
		for _, line := range strings.Split(d.Content, "\n") {
			// Parse hunk headers: "@@ -a,b +c,d @@"
			if strings.HasPrefix(line, "@@ ") {
				start, count := parseHunkHeader(line)
				if end := start + count; end > maxLine {
					maxLine = end
				}
			}
		}
		if maxLine == 0 {
			// If no hunk headers found, use a generous default.
			maxLine = 10000
		}
		result[d.Path] = maxLine
	}

	return result
}

// parseHunkHeader extracts start line and count from the "+" side of a hunk header.
// Format: "@@ -old_start,old_count +new_start,new_count @@"
func parseHunkHeader(header string) (start, count int) {
	// Find the +start,count portion
	plusIdx := strings.Index(header, "+")
	if plusIdx < 0 {
		return 0, 0
	}

	rest := header[plusIdx+1:]
	endIdx := strings.Index(rest, " ")
	if endIdx < 0 {
		endIdx = len(rest)
	}
	hunk := rest[:endIdx]

	parts := strings.SplitN(hunk, ",", 2)
	start, _ = strconv.Atoi(parts[0])
	if len(parts) == 2 {
		count, _ = strconv.Atoi(parts[1])
	} else {
		count = 1
	}

	return start, count
}
