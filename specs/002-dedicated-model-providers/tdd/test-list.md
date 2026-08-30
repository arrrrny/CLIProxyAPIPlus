---
feature: 002-dedicated-model-providers
loop: outside-in
profile: .specify/memory/tdd-profile.md
spec_criteria: 4
planned_at: 806b3555
updated_at: 806b3555
suite_baseline: red
---

# Test List: Dedicated Model Providers for Context Windows

## Outer loop: acceptance behaviors

One per acceptance criterion in `spec.md`. Each stays red until the feature works end to
end through its real entry point (`GET /v1/models`, `GET /api.json`). The HTTP contract
is established by feature 001's acceptance suite (`sdk/api/handlers/openai/acceptance_test.go`);
002's outer behaviors confirm the per-provider **endpoint + shape** change flows through.

| id  | behavior | traces | kind    | state    | test |
| --- | -------- | ------ | ------- | -------- | ---- |
| A1  | `openrouter` fetched from `…/api/v1/models` (top-level) → `/api.json` `limit.context` is the provider's reported window (non-zero) | SC-001, FR-002 | example  | DONE | `sdk/api/handlers/openai/acceptance_test.go::TestAcceptance_A1_ContextLengthReturned` |
| A2  | `opencode`/`opencode-go` fetched from `…/zen/v1/models`, window supplied by curated table → `/api.json` `limit.context` correct for `claude-opus-5` | SC-001, FR-002, FR-008 | example  | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_OpenCodeRegistersAndUsesCuratedTable` |
| A3  | `z-ai` fetched from its real endpoint (`/api/v1/models`, top-level) → `/api.json` `limit.context` correct | SC-001, FR-002 | example  | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_ZAiTopLevelRegistersNewModel` |
| A4  | `kilo` nested `top_provider.context_length` surfaced on `/api.json` (inherited from 001) | SC-001, FR-003 | example  | DONE | `sdk/api/handlers/openai/acceptance_test.go::TestAcceptance_A6_OnboardingSurfacesLiveWindow` |
| A5  | A model with no known window → `/api.json` omits `limit` (never `0`) | SC-002, FR-006 | example  | DONE | `sdk/api/handlers/openai/api_json_test.go::TestAPIJSON_ExposesLimitContextOutputForKnownWindows` |

## Inner loop: unit behaviors

Grouped by the component from `plan.md` that owns them.

### `internal/config/config.go` (DedicatedProviderConfigs)

| id  | behavior | traces | kind | state    | test |
| --- | -------- | ------ | ---- | -------- | ---- |
| U13 | `DedicatedProviderConfigs` populates `ModelsPath`/`ParseStyle` for each enabled provider | FR-001, FR-002 | example | DONE | `internal/config/provider_configs_test.go::TestDedicatedProviderConfigs_MapsEnabledProviders` |
| U19 | opencode → `ModelsPath:/zen/v1/models` + `ParseStyle:opencode`; z-ai → `/models` + top-level; an explicit `models-path`/`parse-style` config override wins | FR-001, FR-002, FR-003 | example | DONE | `internal/config/provider_configs_test.go::TestDedicatedProviderConfigs_OpenCodeInfersEndpoint` |

### `internal/registry/provider_models.go` (fetch + parse + merge)

| id  | behavior | traces | kind | state    | test |
| --- | -------- | ------ | ---- | -------- | ---- |
| U20 | opencode endpoint (`/zen/v1/models`) → new models registered with `OwnedBy:"opencode"`; curated `claude-opus-5` → 200000/64000, `gemini-3.6-flash` → 1048576/65536; parsed value wins over curated | FR-002, FR-004, FR-008 | example | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_OpenCodeRegistersAndUsesCuratedTable` |
| U21 | z-ai top-level endpoint auto-registers a new model `z-ai/glm-4.6` with `ContextLength:128000`, `MaxCompletionTokens:8000` | FR-002, FR-004 | example | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_ZAiTopLevelRegistersNewModel` |
| U22 | curated fallback: a curated opencode model receives its window; a model absent from both live response and curated table gets `ContextLength:0` (not fabricated) | FR-006, FR-008 | example | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_OpenCodeCuratedFallbackAndOmission` |

### Inherited from feature 001 (still DONE, unchanged by 002)

| id  | behavior | traces | kind | state    | test |
| --- | -------- | ------ | ---- | -------- | ---- |
| U7  | `FetchAndMerge` GETs the provider URL with `Authorization: Bearer` and sets windows per returned id | FR-004 | example | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_FetchAndMergeSetsContextLength` |
| U12 | `resolveModelsURL` with explicit `ModelsPath` appends directly; empty path ⇒ legacy `/v1/models` normalization | FR-002 | example | DONE | `internal/registry/provider_models_test.go::TestProviderFetcher_NormalizesBaseURL` |
| U14 | `ProviderModelsRefresher` re-runs `FetchAndMerge` for all dedicated providers on startup + ticker | FR-007 | example | DONE | `internal/registry/provider_refresher_test.go::TestProviderRefresher_StartRefreshesImmediatelyAndPeriodically` |

## Invariants and edge cases still to place

- Concurrent/overlapping refreshes of the same provider must not wipe values (idempotent merge) — covered by U7/U14 merge semantics; add a race unit behavior only if one surfaces.
- A provider returning a model id that already exists from the static table must merge (not duplicate) — covered by U20 merge path.

## Out of scope

- Downstream Quotio `ProxyBridge` static-table removal: separate repo, recorded in spec Assumptions.
- `kilo` fetch internals: reused as-is (`FetchKiloModels`), not changed by 002.
- z.ai exact response shape beyond top-level `context_length`: assumed top-level; corrected later as a config/data change if the live probe differs (FR-002 architecture supports it).
- Performance/load behavior of the refresh ticker: no requirement, no test.

## Verification commands

Copied verbatim from `.specify/memory/tdd-profile.md` at planning time:

- Single test: `go test {file} -run '^{name}$' -v -count=1`
- Full suite: `go test ./...`
- Coverage: `go test {file} -cover`

**Guard (from profile):** `go test {file} -run '^{name}$'` exits 0 when the name matches
nothing ("no tests to run"). Always confirm `--- PASS: {name}` or `--- FAIL: {name}`
appears in the `-v` output; absence means a mistyped name and a false green.

`suite_baseline` is `red` because the module still contains 4 pre-existing failing tests
that are unrelated to this feature and verified to fail on a clean `HEAD` (stashed):
`internal/registry.TestGetAvailableModelsClaudeIncludesTokenLimits`, and
`internal/config` `TestCloneForRuntimeDeepCopiesConfig` /
`TestCloneForRuntimeDoesNotShareReferenceFields` /
`TestParseConfigBytesIgnoresHomeConfig`. The feature-scoped packages
`internal/registry` (U7, U12, U14, U20–U22), `internal/config` (U13, U19), and
`internal/constant` are all green.

## Notes

- **Feature 002 is an extension of 001**, not a rewrite. 001 built `/api.json`, the
  generic `ProviderModelsFetcher`, and 6-provider onboarding. 002 adds the per-provider
  `ModelsPath` + `ParseStyle` so the *correct* endpoint and response shape are used — the
  original point of the spec ("they don't all use the same baseurl/v1/models").
- **Loop status**: the loop ran during implementation; every behavior (U13, U19–U22,
  A1–A5) has a failing-then-passing test. Behaviors are recorded DONE with their real test
  references.
- **z.ai shape unconfirmed**: assumed top-level `context_length` per FR-002
  architecture; if the live probe differs, the fix is a data/parse-style change, not a
  code change.
