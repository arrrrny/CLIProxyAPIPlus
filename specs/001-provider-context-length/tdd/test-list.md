---
feature: 001-provider-context-length
loop: outside-in
profile: .specify/memory/tdd-profile.md
spec_criteria: 6
planned_at: e4d96ae0
updated_at: e4d96ae0
suite_baseline: red
---

# Test List: Accurate model context-length via dedicated provider endpoints

## Outer loop: acceptance behaviors

One per acceptance criterion in `spec.md`. Each stays red until the feature works
end to end through its real entry point (the HTTP routes `GET /v1/models` and
`GET /api.json`).

| id  | behavior | traces | kind    | state    | test |
| --- | -------- | ------ | ------- | -------- | ---- |
| A1  | After connecting a provider, `GET /v1/models` returns the provider's reported `context_length` (non-zero) for a model it returns with one | SC-001 | example  | DONE | `sdk/api/handlers/openai/acceptance_test.go::TestAcceptance_A1_ContextLengthReturned` |
| A2  | For a free-model variant (e.g. `ocd/deepseek-v4-flash-free`), `GET /v1/models` returns the provider's real `context_length` (e.g. 1,000,000), not the static-table value | SC-002 | example  | DONE | `sdk/api/handlers/openai/acceptance_test.go::TestAcceptance_A2_FreeModelRealWindow` |
| A3  | `GET /v1/models` and `GET /api.json` preserve namespaced ids verbatim (`ocd/`, `ocg-1/`, `z-ai/`, `kilo/`, `kiro-`, `k3`, `k3-256k`) | SC-003 | example  | DONE | `sdk/api/handlers/openai/acceptance_test.go::TestAcceptance_A3_NamespacedIDsPreserved` |
| A4  | `GET /api.json` exposes `limit.context` and `limit.output` for every model that has a known window | SC-004 | approval | DONE | `sdk/api/handlers/openai/acceptance_test.go::TestAcceptance_A4_APIJSONLimitFields` |
| A5  | After a provider fetch error (bad key/offline), `GET /v1/models` and `GET /api.json` still return the last-known-good `context_length` (no zeroing) | SC-005 | example  | DONE | `sdk/api/handlers/openai/acceptance_test.go::TestAcceptance_A5_FetchErrorKeepsLastKnownGood` |
| A6  | Onboarding a new dedicated provider via `openai-compatibility` config (base-url + api-key) surfaces its live `context_length` with no code change | SC-006 | example  | DONE | `sdk/api/handlers/openai/acceptance_test.go::TestAcceptance_A6_OnboardingSurfacesLiveWindow` |

## Inner loop: unit behaviors

Grouped by the component from `plan.md` that owns them. Each line names one
observable result.

### `sdk/api/handlers/openai/openai_handlers.go` (OpenAIModels)

| id  | behavior | traces | kind | state    | test |
| --- | -------- | ------ | ---- | -------- | ---- |
| U1  | `OpenAIModels` includes `context_length` in the entry when the registry value is > 0 | FR-001, SC-001 | example | DONE | `sdk/api/handlers/openai/openai_handlers_test.go::TestOpenAIModels_EmitsContextLengthWhenPositiveAndOmitsWhenZero` |
| U2  | `OpenAIModels` includes `max_completion_tokens` in the entry when > 0 | FR-001 | example | DONE | `sdk/api/handlers/openai/openai_handlers_test.go::TestOpenAIModels_EmitsContextLengthWhenPositiveAndOmitsWhenZero` |
| U3  | `OpenAIModels` omits `context_length` from the entry when the registry value is `0` (does not emit `0`) | FR-001 | example | DONE | `sdk/api/handlers/openai/openai_handlers_test.go::TestOpenAIModels_EmitsContextLengthWhenPositiveAndOmitsWhenZero` |
| U4  | `OpenAIModels` omits `max_completion_tokens` when the registry value is `0` | FR-001 | example | DONE | `sdk/api/handlers/openai/openai_handlers_test.go::TestOpenAIModels_EmitsContextLengthWhenPositiveAndOmitsWhenZero` |
| U5  | `OpenAIModels` includes `input_token_limit` and `output_token_limit` when > 0 | FR-001 | example | OPEN | **DEFERRED — key casing open question** (see Notes below) |
| U6  | The `?client_version=` codex path still builds its own response (unchanged) — regression guard | plan T005 | characterization | DONE | `sdk/api/handlers/openai/codex_client_models_test.go` |

