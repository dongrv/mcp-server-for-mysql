# MySQL 5.7 Describe Table Compatibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make MySQL `describe_table` index discovery compatible with servers both with and without `information_schema.statistics.expression`.

**Architecture:** Keep compatibility inside `MySQLDialect`. Probe the metadata schema for expression support, select a modern or legacy query, and preserve the existing normalized scanner and MCP contract.

**Tech Stack:** Go 1.24, `database/sql`, `go-sql-driver/mysql`, `go-sqlmock`, MCP Go SDK.

---

### Task 1: Add Version-Agnostic MySQL Index Metadata Selection

**Files:**
- Modify: `internal/database/metadata_test.go`
- Modify: `internal/database/mysql.go`

- [ ] **Step 1: Write the failing legacy regression test**

Add `TestMySQLDescribeTableSupportsLegacyIndexMetadata` to
`internal/database/metadata_test.go`. Set expectations for the existing column
query, then the capability query returning `0`, then this legacy index query:

```sql
SELECT index_name, non_unique, index_type, column_name, NULL AS index_expression
FROM information_schema.statistics
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY index_name, seq_in_index
```

Return `PRIMARY(id)` and `idx_email(email)` rows and assert the normalized
`TableDescription` contains both indexes without expressions.

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```powershell
go test ./internal/database -run TestMySQLDescribeTableSupportsLegacyIndexMetadata -count=1 -v
```

Expected: FAIL because the current implementation issues the modern index query
instead of the new capability query and legacy query.

- [ ] **Step 3: Update the modern regression fixture**

Before its existing modern index expectation,
`TestMySQLDescribeTableNormalizesColumnsAndFunctionalIndexes` must expect:

```sql
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = 'information_schema'
  AND table_name = 'STATISTICS'
  AND column_name = 'EXPRESSION'
```

Return one row with count `1`, preserving its current functional-index
assertion.

- [ ] **Step 4: Implement minimal capability-based query selection**

In `internal/database/mysql.go`, add query constants for the capability probe,
modern index metadata, and legacy index metadata. Add this focused helper:

```go
func mysqlSupportsIndexExpressions(ctx context.Context, db *sql.DB) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx, mysqlIndexExpressionSupportQuery).Scan(&count); err != nil {
		return false, fmt.Errorf("detect MySQL index expression support: %w", err)
	}
	return count > 0, nil
}
```

At the start of `mysqlIndexes`, call the helper and select
`mysqlIndexesModernQuery` or `mysqlIndexesLegacyQuery`. Keep the current row
scan and normalization loop unchanged.

- [ ] **Step 5: Run focused database tests and verify GREEN**

Run:

```powershell
go test ./internal/database -run 'TestMySQLDescribeTable' -count=1 -v
```

Expected: both the modern functional-index test and legacy compatibility test
PASS.

- [ ] **Step 6: Run repository verification**

Run:

```powershell
gofmt -w internal/database/mysql.go internal/database/metadata_test.go
go test ./...
go vet ./...
go build ./cmd
```

Expected: every command exits `0`, with no test failures or vet diagnostics.

- [ ] **Step 7: Commit the implementation**

```powershell
git add internal/database/mysql.go internal/database/metadata_test.go
git commit -m "fix: support MySQL 5.7 table metadata"
```

### Task 2: Read-Only MySQL 5.7 Acceptance Check

**Files:**
- No persistent file changes.

- [ ] **Step 1: Build an isolated verification binary**

Build the current worktree to a temporary path outside the repository. Do not
replace the binary used by Zed.

- [ ] **Step 2: Invoke the configured source securely**

Resolve only the environment variable names referenced by the external config,
read their values from the local Zed configuration without printing them, and
start the temporary MCP binary. Call `describe_table` with:

```json
{"source_id":"mcp_server","table":"users"}
```

The call is metadata-only. Do not issue any write SQL.

- [ ] **Step 3: Verify and clean up**

Expected response state: `executed`, with three columns and index metadata.
Delete all temporary scripts and binaries, verify no diagnostic process remains,
and confirm `git status --short` contains only the intended committed changes.
