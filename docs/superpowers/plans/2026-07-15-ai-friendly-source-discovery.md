# AI-Friendly Source Discovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add bounded business metadata to every configured database source, expose it safely to MCP clients, show the selected source in confirmation previews, and provide a beginner guide for Claude Desktop and Codex using local binaries or Docker.

**Architecture:** Configuration owns validation and normalization of the business profile. The database registry carries an immutable profile with each source, while the MCP service exposes the profile without connection data. Natural-language matching remains the AI client's responsibility; the server continues to accept only exact `source_id` values and stores no selection state.

**Tech Stack:** Go 1.24, YAML v3, Model Context Protocol Go SDK, Go standard tests, sqlmock, Docker Compose, Markdown/JSON client examples.

---

## File map

- `internal/config/config.go`: YAML fields, normalization, bounded validation.
- `internal/config/config_test.go`: profile validation and secret-redaction tests.
- `internal/database/types.go`: immutable public `SourceProfile` and `Source.Profile()` contract.
- `internal/database/registry.go`: transfer normalized profiles into opened sources.
- `internal/database/registry_test.go`: profile preservation and defensive-copy tests.
- `internal/execution/model.go`: response-only source reference in previews.
- `internal/execution/policy_test.go`: proof that display metadata does not affect confirmation hashes.
- `internal/tools/service.go`: `list_sources` profile response and preview source reference.
- `internal/tools/tools_test.go`: deterministic discovery output, no-secret checks, preview display tests.
- `internal/integration/integration_test.go`: required profile fixture configuration.
- `internal/integration/mysql_test.go`: live source-discovery assertions.
- `cmd/main_test.go`, `internal/execution/executor_test.go`: update fake `database.Source` implementations.
- `config.example.yaml`, `README.md`, `TOOLS_SCHEMA.md`: current public configuration and tool contract.
- `docs/getting-started-local-clients.md`: beginner Claude Desktop and Codex guide.
- `docs/security-audit.md`: source-selection threat model and residual risks.

## Task 1: Validate and normalize AI-friendly source profiles

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `config.example.yaml`

- [ ] **Step 1: Write failing happy-path and normalization tests**

Add a configuration with two MySQL sources and one ClickHouse source. Assert that `display_name`, `description`, aliases, and keywords are trimmed and retained:

```go
func TestLoadNormalizesSourceProfilesForSameEngineInstances(t *testing.T) {
	path := writeConfig(t, `
sources:
  - name: orders
    display_name: " 订单库 "
    description: " 用户充值订单、支付状态和退款记录 "
    aliases: [" 充值库 ", "支付订单"]
    keywords: [" 用户充值 ", "退款"]
    type: mysql
    dsn: ${ORDERS_DSN}
  - name: logs
    display_name: 日志库
    description: 服务运行和错误日志
    aliases: [服务日志]
    keywords: [报错]
    type: mysql
    dsn: ${LOGS_DSN}
  - name: analytics
    display_name: 分析库
    description: 聚合分析数据
    type: clickhouse
    dsn: ${ANALYTICS_DSN}
`)

	cfg, err := Load(path, envLookup(map[string]string{
		"ORDERS_DSN": "orders-secret", "LOGS_DSN": "logs-secret", "ANALYTICS_DSN": "analytics-secret",
	}))
	if err != nil { t.Fatalf("Load: %v", err) }
	if got := cfg.Sources[0].DisplayName; got != "订单库" { t.Fatalf("display name = %q", got) }
	if got := cfg.Sources[0].Aliases[0]; got != "充值库" { t.Fatalf("alias = %q", got) }
}
```

- [ ] **Step 2: Run the focused test and observe RED**

Run: `go test ./internal/config -run TestLoadNormalizesSourceProfilesForSameEngineInstances -count=1`

Expected: FAIL because the profile fields do not exist.

- [ ] **Step 3: Add the configuration fields and concrete limits**

Implement:

```go
const (
	MaxDisplayNameRunes = 80
	MaxDescriptionRunes = 500
	MaxProfileItemRunes = 80
	MaxAliases = 20
	MaxKeywords = 30
)

type SourceConfig struct {
	Name        string   `yaml:"name"`
	DisplayName string   `yaml:"display_name"`
	Description string   `yaml:"description"`
	Aliases     []string `yaml:"aliases,omitempty"`
	Keywords    []string `yaml:"keywords,omitempty"`
	Type        string   `yaml:"type"`
	DSN         string   `yaml:"dsn"`
}
```

Add focused helpers that trim strings, validate UTF-8/rune counts, reject empty required fields, reject empty list entries, and reject exact case-sensitive duplicates after trimming. Validate the profile before resolving the DSN so failures never need connection data.

- [ ] **Step 4: Add table-driven rejection tests**

Cover missing `display_name`, missing `description`, invalid UTF-8 where constructible, 81-rune display names, 501-rune descriptions, 81-rune items, 21 aliases, 31 keywords, empty list entries, and trimmed duplicates. Each failure must mention only the source name and field; assert that a resolved secret DSN is absent from the error.

- [ ] **Step 5: Update all existing config fixtures and comparisons**

Every valid YAML fixture must include required metadata. Replace direct `SourceConfig` struct inequality with `reflect.DeepEqual` because slices make the struct non-comparable. Update `config.example.yaml` with `orders`, `logs`, and `analytics`, using different DSN environment names and concise Chinese business profiles.

- [ ] **Step 6: Verify and commit**

Run:

```bash
go test ./internal/config -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

Expected: PASS.

Commit:

```bash
git add internal/config config.example.yaml
git commit -m "feat: validate AI-friendly source profiles"
```

## Task 2: Carry immutable profiles through the database registry

**Files:**
- Modify: `internal/database/types.go`
- Modify: `internal/database/registry.go`
- Modify: `internal/database/registry_test.go`
- Modify: `cmd/main_test.go`
- Modify: `internal/execution/executor_test.go`
- Modify: `internal/tools/tools_test.go`

- [ ] **Step 1: Write a failing registry profile test**

Use `openRegistry` with a factory that records `config.SourceConfig`, then return a source carrying the equivalent profile. Assert `registry.Sources()` preserves deterministic ID ordering and exact profile values. Mutate an alias slice returned by `Profile()` and assert a second call is unchanged.

- [ ] **Step 2: Run the focused test and observe RED**

Run: `go test ./internal/database -run 'TestRegistrySourcesPreserveProfiles|TestSourceProfileIsDefensivelyCopied' -count=1`

Expected: FAIL because `SourceProfile` and `Source.Profile()` do not exist.

- [ ] **Step 3: Define one shared profile contract**

Add:

```go
type SourceProfile struct {
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
	Keywords    []string `json:"keywords"`
}

type Source interface {
	ID() string
	Engine() string
	Profile() SourceProfile
	DB() *sql.DB
	Dialect() Dialect
	Capabilities() Capability
	Close() error
}
```

Implement an unexported clone helper for both slices. `sqlSource.Profile()` must return a defensive copy. `openSource` maps the already-normalized configuration profile into `sqlSource` without retaining the DSN in the profile.

- [ ] **Step 4: Update every fake source implementation**

Add `Profile() SourceProfile` to fakes in database, executor, tools, and main tests. Do not weaken the interface or add optional type assertions. Give tools-facing fakes meaningful profiles so later discovery tests can use them.

- [ ] **Step 5: Verify and commit**

Run:

```bash
go test ./internal/database ./internal/execution ./internal/tools ./cmd -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

Expected: PASS.

Commit:

```bash
git add internal/database internal/execution/executor_test.go internal/tools/tools_test.go cmd/main_test.go
git commit -m "feat: retain database source profiles"
```

## Task 3: Expose profiles and selected sources through MCP

**Files:**
- Modify: `internal/execution/model.go`
- Modify: `internal/execution/policy_test.go`
- Modify: `internal/tools/service.go`
- Modify: `internal/tools/tools_test.go`
- Modify: `TOOLS_SCHEMA.md`

- [ ] **Step 1: Write failing `list_sources` response tests**

