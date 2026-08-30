---
feature: 001-provider-context-length
loop: outside-in
baseline_commit: e4d96ae0
baseline_state: red
---

# Cycle Log: Accurate model context-length via dedicated provider endpoints

## Baseline (planned_at e4d96ae0)

- **Suite command**: `go test ./...`
- **Result**: `red`
- **Counts**: build failure blocks the suite; two independent red sources, both
  pre-existing and unrelated to this feature (neither touches
  `sdk/api/handlers/openai` or `internal/registry/provider_models.go`):

  1. `go build ./...` fails: `internal/api/modules/amp/model_mapping.go` references
     undefined symbols `config.AmpModelMapping` / `config.AmpUpstreamAPIKeyEntry`
     (dirty working tree / WIP on the `amp` module).
  2. Pre-existing failing test in `internal/registry`:
     `TestGetAvailableModelsClaudeIncludesTokenLimits`
     (observed `expected max_input_tokens 200000, got <nil>` during setup).

- **Action required before trusting the loop's red/green signal**: restore a green
  baseline. The two feature packages (`sdk/api/handlers/openai`,
  `internal/registry`) still compile and test independently, so the inner loop can
  run scoped: `go test ./sdk/api/handlers/openai/...` and
  `go test ./internal/registry/...`.

- **False-green guard**: `go test {file} -run '^{name}$'` exits 0 when the name
  matches nothing. The loop must assert `--- PASS: {name}` (or `--- FAIL: {name}`)
  appears in `-v` output, not just a zero exit code.

<!-- The loop appends a cycle entry per behavior (A*/U*) here. Do not write cycle entries manually. -->

## Cycle 1 — U1 + U3 (context_length present/absent)

- **Test**: `sdk/api/handlers/openai/openai_handlers_test.go::TestOpenAIModels_EmitsContextLengthWhenPositiveAndOmitsWhenZero`
- **Red command**: `go test ./sdk/api/handlers/openai/ -run '^TestOpenAIModels_EmitsContextLengthWhenPositiveAndOmitsWhenZero$' -v -count=1`
- **Red output**: `--- FAIL: TestOpenAIModels_EmitsContextLengthWhenPositiveAndOmitsWhenZero` / `openai_handlers_test.go:57: with-window context_length = <nil> (present=false), want 200000`
- **Green**: `OpenAIModels` now carries `model["context_length"]` into the entry via a new `positiveInt` helper (int and float64), only when > 0. `go test ./sdk/api/handlers/openai/ -run '^TestOpenAIModels_EmitsContextLengthWhenPositiveAndOmitsWhenZero$' -v -count=1` → `--- PASS`, full `./sdk/api/handlers/openai/...` still green.
- **Refactor**: none needed; `positiveInt` extracted for reuse by U2/U4.
- **Commit**: not committed (`--no-commit`); repo convention not confirmed.
- **Notes**: valid red (missing-behavior assertion), not a build/import failure.

## Cycle 2 — U2 + U4 (max_completion_tokens present/absent)

- **Test**: same as Cycle 1 (extended with `max_completion_tokens` assertions for both models).
- **Red command**: same single-test command.
- **Red output**: `--- FAIL: TestOpenAIModels_EmitsContextLengthWhenPositiveAndOmitsWhenZero` / `openai_handlers_test.go:61: with-window max_completion_tokens = <nil> (present=false), want 64000`
- **Green**: `OpenAIModels` now carries `model["max_completion_tokens"]` when > 0 (same `positiveInt` rule). Single-test → `--- PASS`; full `./sdk/api/handlers/openai/...` green.
- **Refactor**: none.
- **Commit**: not committed (`--no-commit`).
- **Notes**: U5 (input/output token limits, camelCase key conflict) deferred — see test-list Notes. T003/T004 remain open pending U5.

## Cycle 3 — US4 (U16 / U17 / U18: /api.json limit fields + namespaced keys + omit-when-absent)

- **Test**: `sdk/api/handlers/openai/api_json_test.go::TestAPIJSON_ExposesLimitContextOutputForKnownWindows`
- **Red command**: `go test ./sdk/api/handlers/openai/ -run '^TestAPIJSON_ExposesLimitContextOutputForKnownWindows$' -v -count=1`
- **Red output**: build failure on first write (`api_json.go:7: "internal/registry" imported and not used`); then after removing the unused import the seeded stub returned `nil` and the test unmarshalled an empty body (`api_json_test.go:50: unmarshal /api.json: unexpected end of JSON input`). Both are missing-behavior signals (no `buildCatalog` yet).
- **Green**: added `APIJSON` handler + `buildCatalog` (models.dev format, `cliproxy` namespace), `modelName` (display_name→id fallback) and `toIntValue` (int/float64→int) helpers. `limit` is omitted entirely when `context_length` is absent/zero (U17); `limit.output` carries `max_completion_tokens` only when present and > 0 (U18); top-level key is `cliproxy` not `openai` (U16). `go test ./sdk/api/handlers/openai/ -run '^TestAPIJSON_ExposesLimitContextOutputForKnownWindows$' -v -count=1` → `--- PASS`; full `./sdk/api/handlers/openai/...` green.
- **Refactor**: none.
- **Commit**: not committed (`--no-commit`).
- **Notes**: top-level provider key decided as `cliproxy` (matches the vendor namespace used elsewhere in the server); U5 still deferred.

