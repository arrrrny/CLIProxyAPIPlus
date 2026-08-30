# Quickstart: Verify dedicated provider context windows

**Feature**: `002-dedicated-model-providers`

## Prerequisites

- Go 1.26+, the repo at branch `002-dedicated-model-providers`.
- Optional: real provider API keys for a live end-to-end check (otherwise unit tests cover behavior).

## 1. Unit-level verification (no keys needed)

```bash
go test ./internal/config/... -run 'TestDedicatedProviderConfigs_' -v -count=1
go test ./internal/registry/... -run 'TestProviderFetcher_' -v -count=1
```

Expected: all `--- PASS` for U13/U19 (config inference) and U20/U21/U22 (registry fetch/parse/curated).

## 2. Configure providers

In `config.yaml` (see `config.example.yaml` "Dedicated provider onboarding" block), add
`openai-compatibility` entries named `openrouter`, `opencode`, `opencode-go`, `z-ai`
(each with its `base-url` and api key). `kilo` is already dedicated.

## 3. Live check (needs keys)

```bash
go run ./cmd/server   # or: go build -o cli-proxy-api ./cmd/server && ./cli-proxy-api
curl -s localhost:<port>/api.json | jq '.cliproxy.models["openrouter/<model>"].limit'
curl -s localhost:<port>/api.json | jq '.cliproxy.models["claude-opus-5"].limit'  # opencode curated
```

Expect `openrouter/*` windows from the live endpoint, and `claude-opus-5` windows from
the curated table (FR-008) when `opencode` is configured.
