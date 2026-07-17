package database

import (
	"context"
	"database/sql"
	"fmt"
)

// MySQLDialect implements metadata access for MySQL.
type MySQLDialect struct{}

func (MySQLDialect) Name() string { return "mysql" }
func (MySQLDialect) Capabilities() Capability {
	return Capability{Transactions: true, AtomicBatches: true, CopyTable: true, AlterColumns: true}
}
func (MySQLDialect) ValidateIdentifier(name string) error        { return validateIdentifier(name) }
func (MySQLDialect) QuoteIdentifier(name string) (string, error) { return quoteIdentifier(name) }

// ListTables returns tables in the selected MySQL database.
func (MySQLDialect) ListTables(ctx context.Context, db *sql.DB) ([]Table, error) {
	rows, err := db.QueryContext(ctx, `
SELECT table_name, table_type, table_comment
FROM information_schema.tables
WHERE table_schema = DATABASE()
ORDER BY table_name`)
	if err != nil {
		return nil, fmt.Errorf("list MySQL tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var table Table
		if err := rows.Scan(&table.Name, &table.Type, &table.Comment); err != nil {
			return nil, fmt.Errorf("scan MySQL table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate MySQL tables: %w", err)
	}
	return tables, nil
}

// DescribeTable returns normalized column and index metadata for tableName.
func (MySQLDialect) DescribeTable(ctx context.Context, db *sql.DB, tableName string) (TableDescription, error) {
	if err := validateIdentifier(tableName); err != nil {
		return TableDescription{}, err
	}
	columns, err := mysqlColumns(ctx, db, tableName)
	if err != nil {
		return TableDescription{}, err
	}
	indexes, err := mysqlIndexes(ctx, db, tableName)
	if err != nil {
		return TableDescription{}, err
	}
	return TableDescription{Table: Table{Name: tableName}, Columns: columns, Indexes: indexes}, nil
}

func mysqlColumns(ctx context.Context, db *sql.DB, tableName string) ([]Column, error) {
	rows, err := db.QueryContext(ctx, `
SELECT column_name, column_type, is_nullable, column_default, column_comment, ordinal_position
FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY ordinal_position`, tableName)
	if err != nil {
		return nil, fmt.Errorf("describe MySQL columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var column Column
		var nullable string
		var defaultValue sql.NullString
		if err := rows.Scan(&column.Name, &column.Type, &nullable, &defaultValue, &column.Comment, &column.Position); err != nil {
			return nil, fmt.Errorf("scan MySQL column: %w", err)
		}
		column.Nullable = nullable == "YES"
		if defaultValue.Valid {
			column.DefaultValue = &defaultValue.String
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate MySQL columns: %w", err)
	}
	return columns, nil
}

func mysqlIndexes(ctx context.Context, db *sql.DB, tableName string) ([]Index, error) {
	rows, err := db.QueryContext(ctx, `
SELECT index_name, non_unique, index_type, column_name, expression
FROM information_schema.statistics
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY index_name, seq_in_index`, tableName)
	if err != nil {
		return nil, fmt.Errorf("describe MySQL indexes: %w", err)
	}
	defer rows.Close()

	byName := make(map[string]int)
	var indexes []Index
	for rows.Next() {
		var name, indexType string
		var column, expression sql.NullString
		var nonUnique int
		if err := rows.Scan(&name, &nonUnique, &indexType, &column, &expression); err != nil {
			return nil, fmt.Errorf("scan MySQL index: %w", err)
		}
		position, found := byName[name]
		if !found {
			position = len(indexes)
			byName[name] = position
			indexes = append(indexes, Index{Name: name, Type: indexType, Unique: nonUnique == 0})
		}
		if column.Valid && column.String != "" {
			indexes[position].Columns = append(indexes[position].Columns, column.String)
		} else if expression.Valid && expression.String != "" {
			indexes[position].Expression = expression.String
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate MySQL indexes: %w", err)
	}
	return indexes, nil
}
