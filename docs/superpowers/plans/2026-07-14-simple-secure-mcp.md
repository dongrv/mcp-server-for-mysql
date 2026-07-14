# Simple Secure Multi-Source MCP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the single-MySQL, unsafe MCP implementation with a query-first, stateless MCP server that supports multiple MySQL and ClickHouse sources and requires preview confirmation for configured high-risk work.

**Architecture:** MCP handlers decode typed inputs and delegate every database action to one pipeline: dialect planning, SQL safety analysis, risk policy, preview hashing, then bounded execution. Source adapters own connection pools, metadata, identifier quoting, SQL parsing, and capability declarations; high-level table, column, and index requests become dialect operations instead of SQL assembled inside handlers.

**Tech Stack:** Go 1.24 toolchain, MCP Go SDK, `database/sql`, `go-sql-driver/mysql`, `clickhouse-go/v2`, Vitess SQL parser, AfterShip ClickHouse SQL parser, `gopkg.in/yaml.v3`, Docker Compose integration fixtures.

---

## File Structure

| Path | Responsibility |
| --- | --- |
| `internal/config/config.go` | Parse and validate one-file YAML configuration and environment references. |
| `internal/database/*.go` | Own source pools, metadata, quoting, and MySQL/ClickHouse capabilities. |
| `internal/sqlguard/*.go` | Parse SQL by dialect, normalize statements, and classify risk. |
| `internal/operation/*.go` | Build safe source-specific SQL from structured schema inputs. |
| `internal/execution/*.go` | Apply quick/strict policy, build preview hashes, and execute bounded plans. |
| `internal/tools/*.go` | Decode MCP inputs and expose the generic query-first tool contract. |
| `internal/observability/logger.go` | Emit credential-free structured operational logs without audit persistence. |
| `internal/integration/*.go` | Build-tagged live MySQL and ClickHouse verification. |
| `docs/security-audit.md` | Record current vulnerabilities, remediations, tests, and residual risks. |

The legacy `internal/mysql` package, cross-request transaction handlers, and hand-maintained `mysql_*` schemas are removed only after their source-aware replacements are tested. Cross-request transactions have no replacement because they conflict with the accepted stateless boundary.

## Task 1: Configuration and dependencies

**Files:**
- Create: `config.example.yaml`
- Modify: `go.mod`
- Modify: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Delete: `env.example`

- [ ] **Step 1: Write failing configuration tests**

```go
func TestLoadExpandsSourceDSNAndDefaultsToQuick(t *testing.T) {
    path := writeConfig(t, "sources:\n  - name: orders\n    type: mysql\n    dsn: ${ORDERS_DSN}\n")
    cfg, err := Load(path, func(key string) string {
        if key == "ORDERS_DSN" { return "user:pass@tcp(db:3306)/orders" }
        return ""
    })
    require.NoError(t, err)
    require.Equal(t, QuickMode, cfg.Mode)
    require.Equal(t, "orders", cfg.Sources[0].Name)
}

func TestLoadRejectsMissingDSNAndDuplicateSourceNames(t *testing.T) {
    _, err := Load(writeConfig(t, "sources:\n  - name: orders\n    type: mysql\n    dsn: ${MISSING}\n"), func(string) string { return "" })
    require.ErrorContains(t, err, "MISSING")
    _, err = Load(writeConfig(t, "sources:\n  - name: orders\n    type: mysql\n    dsn: x\n  - name: orders\n    type: clickhouse\n    dsn: y\n"), os.Getenv)
    require.ErrorContains(t, err, "duplicate source name")
}

func writeConfig(t *testing.T, content string) string {
    t.Helper()
    path := filepath.Join(t.TempDir(), "config.yaml")
    require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
    return path
}
```

- [ ] **Step 2: Run the tests to prove the contract is absent**

Run: `go test ./internal/config -run 'TestLoad' -count=1`

Expected: FAIL because `Load`, `QuickMode`, and YAML source fields do not exist.

- [ ] **Step 3: Implement the exact configuration contract**

Run: `go get gopkg.in/yaml.v3@v3.0.1 github.com/stretchr/testify@v1.10.0 github.com/DATA-DOG/go-sqlmock@v1.5.2`

```go
type Mode string

const (
    QuickMode  Mode = "quick"
    StrictMode Mode = "strict"
)

type SourceConfig struct {
    Name string `yaml:"name"`
    Type string `yaml:"type"`
    DSN  string `yaml:"dsn"`
}

type Config struct {
    Mode    Mode           `yaml:"mode"`
    Sources []SourceConfig `yaml:"sources"`
}

func Load(path string, lookupEnv func(string) string) (Config, error)
```

`Load` replaces only exact `${NAME}` tokens; rejects empty substitutions, absent sources, duplicate names, and source types other than `mysql` or `clickhouse`; defaults an absent mode to `quick`; and never returns a DSN in an error.

- [ ] **Step 4: Add the minimal example and remove old variables**

```yaml
mode: quick

sources:
  - name: orders
    type: mysql
    dsn: ${ORDERS_MYSQL_DSN}
  - name: analytics
    type: clickhouse
    dsn: ${ANALYTICS_CLICKHOUSE_DSN}
```

