//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/database"
	"github.com/dongrv/mcp-server-for-mysql/internal/tools"
)

const (
	defaultMySQLDSN      = "orders_test:orders_test@tcp(127.0.0.1:13306)/orders_test?parseTime=true"
	defaultLogsMySQLDSN  = "logs_test:logs_test@tcp(127.0.0.1:23306)/logs_test?parseTime=true"
	defaultClickHouseDSN = "clickhouse://mcp:mcp@127.0.0.1:19000/mcp_test?dial_timeout=10s"
)

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_DATABASE_INTEGRATION") != "1" {
		t.Skip("set RUN_DATABASE_INTEGRATION=1 after starting docker-compose.integration.yml")
	}
}

func openIntegrationService(t *testing.T, mode config.Mode) (*tools.Service, *database.Registry) {
	t.Helper()
	requireIntegration(t)
	t.Setenv("INTEGRATION_MYSQL_DSN", integrationMySQLDSN())
	t.Setenv("INTEGRATION_LOGS_MYSQL_DSN", integrationLogsMySQLDSN())
	t.Setenv("INTEGRATION_CLICKHOUSE_DSN", integrationClickHouseDSN())
	path := filepath.Join(t.TempDir(), "config.yaml")
	contents := fmt.Sprintf(`mode: %s
sources:
  - name: orders
    display_name: 订单库
    description: 充值与订单业务数据
    aliases: [充值库, 订单系统]
    keywords: [充值, 订单]
    type: mysql
    dsn: ${INTEGRATION_MYSQL_DSN}
  - name: logs
    display_name: 日志库
    description: 服务运行与错误日志
    aliases: [服务日志]
    keywords: [日志, 错误]
    type: mysql
    dsn: ${INTEGRATION_LOGS_MYSQL_DSN}
  - name: analytics
    display_name: 分析库
    description: 业务分析与指标数据
    aliases: [数据分析]
    keywords: [分析, 指标]
    type: clickhouse
    dsn: ${INTEGRATION_CLICKHOUSE_DSN}
`, mode)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write integration configuration: %v", err)
	}
	cfg, err := config.Load(path, os.LookupEnv)
	if err != nil {
		t.Fatalf("load integration configuration: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	registry, err := database.OpenRegistry(ctx, cfg)
	if err != nil {
		t.Fatalf("open integration registry: %v", err)
	}
	t.Cleanup(func() {
		if err := registry.Close(); err != nil {
			t.Errorf("close integration registry: %v", err)
		}
	})
	return tools.NewService(registry, cfg.Mode, nil), registry
}

func integrationMySQLDSN() string {
	if dsn := os.Getenv("INTEGRATION_MYSQL_DSN"); dsn != "" {
		return dsn
	}
	return defaultMySQLDSN
}

func integrationLogsMySQLDSN() string {
	if dsn := os.Getenv("INTEGRATION_LOGS_MYSQL_DSN"); dsn != "" {
		return dsn
	}
	return defaultLogsMySQLDSN
}

func integrationClickHouseDSN() string {
	if dsn := os.Getenv("INTEGRATION_CLICKHOUSE_DSN"); dsn != "" {
		return dsn
	}
	return defaultClickHouseDSN
}

func integrationSource(t *testing.T, registry *database.Registry, id string) database.Source {
	t.Helper()
	source, err := registry.Source(id)
	if err != nil {
		t.Fatalf("source %q: %v", id, err)
	}
	return source
}

func executeDirect(t *testing.T, db *sql.DB, statement string, args ...any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, statement, args...); err != nil {
		t.Fatalf("execute fixture SQL %q: %v", statement, err)
	}
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var name string
	err := db.QueryRowContext(context.Background(), "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", table).Scan(&name)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("check fixture table %q: %v", table, err)
	}
	return true
}
