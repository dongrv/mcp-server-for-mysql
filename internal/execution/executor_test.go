package execution

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dongrv/mcp-server-for-mysql/internal/database"
	"github.com/dongrv/mcp-server-for-mysql/internal/sqlguard"
)

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("expected %v, got %v", target, err)
	}
}

func requireErrorContains(t *testing.T, err error, text string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), text) {
		t.Fatalf("expected error containing %q, got %v", text, err)
	}
}

func requireEqual[T any](t *testing.T, want, got T) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %#v, got %#v", want, got)
	}
}

func requireTrue(t *testing.T, value bool) {
	t.Helper()
	if !value {
		t.Fatal("expected true")
	}
}

func requireFalse(t *testing.T, value bool) {
	t.Helper()
	if value {
		t.Fatal("expected false")
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}

type testSource struct {
	db   *sql.DB
	caps database.Capability
}

func (s testSource) ID() string     { return "analytics" }
func (s testSource) Engine() string { return "mysql" }
func (s testSource) Profile() database.SourceProfile {
	return database.SourceProfile{
		DisplayName: "Analytics",
		Description: "Test analytics source",
		Aliases:     []string{"warehouse"},
		Keywords:    []string{"analytics"},
	}
}
func (s testSource) DB() *sql.DB                       { return s.db }
func (s testSource) Dialect() database.Dialect         { return database.MySQLDialect{} }
func (s testSource) Capabilities() database.Capability { return s.caps }
func (s testSource) Close() error                      { return nil }

func mysqlTestSource(db *sql.DB) testSource {
	return testSource{db: db, caps: database.Capability{AtomicBatches: true}}
}

type metadataErrorResult struct{}

func (metadataErrorResult) LastInsertId() (int64, error) {
	return 0, errors.New("last insert ID is unsupported")
}

func (metadataErrorResult) RowsAffected() (int64, error) {
	return 3, nil
}

type resultExecutor struct {
	result sql.Result
}

func (e resultExecutor) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return e.result, nil
}

func TestQueryStopsAfterLimitAndMarksResultTruncated(t *testing.T) {
	db, mock, err := sqlmock.New()
	requireNoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT id, payload FROM orders").
		WillReturnRows(sqlmock.NewRows([]string{"id", "payload"}).
			AddRow(1, []byte("first")).
			AddRow(2, []byte("second")).
			AddRow(3, []byte("third"))).
		RowsWillBeClosed()

	result, err := NewExecutor(50*time.Millisecond, 2).Query(context.Background(), db, "SELECT id, payload FROM orders", nil)
	requireNoError(t, err)
	requireEqual(t, []string{"id", "payload"}, result.Columns)
	requireEqual(t, [][]any{
		{int64(1), "first"},
		{int64(2), "second"},
	}, result.Rows)
	requireTrue(t, result.Truncated)
	requireNoError(t, mock.ExpectationsWereMet())
}

func TestQueryPreservesDuplicateColumnLabelsInOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	requireNoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT 1 AS value, 2 AS value").
		WillReturnRows(sqlmock.NewRows([]string{"value", "value"}).AddRow(1, 2)).
		RowsWillBeClosed()

	result, err := NewExecutor(time.Second, 100).Query(context.Background(), db, "SELECT 1 AS value, 2 AS value", nil)
	requireNoError(t, err)
	requireEqual(t, []string{"value", "value"}, result.Columns)
	if !reflect.DeepEqual([][]any{{int64(1), int64(2)}}, result.Rows) {
		t.Fatalf("want ordered values for duplicate labels, got %#v", result.Rows)
	}
	requireNoError(t, mock.ExpectationsWereMet())
}

func TestQueryHonorsCanceledContext(t *testing.T) {
	db, _, err := sqlmock.New()
	requireNoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = NewExecutor(time.Second, 100).Query(ctx, db, "SELECT 1", nil)
	requireErrorIs(t, err, context.Canceled)
}

func TestExecuteMultipleStatementsReportsNonAtomicSource(t *testing.T) {
	db, mock, err := sqlmock.New()
	requireNoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectExec("INSERT INTO a").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO b").WillReturnResult(sqlmock.NewResult(0, 1))
	plan := sqlguard.Plan{Statements: []sqlguard.Statement{
		{NormalizedSQL: "INSERT INTO a VALUES (1)", Kind: sqlguard.Write},
		{NormalizedSQL: "INSERT INTO b VALUES (2)", Kind: sqlguard.Write},
	}, Risk: sqlguard.LowRisk}

	result, err := NewExecutor(time.Second, 100).ExecutePlan(context.Background(), testSource{db: db}, plan, nil)
	requireNoError(t, err)
	requireFalse(t, result.Atomic)
	requireEqual(t, []StatementResult{{Index: 0, RowsAffected: int64Pointer(1), LastInsertID: int64Pointer(0)}, {Index: 1, RowsAffected: int64Pointer(1), LastInsertID: int64Pointer(0)}}, result.Statements)
	requireNoError(t, mock.ExpectationsWereMet())
}

