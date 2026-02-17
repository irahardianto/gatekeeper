package parser

import "context"

// MockParser is a test double for Parser.
type MockParser struct {
	Result *ParseResult
	Err    error
}

func (m *MockParser) Parse(_ context.Context, _, _ []byte, _ int) (*ParseResult, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Result, nil
}
