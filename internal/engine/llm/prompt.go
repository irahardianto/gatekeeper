package llm

import (
	"fmt"
	"strings"

	"github.com/irahardianto/gatekeeper/internal/engine/git"
)

const promptTemplate = `You are a code reviewer for a pre-commit hook. Review the following diff and identify issues. Respond ONLY with a JSON array matching the required schema.
If no issues, return: []

Review rules: %s
Language: %s

%s`

// BuildPrompt constructs a review prompt from a user prompt and file diffs.
func BuildPrompt(userPrompt, language string, diffs []git.FileDiff) string {
	if language == "" {
		language = "auto-detect"
	}

	var diffContent strings.Builder
	for _, d := range diffs {
		diffContent.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", d.Path, d.Content))
	}

	return fmt.Sprintf(promptTemplate, userPrompt, language, diffContent.String())
}
