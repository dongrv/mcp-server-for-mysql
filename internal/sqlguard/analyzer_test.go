package sqlguard

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func newMySQLAnalyzer(t *testing.T, serverVersion string) Analyzer {
	t.Helper()
	analyzer, err := NewMySQLAnalyzer(serverVersion)
	if err != nil {
		t.Fatalf("NewMySQLAnalyzer(%q): %v", serverVersion, err)
	}
	return analyzer
}

func TestMySQLAnalyzerClassifiesStatementsAndRisk(t *testing.T) {
	analyzer := newMySQLAnalyzer(t, "")

	plan, err := analyzer.Analyze("SELECT id FROM orders WHERE id = ?")
	if err != nil {
		t.Fatalf("Analyze SELECT: %v", err)
	}
	if got, want := plan.Statements[0].Kind, ReadOnly; got != want {
		t.Errorf("kind = %q, want %q", got, want)
	}
	if !plan.ReadOnly {
		t.Error("read-only SELECT plan marked writable")
	}
	if got, want := plan.Statements[0].NormalizedSQL, "select id from orders where id = :v1"; got != want {
		t.Errorf("normalized SQL = %q, want %q", got, want)
	}

	for _, tc := range []struct {
		name     string
		sql      string
		kind     StatementKind
		risk     Risk
		hasWhere bool
	}{
		{name: "delete without where", sql: "DELETE FROM orders", kind: Write, risk: HighRisk},
		{name: "delete with where", sql: "DELETE FROM orders WHERE id = ?", kind: Write, risk: LowRisk, hasWhere: true},
		{name: "update without where", sql: "UPDATE orders SET status = 'done'", kind: Write, risk: HighRisk},
		{name: "insert", sql: "INSERT INTO orders (id) VALUES (1)", kind: Write, risk: LowRisk},
		{name: "ddl", sql: "ALTER TABLE orders ADD COLUMN note TEXT", kind: DDL, risk: HighRisk},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := analyzer.Analyze(tc.sql)
			if err != nil {
				t.Fatalf("Analyze: %v", err)
			}
			statement := plan.Statements[0]
			if statement.Kind != tc.kind || plan.Risk != tc.risk || statement.HasWhereClause != tc.hasWhere {
				t.Errorf("got statement=%+v plan=%+v", statement, plan)
			}
		})
	}
}

func TestMySQLAnalyzerRejectsUnsafeOrUnsupportedSQL(t *testing.T) {
	analyzer := newMySQLAnalyzer(t, "")
	for _, sql := range []string{
		"",
		"-- standalone comment",
		"SELECT 1; totally_not_sql",
		"START TRANSACTION",
		"SET sql_mode = 'ANSI'",
		"WITH changed AS (DELETE FROM orders RETURNING id) SELECT * FROM changed",
	} {
		t.Run(sql, func(t *testing.T) {
			_, err := analyzer.Analyze(sql)
			if !errors.Is(err, ErrUnsafeOrUnsupportedSQL) {
				t.Errorf("Analyze(%q) error = %v, want ErrUnsafeOrUnsupportedSQL", sql, err)
			}
		})
	}
}

func TestMySQLAnalyzerRejectsSelectIntoFile(t *testing.T) {
	analyzer := newMySQLAnalyzer(t, "")
	for _, sql := range []string{
		"SELECT 1 INTO OUTFILE '/tmp/export'",
		"SELECT 1 INTO DUMPFILE '/tmp/export'",
		"SELECT 1 UNION SELECT 2 INTO OUTFILE '/tmp/export'",
	} {
		t.Run(sql, func(t *testing.T) {
			_, err := analyzer.Analyze(sql)
			if !errors.Is(err, ErrUnsafeOrUnsupportedSQL) {
				t.Errorf("Analyze(%q) error = %v, want ErrUnsafeOrUnsupportedSQL", sql, err)
			}
		})
	}

	plan, err := analyzer.Analyze("SELECT 1")
	if err != nil {
		t.Fatalf("Analyze ordinary SELECT: %v", err)
	}
	if !plan.ReadOnly || plan.Statements[0].Kind != ReadOnly {
		t.Errorf("ordinary SELECT plan = %+v, want read-only", plan)
	}
}

