package parser

import (
	"context"
	"sync"
)

// ParseResult holds the outcome of a parser execution.
type ParseResult struct {
	Passed bool
	Errors []StructuredError
}

// StructuredError represents a single issue found by a tool.
type StructuredError struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"`       // error, warning, info
	Rule     string `json:"rule,omitempty"` // e.g., "gosec:G101"
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
	Tool     string `json:"tool"`
}

// Parser parses raw tool output into structured results.
type Parser interface {
	Parse(ctx context.Context, stdout, stderr []byte, exitCode int) (*ParseResult, error)
}

// Registry manages available parsers.
type Registry struct {
	parsers map[string]Parser
	mu      sync.RWMutex
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[string]Parser),
	}
}

// Register adds a parser to the registry.
func (r *Registry) Register(name string, p Parser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parsers[name] = p
}

// Get returns a parser by name. Returns nil if not found.
func (r *Registry) Get(name string) Parser {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.parsers[name]
}

// GetOrDefault returns a parser by name, or the generic parser if not found.
func (r *Registry) GetOrDefault(name string) Parser {
	p := r.Get(name)
	if p != nil {
		return p
	}
	return NewGenericParser()
}
