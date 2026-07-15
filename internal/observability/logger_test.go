package observability

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

func TestLogEventEmitsOnlyDeclaredOperationalFields(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	LogEvent(logger, Event{
		RequestID: "request-1", Tool: "query", SourceID: "orders", State: "executed",
		Duration: 125 * time.Millisecond, ErrorCode: "",
	})

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode log record: %v", err)
	}
	for key := range record {
		switch key {
		case "time", "level", "msg", "request_id", "tool", "source_id", "state", "duration_ms", "error_code":
		default:
			t.Errorf("unexpected logged field %q", key)
		}
	}
	for _, forbidden := range []string{"sql", "parameters", "result", "dsn", "host", "username", "password"} {
		if _, ok := record[forbidden]; ok {
			t.Errorf("sensitive field %q was logged", forbidden)
		}
	}
}
