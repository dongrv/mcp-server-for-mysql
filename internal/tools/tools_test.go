package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/database"
	"github.com/dongrv/mcp-server-for-mysql/internal/execution"
)

type serviceTestSource struct {
	id      string
	engine  string
	profile database.SourceProfile
	db      *sql.DB
	dialect database.Dialect
	caps    database.Capability
	secret  string
}

func (s serviceTestSource) ID() string     { return s.id }
func (s serviceTestSource) Engine() string { return s.engine }
func (s serviceTestSource) Profile() database.SourceProfile {
	profile := s.profile
	if profile.DisplayName == "" {
		profile = database.SourceProfile{
			DisplayName: s.id + " test source",
			Description: "Test database source for " + s.id,
			Aliases:     []string{s.id},
			Keywords:    []string{s.engine},
		}
	}
	if profile.Aliases != nil {
		profile.Aliases = append([]string{}, profile.Aliases...)
	}
	if profile.Keywords != nil {
		profile.Keywords = append([]string{}, profile.Keywords...)
	}
	return profile
}
func (s serviceTestSource) DB() *sql.DB                       { return s.db }
func (s serviceTestSource) Dialect() database.Dialect         { return s.dialect }
func (s serviceTestSource) Capabilities() database.Capability { return s.caps }
func (s serviceTestSource) Close() error                      { return nil }

func newServiceWithSource(t *testing.T, mode config.Mode, source database.Source) *Service {
	t.Helper()
	registry, err := database.NewRegistry([]database.Source{source})
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	return NewService(registry, mode, nil)
}

