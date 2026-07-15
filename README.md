# Secure Database MCP Server

This query-first, stateless MCP server connects to multiple configured MySQL and ClickHouse sources. It is designed for teams to inspect known tables and run bounded read-only queries, with dialect-aware safety controls for data and schema changes.

## Configuration

Copy `config.example.yaml` to a local `config.yaml`, then provide each DSN through the environment:

```yaml
mode: quick
sources:
  - name: orders
    type: mysql
    dsn: ${ORDERS_DSN}
  - name: analytics
    type: clickhouse
    dsn: ${ANALYTICS_DSN}
```

Each name is the unique `source_id` used in MCP calls. Every DSN must be an exact environment reference. Unknown YAML fields, duplicate names, unsupported source types, and absent or empty DSN values fail startup without displaying the resolved DSN.

Build with `go build -o build/mcp-database ./cmd`, then run `./build/mcp-database -config config.yaml`. On Windows use `start.bat run path\\to\\config.yaml`; on Unix-like systems use `CONFIG_PATH=path/to/config.yaml ./start.sh run`.

## Safety Model

`quick` mode directly executes read-only requests and low-risk single-statement writes. It previews high-impact work: DDL, migrations, `DELETE` or `UPDATE` without `WHERE`, and all multi-statement mutations. `strict` mode previews every non-empty SQL intent, including reads.

A preview contains canonical SQL, risk, atomicity, and `preview_hash`. After a human has reviewed the complete preview, repeat the identical call with `confirm: true` and that hash. A changed request returns `preview_mismatch` and does not execute. The hash binds source, tool, normalized SQL order, parameters, risk, and atomicity.

This process intentionally stores no users, permissions, confirmations, or audit records. It cannot prove a human confirmation or prevent replay by a direct caller. An authenticated gateway must own identity, source authorization, confirmation display, single-use authorization, replay prevention, and persistent auditing. See `docs/prd/2026-07-14-wecom-database-gateway-prd.md`.

`query` permits exactly one read-only statement. It uses a 30-second deadline and returns at most 100 rows, marking additional rows as `truncated`. The SQL guards fail closed for unsupported syntax, transaction/session control, unsafe table functions, and stateful or locking MySQL expressions. Parameters are accepted only for a single statement.

Use separate least-privilege database accounts for each source. Query-only accounts should have only metadata and `SELECT` permissions. Do not expose this stdio MCP process directly to untrusted users or autonomous agents.

## Clients, Docker, and Verification

`zed-config-example.json` demonstrates a client configuration that passes only `-config /path/to/config.yaml`; keep DSNs in the client or supervisor environment. The generic tool surface and confirmation protocol are documented in `TOOLS_SCHEMA.md`.

The Docker image includes only the binary and credential-free configuration example. Mount a real config file and pass DSN environment variables at runtime. The server is stdio-only and has no HTTP health endpoint.

Run `go test ./... -count=1`, `go vet ./...`, and `go build ./cmd` for static validation. For live fixtures: run `docker compose -f docker-compose.integration.yml up -d --wait`; set `RUN_DATABASE_INTEGRATION=1`; run `go test -tags=integration ./internal/integration -count=1`; finally run `docker compose -f docker-compose.integration.yml down -v`.

See `docs/security-audit.md` for findings, verification evidence, and residual risk.
