# Data Model: Accurate model context-length via dedicated provider endpoints

**Phase**: 1 (Design) | **Date**: 2026-08-30 | **Feature**: `001-provider-context-length`

Entities are derived from the spec (Key Entities) and the verified `ModelInfo` struct in `internal/registry/model_registry.go`. This is the data the feature reads, merges, and serves — no new persistence layer is introduced (the global registry is in-memory).

## Entities

### Provider
A distinct upstream model source, onboarded through `openai-compatibility` config.

| Field | Type | Notes |
|-------|------|-------|
| `id` / `name` | string | Dedicated provider id, also used as registry `Type`/`OwnedBy` (e.g. `openrouter`, `z.ai`, `opencode`, `opencode-go`, `kimi`, `kilo`) |
| `baseURL` | string | OpenAI-compatible base URL; `/v1/models` is appended (handle both `/v1` and bare) |
| `apiKey` | string (secret) | Resolved from `api-key-entries[].api-key` (env-substituted); sent as `Authorization: Bearer` |
| `models` | []Model | The models this provider contributes (from config `models:` and/or live fetch) |

**Relationships**: Provider `1—*` Model. Provider is the source of a Model's `context_length`/`max_completion_tokens` when fetched.

### Model
One model offered by a provider, identified by the **exact id the provider returns** (may be namespaced).

| Field | Type | Source / Validation |
|-------|------|---------------------|
| `id` | string | Exact provider-returned id; **namespace preserved, never normalized** |
| `object` | string | Constant `"model"` |
| `owned_by` | string | Provider id |
| `type` | string | Provider id (dedicated vs generic `openai`) |
| `context_length` | int | **Merged value.** Fetched wins when > 0; else static table; else 0 (omitted) |
| `max_completion_tokens` | int | **Merged value.** Same precedence as above |
| `input_token_limit` | int | Optional, carried when > 0 |
| `output_token_limit` | int | Optional, carried when > 0 |
| `created` | int64 | Registration timestamp |

**Validation rules** (from FR-001..FR-009):
- `context_length`/`max_completion_tokens` are emitted in `/v1/models` only when > 0.
- Fetched value overrides static only when fetched > 0 (FR-003).
- On fetch error, prior value is retained; never set to 0/guessed (FR-008, D3).
- `id` is stored/served verbatim including namespaces `ocd/`, `ocg-1/`, `z-ai/`, `kilo/`, `kiro-`, `k3`, `k3-256k` (FR-005).

### GlobalModelRegistry
The single source of truth holding the merged `Model` set.

| Aspect | Detail |
|--------|--------|
| Key | `ModelInfo.ID` (exact id) |
| Merge | `ProviderModelsFetcher.FetchAndMerge` updates `ContextLength`/`MaxCompletionTokens` for matched ids |
| Read path | `GetAvailableModels("openai")` → `[]map[string]any` (already includes the fields) |
| Concurrency | Existing registry locking applies; fetcher writes are best-effort and error-safe |

### Catalog (`/api.json`)
A models.dev-format document built from the registry, exposing per-model limits.

| Field | Maps to |
|-------|---------|
| `cliproxy.models[<id>].limit.context` | `Model.context_length` |
| `cliproxy.models[<id>].limit.output` | `Model.max_completion_tokens` |
| `cliproxy.models[<id>].context_length` | `Model.context_length` |
| `cliproxy.models[<id>].max_completion_tokens` | `Model.max_completion_tokens` |

## State transitions

**Model context window lifecycle** (per id):

```text
[static table value]
   │  startup / account-connect / periodic refresh
   ▼
ProviderModelsFetcher.FetchAndMerge(provider /v1/models)
   │
   ├─ fetched > 0  ─────────────►  [fetched value]  (wins, FR-003)
   ├─ fetched == 0 / missing ───►  [keep prior value] (FR-008, D3)
   └─ error (offline/401/timeout) ►  [keep last-good, log, retry next tick]
   │
   ▼
GET /v1/models  ──► emits context_length / max_completion_tokens (FR-001)
GET /api.json   ──► emits limit.context / limit.output (FR-009)
```

**Fetch lifecycle** (per provider): `idle → (config load / account connect) → fetching → merged → [ticker every 6h] → fetching → ...`; any non-success state leaves the last merged values in place.
