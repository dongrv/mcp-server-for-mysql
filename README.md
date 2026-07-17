# Secure Database MCP Server

This query-first, stateless MCP server connects to multiple configured MySQL and ClickHouse sources. It is designed for teams to inspect known tables and run bounded read-only queries, with dialect-aware safety controls for data and schema changes.

New to local MCP clients? Follow the Chinese beginner guide for complete Claude Desktop and Codex setup with a local binary or Docker: **[Claude Desktop 与 Codex 本地接入指南](docs/getting-started-local-clients.md)**.

## Configuration

Copy `config.example.yaml` to a local `config.yaml`, then provide each DSN through the environment:

```yaml
mode: quick
sources:
  - name: orders
    display_name: 订单库
    description: 用户充值订单、支付状态与退款记录
    aliases: [充值库, 支付订单]
    keywords: [充值, 支付, 退款]
    type: mysql
    dsn: ${ORDERS_DSN}
  - name: logs
    display_name: 日志库
    description: 服务运行日志、错误事件与审计记录
    aliases: [服务日志, 错误日志]
    keywords: [报错, 审计, 故障排查]
    type: mysql
    dsn: ${LOGS_DSN}
  - name: analytics
    display_name: 分析库
    description: 聚合指标、用户行为与经营分析数据
    aliases: [数仓, 经营分析]
    keywords: [指标, 趋势, 用户行为]
    type: clickhouse
    dsn: ${ANALYTICS_DSN}
```

Each name is the unique `source_id` used in MCP calls. The required `display_name` and `description`, plus optional aliases and keywords, help AI clients explain source choices but never replace the exact ID. Every DSN must be an exact environment reference. Unknown YAML fields, duplicate names, invalid source profiles, unsupported source types, and absent or empty DSN values fail startup without displaying the resolved DSN.

Build with `go build -o build/mcp-database ./cmd`, then run `./build/mcp-database -config config.yaml`. On Windows use `start.bat run path\\to\\config.yaml`; on Unix-like systems use `CONFIG_PATH=path/to/config.yaml ./start.sh run`.

## Safety Model

`quick` mode directly executes read-only requests and low-risk single-statement writes. It previews high-impact work: DDL, migrations, `DELETE` or `UPDATE` without `WHERE`, and all multi-statement mutations. `strict` mode previews every non-empty SQL intent, including reads.

A preview contains canonical SQL, risk, atomicity, and `preview_hash`. After a human has reviewed the complete preview, repeat the identical call with `confirm: true` and that hash. A changed request returns `preview_mismatch` and does not execute. The hash binds source, tool, normalized SQL order, parameters, risk, and atomicity.

This process intentionally stores no users, permissions, confirmations, or audit records. It cannot prove a human confirmation or prevent replay by a direct caller. An authenticated gateway must own identity, source authorization, confirmation display, single-use authorization, replay prevention, and persistent auditing. See `docs/prd/2026-07-14-wecom-database-gateway-prd.md`.

`query` permits exactly one read-only statement. It uses a 30-second deadline and returns at most 100 rows, marking additional rows as `truncated`. The SQL guards fail closed for unsupported syntax, transaction/session control, unsafe table functions, and stateful or locking MySQL expressions. Parameters are accepted only for a single statement.

Use separate least-privilege database accounts for each source. Query-only accounts should have only metadata and `SELECT` permissions. Do not expose this stdio MCP process directly to untrusted users or autonomous agents.

## Clients, Docker, and Verification

The primary local-client walkthrough is [Claude Desktop 与 Codex 本地接入指南](docs/getting-started-local-clients.md). It covers binary builds, stdio client configuration, Docker, source selection, confirmation, and troubleshooting. The generic tool surface and confirmation protocol are documented in `TOOLS_SCHEMA.md`.

Zed is not part of that primary walkthrough. `zed-config-example.json` is a separate, credential-free optional example using Zed's current `context_servers` setting; keep DSNs in the Zed process environment.

The Docker image includes only the binary and credential-free configuration example. Mount a real config file and pass DSN environment variables at runtime. The server is stdio-only and has no HTTP health endpoint.

Run `go test ./... -count=1`, `go vet ./...`, and `go build ./cmd` for static validation. For live fixtures: run `docker compose -f docker-compose.integration.yml up -d --wait`; set `RUN_DATABASE_INTEGRATION=1`; run `go test -tags=integration ./internal/integration -count=1`; finally run `docker compose -f docker-compose.integration.yml down -v`.

See `docs/security-audit.md` for findings, verification evidence, and residual risk.
