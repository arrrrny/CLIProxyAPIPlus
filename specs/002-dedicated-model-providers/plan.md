# Implementation Plan: Dedicated Model Providers for Context Windows

**Branch**: `002-dedicated-model-providers` | **Date**: 2026-08-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-dedicated-model-providers/spec.md`.
Builds on `001-provider-context-length`, which introduced the catalog, `/api.json`
handler, and the generic `ProviderModelsFetcher`. This feature makes each provider's
**own** models endpoint address and response shape first-class, instead of assuming a
uniform `<base-url>/v1/models` with a top-level `context_length`.

## Summary

Feature 001 proved that a single `/v1/models` path with a top-level `context_length`
does not hold for every provider. In reality:

- `openrouter` → `…/api/v1/models` with a **top-level** `context_length` (verified live).
- `kilo` → `…/api/openrouter/models` with the window **nested** under `top_provider` (handled by the existing `FetchKiloModels`).
- `opencode` / `opencode-go` → `…/zen/v1/models` returning `id`/`object`/`created`/`owned_by` only — **no** `context_length` at all.
- `z.ai` → real path is `…/api/v1/models` (the previously-assumed `/v1/models` returns 404); response shape assumed top-level `context_length` pending auth-confirmed probe.

This plan adds a per-provider **ModelsPath** (endpoint path under the configured base URL)
and a per-provider **ParseStyle** (`top-level` vs `opencode`) so the fetcher requests the
right URL and decodes the right shape. Where the live endpoint omits the window
(opencode/open-code-go), a **curated static table** keyed by model id supplies known
windows; a model with neither source gets `limit` omitted (never fabricated).

## Technical Context

**Language/Version**: Go 1.26+ (per AGENTS.md).

**Primary Dependencies**: `internal/registry`, `internal/config`, `internal/constant`; the
existing `OpenAICompatibility` config and `ProviderModelsFetcher`/`ProviderModelsRefresher`
from 001.

**Storage**: In-memory global model registry (no external store). Curated table is an
embedded Go map.

**Testing**: Standard Go `testing` (`go test ./...`); `net/http/httptest` for mock
provider endpoints; table-driven unit tests for path/parser inference.

**Target Platform**: Linux/macOS API proxy server (same binary).

**Performance Goals**: No added request-serving latency; refresh is async/non-blocking
(mirrors 001). Curated lookup is O(1).

**Constraints**:
- No `log.Fatal`; use logrus; never leak API keys/tokens.
- Namespaced model ids preserved verbatim (no normalization).
- Fetched value wins only when > 0; zero/missing keeps last-good; never fabricate.
- `gofmt -w` required after changes.
- Per AGENTS.md, do **not** make standalone changes to `internal/translator/` (untouched).

**Scale/Scope**: 5 dedicated providers (`openrouter`, `kilo`, `opencode`, `opencode-go`,
`z.ai`). `kilo` reuses its existing dynamic fetch; this feature only adds the
ModelsPath/ParseStyle inference + opencode curated table + z.ai top-level path.

## Constitution Check

The project constitution (`.specify/memory/constitution.md`) Principle III makes TDD
non-negotiable: every behavior change is driven by a test that failed first, test tasks
are not optional, and tests are never weakened to reach green. This feature complies:
each new behavior (U19–U22) has a dedicated unit test that failed before its
implementation and passes after. Pre-existing, unrelated test failures are outside this
feature's scope (verified on clean `HEAD` via stash).

**Verdict**: PASS. The design extends existing config fields and the existing fetcher
(rather than introducing a new architectural layer), reuses `ProviderModelsRefresher`,
and adds one new file (`dedicated_provider_models.go`) holding only data + a lookup
helper. No complexity warrants a violation justification.

## Project Structure

### Documentation (this feature)

```text
specs/002-dedicated-model-providers/
├── plan.md              # This file
├── research.md          # Phase 0 output (endpoint probes)
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── provider-models.md # per-provider endpoint + shape contract
├── checklists/
│   └── requirements.md  # Spec quality checklist (from /skill:speckit-specify)
├── tasks.md             # Phase 2 output
└── tdd/
    ├── test-list.md     # behaviors (this feature)
    ├── cycle-log.md     # baseline + cycle notes
    └── verification.md  # TDD audit verdict
```

### Source Code (repository root)

```text
internal/constant/
└── constant.go                     # EDIT: OpenRouter/OpenCode/OpenCodeGo/ZAi constants

internal/config/
├── config.go                       # EDIT: OpenAICompatibility adds ModelsPath + ParseStyle;
│                                   #       DedicatedProviderConfigs infers both per provider;
│                                   #       helpers defaultProviderModelsPath / defaultProviderParseStyle
└── provider_configs_test.go        # EDIT: U13 updated want; U19 new inference test

internal/registry/
├── provider_models.go              # EDIT: resolveModelsURL, parseTopLevelModels, parseOpenCodeModels,
│                                   #       FetchAndMerge dispatch, MergeProviderContextWindows,
│                                   #       ensureProviderModelLocked; ProviderParseStyle type
├── provider_models_test.go         # EDIT: U20/U21/U22 new tests + opencodeBody helper
└── dedicated_provider_models.go    # NEW: curated opencode/open-code-go windows table + curatedWindowsFor

config.example.yaml                 # EDIT: dedicated-provider onboarding block documents real per-provider endpoints
```

**Structure Decision**: Single Go module; the change is a vertical extension of the
existing registry/config/constant layers that 001 established. No new top-level package.
The curated table lives in its own file (`dedicated_provider_models.go`) so it is pure
data + a lookup helper, easy to extend.

## Complexity Tracking

> No constitution violations detected — table not required.
