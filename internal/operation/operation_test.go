package operation

import (
	"errors"
	"strings"
	"testing"

	"github.com/dongrv/mcp-server-for-mysql/internal/database"
	"github.com/dongrv/mcp-server-for-mysql/internal/sqlguard"
)

func TestMySQLBuilderBuildsAllRetainedOperations(t *testing.T) {
	builder := MySQLBuilder{}

	tests := []struct {
		name string
		got  func() ([]string, error)
		want []string
	}{
		{"create table", func() ([]string, error) {
			return builder.CreateTable(CreateTableRequest{Table: "orders", Columns: []ColumnSpec{{Name: "id", Kind: "bigint", Nullable: false}, {Name: "note", Kind: "varchar", Length: 120}}})
		}, []string{"CREATE TABLE `orders` (`id` BIGINT NOT NULL, `note` VARCHAR(120) NOT NULL)"}},
		{"drop table", func() ([]string, error) { return builder.DropTable(DropTableRequest{Table: "orders"}) }, []string{"DROP TABLE `orders`"}},
		{"add columns", func() ([]string, error) {
			return builder.AddColumns(AddColumnsRequest{Table: "orders", Columns: []ColumnSpec{{Name: "note", Kind: "varchar", Length: 120}}})
		}, []string{"ALTER TABLE `orders` ADD COLUMN `note` VARCHAR(120) NOT NULL"}},
		{"drop columns", func() ([]string, error) {
			return builder.DropColumns(DropColumnsRequest{Table: "orders", Columns: []string{"note", "legacy"}})
		}, []string{"ALTER TABLE `orders` DROP COLUMN `note`, DROP COLUMN `legacy`"}},
		{"modify columns", func() ([]string, error) {
			return builder.ModifyColumns(ModifyColumnsRequest{Table: "orders", Columns: []ColumnSpec{{Name: "note", Kind: "text", Nullable: false}}})
		}, []string{"ALTER TABLE `orders` MODIFY COLUMN `note` TEXT NOT NULL"}},
		{"create index", func() ([]string, error) {
			return builder.CreateIndex(CreateIndexRequest{Table: "orders", Index: "idx_note", Columns: []string{"note"}, Unique: true})
		}, []string{"CREATE UNIQUE INDEX `idx_note` ON `orders` (`note`)"}},
		{"drop index", func() ([]string, error) {
			return builder.DropIndex(DropIndexRequest{Table: "orders", Index: "idx_note"})
		}, []string{"DROP INDEX `idx_note` ON `orders`"}},
		{"rename table", func() ([]string, error) {
			return builder.RenameTable(RenameTableRequest{From: "orders", To: "archived_orders"})
		}, []string{"RENAME TABLE `orders` TO `archived_orders`"}},
		{"copy structure", func() ([]string, error) {
			return builder.CopyTable(CopyTableRequest{Source: "orders", Destination: "orders_copy"})
		}, []string{"CREATE TABLE `orders_copy` LIKE `orders`"}},
		{"copy with data", func() ([]string, error) {
			return builder.CopyTable(CopyTableRequest{Source: "orders", Destination: "orders_copy", WithData: true})
		}, []string{"CREATE TABLE `orders_copy` LIKE `orders`", "INSERT INTO `orders_copy` SELECT * FROM `orders`"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.got()
			if err != nil {
				t.Fatalf("build operation: %v", err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("statement count = %d, want %d", len(got), len(test.want))
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Errorf("statement %d = %q, want %q", i, got[i], test.want[i])
				}
			}
		})
	}
}

func TestClickHouseBuilderBuildsSupportedOperations(t *testing.T) {
	builder := ClickHouseBuilder{}

	got, err := builder.CreateTable(CreateTableRequest{Table: "events", Columns: []ColumnSpec{{Name: "id", Kind: "uint64", Nullable: false}, {Name: "at", Kind: "datetime", Nullable: true}}})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	want := "CREATE TABLE `events` (`id` UInt64, `at` Nullable(DateTime)) ENGINE = MergeTree ORDER BY tuple()"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("create table = %#v, want %q", got, want)
	}

	got, err = builder.AddColumns(AddColumnsRequest{Table: "events", Columns: []ColumnSpec{{Name: "name", Kind: "string"}}})
	if err != nil || len(got) != 1 || got[0] != "ALTER TABLE `events` ADD COLUMN `name` String" {
		t.Fatalf("add columns = %#v, %v", got, err)
	}

	got, err = builder.DropColumns(DropColumnsRequest{Table: "events", Columns: []string{"name"}})
	if err != nil || len(got) != 1 || got[0] != "ALTER TABLE `events` DROP COLUMN `name`" {
		t.Fatalf("drop columns = %#v, %v", got, err)
	}

	got, err = builder.ModifyColumns(ModifyColumnsRequest{Table: "events", Columns: []ColumnSpec{{Name: "name", Kind: "string", Nullable: false}}})
	if err != nil || len(got) != 1 || got[0] != "ALTER TABLE `events` MODIFY COLUMN `name` String" {
		t.Fatalf("modify columns = %#v, %v", got, err)
	}

	got, err = builder.RenameTable(RenameTableRequest{From: "events", To: "archived_events"})
	if err != nil || len(got) != 1 || got[0] != "RENAME TABLE `events` TO `archived_events`" {
		t.Fatalf("rename = %#v, %v", got, err)
	}
}

