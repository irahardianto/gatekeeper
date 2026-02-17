package git

import (
	"testing"
)

func TestSplitDiffs_MultipleFiles(t *testing.T) {
	rawDiff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
+import "fmt"
diff --git a/utils.go b/utils.go
--- a/utils.go
+++ b/utils.go
@@ -5,2 +5,3 @@
 func helper() {
+    return nil
`

	diffs := SplitDiffs(rawDiff)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}

	if diffs[0].Path != "main.go" {
		t.Errorf("expected first diff path 'main.go', got %q", diffs[0].Path)
	}
	if diffs[1].Path != "utils.go" {
		t.Errorf("expected second diff path 'utils.go', got %q", diffs[1].Path)
	}
}

func TestSplitDiffs_SingleFile(t *testing.T) {
	rawDiff := `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1 +1,2 @@
+Hello
`

	diffs := SplitDiffs(rawDiff)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}

	if diffs[0].Path != "README.md" {
		t.Errorf("expected path 'README.md', got %q", diffs[0].Path)
	}
}

func TestSplitDiffs_Empty(t *testing.T) {
	diffs := SplitDiffs("")
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for empty input, got %d", len(diffs))
	}

	diffs = SplitDiffs("   \n\n  ")
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for whitespace input, got %d", len(diffs))
	}
}

func TestFilterBySize_WithinLimit(t *testing.T) {
	diffs := []FileDiff{
		{Path: "small.go", Content: "short"},
		{Path: "medium.go", Content: "medium content here"},
	}

	included, skipped := FilterBySize(diffs, 100)
	if len(included) != 2 {
		t.Errorf("expected 2 included, got %d", len(included))
	}
	if len(skipped) != 0 {
		t.Errorf("expected 0 skipped, got %d", len(skipped))
	}
}

func TestFilterBySize_ExceedsLimit(t *testing.T) {
	diffs := []FileDiff{
		{Path: "small.go", Content: "short"},
		{Path: "big.go", Content: "this content is way too long for the limit we set"},
	}

	included, skipped := FilterBySize(diffs, 10)
	if len(included) != 1 {
		t.Errorf("expected 1 included, got %d", len(included))
	}
	if included[0].Path != "small.go" {
		t.Errorf("expected 'small.go' included, got %q", included[0].Path)
	}
	if len(skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d", len(skipped))
	}
	if skipped[0].Path != "big.go" {
		t.Errorf("expected 'big.go' skipped, got %q", skipped[0].Path)
	}
}

func TestFilterBySize_NoLimit(t *testing.T) {
	diffs := []FileDiff{
		{Path: "file.go", Content: "any content"},
	}

	included, skipped := FilterBySize(diffs, 0)
	if len(included) != 1 {
		t.Errorf("expected all included with 0 limit, got %d", len(included))
	}
	if len(skipped) != 0 {
		t.Errorf("expected 0 skipped with 0 limit, got %d", len(skipped))
	}
}

func TestFilterBySize_AllSkipped(t *testing.T) {
	diffs := []FileDiff{
		{Path: "a.go", Content: "too long content"},
		{Path: "b.go", Content: "also too long content"},
	}

	included, skipped := FilterBySize(diffs, 5)
	if len(included) != 0 {
		t.Errorf("expected 0 included, got %d", len(included))
	}
	if len(skipped) != 2 {
		t.Errorf("expected 2 skipped, got %d", len(skipped))
	}
}

// --- extractFilePath edge cases ---

func TestExtractFilePath_NoBPrefix(t *testing.T) {
	// When there's no "b/" prefix, the raw destination path should be returned.
	result := extractFilePath("a/foo.go bar.go\n@@ rest")
	if result != "bar.go" {
		t.Errorf("expected 'bar.go', got %q", result)
	}
}

func TestExtractFilePath_SinglePart(t *testing.T) {
	// When only a single part exists (no space separator), fallback to a/ stripping.
	result := extractFilePath("a/only.go")
	if result != "only.go" {
		t.Errorf("expected 'only.go', got %q", result)
	}
}

func TestExtractFilePath_SinglePartNoAPrefix(t *testing.T) {
	// When only a single part exists with no a/ prefix, return as-is.
	result := extractFilePath("rawfile.go")
	if result != "rawfile.go" {
		t.Errorf("expected 'rawfile.go', got %q", result)
	}
}
