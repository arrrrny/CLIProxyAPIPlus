package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// ProviderConfig describes a dedicated OpenAI-compatible provider whose live
// /v1/models endpoint is the source of truth for context windows (FR-002, FR-004).
type ProviderConfig struct {
	// Name is the provider identifier (e.g. "openrouter") used to tag the
	// per-provider ModelInfo override.
	Name string
	// BaseURL is the OpenAI-compatible base URL. A trailing "/v1" is tolerated and
	// stripped before appending "/v1/models" (FR-002, U12).
	BaseURL string
	// APIKey is sent as a Bearer token in the Authorization header.
	APIKey string
}

// ProviderModelLimit is one model's reported window from a provider /v1/models.
type ProviderModelLimit struct {
	ID                  string `json:"id"`
	ContextLength       int    `json:"context_length"`
	MaxCompletionTokens int    `json:"max_completion_tokens"`
}

// ProviderModelsFetcher fetches /v1/models from a dedicated provider and merges
// the reported context_length / max_completion_tokens back into the registry.
// It never overwrites a known value with a zero, and on any fetch error it keeps
// the last-known-good values (FR-003, FR-008). Construction is cheap; reuse one
// instance across refreshes.
type ProviderModelsFetcher struct {
	registry *ModelRegistry
	client   *http.Client
}

// NewProviderModelsFetcher builds a fetcher bound to the given registry.
func NewProviderModelsFetcher(reg *ModelRegistry) *ProviderModelsFetcher {
	return &ProviderModelsFetcher{
		registry: reg,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchAndMerge GETs <baseURL>/v1/models with Authorization: Bearer <APIKey> and
// merges the reported windows into the registry keyed by the exact model id. A
// non-200 status or transport error is returned without wiping any value; the
// caller decides retry policy (FR-007, FR-008).
func (f *ProviderModelsFetcher) FetchAndMerge(ctx context.Context, prov ProviderConfig) error {
	base := strings.TrimRight(prov.BaseURL, "/")
	base = strings.TrimSuffix(base, "/v1")
	url := base + "/v1/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build provider models request for %s: %w", prov.Name, err)
	}
	req.Header.Set("Authorization", "Bearer "+prov.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch provider models for %s: %w", prov.Name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch provider models for %s: unexpected status %d", prov.Name, resp.StatusCode)
	}

	var payload struct {
		Data []ProviderModelLimit `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode provider models for %s: %w", prov.Name, err)
	}

	f.registry.MergeProviderContextLengths(prov.Name, payload.Data)
	return nil
}

// MergeProviderContextLengths applies fetched windows into the registry, keyed by
// the exact model id. A positive fetched value overrides the stored one; a zero
// or missing value is ignored so known windows are never zeroed or guessed
// (FR-003, FR-008). The per-provider ModelInfo override (InfoByProvider) is kept
// in sync when present, and the available-models cache is invalidated.
func (r *ModelRegistry) MergeProviderContextLengths(provider string, items []ProviderModelLimit) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, item := range items {
		reg, ok := r.models[item.ID]
		if !ok || reg == nil || reg.Info == nil {
			continue
		}
		if item.ContextLength > 0 {
			reg.Info.ContextLength = item.ContextLength
			if pi := providerInfo(reg, provider); pi != nil {
				pi.ContextLength = item.ContextLength
			}
		}
		if item.MaxCompletionTokens > 0 {
			reg.Info.MaxCompletionTokens = item.MaxCompletionTokens
			if pi := providerInfo(reg, provider); pi != nil {
				pi.MaxCompletionTokens = item.MaxCompletionTokens
			}
		}
	}

	r.invalidateAvailableModelsCacheLocked()
}

// providerInfo returns the per-provider ModelInfo override for a registration.
func providerInfo(reg *ModelRegistration, provider string) *ModelInfo {
	if reg.InfoByProvider == nil {
		return nil
	}
	if pi, ok := reg.InfoByProvider[provider]; ok && pi != nil {
		return pi
	}
	return nil
}

// DefaultProviderRefreshInterval is the periodic refresh cadence for dedicated
// provider context windows when none is configured (FR-007).
const DefaultProviderRefreshInterval = 6 * time.Hour

// ProviderModelsRefresher runs ProviderModelsFetcher across a fixed set of
// dedicated providers: once immediately (startup / account-connect) and then on
// a configurable ticker (FR-007). A failed refresh logs and keeps the
// last-known-good values; the next tick simply retries, so no cached window is
// ever wiped (FR-008).
type ProviderModelsRefresher struct {
	fetcher   *ProviderModelsFetcher
	providers []ProviderConfig
	interval  time.Duration

	mu   sync.Mutex
	stop chan struct{}
}

// NewProviderModelsRefresher builds a refresher. A non-positive interval falls
// back to DefaultProviderRefreshInterval.
func NewProviderModelsRefresher(fetcher *ProviderModelsFetcher, providers []ProviderConfig, interval time.Duration) *ProviderModelsRefresher {
	if interval <= 0 {
		interval = DefaultProviderRefreshInterval
	}
	return &ProviderModelsRefresher{
		fetcher:   fetcher,
		providers: providers,
		interval:  interval,
	}
}

// RefreshNow merges context windows for every configured provider. It is
// best-effort: a failure for one provider does not abort the others, and the
// first error encountered is returned for logging. Cached values are never wiped
// (FR-008).
func (r *ProviderModelsRefresher) RefreshNow(ctx context.Context) error {
	var firstErr error
	for _, p := range r.providers {
		if err := r.fetcher.FetchAndMerge(ctx, p); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			log.Warnf("[provider-refresh] %s: %v", p.Name, err)
		}
	}
	return firstErr
}

// Start begins periodic refreshes on the configured interval until ctx is
// cancelled or Stop is called (FR-007). The initial refresh runs inside the
// ticker goroutine as its first tick, so Start returns immediately and never
// blocks server startup on slow/unreachable providers (FR-008). Calling Start
// more than once is a no-op.
func (r *ProviderModelsRefresher) Start(ctx context.Context) {
	r.mu.Lock()
	if r.stop != nil {
		r.mu.Unlock()
		return
	}
	r.stop = make(chan struct{})
	stop := r.stop
	r.mu.Unlock()

	ticker := time.NewTicker(r.interval)
	go func() {
		defer ticker.Stop()
		// Best-effort immediate refresh without blocking server startup.
		if err := r.RefreshNow(ctx); err != nil {
			log.Warnf("[provider-refresh] initial refresh failed: %v", err)
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				r.RefreshNow(ctx)
			}
		}
	}()
}

// Stop signals the ticker goroutine started by Start to exit. It is safe to
// call multiple times and before Start (it is then a no-op).
func (r *ProviderModelsRefresher) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stop != nil {
		close(r.stop)
		r.stop = nil
	}
}