func TestMySQLAnalyzerRejectsLockingAndConnectionStateSelects(t *testing.T) {
	analyzer := newMySQLAnalyzer(t, "")
	for _, sql := range []string{
		"SELECT id FROM orders FOR UPDATE",
		"SELECT id FROM orders LOCK IN SHARE MODE",
		"SELECT id FROM orders UNION SELECT id FROM archived_orders FOR UPDATE",
		"SELECT GET_LOCK('mcp-lock', 1)",
		"SELECT RELEASE_LOCK('mcp-lock')",
		"SELECT IS_FREE_LOCK('mcp-lock')",
		"SELECT SLEEP(1)",
	} {
		t.Run(sql, func(t *testing.T) {
			_, err := analyzer.Analyze(sql)
			if !errors.Is(err, ErrUnsafeOrUnsupportedSQL) {
				t.Errorf("Analyze(%q) error = %v, want ErrUnsafeOrUnsupportedSQL", sql, err)
			}
		})
	}

	plan, err := analyzer.Analyze("SELECT COUNT(*) FROM orders")
	if err != nil {
		t.Fatalf("Analyze ordinary SELECT: %v", err)
	}
	if !plan.ReadOnly || plan.Statements[0].Kind != ReadOnly {
		t.Errorf("ordinary SELECT plan = %+v, want read-only", plan)
	}
}

func TestMySQLAnalyzerRejectsUserVariableAssignments(t *testing.T) {
	analyzer := newMySQLAnalyzer(t, "")
	for _, sql := range []string{
		"SELECT @mcp_state := 1",
		"SELECT COALESCE(@mcp_state := 1, 0)",
		"SELECT 1 UNION SELECT @mcp_state := 1",
	} {
		t.Run(sql, func(t *testing.T) {
			_, err := analyzer.Analyze(sql)
			if !errors.Is(err, ErrUnsafeOrUnsupportedSQL) {
				t.Errorf("Analyze(%q) error = %v, want ErrUnsafeOrUnsupportedSQL", sql, err)
			}
		})
	}

	// Vitess represents reads as a variable expression, distinct from AssignmentExpr.
	plan, err := analyzer.Analyze("SELECT @mcp_state")
	if err != nil {
		t.Fatalf("Analyze user variable read: %v", err)
	}
	if !plan.ReadOnly || plan.Statements[0].Kind != ReadOnly {
		t.Errorf("user variable read plan = %+v, want read-only", plan)
	}
}

func TestMySQLAnalyzerRejectsGTIDWaitFunctions(t *testing.T) {
	analyzer := newMySQLAnalyzer(t, "")
	for _, sql := range []string{
		"SELECT WAIT_FOR_EXECUTED_GTID_SET('3E11FA47-71CA-11E1-9E33-C80AA9429562:23')",
		"SELECT WAIT_UNTIL_SQL_THREAD_AFTER_GTIDS('3E11FA47-71CA-11E1-9E33-C80AA9429562:23')",
	} {
		t.Run(sql, func(t *testing.T) {
			_, err := analyzer.Analyze(sql)
			if !errors.Is(err, ErrUnsafeOrUnsupportedSQL) {
				t.Errorf("Analyze(%q) error = %v, want ErrUnsafeOrUnsupportedSQL", sql, err)
			}
		})
	}
}

func TestNewMySQLAnalyzerVersionHandling(t *testing.T) {
	newMySQLAnalyzer(t, "")
	newMySQLAnalyzer(t, "8.0.36")

	if _, err := NewMySQLAnalyzer("not-a-version"); err == nil {
		t.Error("NewMySQLAnalyzer accepted malformed server version")
	}
}

