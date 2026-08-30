# Feature Specification: Accurate model context-length via dedicated provider endpoints

**Feature Branch**: `001-provider-context-length`

**Created**: 2026-08-30

**Status**: Draft

**Input**: User description: "docs/api-json-context-length-spec.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Downstream clients receive the real context window per model (Priority: P1)

A downstream API consumer (for example, Quotio's ProxyBridge building the model catalog at `localhost:8317/api.json`, or the Kimi Code custom registry) queries CLIProxyApiPlus's models-listing endpoint and receives the real `context_length` and maximum output tokens for every model — including namespaced and free variants — instead of a wrong or missing value. The consumer no longer falls back to its own incorrect static table.

**Why this priority**: This is the core value of the feature. Correct context limits prevent downstream tools from misconfiguring or over/under-stating a model's usable window, which is the root defect being fixed.

**Independent Test**: Configure one provider (for example, openrouter) with a valid API key, start the server, and call the models-listing endpoint for a known free model (e.g. `ocd/deepseek-v4-flash-free`); assert the returned `context_length` equals the provider's real reported value rather than `0` or the static-table value.

**Acceptance Scenarios**:

1. **Given** a provider is connected with a valid API key, **When** a client requests the models list, **Then** every model that the provider reports with a `context_length` is returned with that `context_length` populated (never `0`).
2. **Given** a free-model variant whose real context window differs from the advertised/full model, **When** the client requests the models list, **Then** the returned `context_length` matches the provider's real value, not the static table entry.

---

### User Story 2 - Context length is sourced live from each provider, not a static table (Priority: P1)

The system fetches real model metadata (context window and maximum output tokens) directly from each dedicated provider's OpenAI-compatible models endpoint on startup, on account connect, and periodically thereafter, and merges it into a single global registry of truth. A fetched value overrides the built-in static table so free-model sizes are always correct.

**Why this priority**: This is the mechanism that makes User Story 1 durable and accurate; without live fetching, the static table will keep drifting wrong for free models.

**Independent Test**: Stand up a mock provider models endpoint returning a specific `context_length` for a free model; point the server at it; assert the registry's stored `context_length` equals the mock value and overrides any conflicting built-in value.

**Acceptance Scenarios**:

1. **Given** the server starts with a dedicated provider configured, **When** the provider's models endpoint is reachable, **Then** the registry stores the `context_length`/`max_completion_tokens` the provider returned.
2. **Given** the provider reports a `context_length` that differs from the built-in static table, **When** the fetch completes, **Then** the fetched value is what the registry keeps (fetched wins when > 0).

---

### User Story 3 - The six providers are first-class dedicated providers (Priority: P2)

`opencode`, `opencode-go`, `openrouter`, `kilo`, `kimi`, and `z.ai` are treated as distinct, dedicated providers (not generic "custom"/"openai"), each with a known OpenAI-compatible base URL and its own API-key entry, and each distinguishable in the registry.

**Why this priority**: Dedicated status is what lets the system fetch and label each provider's models correctly and onboard them through configuration rather than code changes.

**Independent Test**: Provide documented configuration blocks for all six providers; after load, assert each is present in the registry as a distinct provider type and that a fetch was attempted for each.

**Acceptance Scenarios**:

1. **Given** configuration declares the six providers with correct base URLs and API-key entries, **When** the server loads, **Then** each appears as a dedicated, distinguishable provider and a best-effort live fetch is triggered for each.

---

### User Story 4 - A machine-readable catalog exposes correct per-model limits (Priority: P2)

CLIProxyApiPlus serves a catalog endpoint (models.dev format) built from the enriched registry, carrying `limit.context` (context window) and `limit.output` (max completion tokens) for every model. Downstream consumers can read this catalog directly and trust the limits by construction.

**Why this priority**: Gives downstream tools (and Quotio) a single authoritative catalog so they stop rebuilding/guessing limits themselves.

**Independent Test**: With providers configured and fetched, `curl` the catalog endpoint and assert its shape matches the expected format and that namespaced free models carry correct `limit.context`/`limit.output`.

**Acceptance Scenarios**:

1. **Given** the registry is enriched with fetched context windows, **When** a client requests the catalog endpoint, **Then** each model entry includes `limit.context` and `limit.output` derived from the registry's `context_length`/`max_completion_tokens`.

---

### Edge Cases

- What happens when a provider's models endpoint is unreachable, returns 401, or times out? The system keeps the last-known-good values and never wipes previously stored context windows; it retries on the next refresh.
- What happens when a provider returns a model with only an `id` and no `context_length`? The system omits the limit for that model rather than guessing, and keeps any previously known value.
- What happens when the same model id is returned by multiple sources (static table vs live fetch)? The live fetched value wins when it is greater than `0`; otherwise the existing value is preserved.
- What happens with namespaced model ids (`ocd/...`, `ocg-1/...`, `z-ai/...`, `kilo/...`, `kiro-...`, `k3`, `k3-256k`)? They are stored and served under the exact id the provider returns; no namespace normalization or prefix stripping is performed.
- What happens on a transient provider outage during periodic refresh? Cached values remain served; no user-visible gap.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The models-listing endpoint MUST include `context_length` and `max_completion_tokens` for every model that has those values available in the registry.
- **FR-002**: The system MUST fetch model metadata (`context_length`, `max_completion_tokens`) from each dedicated provider's OpenAI-compatible models endpoint.
- **FR-003**: A fetched `context_length`/`max_completion_tokens` value MUST override the built-in static-table value when the fetched value is greater than `0`.
- **FR-004**: The six providers (`opencode`, `opencode-go`, `openrouter`, `kilo`, `kimi`, `z.ai`) MUST be treated as dedicated, distinguishable providers rather than generic custom/openai.
- **FR-005**: Fetched metadata MUST be merged into the global registry keyed by the exact model id the provider returns, preserving provider namespaces.
- **FR-006**: The system MUST authenticate provider fetches using each provider's configured API-key entry (Bearer token).
- **FR-007**: The system MUST periodically refresh fetched metadata for all dedicated providers (default cadence configurable; 6 hours proposed).
- **FR-008**: On fetch error (offline, 401, timeout, or missing `context_length`), the system MUST retain last-known-good values and MUST NOT overwrite them with zeros or guesses.
- **FR-009**: The system MUST serve a catalog endpoint in models.dev format carrying per-model `limit.context` and `limit.output` derived from the enriched registry.
- **FR-010**: New dedicated providers MUST be onboardable via configuration (base URL + API-key entry) without code changes.

### Key Entities *(include if feature involves data)*

- **Provider**: A distinct upstream model source with an id, an OpenAI-compatible base URL, and an API-key entry. The six dedicated providers are first-class instances.
- **Model**: A model offered by a provider, identified by the exact id the provider returns (may be namespaced), with attributes `context_length` and `max_completion_tokens`.
- **Global Model Registry**: The single source of truth holding merged `Model` metadata (combining built-in definitions and live-fetched values).
- **Catalog**: The published models.dev-format document exposing per-model `limit.context`/`limit.output`, built from the registry.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After a provider is connected, the models-listing endpoint reflects that provider's reported context window for at least 99% of models that the provider returns with a `context_length`, within one refresh cycle (no zeros where the provider provides a value).
- **SC-002**: For free-model variants of the six dedicated providers, the reported context window matches the provider's actual value (validated against a representative sample set), eliminating the prior static-table errors (e.g. `glm-5` ~203k and `grok-4` ~256k rather than the previously overstated values).
- **SC-003**: Namespaced model identifiers are preserved exactly end-to-end with zero dropped/normalized namespaces across a defined test set.
- **SC-004**: The published catalog endpoint exposes a per-model `limit.context` and `limit.output` for 100% of models with a known window fetched from providers.
- **SC-005**: A transient provider outage or auth error causes zero loss of previously known context windows; values remain available and are only updated by a successful subsequent refresh.
- **SC-006**: A new dedicated provider can be onboarded and its live context windows surfaced by editing configuration only, with no code change required.

## Assumptions

- **Provider base URLs**: Reasonable, documented defaults are assumed per provider — e.g. openrouter `https://openrouter.ai/api/v1`, z.ai `https://api.z.ai/v1`, opencode `https://opencode.ai/zen/v1`, with `opencode-go`, `kilo`, and `kimi` using their respective known OpenAI-compatible base URLs. Exact canonical URLs to be confirmed against existing config/account managers.
- **Refresh cadence**: A default periodic refresh of 6 hours, configurable, is assumed (mirrors the existing models-updater mechanics).
- **Missing `context_length`**: When a provider returns no `context_length`, the system keeps last-good values and omits the limit rather than guessing.
- **Catalog authority**: CLIProxyApiPlus is the authoritative source serving the catalog; downstream consumers (Quotio) read/proxy it rather than rebuilding limits from a static table.
- **Config reuse**: The existing OpenAI-compatibility configuration schema (base URL, API-key entries, models) is reused; no new config schema is introduced.
- **Scope boundary**: Modifications to the downstream Quotio `ProxyBridge` (removing its static `modelLimits` table) are an optional, out-of-repository follow-up and are out of scope for this feature in CLIProxyApiPlus.
- **Existing handler**: The Codex `?client_version=` path of the models endpoint is unchanged and builds its own response.
