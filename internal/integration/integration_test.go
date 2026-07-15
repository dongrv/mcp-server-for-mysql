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
	defaultMySQLDSN      = "mcp:mcp@tcp(127.0.0.1:13306)/mcp_test?parseTime=true"
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
	t.Setenv("INTEGRATION_CLICKHOUSE_DSN", integrationClickHouseDSN())
	path := filepath.Join(t.TempDir(), "config.yaml")
	contents := fmt.Sprintf("mode: %s\nsources:\n  - name: orders\n    type: mysql\n    dsn: ${INTEGRATION_MYSQL_DSN}\n  - name: analytics\n    type: clickhouse\n    dsn: ${INTEGRATION_CLICKHOUSE_DSN}\n", mode)
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