Replace the ID/engine-only assertion with an exact assertion for ID, engine, display name, description, aliases, and keywords. Marshal the result and assert it contains none of `dsn`, `password`, `host`, `username`, or a configured secret value.

- [ ] **Step 2: Write failing preview source-reference tests**

For `DropTable`, assert the preview contains:

```go
Source: &execution.SourceReference{ID: "orders", DisplayName: "订单库"}
```

Build the preview once with display name `订单库`, then change only the fake source's display name to `充值订单库`; repeat the same authorization intent and assert the preview hash is unchanged. The stable ID, not the label, remains security-bound.

- [ ] **Step 3: Run focused tests and observe RED**

Run: `go test ./internal/tools ./internal/execution -run 'TestListSources|TestPreview' -count=1`

Expected: FAIL because public profile fields and `SourceReference` do not exist.

- [ ] **Step 4: Add response-only source references**

Add to `internal/execution/model.go`:

```go
type SourceReference struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Preview struct {
	State       string           `json:"state"`
	Source      *SourceReference `json:"source,omitempty"`
	SQL         []string         `json:"sql"`
	Risk        string           `json:"risk"`
	Atomic      bool             `json:"atomic"`
	PreviewHash string           `json:"preview_hash"`
}
```

Do not add `DisplayName` to `Intent` or `previewEnvelope`. In `Service.authorize`, attach `SourceReference` only after `execution.Authorize` returns its hash-bearing preview.

- [ ] **Step 5: Expand `SourceInfo` from the shared profile**

`SourceInfo` keeps `id` and `engine` and adds all four profile fields. Copy slices when building the response. Keep `list_sources` ordering inherited from `Registry.Sources()`.

- [ ] **Step 6: Update the public tool contract**

Update `TOOLS_SCHEMA.md` examples and field descriptions for `list_sources` and preview responses. State explicitly that clients must use `id` as `source_id`; display names and aliases are discovery metadata only. Document the mandatory user-choice behavior for ambiguous candidates.

- [ ] **Step 7: Verify and commit**

Run:

```bash
go test ./internal/execution ./internal/tools -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

Expected: PASS.

Commit:

```bash
git add internal/execution internal/tools TOOLS_SCHEMA.md
git commit -m "feat: expose source profiles to MCP clients"
```

## Task 4: Add the beginner Claude Desktop and Codex guide

**Files:**
- Create: `docs/getting-started-local-clients.md`
- Modify: `README.md`
- Modify: `zed-config-example.json`
- Modify: `docs/security-audit.md`

- [ ] **Step 1: Write a documentation contract test**

Create a focused Go test in `internal/config/config_test.go` that reads the guide and asserts it contains these stable section anchors: `Claude Desktop`, `Codex`, `本地二进制`, `Docker`, `list_sources`, `多候选`, `preview_hash`, and `source_id`. Assert it does not contain legacy `MYSQL_HOST`, `mysql_begin_transaction`, or ChatGPT web setup.

- [ ] **Step 2: Run the documentation test and observe RED**

Run: `go test ./internal/config -run TestLocalClientGuideContract -count=1`

Expected: FAIL because the guide does not exist.

- [ ] **Step 3: Write the linear beginner guide**

The guide must include copyable Windows and Unix examples for:

- building `mcp-database`;
- setting DSN environment variables;
- validating `config.yaml` shape;
- Claude Desktop `mcpServers` JSON for a local binary;
- `codex mcp add secure-database --env KEY=VALUE -- <binary> -config <config>` and equivalent `~/.codex/config.toml` shape;
- Claude Desktop and Codex Docker stdio launch using `docker run --rm -i`, a read-only config mount, and `--env-file`;
- verifying with `list_sources` and `list_tables`;
- a unique natural-language match;
- an ambiguous match where the assistant lists candidates and waits for the user's choice;
- a read query, strict-mode confirmation, high-risk confirmation, and mismatch recovery;
- troubleshooting process paths, GUI environment inheritance, YAML indentation, env files, Docker mounts, source ambiguity, and connection failures.

Do not put real credentials in examples. Clearly label platform-specific path escaping. Cite the current official OpenAI Codex MCP CLI/help source and Anthropic local MCP documentation links used to verify syntax.

- [ ] **Step 4: Update entry points and security guidance**

Link the guide near the top of `README.md`. Update its configuration example with required profiles. Convert `zed-config-example.json` into a generic credential-free local MCP example or rename its explanatory text so it does not imply Zed is part of the supported guide. Add the natural-language selection threat model to `docs/security-audit.md`: metadata can be stale or overlapping, ambiguity must lead to user choice, and database authorization still depends on exact `source_id`.

- [ ] **Step 5: Verify and commit**

Run:

```bash
go test ./internal/config -run TestLocalClientGuideContract -count=1
rg -n "MYSQL_HOST|mysql_begin_transaction|ChatGPT web" README.md TOOLS_SCHEMA.md docs/getting-started-local-clients.md zed-config-example.json
git diff --check
```

Expected: test PASS; `rg` returns no legacy setup instruction (a clearly worded exclusion sentence may mention ChatGPT web only if the test is adjusted to accept that exact sentence).

Commit:

```bash
git add docs/getting-started-local-clients.md README.md zed-config-example.json docs/security-audit.md internal/config/config_test.go
git commit -m "docs: add local AI client onboarding guide"
```

## Task 5: Update live fixtures and complete security acceptance

**Files:**
- Modify: `internal/integration/integration_test.go`
- Modify: `internal/integration/mysql_test.go`
- Modify: `docker-compose.integration.yml` only if a second MySQL container is required for physical-instance coverage
- Modify: `docs/security-audit.md`

- [ ] **Step 1: Add a failing integration discovery assertion**

Configure `orders` and `logs` as two physically separate MySQL containers with different databases, ports, DSNs, fixture rows, and business profiles, plus `analytics` as ClickHouse. Assert live `list_sources` returns all three profiles and querying each exact source ID reaches only its intended fixture.

- [ ] **Step 2: Run the integration test and observe RED**

Run:

```powershell
$env:RUN_DATABASE_INTEGRATION='1'
go test -tags=integration ./internal/integration -run TestMultipleSameEngineSourceProfiles -count=1
```

Expected: FAIL until the integration fixture and assertions are implemented.

- [ ] **Step 3: Implement the minimal fixture changes**

Update generated integration YAML with required profile fields. Add a second MySQL container with a distinct host port, database, health check, volume, fixture marker, and DSN environment variable. Never include production credentials or reuse a production DSN.

- [ ] **Step 4: Run complete verification**

Run:

```bash
go test ./... -count=1
go vet ./...
go build ./cmd
docker compose -f docker-compose.integration.yml config
docker compose -f docker-compose.integration.yml up -d --wait
```

Then on PowerShell:

```powershell
$env:RUN_DATABASE_INTEGRATION='1'
go test -tags=integration ./internal/integration -count=1
```

Finally:

```bash
docker compose -f docker-compose.integration.yml down -v
git diff --check
```

Expected: all commands PASS; containers are removed after testing.

- [ ] **Step 5: Audit requirements and commit**

Update `docs/security-audit.md` with verification evidence for profile redaction, exact-ID execution, display-name non-binding, ambiguity guidance, same-engine multi-instance selection, and residual risk from stale business descriptions.

Commit:

```bash
git add internal/integration docker-compose.integration.yml docs/security-audit.md
git commit -m "test: verify AI-friendly multi-source discovery"
```

## Review and acceptance protocol

For each task:

1. Dispatch a fresh Terra-high subagent with only that task and the approved specification.
2. Require TDD evidence and no unrelated changes.
3. Run an independent specification-compliance review.
4. Run an independent code-quality and security review.
5. Return all findings to a fresh Terra-high fix subagent; do not accept with unresolved P1/P2 findings.
6. The primary agent performs read-only diff inspection and verification orchestration only.

Final acceptance requires every task commit, a clean worktree, full unit/vet/build success, live MySQL/ClickHouse integration success, credential-free examples, and no regression of stateless confirmation behavior.
