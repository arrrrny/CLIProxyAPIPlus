package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// ProviderParseStyle selects how a dedicated provider's /models response is parsed
// for context windows (FR-003).
type ProviderParseStyle string

const (
	// ParseStyleTopLevel parses models that expose a top-level context_length /
	// max_completion_tokens (openrouter, z.ai).
	ParseStyleTopLevel ProviderParseStyle = "top-level"
	// ParseStyleOpenCode parses models that return ids only (opencode /
	// opencode-go); context windows are sourced from the curated table (FR-008).
	ParseStyleOpenCode ProviderParseStyle = "opencode"
)

// ProviderConfig describes a dedicated OpenAI-compatible provider whose live
// models endpoint is the source of truth for context windows (FR-002, FR-004).
type ProviderConfig struct {
	// Name is the provider identifier (e.g. "openrouter") used to tag the
	// per-provider ModelInfo override.
	Name string
	// BaseURL is the OpenAI-compatible base URL. The provider-specific models path
	// (ModelsPath) is appended to it (FR-002).
	BaseURL string
	// APIKey is sent as a Bearer token in the Authorization header.
	APIKey string
	// ModelsPath is the provider-specific models-endpoint path appended to BaseURL
	// (FR-002). Empty falls back to the legacy "/v1/models" normalization.
	ModelsPath string
	// ParseStyle selects the response parser (FR-003). Empty defaults to top-level.
	ParseStyle ProviderParseStyle
}

// ProviderModelLimit is one model's reported window from a provider /models.
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

// FetchAndMerge GETs the provider's models endpoint (BaseURL + ModelsPath) with
// Authorization: Bearer <APIKey> and merges the reported windows into the registry
// keyed by the exact model id. A non-200 status or transport error is returned
// without wiping any value; the caller decides retry policy (FR-007, FR-008).
func (f *ProviderModelsFetcher) FetchAndMerge(ctx context.Context, prov ProviderConfig) error {
	url := resolveModelsURL(prov.BaseURL, prov.ModelsPath)

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read provider models for %s: %w", prov.Name, err)
	}

	items, err := parseProviderModels(prov.ParseStyle, body)
	if err != nil {
		return fmt.Errorf("parse provider models for %s: %w", prov.Name, err)
	}

	f.registry.MergeProviderContextWindows(prov.Name, items, curatedWindowsFor(prov.Name))
	return nil
}

// resolveModelsURL builds the full models URL. When modelsPath is empty it falls
// back to the legacy behavior: strip a trailing "/v1" from base then append
// "/v1/models" (U12). Otherwise modelsPath is appended to the trimmed base URL
// exactly as configured, so each provider reaches its real endpoint (FR-002).
func resolveModelsURL(baseURL, modelsPath string) string {
	base := strings.TrimRight(baseURL, "/")
	if modelsPath == "" {
		base = strings.TrimSuffix(base, "/v1")
		return base + "/v1/models"
	}
	if !strings.HasPrefix(modelsPath, "/") {
		modelsPath = "/" + modelsPath
	}
	return base + modelsPath
}

// parseProviderModels dispatches to the response parser for the provider's shape
// (FR-003).
func parseProviderModels(style ProviderParseStyle, body []byte) ([]ProviderModelLimit, error) {
	if style == ParseStyleOpenCode {
		return parseOpenCodeModels(body)
	}
	return parseTopLevelModels(body)
}

// parseTopLevelModels decodes an OpenAI-style list whose data items carry a
// top-level context_length / max_completion_tokens (openrouter, z.ai).
func parseTopLevelModels(body []byte) ([]ProviderModelLimit, error) {
	var payload struct {
		Data []ProviderModelLimit `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode top-level models: %w", err)
	}
	return payload.Data, nil
}

// parseOpenCodeModels decodes an OpenCode-style list whose data items carry an id
// only; context windows are filled later from the curated table (FR-008).
func parseOpenCodeModels(body []byte) ([]ProviderModelLimit, error) {
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode opencode models: %w", err)
	}
	out := make([]ProviderModelLimit, 0, len(payload.Data))
	for _, m := range payload.Data {
		if id := strings.TrimSpace(m.ID); id != "" {
			out = append(out, ProviderModelLimit{ID: id})
		}
	}
	return out, nil
}

// MergeProviderContextWindows applies fetched windows into the registry, keyed by
// the exact model id. A model id absent from the registry is registered under the
// provider so the unified catalog (US2, FR-009) can surface it. A positive fetched
// value overrides the stored one; when the endpoint omits a window, the curated
// table (FR-008) supplies it; a zero / missing value is otherwise ignored so known
// windows are never zeroed or guessed (FR-003, FR-006, FR-008). The per-provider
// ModelInfo override (InfoByProvider) is kept in sync, and the cache is invalidated.
func (r *ModelRegistry) MergeProviderContextWindows(provider string, items []ProviderModelLimit, curated map[string]ProviderModelLimit) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	now := time.Now()

	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		reg, ok := r.models[id]
		if !ok || reg == nil || reg.Info == nil {
			reg = r.ensureProviderModelLocked(provider, id, now)
		}
		info := reg.Info

		ctxLen := item.ContextLength
		if ctxLen <= 0 {
			if c, ok := curated[id]; ok {
				ctxLen = c.ContextLength
			}
		}
		mct := item.MaxCompletionTokens
		if mct <= 0 {
			if c, ok := curated[id]; ok {
				mct = c.MaxCompletionTokens
			}
		}

		if ctxLen > 0 {
			info.ContextLength = ctxLen
		}
		if mct > 0 {
			info.MaxCompletionTokens = mct
		}
		if pi := providerInfo(reg, provider); pi != nil {
			if ctxLen > 0 {
				pi.ContextLength = ctxLen
			}
			if mct > 0 {
				pi.MaxCompletionTokens = mct
			}
		}
	}

	r.invalidateAvailableModelsCacheLocked()
}

// ensureProviderModelLocked registers a model under the given provider if it is
// not already present. Caller must hold r.mutex. It mirrors the fields set by the
// internal model registration so the model is immediately available in the catalog.
func (r *ModelRegistry) ensureProviderModelLocked(provider, id string, now time.Time) *ModelRegistration {
	if reg, ok := r.models[id]; ok && reg != nil && reg.Info != nil {
		return reg
	}
	info := &ModelInfo{
		ID:      id,
		Object:  "model",
		OwnedBy: provider,
		Type:    provider,
		Created: now.Unix(),
	}
	reg := &ModelRegistration{
		Info:                 info,
		InfoByProvider:       map[string]*ModelInfo{provider: info},
		Count:                1,
		LastUpdated:          now,
		QuotaExceededClients: map[string]*time.Time{},
		SuspendedClients:     map[string]string{},
		Providers:            map[string]int{provider: 1},
	}
	r.models[id] = reg
	return reg
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
