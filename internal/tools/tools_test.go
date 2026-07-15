package tools

import (
	"context"
	"database/sql"
	"errors"
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
	profile.Aliases = append([]string(nil), profile.Aliases...)
	profile.Keywords = append([]string(nil), profile.Keywords...)
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

func TestListSourcesExposesOnlyConfiguredIDsAndEngines(t *testing.T) {
	registry, err := database.NewRegistry([]database.Source{
		serviceTestSource{
			id: "orders", engine: "mysql", dialect: database.MySQLDialect{},
			profile: database.SourceProfile{DisplayName: "Orders", Description: "Customer payment orders", Aliases: []string{"payments"}, Keywords: []string{"orders", "refunds"}},
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
	if len(sources) != 2 || sources[0].ID != "events" || sources[1].ID != "orders" {
		t.Fatalf("sources = %#v", sources)
	}
}

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
