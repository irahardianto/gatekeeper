package gate

import (
	"testing"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
)

func TestShouldRun_NoFilters(t *testing.T) {
	cfg := config.Gate{Name: "lint"}
	if !ShouldRun(cfg, []string{"main.go", "util.go"}) {
		t.Error("expected gate to run when no filters set")
	}
}

func TestShouldRun_OnlyMatch(t *testing.T) {
	cfg := config.Gate{
		Name: "golint",
		Only: []string{"*.go"},
	}

	if !ShouldRun(cfg, []string{"main.go", "README.md"}) {
		t.Error("expected gate to run — main.go matches *.go")
	}
}

func TestShouldRun_OnlyNoMatch(t *testing.T) {
	cfg := config.Gate{
		Name: "golint",
		Only: []string{"*.go"},
	}

	if ShouldRun(cfg, []string{"README.md", "package.json"}) {
		t.Error("expected gate to skip — no Go files staged")
	}
}

func TestShouldRun_ExceptMatch(t *testing.T) {
	cfg := config.Gate{
		Name:   "lint",
		Except: []string{"*.md"},
	}

	// Only .md files staged — all excluded
	if ShouldRun(cfg, []string{"README.md", "CHANGELOG.md"}) {
		t.Error("expected gate to skip — all files excluded")
	}
}

func TestShouldRun_ExceptPartial(t *testing.T) {
	cfg := config.Gate{
		Name:   "lint",
		Except: []string{"*.md"},
	}

	// Some .md, some .go — should run for .go files
	if !ShouldRun(cfg, []string{"README.md", "main.go"}) {
		t.Error("expected gate to run — main.go survives except filter")
	}
}

func TestShouldRun_OnlyAndExcept(t *testing.T) {
	cfg := config.Gate{
		Name:   "golint",
		Only:   []string{"*.go"},
		Except: []string{"*_test.go"},
	}

	// Only test files — should skip (except removes them, only matches nothing)
	if ShouldRun(cfg, []string{"main_test.go"}) {
		t.Error("expected gate to skip — test files excluded")
	}
}

func TestShouldRun_OnlyAndExceptPass(t *testing.T) {
	cfg := config.Gate{
		Name:   "golint",
		Only:   []string{"*.go"},
		Except: []string{"*_test.go"},
	}

	// Has both test and non-test Go files
	if !ShouldRun(cfg, []string{"main.go", "main_test.go"}) {
		t.Error("expected gate to run — main.go matches only and not excluded")
	}
}

func TestShouldRun_PathMatching(t *testing.T) {
	cfg := config.Gate{
		Name: "golint",
		Only: []string{"*.go"},
	}

	// Nested path — should match on base name
	if !ShouldRun(cfg, []string{"cmd/server/main.go"}) {
		t.Error("expected gate to run — main.go base name matches *.go")
	}
}

func TestShouldRun_EmptyFiles(t *testing.T) {
	cfg := config.Gate{
		Name: "lint",
		Only: []string{"*.go"},
	}

	if ShouldRun(cfg, nil) {
		t.Error("expected gate to skip when no staged files")
	}
}

func TestFilterGates(t *testing.T) {
	gates := []config.Gate{
		{Name: "golint", Only: []string{"*.go"}},
		{Name: "eslint", Only: []string{"*.js", "*.ts"}},
		{Name: "all" /* no filters */},
	}

	stagedFiles := []string{"main.go", "README.md"}

	result := FilterGates(gates, stagedFiles)

	if len(result) != 2 {
		t.Fatalf("expected 2 gates (golint + all), got %d", len(result))
	}

	names := make(map[string]bool)
	for _, g := range result {
		names[g.Name] = true
	}

	if !names["golint"] {
		t.Error("expected golint to be included")
	}
	if !names["all"] {
		t.Error("expected 'all' gate to be included (no filters)")
	}
	if names["eslint"] {
		t.Error("expected eslint to be excluded (no JS/TS files)")
	}
}

func TestFilterGates_NoStagedFiles(t *testing.T) {
	gates := []config.Gate{
		{Name: "lint", Only: []string{"*.go"}},
	}

	// When no staged files, all gates run (no filtering applied)
	result := FilterGates(gates, nil)
	if len(result) != 1 {
		t.Errorf("expected 1 gate when no staged files, got %d", len(result))
	}
}
