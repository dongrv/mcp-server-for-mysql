package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
)

type fakeSource struct {
	id         string
	engine     string
	closeErr   error
	closeCalls int
}

func newFakeSource(id string) *fakeSource {
	return &fakeSource{id: id, engine: "mysql"}
}

func (s *fakeSource) ID() string               { return s.id }
func (s *fakeSource) Engine() string           { return s.engine }
func (s *fakeSource) DB() *sql.DB              { return nil }
func (s *fakeSource) Dialect() Dialect         { return MySQLDialect{} }
func (s *fakeSource) Capabilities() Capability { return MySQLDialect{}.Capabilities() }
func (s *fakeSource) Close() error {
	s.closeCalls++
	return s.closeErr
}

func mustNewRegistry(t *testing.T, sources []Source) *Registry {
	t.Helper()
	registry, err := NewRegistry(sources)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	return registry
}

func TestRegistryRejectsUnknownSourceAndClosesEveryPoolOnce(t *testing.T) {
	first, second := newFakeSource("orders"), newFakeSource("analytics")
	registry := mustNewRegistry(t, []Source{first, second})

	if _, err := registry.Source("missing"); !errors.Is(err, ErrUnknownSource) {
		t.Fatalf("Source(missing) error = %v, want ErrUnknownSource", err)
	}
	if err := registry.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := registry.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if first.closeCalls != 1 || second.closeCalls != 1 {
		t.Fatalf("close calls = (%d, %d), want (1, 1)", first.closeCalls, second.closeCalls)
	}
}

func TestRegistryCloseAggregatesErrors(t *testing.T) {
	firstErr, secondErr := errors.New("first close"), errors.New("second close")
	first, second := newFakeSource("first"), newFakeSource("second")
	first.closeErr = firstErr
	second.closeErr = secondErr
	registry := mustNewRegistry(t, []Source{
		first,
		second,
	})

	err := registry.Close()
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("Close() error = %v, want both close errors", err)
	}
}

func TestNewRegistryRejectsDuplicateSourceIDAndClosesSources(t *testing.T) {
	first, second := newFakeSource("orders"), newFakeSource("orders")

	registry, err := NewRegistry([]Source{first, second})
	if registry != nil {
		t.Fatal("NewRegistry() returned a registry for duplicate source IDs")
	}
	if !errors.Is(err, ErrDuplicateSource) {
		t.Fatalf("NewRegistry() error = %v, want ErrDuplicateSource", err)
	}
	if first.closeCalls != 1 || second.closeCalls != 1 {
		t.Fatalf("close calls = (%d, %d), want (1, 1)", first.closeCalls, second.closeCalls)
	}
}

func TestOpenClickHouseDBBuildsSQLHandleWithoutConnecting(t *testing.T) {
	db, err := openClickHouseDB("clickhouse://test:test@localhost:9000/default")
	if err != nil {
		t.Fatalf("openClickHouseDB() error = %v", err)
	}
	if db == nil {
		t.Fatal("openClickHouseDB() returned a nil database handle")
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestMySQLIdentifierRejectsInjectionAndQuotesSafeNames(t *testing.T) {
	dialect := MySQLDialect{}
	got, err := dialect.QuoteIdentifier("order_items")
	if err != nil {
		t.Fatalf("QuoteIdentifier() error = %v", err)
	}
	if got != "`order_items`" {
		t.Fatalf("QuoteIdentifier() = %q, want %q", got, "`order_items`")
	}
	if err := dialect.ValidateIdentifier("orders`; DROP TABLE users; --"); err == nil {
		t.Fatal("ValidateIdentifier() accepted a SQL fragment")
	}
}

func TestClickHouseCapabilitiesAreExplicit(t *testing.T) {
	caps := ClickHouseDialect{}.Capabilities()
	if caps.Transactions || caps.AtomicBatches {
		t.Fatalf("ClickHouse capabilities = %#v, want transactions and atomic batches disabled", caps)
	}
	mysqlCaps := (MySQLDialect{}).Capabilities()
	if !mysqlCaps.Transactions || !mysqlCaps.AtomicBatches {
		t.Fatal("MySQL must advertise transactions and atomic batches")
	}
}

func TestOpenRegistryClosesPartialRegistryOnFactoryFailure(t *testing.T) {
	first := newFakeSource("orders")
	factoryErr := errors.New("cannot open source")
	factory := func(_ context.Context, source config.SourceConfig) (Source, error) {
		if source.Name == "orders" {
			return first, nil
		}
		return nil, factoryErr
	}

	_, err := openRegistry(context.Background(), config.Config{Sources: []config.SourceConfig{
		{Name: "orders", Type: "mysql"},
		{Name: "analytics", Type: "clickhouse"},
	}}, factory)
	if !errors.Is(err, factoryErr) {
		t.Fatalf("openRegistry() error = %v, want factory error", err)
	}
	if first.closeCalls != 1 {
		t.Fatalf("partial source close calls = %d, want 1", first.closeCalls)
	}
}

func TestOpenRegistryClosesSourcesOnDuplicateSourceID(t *testing.T) {
	first, second := newFakeSource("orders"), newFakeSource("orders")
	factoryCalls := 0
	factory := func(_ context.Context, _ config.SourceConfig) (Source, error) {
		factoryCalls++
		if factoryCalls == 1 {
			return first, nil
		}
		return second, nil
	}

	registry, err := openRegistry(context.Background(), config.Config{Sources: []config.SourceConfig{
		{Name: "orders", Type: "mysql"},
		{Name: "analytics", Type: "clickhouse"},
	}}, factory)
	if registry != nil {
		t.Fatal("openRegistry() returned a registry for duplicate source IDs")
	}
	if !errors.Is(err, ErrDuplicateSource) {
		t.Fatalf("openRegistry() error = %v, want ErrDuplicateSource", err)
	}
	if first.closeCalls != 1 || second.closeCalls != 1 {
		t.Fatalf("close calls = (%d, %d), want (1, 1)", first.closeCalls, second.closeCalls)
	}
}

func TestMetadataRejectsUnsafeTableIdentifiersBeforeQuerying(t *testing.T) {
	for _, dialect := range []Dialect{MySQLDialect{}, ClickHouseDialect{}} {
		if _, err := dialect.DescribeTable(context.Background(), nil, "orders; DROP TABLE users"); err == nil {
			t.Fatalf("%s DescribeTable() accepted a SQL fragment", dialect.Name())
		}
	}
}