Delete `env.example`, whose single-MySQL variables no longer describe the application.

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/config -count=1`

Expected: PASS.

```bash
git add go.mod go.sum config.example.yaml internal/config/config.go internal/config/config_test.go env.example && git commit -m "feat: load simple multi-source configuration"
```

## Task 2: Source registry and dialect boundary

**Files:**
- Create: `internal/database/types.go`
- Create: `internal/database/registry.go`
- Create: `internal/database/mysql.go`
- Create: `internal/database/clickhouse.go`
- Create: `internal/database/identifier.go`
- Create: `internal/database/registry_test.go`
- Modify: `go.mod`

- [ ] **Step 1: Write failing registry and identifier tests**

```go
type fakeSource struct {
    id     string
    closed bool
}

func newFakeSource(id string) *fakeSource { return &fakeSource{id: id} }
func (s *fakeSource) ID() string { return s.id }
func (s *fakeSource) Engine() string { return "mysql" }
func (s *fakeSource) DB() *sql.DB { return nil }
func (s *fakeSource) Dialect() Dialect { return MySQLDialect{} }
func (s *fakeSource) Capabilities() Capability { return MySQLDialect{}.Capabilities() }
func (s *fakeSource) Close() error { s.closed = true; return nil }

func TestRegistryRejectsUnknownSourceAndClosesEveryPool(t *testing.T) {
    first, second := newFakeSource("orders"), newFakeSource("analytics")
    registry := NewRegistry([]Source{first, second})
    _, err := registry.Source("missing")
    require.ErrorIs(t, err, ErrUnknownSource)
    require.NoError(t, registry.Close())
    require.True(t, first.closed)
    require.True(t, second.closed)
}

func TestMySQLIdentifierRejectsInjectionAndQuotesSafeNames(t *testing.T) {
    got, err := MySQLDialect{}.QuoteIdentifier("order_items")
    require.NoError(t, err)
    require.Equal(t, "`order_items`", got)
    require.Error(t, MySQLDialect{}.ValidateIdentifier("orders`; DROP TABLE users; --"))
}
```

- [ ] **Step 2: Run the registry tests**

Run: `go test ./internal/database -run 'TestRegistry|TestMySQLIdentifier' -count=1`

Expected: FAIL because `Registry`, `Source`, and dialect types do not exist.

- [ ] **Step 3: Implement the source abstraction**

Run: `go get github.com/ClickHouse/clickhouse-go/v2@v2.41.0`

```go
type Capability struct {
    Transactions  bool
    AtomicBatches bool
    CopyTable     bool
    AlterColumns  bool
}

type Source interface {
    ID() string
    Engine() string
    DB() *sql.DB
    Dialect() Dialect
    Capabilities() Capability
    Close() error
}

type Dialect interface {
    Name() string
    Capabilities() Capability
    ValidateIdentifier(string) error
    QuoteIdentifier(string) (string, error)
    ListTables(context.Context, *sql.DB) ([]Table, error)
    DescribeTable(context.Context, *sql.DB, string) (TableDescription, error)
}

func OpenRegistry(ctx context.Context, cfg config.Config) (*Registry, error)
func (r *Registry) Source(id string) (Source, error)
func (r *Registry) Close() error
```

`OpenRegistry` creates and `PingContext` checks every configured pool before returning. MySQL uses the current driver; ClickHouse uses `clickhouse.OpenDB`. Both pools use fixed safe defaults and idempotent close. Valid identifiers are non-empty ASCII names beginning with a letter or underscore, followed only by letters, digits, or underscores; both initial dialects quote validated names with backticks.

- [ ] **Step 4: Add metadata and declared capabilities**

```go
func TestClickHouseCapabilitiesAreExplicit(t *testing.T) {
    caps := ClickHouseDialect{}.Capabilities()
    require.False(t, caps.Transactions)
    require.False(t, caps.AtomicBatches)
}
```

Use `information_schema` for MySQL and `system.tables`, `system.columns`, and `system.data_skipping_indices` for ClickHouse. Return normalized `Table`, `Column`, and `Index` data; handlers must not issue engine-specific metadata SQL.

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/database -count=1`

Expected: PASS.

```bash
git add go.mod go.sum internal/database && git commit -m "feat: add multi-source database registry"
```

## Task 3: SQL parsing and risk classification

**Files:**
- Create: `internal/sqlguard/model.go`
- Create: `internal/sqlguard/analyzer.go`
- Create: `internal/sqlguard/mysql.go`
- Create: `internal/sqlguard/clickhouse.go`
- Create: `internal/sqlguard/analyzer_test.go`
- Modify: `go.mod`

- [ ] **Step 1: Write failing risk-classification fixtures**

