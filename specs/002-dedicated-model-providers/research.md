# Research: Provider Model Endpoint Probes

**Feature**: `002-dedicated-model-providers` | **Date**: 2026-08-30

## R1 — Live endpoint verification (the core insight of this feature)

Each provider exposes its models at a **different** path and **different** response shape.
A single `<base-url>/v1/models` assumption (feature 001's limit) is wrong.

| Provider       | Real models endpoint (verified)           | Response shape for `context_length`                         | Confirmed |
| -------------- | ----------------------------------------- | ----------------------------------------------------------- | --------- |
| `openrouter`   | `https://openrouter.ai/api/v1/models`     | top-level `data[].context_length` (396 models)              | yes       |
| `kilo`         | `https://api.kilo.ai/api/openrouter/models` | nested `data[].top_provider.context_length`               | yes (existing `FetchKiloModels`) |
| `opencode`     | `https://opencode.ai/zen/v1/models`       | `data[].id` only — **no** context_length (63 models)        | yes       |
| `opencode-go`  | same `zen/v1/models` as opencode          | `data[].id` only — **no** context_length                    | yes       |
| `z.ai`         | `https://api.z.ai/api/v1/models`          | top-level `context_length` (assumed; `/v1/models` → 404)     | path confirmed; shape assumed top-level |

## R2 — Consequence for the fetcher

- Add `ModelsPath` per provider: empty ⇒ legacy `/v1/models` normalization (back-compat);
  explicit ⇒ appended directly to the trimmed base URL.
- Add `ParseStyle` per provider:
  - `top-level` → decode `data[].context_length` (openrouter, z.ai, default).
  - `opencode` → `id`s only; windows come from the curated table (opencode, opencode-go).

## R3 — Curated table for opencode / opencode-go

The `zen/v1/models` endpoint returns no window. A curated map keyed by model id supplies
known windows for the major families (claude, gemini, gpt, grok, deepseek, glm, minimax,
kimi, qwen, muse). Obscure/free models with no reliable public window are intentionally
**omitted** (FR-006: omit rather than fabricate).

## R4 — /v1/models and /api.json unchanged in shape

The consumer contract (models.dev `cliproxy.*` namespace on `/api.json`, `limit.context`
/ `limit.output`) is established by feature 001 and is preserved. This feature only
changes *where the numbers come from*.

## R5 — Pre-existing test failures (out of scope, verified)

`git stash -u` on clean `HEAD` still fails:
- `internal/registry.TestGetAvailableModelsClaudeIncludesTokenLimits` (expects `max_input_tokens` keys the registry does not emit),
- `internal/config` `TestCloneForRuntimeDeepCopiesConfig`, `TestCloneForRuntimeDoesNotShareReferenceFields`, `TestParseConfigBytesIgnoresHomeConfig`.

These are unrelated to this feature (confirmed via stash). Feature-scoped packages are green.
