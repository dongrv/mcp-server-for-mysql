// Package database provides source-aware database connection and metadata APIs.
package database

import (
	"context"
	"database/sql"
	"errors"
)

// ErrUnsupportedCapability is returned when a source cannot safely express a
// requested structured operation.
var ErrUnsupportedCapability = errors.New("unsupported database capability")

// Capability declares the high-level operations supported by a source.
// Capabilities not explicitly enabled are unsupported.
type Capability struct {
	Transactions  bool
	AtomicBatches bool
	CopyTable     bool
	AlterColumns  bool
}

// SourceProfile contains business metadata that helps identify a source.
type SourceProfile struct {
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
	Keywords    []string `json:"keywords"`
}

// Table is normalized metadata for a database table.
type Table struct {
	Name    string
	Type    string
	Comment string
}

// Column is normalized metadata for a table column.
type Column struct {
	Name         string
	Type         string
	Nullable     bool
	DefaultValue *string
	Comment      string
	Position     int
}

// Index is normalized metadata for a table index.
type Index struct {
	Name       string
	Type       string
	Columns    []string
	Expression string
	Unique     bool
}

// TableDescription contains normalized table, column, and index metadata.
type TableDescription struct {
	Table   Table
	Columns []Column
	Indexes []Index
}

// Source is one configured database source.
type Source interface {
	ID() string
	Engine() string
	// Profile returns business metadata with caller-owned Aliases and Keywords.
	// Nil slices remain nil, and non-nil empty slices remain non-nil.
	Profile() SourceProfile
	DB() *sql.DB
	Dialect() Dialect
	Capabilities() Capability
	Close() error
}

// Dialect provides engine-specific metadata behavior.
type Dialect interface {
	Name() string
	Capabilities() Capability
	ValidateIdentifier(string) error
	QuoteIdentifier(string) (string, error)
	ListTables(context.Context, *sql.DB) ([]Table, error)
	DescribeTable(context.Context, *sql.DB, string) (TableDescription, error)
}