```go
func TestAnalyzeMySQLClassifiesRiskAndRejectsUnknownSyntax(t *testing.T) {
    analyzer := NewMySQLAnalyzer()
    plan, err := analyzer.Analyze("SELECT id FROM orders WHERE id = ?")
    require.NoError(t, err)
    require.Equal(t, ReadOnly, plan.Statements[0].Kind)

    plan, err = analyzer.Analyze("DELETE FROM orders")
    require.NoError(t, err)
    require.Equal(t, HighRisk, plan.Risk)

    _, err = analyzer.Analyze("SELECT 1; totally_not_sql")
    require.ErrorIs(t, err, ErrUnsafeOrUnsupportedSQL)
}

func TestAnalyzeClickHouseSplitsStatementsWithoutSplittingStrings(t *testing.T) {
    plan, err := NewClickHouseAnalyzer().Analyze("SELECT ';' AS value; DROP TABLE events")
    require.NoError(t, err)
    require.Len(t, plan.Statements, 2)
    require.Equal(t, HighRisk, plan.Risk)
}

func TestAnalyzerRejectsMoreThanDefaultMaximumStatements(t *testing.T) {
    _, err := NewMySQLAnalyzer().Analyze(strings.Repeat("SELECT 1;", DefaultMaxStatements+1))
    require.ErrorIs(t, err, ErrUnsafeOrUnsupportedSQL)
}
```

- [ ] **Step 2: Run the SQL guard tests**

Run: `go test ./internal/sqlguard -count=1`

Expected: FAIL because the analyzer package does not exist.

- [ ] **Step 3: Add parser dependencies and analyzer types**

Run: `go get vitess.io/vitess@v24.0.2 github.com/AfterShip/clickhouse-sql-parser@v0.5.2`

```go
type StatementKind string

const (
    ReadOnly StatementKind = "read_only"
    Write    StatementKind = "write"
    DDL      StatementKind = "ddl"
    Session  StatementKind = "session"
)

type Risk string

const (
    LowRisk  Risk = "low"
    HighRisk Risk = "high"
)

var ErrUnsafeOrUnsupportedSQL = errors.New("unsafe or unsupported SQL")

type Statement struct {
    SQL            string
    NormalizedSQL  string
    Kind           StatementKind
    HasWhereClause bool
}

type Plan struct {
    Statements []Statement
    Risk       Risk
    ReadOnly   bool
}

type Analyzer interface { Analyze(sql string) (Plan, error) }

const DefaultMaxStatements = 50
```

The MySQL analyzer uses a Vitess AST parser. The ClickHouse analyzer uses `parser.NewParser(sql).ParseStmts()` from AfterShip and formats each AST into `NormalizedSQL`. Neither implementation may use a raw string-prefix classification. Both reject parser errors, empty groups, transaction control, session changes, and comments used as standalone statements.

- [ ] **Step 4: Add complete rule coverage**

```go
func TestRiskRules(t *testing.T) {
    cases := []struct{ sql string; want Risk }{
        {"DROP TABLE orders", HighRisk},
        {"TRUNCATE TABLE orders", HighRisk},
        {"ALTER TABLE orders ADD COLUMN note TEXT", HighRisk},
        {"RENAME TABLE orders TO archived_orders", HighRisk},
        {"UPDATE orders SET state = 'done'", HighRisk},
        {"DELETE FROM orders WHERE id = ?", LowRisk},
    }
    for _, tc := range cases {
        t.Run(tc.sql, func(t *testing.T) {
            plan, err := NewMySQLAnalyzer().Analyze(tc.sql)
            require.NoError(t, err)
            require.Equal(t, tc.want, plan.Risk)
        })
    }
}
```

`query` accepts only plans whose `ReadOnly` value is true. A migration is a raw SQL operation but always receives high-risk policy treatment. A multi-statement plan is high risk whenever its source does not declare `AtomicBatches`.

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/sqlguard -count=1 && go vet ./internal/sqlguard`

Expected: PASS.

```bash
git add go.mod go.sum internal/sqlguard && git commit -m "feat: analyze SQL safely by database dialect"
```

## Task 4: Stateless preview and confirmation policy

**Files:**
- Create: `internal/execution/model.go`
- Create: `internal/execution/policy.go`
- Create: `internal/execution/preview.go`
- Create: `internal/execution/policy_test.go`

- [ ] **Step 1: Write failing quick, strict, and altered-preview tests**

```go
func TestQuickModePreviewsOnlyHighRiskPlans(t *testing.T) {
    decision := Decide(config.QuickMode, sqlguard.Plan{Risk: sqlguard.HighRisk}, Confirmation{})
    require.Equal(t, PreviewRequired, decision.State)
    decision = Decide(config.QuickMode, sqlguard.Plan{Risk: sqlguard.LowRisk}, Confirmation{})
    require.Equal(t, ExecuteNow, decision.State)
}

func TestStrictModeRejectsAlteredPreview(t *testing.T) {
    first := BuildPreview("orders", "execute_sql", []string{"DELETE FROM orders WHERE id = ?"}, []any{int64(1)}, true, sqlguard.HighRisk)
    plan := sqlguard.Plan{Statements: []sqlguard.Statement{{NormalizedSQL: "SELECT 1", Kind: sqlguard.ReadOnly}}, Risk: sqlguard.LowRisk, ReadOnly: true}
    decision := Decide(config.StrictMode, plan, Confirmation{Confirm: true, PreviewHash: first.PreviewHash + "x"})
    require.Equal(t, PreviewMismatch, decision.State)
}
```

- [ ] **Step 2: Run the policy tests**

Run: `go test ./internal/execution -run 'TestQuickMode|TestStrictMode' -count=1`

Expected: FAIL because the policy package does not exist.

- [ ] **Step 3: Implement deterministic preview data**

```go
type Confirmation struct {
    Confirm     bool   `json:"confirm"`
    PreviewHash string `json:"preview_hash,omitempty"`
}