func TestClickHouseAnalyzerSplitsStatementsAndDefaultsToDeny(t *testing.T) {
	analyzer := NewClickHouseAnalyzer()

	plan, err := analyzer.Analyze("SELECT ';' AS value; DROP TABLE events")
	if err != nil {
		t.Fatalf("Analyze quoted semicolon: %v", err)
	}
	if got, want := len(plan.Statements), 2; got != want {
		t.Fatalf("statement count = %d, want %d", got, want)
	}
	if plan.Statements[0].Kind != ReadOnly || plan.Statements[1].Kind != DDL || plan.Risk != HighRisk {
		t.Errorf("got statements=%+v risk=%q", plan.Statements, plan.Risk)
	}

	for _, sql := range []string{
		"SELECT FROM",
		"totally_not_sql",
		"SET max_threads = 1",
		"BEGIN TRANSACTION",
		"/* standalone comment */",
		"WITH changed AS (DELETE FROM events WHERE id = 1) SELECT * FROM changed",
	} {
		t.Run(sql, func(t *testing.T) {
			_, err := analyzer.Analyze(sql)
			if !errors.Is(err, ErrUnsafeOrUnsupportedSQL) {
				t.Errorf("Analyze(%q) error = %v, want ErrUnsafeOrUnsupportedSQL", sql, err)
			}
		})
	}
}

func TestClickHouseAnalyzerRejectsTableFunctions(t *testing.T) {
	analyzer := NewClickHouseAnalyzer()
	for _, sql := range []string{
		"SELECT * FROM file('/tmp/data.csv', 'CSV')",
		"SELECT * FROM url('https://example.test/data.csv', 'CSV')",
		"SELECT * FROM remote('cluster.example', 'db', 'table')",
		"SELECT * FROM mysql('mysql.example:3306', 'db', 'table', 'user', 'password')",
		"SELECT * FROM postgres('postgres.example:5432', 'db', 'table', 'user', 'password')",
		"SELECT * FROM s3('https://example.test/data.csv', 'CSV')",
		"SELECT * FROM hdfs('hdfs://cluster.example/data.csv', 'CSV')",
		"SELECT * FROM azureBlobStorage('https://example.test/container/blob.csv', 'account', 'key', 'CSV')",
	} {
		t.Run(sql, func(t *testing.T) {
			_, err := analyzer.Analyze(sql)
			if !errors.Is(err, ErrUnsafeOrUnsupportedSQL) {
				t.Errorf("Analyze(%q) error = %v, want ErrUnsafeOrUnsupportedSQL", sql, err)
			}
		})
	}

	plan, err := analyzer.Analyze("SELECT count() FROM events WHERE kind = 'login'")
	if err != nil {
		t.Fatalf("Analyze ordinary table SELECT: %v", err)
	}
	if !plan.ReadOnly || plan.Statements[0].Kind != ReadOnly {
		t.Errorf("ordinary table SELECT plan = %+v, want read-only", plan)
	}
}

func TestAnalyzersRejectEmptyAndCommentOnlyStatementGroups(t *testing.T) {
	for _, dialect := range []struct {
		name     string
		analyzer Analyzer
	}{
		{name: "mysql", analyzer: newMySQLAnalyzer(t, "")},
		{name: "clickhouse", analyzer: NewClickHouseAnalyzer()},
	} {
		t.Run(dialect.name, func(t *testing.T) {
			for _, sql := range []string{
				"SELECT 1;;SELECT 2",
				";SELECT 1",
				"SELECT 1;;",
				"SELECT 1; /* standalone */; SELECT 2",
			} {
				t.Run(sql, func(t *testing.T) {
					_, err := dialect.analyzer.Analyze(sql)
					if !errors.Is(err, ErrUnsafeOrUnsupportedSQL) {
						t.Errorf("Analyze(%q) error = %v, want ErrUnsafeOrUnsupportedSQL", sql, err)
					}
				})
			}
		})
	}
}

