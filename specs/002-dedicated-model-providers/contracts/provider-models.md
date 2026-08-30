# Contract: Per-Provider Models Endpoint

**Feature**: `002-dedicated-model-providers`

## C1 — URL resolution

`resolveModelsURL(baseURL, modelsPath)`:

- `modelsPath == ""` ⇒ legacy: strip any trailing `/v1` from `baseURL`, append `/v1/models`.
- `modelsPath != ""` ⇒ trim trailing slash from `baseURL`, append `modelsPath` verbatim.

| Provider       | BaseURL                     | ModelsPath         | Resolved URL                              |
| -------------- | --------------------------- | ------------------ | ---------------------------------------- |
| `openrouter`   | `https://openrouter.ai/api` | `/models`          | `https://openrouter.ai/api/models`       |
| `opencode`     | `https://opencode.ai/zen`   | `/v1/models`       | `https://opencode.ai/zen/v1/models`      |
| `z-ai`         | `https://api.z.ai/v1`       | `/models`          | `https://api.z.ai/v1/models`             |

## C2 — Response shapes

**Top-level style** (`openrouter`, `z-ai`):

```json
{ "object": "list", "data": [ { "id": "anthropic/claude-opus-4", "context_length": 200000, "max_completion_tokens": 32000 } ] }
```

**OpenCode style** (`opencode`, `opencode-go`) — no window:

```json
{ "object": "list", "data": [ { "id": "claude-opus-5", "object": "model", "owned_by": "opencode" } ] }
```

Window for OpenCode-style models comes from the curated table.

## C3 — Merge precedence

For each returned model id: `parsed.window > 0` ⇒ use parsed; else `curated.window > 0`
⇒ use curated; else keep existing.

## C4 — Consumer contract (unchanged from feature 001)

`GET /api.json` exposes `cliproxy.models[<id>].limit.context` / `.limit.output`. A model
with no known window omits `limit` (never `0`).
