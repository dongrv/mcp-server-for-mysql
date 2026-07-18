# MySQL 5.7 Describe Table Compatibility Design

## Context

`describe_table` currently reads `information_schema.statistics.expression`
unconditionally. The production-like `mcp_server` source runs MySQL 5.7.44,
where that column does not exist, so index discovery fails with MySQL error
1054 even though the table and its column metadata are readable.

## Scope

This change makes MySQL index metadata discovery work on both legacy servers
without `STATISTICS.EXPRESSION` and modern servers that expose functional index
expressions. It does not change the MCP tool schema, error envelope, source
configuration, ClickHouse behavior, or any data-changing path.

## Design

The MySQL dialect will detect support by querying
`information_schema.columns` for the `EXPRESSION` field on the `STATISTICS`
metadata table. It will then select one of two internal index queries:

- supported: select the real `expression` field;
- unsupported: select `NULL AS index_expression` in the same result position.

Both variants feed the existing scanner, so normalized `Index` output and the
public tool contract remain unchanged. Capability detection is based on the
server schema rather than parsing `VERSION()`, which also accommodates MySQL
8.0.12 and compatible engines whose version strings do not follow Oracle MySQL
formatting.

If capability detection itself fails, `describe_table` returns the existing
`execution_failure` response through the current error mapping. There is no
fallback that could hide permission or connection failures.

## Verification

Unit coverage must prove both paths:

- modern metadata retains functional index expressions;
- legacy metadata uses the compatible query and returns ordinary indexes.

The legacy test must be written and observed failing before production code is
changed. After implementation, run the focused database tests, all Go tests,
`go vet`, and a build. Finally, run `describe_table` against the configured
MySQL 5.7.44 source through a newly built binary using read-only metadata calls.

## Acceptance Criteria

- `describe_table(source_id=mcp_server, table=users)` succeeds on MySQL 5.7.44.
- MySQL 8.x functional index expressions remain present in normalized metadata.
- No SQL mutation is issued during implementation or verification.
- No DSN, credential, SQL parameter, or database content is logged or committed.
- The MCP response and error protocols remain unchanged.
