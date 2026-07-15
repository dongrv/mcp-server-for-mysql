//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/database"
	"github.com/dongrv/mcp-server-for-mysql/internal/tools"
	mysql "github.com/go-sql-driver/mysql"
)

func TestMultipleSameEngineSourceProfiles(t *testing.T) {
	service, registry := openIntegrationService(t, config.QuickMode)

	sources, err := service.ListSources(context.Background(), tools.RequestMeta{})
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	encoded, err := json.Marshal(sources)
	if err != nil {
		t.Fatalf("marshal sources: %v", err)
	}
	want := `[{"id":"analytics","engine":"clickhouse","display_name":"分析库","description":"业务分析与指标数据","aliases":["数据分析"],"keywords":["分析","指标"]},{"id":"logs","engine":"mysql","display_name":"日志库","description":"服务运行与错误日志","aliases":["服务日志"],"keywords":["日志","错误"]},{"id":"orders","engine":"mysql","display_name":"订单库","description":"充值与订单业务数据","aliases":["充值库","订单系统"],"keywords":["充值","订单"]}]`
	if string(encoded) != want {
		t.Fatalf("list_sources JSON = %s, want %s", encoded, want)
	}
	for _, forbidden := range []string{"dsn", "host", "username", "password", "pool", "capabilities", "127.0.0.1", "13306", "23306"} {
		if strings.Contains(strings.ToLower(string(encoded)), forbidden) {
			t.Errorf("list_sources JSON contains forbidden connection detail %q: %s", forbidden, encoded)
		}
	}

	orders := integrationSource(t, registry, "orders")
	logs := integrationSource(t, registry, "logs")
	executeDirect(t, orders.DB(), "DROP TABLE IF EXISTS orders_instance_marker")
	executeDirect(t, logs.DB(), "DROP TABLE IF EXISTS logs_instance_marker")
	executeDirect(t, orders.DB(), "CREATE TABLE orders_instance_marker (marker VARCHAR(64) NOT NULL)")
	executeDirect(t, logs.DB(), "CREATE TABLE logs_instance_marker (marker VARCHAR(64) NOT NULL)")
	t.Cleanup(func() {
		executeDirect(t, orders.DB(), "DROP TABLE IF EXISTS orders_instance_marker")
		executeDirect(t, logs.DB(), "DROP TABLE IF EXISTS logs_instance_marker")
	})
	executeDirect(t, orders.DB(), "INSERT INTO orders_instance_marker (marker) VALUES ('orders-instance')")
	executeDirect(t, logs.DB(), "INSERT INTO logs_instance_marker (marker) VALUES ('logs-instance')")

	ordersUUID := assertInstanceMarker(t, service, "orders", "orders_instance_marker", "orders-instance", "orders_test")
	logsUUID := assertInstanceMarker(t, service, "logs", "logs_instance_marker", "logs-instance", "logs_test")
	if ordersUUID == logsUUID {
		t.Fatalf("orders and logs server UUIDs are both %q, want physically separate MySQL instances", ordersUUID)
	}
	assertSourceCannotQueryTable(t, service, "orders", "logs_instance_marker")
	assertSourceCannotQueryTable(t, service, "logs", "orders_instance_marker")
	_, err = service.Query(context.Background(), tools.QueryInput{
		RequestMeta: tools.RequestMeta{SourceID: "订单库"},
		SQL:         "SELECT marker FROM orders_instance_marker",
	})
	if !errors.Is(err, database.ErrUnknownSource) {
		t.Fatalf("display-name source selection error = %v, want unknown exact source ID", err)
	}
}

func assertInstanceMarker(t *testing.T, service *tools.Service, sourceID, table, wantMarker, wantDatabase string) string {
	t.Helper()
	response, err := service.Query(context.Background(), tools.QueryInput{
		RequestMeta: tools.RequestMeta{SourceID: sourceID},
		SQL:         fmt.Sprintf("SELECT marker, DATABASE() AS database_name, @@server_uuid AS server_uuid FROM %s", table),
	})
	if err != nil {
		t.Fatalf("query %s marker through exact source ID: %v", sourceID, err)
	}
	if response.State != tools.StateExecuted || response.Query == nil || len(response.Query.Rows) != 1 || len(response.Query.Rows[0]) != 3 {
		t.Fatalf("%s marker query = %#v, want one marker/database/server UUID row", sourceID, response)
	}
	row := response.Query.Rows[0]
	if marker := fmt.Sprint(row[0]); marker != wantMarker {
		t.Fatalf("%s marker = %q, want %q", sourceID, marker, wantMarker)
	}
	if databaseName := fmt.Sprint(row[1]); databaseName != wantDatabase {
		t.Fatalf("%s database = %q, want %q", sourceID, databaseName, wantDatabase)
	}
	serverUUID := fmt.Sprint(row[2])
	if serverUUID == "" || serverUUID == "<nil>" {
		t.Fatalf("%s server UUID = %q, want non-empty physical-instance identity", sourceID, serverUUID)
	}
	return serverUUID
}

func TestIsMySQLMissingTableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "missing table", err: &mysql.MySQLError{Number: 1146}, want: true},
		{name: "wrapped missing table", err: fmt.Errorf("query failed: %w", &mysql.MySQLError{Number: 1146}), want: true},
		{name: "access denied", err: &mysql.MySQLError{Number: 1045}, want: false},
		{name: "unrelated error", err: errors.New("context canceled"), want: false},
		{name: "nil", err: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMySQLMissingTableError(tt.err); got != tt.want {
				t.Fatalf("isMySQLMissingTableError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func isMySQLMissingTableError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1146
}

func assertSourceCannotQueryTable(t *testing.T, service *tools.Service, sourceID, table string) {
	t.Helper()
	_, err := service.Query(context.Background(), tools.QueryInput{
		RequestMeta: tools.RequestMeta{SourceID: sourceID},
		SQL:         fmt.Sprintf("SELECT marker FROM %s", table),
	})
	if !isMySQLMissingTableError(err) {
		t.Fatalf("source %q query error for other-instance table %q = %v, want MySQL missing-table error 1146", sourceID, table, err)
	}
}

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
	if len(sources) != 3 || sources[0].ID != "analytics" || sources[0].Engine != "clickhouse" || sources[1].ID != "logs" || sources[1].Engine != "mysql" || sources[2].ID != "orders" || sources[2].Engine != "mysql" {
		t.Fatalf("sources = %#v, want all configured source IDs and engines", sources)
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
