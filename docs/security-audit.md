# First-Release Security Audit

| Finding | Existing location | Remediation | Verification |
| --- | --- | --- | --- |
| Query tool permits writes | Retired query handlers | AST read-only guard before database access | `TestQueryRejectsWriteSQLBeforeOpeningDatabase` |
| Identifier SQL injection | Retired table, column, index, and copy handlers | Validated dialect quoting and typed columns | `TestMySQLIdentifierRejectsInjectionAndQuotesSafeNames` |
| Destructive work executes immediately | Retired mutation handlers | Quick/strict preview policy binding exact intent | `TestDropTableReturnsPreviewThenExecutesMatchingConfirmation`; `TestMySQLPreviewDoesNotDropThenConfirmationDoes` |
| Unbounded result collection | Retired query handler | 30-second deadline and 100-row truncating collector | `TestQueryStopsAfterLimitAndMarksResultTruncated` |
| Cross-request transaction state | `internal/mysql/txmanager.go` (removed) | Stateless generic tool surface | `TestBuildApplicationRegistersGenericToolsForMultipleSources` |
| Credentials in examples | Retired client configuration | Credential-free YAML and Zed examples | `TestExampleConfigurationContainsNoCredentialValue` |

The server accepts only configured source IDs and exact environment-reference DSNs, parses SQL by dialect, and rejects unsupported, session, transaction, external-table, locking, stateful, and comment-only input. Runtime diagnostics omit SQL, parameters, results, DSNs, hostnames, and credentials.

Residual risk is intentional: a matching hash proves only that an execution request matches its preview. A direct caller can replay a previously obtained hash, and the MCP process cannot prove a human approved it. An upstream gateway must authenticate users, authorize source/action access, display previews, consume approvals once, prevent replay, and persist audit records. See `prd/2026-07-14-wecom-database-gateway-prd.md`.

Integration uses `docker-compose.integration.yml` and `RUN_DATABASE_INTEGRATION=1 go test -tags=integration ./internal/integration -count=1`. Docker was initially blocked in this environment because its configured registry mirror could not resolve through DNS. Docker is now reachable and fixture startup succeeds; a future mirror-DNS image-pull failure is infrastructure failure, not an MCP test result.
