//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/tools"
)

func TestClickHouseMetadataAndUnsupportedCopy(t *testing.T) {
	service, registry := openIntegrationService(t, config.QuickMode)
	analytics := integrationSource(t, registry, "analytics")
	executeDirect(t, analytics.DB(), "DROP TABLE IF EXISTS integration_events")
	executeDirect(t, analytics.DB(), "CREATE TABLE integration_events (id UInt64, event String) ENGINE = Memory")
	t.Cleanup(func() { executeDirect(t, analytics.DB(), "DROP TABLE IF EXISTS integration_events") })
	executeDirect(t, analytics.DB(), "INSERT INTO integration_events VALUES (1, 'created')")

	description, err := service.DescribeTable(context.Background(), tools.TableInput{RequestMeta: tools.RequestMeta{SourceID: "analytics"}, Table: "integration_events"})
	if err != nil || description.State != tools.StateExecuted {
		t.Fatalf("describe ClickHouse table = %#v, %v", description, err)
	}
	query, err := service.Query(context.Background(), tools.QueryInput{RequestMeta: tools.RequestMeta{SourceID: "analytics"}, SQL: "SELECT id, event FROM integration_events"})
	if err != nil || query.Query == nil || len(query.Query.Rows) != 1 {
		t.Fatalf("query ClickHouse table = %#v, %v", query, err)
	}

	_, err = service.CopyTable(context.Background(), tools.CopyTableInput{RequestMeta: tools.RequestMeta{SourceID: "analytics"}, Source: "integration_events", Destination: "integration_events_copy"})
	if !errors.Is(err, tools.ErrUnsupported) {
		t.Fatalf("ClickHouse copy error = %v, want unsupported capability", err)
	}
}

func TestClickHouseMultiStatementPreviewReportsNonAtomic(t *testing.T) {
	service, _ := openIntegrationService(t, config.QuickMode)
	response, err := service.ExecuteSQL(context.Background(), tools.ExecuteSQLInput{
		RequestMeta: tools.RequestMeta{SourceID: "analytics"},
		SQL:         "CREATE TABLE integration_batch (id UInt64) ENGINE = Memory; DROP TABLE integration_batch",
	})
	if err != nil {
		t.Fatalf("preview ClickHouse batch: %v", err)
	}
	if response.State != tools.StateConfirmationRequired || response.Preview == nil || response.Preview.Atomic {
		t.Fatalf("response = %#v, want non-atomic confirmation preview", response)
	}
}