type DecisionState string

const (
    ExecuteNow      DecisionState = "execute"
    PreviewRequired DecisionState = "confirmation_required"
    PreviewMismatch DecisionState = "preview_mismatch"
)

type Decision struct { State DecisionState }

type Preview struct {
    State       string   `json:"state"`
    SQL         []string `json:"sql"`
    Risk        string   `json:"risk"`
    Atomic      bool     `json:"atomic"`
    PreviewHash string   `json:"preview_hash"`
}

func BuildPreview(sourceID, tool string, sql []string, parameters []any, atomic bool, risk sqlguard.Risk) Preview
func Decide(mode config.Mode, plan sqlguard.Plan, confirmation Confirmation) Decision
```

`BuildPreview` canonicalizes source ID, tool name, normalized statements in order, JSON parameter values, risk, and atomicity; hashes those bytes with SHA-256; and encodes lower-case hexadecimal. `Decide` returns only `ExecuteNow`, `PreviewRequired`, or `PreviewMismatch` and never stores a hash. Strict mode previews every non-empty plan; quick mode previews every high-risk plan.

- [ ] **Step 4: Bind source, statement order, and parameters in tests**

```go
func TestPreviewHashBindsSourceStatementOrderAndParameters(t *testing.T) {
    a := BuildPreview("orders", "execute_sql", []string{"UPDATE a SET x = ?", "UPDATE b SET x = ?"}, []any{1, 2}, true, sqlguard.HighRisk)
    b := BuildPreview("analytics", "execute_sql", []string{"UPDATE a SET x = ?", "UPDATE b SET x = ?"}, []any{1, 2}, true, sqlguard.HighRisk)
    c := BuildPreview("orders", "execute_sql", []string{"UPDATE b SET x = ?", "UPDATE a SET x = ?"}, []any{1, 2}, true, sqlguard.HighRisk)
    require.NotEqual(t, a.PreviewHash, b.PreviewHash)
    require.NotEqual(t, a.PreviewHash, c.PreviewHash)
}
```

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/execution -count=1`

Expected: PASS.

```bash
git add internal/execution && git commit -m "feat: require stateless previews for risky SQL"
```

## Task 5: Bounded execution and compact query results

**Files:**
- Create: `internal/execution/executor.go`
- Create: `internal/execution/rows.go`
- Create: `internal/execution/executor_test.go`

- [ ] **Step 1: Write failing executor tests with a fake SQL driver**

```go
type testSource struct {
    db   *sql.DB
    caps database.Capability
}

func (s testSource) ID() string { return "analytics" }
func (s testSource) Engine() string { return "clickhouse" }
func (s testSource) DB() *sql.DB { return s.db }
func (s testSource) Dialect() database.Dialect { return database.ClickHouseDialect{} }
func (s testSource) Capabilities() database.Capability { return s.caps }
func (s testSource) Close() error { return nil }
func mysqlTestSource(db *sql.DB) testSource { return testSource{db: db, caps: database.Capability{AtomicBatches: true}} }

func TestQueryStopsAfterLimitAndMarksResultTruncated(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()
    mock.ExpectQuery("SELECT id FROM orders").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2).AddRow(3))
    result, err := NewExecutor(50*time.Millisecond, 2).Query(context.Background(), db, "SELECT id FROM orders", nil)
    require.NoError(t, err)
    require.Equal(t, 2, len(result.Rows))
    require.True(t, result.Truncated)
    require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteMultipleStatementsReportsNonAtomicSource(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()
    mock.ExpectExec("INSERT INTO a").WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec("INSERT INTO b").WillReturnResult(sqlmock.NewResult(0, 1))
    plan := sqlguard.Plan{Statements: []sqlguard.Statement{{NormalizedSQL: "INSERT INTO a VALUES (1)", Kind: sqlguard.Write}, {NormalizedSQL: "INSERT INTO b VALUES (2)", Kind: sqlguard.Write}}, Risk: sqlguard.LowRisk}
    source := testSource{db: db, caps: database.Capability{AtomicBatches: false}}
    result, err := NewExecutor(time.Second, 100).ExecutePlan(context.Background(), source, plan, nil)
    require.NoError(t, err)
    require.False(t, result.Atomic)
    require.NoError(t, mock.ExpectationsWereMet())
}
```

- [ ] **Step 2: Run the executor tests**

Run: `go test ./internal/execution -run 'TestQueryStops|TestExecuteMultiple' -count=1`

Expected: FAIL because `Executor` does not exist.

- [ ] **Step 3: Implement bounded reads and batch execution**

```go
type QueryResult struct {
    Columns   []string         `json:"columns"`
    Rows      []map[string]any `json:"rows"`
    Truncated bool             `json:"truncated"`
}

type ExecuteResult struct {
    Statements []StatementResult `json:"statements"`
    Atomic     bool              `json:"atomic"`
}

type StatementResult struct {
    Index        int   `json:"index"`
    RowsAffected int64 `json:"rows_affected"`
    LastInsertID int64 `json:"last_insert_id"`
}

type Executor struct { timeout time.Duration; maxRows int }

const (
    DefaultQueryTimeout = 30 * time.Second
    DefaultMaxRows      = 100
)

func NewDefaultExecutor() Executor
func (e Executor) Query(ctx context.Context, db *sql.DB, sql string, args []any) (QueryResult, error)
func (e Executor) ExecutePlan(ctx context.Context, source database.Source, plan sqlguard.Plan, args []any) (ExecuteResult, error)
```