## Cycle 4 — US2 (U7 / U8 / U9 / U10 / U11 / U12: ProviderModelsFetcher)

- **Test**: `internal/registry/provider_models_test.go` (6 funcs, one per behavior: `TestProviderFetcher_FetchAndMergeSetsContextLength`, `_MergesByExactID`, `_OverridesStaticWhenPositive`, `_KeepsExistingWhenZeroOrMissing`, `_ErrorKeepsLastKnownGood`, `_NormalizesBaseURL`)
- **Red command**: `go test ./internal/registry/ -run '^TestProviderFetcher_' -v -count=1`
- **Red output**: build failure (`undefined: NewProviderModelsFetcher` / `ProviderConfig`) — no implementation yet.
- **Green**: added `internal/registry/provider_models.go` with `ProviderModelsFetcher` + `ProviderConfig` + `ProviderModelLimit`, `FetchAndMerge` (GET `<baseURL>/v1/models` with Bearer, tolerant of trailing `/v1`, decodes OpenAI-style `data[]`), and `ModelRegistry.MergeProviderContextLengths` (keyed by exact id, overrides only when > 0, never zeroes, invalidates the available-models cache). A transport/non-200 error returns without wiping. `go test ./internal/registry/ -run '^TestProviderFetcher_' -v -count=1` → all 6 `--- PASS`; the only failing test in `./internal/registry` remains the pre-existing, unrelated `TestGetAvailableModelsClaudeIncludesTokenLimits` (keys `max_input_tokens`/`max_tokens`, not this feature's `context_length`).
- **Refactor**: extracted `providerInfo` helper for per-provider override sync; one bug fixed during green — `ProviderModelLimit` needed explicit `json:"context_length"`/`json:"max_completion_tokens"` tags (default decoder skips the underscore).
- **Commit**: not committed (`--no-commit`).
- **Notes**: T006/T007/T009 ticked DONE (U7–U12). T008 (kick-off on config load) and US3 ticker (U14/U15) remain; they need the dedicated-provider config source.

## Cycle 5 — US3 (U13 config mapping / U14 / U15 periodic refresher)

- **Test**: `internal/registry/provider_refresher_test.go` (`TestProviderRefresher_StartRefreshesImmediatelyAndPeriodically` for U14, `TestProviderRefresher_FailedRefreshKeepsCachedAndRetries` for U15) and `internal/config/provider_configs_test.go::TestDedicatedProviderConfigs_MapsEnabledProviders` for U13.
- **Red command**: `go test ./internal/registry/ -run '^TestProviderRefresher_' -v -count=1` and `go test ./internal/config/ -run '^TestDedicatedProviderConfigs_' -v -count=1`
- **Red output**: `undefined: NewProviderModelsRefresher` (registry) — no implementation; config helper had no test initially (added green alongside its impl).
- **Green**: added `ProviderModelsRefresher` (`NewProviderModelsRefresher`, `RefreshNow`, `Start`) to `internal/registry/provider_models.go` — immediate refresh on `Start` plus a `time.Ticker` at `DefaultProviderRefreshInterval` (6h), best-effort across providers, errors logged and never wipe cached values. Added `Config.DedicatedProviderConfigs()` to `internal/config/config.go` mapping enabled `openai-compatibility` blocks (Name/BaseURL/first APIKey) to `[]registry.ProviderConfig`, skipping disabled / no-Name / no-BaseURL. Registry tests pass under `-race`; config test passes. T011 ticked DONE (U14/U15); U13 ticked DONE at unit level (full Type/OwnedBy registration tracked under A6).
- **Refactor**: none.
- **Commit**: not committed (`--no-commit`).
- **Notes**: the refresher COMPONENT is done and testable scoped, but its *wiring* into server startup (T008, T010, T011 server half) and the `GET /api.json` route (T015) need `go build ./...`, which is blocked by the pre-existing `internal/api/modules/amp` build break. Those remain open until the baseline is restored. U5 still deferred.

<!-- The loop appends a cycle entry per behavior (A*/U*) here. Do not write cycle entries manually. -->
