package operation

import (
	"fmt"

	"github.com/dongrv/mcp-server-for-mysql/internal/database"
)

// MySQLBuilder builds safe schema SQL for MySQL.
type MySQLBuilder struct{}

func (MySQLBuilder) dialect() database.Dialect { return database.MySQLDialect{} }

func (b MySQLBuilder) CreateTable(request CreateTableRequest) ([]string, error) {
	table, err := quote(b.dialect(), request.Table)
	if err != nil {
		return nil, err
	}
	columns, err := buildColumns(b.dialect(), request.Columns, mysqlColumn)
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("CREATE TABLE %s (%s)", table, join(columns))}, nil
}

func (b MySQLBuilder) DropTable(request DropTableRequest) ([]string, error) {
	table, err := quote(b.dialect(), request.Table)
	if err != nil {
		return nil, err
	}
	return []string{"DROP TABLE " + table}, nil
}

func (b MySQLBuilder) AddColumns(request AddColumnsRequest) ([]string, error) {
	table, err := quote(b.dialect(), request.Table)
	if err != nil {
		return nil, err
	}
	columns, err := buildColumns(b.dialect(), request.Columns, mysqlColumn)
	if err != nil {
		return nil, err
	}
	for i := range columns {
		columns[i] = "ADD COLUMN " + columns[i]
	}
	return []string{fmt.Sprintf("ALTER TABLE %s %s", table, join(columns))}, nil
}

func (b MySQLBuilder) DropColumns(request DropColumnsRequest) ([]string, error) {
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

func (b MySQLBuilder) ModifyColumns(request ModifyColumnsRequest) ([]string, error) {
	table, err := quote(b.dialect(), request.Table)
	if err != nil {
		return nil, err
	}
	columns, err := buildColumns(b.dialect(), request.Columns, mysqlColumn)
	if err != nil {
		return nil, err
	}
	for i := range columns {
		columns[i] = "MODIFY COLUMN " + columns[i]
	}
	return []string{fmt.Sprintf("ALTER TABLE %s %s", table, join(columns))}, nil
}

func (b MySQLBuilder) CreateIndex(request CreateIndexRequest) ([]string, error) {
	table, err := quote(b.dialect(), request.Table)
	if err != nil {
		return nil, err
	}
	index, err := quote(b.dialect(), request.Index)
	if err != nil {
		return nil, err
	}
	columns, err := quotedColumns(b.dialect(), request.Columns)
	if err != nil {
		return nil, err
	}
	prefix := "CREATE INDEX"
	if request.Unique {
		prefix = "CREATE UNIQUE INDEX"
	}
	return []string{fmt.Sprintf("%s %s ON %s (%s)", prefix, index, table, join(columns))}, nil
}

func (b MySQLBuilder) DropIndex(request DropIndexRequest) ([]string, error) {
	table, err := quote(b.dialect(), request.Table)
	if err != nil {
		return nil, err
	}
	index, err := quote(b.dialect(), request.Index)
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("DROP INDEX %s ON %s", index, table)}, nil
}

func (b MySQLBuilder) RenameTable(request RenameTableRequest) ([]string, error) {
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

func (b MySQLBuilder) CopyTable(request CopyTableRequest) ([]string, error) {
	source, err := quote(b.dialect(), request.Source)
	if err != nil {
		return nil, err
	}
	destination, err := quote(b.dialect(), request.Destination)
	if err != nil {
		return nil, err
	}
	statements := []string{fmt.Sprintf("CREATE TABLE %s LIKE %s", destination, source)}
	if request.WithData {
		statements = append(statements, fmt.Sprintf("INSERT INTO %s SELECT * FROM %s", destination, source))
	}
	return statements, nil
}
