package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// ClickHouseDialect implements metadata access for ClickHouse.
type ClickHouseDialect struct{}

func (ClickHouseDialect) Name() string                                { return "clickhouse" }
func (ClickHouseDialect) Capabilities() Capability                    { return Capability{AlterColumns: true} }
func (ClickHouseDialect) ValidateIdentifier(name string) error        { return validateIdentifier(name) }
func (ClickHouseDialect) QuoteIdentifier(name string) (string, error) { return quoteIdentifier(name) }

// ListTables returns tables in the selected ClickHouse database.
func (ClickHouseDialect) ListTables(ctx context.Context, db *sql.DB) ([]Table, error) {
	rows, err := db.QueryContext(ctx, `
SELECT name, engine, comment
FROM system.tables
WHERE database = currentDatabase()
ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list ClickHouse tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var table Table
		if err := rows.Scan(&table.Name, &table.Type, &table.Comment); err != nil {
			return nil, fmt.Errorf("scan ClickHouse table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ClickHouse tables: %w", err)
	}
	return tables, nil
}

// DescribeTable returns normalized column and index metadata for tableName.
func (ClickHouseDialect) DescribeTable(ctx context.Context, db *sql.DB, tableName string) (TableDescription, error) {
	if err := validateIdentifier(tableName); err != nil {
		return TableDescription{}, err
	}
	columns, err := clickHouseColumns(ctx, db, tableName)
	if err != nil {
		return TableDescription{}, err
	}
	indexes, err := clickHouseIndexes(ctx, db, tableName)
	if err != nil {
		return TableDescription{}, err
	}
	return TableDescription{Table: Table{Name: tableName}, Columns: columns, Indexes: indexes}, nil
}

func clickHouseColumns(ctx context.Context, db *sql.DB, tableName string) ([]Column, error) {
	rows, err := db.QueryContext(ctx, `
SELECT name, type, default_expression, comment, position
FROM system.columns
WHERE database = currentDatabase() AND table = ?
ORDER BY position`, tableName)
	if err != nil {
		return nil, fmt.Errorf("describe ClickHouse columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var column Column
		var defaultValue string
		if err := rows.Scan(&column.Name, &column.Type, &defaultValue, &column.Comment, &column.Position); err != nil {
			return nil, fmt.Errorf("scan ClickHouse column: %w", err)
		}
		column.Nullable = isClickHouseNullable(column.Type)
		if defaultValue != "" {
			column.DefaultValue = &defaultValue
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ClickHouse columns: %w", err)
	}
	return columns, nil
}

func isClickHouseNullable(typeName string) bool {
	wrapper, inner, ok := clickHouseOuterWrapper(typeName)
	for ok && wrapper == "lowcardinality" {
		wrapper, inner, ok = clickHouseOuterWrapper(inner)
	}
	return ok && wrapper == "nullable"
}

func clickHouseOuterWrapper(typeName string) (wrapper, inner string, ok bool) {
	normalized := strings.ToLower(strings.Join(strings.Fields(typeName), ""))
	open := strings.IndexByte(normalized, '(')
	if open <= 0 || !strings.HasSuffix(normalized, ")") {
		return "", "", false
	}

	depth := 0
	for i := open; i < len(normalized); i++ {
		switch normalized[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 || (depth == 0 && i != len(normalized)-1) {
				return "", "", false
			}
		}
	}
	if depth != 0 {
		return "", "", false
	}
	return normalized[:open], normalized[open+1 : len(normalized)-1], true
}

func clickHouseIndexes(ctx context.Context, db *sql.DB, tableName string) ([]Index, error) {
	rows, err := db.QueryContext(ctx, `
SELECT name, expr, type
FROM system.data_skipping_indices
WHERE database = currentDatabase() AND table = ?
ORDER BY name`, tableName)
	if err != nil {
		return nil, fmt.Errorf("describe ClickHouse indexes: %w", err)
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var index Index
		if err := rows.Scan(&index.Name, &index.Expression, &index.Type); err != nil {
			return nil, fmt.Errorf("scan ClickHouse index: %w", err)
		}
		indexes = append(indexes, index)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ClickHouse indexes: %w", err)
	}
	return indexes, nil
}
