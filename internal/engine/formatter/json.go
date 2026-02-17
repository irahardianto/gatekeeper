package formatter

import (
	"encoding/json"
)

// JSONFormatter outputs RunResult as pretty-printed JSON.
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSONFormatter.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Format returns the RunResult as indented JSON.
func (f *JSONFormatter) Format(result RunResult) string {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		// Fallback: should never happen since RunResult is fully serializable.
		return `{"error": "failed to marshal result"}`
	}
	return string(data)
}
