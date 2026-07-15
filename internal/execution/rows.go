package execution

import (
	"database/sql"
	"fmt"
)

// QueryResult is a compact tabular response suitable for an MCP tool result.
type QueryResult struct {
	Columns   []string `json:"columns"`
	Rows      [][]any  `json:"rows"`
	Truncated bool     `json:"truncated"`
}

// ExecuteResult reports each successfully executed statement. Atomic is true
// only when all statements ran inside one source-supported transaction.
type ExecuteResult struct {
	Statements []StatementResult `json:"statements"`
	Atomic     bool              `json:"atomic"`
}

// StatementResult captures the portable mutation metadata returned by a SQL
// driver for one statement.
type StatementResult struct {
	Index        int    `json:"index"`
	RowsAffected *int64 `json:"rows_affected,omitempty"`
	LastInsertID *int64 `json:"last_insert_id,omitempty"`
}

func (e Executor) collectRows(rows *sql.Rows) (QueryResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		return QueryResult{}, fmt.Errorf("read query columns: %w", err)
	}
	result := QueryResult{Columns: columns, Rows: make([][]any, 0, e.effectiveMaxRows())}

	for rows.Next() {
		if len(result.Rows) == e.effectiveMaxRows() {
			result.Truncated = true
			break
		}

		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return QueryResult{}, fmt.Errorf("scan query row: %w", err)
		}

		for i := range values {
			if bytes, ok := values[i].([]byte); ok {
				values[i] = string(bytes)
			}
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return QueryResult{}, fmt.Errorf("iterate query rows: %w", err)
	}
	return result, nil
}

func (e Executor) effectiveMaxRows() int {
	if e.maxRows <= 0 {
		return DefaultMaxRows
	}
	return e.maxRows
}
