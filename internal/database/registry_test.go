package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
)

type fakeSource struct {
	id         string
	engine     string
	profile    SourceProfile
	closeErr   error
	closeCalls int
}

func newFakeSource(id string) *fakeSource {
	return &fakeSource{id: id, engine: "mysql"}
}

func (s *fakeSource) ID() string             { return s.id }
func (s *fakeSource) Engine() string         { return s.engine }
func (s *fakeSource) Profile() SourceProfile { return cloneSourceProfile(s.profile) }
func (s *fakeSource) DB() *sql.DB            { return nil }
func (s *fakeSource) Dialect() Dialect       { return MySQLDialect{} }
func (s *fakeSource) Capabilities() Capability {
	return MySQLDialect{}.Capabilities()
}
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

func TestRegistrySourcesPreserveProfiles(t *testing.T) {
	analyticsProfile := SourceProfile{
		DisplayName: "Analytics",
		Description: "Aggregated event analytics",
		Aliases:     []string{"warehouse"},
		Keywords:    []string{"events", "metrics"},
	}
	ordersProfile := SourceProfile{
		DisplayName: "Orders",
		Description: "Customer payment orders",
		Aliases:     []string{"payments"},
		Keywords:    []string{"orders", "refunds"},
	}
	analytics := newFakeSource("analytics")
	analytics.profile = analyticsProfile
	orders := newFakeSource("orders")
	orders.profile = ordersProfile
	registry := mustNewRegistry(t, []Source{orders, analytics})

	sources := registry.Sources()
	if len(sources) != 2 {
		t.Fatalf("Sources() length = %d, want 2", len(sources))
	}
	if sources[0].ID() != "analytics" || sources[1].ID() != "orders" {
		t.Fatalf("Sources() IDs = %v, want [analytics orders]", []string{sources[0].ID(), sources[1].ID()})
	}
	if got := sources[0].Profile(); !reflect.DeepEqual(got, analyticsProfile) {
		t.Errorf("analytics profile = %#v, want %#v", got, analyticsProfile)
	}
	if got := sources[1].Profile(); !reflect.DeepEqual(got, ordersProfile) {
		t.Errorf("orders profile = %#v, want %#v", got, ordersProfile)
	}
}

func TestSourceProfileIsDefensivelyCopied(t *testing.T) {
	source := &sqlSource{profile: SourceProfile{
		DisplayName: "Orders",
		Description: "Customer payment orders",
		Aliases:     []string{"payments"},
		Keywords:    []string{"orders"},
	}}

	profile := source.Profile()
	profile.Aliases[0] = "mutated alias"
	profile.Keywords[0] = "mutated keyword"

	got := source.Profile()
	if got.Aliases[0] != "payments" {
		t.Errorf("Profile().Aliases = %v, want stored aliases unchanged", got.Aliases)
	}
	if got.Keywords[0] != "orders" {
		t.Errorf("Profile().Keywords = %v, want stored keywords unchanged", got.Keywords)
	}
}

