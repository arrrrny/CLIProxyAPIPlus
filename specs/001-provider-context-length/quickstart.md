# Quickstart & Validation Guide

**Feature**: `001-provider-context-length` | **Date**: 2026-08-30
Proves the feature works end-to-end. Implementation details belong in `tasks.md` / the implementation phase.

## Prerequisites
- Go 1.26+ toolchain; repo builds (`go build -o cli-proxy-api ./cmd/server`).
- API keys for at least `openrouter` and `z.ai` (and `kimi` if available) exported as env vars referenced by `api-key-entries` in config.
- A provider account connected (or `openai-compatibility` blocks present in config).

## Validate the handler fix (FR-001)
```bash
go run ./cmd/server --config config.example.yaml
# in another shell:
curl -s localhost:18317/v1/models | jq '.data[] | select(.id=="ocd/deepseek-v4-flash-free")'
```
**Expected**: the entry now includes `context_length` and `max_completion_tokens` (non-zero), not just `id`/`object`/`created`/`owned_by`.

## Validate live fetch overrides the static table (FR-002, FR-003, SC-002)
Configure `openrouter` (free model `ocd/deepseek-v4-flash-free`) and `z.ai` (`k3-256k`) as dedicated providers, then:
```bash
curl -s localhost:18317/v1/models | jq '.data[] | {id, context_length}'
```
**Expected**: free-model `context_length` equals the provider's real value (e.g. `k3-256k` → 256000; `ocd/deepseek-v4-flash-free` → 1,000,000), not the over/under-stated static-table value.

## Validate the catalog endpoint (FR-009, SC-004)
```bash
curl -s localhost:18317/api.json | jq '.cliproxy.models["k3-256k"].limit'
```
**Expected**: `{ "context": 256000, "output": <max_completion_tokens> }` — correct by construction.

## Validate namespace preservation (FR-005, SC-003)
```bash
curl -s localhost:18317/api.json | jq '.cliproxy.models | keys[]' | grep -E 'ocd/|ocg-1/|z-ai/|kilo/|kiro-|k3|k3-256k'
```
**Expected**: namespaced ids appear verbatim; none are stripped/renamed.

## Validate error safety (FR-008, SC-005)
1. Stop the network / use a bad API key for one provider; restart.
2. `curl` `/v1/models` and `/api.json`.
**Expected**: previously known `context_length` values are retained (last-good); the broken provider's models keep their prior limits rather than being zeroed.

## Unit-test entry points (run during implementation)
```bash
go test ./internal/registry/...   # ProviderModelsFetcher: mock httptest /v1/models, assert merge + override
go test ./sdk/api/handlers/openai/...  # OpenAIModels emits context_length; /api.json shape
```
**Expected**: all green; fetcher test asserts fetched value wins over static and namespaces are preserved.
