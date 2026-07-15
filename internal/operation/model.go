// Package operation turns typed schema requests into dialect-safe SQL.
package operation

// ColumnSpec deliberately exposes only a small, typed column surface. It does
// not accept defaults, placement clauses, expressions, or raw SQL fragments.
type ColumnSpec struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Length    int    `json:"length,omitempty"`
	Precision int    `json:"precision,omitempty"`
	Scale     int    `json:"scale,omitempty"`
	Nullable  bool   `json:"nullable"`
}

type CreateTableRequest struct {
	Table   string       `json:"table"`
	Columns []ColumnSpec `json:"columns"`
}

type DropTableRequest struct {
	Table string `json:"table"`
}

type AddColumnsRequest struct {
	Table   string       `json:"table"`
	Columns []ColumnSpec `json:"columns"`
}

type DropColumnsRequest struct {
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
}

type ModifyColumnsRequest struct {
	Table   string       `json:"table"`
	Columns []ColumnSpec `json:"columns"`
}

type CreateIndexRequest struct {
	Table   string   `json:"table"`
	Index   string   `json:"index"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

type DropIndexRequest struct {
	Table string `json:"table"`
	Index string `json:"index"`
}

type RenameTableRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type CopyTableRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	WithData    bool   `json:"with_data"`
}

// Builder is implemented independently for each supported database engine.
type Builder interface {
	CreateTable(CreateTableRequest) ([]string, error)
	DropTable(DropTableRequest) ([]string, error)
	AddColumns(AddColumnsRequest) ([]string, error)
	DropColumns(DropColumnsRequest) ([]string, error)
	ModifyColumns(ModifyColumnsRequest) ([]string, error)
	CreateIndex(CreateIndexRequest) ([]string, error)
	DropIndex(DropIndexRequest) ([]string, error)
	RenameTable(RenameTableRequest) ([]string, error)
	CopyTable(CopyTableRequest) ([]string, error)
}
