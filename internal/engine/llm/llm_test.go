package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/irahardianto/gatekeeper/internal/engine/git"
	"github.com/irahardianto/gatekeeper/internal/engine/parser"
	"google.golang.org/genai"
)

// --- Prompt Tests ---

func TestBuildPrompt_ContainsUserPrompt(t *testing.T) {
	prompt := BuildPrompt("check for security issues", "go", []git.FileDiff{
		{Path: "main.go", Content: "+fmt.Println(\"hello\")"},
	})

	if !strings.Contains(prompt, "check for security issues") {
		t.Errorf("expected prompt to contain user instruction, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "main.go") {
		t.Errorf("expected prompt to contain file path, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "+fmt.Println") {
		t.Errorf("expected prompt to contain diff content, got:\n%s", prompt)
	}
}

func TestBuildPrompt_MultipleDiffs(t *testing.T) {
	diffs := []git.FileDiff{
		{Path: "a.go", Content: "+line1"},
		{Path: "b.go", Content: "+line2"},
	}

	prompt := BuildPrompt("review", "go", diffs)

	if !strings.Contains(prompt, "a.go") {
		t.Errorf("expected prompt to contain 'a.go', got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "b.go") {
		t.Errorf("expected prompt to contain 'b.go', got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "+line1") {
		t.Errorf("expected prompt to contain '+line1', got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "+line2") {
		t.Errorf("expected prompt to contain '+line2', got:\n%s", prompt)
	}
}

func TestBuildPrompt_EmptyLanguage(t *testing.T) {
	prompt := BuildPrompt("review", "", []git.FileDiff{
		{Path: "f.txt", Content: "x"},
	})

	if !strings.Contains(prompt, "auto-detect") {
		t.Errorf("expected prompt to contain 'auto-detect', got:\n%s", prompt)
	}
}

func TestBuildPrompt_EmptyDiffs(t *testing.T) {
	prompt := BuildPrompt("review", "go", nil)

	if !strings.Contains(prompt, "review") {
		t.Errorf("expected prompt to contain 'review', got:\n%s", prompt)
	}
	// Should still produce valid prompt template
	if !strings.Contains(prompt, "Review rules") {
		t.Errorf("expected prompt to contain 'Review rules', got:\n%s", prompt)
	}
}

// --- Validate Tests ---

func TestValidateLineNumbers_ValidLines(t *testing.T) {
	diffs := []git.FileDiff{
		{
			Path:    "main.go",
			Content: "@@ -1,3 +1,5 @@\n+line1\n+line2",
		},
	}

	errors := []parser.StructuredError{
		{File: "main.go", Line: 3, Severity: "error", Message: "issue"},
	}

	result := ValidateLineNumbers(errors, diffs)
	if len(result) != 1 {
		t.Errorf("expected 1 valid error, got %d", len(result))
	}
}

func TestValidateLineNumbers_HallucinatedLine(t *testing.T) {
	diffs := []git.FileDiff{
		{
			Path:    "main.go",
			Content: "@@ -1,3 +1,5 @@\n+line1",
		},
	}

	errors := []parser.StructuredError{
		{File: "main.go", Line: 100, Severity: "error", Message: "hallucination"},
	}

	result := ValidateLineNumbers(errors, diffs)
	if len(result) != 0 {
		t.Errorf("expected hallucinated line to be filtered, got %d results", len(result))
	}
}

func TestValidateLineNumbers_UnknownFile(t *testing.T) {
	diffs := []git.FileDiff{
		{Path: "main.go", Content: "@@ -1,3 +1,5 @@\n+line1"},
	}

	errors := []parser.StructuredError{
		{File: "nonexistent.go", Line: 1, Severity: "error", Message: "hallucinated file"},
	}

	result := ValidateLineNumbers(errors, diffs)
	if len(result) != 0 {
		t.Errorf("expected unknown file error to be filtered, got %d results", len(result))
	}
}

func TestValidateLineNumbers_MixedResults(t *testing.T) {
	diffs := []git.FileDiff{
		{Path: "a.go", Content: "@@ -1,3 +10,20 @@\n+code"},
	}

	errors := []parser.StructuredError{
		{File: "a.go", Line: 15, Severity: "error", Message: "valid"},
		{File: "a.go", Line: 999, Severity: "error", Message: "hallucinated"},
		{File: "unknown.go", Line: 1, Severity: "error", Message: "wrong file"},
	}

	result := ValidateLineNumbers(errors, diffs)
	if len(result) != 1 {
		t.Fatalf("expected 1 valid result, got %d", len(result))
	}
	if result[0].Message != "valid" {
		t.Errorf("expected 'valid' message, got %q", result[0].Message)
	}
}

func TestValidateLineNumbers_EmptyInput(t *testing.T) {
	result := ValidateLineNumbers(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %d", len(result))
	}
}

// --- MockClient Tests ---

func TestMockClient_ReturnsConfigured(t *testing.T) {
	expected := []parser.StructuredError{
		{File: "test.go", Line: 1, Severity: "error", Message: "mock issue"},
	}
	mock := &MockClient{Result: expected}

	result, err := mock.Review(context.Background(), "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Message != "mock issue" {
		t.Errorf("unexpected result: %v", result)
	}
}

// --- extractText Tests ---

func TestExtractText_NilResponse(t *testing.T) {
	_, err := extractText(nil)
	if err == nil {
		t.Error("expected error for nil response")
	}
}

func TestExtractText_EmptyCandidates(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{},
	}
	_, err := extractText(resp)
	if err == nil {
		t.Error("expected error for empty candidates")
	}
}

func TestExtractText_NilContent(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{Content: nil},
		},
	}
	_, err := extractText(resp)
	if err == nil {
		t.Error("expected error for nil content")
	}
}

func TestExtractText_EmptyParts(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{Content: &genai.Content{Parts: []*genai.Part{}}},
		},
	}
	_, err := extractText(resp)
	if err == nil {
		t.Error("expected error for empty parts")
	}
}

