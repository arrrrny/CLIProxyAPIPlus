# Feature Specification: Dedicated Model Providers for Context Windows

**Feature Branch**: `002-dedicated-model-providers`

**Created**: 2026-08-30

**Status**: Draft

**Input**: User description: "make z.ai, opencode-go, opencode, kilo, openrouter as dedicated providers"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Each provider reports accurate context windows from its real endpoint (Priority: P1)

As an operator of CLIProxyAPIPlus, I want every dedicated provider (`openrouter`, `kilo`, `opencode`, `opencode-go`, `z.ai`) to expose correct per-model `context_length` / `max_completion_tokens` sourced from *that provider's own* models endpoint, so that `/v1/models` and `/api.json` are accurate per provider and downstream consumers (e.g. quota/proxy bridges) read correct limits.

**Why this priority**: A previous implementation assumed all providers expose a uniform `/v1/models` shape with a top-level `context_length`. In reality each provider differs: `openrouter` exposes a top-level `context_length`; `kilo` nests it under `top_provider`; `opencode` / `opencode-go` expose a models list with **no** `context_length`; `z.ai`'s assumed path returns 404. Accurate per-provider windows are the core value of this feature.

**Independent Test**: Configure each provider; after a refresh, `GET /api.json` shows correct `limit.context` / `limit.output` for a known model of each provider (e.g. an `openrouter` model with 200000, `kilo/auto` with 200000).

**Acceptance Scenarios**:

1. **Given** `openrouter` configured, **When** the refresher runs, **Then** `openrouter` models carry `context_length` taken from the provider's real top-level field.
2. **Given** `kilo` configured, **When** the dynamic fetch runs, **Then** `kilo` models carry `context_length` taken from the provider's nested field.
3. **Given** `opencode` / `opencode-go` configured, **When** the refresher runs against the provider's real models endpoint, **Then** models carry a context window (from the endpoint where present, else from the curated table) rather than being silently omitted.
4. **Given** `z.ai` configured, **When** the refresher runs against `z.ai`'s real models endpoint, **Then** `z.ai` models carry `context_length`.

---

### User Story 2 - Unified catalog exposes per-provider windows (Priority: P2)

`GET /api.json` and `GET /v1/models` return the merged, per-provider context windows for all dedicated providers, with each model keyed by its provider namespace.

**Why this priority**: Consumers read the single catalog; per-provider accuracy must be preserved end-to-end, not only at fetch time.

**Independent Test**: With multiple providers configured, `GET /api.json` shows each provider's models under their own namespace with correct `limit.context` / `limit.output`.

**Acceptance Scenarios**:

1. **Given** multiple providers configured, **When** `GET /api.json`, **Then** each model key retains its provider namespace and shows `limit.context` / `limit.output`.
2. **Given** a model with no known window, **When** `GET /api.json`, **Then** `limit` is omitted (never a fabricated zero).

---

### User Story 3 - Resilient refresh (Priority: P3)

Refresh tolerates provider errors and endpoint shape differences without wiping known values.

**Why this priority**: Live provider endpoints are unreliable; a bad cycle must never degrade already-correct windows.

**Independent Test**: Point a provider at an unreachable endpoint, run refresh, confirm a previously correct window is unchanged.

**Acceptance Scenarios**:

1. **Given** a provider endpoint is unreachable / 404, **When** refresh runs, **Then** last-known-good values are kept and the next cycle retries.
2. **Given** a provider returns a window of 0, **When** refresh runs, **Then** the existing positive value is not overwritten.

---

### Edge Cases

- Provider models endpoint returns 404 / 401 / timeout to a single provider → keep that provider's last-known-good, log, retry next cycle; other providers unaffected.
- Provider response lacks `context_length` for some or all models → fall back to the curated static table per model id; if neither source has a window, omit `limit`.
- Two providers report the same model id with different windows → stored per-provider, not cross-contaminated.
- Fetched window is 0 → ignored (positive stored value preserved).
- Nested field (e.g. `kilo` `top_provider.context_length`) missing in the provider response → treated as absent for that model.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST treat `openrouter`, `kilo`, `opencode`, `opencode-go`, `z.ai` as distinct dedicated providers, each with its own configured base URL and models-endpoint path.
- **FR-002**: System MUST fetch each provider's models from that provider's real, provider-specific models endpoint — not a single assumed `/v1/models` path.
- **FR-003**: System MUST parse each provider's actual response shape, including nested fields (e.g. `kilo`'s `top_provider.context_length`), to extract `context_length` and `max_completion_tokens`.
- **FR-004**: System MUST merge the fetched window into the registry keyed by exact model id; a positive fetched value overrides a stored one; a zero / missing value is ignored.
- **FR-005**: System MUST expose the merged windows on `GET /v1/models` and `GET /api.json` as `limit.context` / `limit.output`.
- **FR-006**: System MUST omit `limit` for a model with no known window rather than emitting a fabricated zero.
- **FR-007**: System MUST refresh on a configurable interval with an immediate refresh on startup / account-connect; a failed refresh keeps last-known-good and retries next cycle.
- **FR-008**: Where a provider's live endpoint omits `context_length`, System MUST fall back to the curated static table for that model (keyed by model id) rather than leaving the window absent — and MUST NOT fabricate a value not sourced from the provider or the curated table.
- **FR-009**: System MUST distinguish providers in the registry by name (`Type` / `OwnedBy`) so windows are not cross-contaminated between providers.

### Key Entities *(include if feature involves data)*

- **Dedicated Provider**: name, base URL, models-endpoint path, API key, response-parse strategy.
- **Model Context Window**: model id, `context_length`, `max_completion_tokens`, source (`provider fetch` | `curated table` | `static`).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For each of the 5 dedicated providers, `GET /api.json` returns the correct `limit.context` for at least one known model, verified against the provider's real response.
- **SC-002**: 100% of models that have a known window expose it on `/api.json`; 0 models emit a fabricated zero `limit`.
- **SC-003**: A provider endpoint outage causes no loss of previously known windows (last-known-good retained across cycles).
- **SC-004**: Refresh completes startup without blocking beyond the configured immediate-refresh timeout per provider.

## Assumptions

- Each provider has a single real models endpoint. Confirmed against live probes where possible: `openrouter` → `…/api/v1/models` (top-level `context_length`), `kilo` → `…/api/openrouter/models` (nested `top_provider.context_length`), `opencode` / `opencode-go` → `…/zen/v1/models` (returns models, no `context_length`). `z.ai`'s correct path and shape to be confirmed against `z.ai` API docs during implementation.
- `opencode` / `opencode-go` live endpoints return no `context_length`; their windows come from the curated static table (FR-008), keyed by model id.
- `z.ai`'s correct models endpoint and response shape will be confirmed against `z.ai` API documentation during implementation; the architecture supports a per-provider endpoint + parse strategy so correcting it is a config change, not a code change.
- `kilo`'s existing dynamic fetch (`FetchKiloModels`) is reused as that provider's dedicated strategy.
- Auth / API keys are supplied via existing configuration; no new secret storage is introduced.
