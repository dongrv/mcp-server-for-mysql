# AI-friendly source discovery and local client onboarding

Date: 2026-07-15
Status: Approved design, pending implementation plan

## Purpose

Users describe business intent in natural language, such as "query a user's recharge orders", rather than naming a configured source ID such as `orders`. The MCP server must expose enough concise business metadata for Claude Desktop and Codex to select the correct configured database instance without adding server-side session state or opaque semantic routing.

This change also adds a beginner-oriented local client guide covering Claude Desktop and Codex with both local binary and Docker execution.

## Scope

### Included

- Extend each configured source with an AI-friendly business profile.
- Return that profile from `list_sources`.
- Preserve `name` as the unique, stable machine identifier used as `source_id`.
- Document the required AI selection behavior, including mandatory user choice when multiple sources match.
- Display both source ID and business display name in operation previews where a preview is returned.
- Add a beginner guide for Claude Desktop and Codex.
- Cover local binary execution first and Docker as an alternative.
- Add configuration, service, MCP schema, integration, and documentation tests.

### Excluded

- ChatGPT web integration or remote MCP gateways.
- Server-side LLM calls, embeddings, vector indexes, or fuzzy semantic routing.
- Persistent source selections or conversation state in the MCP server.
- Automatic querying of multiple candidate sources.
- User identity, RBAC, or gateway authorization changes.

## Configuration model

Each source supports these fields:

```yaml
mode: quick
sources:
  - name: orders
    display_name: 订单库
    description: 存储用户充值订单、支付状态、退款记录和渠道流水
    aliases:
      - 充值库
      - 支付订单
    keywords:
      - 用户充值
      - 支付
      - 退款
      - 订单状态
    type: mysql
    dsn: ${ORDERS_DSN}
```

Field semantics:

- `name`: required, unique machine identifier. It remains the only value accepted as `source_id`.
- `display_name`: required short business-facing name.
- `description`: required concise description of the source's business boundary and typical data.
- `aliases`: optional unique, non-empty alternative names used by the team.
- `keywords`: optional unique, non-empty business intent phrases.
- `type` and `dsn`: retain their current meanings.

Validation rules:

- Trim surrounding whitespace before validation and storage.
- Reject empty required values.
- Reject empty alias or keyword entries.
- Reject duplicate aliases or keywords within one source after trimming.
- Measure text limits in Unicode code points: `display_name` at most 80, `description` at most 500, each alias or keyword at most 80.
- Permit at most 20 aliases and 30 keywords per source.
- Treat duplicate aliases and keywords as exact, case-sensitive matches after trimming.
- Keep existing duplicate `name`, engine, DSN-reference, and secret-redaction behavior.
- Do not impose global alias uniqueness: two sources may share a business term, which intentionally creates an ambiguous match that must be resolved by the user.

Existing configurations without the new required fields are a deliberate configuration-breaking change. Startup must fail with a clear source-specific error rather than silently exposing a source that the AI cannot identify reliably.

## MCP source discovery contract

`list_sources` returns sources ordered by `name`, with this public shape:

```json
{
  "id": "orders",
  "display_name": "订单库",
  "description": "存储用户充值订单、支付状态、退款记录和渠道流水",
  "aliases": ["充值库", "支付订单"],
  "keywords": ["用户充值", "支付", "退款", "订单状态"],
  "engine": "mysql"
}
```

DSNs, hosts, credentials, pool settings, and other connection details must never be returned.

The server does not accept display names or aliases in `source_id`; this avoids hidden resolution rules and preserves exact source binding in confirmation hashes.

## AI selection behavior

The local client guide defines the expected model workflow:

1. When a request introduces a database intent and the source is not already unambiguous in the conversation, call `list_sources`.
2. Compare the user's words with `display_name`, `aliases`, `keywords`, and `description`.
3. If exactly one source clearly matches, use its `id` as `source_id`.
4. If no source clearly matches, ask the user for clarification.
5. If multiple sources plausibly match, show short choices using `display_name` and `description`, then wait for the user to choose.
6. Never silently choose among multiple candidates and never query all candidates automatically.
7. The client conversation may remember the user's choice; the MCP server remains stateless.

This is guidance for MCP-capable AI clients, not a server-side natural-language classifier. The approach is explainable and keeps the first version small.

## Preview and response behavior

Confirmation hashes remain bound to the stable source ID. Preview responses additionally expose a source reference:

```json
{
  "source": {
    "id": "orders",
    "display_name": "订单库"
  }
}
```

The display name is explanatory metadata and is not part of the execution authorization hash. The stable source ID remains hash-bound. This lets a user verify the selected instance before a high-impact operation without making a mutable label security-sensitive.

Adding display metadata must not weaken existing quick/strict confirmation policy, multi-statement confirmation, SQL parsing, or source isolation.

## Beginner onboarding guide

Create `docs/getting-started-local-clients.md` and link it prominently from `README.md`.

The guide follows one linear path:

1. Prerequisites and database account recommendations.
2. Create `config.yaml` with multiple MySQL/ClickHouse examples and business profiles.
3. Define DSN environment variables without placing credentials in YAML or client configuration examples where avoidable.
4. Build or download the local binary.
5. Configure Claude Desktop for local binary execution.
6. Configure Codex for local binary execution.
7. Configure each client to launch the server through Docker.
8. Verify connection with `list_sources` and `list_tables`.
9. Show natural-language source selection with a unique match.
10. Show a multiple-candidate interaction that asks the user to choose.
11. Show a read query, strict-mode preview, high-risk preview, matching confirmation, and preview mismatch.
12. Troubleshoot paths, environment inheritance, YAML structure, Docker mounts, DSN resolution, and source ambiguity.

The guide does not cover ChatGPT web, public remote MCP endpoints, or the future enterprise gateway.

## Error handling

- Configuration errors name the affected source but never include resolved DSNs.
- Unknown `source_id` keeps the existing `unknown_source` error.
- Empty or oversized metadata fails during configuration load.
- Ambiguity is a client conversation outcome, not an MCP error code.
- Client examples explicitly instruct the model to ask the user rather than guessing.

## Testing and acceptance

- Configuration tests cover required metadata, trimming, empty entries, duplicates, limits, multiple sources of the same engine, and secret-redacted errors.
- Registry/service tests prove metadata is preserved and `list_sources` is deterministic.
- MCP schema/response tests prove all public profile fields are returned and no connection data is exposed.
- Preview tests prove the human label is present while the hash remains bound to the stable source ID.
- Integration tests configure two MySQL instances/source entries plus ClickHouse metadata and verify exact selection by `source_id`.
- Documentation checks reject legacy source shapes and verify Claude Desktop/Codex local and Docker sections exist.
- Full unit tests, vet, integration-tag tests, and Docker-backed MySQL/ClickHouse tests must pass before acceptance.

## Implementation ownership

The primary agent owns design, task decomposition, review, and acceptance only. Product code, tests, and onboarding documentation are implemented by Terra-high subagents. No implementation task may be accepted without specification review, code-quality review, and evidence from the required verification commands.