func TestExtractText_EmptyText(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{Content: &genai.Content{Parts: []*genai.Part{
				{Text: ""},
			}}},
		},
	}
	_, err := extractText(resp)
	if err == nil {
		t.Error("expected error for empty text")
	}
}

func TestExtractText_ValidResponse(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{Content: &genai.Content{Parts: []*genai.Part{
				{Text: `[{"file":"main.go","line":1,"severity":"error","message":"issue"}]`},
			}}},
		},
	}
	text, err := extractText(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "main.go") {
		t.Errorf("expected text to contain 'main.go', got: %s", text)
	}
}

// --- structuredErrorSchema Tests ---

func TestStructuredErrorSchema_ArrayType(t *testing.T) {
	schema := structuredErrorSchema()
	if schema.Type != genai.TypeArray {
		t.Errorf("expected schema type Array, got %v", schema.Type)
	}
}

func TestStructuredErrorSchema_ItemType(t *testing.T) {
	schema := structuredErrorSchema()
	if schema.Items == nil {
		t.Fatal("expected non-nil Items schema")
	}
	if schema.Items.Type != genai.TypeObject {
		t.Errorf("expected items type Object, got %v", schema.Items.Type)
	}
}

func TestStructuredErrorSchema_Properties(t *testing.T) {
	schema := structuredErrorSchema()
	expectedProps := []string{"file", "line", "severity", "message", "hint"}
	for _, prop := range expectedProps {
		if _, ok := schema.Items.Properties[prop]; !ok {
			t.Errorf("expected property %q in schema", prop)
		}
	}
}

func TestStructuredErrorSchema_PropertyTypes(t *testing.T) {
	schema := structuredErrorSchema()
	props := schema.Items.Properties

	if props["file"].Type != genai.TypeString {
		t.Errorf("expected 'file' type String, got %v", props["file"].Type)
	}
	if props["line"].Type != genai.TypeInteger {
		t.Errorf("expected 'line' type Integer, got %v", props["line"].Type)
	}
	if props["severity"].Type != genai.TypeString {
		t.Errorf("expected 'severity' type String, got %v", props["severity"].Type)
	}
	if props["message"].Type != genai.TypeString {
		t.Errorf("expected 'message' type String, got %v", props["message"].Type)
	}
	if props["hint"].Type != genai.TypeString {
		t.Errorf("expected 'hint' type String, got %v", props["hint"].Type)
	}
}

func TestStructuredErrorSchema_RequiredFields(t *testing.T) {
	schema := structuredErrorSchema()
	required := schema.Items.Required

	expectedRequired := map[string]bool{
		"file": true, "line": true, "severity": true, "message": true,
	}

	if len(required) != len(expectedRequired) {
		t.Fatalf("expected %d required fields, got %d: %v", len(expectedRequired), len(required), required)
	}

	for _, r := range required {
		if !expectedRequired[r] {
			t.Errorf("unexpected required field: %q", r)
		}
	}
}

func TestStructuredErrorSchema_SeverityEnum(t *testing.T) {
	schema := structuredErrorSchema()
	sevProp := schema.Items.Properties["severity"]

	expectedEnums := map[string]bool{"error": true, "warning": true, "info": true}
	if len(sevProp.Enum) != len(expectedEnums) {
		t.Fatalf("expected %d severity enums, got %d: %v", len(expectedEnums), len(sevProp.Enum), sevProp.Enum)
	}
	for _, e := range sevProp.Enum {
		if !expectedEnums[e] {
			t.Errorf("unexpected severity enum: %q", e)
		}
	}
}

// --- parseHunkHeader Tests ---

func TestParseHunkHeader_Standard(t *testing.T) {
	start, count := parseHunkHeader("@@ -1,3 +10,20 @@")
	if start != 10 {
		t.Errorf("expected start=10, got %d", start)
	}
	if count != 20 {
		t.Errorf("expected count=20, got %d", count)
	}
}

func TestParseHunkHeader_SingleLine(t *testing.T) {
	start, count := parseHunkHeader("@@ -1 +5 @@")
	if start != 5 {
		t.Errorf("expected start=5, got %d", start)
	}
	if count != 1 {
		t.Errorf("expected count=1, got %d", count)
	}
}
