# Contract: GET /v1/models (OpenAI-compatible models list)

**Feature**: `001-provider-context-length` | **Owner**: `OpenAIModels` in `sdk/api/handlers/openai/openai_handlers.go`
**Related**: spec FR-001; plan §Summary (root cause #1)

## Purpose
This contract defines the **fixed** response shape. Today the handler returns only `id`, `object`, `created`, `owned_by` and drops `context_length`/`max_completion_tokens`. After the fix it carries the capability fields the registry already computes.

## Request
`GET /v1/models` (query param `?client_version=` continues to take the separate codex path and is **unchanged**).

## Response (200 OK) — fixed shape
```json
{
  "object": "list",
  "data": [
    {
      "id": "ocd/deepseek-v4-flash-free",
      "object": "model",
      "created": 1700000000,
      "owned_by": "openrouter",
      "context_length": 1000000,
      "max_completion_tokens": 32000,
      "input_token_limit": 1000000,
      "output_token_limit": 32000
    }
  ]
}
```

## Field rules
- `id`, `object`, `created`, `owned_by` — always present (unchanged behavior).
- `context_length` — present **only when > 0** (registry value; may be static or fetched).
- `max_completion_tokens` — present **only when > 0**.
- `input_token_limit` / `output_token_limit` — present **only when > 0** (optional capability fields).
- Namespaced ids (`ocd/...`, `z-ai/...`, `k3-256k`, ...) are preserved verbatim.
- The codex `?client_version=` path is out of scope and keeps its existing (already-correct) shape.

## Non-goals
- No change to field names, pagination, or the `object: "list"` envelope.
- No schema version bump; additive fields only (backward compatible for clients ignoring them).
