# First-Release Security Audit

| Finding | Existing location | Remediation | Verification |
| --- | --- | --- | --- |
| Query tool permits writes | Retired query handlers | AST read-only guard before database access | `TestQueryRejectsWriteSQLBeforeOpeningDatabase` |
| Identifier SQL injection | Retired table, column, index, and copy handlers | Validated dialect quoting and typed columns | `TestMySQLIdentifierRejectsInjectionAndQuotesSafeNames` |
| Destructive work executes immediately | Retired mutation handlers | Quick/strict preview policy binding exact intent | `TestDropTableReturnsPreviewThenExecutesMatchingConfirmation`; `TestMySQLPreviewDoesNotDropThenConfirmationDoes` |
| Unbounded result collection | Retired query handler | 30-second deadline and 100-row truncating collector | `TestQueryStopsAfterLimitAndMarksResultTruncated` |
| Cross-request transaction state | `internal/mysql/txmanager.go` (removed) | Stateless generic tool surface | `TestBuildApplicationRegistersGenericToolsForMultipleSources` |
| Credentials in examples or local env files | Client configuration and `mcp-database.env` | Keep committed examples credential-free, ignore only the documented root env file, and document restrictive local permissions | `TestExampleConfigurationContainsNoCredentialValue`; `git check-ignore mcp-database.env`; guide permission checklist review |
| Stale or overlapping source metadata | `display_name`, `description`, aliases, and keywords returned by `list_sources` | Treat metadata only as discovery hints; when multiple sources are plausible, the AI client must show each candidate's display name and description, ask the user to choose, and never guess or query all candidates | `TestLocalClientGuideContract` exact unique/ambiguous/no-guess rules and synthetic safe/unsafe guidance fixtures; `TestListSourcesExposesBusinessProfilesWithoutConnectionData` |
| Display metadata mistaken for an execution identity | Source discovery and confirmation previews | Every source-specific tool requires the exact `source_id`; display names, descriptions, aliases, and keywords are never selectors | `TestListSourcesExposesBusinessProfilesWithoutConnectionData`; `TestPreviewHashDoesNotBindSourceDisplayName`; `TestBuildPreviewHashBindsSourceToolRiskAtomicStatementOrderAndParameters` |

The server accepts only configured source IDs and exact environment-reference DSNs, parses SQL by dialect, and rejects unsupported, session, transaction, external-table, locking, stateful, and comment-only input. Runtime diagnostics omit SQL, parameters, results, DSNs, hostnames, and credentials.

Source metadata has a separate trust boundary from execution identity. Profiles can become stale and their business terms can overlap, so a client must not infer a unique target when several candidates are plausible. It must show candidate `display_name` and `description` values and wait for an explicit user choice. Execution is controlled only by the exact `source_id` returned as `id` from `list_sources`; the display name is response-only context and does not bind or alter `preview_hash`.

Residual risk is intentional: a matching hash proves only that an execution request matches its preview. A direct caller can replay a previously obtained hash, and the MCP process cannot prove a human approved it. An upstream gateway must authenticate users, authorize source/action access, display previews, consume approvals once, prevent replay, and persist audit records. See `prd/2026-07-14-wecom-database-gateway-prd.md`.

Integration uses `docker-compose.integration.yml` and `RUN_DATABASE_INTEGRATION=1 go test -tags=integration ./internal/integration -count=1`. Docker was initially blocked in this environment because its configured registry mirror could not resolve through DNS. Docker is now reachable and fixture startup succeeds; a future mirror-DNS image-pull failure is infrastructure failure, not an MCP test result.