func TestSourceProfileFromConfigMapsBusinessMetadataOnly(t *testing.T) {
	cfg := config.SourceConfig{
		Name:        "orders",
		DisplayName: "Orders",
		Description: "Customer payment orders",
		Aliases:     []string{"payments"},
		Keywords:    []string{"orders", "refunds"},
		Type:        "mysql",
		DSN:         "user:super-secret-password@tcp(database.example:3306)/orders",
	}
	source := &sqlSource{profile: profileFromConfig(cfg)}
	cfg.Aliases[0] = "mutated alias"
	cfg.Keywords[0] = "mutated keyword"

	got := source.Profile()
	want := SourceProfile{
		DisplayName: "Orders",
		Description: "Customer payment orders",
		Aliases:     []string{"payments"},
		Keywords:    []string{"orders", "refunds"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Profile() = %#v, want %#v", got, want)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal(Profile()) error = %v", err)
	}
	if strings.Contains(string(encoded), cfg.DSN) || strings.Contains(string(encoded), "super-secret-password") {
		t.Fatalf("profile leaked connection details: %s", encoded)
	}
}

func TestSourceProfilePreservesNilAndEmptySlices(t *testing.T) {
	tests := []struct {
		name     string
		aliases  []string
		keywords []string
		wantJSON string
	}{
		{
			name:     "nil slices",
			wantJSON: `{"display_name":"Orders","description":"Customer payment orders","aliases":null,"keywords":null}`,
		},
		{
			name:     "non-nil empty slices",
			aliases:  []string{},
			keywords: []string{},
			wantJSON: `{"display_name":"Orders","description":"Customer payment orders","aliases":[],"keywords":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &sqlSource{profile: profileFromConfig(config.SourceConfig{
				DisplayName: "Orders",
				Description: "Customer payment orders",
				Aliases:     tt.aliases,
				Keywords:    tt.keywords,
			})}

			first := source.Profile()
			if (first.Aliases == nil) != (tt.aliases == nil) {
				t.Fatalf("first Profile().Aliases nil = %t, want %t", first.Aliases == nil, tt.aliases == nil)
			}
			if (first.Keywords == nil) != (tt.keywords == nil) {
				t.Fatalf("first Profile().Keywords nil = %t, want %t", first.Keywords == nil, tt.keywords == nil)
			}
			first.Aliases = append(first.Aliases, "caller alias")
			first.Keywords = append(first.Keywords, "caller keyword")

			second := source.Profile()
			if len(second.Aliases) != 0 || (second.Aliases == nil) != (tt.aliases == nil) {
				t.Errorf("second Profile().Aliases = %#v, want original slice presence", second.Aliases)
			}
			if len(second.Keywords) != 0 || (second.Keywords == nil) != (tt.keywords == nil) {
				t.Errorf("second Profile().Keywords = %#v, want original slice presence", second.Keywords)
			}
			encoded, err := json.Marshal(second)
			if err != nil {
				t.Fatalf("json.Marshal(Profile()) error = %v", err)
			}
			if string(encoded) != tt.wantJSON {
				t.Errorf("json.Marshal(Profile()) = %s, want %s", encoded, tt.wantJSON)
			}
		})
	}
}

func TestOpenRegistryPreservesSourceConfigProfiles(t *testing.T) {
	cfg := config.Config{Sources: []config.SourceConfig{
		{
			Name: "orders", DisplayName: "Orders", Description: "Customer payment orders",
			Aliases: []string{"payments"}, Keywords: []string{"orders", "refunds"}, Type: "mysql", DSN: "orders-secret",
		},
		{
			Name: "analytics", DisplayName: "Analytics", Description: "Aggregated event analytics",
			Aliases: []string{"warehouse"}, Keywords: []string{"events", "metrics"}, Type: "clickhouse", DSN: "analytics-secret",
		},
	}}
	factory := func(_ context.Context, sourceConfig config.SourceConfig) (Source, error) {
		source := newFakeSource(sourceConfig.Name)
		source.engine = sourceConfig.Type
		source.profile = profileFromConfig(sourceConfig)
		return source, nil
	}

	registry, err := openRegistry(context.Background(), cfg, factory)
	if err != nil {
		t.Fatalf("openRegistry() error = %v", err)
	}
	t.Cleanup(func() { _ = registry.Close() })
	sources := registry.Sources()
	if got := sources[0].Profile(); !reflect.DeepEqual(got, profileFromConfig(cfg.Sources[1])) {
		t.Errorf("analytics profile = %#v, want %#v", got, profileFromConfig(cfg.Sources[1]))
	}
	if got := sources[1].Profile(); !reflect.DeepEqual(got, profileFromConfig(cfg.Sources[0])) {
		t.Errorf("orders profile = %#v, want %#v", got, profileFromConfig(cfg.Sources[0]))
	}
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
	if caps.Transactions || caps.AtomicBatches || !caps.AlterColumns || caps.CopyTable {
		t.Fatalf("ClickHouse capabilities = %#v, want transactions and atomic batches disabled", caps)
	}
	mysqlCaps := (MySQLDialect{}).Capabilities()
	if !mysqlCaps.Transactions || !mysqlCaps.AtomicBatches || !mysqlCaps.CopyTable || !mysqlCaps.AlterColumns {
		t.Fatal("MySQL must advertise its supported transaction, copy, and alter operations")
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
