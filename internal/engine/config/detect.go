package config

import "strings"

// Stack represents a detected technology stack.
type Stack string

const (
	// StackGo indicates a Go project (detected by go.mod).
	StackGo Stack = "go"
	// StackNode indicates a Node.js project (detected by package.json).
	StackNode Stack = "node"
	// StackPython indicates a Python project (detected by requirements.txt or pyproject.toml).
	StackPython Stack = "python"
)

// markerFiles maps file names to their corresponding stack.
var markerFiles = map[string]Stack{
	"go.mod":           StackGo,
	"package.json":     StackNode,
	"requirements.txt": StackPython,
	"pyproject.toml":   StackPython,
}

// DetectStacks scans file names for well-known marker files and returns
// the detected technology stacks. Pure function — no I/O.
// Each stack is returned at most once even if multiple markers match.
func DetectStacks(files []string) []Stack {
	seen := make(map[Stack]bool)
	var stacks []Stack

	for _, f := range files {
		if stack, ok := markerFiles[f]; ok && !seen[stack] {
			seen[stack] = true
			stacks = append(stacks, stack)
		}
	}

	return stacks
}

// GenerateGatesYAML produces a gates.yaml configuration string for the given stacks.
// If no stacks are provided, a minimal example config with commented gates is returned.
// Generated commands use diff/check mode for read-only, AI-friendly output.
func GenerateGatesYAML(stacks []Stack) string {
	if len(stacks) == 0 {
		return fallbackYAML
	}

	var b strings.Builder
	b.WriteString(yamlHeader)

	for _, s := range stacks {
		switch s {
		case StackGo:
			b.WriteString(goGates)
		case StackNode:
			b.WriteString(nodeGates)
		case StackPython:
			b.WriteString(pythonGates)
		}
	}

	return b.String()
}

const yamlHeader = `# Gatekeeper configuration — auto-generated
# Customize gates to match your project's needs.
# Docs: https://github.com/irahardianto/gatekeeper
version: 1

defaults:
  timeout: 60s
  blocking: true
  on_error: block

gates:
`

const goGates = `  # --- Go ---
  - name: go-vet
    type: exec
    command: "go vet ./..."
    container: "golang:1.23"
    only: ["*.go"]

  - name: go-test
    type: exec
    command: "go test -race ./..."
    container: "golang:1.23"
    timeout: 120s
    only: ["*.go"]

  # - name: golangci-lint
  #   type: exec
  #   command: "golangci-lint run --out-format sarif ./..."
  #   container: "golangci/golangci-lint:latest"
  #   parser: sarif
  #   only: ["*.go"]

`

const nodeGates = `  # --- Node.js ---
  - name: eslint
    type: exec
    command: "npx eslint --format json ."
    container: "node:20"
    only: ["*.js", "*.ts", "*.jsx", "*.tsx"]

  # - name: vitest
  #   type: exec
  #   command: "npx vitest run"
  #   container: "node:20"
  #   timeout: 120s
  #   only: ["*.js", "*.ts", "*.jsx", "*.tsx"]

  # - name: prettier-check
  #   type: exec
  #   command: "npx prettier --check ."
  #   container: "node:20"

`

const pythonGates = `  # --- Python ---
  - name: ruff
    type: exec
    command: "ruff check --output-format sarif ."
    container: "python:3.12"
    parser: sarif
    only: ["*.py"]

  # - name: pytest
  #   type: exec
  #   command: "pytest"
  #   container: "python:3.12"
  #   timeout: 120s
  #   only: ["*.py"]

  # - name: ruff-format-check
  #   type: exec
  #   command: "ruff format --check ."
  #   container: "python:3.12"

`

const fallbackYAML = `# Gatekeeper configuration
# No technology stack detected. Add gates below to get started.
# Docs: https://github.com/irahardianto/gatekeeper
version: 1

defaults:
  timeout: 60s
  blocking: true
  on_error: block

gates:
  # Example gate — uncomment and customize:
  # - name: lint
  #   type: exec
  #   command: "echo 'Add your linter command here'"
  #   container: "alpine:latest"
`