func TestBuildersRejectUnsafeOrInvalidInput(t *testing.T) {
	builders := []Builder{MySQLBuilder{}, ClickHouseBuilder{}}
	for _, builder := range builders {
		t.Run("unsafe identifiers", func(t *testing.T) {
			_, err := builder.CreateTable(CreateTableRequest{Table: "orders; DROP TABLE users", Columns: []ColumnSpec{{Name: "id", Kind: "int"}}})
			if !errors.Is(err, database.ErrInvalidIdentifier) {
				t.Fatalf("error = %v, want invalid identifier", err)
			}
		})
	}

	_, err := MySQLBuilder{}.AddColumns(AddColumnsRequest{Table: "orders", Columns: []ColumnSpec{{Name: "note", Kind: "varchar; DROP TABLE orders"}}})
	if err == nil {
		t.Fatal("unknown type was accepted")
	}

	invalid := []ColumnSpec{
		{Name: "note", Kind: "varchar"},
		{Name: "amount", Kind: "decimal", Precision: 4, Scale: 5},
		{Name: "amount", Kind: "decimal", Precision: 66, Scale: 2},
	}
	for _, column := range invalid {
		_, err := MySQLBuilder{}.AddColumns(AddColumnsRequest{Table: "orders", Columns: []ColumnSpec{column}})
		if err == nil {
			t.Errorf("invalid column %#v was accepted", column)
		}
	}

	_, err = MySQLBuilder{}.AddColumns(AddColumnsRequest{Table: "orders", Columns: []ColumnSpec{{Name: "same", Kind: "int"}, {Name: "same", Kind: "int"}}})
	if err == nil {
		t.Fatal("duplicate columns were accepted")
	}
}

func TestClickHouseUnsupportedOperationsReturnCapabilityError(t *testing.T) {
	builder := ClickHouseBuilder{}
	tests := []func() ([]string, error){
		func() ([]string, error) {
			return builder.CreateIndex(CreateIndexRequest{Table: "events", Index: "idx", Columns: []string{"id"}})
		},
		func() ([]string, error) { return builder.DropIndex(DropIndexRequest{Table: "events", Index: "idx"}) },
		func() ([]string, error) {
			return builder.CopyTable(CopyTableRequest{Source: "events", Destination: "copy"})
		},
	}
	for _, operation := range tests {
		_, err := operation()
		if !errors.Is(err, database.ErrUnsupportedCapability) {
			t.Errorf("error = %v, want unsupported capability", err)
		}
	}
}

func TestMySQLRetainedOperationsAreHighRisk(t *testing.T) {
	builder := MySQLBuilder{}
	statements, err := builder.CopyTable(CopyTableRequest{Source: "orders", Destination: "orders_copy", WithData: true})
	if err != nil {
		t.Fatalf("copy table: %v", err)
	}
	analyzer, err := sqlguard.NewMySQLAnalyzer("")
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	plan, err := analyzer.Analyze(strings.Join(statements, "; "))
	if err != nil {
		t.Fatalf("analyze operation: %v", err)
	}
	if plan.Risk != sqlguard.HighRisk {
		t.Errorf("operation risk = %s, want %s", plan.Risk, sqlguard.HighRisk)
	}
}

func TestGeneratedMutationsAreHighRiskForEachDialect(t *testing.T) {
	tests := []struct {
		name    string
		analyze func(string) (sqlguard.Plan, error)
		build   func() ([]string, error)
		atomic  bool
	}{
		{"mysql drop", mustMySQLAnalyzer(t), func() ([]string, error) { return MySQLBuilder{}.DropTable(DropTableRequest{Table: "orders"}) }, true},
		{"mysql rename", mustMySQLAnalyzer(t), func() ([]string, error) {
			return MySQLBuilder{}.RenameTable(RenameTableRequest{From: "orders", To: "archived_orders"})
		}, true},
		{"mysql index", mustMySQLAnalyzer(t), func() ([]string, error) {
			return MySQLBuilder{}.CreateIndex(CreateIndexRequest{Table: "orders", Index: "idx_id", Columns: []string{"id"}})
		}, true},
		{"mysql copy", mustMySQLAnalyzer(t), func() ([]string, error) {
			return MySQLBuilder{}.CopyTable(CopyTableRequest{Source: "orders", Destination: "orders_copy", WithData: true})
		}, false},
		{"clickhouse drop", sqlguard.NewClickHouseAnalyzer().Analyze, func() ([]string, error) { return ClickHouseBuilder{}.DropTable(DropTableRequest{Table: "events"}) }, false},
		{"clickhouse add column", sqlguard.NewClickHouseAnalyzer().Analyze, func() ([]string, error) {
			return ClickHouseBuilder{}.AddColumns(AddColumnsRequest{Table: "events", Columns: []ColumnSpec{{Name: "note", Kind: "string"}}})
		}, false},
		{"clickhouse rename", sqlguard.NewClickHouseAnalyzer().Analyze, func() ([]string, error) {
			return ClickHouseBuilder{}.RenameTable(RenameTableRequest{From: "events", To: "archived_events"})
		}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			statements, err := test.build()
			if err != nil {
				t.Fatalf("build operation: %v", err)
			}
			plan, err := test.analyze(strings.Join(statements, "; "))
			if err != nil {
				t.Fatalf("analyze operation: %v", err)
			}
			if risk := plan.RiskForAtomicBatches(test.atomic); risk != sqlguard.HighRisk {
				t.Errorf("operation risk = %s, want %s", risk, sqlguard.HighRisk)
			}
		})
	}
}

func mustMySQLAnalyzer(t *testing.T) func(string) (sqlguard.Plan, error) {
	t.Helper()
	analyzer, err := sqlguard.NewMySQLAnalyzer("")
	if err != nil {
		t.Fatalf("new MySQL analyzer: %v", err)
	}
	return analyzer.Analyze
}
