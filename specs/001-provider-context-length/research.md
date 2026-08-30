# Research: Accurate model context-length via dedicated provider endpoints

**Phase**: 0 (Outline & Research) | **Date**: 2026-08-30 | **Feature**: `001-provider-context-length`

This document resolves the spec's open decisions and records verified codebase facts that the plan depends on. Every `NEEDS CLARIFICATION` from the spec was already resolved with a documented default (see spec Assumptions); the items below are the technical unknowns confirmed during planning.

## Verified codebase facts (from exploration)

- **`OpenAIModels` strips the fields** — `sdk/api/handlers/openai/openai_handlers.go:53-98`. The standard path builds each entry with only `id`, `object`, `created`, `owned_by` and discards everything else. The separate `?client_version=` codex path (`codexClientModelsResponse`) is unchanged and already emits `context_length`.
- **The registry already computes the fields** — `internal/registry/model_registry.go` `ModelInfo` has `ContextLength`, `MaxCompletionTokens`, `InputTokenLimit`, `OutputTokenLimit`, `Type`, `OwnedBy`. `convertModelToMap` (openai case, ~lines 1175-1212) already emits `context_length` / `max_completion_tokens` into the map when > 0. So `GetAvailableModels("openai")` already returns them; the handler is the only thing dropping them. **Fix is purely in the handler**, not the registry.
- **Fetcher target exists conceptually** — `kilo_executor.go:440` shows the exact read pattern to mirror: `ContextLength: int(value.Get("context_length").Int())`, `OwnedBy: "kilo"`, `Type: "kilo"`. `kilo` already has a dedicated constant and fetch mechanism.
- **Config reuse confirmed** — `internal/config/config.go:700` `OpenAICompatibility{ Name, BaseURL, APIKeyEntries, Models, Headers, ... }` is already parsed from `openai-compatibility:` YAML. No new config schema needed.
- **Refresh mechanics exist** — `internal/registry/model_updater.go:56` `StartModelsUpdater(ctx)` runs `tryStartupRefresh` then a `time.NewTicker(modelsRefreshInterval)` (currently 3h for the remote models.json). We mirror this with a separate, configurable interval for provider fetches.
- **No `/api.json` and no `ProviderModelsFetcher` exist yet.** Routes are registered in `internal/api/server.go` `setupRoutes()` (e.g. `v1.GET("/models", ...)` at ~line 532). The new `/api.json` route is added there.

## Resolved decisions

### D1 — Exact `base-url` per provider
| Provider | OpenAI-compatible `/v1/models` base URL | Confidence |
|----------|------------------------------------------|-----------|
| openrouter | `https://openrouter.ai/api/v1` | Confirmed in `config.example.yaml:400` + wiki |
| z.ai | `https://api.z.ai/v1` | Per source doc |
| opencode | `https://opencode.ai/zen/v1` | Per source doc |
| opencode-go | `https://opencode.ai/zen/v1` (shares the zen endpoint; the `-go` variant may use a distinct path — **confirm at implement time**) | Reasonable default |
| kimi | `https://api.kimi.com/coding/v1` | Derived from `internal/auth/kimi/kimi.go` `KimiAPIBaseURL = "https://api.kimi.com/coding"` |
| kilo | Already a dedicated provider (existing `kilo_executor` fetch); reuse, do not re-fetch via generic path | Confirmed |

**Decision**: Use the table above. `opencode-go` uses the same zen base URL as `opencode` unless implementation discovers otherwise; the base URL is config-driven, so correction is a one-line config change, not a code change.

### D2 — Refresh cadence
**Decision**: Default periodic refresh every **6 hours**, configurable via a new config key (e.g. `provider-models-refresh-interval`, duration string, default `6h`). Mirrors `StartModelsUpdater` ticker mechanics. Startup refresh runs immediately (best-effort, async, non-blocking) on config load / account connect.

### D3 — Providers whose `/v1/models` returns only `id`
**Decision**: Keep last-known-good `context_length`/`max_completion_tokens` (from static table or prior fetch); omit the `limit` for that model in `/api.json` rather than guessing. Never overwrite a good value with `0`.

### D4 — Where `/api.json` is served
**Decision**: CLIProxyApiPlus is authoritative and serves `/api.json` on its own port (the proxy's public base, e.g. `18317`). Downstream Quotio proxies/reads it instead of rebuilding limits from its static `modelLimits` table. Confirmed approach per source doc §4.5/§4.6.

## Design decisions (derived)

- **Dedicated providers via config, not new executors.** Rather than writing six executors, onboard the five new providers (`opencode`, `opencode-go`, `openrouter`, `kimi`, `z.ai`) through the existing `openai-compatibility` config, each with a `name` used as the registry `Type`/`OwnedBy`. A single generic `ProviderModelsFetcher` iterates these config entries and merges fetched metadata. `kilo` already has dedicated handling and is reused.
- **Merge key = exact provider-returned id.** `ProviderModelsFetcher.FetchAndMerge` looks up/create `ModelInfo` by the exact `id` the provider returns, preserving namespaces. Fetched `ContextLength`/`MaxCompletionTokens` overwrite the static value only when > 0.
- **Handler fix is minimal.** `OpenAIModels` carries `context_length`, `max_completion_tokens`, `input_token_limit`, `output_token_limit` from the registry map into `filteredModel` when present and > 0. The codex path is untouched.
- **Catalog built from registry.** `/api.json` serializes `GetAvailableModels(...)` into models.dev shape: `limit.context = ContextLength`, `limit.output = MaxCompletionTokens`.

## Alternatives considered

- *Maintain/curate the static table* — rejected: free-model sizes drift and models.dev is itself wrong for free models; only the live provider endpoint is correct.
- *One executor per provider* — rejected in favor of config-driven generic fetcher: less code, onboarding is config-only (FR-010), consistent with `OpenAICompatibility`.
- *Patch Quotio* as part of this feature — rejected: different repo/language (Swift) and out of CLIProxyApiPlus scope; recorded as downstream follow-up.
