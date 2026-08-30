# Implementation Plan: Accurate model context-length via dedicated provider endpoints

**Branch**: `001-provider-context-length` | **Date**: 2026-08-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-provider-context-length/spec.md`

## Summary

CLIProxyApiPlus must become the single authoritative source of per-model `context_length` / `max_completion_tokens`. Today two defects stop that: (1) the `OpenAIModels` handler strips `context_length` from the `/v1/models` response even though the registry already computes it, and (2) built-in context windows are a hand-maintained static table that is wrong for free-model variants. The fix reads the real `context_length` from each dedicated provider's OpenAI-compatible `/v1/models` endpoint, merges it into the global registry (fetched value wins when > 0), emits it from `/v1/models`, and serves a models.dev-format `/api.json` catalog. Six providers (`opencode`, `opencode-go`, `openrouter`, `kilo`, `kimi`, `z.ai`) become first-class dedicated providers onboarded via the existing `openai-compatibility` config.

## Technical Context

**Language/Version**: Go 1.26+ (per AGENTS.md)

**Primary Dependencies**: Gin (HTTP routing/handlers); standard `net/http` for the provider fetcher; existing `internal/registry`, `internal/config`, `internal/constant` packages; embedded `models/models.json` for static definitions; `models.dev`-format catalog output.

**Storage**: In-memory global model registry (`ModelRegistry`); embedded static model JSON. No external database introduced by this feature.

**Testing**: Standard Go `testing` (`go test ./...`); `net/http/httptest` for mock provider `/v1/models` endpoints; table-driven unit tests for the fetcher, handler, and catalog serializer.

**Target Platform**: Linux/macOS API proxy server (same binary as the rest of CLIProxyApiPlus).

**Project Type**: web-service (OpenAI/Gemini/Claude/Codex-compatible API proxy)

**Performance Goals**: `/v1/models` and `/api.json` served with negligible added latency (registry reads are in-memory/cached); provider fetches run async/non-blocking; periodic refresh default every 6h, configurable, must not block request serving.

**Constraints**:
- No network timeouts set after an upstream connection is established (AGENTS.md rule); only credential acquisition may time out.
- No `log.Fatal`/`log.Fatalf`; use logrus, wrap errors, never leak API keys/tokens in logs.
- Namespaced model ids (`ocd/`, `ocg-1/`, `z-ai/`, `kilo/`, `kiro-`, `k3`, `k3-256k`) preserved exactly — no normalization.
- Fetched value wins only when > 0; on any fetch error keep last-good values (never wipe).
- `gofmt -w` required after changes; goimports-style imports.

**Scale/Scope**: 6 dedicated providers; each may return hundreds of models. Bounded to CLIProxyApiPlus; downstream Quotio changes are out of scope (see spec Assumptions).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The project constitution (`.specify/memory/constitution.md`) is still a template and not ratified, so there are no formal constitutional gates to enforce. The operative guidance is `AGENTS.md`: Go 1.26+, logrus structured logging, no `log.Fatal`, `gofmt`, no post-connection timeouts, no secret leakage, minimal KISS changes, and — specifically — do not make standalone changes to `internal/translator/` (not touched by this feature).

**Verdict**: PASS. The design reuses existing patterns (`OpenAICompatibility` config, `StartModelsUpdater` refresh mechanics, `kilo_executor` fetch pattern) and introduces no new architectural layers, external services, or schema migrations. No complexity warrants a violation justification.

## Project Structure

### Documentation (this feature)

```text
specs/001-provider-context-length/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── openai-models.md # /v1/models response contract (the fix)
│   └── api-json.md      # /api.json models.dev catalog contract
├── checklists/
│   └── requirements.md  # Spec quality checklist (from /skill:speckit-specify)
└── tasks.md             # Phase 2 output (/skill:speckit-tasks - NOT created here)
```

### Source Code (repository root)

```text
sdk/api/handlers/openai/
├── openai_handlers.go   # EDIT: OpenAIModels carries context_length/max_completion_tokens
└── api_json.go          # NEW: /api.json handler (models.dev format)

internal/registry/
├── model_registry.go    # context: GetAvailableModels, convertModelToMap (already emits fields)
├── model_definitions.go # context: static table + provider channel switch
├── model_updater.go     # context: StartModelsUpdater refresh mechanics (mirror)
├── provider_models.go   # NEW: ProviderModelsFetcher (fetch + merge)
└── provider_models_test.go # NEW: fetcher unit tests

internal/config/
└── config.go            # reuse OpenAICompatibility (BaseURL, APIKeyEntries, Models)

internal/constant/
└── constant.go          # NEW: dedicated provider type constants (opencode, openrouter, kimi, z.ai, ...)

internal/api/
└── server.go            # EDIT: register GET /api.json route in setupRoutes()

config.example.yaml      # EDIT: documented openai-compatibility blocks for the 6 providers
```

**Structure Decision**: Single Go module; the feature is a vertical slice across the existing registry/handler/config layers. No new top-level package is introduced — the fetcher lives in `internal/registry` beside the registry it mutates, mirroring how `kilo_executor` already fetches kilo models. The catalog handler lives in `sdk/api/handlers/openai` beside `OpenAIModels`.

## Complexity Tracking

> No constitution violations detected — table not required.
