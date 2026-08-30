---
feature: 001-provider-context-length
verdict: PASS_WITH_GAPS
standard: .specify/extensions/tdd/templates/tdd-test-quality-rubric.md
verified_at: e4d96ae0
behaviors: 24
proven: 0
likely: 23
test_after: 0
no_test: 0
open_deferred: 1
high_smells: 0
med_smells: 0
criteria_total: 6
criteria_covered: 6
fr_total: 10
fr_covered: 10
mutation_tool: null
mutants_sampled: 3
mutants_survived: 0
suite: feature packages green; full suite red from pre-existing, feature-unrelated failures (verified)
---

# TDD Verification: Accurate model context-length via dedicated provider endpoints

**Verdict: PASS_WITH_GAPS.** Discipline holds and every acceptance criterion is
covered end-to-end; no HIGH or MED smells and no surviving mutant. The gaps are in
*evidence strength*, not behavior: all test-first evidence is `LIKELY` (the work is
uncommitted, so git history cannot corroborate ordering), the loop did not log
outer-loop cycle entries for A1–A6, and mutation is measured by deliberate mutants
on a 3-behavior sample rather than a tool.

## Test-first evidence

| Behavior | Class    | Evidence |
| -------- | -------- | -------- |
| U1–U4    | LIKELY   | cycle-log cycles 1–2 record the red (`openai_handlers_test.go:57/61` missing `context_length`/`max_completion_tokens`) and the green; no git history (uncommitted) |
| U6       | LIKELY   | characterization baseline (guard for unchanged codex path); no red needed, green by definition |
| U7–U12   | LIKELY   | cycle-log cycle 4 records red (`undefined: NewProviderModelsFetcher`) → green; uncommitted |
| U13      | LIKELY   | `provider_configs_test.go` added green alongside `Config.DedicatedProviderConfigs`; uncommitted |
| U14,U15  | LIKELY   | cycle-log cycle 5 records red (`undefined: NewProviderModelsRefresher`) → green; uncommitted |
| U16–U18  | LIKELY   | cycle-log cycle 3 records red (stub returned nil / build error) → green; uncommitted |
| A1–A6    | LIKELY   | all 6 acceptance tests run GREEN against the live gin routes via `httptest`; no cycle-log entry written for the outer loop (loop appended cycles only for U*), and uncommitted |
| U5       | OPEN     | deferred (camelCase vs snake_case key for input/output limits); documented in test-list Notes and tasks.md; not counted as NO_TEST because the gap is explicit and FR-001 (context_length + max_completion_tokens) is fully covered by U1–U4 |

**No existing test was weakened.** `git diff` shows the feature added six new test
files and edited only source (`openai_handlers.go`, `api_json.go`, `config.go`,
`server.go`, `main.go`). No existing test file was modified, renamed, skipped, or
had its assertions loosened. The pre-existing `internal/registry` /
`internal/config` failures are in files this feature does not touch and were
verified to fail on a clean `HEAD` via `git stash -u`.

## Findings

No HIGH or MED findings. The fresh-context smell pass (delegated, read-only) graded
all six new test files against the rubric catalogue and the repository exemplars
(`model_registry_cache_test.go`, `codex_client_models_test.go`) and returned clean:
white-box same-package, stdlib-only assertions, `GetGlobalRegistry()` + `t.Cleanup`
isolation, value-exact asserts, no tautologies, no re-implemented expectations, no
doubled subject. LOW-level readability notes: none material enough to record.

## Mutation results

No mutation tool in `go.mod` (`mutation: null` in the profile). Deliberate mutants
on the three highest-risk behaviors, each: mutate → confirm the behavior's test
fails → restore exactly → re-run suite to confirm green. None survived.

| Mutant | Behavior | Survived | Judgment |
| ------ | -------- | -------- | -------- |
| `openai_handlers.go:92` `positiveInt(cl)` → `false` (drops `context_length` carry) | U1 (SC-001) | No | Caught by `TestOpenAIModels_…` and `TestAcceptance_A1_ContextLengthReturned` |
| `provider_models.go:103` `> 0` → `< 0` (fetched value no longer overrides static) | U9 (FR-003) | No | Caught by `TestProviderFetcher_OverridesStaticWhenPositive` |
| `api_json.go:34` `limit["context"] = clVal` → `0` (catalog reports wrong window) | U16 (FR-009/SC-004) | No | Caught by `TestAPIJSON_…` and `TestAcceptance_A4_APIJSONLimitFields` |

Scope note: 3 of 24 behaviors sampled — not exhaustive. Boundary-pair and
error-path behaviors (U3/U4 omit-when-zero, U10/U11 keep-last-good, U12 URL
normalize, U17/U18 omit-limit) were not individually mutated but are value-exact
and share the same helpers the sampled mutants already exercised.

## Traceability

| Criterion | Tests (traces) | End to end |
| --------- | -------------- | ---------- |
| SC-001 | A1, U1 | Yes (`GET /v1/models` httptest) |
| SC-002 | A2, U2, U9, U12 | Yes |
| SC-003 | A3, U8, U18 | Yes (both routes) |
| SC-004 | A4, U16, U17 | Yes (`GET /api.json` httptest) |
| SC-005 | A5, U10, U11, U15 | Yes (mock 401/offline + ticker) |
| SC-006 | A6, U13 | Yes (onboard via config, live window) |

| Requirement | Covered by |
| ----------- | ---------- |
| FR-001 | U1, U2, U3, U4 |
| FR-002 | U7, U12 |
| FR-003 | U9 |
| FR-004 | U13 |
| FR-005 | U8 |
| FR-006 | U7 (Bearer) |
| FR-007 | U14, U15, T008/T010 wiring |
| FR-008 | U10, U11, U15 |
| FR-009 | U16, U17, U18 |
| FR-010 | U13, A6 |

Every acceptance criterion has at least one end-to-end test through the real HTTP
entry point (`httptest` boot of the gin engine serving `/v1/models` and
`/api.json`). No criterion traces to a test that does not exist; no test traces to
nothing. Untested criteria: none.

## What was not audited

- **Git history ordering.** All feature work is uncommitted (the loop used
  `--no-commit`); `git log` cannot corroborate test-before-source. Evidence is
  `LIKELY`, not `PROVEN`. Committing the feature would let a future verify reach
  `PROVEN`.
- **Outer-loop cycle log.** A1–A6 were added and run green but the loop did not
  append cycle entries for them; the red/green for A1–A6 is evidenced by the test
  run, not by the cycle log.
- **Exhaustive mutation.** No tool available; only 3 behaviors sampled deliberately.
- **Full `go test ./...`.** Red at baseline from pre-existing, feature-unrelated
  failures (`internal/registry.TestGetAvailableModelsClaudeIncludesTokenLimits`,
  three `internal/config` clone/home tests), each verified to fail on a stashed
  clean `HEAD`. The feature's own packages (`sdk/api/handlers/openai`,
  `internal/registry` unit behaviors, `internal/config` unit) are green.
- **U5 (input/output token limits).** Deferred: key casing unresolved; out of scope
  for FR-001 and explicitly documented.
- **Performance / load.** No criterion, no test, not assessed.
