//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/tools"
)

func TestMySQLSourceSelectionMetadataAndBoundedQuery(t *testing.T) {
	service, registry := openIntegrationService(t, config.QuickMode)
	orders := integrationSource(t, registry, "orders")
	executeDirect(t, orders.DB(), "DROP TABLE IF EXISTS integration_rows")
	executeDirect(t, orders.DB(), "CREATE TABLE integration_rows (id INT PRIMARY KEY, label VARCHAR(32) NOT NULL)")
	t.Cleanup(func() { executeDirect(t, orders.DB(), "DROP TABLE IF EXISTS integration_rows") })

	values := make([]string, 0, 101)
	for id := 1; id <= 101; id++ {
		values = append(values, fmt.Sprintf("(%d, 'row-%d')", id, id))
	}
	executeDirect(t, orders.DB(), "INSERT INTO integration_rows (id, label) VALUES "+strings.Join(values, ","))

	sources, err := service.ListSources(context.Background(), tools.RequestMeta{})
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 2 || sources[0].ID != "analytics" || sources[0].Engine != "clickhouse" || sources[1].ID != "orders" || sources[1].Engine != "mysql" {
		t.Fatalf("sources = %#v, want both configured source IDs and engines", sources)
	}

	tables, err := service.ListTables(context.Background(), tools.RequestMeta{SourceID: "orders"})
	if err != nil || tables.State != tools.StateExecuted {
		t.Fatalf("list MySQL tables = %#v, %v", tables, err)
	}
	description, err := service.DescribeTable(context.Background(), tools.TableInput{RequestMeta: tools.RequestMeta{SourceID: "orders"}, Table: "integration_rows"})
	if err != nil || description.State != tools.StateExecuted {
		t.Fatalf("describe MySQL table = %#v, %v", description, err)
	}

	query, err := service.Query(context.Background(), tools.QueryInput{RequestMeta: tools.RequestMeta{SourceID: "orders"}, SQL: "SELECT id, label FROM integration_rows ORDER BY id"})
	if err != nil {
		t.Fatalf("bounded MySQL query: %v", err)
	}
	if query.State != tools.StateExecuted || query.Query == nil || len(query.Query.Rows) != 100 || !query.Query.Truncated {
		t.Fatalf("query = %#v, want 100 truncated rows", query)
	}
}

func TestMySQLQueryRejectsWriteAndCanceledContext(t *testing.T) {
	service, _ := openIntegrationService(t, config.QuickMode)
	_, err := service.Query(context.Background(), tools.QueryInput{RequestMeta: tools.RequestMeta{SourceID: "orders"}, SQL: "DELETE FROM integration_rows"})
	if !errors.Is(err, tools.ErrReadOnlySQLRequired) {
		t.Fatalf("write query error = %v, want read-only rejection", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = service.Query(ctx, tools.QueryInput{RequestMeta: tools.RequestMeta{SourceID: "orders"}, SQL: "SELECT 1"})
	if err == nil {
		t.Fatal("canceled query succeeded")
	}
}
