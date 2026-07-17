package operation

import (
	"fmt"

	"github.com/dongrv/mcp-server-for-mysql/internal/database"
)

// ClickHouseBuilder builds the supported, typed ClickHouse schema operations.
type ClickHouseBuilder struct{}

func (ClickHouseBuilder) dialect() database.Dialect { return database.ClickHouseDialect{} }

func (b ClickHouseBuilder) CreateTable(request CreateTableRequest) ([]string, error) {
	table, err := quote(b.dialect(), request.Table)
	if err != nil {
		return nil, err
	}
	columns, err := buildColumns(b.dialect(), request.Columns, clickHouseColumn)
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("CREATE TABLE %s (%s) ENGINE = MergeTree ORDER BY tuple()", table, join(columns))}, nil
}

func (b ClickHouseBuilder) DropTable(request DropTableRequest) ([]string, error) {
	table, err := quote(b.dialect(), request.Table)
	if err != nil {
		return nil, err
	}
	return []string{"DROP TABLE " + table}, nil
}

func (b ClickHouseBuilder) AddColumns(request AddColumnsRequest) ([]string, error) {
	table, err := quote(b.dialect(), request.Table)
	if err != nil {
		return nil, err
	}
	columns, err := buildColumns(b.dialect(), request.Columns, clickHouseColumn)
	if err != nil {
		return nil, err
	}
	for i := range columns {
		columns[i] = "ADD COLUMN " + columns[i]
	}
	return []string{fmt.Sprintf("ALTER TABLE %s %s", table, join(columns))}, nil
}

func (b ClickHouseBuilder) DropColumns(request DropColumnsRequest) ([]string, error) {
	table, err := quote(b.dialect(), request.Table)
	if err != nil {
		return nil, err
	}
	columns, err := quotedColumns(b.dialect(), request.Columns)
	if err != nil {
		return nil, err
	}
	for i := range columns {
		columns[i] = "DROP COLUMN " + columns[i]
	}
	return []string{fmt.Sprintf("ALTER TABLE %s %s", table, join(columns))}, nil
}

func (b ClickHouseBuilder) ModifyColumns(request ModifyColumnsRequest) ([]string, error) {
	table, err := quote(b.dialect(), request.Table)
	if err != nil {
		return nil, err
	}
	columns, err := buildColumns(b.dialect(), request.Columns, clickHouseColumn)
	if err != nil {
		return nil, err
	}
	for i := range columns {
		columns[i] = "MODIFY COLUMN " + columns[i]
	}
	return []string{fmt.Sprintf("ALTER TABLE %s %s", table, join(columns))}, nil
}

func (b ClickHouseBuilder) CreateIndex(CreateIndexRequest) ([]string, error) {
	return nil, database.ErrUnsupportedCapability
}

func (b ClickHouseBuilder) DropIndex(DropIndexRequest) ([]string, error) {
	return nil, database.ErrUnsupportedCapability
}

func (b ClickHouseBuilder) RenameTable(request RenameTableRequest) ([]string, error) {
	from, err := quote(b.dialect(), request.From)
	if err != nil {
		return nil, err
	}
	to, err := quote(b.dialect(), request.To)
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("RENAME TABLE %s TO %s", from, to)}, nil
}

func (ClickHouseBuilder) CopyTable(CopyTableRequest) ([]string, error) {
	return nil, database.ErrUnsupportedCapability
}
