package formatter

import (
	"fmt"
	"strings"

	"github.com/irahardianto/gatekeeper/internal/engine/parser"
)

// ANSI color codes.
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiDim    = "\033[2m"
)

// CLIFormatter outputs RunResult as a human-readable CLI report.
type CLIFormatter struct {
	Color   bool
	Verbose bool
}

// NewCLIFormatter creates a new CLIFormatter.
func NewCLIFormatter(color, verbose bool) *CLIFormatter {
	return &CLIFormatter{Color: color, Verbose: verbose}
}

// Format returns a formatted CLI report.
func (f *CLIFormatter) Format(result RunResult) string {
	var b strings.Builder

	// Header
	icon := f.colorize("✅", ansiGreen)
	status := "passed"
	if !result.Passed {
		icon = f.colorize("❌", ansiRed)
		status = "failed"
	}
	b.WriteString(fmt.Sprintf("\n%s %s — %s in %dms\n\n",
		icon,
		f.colorize("Gatekeeper", ansiBold),
		status,
		result.DurationMs))

	// Gate summary
	for _, g := range result.Gates {
		gateIcon := f.gateIcon(g)
		duration := fmt.Sprintf("%dms", g.DurationMs)

		b.WriteString(fmt.Sprintf("  %s %s %s\n",
			gateIcon,
			f.colorize(g.Name, ansiBold),
			f.colorize(duration, ansiDim)))

		// System error
		if g.SystemError != "" {
			b.WriteString(fmt.Sprintf("    💥 %s\n", f.colorize(g.SystemError, ansiRed)))
		}

		// Structured errors
		for _, e := range g.Errors {
			f.writeError(&b, e)
		}

		// Raw output in verbose mode
		if f.Verbose && g.RawOutput != "" {
			b.WriteString(fmt.Sprintf("\n    %s\n", f.colorize("--- raw output ---", ansiDim)))
			for _, line := range strings.Split(g.RawOutput, "\n") {
				b.WriteString(fmt.Sprintf("    %s\n", f.colorize(line, ansiDim)))
			}
		}
	}

	return b.String()
}

func (f *CLIFormatter) writeError(b *strings.Builder, e parser.StructuredError) {
	// Location
	loc := ""
	if e.File != "" {
		loc = e.File
		if e.Line > 0 {
			loc = fmt.Sprintf("%s:%d", loc, e.Line)
			if e.Column > 0 {
				loc = fmt.Sprintf("%s:%d", loc, e.Column)
			}
		}
		loc = f.colorize(loc, ansiCyan) + " "
	}

	// Severity icon
	sevIcon := "ℹ️"
	sevColor := ansiDim
	switch e.Severity {
	case "error":
		sevIcon = "❌"
		sevColor = ansiRed
	case "warning":
		sevIcon = "⚠️"
		sevColor = ansiYellow
	}

	// Rule
	rule := ""
	if e.Rule != "" {
		rule = f.colorize("["+e.Rule+"]", ansiDim) + " "
	}

	b.WriteString(fmt.Sprintf("    %s %s%s%s\n", sevIcon, loc, rule, f.colorize(e.Message, sevColor)))

	// Hint
	if e.Hint != "" {
		b.WriteString(fmt.Sprintf("      💡 %s\n", e.Hint))
	}
}

func (f *CLIFormatter) gateIcon(g GateResult) string {
	if g.Skipped {
		return "⏭️"
	}
	if g.SystemError != "" {
		return "💥"
	}
	if g.Passed {
		return f.colorize("✅", ansiGreen)
	}
	return f.colorize("❌", ansiRed)
}

func (f *CLIFormatter) colorize(s, code string) string {
	if !f.Color {
		return s
	}
	return code + s + ansiReset
}
