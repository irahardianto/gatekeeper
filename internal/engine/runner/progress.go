package runner

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Progress tracks and renders gate execution status to an io.Writer (typically stderr).
// Output is suppressed in JSON mode to avoid corrupting machine-readable output.
type Progress struct {
	w          io.Writer
	suppressed bool
	total      int
	mu         sync.Mutex
	completed  int
	results    []gateStatus
}

type gateStatus struct {
	name     string
	passed   bool
	sysErr   bool
	errMsg   string
	duration time.Duration
}

// NewProgress creates a new progress tracker writing to w.
// If suppressed is true, no output is produced (for --json mode).
func NewProgress(w io.Writer, suppressed bool, totalGates int) *Progress {
	p := &Progress{
		w:          w,
		suppressed: suppressed,
		total:      totalGates,
	}

	if !suppressed && totalGates > 0 {
		fmt.Fprintf(w, "⏳ Running %d gate(s)...\n", totalGates)
	}

	return p
}

// OnStart is called when a gate begins execution.
func (p *Progress) OnStart(name string) {
	if p.suppressed {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	fmt.Fprintf(p.w, "  ⏳ %s\n", name)
}

// OnComplete is called when a gate finishes execution.
func (p *Progress) OnComplete(name string, passed bool, sysErr bool, errMsg string, dur time.Duration) {
	if p.suppressed {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.completed++
	p.results = append(p.results, gateStatus{
		name:     name,
		passed:   passed,
		sysErr:   sysErr,
		errMsg:   errMsg,
		duration: dur,
	})

	icon := "✅"
	if sysErr {
		icon = "💥"
	} else if !passed {
		icon = "❌"
	}

	durStr := formatDuration(dur)
	fmt.Fprintf(p.w, "  %s %s  %s\n", icon, name, durStr)
}

// Finish prints a summary line after all gates complete.
func (p *Progress) Finish() {
	if p.suppressed {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	passed := 0
	failed := 0
	errors := 0
	for _, r := range p.results {
		switch {
		case r.sysErr:
			errors++
		case !r.passed:
			failed++
		default:
			passed++
		}
	}

	fmt.Fprintf(p.w, "\n")
	if failed == 0 && errors == 0 {
		fmt.Fprintf(p.w, "✅ All %d gate(s) passed\n", passed)
	} else {
		fmt.Fprintf(p.w, "Results: %d passed, %d failed, %d errors\n", passed, failed, errors)
	}
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