`Query` creates a child timeout context, reads at most `maxRows + 1` rows, closes `Rows` on every path, converts `[]byte` to strings, and sets `Truncated` when it sees the extra row. `ExecutePlan` uses one `sql.Tx` only when the source declares `AtomicBatches` and every statement is a write. Otherwise it executes in order, stops at the first failure, and returns `Atomic: false`; the preview exposes this before confirmation.

- [ ] **Step 4: Add timeout and cleanup coverage**

```go
func TestQueryHonorsCanceledContext(t *testing.T) {
    db, _, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()
    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    _, err = NewExecutor(time.Second, 100).Query(ctx, db, "SELECT 1", nil)
    require.ErrorIs(t, err, context.Canceled)
}

func TestAtomicBatchRollsBackOnSecondStatementFailure(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()
    mock.ExpectBegin()
    mock.ExpectExec("UPDATE a").WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec("UPDATE b").WillReturnError(errors.New("second statement failed"))
    mock.ExpectRollback()
    plan := sqlguard.Plan{Statements: []sqlguard.Statement{{NormalizedSQL: "UPDATE a SET x = 1", Kind: sqlguard.Write}, {NormalizedSQL: "UPDATE b SET x = 2", Kind: sqlguard.Write}}}
    _, err = NewExecutor(time.Second, 100).ExecutePlan(context.Background(), mysqlTestSource(db), plan, nil)
    require.ErrorContains(t, err, "second statement failed")
    require.NoError(t, mock.ExpectationsWereMet())
}

func TestNonAtomicBatchStopsAtFirstFailedStatement(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()
    mock.ExpectExec("UPDATE a").WillReturnError(errors.New("first statement failed"))
    plan := sqlguard.Plan{Statements: []sqlguard.Statement{{NormalizedSQL: "UPDATE a SET x = 1", Kind: sqlguard.Write}, {NormalizedSQL: "UPDATE b SET x = 2", Kind: sqlguard.Write}}}
    _, err = NewExecutor(time.Second, 100).ExecutePlan(context.Background(), testSource{db: db}, plan, nil)
    require.ErrorContains(t, err, "first statement failed")
    require.NoError(t, mock.ExpectationsWereMet())
}
```

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/execution -count=1 && go vet ./internal/execution`

Expected: PASS.

```bash
git add internal/execution && git commit -m "feat: bound query results and execute SQL plans"
```

## Task 6: Safe high-level schema operations

**Files:**
- Create: `internal/operation/model.go`
- Create: `internal/operation/table.go`
- Create: `internal/operation/column.go`
- Create: `internal/operation/index.go`
- Create: `internal/operation/copy.go`
- Create: `internal/operation/operation_test.go`

- [ ] **Step 1: Write failing structured-operation tests**

```go
func TestAddColumnQuotesIdentifiersAndRejectsUnknownType(t *testing.T) {
    request := AddColumnsRequest{Table: "orders", Columns: []ColumnSpec{{Name: "note", Kind: "varchar", Length: 120}}}
    sql, err := MySQLBuilder{}.AddColumns(request)
    require.NoError(t, err)
    require.Equal(t, "ALTER TABLE `orders` ADD COLUMN `note` VARCHAR(120) NULL", sql[0])
    _, err = MySQLBuilder{}.AddColumns(AddColumnsRequest{Table: "orders", Columns: []ColumnSpec{{Name: "x", Kind: "varchar; DROP TABLE orders"}}})
    require.Error(t, err)
}
```

- [ ] **Step 2: Run the operation tests**

Run: `go test ./internal/operation -count=1`

Expected: FAIL because structured requests and builders do not exist.

- [ ] **Step 3: Implement typed operations without raw SQL fragments**

```go
type ColumnSpec struct {
    Name      string `json:"name"`
    Kind      string `json:"kind"`
    Length    int    `json:"length,omitempty"`
    Precision int    `json:"precision,omitempty"`
    Scale     int    `json:"scale,omitempty"`
    Nullable  bool   `json:"nullable"`
}

type CreateTableRequest struct { Table string; Columns []ColumnSpec }
type DropTableRequest struct { Table string }
type AddColumnsRequest struct { Table string; Columns []ColumnSpec }
type DropColumnsRequest struct { Table string; Columns []string }
type ModifyColumnsRequest struct { Table string; Columns []ColumnSpec }
type CreateIndexRequest struct { Table, Index string; Columns []string; Unique bool }
type DropIndexRequest struct { Table, Index string }
type RenameTableRequest struct { From, To string }
type CopyTableRequest struct { Source, Destination string; WithData bool }

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
```

Support exactly these kinds: MySQL `int`, `bigint`, `varchar`, `text`, `decimal`, `boolean`, `date`, `datetime`, `timestamp`; ClickHouse `int64`, `uint64`, `string`, `decimal`, `bool`, `date`, `datetime`. Validate identifier names, lengths, precision, and scale before SQL generation. Do not accept raw `type`, `columns`, `default_value`, `after_column`, or identifier SQL fragments. Return `database.ErrUnsupportedCapability` when an engine cannot express an operation.

- [ ] **Step 4: Cover all retained high-level operations**

```go
func TestDropRenameCopyAndIndexOperationsBecomeHighRiskPlans(t *testing.T) {
    builder := MySQLBuilder{}
    statements, err := builder.RenameTable(RenameTableRequest{From: "orders", To: "archived_orders"})
    require.NoError(t, err)
    plan, err := sqlguard.NewMySQLAnalyzer().Analyze(statements[0])
    require.NoError(t, err)
    require.Equal(t, sqlguard.HighRisk, plan.Risk)
}

