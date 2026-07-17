package database

import (
	"context"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMySQLDescribeTableNormalizesColumnsAndFunctionalIndexes(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled SQL expectations: %v", err)
		}
	})

	mock.ExpectQuery(`
SELECT column_name, column_type, is_nullable, column_default, column_comment, ordinal_position
FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY ordinal_position`).
		WithArgs("orders").
		WillReturnRows(sqlmock.NewRows([]string{
			"column_name", "column_type", "is_nullable", "column_default", "column_comment", "ordinal_position",
		}).
			AddRow("id", "bigint unsigned", "NO", nil, "primary key", 1).
			AddRow("email", "varchar(255)", "YES", "'unknown'", "", 2))
	mock.ExpectQuery(`
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = 'information_schema'
  AND table_name = 'STATISTICS'
  AND column_name = 'EXPRESSION'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`
SELECT index_name, non_unique, index_type, column_name, expression
FROM information_schema.statistics
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY index_name, seq_in_index`).
		WithArgs("orders").
		WillReturnRows(sqlmock.NewRows([]string{
			"index_name", "non_unique", "index_type", "column_name", "expression",
		}).
			AddRow("PRIMARY", 0, "BTREE", "id", nil).
			AddRow("idx_email", 1, "BTREE", "email", nil).
			AddRow("idx_lower_email", 1, "BTREE", nil, "lower(`email`)"))
	mock.ExpectClose()

	got, err := (MySQLDialect{}).DescribeTable(context.Background(), db, "orders")
	if err != nil {
		t.Fatalf("DescribeTable() error = %v", err)
	}
	defaultValue := "'unknown'"
	want := TableDescription{
		Table: Table{Name: "orders"},
		Columns: []Column{
			{Name: "id", Type: "bigint unsigned", Comment: "primary key", Position: 1},
			{Name: "email", Type: "varchar(255)", Nullable: true, DefaultValue: &defaultValue, Position: 2},
		},
		Indexes: []Index{
			{Name: "PRIMARY", Type: "BTREE", Columns: []string{"id"}, Unique: true},
			{Name: "idx_email", Type: "BTREE", Columns: []string{"email"}},
			{Name: "idx_lower_email", Type: "BTREE", Expression: "lower(`email`)"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DescribeTable() = %#v, want %#v", got, want)
	}
}

func TestMySQLDescribeTableSupportsLegacyIndexMetadata(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled SQL expectations: %v", err)
		}
	})

	mock.ExpectQuery(`
SELECT column_name, column_type, is_nullable, column_default, column_comment, ordinal_position
FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY ordinal_position`).
		WithArgs("users").
		WillReturnRows(sqlmock.NewRows([]string{
			"column_name", "column_type", "is_nullable", "column_default", "column_comment", "ordinal_position",
		}).
			AddRow("id", "bigint unsigned", "NO", nil, "", 1).
			AddRow("email", "varchar(255)", "NO", nil, "", 2))
	mock.ExpectQuery(`
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = 'information_schema'
  AND table_name = 'STATISTICS'
  AND column_name = 'EXPRESSION'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`
SELECT index_name, non_unique, index_type, column_name, NULL AS index_expression
FROM information_schema.statistics
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY index_name, seq_in_index`).
		WithArgs("users").
		WillReturnRows(sqlmock.NewRows([]string{
			"index_name", "non_unique", "index_type", "column_name", "index_expression",
		}).
			AddRow("PRIMARY", 0, "BTREE", "id", nil).
			AddRow("idx_email", 1, "BTREE", "email", nil))
	mock.ExpectClose()

	got, err := (MySQLDialect{}).DescribeTable(context.Background(), db, "users")
	if err != nil {
		t.Fatalf("DescribeTable() error = %v", err)
	}
	want := TableDescription{
		Table: Table{Name: "users"},
		Columns: []Column{
			{Name: "id", Type: "bigint unsigned", Position: 1},
			{Name: "email", Type: "varchar(255)", Position: 2},
		},
		Indexes: []Index{
			{Name: "PRIMARY", Type: "BTREE", Columns: []string{"id"}, Unique: true},
			{Name: "idx_email", Type: "BTREE", Columns: []string{"email"}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DescribeTable() = %#v, want %#v", got, want)
	}
}

func TestClickHouseDescribeTableRecognizesWrappedNullableTypes(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled SQL expectations: %v", err)
		}
	})

	mock.ExpectQuery(`
SELECT name, type, default_expression, comment, position
FROM system.columns
WHERE database = currentDatabase() AND table = ?
ORDER BY position`).
		WithArgs("events").
		WillReturnRows(sqlmock.NewRows([]string{
			"name", "type", "default_expression", "comment", "position",
		}).
			AddRow("id", "UInt64", "", "event identifier", 1).
			AddRow("metadata", "LowCardinality( nullable( String ) )", "", "optional metadata", 2))
	mock.ExpectQuery(`
SELECT name, expr, type
FROM system.data_skipping_indices
WHERE database = currentDatabase() AND table = ?
ORDER BY name`).
		WithArgs("events").
		WillReturnRows(sqlmock.NewRows([]string{"name", "expr", "type"}))
	mock.ExpectClose()

	got, err := (ClickHouseDialect{}).DescribeTable(context.Background(), db, "events")
	if err != nil {
		t.Fatalf("DescribeTable() error = %v", err)
	}
	want := TableDescription{
		Table: Table{Name: "events"},
		Columns: []Column{
			{Name: "id", Type: "UInt64", Comment: "event identifier", Position: 1},
			{Name: "metadata", Type: "LowCardinality( nullable( String ) )", Nullable: true, Comment: "optional metadata", Position: 2},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DescribeTable() = %#v, want %#v", got, want)
	}
}

func TestClickHouseNullableRecognizesOnlyOuterNullableWrapper(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     bool
	}{
		{name: "direct nullable", typeName: "Nullable(String)", want: true},
		{name: "low cardinality nullable", typeName: "LowCardinality(Nullable(String))", want: true},
		{name: "case and whitespace", typeName: " lowcardinality ( nullable ( String ) ) ", want: true},
		{name: "array of nullable", typeName: "Array(Nullable(String))", want: false},
		{name: "map nullable value", typeName: "Map(String, Nullable(String))", want: false},
		{name: "tuple nullable field", typeName: "Tuple(Nullable(String))", want: false},
		{name: "low cardinality non-nullable", typeName: "LowCardinality(String)", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClickHouseNullable(tt.typeName); got != tt.want {
				t.Errorf("isClickHouseNullable(%q) = %t, want %t", tt.typeName, got, tt.want)
			}
		})
	}
}