func TestServiceTestSourceProfilePreservesSlicePresence(t *testing.T) {
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
			source := serviceTestSource{profile: database.SourceProfile{
				DisplayName: "Orders",
				Description: "Customer payment orders",
				Aliases:     tt.aliases,
				Keywords:    tt.keywords,
			}}

			first := source.Profile()
			first.Aliases = append(first.Aliases, "caller alias")
			first.Keywords = append(first.Keywords, "caller keyword")
			second := source.Profile()
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

func TestQueryRejectsWriteSQLBeforeOpeningDatabase(t *testing.T) {
	source := serviceTestSource{
		id: "orders", engine: "mysql", dialect: database.MySQLDialect{},
		caps: database.MySQLDialect{}.Capabilities(),
	}
	service := newServiceWithSource(t, config.QuickMode, source)

	_, err := service.Query(context.Background(), QueryInput{
		RequestMeta: RequestMeta{SourceID: "orders"},
		SQL:         "DELETE FROM orders",
	})
	if !errors.Is(err, ErrReadOnlySQLRequired) {
		t.Fatalf("query error = %v, want %v", err, ErrReadOnlySQLRequired)
	}
}

func TestQueryAcceptsSafeExplainSelectForBothDialects(t *testing.T) {
	tests := []struct {
		name    string
		engine  string
		dialect database.Dialect
		sql     string
	}{
		{"mysql", "mysql", database.MySQLDialect{}, "EXPLAIN SELECT id FROM orders"},
		{"clickhouse", "clickhouse", database.ClickHouseDialect{}, "EXPLAIN SYNTAX SELECT id FROM events"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("new mock: %v", err)
			}
			t.Cleanup(func() { _ = db.Close() })
			service := newServiceWithSource(t, config.QuickMode, serviceTestSource{
				id: "source", engine: test.engine, db: db, dialect: test.dialect, caps: test.dialect.Capabilities(),
			})
			mock.ExpectQuery("(?i)explain").WillReturnRows(sqlmock.NewRows([]string{"plan"}).AddRow("ok"))
			response, err := service.Query(context.Background(), QueryInput{RequestMeta: RequestMeta{SourceID: "source"}, SQL: test.sql})
			if err != nil || response.State != StateExecuted || response.Query == nil {
				t.Fatalf("query = %#v, %v", response, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDropTableReturnsPreviewThenExecutesMatchingConfirmation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	source := serviceTestSource{
		id: "orders", engine: "mysql", db: db, dialect: database.MySQLDialect{},
		caps: database.MySQLDialect{}.Capabilities(),
		profile: database.SourceProfile{
			DisplayName: "Orders", Description: "Customer payment orders",
		},
	}
	service := newServiceWithSource(t, config.QuickMode, source)

	first, err := service.DropTable(context.Background(), DropTableInput{
		RequestMeta: RequestMeta{SourceID: "orders"}, Table: "orders",
	})
	if err != nil {
		t.Fatalf("preview drop table: %v", err)
	}
	if first.State != StateConfirmationRequired || first.Preview == nil || first.Preview.PreviewHash == "" {
		t.Fatalf("first response = %#v, want confirmation preview", first)
	}
	if first.Preview.Source == nil || first.Preview.Source.ID != "orders" || first.Preview.Source.DisplayName != "Orders" {
		t.Fatalf("preview source = %#v, want orders source reference", first.Preview.Source)
	}

	mock.ExpectExec("(?i)drop table orders").WillReturnResult(sqlmock.NewResult(0, 0))
	second, err := service.DropTable(context.Background(), DropTableInput{
		RequestMeta: RequestMeta{SourceID: "orders", Confirm: true, PreviewHash: first.Preview.PreviewHash},
		Table:       "orders",
	})
	if err != nil {
		t.Fatalf("execute drop table: %v", err)
	}
	if second.State != StateExecuted || second.Execution == nil {
		t.Fatalf("second response = %#v, want executed response", second)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPreviewHashDoesNotBindSourceDisplayName(t *testing.T) {
	source := &serviceTestSource{
		id: "orders", engine: "mysql", dialect: database.MySQLDialect{},
		caps: database.MySQLDialect{}.Capabilities(),
		profile: database.SourceProfile{
			DisplayName: "Orders", Description: "Customer payment orders",
		},
	}
	service := newServiceWithSource(t, config.QuickMode, source)
	input := DropTableInput{RequestMeta: RequestMeta{SourceID: "orders"}, Table: "orders"}

	first, err := service.DropTable(context.Background(), input)
	if err != nil {
		t.Fatalf("first drop table preview: %v", err)
	}
	if first.Preview == nil || first.Preview.Source == nil || first.Preview.Source.DisplayName != "Orders" {
		t.Fatalf("first preview = %#v, want Orders source reference", first.Preview)
	}

	source.profile.DisplayName = "Recharge Orders"
	second, err := service.DropTable(context.Background(), input)
	if err != nil {
		t.Fatalf("second drop table preview: %v", err)
	}
	if second.Preview == nil || second.Preview.Source == nil || second.Preview.Source.DisplayName != "Recharge Orders" {
		t.Fatalf("second preview = %#v, want updated source display name", second.Preview)
	}
	if first.Preview.PreviewHash != second.Preview.PreviewHash {
		t.Fatalf("display name changed preview hash: %q != %q", first.Preview.PreviewHash, second.Preview.PreviewHash)
	}
	if first.Preview.Source == second.Preview.Source {
		t.Fatal("preview responses must not share source reference pointers")
	}
	if first.Preview.Source.DisplayName != "Orders" {
		t.Fatalf("first preview source changed through shared state: %#v", first.Preview.Source)
	}
}

func TestPreviewMismatchCarriesReplacementSourceReference(t *testing.T) {
	service := newServiceWithSource(t, config.QuickMode, serviceTestSource{
		id: "orders", engine: "mysql", dialect: database.MySQLDialect{},
		caps: database.MySQLDialect{}.Capabilities(),
		profile: database.SourceProfile{
			DisplayName: "Orders", Description: "Customer payment orders",
		},
	})
	input := DropTableInput{RequestMeta: RequestMeta{SourceID: "orders"}, Table: "orders"}

	first, err := service.DropTable(context.Background(), input)
	if err != nil {
		t.Fatalf("initial drop table preview: %v", err)
	}
	input.Confirm = true
	input.PreviewHash = "stale-preview-hash"
	replacement, err := service.DropTable(context.Background(), input)
	if !errors.Is(err, ErrPreviewMismatch) {
		t.Fatalf("mismatched confirmation error = %v, want %v", err, ErrPreviewMismatch)
	}
	if replacement.State != StatePreviewMismatch || replacement.Preview == nil {
		t.Fatalf("replacement response = %#v, want preview mismatch", replacement)
	}
	if replacement.Preview.PreviewHash != first.Preview.PreviewHash {
		t.Fatalf("replacement preview hash = %q, want %q", replacement.Preview.PreviewHash, first.Preview.PreviewHash)
	}
	if replacement.Preview.Source == nil || replacement.Preview.Source.ID != "orders" || replacement.Preview.Source.DisplayName != "Orders" {
		t.Fatalf("replacement preview source = %#v, want orders source reference", replacement.Preview.Source)
	}
}

func TestStrictModePreviewsReadOnlyQueryAndMatchingConfirmationQueries(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	source := serviceTestSource{
		id: "orders", engine: "mysql", db: db, dialect: database.MySQLDialect{},
		caps: database.MySQLDialect{}.Capabilities(),
	}
	service := newServiceWithSource(t, config.StrictMode, source)
	input := QueryInput{RequestMeta: RequestMeta{SourceID: "orders"}, SQL: "SELECT id FROM orders"}

	first, err := service.Query(context.Background(), input)
	if err != nil {
		t.Fatalf("preview query: %v", err)
	}
	if first.State != StateConfirmationRequired || first.Preview == nil {
		t.Fatalf("first response = %#v, want confirmation preview", first)
	}

	mock.ExpectQuery("(?i)select id from orders").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	input.Confirm = true
	input.PreviewHash = first.Preview.PreviewHash
	second, err := service.Query(context.Background(), input)
	if err != nil {
		t.Fatalf("confirmed query: %v", err)
	}
	if second.State != StateExecuted || second.Query == nil || len(second.Query.Rows) != 1 {
		t.Fatalf("second response = %#v, want query result", second)
	}
}

func TestListSourcesExposesBusinessProfilesWithoutConnectionData(t *testing.T) {
	const fakeSecret = "mysql://source-user:fake-secret-password@db.internal/orders"
	registry, err := database.NewRegistry([]database.Source{
		serviceTestSource{
			id: "orders", engine: "mysql", dialect: database.MySQLDialect{},
			profile: database.SourceProfile{DisplayName: "Orders", Description: "Customer payment orders", Aliases: []string{"payments"}, Keywords: []string{"orders", "refunds"}},
			secret:  fakeSecret,
		},
		serviceTestSource{
			id: "events", engine: "clickhouse", dialect: database.ClickHouseDialect{},
			profile: database.SourceProfile{DisplayName: "Events", Description: "Product event analytics", Aliases: []string{"analytics"}, Keywords: []string{"events", "metrics"}},
		},
	})
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	service := NewService(registry, config.QuickMode, nil)

	sources, err := service.ListSources(context.Background(), RequestMeta{})
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	encoded, err := json.Marshal(sources)
	if err != nil {
		t.Fatalf("json.Marshal(ListSources()) error = %v", err)
	}
	want := `[{"id":"events","engine":"clickhouse","display_name":"Events","description":"Product event analytics","aliases":["analytics"],"keywords":["events","metrics"]},{"id":"orders","engine":"mysql","display_name":"Orders","description":"Customer payment orders","aliases":["payments"],"keywords":["orders","refunds"]}]`
	if string(encoded) != want {
		t.Fatalf("json.Marshal(ListSources()) = %s, want %s", encoded, want)
	}
	for _, forbidden := range []string{"dsn", "host", "username", "password", "pool", "capabilities", fakeSecret} {
		if strings.Contains(strings.ToLower(string(encoded)), strings.ToLower(forbidden)) {
			t.Errorf("list_sources JSON contains forbidden connection detail %q: %s", forbidden, encoded)
		}
	}
}

func TestListSourcesPreservesSlicePresenceAndReturnsOwnedSlices(t *testing.T) {
	shared := &sharedProfileSource{serviceTestSource: serviceTestSource{
		id: "shared", engine: "mysql", dialect: database.MySQLDialect{},
		profile: database.SourceProfile{
			DisplayName: "Shared", Description: "Shared source",
			Aliases: []string{"shared-alias"}, Keywords: []string{"shared-keyword"},
		},
	}}
	registry, err := database.NewRegistry([]database.Source{
		serviceTestSource{
			id: "nil", engine: "mysql", dialect: database.MySQLDialect{},
			profile: database.SourceProfile{DisplayName: "Nil", Description: "Nil slices"},
		},
		serviceTestSource{
			id: "empty", engine: "mysql", dialect: database.MySQLDialect{},
			profile: database.SourceProfile{DisplayName: "Empty", Description: "Empty slices", Aliases: []string{}, Keywords: []string{}},
		},
		shared,
	})
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	service := NewService(registry, config.QuickMode, nil)

	first, err := service.ListSources(context.Background(), RequestMeta{})
	if err != nil {
		t.Fatalf("first list sources: %v", err)
	}
	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("json.Marshal(first ListSources()) error = %v", err)
	}
	want := `[{"id":"empty","engine":"mysql","display_name":"Empty","description":"Empty slices","aliases":[],"keywords":[]},{"id":"nil","engine":"mysql","display_name":"Nil","description":"Nil slices","aliases":null,"keywords":null},{"id":"shared","engine":"mysql","display_name":"Shared","description":"Shared source","aliases":["shared-alias"],"keywords":["shared-keyword"]}]`
	if string(encoded) != want {
		t.Fatalf("json.Marshal(first ListSources()) = %s, want %s", encoded, want)
	}

	first[2].Aliases[0] = "caller-alias"
	first[2].Keywords[0] = "caller-keyword"
	if shared.profile.Aliases[0] != "shared-alias" || shared.profile.Keywords[0] != "shared-keyword" {
		t.Fatalf("source profile was mutated through list response: %#v", shared.profile)
	}
	second, err := service.ListSources(context.Background(), RequestMeta{})
	if err != nil {
		t.Fatalf("second list sources: %v", err)
	}
	if second[2].Aliases[0] != "shared-alias" || second[2].Keywords[0] != "shared-keyword" {
		t.Fatalf("second list response shares slices with first: %#v", second[2])
	}
}

type sharedProfileSource struct {
	serviceTestSource
}

func (s *sharedProfileSource) Profile() database.SourceProfile { return s.profile }

func TestQuickModePreviewsMultiStatementExecuteSQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	service := newServiceWithSource(t, config.QuickMode, serviceTestSource{
		id: "orders", engine: "mysql", db: db, dialect: database.MySQLDialect{},
		caps: database.MySQLDialect{}.Capabilities(),
	})

	response, err := service.ExecuteSQL(context.Background(), ExecuteSQLInput{
		RequestMeta: RequestMeta{SourceID: "orders"},
		SQL:         "INSERT INTO orders (id) VALUES (1); INSERT INTO orders (id) VALUES (2)",
	})
	if err != nil {
		t.Fatalf("execute SQL: %v", err)
	}
	if response.State != StateConfirmationRequired || response.Preview == nil {
		t.Fatalf("response = %#v, want confirmation preview", response)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigratePreviewsEvenForLowRiskInsert(t *testing.T) {
	service := newServiceWithSource(t, config.QuickMode, serviceTestSource{
		id: "orders", engine: "mysql", dialect: database.MySQLDialect{},
		caps: database.MySQLDialect{}.Capabilities(),
	})

	response, err := service.Migrate(context.Background(), MigrateInput{
		RequestMeta: RequestMeta{SourceID: "orders"},
		SQL:         "INSERT INTO orders (id) VALUES (1)",
	})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if response.State != StateConfirmationRequired || response.Preview == nil || response.Preview.Risk != "high" {
		t.Fatalf("response = %#v, want high-risk confirmation preview", response)
	}
}

func TestPreviewsExposeActualAtomicityForDDLAndMixedBatches(t *testing.T) {
	service := newServiceWithSource(t, config.QuickMode, serviceTestSource{
		id: "orders", engine: "mysql", dialect: database.MySQLDialect{},
		caps: database.MySQLDialect{}.Capabilities(),
	})

	tests := []struct {
		name string
		call func() (Response, error)
	}{
		{
			name: "ddl",
			call: func() (Response, error) {
				return service.DropTable(context.Background(), DropTableInput{RequestMeta: RequestMeta{SourceID: "orders"}, Table: "orders"})
			},
		},
		{
			name: "mixed copy",
			call: func() (Response, error) {
				return service.CopyTable(context.Background(), CopyTableInput{RequestMeta: RequestMeta{SourceID: "orders"}, Source: "orders", Destination: "orders_copy", WithData: true})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := test.call()
			if err != nil {
				t.Fatalf("preview: %v", err)
			}
			if response.Preview == nil || response.Preview.Atomic {
				t.Fatalf("response = %#v, want non-atomic preview", response)
			}
		})
	}
}

func TestApplyToolErrorPreservesPreviewMismatchResponse(t *testing.T) {
	preview := &execution.Preview{PreviewHash: "replacement"}
	response := applyToolError(Response{RequestID: "request-1", State: StatePreviewMismatch, Preview: preview}, "fallback", newToolError(CodePreviewMismatch, ErrPreviewMismatch))
	if response.State != StatePreviewMismatch || response.Preview != preview || response.Error == nil || *response.Error != CodePreviewMismatch {
		t.Fatalf("response = %#v, want preserved preview mismatch", response)
	}
}

func TestMCPResponseMarksToolFailures(t *testing.T) {
	code := CodeUnknownSource
	result := mcpResponse(Response{State: StateError, Error: &code})
	if !result.IsError {
		t.Fatal("MCP result must mark a tool error")
	}
	if mcpResponse(Response{State: StateExecuted}).IsError {
		t.Fatal("successful MCP result must not be an error")
	}
}