func TestClickHouseUnsupportedCopyReturnsCapabilityError(t *testing.T) {
    _, err := ClickHouseBuilder{}.CopyTable(CopyTableRequest{Source: "a", Destination: "b", WithData: true})
    require.ErrorIs(t, err, database.ErrUnsupportedCapability)
}
```

Migration is a dedicated raw-SQL operation whose plan is always high risk. `copy_table_structure` maps to `CopyTableRequest{WithData:false}`. MySQL batch DML may use the executor transaction path; ClickHouse never advertises atomic multi-statement support.

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/operation ./internal/sqlguard ./internal/execution -count=1`

Expected: PASS.

```bash
git add internal/operation && git commit -m "feat: build safe high-level schema operations"
```

## Task 7: Query-first MCP handlers

**Files:**
- Create: `internal/tools/input.go`
- Create: `internal/tools/response.go`
- Create: `internal/tools/sources.go`
- Create: `internal/tools/metadata.go`
- Create: `internal/tools/query.go`
- Create: `internal/tools/execute.go`
- Create: `internal/tools/schema.go`
- Modify: `internal/tools/registry.go`
- Create: `internal/tools/tools_test.go`
- Delete: `internal/tools/transaction.go`
- Delete: `internal/tools/schema/schemas.go`
- Delete: `internal/tools/schema/documentation.go`

- [ ] **Step 1: Write handler tests against the service boundary**

```go
func TestQueryRejectsWriteSQLBeforeOpeningDatabase(t *testing.T) {
    source := fakeSource()
    service := newServiceWithSource(t, "orders", source)
    _, err := service.Query(context.Background(), QueryInput{RequestMeta: RequestMeta{SourceID: "orders"}, SQL: "DELETE FROM orders"})
    require.ErrorIs(t, err, ErrReadOnlySQLRequired)
    require.Zero(t, source.queryCount)
}

func TestDropTableReturnsPreviewThenExecutesMatchingConfirmation(t *testing.T) {
    service := newServiceWithSource(t, "orders", fakeSource())
    first, err := service.DropTable(context.Background(), DropTableInput{RequestMeta: RequestMeta{SourceID: "orders"}, Table: "orders"})
    require.NoError(t, err)
    require.Equal(t, "confirmation_required", first.State)
    second, err := service.DropTable(context.Background(), DropTableInput{RequestMeta: RequestMeta{SourceID: "orders", Confirm: true, PreviewHash: first.Preview.PreviewHash}, Table: "orders"})
    require.NoError(t, err)
    require.Equal(t, "executed", second.State)
}
```

- [ ] **Step 2: Run the handler tests**

Run: `go test ./internal/tools -count=1`

Expected: FAIL because generic source-aware inputs and the service do not exist.

- [ ] **Step 3: Define shared inputs and response envelopes**

```go
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

type Response struct {
    RequestID string                     `json:"request_id"`
    State     string                     `json:"state"`
    Preview   *execution.Preview         `json:"preview,omitempty"`
    Query     *execution.QueryResult     `json:"query,omitempty"`
    Execution *execution.ExecuteResult   `json:"execution,omitempty"`
}

type DropTableInput struct {
    RequestMeta
    Table string `json:"table"`
}

type Service struct {
    registry *database.Registry
    mode     config.Mode
    executor execution.Executor
}

var ErrReadOnlySQLRequired = errors.New("query requires read-only SQL")

func (s *Service) Query(context.Context, QueryInput) (Response, error)
func (s *Service) DropTable(context.Context, DropTableInput) (Response, error)
```

Decode `parameters` with `json.Decoder.UseNumber`, then convert each `json.Number` deterministically to `int64` or `float64` before calling the database driver. Generate a cryptographically random request ID when omitted. Return stable error codes for invalid input, unknown source, unsafe SQL, confirmation required, preview mismatch, unsupported capability, timeout, connection failure, and execution failure.

- [ ] **Step 4: Register the generic source-aware tool names**

Register `list_sources`, `list_tables`, `describe_table`, `query`, `execute_sql`, `create_table`, `drop_table`, `add_columns`, `drop_columns`, `modify_columns`, `create_index`, `drop_index`, `list_indexes`, `rename_table`, `copy_table`, `copy_table_structure`, `migrate`, and `pool_status`. `list_sources` returns only source ID and engine; source IDs are the human-readable selection hint. Every mutating tool uses the same path: build, analyze, preview, compare hash, and execute. Do not register `mysql_begin_transaction`, `mysql_commit_transaction`, or `mysql_rollback_transaction`.

