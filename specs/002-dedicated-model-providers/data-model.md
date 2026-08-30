# Data Model: Dedicated Model Providers for Context Windows

**Feature**: `002-dedicated-model-providers` | **Date**: 2026-08-30

## Entities

### Dedicated Provider Configuration (config layer)

Extends `OpenAICompatibility` (existing struct in `internal/config/config.go`):

| Field          | Type   | Source                        | Meaning                                              |
| -------------- | ------ | ----------------------------- | ---------------------------------------------------- |
| `Name`         | string | config                        | provider key; becomes registry `Type`/`OwnedBy`      |
| `BaseURL`      | string | config                        | provider base URL (no models path)                   |
| `APIKeyEntries`| []...  | config                        | bearer credentials                                  |
| `ModelsPath`   | string | **inferred** or config override | path appended to `BaseURL` for the models endpoint |
| `ParseStyle`   | string | **inferred** or config override | `top-level` \| `opencode` — how to decode the body |

Inference helpers (`defaultProviderModelsPath` / `defaultProviderParseStyle`):

| Provider       | ModelsPath            | ParseStyle   |
| -------------- | --------------------- | ------------ |
| `opencode`     | `/zen/v1/models`      | `opencode`   |
| `opencode-go`  | `/zen/v1/models`      | `opencode`   |
| `openrouter`   | `/models`             | `top-level`  |
| `z-ai`         | `/models`             | `top-level`  |
| *(default)*    | `/v1/models`          | `top-level`  |

Explicit `models-path` / `parse-style` config fields override inference.

### Provider Model Limit (registry layer)

```go
type ProviderParseStyle int
const (
    ParseStyleTopLevel ProviderParseStyle = iota  // decode data[].context_length
    ParseStyleOpenCode                            // data[].id only; curated table supplies window
)
```

`ProviderConfig` (from 001) extended with `ModelsPath string` and `ParseStyle ProviderParseStyle`.

### Curated Window (data file)

```go
// internal/registry/dedicated_provider_models.go
var (
  openCodeCuratedWindows = map[string]ModelWindow{ /* id -> {ContextLength, MaxOutput} */ }
  curatedProviderWindows = map[string]map[string]ModelWindow{
      "opencode":    openCodeCuratedWindows,
      "opencode-go": openCodeCuratedWindows,
  }
)
func curatedWindowsFor(provider string) map[string]ModelWindow // nil for openrouter/z-ai/kilo
```

A model with a curated entry but no live window gets the curated window. A model with
neither → `limit` omitted (FR-006).

### Merged Registry Window

`ModelRegistration.Info.ContextLength` / `MaxCompletionTokens` — set by
`MergeProviderContextWindows(provider, items, curated)`: parsed > 0 wins, else curated
> 0, else unchanged. Propagated to `InfoByProvider[provider]`.

## Invariants

- I1: `resolveModelsURL` with empty `ModelsPath` reproduces legacy `/v1/models` normalization.
- I2: `parseOpenCodeModels` decodes only `id`; never invents a window.
- I3: Merge never overwrites a positive stored value with `0`.
- I4: A model known only to the curated table is auto-registered (so `/api.json` exposes it) but with `limit` omitted if no window exists (FR-006).
