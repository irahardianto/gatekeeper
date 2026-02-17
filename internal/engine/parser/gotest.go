package parser

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// GoTestParser parses `go test -json` output (newline-delimited TestEvent objects).
type GoTestParser struct{}

// NewGoTestParser creates a new GoTestParser.
func NewGoTestParser() *GoTestParser {
	return &GoTestParser{}
}

// testEvent represents a single event emitted by `go test -json`.
// See: https://pkg.go.dev/cmd/test2json#hdr-Output_Format
type testEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

// Parse implements the Parser interface for `go test -json` output.
//
// It processes the newline-separated JSON stream and extracts failures.
// For each test (or package) with Action=="fail", it collects the associated
// output lines and produces a StructuredError.
func (p *GoTestParser) Parse(_ context.Context, stdout, stderr []byte, exitCode int) (*ParseResult, error) {
	// Fail-closed on empty stdout.
	if len(bytes.TrimSpace(stdout)) == 0 {
		if exitCode != 0 {
			msg := strings.TrimSpace(string(stderr))
			if msg == "" {
				msg = "go test failed with non-zero exit code and empty output"
			}
			return &ParseResult{
				Passed: false,
				Errors: []StructuredError{
					{
						Severity: "error",
						Message:  msg,
						Tool:     "go-test",
					},
				},
			}, nil
		}
		return &ParseResult{Passed: true}, nil
	}

	// Parse all events from the JSON stream.
	events, err := parseEvents(stdout)
	if err != nil {
		return nil, fmt.Errorf("parsing go test JSON output: %w", err)
	}

	// Collect output lines per test (keyed by "package::test").
	// Package-level output uses the key "package::" (empty test name).
	outputs := make(map[string][]string)
	for _, ev := range events {
		if ev.Action == "output" {
			key := ev.Package + "::" + ev.Test
			outputs[key] = append(outputs[key], ev.Output)
		}
	}

	// Collect test-level failures first, track which packages had test failures.
	var errors []StructuredError
	failedPackages := make(map[string]bool)

	for _, ev := range events {
		if ev.Action != "fail" || ev.Test == "" {
			continue
		}

		failedPackages[ev.Package] = true
		key := ev.Package + "::" + ev.Test

		msg := strings.TrimSpace(strings.Join(outputs[key], ""))
		if msg == "" {
			msg = fmt.Sprintf("test %s failed", ev.Test)
		}

		errors = append(errors, StructuredError{
			File:     ev.Package,
			Severity: "error",
			Message:  msg,
			Tool:     "go-test",
		})
	}

	// Collect package-level failures only when no test-level failures exist
	// (e.g., build errors where the package fails without any test running).
	for _, ev := range events {
		if ev.Action != "fail" || ev.Test != "" {
			continue
		}
		if failedPackages[ev.Package] {
			// Already reported via individual test failures.
			continue
		}

		key := ev.Package + "::"
		msg := strings.TrimSpace(strings.Join(outputs[key], ""))
		if msg == "" {
			msg = fmt.Sprintf("package %s failed", ev.Package)
		}

		errors = append(errors, StructuredError{
			File:     ev.Package,
			Severity: "error",
			Message:  msg,
			Tool:     "go-test",
		})
	}

	return &ParseResult{
		Passed: len(errors) == 0 && exitCode == 0,
		Errors: errors,
	}, nil
}

// parseEvents decodes newline-delimited JSON into a slice of testEvent.
func parseEvents(data []byte) ([]testEvent, error) {
	var events []testEvent
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var ev testEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, fmt.Errorf("line %q: %w", string(line), err)
		}
		events = append(events, ev)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning input: %w", err)
	}

	return events, nil
}
