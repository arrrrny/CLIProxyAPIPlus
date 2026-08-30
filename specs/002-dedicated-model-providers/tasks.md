# Tasks: Dedicated Model Providers for Context Windows

**Input**: Design documents from `/specs/002-dedicated-model-providers/`

**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Included and MANDATORY. Every behavior change is driven by a test that failed
first (constitution Principle III). Test tasks precede implementation tasks and carry the
behavior marker `[U#]` / `[A#]`; the TDD loop ticks a task only when it can read that
marker, and `/skill:speckit-implement` implements whatever remains unticked.

**Organization**: Tasks are grouped by user story from `spec.md`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1..US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Capture baseline: `go build ./internal/... ./cmd/server/...` (feature packages) and `go test ./internal/registry/... ./internal/config/...` scoped — confirm 4 pre-existing failures are unrelated (verified via `git stash -u` on clean `HEAD`)

---

## Phase 2: Foundational (constants)

- [X] T002 [P] [US1] [U13] Add provider type constants `OpenRouter`, `OpenCode`, `OpenCodeGo`, `ZAi` to `internal/constant/constant.go` (reuse existing `Kilo = "kilo"`); `Kilo` untouched

**Checkpoint**: Provider identifiers exist for `Type`/`OwnedBy` and for the curated-table lookup.

---

## Phase 3: User Story 1 — Per-provider endpoint + parse style (Priority: P1) 🎯 MVP

**Goal**: Each dedicated provider's `OpenAICompatibility` config infers the correct
`ModelsPath` (endpoint path) and `ParseStyle` (`top-level` vs `opencode`), overridable by
explicit config fields.

### Tests for User Story 1

- [X] T003 [P] [US1] [U19] Unit test `Config.DedicatedProviderConfigs` infers `ModelsPath`/`ParseStyle` per provider (openrouter/z-ai → top-level + `/models`; opencode → `/zen/v1/models` + opencode style; explicit override wins) — `internal/config/provider_configs_test.go::TestDedicatedProviderConfigs_OpenCodeInfersEndpoint`
- [X] T004 [P] [US1] [U13] Extend `TestDedicatedProviderConfigs_MapsEnabledProviders` want to include `ModelsPath`/`ParseStyle` — `internal/config/provider_configs_test.go`

### Implementation for User Story 1

- [X] T005 [US1] [U19] Add `ModelsPath` + `ParseStyle` fields to `OpenAICompatibility` (`internal/config/config.go`); populate them in `DedicatedProviderConfigs()` via `defaultProviderModelsPath` / `defaultProviderParseStyle` helpers, overridable by explicit config fields
- [X] T006 [US1] [U13] Add `ProviderParseStyle` type + constants to `internal/registry/provider_models.go`

**Checkpoint**: Config produces a correct per-provider `ProviderConfig` (path + style).

---

## Phase 4: User Story 2 — Fetch + parse per provider shape (Priority: P1)

**Goal**: `FetchAndMerge` requests the per-provider URL and decodes the right shape:
top-level `context_length` (openrouter/z-ai) or `id`-only with curated fallback
(opencode/opencode-go). Unknown windows are omitted, never fabricated.

### Tests for User Story 2

- [X] T007 [P] [US2] [U20] Unit test opencode endpoint → `/zen/v1/models`, new models registered with `OwnedBy:"opencode"`, curated `claude-opus-5` → 200000/64000, `gemini-3.6-flash` → 1048576/65536 — `internal/registry/provider_models_test.go::TestProviderFetcher_OpenCodeRegistersAndUsesCuratedTable`
- [X] T008 [P] [US2] [U21] Unit test z.ai top-level registers new model `z-ai/glm-4.6` with `ContextLength:128000`/`MaxCompletionTokens:8000` — `internal/registry/provider_models_test.go::TestProviderFetcher_ZAiTopLevelRegistersNewModel`
- [X] T009 [P] [US2] [U22] Unit test opencode curated fallback: curated model gets window; `unknown-model` gets `ContextLength:0` (not fabricated) — `internal/registry/provider_models_test.go::TestProviderFetcher_OpenCodeCuratedFallbackAndOmission`

### Implementation for User Story 2

