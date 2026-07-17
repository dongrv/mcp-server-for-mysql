package tools

import (
	"encoding/json"
)

// formatJSON serializes MCP text content without exposing implementation
// errors as an unstructured Go value.
func formatJSON(data any) string {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return `{"error":"response serialization failed"}`
	}
	return string(encoded)
}