### `internal/registry/provider_models.go` (NEW: ProviderModelsFetcher)

| id  | behavior | traces | kind | state    | test |
| --- | -------- | ------ | ---- | -------- | ---- |
| U7  | `FetchAndMerge` GETs `<baseURL>/v1/models` with `Authorization: Bearer <apiKey>` and sets `ContextLength`/`MaxCompletionTokens` for each returned id | FR-002, FR-006 | example | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_FetchAndMergeSetsContextLength` |
| U8  | `FetchAndMerge` merges keyed by the exact provider-returned id, preserving namespaces (`ocd/deepseek-v4-flash-free` stays verbatim) | FR-005, SC-003 | example | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_MergesByExactID` |
| U9  | A fetched `context_length` > 0 overrides the existing static-table value in the registry | FR-003 | example | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_OverridesStaticWhenPositive` |
| U10 | A fetched `context_length` of `0` (or missing) leaves the existing value retained, never zeroed or guessed | FR-003, FR-008 | example | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_KeepsExistingWhenZeroOrMissing` |
| U11 | A fetch error (401 / offline / timeout) keeps last-known-good values, logs, and does not wipe | FR-008 | example | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_ErrorKeepsLastKnownGood` |
| U12 | Base URLs both with and without a trailing `/v1` resolve to `<base>/v1/models` | FR-002 | example | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_NormalizesBaseURL` |

### `internal/constant/constant.go` + `internal/config/config.go` (dedicated providers)

| id  | behavior | traces | kind | state    | test |
| --- | -------- | ------ | ---- | -------- | ---- |
| U13 | A model registered through `openai-compatibility` with `name: openrouter` carries `Type`/`OwnedBy` = `openrouter`, distinguishable from generic `openai` | FR-004 | example | DONE | `internal/config/provider_configs_test.go::TestDedicatedProviderConfigs_MapsEnabledProviders` |

### `internal/registry/model_updater.go` (periodic refresh)

| id  | behavior | traces | kind | state    | test |
| --- | -------- | ------ | ---- | -------- | ---- |
| U14 | On startup / account-connect and on a configurable ticker (default 6h) `FetchAndMerge` re-runs for all dedicated providers | FR-007 | example | DONE | `internal/registry/provider_refresher_test.go::TestProviderRefresher_StartRefreshesImmediatelyAndPeriodically` |
| U15 | A failed refresh keeps cached values and retries on the next tick | FR-007, FR-008 | example | DONE | `internal/registry/provider_refresher_test.go::TestProviderRefresher_FailedRefreshKeepsCachedAndRetries` |

### `sdk/api/handlers/openai/api_json.go` (NEW: /api.json handler)

| id  | behavior | traces | kind | state    | test |
| --- | -------- | ------ | ---- | -------- | ---- |
| U16 | `GET /api.json` returns `cliproxy.models[<id>].limit.context` = `ContextLength` and `limit.output` = `MaxCompletionTokens` | FR-009, SC-004 | approval | DONE | `sdk/api/handlers/openai/api_json_test.go::TestAPIJSON_ExposesLimitContextOutputForKnownWindows` |
| U17 | For a model with `context_length` == 0 (unknown window), `/api.json` omits `limit` rather than emitting `0` | FR-009, SC-004 | example | DONE | `sdk/api/handlers/openai/api_json_test.go::TestAPIJSON_ExposesLimitContextOutputForKnownWindows` |
| U18 | `/api.json` keys preserve namespaced ids exactly | SC-003 | example | DONE | `sdk/api/handlers/openai/api_json_test.go::TestAPIJSON_ExposesLimitContextOutputForKnownWindows` |

## Invariants and edge cases still to place

- Concurrent/overlapping refreshes of the same provider must not wipe values (idempotent merge). Belongs under `provider_models.go` once the merge is implemented; add as a unit behavior if a race surfaces.
- A provider returning a model id that already exists from the static table must merge (not duplicate) — covered by U8/U9 once the registry merge is observable.

## Out of scope