func TestAppendExecutionResultOmitsUnsupportedDriverMetadata(t *testing.T) {
	results, err := appendExecutionResult(
		context.Background(),
		resultExecutor{result: metadataErrorResult{}},
		nil,
		0,
		"INSERT INTO events VALUES (1)",
		nil,
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, int64Pointer(3), results[0].RowsAffected)
	requireEqual(t, (*int64)(nil), results[0].LastInsertID)

	encoded, err := json.Marshal(results[0])
	requireNoError(t, err)
	requireEqual(t, `{"index":0,"rows_affected":3}`, string(encoded))
}

func TestAtomicBatchCommitsWhenResultMetadataIsUnsupported(t *testing.T) {
	db, mock, err := sqlmock.New()
	requireNoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE events").WillReturnResult(sqlmock.NewErrorResult(errors.New("metadata unsupported")))
	mock.ExpectCommit()
	plan := sqlguard.Plan{Statements: []sqlguard.Statement{{
		NormalizedSQL: "UPDATE events SET status = 'sent'",
		Kind:          sqlguard.Write,
	}}}

	result, err := NewExecutor(time.Second, 100).ExecutePlan(context.Background(), mysqlTestSource(db), plan, nil)
	requireNoError(t, err)
	requireTrue(t, result.Atomic)
	requireEqual(t, []StatementResult{{Index: 0}}, result.Statements)
	requireNoError(t, mock.ExpectationsWereMet())
}

func TestAtomicBatchRollsBackOnSecondStatementFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	requireNoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE a").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE b").WillReturnError(errors.New("second statement failed"))
	mock.ExpectRollback()
	plan := sqlguard.Plan{Statements: []sqlguard.Statement{
		{NormalizedSQL: "UPDATE a SET x = 1", Kind: sqlguard.Write},
		{NormalizedSQL: "UPDATE b SET x = 2", Kind: sqlguard.Write},
	}}

	_, err = NewExecutor(time.Second, 100).ExecutePlan(context.Background(), mysqlTestSource(db), plan, nil)
	requireErrorContains(t, err, "second statement failed")
	requireNoError(t, mock.ExpectationsWereMet())
}

func TestNonAtomicBatchStopsAtFirstFailedStatement(t *testing.T) {
	db, mock, err := sqlmock.New()
	requireNoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectExec("UPDATE a").WillReturnError(errors.New("first statement failed"))
	plan := sqlguard.Plan{Statements: []sqlguard.Statement{
		{NormalizedSQL: "UPDATE a SET x = 1", Kind: sqlguard.Write},
		{NormalizedSQL: "UPDATE b SET x = 2", Kind: sqlguard.Write},
	}}

	_, err = NewExecutor(time.Second, 100).ExecutePlan(context.Background(), testSource{db: db}, plan, nil)
	requireErrorContains(t, err, "first statement failed")
	requireNoError(t, mock.ExpectationsWereMet())
}

func TestExecutePlanRejectsEmptyPlanBeforeAccessingDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	requireNoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = NewExecutor(time.Second, 100).ExecutePlan(context.Background(), testSource{db: db}, sqlguard.Plan{}, nil)
	requireErrorIs(t, err, ErrInvalidPlan)
	requireNoError(t, mock.ExpectationsWereMet())
}

func TestExecutePlanRejectsSharedArgsForMultipleStatements(t *testing.T) {
	db, mock, err := sqlmock.New()
	requireNoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	plan := sqlguard.Plan{Statements: []sqlguard.Statement{
		{NormalizedSQL: "INSERT INTO a VALUES (?)", Kind: sqlguard.Write},
		{NormalizedSQL: "INSERT INTO b VALUES (?)", Kind: sqlguard.Write},
	}}
	_, err = NewExecutor(time.Second, 100).ExecutePlan(context.Background(), testSource{db: db}, plan, []any{1})
	requireErrorIs(t, err, ErrAmbiguousBatchArgs)
	requireNoError(t, mock.ExpectationsWereMet())
}