```go
func RegisterAll(server *mcp.Server, svc *Service) error {
    addTool(server, ListSourcesTool(svc))
    addTool(server, QueryTool(svc))
    addTool(server, ExecuteSQLTool(svc))
    for _, tool := range SchemaTools(svc) { addTool(server, tool) }
    return nil
}
```

Generate JSON schemas from the typed input structs in `internal/tools/schema.go`, eliminating the old hand-maintained schema map.

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/tools -count=1 && go vet ./internal/tools`

Expected: PASS.

```bash
git add internal/tools && git commit -m "feat: expose query-first multi-source MCP tools"
```

## Task 8: Recompose the server and remove stateful single-MySQL code

**Files:**
- Modify: `cmd/main.go`
- Create: `cmd/main_test.go`
- Create: `internal/observability/logger.go`
- Create: `internal/observability/logger_test.go`
- Delete: `internal/mysql/pool.go`
- Delete: `internal/mysql/txmanager.go`
- Modify: `Dockerfile`
- Modify: `start.sh`
- Modify: `start.bat`

- [ ] **Step 1: Write a failing composition test**

```go
func TestBuildApplicationRegistersGenericToolsForMultipleSources(t *testing.T) {
    cfg := config.Config{Mode: config.QuickMode, Sources: []config.SourceConfig{
        {Name: "orders", Type: "mysql", DSN: "mysql-fake"},
        {Name: "analytics", Type: "clickhouse", DSN: "clickhouse-fake"},
    }}
    app, err := buildApplication(context.Background(), cfg, fakeOpenRegistry)
    require.NoError(t, err)
    require.Contains(t, app.ToolNames(), "list_sources")
    require.NotContains(t, app.ToolNames(), "mysql_begin_transaction")
}
```

- [ ] **Step 2: Run the composition test**

Run: `go test ./cmd -run TestBuildApplication -count=1`

Expected: FAIL because `buildApplication` and registry wiring do not exist.

- [ ] **Step 3: Replace main wiring with configuration-file startup**

```go
func main() {
    configPath := flag.String("config", "config.yaml", "path to MCP configuration")
    flag.Parse()
    cfg, err := config.Load(*configPath, os.Getenv)
    if err != nil { log.Fatal(err) }
    app, err := buildApplication(context.Background(), cfg, database.OpenRegistry)
    if err != nil { log.Fatal(err) }
    defer app.Close()
    runUntilSignal(app.Server())
}
```

`buildApplication` constructs the registry, execution service, MCP server, and new handlers. Startup output may contain only application name, version, configured source count, and registered tool count. It must not print a DSN, host, database, username, or password. Shutdown calls registry close exactly once.

```go
type Event struct {
    RequestID string
    Tool      string
    SourceID  string
    State     string
    Duration  time.Duration
    ErrorCode string
}

func LogEvent(logger *slog.Logger, event Event)
```

`LogEvent` writes JSON `slog` records for request completion. It must omit SQL text, parameter values, result rows, DSNs, hosts, usernames, and passwords. Add a test that serializes an event containing a deliberately secret-looking source string and verifies the logger emits only the declared event fields. These are runtime diagnostics, not persisted audit records.

- [ ] **Step 4: Correct launchers and image behavior**

Remove the Docker HTTP health check because a stdio MCP server exposes no HTTP endpoint. Copy `config.example.yaml` into the image only as documentation; runtime configuration is mounted. Keep the non-root user and remove database credentials from image `ENV`. Update shell and batch launchers to pass `-config` and remove all obsolete single-MySQL environment setup.

- [ ] **Step 5: Verify and commit**

Run: `go test ./cmd ./internal/... -count=1 && go vet ./...`

Expected: PASS.

```bash
git add -A cmd internal/mysql Dockerfile start.sh start.bat && git commit -m "refactor: run stateless multi-source MCP server"
```

## Task 9: Live MySQL and ClickHouse integration verification

**Files:**
- Create: `docker-compose.integration.yml`
- Create: `internal/integration/doc.go`
- Create: `internal/integration/mysql_test.go`
- Create: `internal/integration/clickhouse_test.go`
- Create: `internal/integration/confirmation_test.go`

- [ ] **Step 1: Write build-tagged end-to-end tests**

```go
//go:build integration

func TestMySQLPreviewDoesNotDropThenConfirmationDoes(t *testing.T) {
    svc := openIntegrationService(t, "orders")
    first, err := svc.DropTable(context.Background(), tools.DropTableInput{RequestMeta: tools.RequestMeta{SourceID: "orders"}, Table: "preview_target"})
    require.NoError(t, err)
    requireTableExists(t, "preview_target")
    _, err = svc.DropTable(context.Background(), tools.DropTableInput{RequestMeta: tools.RequestMeta{SourceID: "orders", Confirm: true, PreviewHash: first.Preview.PreviewHash}, Table: "preview_target"})
    require.NoError(t, err)
    requireTableMissing(t, "preview_target")
}
```

- [ ] **Step 2: Start fixtures and run the test before implementations are complete**

Run: `docker compose -f docker-compose.integration.yml up -d --wait`

Run: `go test -tags=integration ./internal/integration -count=1`

Expected: FAIL until real MySQL and ClickHouse source support is connected.

- [ ] **Step 3: Create the fixed fixture topology**

```yaml
services:
  mysql:
    image: mysql:8.4
    environment:
      MYSQL_DATABASE: mcp_test
      MYSQL_USER: mcp
      MYSQL_PASSWORD: mcp
      MYSQL_ROOT_PASSWORD: root
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
  clickhouse:
    image: clickhouse/clickhouse-server:25.3
    environment:
      CLICKHOUSE_DB: mcp_test
      CLICKHOUSE_USER: mcp
      CLICKHOUSE_PASSWORD: mcp
