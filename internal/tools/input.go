package tools

import "github.com/dongrv/mcp-server-for-mysql/internal/operation"

// RequestMeta accompanies every source-aware tool invocation.
type RequestMeta struct {
	SourceID    string `json:"source_id"`
	Confirm     bool   `json:"confirm,omitempty"`
	PreviewHash string `json:"preview_hash,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
}

type QueryInput struct {
	RequestMeta
	SQL        string `json:"sql"`
	Parameters []any  `json:"parameters,omitempty"`
}

type ExecuteSQLInput struct {
	RequestMeta
	SQL        string `json:"sql"`
	Parameters []any  `json:"parameters,omitempty"`
}

type TableInput struct {
	RequestMeta
	Table string `json:"table"`
}

type CreateTableInput struct {
	RequestMeta
	Table   string                 `json:"table"`
	Columns []operation.ColumnSpec `json:"columns"`
}

type DropTableInput struct {
	RequestMeta
	Table string `json:"table"`
}

type AddColumnsInput struct {
	RequestMeta
	Table   string                 `json:"table"`
	Columns []operation.ColumnSpec `json:"columns"`
}

type DropColumnsInput struct {
	RequestMeta
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
}

type ModifyColumnsInput struct {
	RequestMeta
	Table   string                 `json:"table"`
	Columns []operation.ColumnSpec `json:"columns"`
}

type CreateIndexInput struct {
	RequestMeta
	Table   string   `json:"table"`
	Index   string   `json:"index"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

type DropIndexInput struct {
	RequestMeta
	Table string `json:"table"`
	Index string `json:"index"`
}

type RenameTableInput struct {
	RequestMeta
	From string `json:"from"`
	To   string `json:"to"`
}

type CopyTableInput struct {
	RequestMeta
	Source      string `json:"source_table"`
	Destination string `json:"destination_table"`
	WithData    bool   `json:"with_data"`
}

type MigrateInput struct {
	RequestMeta
	SQL        string `json:"sql"`
	Parameters []any  `json:"parameters,omitempty"`
}
