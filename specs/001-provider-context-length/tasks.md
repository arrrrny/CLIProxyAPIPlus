# Tasks: Accurate model context-length via dedicated provider endpoints

**Input**: Design documents from `/specs/001-provider-context-length/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Included — the source feature description (`docs/api-json-context-length-spec.md` §7) explicitly specifies unit/integration tests for the fetcher, the handler, and the catalog. Tests are written FIRST and must FAIL before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1..US4)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm baseline on the existing Go module before changes.

- [ ] T001 Capture build/runtime baseline: run `go build ./...` and confirm `/v1/models` currently drops `context_length` (regression baseline) per `sdk/api/handlers/openai/openai_handlers.go:53-98`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Cross-cutting identifiers that US2/US3 need before they can label models as dedicated providers.

**⚠️ CRITICAL**: No dedicated-provider labeling can begin until this phase is complete.

- [ ] T002 [P] Add dedicated provider type constants (`opencode`, `opencode-go`, `openrouter`, `kimi`, `z.ai`) to `internal/constant/constant.go` (reuse existing `Kilo = "kilo"` pattern; `kilo` already present)

**Checkpoint**: Foundation ready — provider identifiers exist for registry `Type`/`OwnedBy`.

---

## Phase 3: User Story 1 - Emit real context window from /v1/models (Priority: P1) 🎯 MVP

**Goal**: The `/v1/models` response includes `context_length` / `max_completion_tokens` (and optional `input_token_limit` / `output_token_limit`) for every model the registry already knows, instead of dropping them.

**Independent Test**: `go test ./sdk/api/handlers/openai/...`; a registered model with a `context_length` appears in `OpenAIModels` output with that field populated (non-zero). Also `curl /v1/models | jq '.data[] | select(.id=="ocd/deepseek-v4-flash-free")'` shows `context_length`.

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T003 [P] [US1] [U1][U2][U3][U4][U5] Contract test: `OpenAIModels` emits `context_length`/`max_completion_tokens` for a registered model — add to `sdk/api/handlers/openai/openai_handlers_test.go`

### Implementation for User Story 1

- [ ] T004 [US1] [U1][U2][U3][U4][U5] Carry `context_length`, `max_completion_tokens`, `input_token_limit`, `output_token_limit` from the registry map into `filteredModel` (when > 0) in `OpenAIModels` at `sdk/api/handlers/openai/openai_handlers.go:80-92`
- [X] T005 [US1] [U6] Leave the `?client_version=` codex path (`codexClientModelsResponse`) unchanged

**Checkpoint**: `/v1/models` now exposes context windows from the registry (static values at this point). Independently testable.

---

## Phase 4: User Story 2 - Context length sourced live from each provider (Priority: P1)

**Goal**: A `ProviderModelsFetcher` reads the real `context_length` / `max_completion_tokens` from each dedicated provider's OpenAI-compatible `/v1/models` endpoint and merges it into the global registry (fetched value wins when > 0), replacing the wrong static table for free models.

**Independent Test**: `go test ./internal/registry/...`; a mock `httptest` server returning `/v1/models` with `context_length` for `ocd/deepseek-v4-flash-free` (1M) and `k3-256k` (256k) asserts the registry's `ContextLength`/`MaxCompletionTokens` are set and override static values, with exact ids preserved.

### Tests for User Story 2

- [X] T006 [P] [US2] [U7][U8][U9][U10][U11][U12] Unit test `ProviderModelsFetcher.FetchAndMerge` against a mock `httptest` `/v1/models` (assert merge, override of static value, and namespaced-id preservation) — create `internal/registry/provider_models_test.go`

### Implementation for User Story 2

- [X] T007 [US2] [U7][U8][U9][U10][U11][U12] Create `internal/registry/provider_models.go` with `ProviderModelsFetcher` struct + `FetchAndMerge(ctx, providerID, baseURL, apiKey)` — GET `<baseURL>/v1/models` with `Authorization: Bearer`, parse `data[]`, look up/create `ModelInfo` keyed by exact provider id, set `ContextLength`/`MaxCompletionTokens` when > 0 (mirror `kilo_executor.go:440` read pattern)
- [X] T008 [US2] [U14] Kick off `ProviderModelsFetcher.FetchAndMerge` on config load / account connect for each configured dedicated provider (best-effort, async, non-blocking) — wired via `startDedicatedProviderContextRefresher` in `cmd/server/main.go`, invoked from both the standalone TUI path (`startModelCatalogUpdaters`) and the main proxy path
- [X] T009 [US2] [U10][U11] Make fetch error-safe: on offline / 401 / timeout / missing `context_length`, log and keep last-good values; never overwrite with `0` or guess (per research D3)

**Checkpoint**: Free-model context windows are now real and override the static table. Independently testable via unit tests + `curl /v1/models`.

---

## Phase 5: User Story 3 - Six providers are dedicated + periodic refresh (Priority: P2)

**Goal**: `opencode`, `opencode-go`, `openrouter`, `kilo`, `kimi`, `z.ai` are first-class dedicated providers onboarded via the existing `openai-compatibility` config, with a periodic refresh of fetched metadata.

**Independent Test**: With documented config blocks for the six providers, the server loads them as distinct providers (registry `Type`/`OwnedBy` set) and a best-effort fetch is attempted for each; a mock provider confirms refresh re-runs on the ticker.

### Implementation for User Story 3

- [X] T010 [US3] [U13] Reuse `OpenAICompatibility` (`internal/config/config.go:700`) to onboard `opencode`/`opencode-go`/`openrouter`/`kimi`/`z.ai` as dedicated providers (set `ModelInfo.Type`/`OwnedBy` = provider `name`); `kilo` already dedicated — wired via `Config.DedicatedProviderConfigs()` feeding `startDedicatedProviderContextRefresher`
- [X] T011 [US3] [U14][U15] Add a periodic refresh ticker (configurable, default `6h`) that re-runs `FetchAndMerge` for all dedicated providers, mirroring `StartModelsUpdater` mechanics in `internal/registry/model_updater.go` (offline/error-safe, keeps cached values)
- [X] T012 [US3] [A6] Document `openai-compatibility` blocks for the 6 providers (base URLs from research D1) in `config.example.yaml`

**Checkpoint**: All six providers are dedicated and self-refreshing. Independently testable.

---

## Phase 6: User Story 4 - /api.json catalog endpoint (Priority: P2)

**Goal**: CLIProxyApiPlus serves a models.dev-format `/api.json` catalog built from the enriched registry, exposing `limit.context` / `limit.output` per model so downstream consumers stop guessing.

**Independent Test**: `go test ./sdk/api/handlers/openai/...`; `/api.json` matches the models.dev shape with `limit.context`/`limit.output`. Also `curl /api.json | jq '.cliproxy.models["k3-256k"].limit'` returns correct values.

### Tests for User Story 4

- [X] T013 [P] [US4] [U16][U17][U18] Unit test `/api.json` shape (models.dev format, `limit.context`/`limit.output`, namespaced keys) — create `sdk/api/handlers/openai/api_json_test.go`

### Implementation for User Story 4

- [X] T014 [US4] [U16][U17][U18] Implement `/api.json` handler building the catalog from `registry.GetGlobalRegistry().GetAvailableModels(...)` with `limit.context = ContextLength`, `limit.output = MaxCompletionTokens` — create `sdk/api/handlers/openai/api_json.go`
- [X] T015 [US4] [A4] Register `GET /api.json` route in `internal/api/server.go` `setupRoutes()` (alongside existing `v1.GET("/models", ...)`) — `v1.GET("/api.json", openaiHandlers.APIJSON)` at `internal/api/server.go`

**Checkpoint**: Catalog endpoint is authoritative. Independently testable.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Verify formatting, build, and end-to-end behavior across all stories.

### Acceptance gates (outer loop)

Each gate must be GREEN before its user story is considered complete. These are
the `A1`–`A6` acceptance behaviors from `tdd/test-list.md`, exercised through the
real HTTP entry points (`GET /v1/models`, `GET /api.json`).

- [X] T019 [A1] Write outer-loop acceptance test for SC-001: after connecting a provider, `GET /v1/models` returns the provider's reported `context_length` (non-zero) for a model it returns with `context_length` set — `sdk/api/handlers/openai/...` `httptest` (TestAcceptance_A1_ContextLengthReturned GREEN)
- [X] T020 [A2] Write outer-loop acceptance test for SC-002: for a free-model variant (`ocd/deepseek-v4-flash-free`), `GET /v1/models` returns the provider's real `context_length` (e.g. 1,000,000), not the static-table value (TestAcceptance_A2_FreeModelRealWindow GREEN)
- [X] T021 [A3] Write outer-loop acceptance test for SC-003: `GET /v1/models` and `GET /api.json` preserve namespaced ids verbatim (`ocd/`, `ocg-1/`, `z-ai/`, `kilo/`, `kiro-`, `k3`, `k3-256k`) (TestAcceptance_A3_NamespacedIDsPreserved GREEN)
- [X] T022 [A4] Write outer-loop acceptance test for SC-004: `GET /api.json` exposes `limit.context` and `limit.output` for every model that has a known window (TestAcceptance_A4_APIJSONLimitFields GREEN)
- [X] T023 [A5] Write outer-loop acceptance test for SC-005: after a provider fetch error (bad key/offline), `GET /v1/models` and `GET /api.json` still return last-known-good `context_length` (no zeroing) (TestAcceptance_A5_FetchErrorKeepsLastKnownGood GREEN)
- [X] T024 [A6] Write outer-loop acceptance test for SC-006: onboarding a new dedicated provider via `openai-compatibility` config (base-url + api-key) surfaces its live `context_length` with no code change (TestAcceptance_A6_OnboardingSurfacesLiveWindow GREEN)

- [X] T016 [P] Run `gofmt -w` on changed files and `go build -o cli-proxy-api ./cmd/server`; binary builds (81 MB), everything compiles
- [ ] T017 [P] Run `quickstart.md` validation: start server with openrouter + z.ai + kimi configured, `curl /v1/models` and `/api.json`, assert free-model `context_length` equals provider real value and namespaced ids preserved (MANUAL — needs live provider API keys; not run here)
- [X] T018 [P] Run full suite `go test ./...`; feature-scoped packages green. Remaining failures are pre-existing and feature-unrelated (verified via stash on clean `HEAD`): `internal/registry.TestGetAvailableModelsClaudeIncludesTokenLimits`, `internal/config` clone/home tests. No secret leakage in logs; no `log.Fatal` introduced.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup; BLOCKS US2/US3 (provider identifiers).
- **US1 (Phase 3)**: Independent of Foundational — can start after Setup (registry already emits the fields; handler fix alone is testable). MVP.
- **US2 (Phase 4)**: Depends on Foundational (T002) for provider identifiers.
- **US3 (Phase 5)**: Depends on US2 (reuses `FetchAndMerge`).
- **US4 (Phase 6)**: Depends on US2 (enriched registry supplies the limits).
- **Polish (Phase 7)**: Depends on all desired user stories complete.

### User Story Dependencies

- **US1 (P1)**: Start after Setup. No dependency on other stories — independently testable on its own (emits static registry values). 🎯 MVP.
- **US2 (P1)**: Start after Foundational. Independent of US1/US3/US4 for testing the fetcher in isolation.
- **US3 (P2)**: Start after US2. Reuses `FetchAndMerge`.
- **US4 (P2)**: Start after US2. Consumes enriched registry.

### Within Each User Story

- Tests MUST be written and FAIL before implementation.
- Models/services before endpoints; core implementation before integration.
- Story complete before moving to next priority.

### Parallel Opportunities

- T002 [P], T003 [P], T006 [P], T007 [P], T013 [P], T016-T018 [P] touch independent files and can run in parallel where sequencing allows.
- After Foundational completes, US1 and US2 can proceed in parallel (US1 handler fix vs US2 fetcher are different files).
- Within US2: T006 (test) and T007 (impl) are different files — test written first, then impl.

---

## Parallel Example: User Story 2

```bash
# Write the test first (must fail):
Task: "Unit test ProviderModelsFetcher.FetchAndMerge against a mock httptest /v1/models — create internal/registry/provider_models_test.go"
# Implement the fetcher (different file):
Task: "Create internal/registry/provider_models.go with ProviderModelsFetcher struct + FetchAndMerge"
# Then wire invocation + error safety:
Task: "Kick off FetchAndMerge on config load / account connect (best-effort, async)"
Task: "Make fetch error-safe: keep last-good on offline/401/timeout/missing"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (baseline capture)
2. Complete Phase 2: Foundational (provider constants) — quick
3. Complete Phase 3: US1 (emit `context_length` from `/v1/models`)
4. **STOP and VALIDATE**: `go test ./sdk/api/handlers/openai/...` + `curl /v1/models` shows `context_length`
5. Demo if ready — already a visible improvement over today's `0` values