- [X] T010 [US2] [U20][U21] Add `resolveModelsURL(baseURL, modelsPath)`, `parseTopLevelModels(body)`, `parseOpenCodeModels(body)`; `FetchAndMerge` dispatches on `ParseStyle` — `internal/registry/provider_models.go`
- [X] T011 [US2] [U20][U22] Add `MergeProviderContextWindows(provider, items, curated)` + `ensureProviderModelLocked` (auto-register, parsed > 0 wins, else curated > 0, else kept) — `internal/registry/provider_models.go`
- [X] T012 [US2] [U20][U22] Add curated table `openCodeCuratedWindows` + `curatedWindowsFor` — `internal/registry/dedicated_provider_models.go` (NEW)

**Checkpoint**: Each provider's real endpoint + shape yields correct windows; opencode uses the curated table; unknown windows omitted.

---

## Phase 5: User Story 3 — Resilient refresh + docs (Priority: P2)

**Goal**: The existing `ProviderModelsRefresher` (001) now drives the per-provider
`ModelsPath`/`ParseStyle`; config.example.yaml documents the real per-provider endpoints.

### Implementation for User Story 3

- [X] T013 [US3] [U19] Document the 5 providers' real endpoints, model paths, and parse styles in `config.example.yaml` (Dedicated provider onboarding block)
- [X] T014 [US3] [U19][U20] `ProviderModelsRefresher` wires `ProviderConfig.ModelsPath`/`ParseStyle` from `DedicatedProviderConfigs()` into `FetchAndMerge` (extends 001's refresher; error-safe, keeps last-known-good)

**Checkpoint**: Docs + wiring complete; refresh uses the correct per-provider endpoint.

---

## Phase 6: Polish & Cross-Cutting Concerns

### Acceptance gates (outer loop)

Each gate must be GREEN before its user story is considered complete. `A1`–`A5` map to
`tdd/test-list.md`; the per-provider path/shape change is exercised end-to-end through the
real HTTP entry points (`GET /v1/models`, `GET /api.json`), which are covered by the 001
acceptance suite (`sdk/api/handlers/openai/acceptance_test.go`) plus the new unit
behaviors U19–U22.

- [X] T015 [A1] `openrouter` fetched from `…/api/v1/models` (top-level) → `/api.json` `limit.context` correct (SC-001)
- [X] T016 [A2] [U20][U22] `opencode`/`opencode-go` fetched from `…/zen/v1/models`, window from curated table → `/api.json` `limit.context` correct (FR-002, FR-008, SC-001)
- [X] T017 [A3] `z-ai` fetched from its real endpoint → `/api.json` `limit.context` correct (SC-001)
- [X] T018 [A4] `kilo` nested `top_provider.context_length` surfaced (inherited from 001)
- [X] T019 [A5] model with no known window → `limit` omitted on `/api.json` (never `0`) (SC-002)

- [X] T020 [P] Run `gofmt -w` on changed files; `go build ./internal/... ./cmd/server/...` compiles
- [X] T021 [P] Run scoped suite `go test ./internal/registry/... ./internal/config/... ./internal/constant/...`; feature-scoped packages green; 4 pre-existing, unrelated failures remain (verified via stash)
- [ ] T022 [P] MANUAL: live end-to-end with real keys (openrouter + z-ai + opencode) via `quickstart.md` (needs provider API keys; not run here)

---

## Dependencies & Execution Order

- **Phase 2 (T002)**: no deps.
- **Phase 3 (US1, T003–T006)**: depends on T002 for constants.
- **Phase 4 (US2, T007–T012)**: depends on T006 (ParseStyle type).
- **Phase 5 (US3, T013–T014)**: depends on US2.
- **Phase 6**: depends on all stories.

## Notes

- Feature 001 established `/api.json`, `/v1/models`, `ProviderModelsFetcher`, and the
  6-provider onboarding. This feature (002) adds the per-provider `ModelsPath`/`ParseStyle`
  so the *correct* endpoint and response shape are used per provider — the original point
  of the spec.
- Behavior markers `[U19]..[U22]` (unit) and `[A1]..[A5]` (acceptance) are load-bearing;
  the loop ticks a task only when it can read the marker.
- `kilo` reuses its existing `FetchKiloModels` nested-shape path and is **not** changed by
  this feature beyond onboarding docs.
- Do **not** modify `internal/translator/` (per AGENTS.md). No new top-level package.
