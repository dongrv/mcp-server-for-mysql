// Package observability emits bounded operational diagnostics without storing
// request history or database content.
package observability

import (
	"log/slog"
	"time"
)

// Event contains the only fields that may be emitted for an MCP request.
// It intentionally has no SQL, parameters, results, credentials, or network
// addresses.
type Event struct {
	RequestID string
	Tool      string
	SourceID  string
	State     string
	Duration  time.Duration
	ErrorCode string
}

// LogEvent writes a structured completion record. It is intentionally not an
// audit log and has no persistence or identity semantics.
func LogEvent(logger *slog.Logger, event Event) {
	if logger == nil {
		return
	}
	logger.Info("MCP tool completed",
		"request_id", event.RequestID,
		"tool", event.Tool,
		"source_id", event.SourceID,
		"state", event.State,
		"duration_ms", event.Duration.Milliseconds(),
		"error_code", event.ErrorCode,
	)
}