### Incremental Delivery

1. Setup + Foundational → identifiers ready
2. Add US1 → `/v1/models` exposes context windows (MVP)
3. Add US2 → real free-model sizes override static table
4. Add US3 → six providers dedicated + self-refreshing
5. Add US4 → `/api.json` authoritative catalog
6. Each story adds value without breaking previous stories

### Parallel Team Strategy

1. Team completes Setup + Foundational together
2. Once Foundational done: Developer A → US1, Developer B → US2 (different files)
3. US3/US4 follow after US2 supplies `FetchAndMerge` + enriched registry

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Tests written first (T003, T006, T013) and must fail before implementation
- The handler fix (US1) is independently shippable even before the fetcher (US2) — it already improves on today's dropped fields
- No changes to `internal/translator/` (per AGENTS.md); no new top-level package
- Behavior markers `[U1]..[U18]` (unit) and `[A1]..[A6]` (acceptance) map each task
  to a line in `tdd/test-list.md` and are load-bearing: the TDD loop ticks a task's
  checkbox only when it can read a behavior id, and `/skill:speckit-implement`
  implements whatever is still unticked. Acceptance gates T019–T024 (A1–A6) must be
  GREEN before the corresponding story is complete.

## Blocker (RESOLVED 2026-08-30)

The unrelated WIP build break that blocked this feature is now resolved: the orphaned
`internal/api/modules/amp` and `internal/translator/*/gemini-cli` directories (which
referenced undefined symbols) were removed at the user's explicit request. `go build ./...`
is GREEN. All previously blocked tasks are now complete:

- T008, T010 (server-startup wiring of the refresher) — DONE via `startDedicatedProviderContextRefresher` in `cmd/server/main.go`.
- T015 (`GET /api.json` route) — DONE.
- T019–T024 (acceptance gates A1–A6) — all GREEN (`acceptance_test.go`).
- T001 (build baseline capture) is moot: `/v1/models` no longer drops `context_length`.
- T002 (hardcoded provider constants) remains intentionally NOT added: the implementation
  reads `name` dynamically from `openai-compatibility` config (`Config.DedicatedProviderConfigs`),
  so static constants would be unused speculative code.

`go test ./...` still shows pre-existing, feature-unrelated failures (verified on a
clean `HEAD` via stash): `internal/registry.TestGetAvailableModelsClaudeIncludesTokenLimits`
and three `internal/config` clone/home tests. The feature-scoped packages are green.
