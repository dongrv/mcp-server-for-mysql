//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/database"
	"github.com/dongrv/mcp-server-for-mysql/internal/tools"
	"gopkg.in/yaml.v3"
)

const (
	defaultMySQLPort      = "13306"
	defaultLogsMySQLPort  = "23306"
	defaultClickHousePort = "19000"
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
	port := os.Getenv("INTEGRATION_MYSQL_PORT")
	if port == "" {
		port = defaultMySQLPort
	}
	return fmt.Sprintf("orders_test:orders_test@tcp(127.0.0.1:%s)/orders_test?parseTime=true", port)
}

func integrationLogsMySQLDSN() string {
	if dsn := os.Getenv("INTEGRATION_LOGS_MYSQL_DSN"); dsn != "" {
		return dsn
	}
	port := os.Getenv("INTEGRATION_LOGS_MYSQL_PORT")
	if port == "" {
		port = defaultLogsMySQLPort
	}
	return fmt.Sprintf("logs_test:logs_test@tcp(127.0.0.1:%s)/logs_test?parseTime=true", port)
}

func integrationClickHouseDSN() string {
	if dsn := os.Getenv("INTEGRATION_CLICKHOUSE_DSN"); dsn != "" {
		return dsn
	}
	port := os.Getenv("INTEGRATION_CLICKHOUSE_PORT")
	if port == "" {
		port = defaultClickHousePort
	}
	return fmt.Sprintf("clickhouse://mcp:mcp@127.0.0.1:%s/mcp_test?dial_timeout=10s", port)
}

func TestIntegrationDSNsUseConfiguredPorts(t *testing.T) {
	t.Setenv("INTEGRATION_MYSQL_DSN", "")
	t.Setenv("INTEGRATION_LOGS_MYSQL_DSN", "")
	t.Setenv("INTEGRATION_CLICKHOUSE_DSN", "")
	t.Setenv("INTEGRATION_MYSQL_PORT", "33306")
	t.Setenv("INTEGRATION_LOGS_MYSQL_PORT", "43306")
	t.Setenv("INTEGRATION_CLICKHOUSE_PORT", "49000")

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "orders", got: integrationMySQLDSN(), want: "orders_test:orders_test@tcp(127.0.0.1:33306)/orders_test?parseTime=true"},
		{name: "logs", got: integrationLogsMySQLDSN(), want: "logs_test:logs_test@tcp(127.0.0.1:43306)/logs_test?parseTime=true"},
		{name: "clickhouse", got: integrationClickHouseDSN(), want: "clickhouse://mcp:mcp@127.0.0.1:49000/mcp_test?dial_timeout=10s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("integration DSN = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestIntegrationDSNOverridesTakePrecedence(t *testing.T) {
	t.Setenv("INTEGRATION_MYSQL_DSN", "orders-override")
	t.Setenv("INTEGRATION_LOGS_MYSQL_DSN", "logs-override")
	t.Setenv("INTEGRATION_CLICKHOUSE_DSN", "clickhouse-override")
	t.Setenv("INTEGRATION_MYSQL_PORT", "33306")
	t.Setenv("INTEGRATION_LOGS_MYSQL_PORT", "43306")
	t.Setenv("INTEGRATION_CLICKHOUSE_PORT", "49000")

	if got := integrationMySQLDSN(); got != "orders-override" {
		t.Fatalf("orders DSN override = %q, want orders-override", got)
	}
	if got := integrationLogsMySQLDSN(); got != "logs-override" {
		t.Fatalf("logs DSN override = %q, want logs-override", got)
	}
	if got := integrationClickHouseDSN(); got != "clickhouse-override" {
		t.Fatalf("ClickHouse DSN override = %q, want clickhouse-override", got)
	}
}

type integrationComposeService struct {
	Ports       []string `yaml:"ports"`
	Healthcheck struct {
		Test []string `yaml:"test"`
	} `yaml:"healthcheck"`
}

func loadIntegrationComposeServices(t *testing.T) map[string]integrationComposeService {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("..", "..", "docker-compose.integration.yml"))
	if err != nil {
		t.Fatalf("read integration Compose file: %v", err)
	}
	var compose struct {
		Services map[string]integrationComposeService `yaml:"services"`
	}
	if err := yaml.Unmarshal(contents, &compose); err != nil {
		t.Fatalf("parse integration Compose file: %v", err)
	}
	return compose.Services
}

func TestIntegrationComposeUsesLoopbackParameterizedPorts(t *testing.T) {
	services := loadIntegrationComposeServices(t)
	want := map[string][]string{
		"mysql-orders": {"127.0.0.1:${INTEGRATION_MYSQL_PORT:-13306}:3306"},
		"mysql-logs":   {"127.0.0.1:${INTEGRATION_LOGS_MYSQL_PORT:-23306}:3306"},
		"clickhouse": {
			"127.0.0.1:${INTEGRATION_CLICKHOUSE_HTTP_PORT:-18123}:8123",
			"127.0.0.1:${INTEGRATION_CLICKHOUSE_PORT:-19000}:9000",
		},
	}
	for service, wantPorts := range want {
		if got := services[service].Ports; !slices.Equal(got, wantPorts) {
			t.Errorf("%s ports = %v, want %v", service, got, wantPorts)
		}
	}
}

func TestIntegrationComposeUsesAuthenticatedDatabaseHealthChecks(t *testing.T) {
	services := loadIntegrationComposeServices(t)
	want := []string{
		"CMD-SHELL",
		`MYSQL_PWD="$${MYSQL_PASSWORD}" mysql --protocol=TCP --host=127.0.0.1 --user="$${MYSQL_USER}" --database="$${MYSQL_DATABASE}" --execute='SELECT 1' --silent`,
	}
	for _, service := range []string{"mysql-orders", "mysql-logs"} {
		if got := services[service].Healthcheck.Test; !slices.Equal(got, want) {
			t.Errorf("%s health check = %v, want %v", service, got, want)
		}
	}
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
