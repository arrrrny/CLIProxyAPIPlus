# Contract: GET /api.json (models.dev-format catalog)

**Feature**: `001-provider-context-length` | **Owner**: new `api_json.go` handler; route registered in `internal/api/server.go`
**Related**: spec FR-009; plan D4 (CLIProxyApiPlus is authoritative)

## Purpose
Expose a single authoritative, models.dev-format catalog built from the enriched global registry so downstream consumers (Quotio ProxyBridge, Kimi Code custom registry) read correct per-model limits instead of guessing from a static table.

## Request
`GET /api.json` (and `GET /` may alias to it). No auth required beyond the server's existing access policy.

## Response (200 OK) — catalog shape
```json
{
  "cliproxy": {
    "id": "cliproxy",
    "name": "CLIProxyAPI",
    "api": "<public base>/v1",
    "type": "openai",
    "env": ["CLIPROXY_API_KEY"],
    "models": {
      "ocd/deepseek-v4-flash-free": {
        "id": "ocd/deepseek-v4-flash-free",
        "name": "DeepSeek V4 Flash Free",
        "tool_call": true,
        "reasoning": false,
        "limit": { "context": 1000000, "output": 32000 },
        "context_length": 1000000,
        "max_completion_tokens": 32000
      },
      "k3-256k": {
        "id": "k3-256k",
        "name": "K3 256K",
        "tool_call": true,
        "reasoning": false,
        "limit": { "context": 256000, "output": 8192 },
        "context_length": 256000,
        "max_completion_tokens": 8192
      }
    }
  }
}
```

## Field rules
- `cliproxy.models[<id>].limit.context` = `Model.context_length` (omit entry's `limit` entirely if `context_length == 0`, per D3).
- `cliproxy.models[<id>].limit.output` = `Model.max_completion_tokens`.
- `context_length` / `max_completion_tokens` mirrored at model top-level for consumers that read them directly.
- `reasoning` is `true` only for known reasoning families; default `false`.
- `tool_call` defaults `true` for chat-capable models.
- Namespaced ids preserved verbatim (keys exactly as the provider returns).
- `api` is the proxy's public base URL + `/v1`.

## Error behavior
- On empty/partial registry, returns the catalog with whatever models are known; never fabricates limits for unknown windows (D3).
- Provider fetch failures do not affect this endpoint — it always reflects last-good registry state.
