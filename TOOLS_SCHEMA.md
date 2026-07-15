# MCP Tool Reference

All source-specific tools take `source_id`, and may take `confirm`, `preview_hash`, and `request_id`. There are no legacy `mysql_*` tools or cross-request transaction tools.

| Tool | Required fields | Purpose |
| --- | --- | --- |
| `list_sources` | none | Return configured source IDs and engine names. |
| `list_tables` | `source_id` | List source tables. |
| `describe_table`, `list_indexes` | `source_id`, `table` | Read normalized metadata. |
| `query` | `source_id`, `sql` | Run one read-only statement. |
| `execute_sql` | `source_id`, `sql` | Execute non-read SQL after policy checks. |
| `migrate` | `source_id`, `sql` | Execute a confirmed migration. |
| `create_table`, `add_columns`, `modify_columns` | `source_id`, `table`, `columns` | Build typed schema SQL. |
| `drop_table` | `source_id`, `table` | Drop a confirmed table. |
| `drop_columns` | `source_id`, `table`, `columns` | Drop confirmed columns. |
| `create_index`, `drop_index` | `source_id`, `table`, `index` | MySQL-only index operations. |
| `rename_table` | `source_id`, `from`, `to` | Rename a confirmed table. |
| `copy_table`, `copy_table_structure` | `source_id`, `source_table`, `destination_table` | MySQL-only copy operations. |
| `pool_status` | `source_id` | Return connection-pool statistics. |

`query` and `execute_sql` accept optional `parameters`, but parameters are valid only for a single statement. Structured `columns` use typed values such as `{"name":"status","kind":"varchar","length":32,"nullable":false}`. MySQL supports `int`, `bigint`, `varchar`, `text`, `decimal`, `boolean`, `date`, `datetime`, and `timestamp`; ClickHouse supports `int64`, `uint64`, `string`, `decimal`, `bool`, `date`, and `datetime`. Raw type fragments, expressions, defaults, and placement clauses are intentionally excluded.

## Confirmation

1. Call a tool without `confirm`.
2. When it returns `confirmation_required`, show complete `preview.sql`, `preview.risk`, and `preview.atomic` to a human.
3. Repeat the exact call with `confirm: true` and `preview_hash` from that response.
4. A changed request returns `preview_mismatch`; request and approve a new preview.

`quick` previews high-risk and all multi-statement mutations. `strict` previews every non-empty SQL intent. The hash binds source, tool, normalized SQL, parameter values, risk, and atomicity. It does not authenticate a human, consume approvals once, or prevent replay; an upstream gateway must provide those controls.

Responses use `executed`, `confirmation_required`, `preview_mismatch`, or `error`. Stable errors include `invalid_input`, `unknown_source`, `unsafe_sql`, `unsupported_capability`, `timeout`, `connection_failure`, and `execution_failure`. Logs and errors omit DSNs, credentials, SQL parameters, and result rows.
