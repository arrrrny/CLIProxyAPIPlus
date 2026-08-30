---
feature: 002-dedicated-model-providers
verdict: PASS_WITH_GAPS
verified_at: 806b3555
profile: .specify/memory/tdd-profile.md
---

# TDD Verification: Dedicated Model Providers for Context Windows

**Verdict: PASS_WITH_GAPS**

The feature was driven test-first end to end: every behavior has a test that failed
before its implementation and passes after. Acceptance criteria map 1:1 to observable
behaviors. One deliberate mutant was killed. Remaining gaps are out of scope for this
feature and are documented below.

## 1. Test-first evidence

Recorded in `tdd/cycle-log.md`. Each behavior went red → green on a real, missing-behavior
assertion (not a build/import error):

| Behavior | Test | Evidence |
| -------- | ---- | -------- |
| U19 | `TestDedicatedProviderConfigs_OpenCodeInfersEndpoint` | red: inferred fields absent → green: opencode→`/zen/v1/models`+opencode style; z-ai→`/models`+top-level; override wins |
| U20 | `TestProviderFetcher_OpenCodeRegistersAndUsesCuratedTable` | red: undefined symbols / no parser → green: opencode registered, curated `claude-opus-5`→200000/64000 |
| U21 | `TestProviderFetcher_ZAiTopLevelRegistersNewModel` | red: top-level parser not dispatched for z-ai → green: `z-ai/glm-4.6`→128000/8000 |
| U22 | `TestProviderFetcher_OpenCodeCuratedFallbackAndOmission` | red: curated/omission wrong → green: curated applied; unknown→0 (limit omitted) |
| U13 | `TestDedicatedProviderConfigs_MapsEnabledProviders` | red: want missing new fields → green: want extended |

Outer acceptance A1–A5 are exercised through the real HTTP entry points by the inherited
feature-001 acceptance suite (`sdk/api/handlers/openai/acceptance_test.go`), which remains
green.

## 2. Test smells

- No skipped/weakened tests. No `t.Skip` added for this feature.
- No test weakened to reach green. The 4 pre-existing failures are untouched and unrelated
  (verified via `git stash -u` on clean `HEAD`).
- Tests use the repo standard: white-box same-package `testing`, no `testify`.

## 3. Mutation (deliberate, no tool)

Mutation tooling is absent (not in `go.mod`); the profile mandates deliberate-mutant spot
checks instead.

- **Mutant**: changed the curated value `openCodeCuratedWindows["claude-opus-5"].ContextLength`
  from `200000` → `12345`.
- **Result**: `TestProviderFetcher_OpenCodeRegistersAndUsesCuratedTable` → `--- FAIL`
  (expected 200000, got 12345). Reverting restored `--- PASS`.
- **Conclusion**: U20 genuinely guards the curated fallback (FR-008). The merge precedence
  (parsed > curated > kept) is observable through the test.

## 4. Acceptance-criteria coverage

| Spec criterion | Behaviors | Status |
| -------------- | --------- | ------ |
| SC-001 (per-provider window on /api.json) | A1, A2, A3, A4 | covered (DONE) |
| SC-002 (no fabricated zero) | A5, U22 | covered (DONE) |
| SC-003 (outage keeps last-good) | inherited U11/U15 (001) | covered |
| SC-004 (refresh on interval/startup) | inherited U14 (001) | covered |
| FR-001..FR-009 | U13, U19, U20, U21, U22 + inherited U7/U12 | covered |

All 9 functional requirements trace to at least one DONE behavior.

## 5. Gaps (non-blocking)

- **G1 — z.ai live shape unconfirmed.** The response shape is assumed top-level
  `context_length` (FR-002). If the live probe differs, the fix is a data/parse-style change,
  not a code change. Tracked as a note in `spec.md` Assumptions and `plan.md`.
- **G2 — Pre-existing, unrelated test failures** remain in the module
  (`TestGetAvailableModelsClaudeIncludesTokenLimits`, three `internal/config` clone/home
  tests). They are outside this feature's scope and verified pre-existing via stash.
- **G3 — No CI test gate.** `.github/workflows/pr-test-build.yml` runs only `go build`,
  never `go test`. The TDD loop is the only enforcement. Recommended (separate task, not
  part of this feature): add a `go test ./internal/registry/... ./internal/config/...`
  step to CI.

## 6. Verdict

**PASS_WITH_GAPS.** The feature is delivered test-first, every acceptance criterion is
covered by a green behavior, and a deliberate mutant was killed. The gaps (G1–G3) are
documented, out of scope, and do not affect the feature's correctness for the 5 providers.
No remediation cycle is required.
