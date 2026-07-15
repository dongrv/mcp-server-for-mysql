package execution

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dongrv/mcp-server-for-mysql/internal/database"
	"github.com/dongrv/mcp-server-for-mysql/internal/sqlguard"
)

const (
	DefaultQueryTimeout = 30 * time.Second
	DefaultMaxRows      = 100
)

var (
	ErrInvalidPlan        = errors.New("invalid SQL execution plan")
	ErrAmbiguousBatchArgs = errors.New("parameters are only supported for a single SQL statement")
	ErrNilDatabase        = errors.New("database connection is required")
	ErrNilSourceDatabase  = errors.New("source database connection is required")
)

// Executor bounds database work so MCP calls cannot consume unbounded time or
// return unbounded result sets.
type Executor struct {
	timeout time.Duration
	maxRows int
}

// NewDefaultExecutor returns the standard safe execution limits.
func NewDefaultExecutor() Executor {
	return NewExecutor(DefaultQueryTimeout, DefaultMaxRows)
}

// NewExecutor applies defaults for invalid limits. This keeps call sites safe
// when optional configuration is omitted or malformed.
func NewExecutor(timeout time.Duration, maxRows int) Executor {
	if timeout <= 0 {
		timeout = DefaultQueryTimeout
	}
	if maxRows <= 0 {
		maxRows = DefaultMaxRows
	}
	return Executor{timeout: timeout, maxRows: maxRows}
}

// Query runs one read query under the configured timeout and row limit.
func (e Executor) Query(ctx context.Context, db *sql.DB, statement string, args []any) (QueryResult, error) {
	if db == nil {
		return QueryResult{}, ErrNilDatabase
	}
	if ctx == nil {
		return QueryResult{}, fmt.Errorf("query context is required")
	}

	ctx, cancel := context.WithTimeout(ctx, e.effectiveTimeout())
	defer cancel()

	rows, err := db.QueryContext(ctx, statement, args...)
	if err != nil {
		return QueryResult{}, err
	}
	defer rows.Close()

	return e.collectRows(rows)
}

// ExecutePlan executes an analyzed non-read execution plan. Parameters bind to
// a single statement only; the API intentionally rejects sharing them across a
// batch because it is ambiguous and can produce surprising writes.
func (e Executor) ExecutePlan(ctx context.Context, source database.Source, plan sqlguard.Plan, args []any) (ExecuteResult, error) {
	if err := validateExecutionPlan(plan); err != nil {
		return ExecuteResult{}, err
	}
	if len(plan.Statements) > 1 && len(args) > 0 {
		return ExecuteResult{}, ErrAmbiguousBatchArgs
	}
	if source == nil || source.DB() == nil {
		return ExecuteResult{}, ErrNilSourceDatabase
	}
	if ctx == nil {
		return ExecuteResult{}, fmt.Errorf("execution context is required")
	}

	ctx, cancel := context.WithTimeout(ctx, e.effectiveTimeout())
	defer cancel()

	atomic := IsAtomicBatch(source, plan)
	if atomic {
		return executeAtomically(ctx, source.DB(), plan.Statements, args)
	}
	return executeSequentially(ctx, source.DB(), plan.Statements, args)
}

// IsAtomicBatch reports whether ExecutePlan will run the complete plan in a
// single transaction. It is shared with preview construction so callers never
// confirm an atomicity guarantee that execution cannot provide.
func IsAtomicBatch(source database.Source, plan sqlguard.Plan) bool {
	return source != nil && source.Capabilities().AtomicBatches && allWrites(plan.Statements)
}

func (e Executor) effectiveTimeout() time.Duration {
	if e.timeout <= 0 {
		return DefaultQueryTimeout
	}
	return e.timeout
}

func validateExecutionPlan(plan sqlguard.Plan) error {
	if len(plan.Statements) == 0 {
		return ErrInvalidPlan
	}
	for index, statement := range plan.Statements {
		if strings.TrimSpace(statement.NormalizedSQL) == "" {
			return fmt.Errorf("%w: statement %d is empty", ErrInvalidPlan, index+1)
		}
		switch statement.Kind {
		case sqlguard.Write, sqlguard.DDL:
		default:
			return fmt.Errorf("%w: statement %d is not executable", ErrInvalidPlan, index+1)
		}
	}
	return nil
}

func allWrites(statements []sqlguard.Statement) bool {
	for _, statement := range statements {
		if statement.Kind != sqlguard.Write {
			return false
		}
	}
	return true
}

func executeAtomically(ctx context.Context, db *sql.DB, statements []sqlguard.Statement, args []any) (result ExecuteResult, err error) {
	result.Atomic = true
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}

	for index, statement := range statements {
		result.Statements, err = appendExecutionResult(ctx, tx, result.Statements, index, statement.NormalizedSQL, executionArgs(index, args))
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return result, fmt.Errorf("execute statement %d: %w; rollback: %v", index+1, err, rollbackErr)
			}
			return result, fmt.Errorf("execute statement %d: %w", index+1, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit execution plan: %w", err)
	}
	return result, nil
}

func executeSequentially(ctx context.Context, db *sql.DB, statements []sqlguard.Statement, args []any) (result ExecuteResult, err error) {
	for index, statement := range statements {
		result.Statements, err = appendExecutionResult(ctx, db, result.Statements, index, statement.NormalizedSQL, executionArgs(index, args))
		if err != nil {
			return result, fmt.Errorf("execute statement %d: %w", index+1, err)
		}
	}
	return result, nil
}

type statementExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func appendExecutionResult(ctx context.Context, executor statementExecutor, results []StatementResult, index int, statement string, args []any) ([]StatementResult, error) {
	executionResult, err := executor.ExecContext(ctx, statement, args...)
	if err != nil {
		return results, err
	}
	result := StatementResult{Index: index}
	if rowsAffected, err := executionResult.RowsAffected(); err == nil {
		result.RowsAffected = &rowsAffected
	}
	if lastInsertID, err := executionResult.LastInsertId(); err == nil {
		result.LastInsertID = &lastInsertID
	}
	return append(results, result), nil
}

func executionArgs(index int, args []any) []any {
	if index == 0 {
		return args
	}
	return nil
}