```

Integration tests must prove source selection, metadata, bounded query results, query-write rejection, MySQL high-risk preview, hash mismatch rejection, ClickHouse capability rejection, context cancellation, and ClickHouse multi-statement `atomic: false` previews.

- [ ] **Step 4: Run passing integration tests and clean fixtures**

Run: `go test -tags=integration ./internal/integration -count=1`

Expected: PASS.

Run: `docker compose -f docker-compose.integration.yml down -v`

Expected: containers and volumes removed.

- [ ] **Step 5: Commit**

```bash
git add docker-compose.integration.yml internal/integration && git commit -m "test: cover MySQL and ClickHouse integration paths"
```

## Task 10: Documentation, audit record, and final verification

**Files:**
- Modify: `README.md`
- Modify: `TOOLS_SCHEMA.md`
- Modify: `zed-config-example.json`
- Create: `docs/security-audit.md`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write a failing example-configuration safety test**

```go
func TestExampleConfigurationContainsNoCredentialValue(t *testing.T) {
    content, err := os.ReadFile("../../config.example.yaml")
    require.NoError(t, err)
    require.NotContains(t, string(content), "password:")
    require.Contains(t, string(content), "${ORDERS_MYSQL_DSN}")
}
```

- [ ] **Step 2: Run the safety test**

Run: `go test ./internal/config -run TestExampleConfiguration -count=1`

Expected: PASS because Task 1 created the example without literal credential fields.

- [ ] **Step 3: Publish the exact migration and security guidance**

README documents `-config`, the YAML example, source IDs, MySQL/ClickHouse support, quick/strict modes, confirmation protocol, resource defaults, and least-privilege database accounts. `TOOLS_SCHEMA.md` documents only the generic tool names and a complete preview/confirm sequence. `zed-config-example.json` passes `args: ["-config", "/path/to/config.yaml"]` and contains no credential.

`docs/security-audit.md` records the following rows and their named verification tests:

```markdown
| Finding | Existing location | Remediation | Verification |
| --- | --- | --- | --- |
| Query tool permits writes | `internal/tools/query.go` | AST read-only guard | `TestQueryRejectsWriteSQLBeforeOpeningDatabase` |
| Identifier SQL injection | table, column, index, copy handlers | validated quoting and typed column specs | `TestMySQLIdentifierRejectsInjectionAndQuotesSafeNames` |
| Destructive work executes immediately | mutation handlers | quick/strict preview policy | `TestMySQLPreviewDoesNotDropThenConfirmationDoes` |
| Unbounded result collection | `internal/tools/query.go` | timeout and max-row executor | `TestQueryStopsAfterLimitAndMarksResultTruncated` |
| Cross-request transaction state | `internal/mysql/txmanager.go` | remove transaction tools | `TestBuildApplicationRegistersGenericToolsForMultipleSources` |
```

The audit also records the intentional residual risk: direct callers can provide `confirm: true`; only the separate gateway project proves a human confirmation, prevents replay, and persists audit history.

- [ ] **Step 4: Run all static and unit quality checks**

Run: `gofmt -w cmd internal`

Run: `go test ./... -count=1`

Run: `go vet ./...`

Expected: formatting completes, tests pass for every package, and vet has no findings.

- [ ] **Step 5: Run live verification, commit, and inspect the tree**

Run: `docker compose -f docker-compose.integration.yml up -d --wait && go test -tags=integration ./internal/integration -count=1 && docker compose -f docker-compose.integration.yml down -v`

Expected: integration tests pass and fixtures are removed.

```bash
git add README.md TOOLS_SCHEMA.md zed-config-example.json config.example.yaml docs/security-audit.md internal/config/config_test.go cmd internal && git commit -m "docs: document secure multi-source MCP usage"
```

Run: `git status --short && go build ./cmd`

Expected: a clean worktree and a successful build.

## Plan Self-Review

| Approved requirement | Implementing tasks |
| --- | --- |
| Simple YAML, multiple MySQL and ClickHouse sources | Tasks 1, 2, and 8 |
| Query-first experience for known tables | Tasks 2, 5, and 7 |
| Retained high-level schema tools | Tasks 6 and 7 |
| Raw SQL with dialect-aware fail-closed analysis | Tasks 3, 4, 5, and 7 |
| Quick and strict modes with stateless two-step preview | Task 4 and Task 7 |
| Multi-statement handling with atomicity transparency | Tasks 3, 4, 5, and 9 |
| No user, authorization, confirmation, or audit persistence in MCP | Tasks 7, 8, and 10 |
| Vulnerability record and reliable verification | Tasks 9 and 10 |

The plan defines every production package before a later task consumes it, names the expected test command for every task, and uses database integration fixtures only under the explicit `integration` build tag. It intentionally leaves identity proof, confirmation replay prevention, RBAC, and persistent audit history to the separate gateway PRD.