- Downstream Quotio `ProxyBridge` static-table removal: separate repo (Swift), recorded as a follow-up in the spec. No test here.
- The `?client_version=` codex path behavior change: explicitly unchanged (guard U6 only).
- Models without a known window: by design omitted from `/api.json` limits (U17), not tested for guessing.
- Performance / load behavior of the refresh ticker: no requirement, no test.

## Verification commands

Copied verbatim from `.specify/memory/tdd-profile.md` at planning time, so this
file is readable on its own:

- Single test: `go test {file} -run '^{name}$' -v -count=1`
- Full suite: `go test ./...`
- Coverage: `go test {file} -cover`

**Guard (from profile):** `go test {file} -run '^{name}$'` exits 0 when the name
matches nothing ("no tests to run"). Always confirm `--- PASS: {name}` or
`--- FAIL: {name}` appears in the `-v` output; absence means a mistyped name and a
false green. `suite_baseline` is `red` — not because the module fails to build (the
`internal/api/modules/amp` and `internal/translator/*/gemini-cli` build break is
now resolved: those directories were orphaned and removed; `go build ./...` is
green), but because the module still contains pre-existing failing tests that are
unrelated to this feature and were verified to fail on a clean `HEAD` (stashed):
`internal/registry.TestGetAvailableModelsClaudeIncludesTokenLimits` (expects
`max_input_tokens` keys the registry does not emit), and `internal/config`
`TestCloneForRuntimeDeepCopiesConfig` / `TestCloneForRuntimeDoesNotShareReferenceFields`
/ `TestParseConfigBytesIgnoresHomeConfig` (pre-existing clone/home deep-copy gaps).
The feature-scoped packages `sdk/api/handlers/openai`, `internal/registry`
(unit behaviors U7–U15), and `internal/config` (`DedicatedProviderConfigs`) are all
green. The loop's red/green signal is reliable for the feature's own behaviors.

## Notes

- **U5 (input_token_limit / output_token_limit) — OPEN question.** `ModelInfo` has
  `InputTokenLimit`/`OutputTokenLimit` fields, but `GetAvailableModels` serializes
  them as camelCase (`inputTokenLimit`/`outputTokenLimit`) at
  `internal/registry/model_registry.go:1260-1264`, whereas the spec/test list and the
  originating doc (`docs/api-json-context-length-spec.md` §4.1) expect snake_case
  `input_token_limit`/`output_token_limit`. FR-001 only requires `context_length` +
  `max_completion_tokens`, which U1–U4 cover. The camelCase-vs-snake_case key for the
  optional input/output limits is unresolved; implement after confirming the
  downstream-expected key. T003/T004 remain open until U5 is resolved.
- **Pre-existing registry RED is unrelated to this feature.** The loop runs scoped on
  `sdk/api/handlers/openai` (green) and `internal/registry` (red only due to
  `TestGetAvailableModelsClaudeIncludesTokenLimits`, which expects `max_input_tokens`/
  `max_tokens` — keys the registry does not emit; separate from `context_length`).
  Verified pre-existing via stash (`git stash -u`, run on clean `HEAD`).
- **Scoped baseline:** `go test ./sdk/api/handlers/openai/...` is green. `go build ./...`
  is now green (amp/gemini-cli build break removed).
- **U13 — unit vs end-to-end.** The server-independent unit (`Config.DedicatedProviderConfigs` returns a `ProviderConfig` keyed by the openai-compatibility `Name`, so the registry override is distinguishable from generic `openai`) is proven by `provider_configs_test.go`. The deeper assertion that a *registered* `ModelInfo.Type`/`OwnedBy` equals the provider `name` is exercised by acceptance criterion A6 (`GET /v1/models` after onboarding a dedicated provider), now GREEN (`TestAcceptance_A6_OnboardingSurfacesLiveWindow`). U13 is DONE at the unit level; A6 completes the end-to-end path.
- **US3 refresher (`ProviderModelsRefresher`) is the U14/U15 component**, tested in
  `provider_refresher_test.go`. It is now wired into server startup (T008/T010:
  `startDedicatedProviderContextRefresher` in `cmd/server/main.go`, invoked from both
  the standalone TUI path via `startModelCatalogUpdaters` and the main proxy path), and
  `GET /api.json` is served (T015: `v1.GET("/api.json", openaiHandlers.APIJSON)` in
  `internal/api/server.go`). Both build under the now-green `go build ./...`.