func TestAnalyzersRejectTrailingCommentOnlyStatementGroups(t *testing.T) {
	for _, dialect := range []struct {
		name     string
		analyzer Analyzer
	}{
		{name: "mysql", analyzer: newMySQLAnalyzer(t, "")},
		{name: "clickhouse", analyzer: NewClickHouseAnalyzer()},
	} {
		t.Run(dialect.name, func(t *testing.T) {
			for _, sql := range []string{
				"SELECT 1; /* standalone */",
				"SELECT 1; -- standalone\n",
			} {
				t.Run(sql, func(t *testing.T) {
					_, err := dialect.analyzer.Analyze(sql)
					if !errors.Is(err, ErrUnsafeOrUnsupportedSQL) {
						t.Errorf("Analyze(%q) error = %v, want ErrUnsafeOrUnsupportedSQL", sql, err)
					}
				})
			}
		})
	}
}

func TestAnalyzersAcceptCommentsWithinStatements(t *testing.T) {
	for _, dialect := range []struct {
		name     string
		analyzer Analyzer
	}{
		{name: "mysql", analyzer: newMySQLAnalyzer(t, "")},
		{name: "clickhouse", analyzer: NewClickHouseAnalyzer()},
	} {
		t.Run(dialect.name, func(t *testing.T) {
			for _, sql := range []string{
				"SELECT /* hint */ 1",
				"SELECT 1;",
			} {
				t.Run(sql, func(t *testing.T) {
					plan, err := dialect.analyzer.Analyze(sql)
					if err != nil {
						t.Fatalf("Analyze: %v", err)
					}
					if got, want := len(plan.Statements), 1; got != want {
						t.Errorf("statement count = %d, want %d", got, want)
					}
				})
			}
		})
	}
}

func TestSplitSQLStatementGroupsRespectsQuotesAndComments(t *testing.T) {
	groups, err := splitSQLStatementGroups("SELECT ';'; /* comment ; */ SELECT \"quoted;value\"; -- comment ;\nSELECT `back;tick`;")
	if err != nil {
		t.Fatalf("splitSQLStatementGroups: %v", err)
	}
	want := []string{
		"SELECT ';'",
		"/* comment ; */ SELECT \"quoted;value\"",
		"-- comment ;\nSELECT `back;tick`",
	}
	if !reflect.DeepEqual(groups, want) {
		t.Errorf("groups = %#v, want %#v", groups, want)
	}
}

func TestAnalyzerRejectsMoreThanDefaultMaximumStatements(t *testing.T) {
	_, err := newMySQLAnalyzer(t, "").Analyze(strings.Repeat("SELECT 1;", DefaultMaxStatements+1))
	if !errors.Is(err, ErrUnsafeOrUnsupportedSQL) {
		t.Errorf("Analyze over limit error = %v, want ErrUnsafeOrUnsupportedSQL", err)
	}
}

func TestPlanRiskForNonAtomicBatches(t *testing.T) {
	plan := Plan{Risk: LowRisk, Statements: []Statement{{Kind: Write}, {Kind: Write}}}
	if got, want := plan.RiskForAtomicBatches(false), HighRisk; got != want {
		t.Errorf("non-atomic risk = %q, want %q", got, want)
	}
	if got, want := plan.RiskForAtomicBatches(true), LowRisk; got != want {
		t.Errorf("atomic risk = %q, want %q", got, want)
	}
}

func TestExplainSelectIsReadOnlyAndExplainAnalyzeIsRejected(t *testing.T) {
	tests := []struct {
		name     string
		analyzer Analyzer
		sql      string
		wantErr  bool
	}{
		{"mysql explain select", newMySQLAnalyzer(t, ""), "EXPLAIN SELECT id FROM orders", false},
		{"mysql explain analyze", newMySQLAnalyzer(t, ""), "EXPLAIN ANALYZE SELECT id FROM orders", true},
		{"clickhouse explain select", NewClickHouseAnalyzer(), "EXPLAIN SYNTAX SELECT id FROM events", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, err := test.analyzer.Analyze(test.sql)
			if test.wantErr {
				if !errors.Is(err, ErrUnsafeOrUnsupportedSQL) {
					t.Fatalf("Analyze error = %v, want unsafe SQL", err)
				}
				return
			}
			if err != nil || !plan.ReadOnly {
				t.Fatalf("Analyze = %#v, %v; want read-only plan", plan, err)
			}
		})
	}
}
