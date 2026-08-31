---
detected_at: e4d96ae0 # short SHA the profile was detected against
ecosystems: [go] # one entry per detected stack
default: go # which one the loop uses when a path is ambiguous
stacks:
  go:
    cwd: . # working directory every command below runs in (module root)
    runner: go test
    # {file} = Go package import path (e.g. ./internal/registry); Go runs tests per package, not per file.
    # {name} = exact test function name (e.g. TestProviderModelsFetcher_MergesContextLength).
    single: "go test {file} -run '^{name}$' -v -count=1"
    file: "go test {file} -count=1"
    suite: "go test ./..."
    watch: null # Go has no built-in watch; re-run `go test {file} -count=1` or use third-party `gow`/`reflex` (not added)
    coverage: "go test {file} -cover"
    mutation: null # gremlins / go-mutesting NOT in go.mod; audit falls back to deliberate mutants
    acceptance: "go test {file} -run '^{name}$' -v -count=1" # same runner; boot gin via net/http/httptest
    property: null # rapid / gopter NOT in deps
    approval: null # no snapshot lib (goldie) in deps
    contract: null # no Pact; /v1/models and /api.json contracts covered by unit/integration tests
    test_glob: "**/*_test.go"
    exemplar: # one per test kind the loop should imitate
      unit_registry: internal/registry/model_registry_cache_test.go
      unit_openai_handler: sdk/api/handlers/openai/codex_client_models_test.go
    helpers: [] # no shared test helper package detected; tests use stdlib testing + registry.GetGlobalRegistry() directly
verified: [single, file, coverage] # each was executed successfully in this repo
suite_baseline: red # go build ./... fails (amp module) AND internal/registry has a pre-existing failing test
suite_seconds: null # full suite not executed blind (network/credential-dependent tests present)
---

# TDD Stack Profile

## Conventions to match

- Test files sit next to the source as `<name>_test.go` and use the **same package** as the code
  (white-box): package `openai` in `sdk/api/handlers/openai`, package `registry` in `internal/registry`.
- **Assertions use only the standard library `testing` package** — `t.Fatalf` / `t.Errorf`.
  There is NO `testify` in this module (confirmed: not in `go.mod`). Do not import `testify`.
- Registry tests use `registry.GetGlobalRegistry()`, `RegisterClient(...)`, and
  `GetAvailableModels("openai")`, then assert on the returned `[]map[string]any` (see exemplar
  `internal/registry/model_registry_cache_test.go`). Always `t.Cleanup` to `UnregisterClient`.
- Handler tests construct the response from the registry and assert on the map (see exemplar
  `sdk/api/handlers/openai/codex_client_models_test.go`). For the new `/api.json` handler and any
  route test, boot the gin engine with `net/http/httptest` (stdlib) — the repo already uses
  `httptest.NewServer`/`NewRecorder` widely; reuse that pattern rather than hand-rolling an HTTP client.
- Logrus warnings like `models catalog: ... contains duplicate model id` are expected startup noise,
  not test failures. Filter them out when asserting on test output.

## Exemplars to imitate

- `internal/registry/model_registry_cache_test.go` — unit test for `GetAvailableModels` output
  (the exact shape this feature changes). Imitate for `provider_models_test.go`.
- `sdk/api/handlers/openai/codex_client_models_test.go` — white-box handler test that registers models
  in the global registry and asserts on the emitted response. Imitate for `openai_handlers_test.go`
  and `api_json_test.go`.

## Notes and constraints (BLOCKING — read before starting the loop)

- **`go build ./...` is currently RED.** `internal/api/modules/amp/model_mapping.go` references
  `config.AmpModelMapping` and `config.AmpUpstreamAPIKeyEntry`, which are undefined in the current
  (dirty) tree. This is pre-existing and unrelated to this feature, but it means the whole module does
  not compile. The two feature packages (`internal/registry`, `sdk/api/handlers/openai`) still compile
  and run their own tests independently — this break is isolated to the amp module.
- **`internal/registry` already has a failing test**: `TestGetAvailableModelsClaudeIncludesTokenLimits`
  fails with `expected max_input_tokens 200000, got <nil>` (observed during detection). This is in the
  very package this feature modifies and is pre-existing. Resolve/understand it before building on the
  registry, or the loop's "nothing else broke" signal is unreliable there.
- **Single-test command false-green guard (CRITICAL).** `go test {file} -run '^{name}$' -v` exits **0**
  when the regex matches no test ("testing: warning: no tests to run"). A mistyped name therefore yields
  a silent green. The loop MUST assert the named test actually executed — require
  `--- PASS: {name}` or `--- FAIL: {name}` to appear in the `-v` output, and treat its absence as a
  cycle failure. (A purist reading of the spec would set `single: null` and run the whole package; the
  command above is recorded because it works when the name is exact and is the ecosystem standard, with
  this mandatory guard.)
- **No CI test gate.** `.github/workflows/pr-test-build.yml` runs only `go build -o test-output ./cmd/server`,
  never `go test`. Tests are not enforced in CI today; the loop is the only enforcement.
- **Coverage** is available (`go test {file} -cover`) and was verified to produce a report.
- **Mutation / property / snapshot / contract tools are absent** (not in `go.mod`). The audit falls back
  to deliberate mutants (break one behavior, confirm a test fails, restore). Ecosystem defaults if added
  later: `gremlins` or `go-mutesting` (mutation), `rapid` or `gopter` (property), `goldie` (snapshot).
- **Full `go test ./...` was NOT executed blind** because the repo contains network/credential-dependent
  tests (provider executors call live endpoints). `suite` is recorded as the authoritative command; the
  baseline is red via the build failure above. Run the loop against the feature's packages
  (`internal/registry`, `sdk/api/handlers/openai`) and verify those in isolation.
