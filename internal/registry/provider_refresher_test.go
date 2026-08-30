package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// U14: on startup / connect, RefreshNow merges for every configured provider, and
// the periodic ticker re-runs on the configured interval.
func TestProviderRefresher_StartRefreshesImmediatelyAndPeriodically(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		_, _ = w.Write([]byte(providerBody(providerItem{ID: "openrouter-refresh-1", ContextLength: 200000})))
	}))
	defer srv.Close()

	reg := GetGlobalRegistry()
	reg.RegisterClient("test-refresh-u14", "openrouter", []*ModelInfo{{ID: "openrouter-refresh-1"}})
	t.Cleanup(func() { reg.UnregisterClient("test-refresh-u14") })

	f := NewProviderModelsFetcher(reg)
	r := NewProviderModelsRefresher(f, []ProviderConfig{{Name: "openrouter", BaseURL: srv.URL, APIKey: "k"}}, 20*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r.Start(ctx)

	// Immediate refresh on start.
	deadline := time.After(500 * time.Millisecond)
	for {
		mu.Lock()
		c := calls
		mu.Unlock()
		if c >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("refresher did not run immediately on start")
		case <-time.After(5 * time.Millisecond):
		}
	}

	// Periodic tick fires at least once more.
	for {
		mu.Lock()
		c := calls
		mu.Unlock()
		if c >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("refresher periodic tick did not fire")
		case <-time.After(5 * time.Millisecond):
		}
	}

	info := reg.GetModelInfo("openrouter-refresh-1", "")
	if info == nil || info.ContextLength != 200000 {
		t.Fatalf("ContextLength = %v, want 200000", info)
	}
}

// U15: a failed refresh keeps cached values and the next tick retries (does not
// wipe and does not panic).
func TestProviderRefresher_FailedRefreshKeepsCachedAndRetries(t *testing.T) {
	reg := GetGlobalRegistry()
	reg.RegisterClient("test-refresh-u15", "openrouter", []*ModelInfo{{ID: "openrouter-retry-1", ContextLength: 1000}})
	t.Cleanup(func() { reg.UnregisterClient("test-refresh-u15") })

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer bad.Close()

	f := NewProviderModelsFetcher(reg)
	r := NewProviderModelsRefresher(f, []ProviderConfig{{Name: "openrouter", BaseURL: bad.URL, APIKey: "bad"}}, time.Hour)

	// First refresh fails; cached value must survive.
	if err := r.RefreshNow(context.Background()); err == nil {
		t.Fatal("RefreshNow error = nil, want error")
	}
	info := reg.GetModelInfo("openrouter-retry-1", "")
	if info == nil || info.ContextLength != 1000 {
		t.Fatalf("after failed refresh ContextLength = %v, want 1000 (cached)", info)
	}

	// Flip to a good server; the next refresh (retry) succeeds and updates.
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(providerBody(providerItem{ID: "openrouter-retry-1", ContextLength: 300000})))
	}))
	defer good.Close()
	r.providers = []ProviderConfig{{Name: "openrouter", BaseURL: good.URL, APIKey: "k"}}

	if err := r.RefreshNow(context.Background()); err != nil {
		t.Fatalf("Retry RefreshNow error = %v, want nil", err)
	}
	info = reg.GetModelInfo("openrouter-retry-1", "")
	if info == nil || info.ContextLength != 300000 {
		t.Fatalf("after retry ContextLength = %v, want 300000", info)
	}
}
