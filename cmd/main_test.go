package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/database"
)

func TestBuildApplicationRegistersGenericToolsForMultipleSources(t *testing.T) {
	cfg := config.Config{Mode: config.QuickMode, Sources: []config.SourceConfig{
		{Name: "orders", Type: "mysql", DSN: "mysql-fake"},
		{Name: "analytics", Type: "clickhouse", DSN: "clickhouse-fake"},
	}}
	var opened config.Config
	app, err := buildApplication(context.Background(), cfg, func(_ context.Context, received config.Config) (*database.Registry, error) {
		opened = received
		return database.NewRegistry(nil)
	})
	if err != nil {
		t.Fatalf("buildApplication() error = %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })
	if len(opened.Sources) != 2 {
		t.Fatalf("registry received %d sources, want 2", len(opened.Sources))
	}
	if !hasTool(app.ToolNames(), "list_sources") {
		t.Errorf("registered tools = %v, want list_sources", app.ToolNames())
	}
	for _, name := range app.ToolNames() {
		if strings.HasPrefix(name, "mysql_") || strings.Contains(name, "transaction") {
			t.Errorf("registered tools include stateful legacy tool %q", name)
		}
	}
}

func TestBuildApplicationRejectsNilRegistryOpener(t *testing.T) {
	if _, err := buildApplication(context.Background(), config.Config{}, nil); err == nil {
		t.Fatal("buildApplication() error = nil, want missing opener error")
	}
}

func TestApplicationCloseClosesRegistryExactlyOnce(t *testing.T) {
	source := &closeCountingSource{}
	app, err := buildApplication(context.Background(), config.Config{Mode: config.QuickMode}, func(context.Context, config.Config) (*database.Registry, error) {
		return database.NewRegistry([]database.Source{source})
	})
	if err != nil {
		t.Fatalf("buildApplication() error = %v", err)
	}
	if err := app.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := app.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if source.closed != 1 {
		t.Errorf("source close count = %d, want 1", source.closed)
	}
}

type closeCountingSource struct{ closed int }

func (*closeCountingSource) ID() string                { return "orders" }
func (*closeCountingSource) Engine() string            { return "mysql" }
func (*closeCountingSource) DB() *sql.DB               { return nil }
func (*closeCountingSource) Dialect() database.Dialect { return database.MySQLDialect{} }
func (*closeCountingSource) Capabilities() database.Capability {
	return database.MySQLDialect{}.Capabilities()
}
func (s *closeCountingSource) Close() error {
	s.closed++
	return nil
}

func hasTool(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}
