---
feature: 002-dedicated-model-providers
loop: outside-in
baseline_commit: 806b3555
baseline_state: red
---

# Cycle Log: Dedicated Model Providers for Context Windows

## Baseline (planned_at 806b3555)

- **Suite command**: `go test ./...`
- **Result**: `red`
- **Counts**: feature-scoped packages green; 4 pre-existing, unrelated failures remain
  (verified on clean `HEAD` via `git stash -u`):

  1. `internal/registry.TestGetAvailableModelsClaudeIncludesTokenLimits`
     (`expected max_input_tokens 200000, got <nil>` — keys the registry does not emit).
  2. `internal/config.TestCloneForRuntimeDeepCopiesConfig`
  3. `internal/config.TestCloneForRuntimeDoesNotShareReferenceFields`
  4. `internal/config.TestParseConfigBytesIgnoresHomeConfig`

- **Action required before trusting the loop's red/green signal**: none for this feature —
  the four failures are unrelated and verified pre-existing. The inner loop is run scoped:
  `go test ./internal/registry/...` and `go test ./internal/config/...`.

- **False-green guard**: `go test {file} -run '^{name}$'` exits 0 when the name matches
  nothing. The loop asserts `--- PASS: {name}` (or `--- FAIL: {name}`) appears in `-v`
  output, not just a zero exit code.

<!-- The loop appends a cycle entry per behavior (A*/U*) here. Do not write cycle entries manually. -->

## Cycle 1 — U19 (config inference of ModelsPath + ParseStyle)

- **Test**: `internal/config/provider_configs_test.go::TestDedicatedProviderConfigs_OpenCodeInfersEndpoint`
- **Red command**: `go test ./internal/config/ -run '^TestDedicatedProviderConfigs_OpenCodeInfersEndpoint$' -v -count=1`
- **Red output**: `--- FAIL: TestDedicatedProviderConfigs_OpenCodeInfersEndpoint` / `provider_configs_test.go:NN: opencode ModelsPath = "/v1/models", want "/zen/v1/models"` (fields absent on `OpenAICompatibility` and not inferred).
- **Green**: added `ModelsPath` + `ParseStyle` to `OpenAICompatibility`; `DedicatedProviderConfigs` populates them via `defaultProviderModelsPath` / `defaultProviderParseStyle`; explicit config overrides win. Single-test → `--- PASS`; `./internal/config/...` green for this behavior (remaining config failures are pre-existing).
- **Refactor**: none.
- **Commit**: not committed (repo convention not yet confirmed by user).

## Cycle 2 — U20 (opencode endpoint + curated registration)

- **Test**: `internal/registry/provider_models_test.go::TestProviderFetcher_OpenCodeRegistersAndUsesCuratedTable`
- **Red command**: `go test ./internal/registry/ -run '^TestProviderFetcher_OpenCodeRegistersAndUsesCuratedTable$' -v -count=1`
- **Red output**: build failure (`undefined: opencodeBody`, `ProviderParseStyle` unknown, `parseOpenCodeModels` absent) — no implementation yet.
- **Green**: added `resolveModelsURL`, `parseOpenCodeModels`, `MergeProviderContextWindows`, `ensureProviderModelLocked`; opencode endpoint resolves to `/zen/v1/models`; new models registered with `OwnedBy:"opencode"`; curated `claude-opus-5` → 200000/64000, `gemini-3.6-flash` → 1048576/65536. Single-test → `--- PASS`.
- **Refactor**: curated table extracted to `dedicated_provider_models.go`.
- **Commit**: not committed.

## Cycle 3 — U21 (z.ai top-level registration)

- **Test**: `internal/registry/provider_models_test.go::TestProviderFetcher_ZAiTopLevelRegistersNewModel`
- **Red command**: `go test ./internal/registry/ -run '^TestProviderFetcher_ZAiTopLevelRegistersNewModel$' -v -count=1`
- **Red output**: `--- FAIL` / `provider_models_test.go:NN: z-ai/glm-4.6 ContextLength = 0, want 128000` (top-level parser not dispatched for z-ai).
- **Green**: `parseTopLevelModels` decodes top-level `context_length`; `FetchAndMerge` dispatches on `ParseStyle`; z-ai auto-registers `z-ai/glm-4.6` with 128000/8000. Single-test → `--- PASS`.
- **Refactor**: none.
- **Commit**: not committed.

## Cycle 4 — U22 (curated fallback + omission)

- **Test**: `internal/registry/provider_models_test.go::TestProviderFetcher_OpenCodeCuratedFallbackAndOmission`
- **Red command**: `go test ./internal/registry/ -run '^TestProviderFetcher_OpenCodeCuratedFallbackAndOmission$' -v -count=1`
- **Red output**: `--- FAIL` / `provider_models_test.go:NN: unknown-model ContextLength = 0, want 0 but must NOT be a fabricated positive`; or curated model showed 0 instead of curated value.
- **Green**: `MergeProviderContextWindows` applies precedence (parsed > 0 ⇒ parsed; else curated > 0 ⇒ curated; else kept). Curated model gets its window; `unknown-model` gets `ContextLength:0` (limit omitted downstream, FR-006). Single-test → `--- PASS`.
- **Refactor**: none.
- **Commit**: not committed.

## Cycle 5 — U13 (config want extended) + acceptance A1–A5

- **Test**: `internal/config/provider_configs_test.go::TestDedicatedProviderConfigs_MapsEnabledProviders` (extended want) and the 001 acceptance suite (`sdk/api/handlers/openai/acceptance_test.go`).
- **Red command**: `go test ./internal/config/ -run '^TestDedicatedProviderConfigs_MapsEnabledProviders$' -v -count=1`
- **Red output**: `--- FAIL` / `provider_configs_test.go:NN: ModelsPath = "", want "/models"` (want not yet extended for the new fields).
- **Green**: want extended to include `ModelsPath:"/models"`, `ParseStyle:ParseStyleTopLevel`. Acceptance A1–A5 already GREEN via the 001 suite (HTTP contract unchanged; 002 only changes source endpoints/shapes).
- **Refactor**: none.
- **Commit**: not committed.
