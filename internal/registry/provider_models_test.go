package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// providerBody builds an OpenAI-style /v1/models payload for the fake server.
func providerBody(items ...providerItem) string {
	payload := struct {
		Object string         `json:"object"`
		Data   []providerItem `json:"data"`
	}{Object: "list", Data: items}
	b, _ := json.Marshal(payload)
	return string(b)
}

type providerItem struct {
	ID                  string `json:"id"`
	ContextLength       int    `json:"context_length"`
	MaxCompletionTokens int    `json:"max_completion_tokens"`
}

// U7: FetchAndMerge GETs <baseURL>/v1/models with Bearer auth and sets
// ContextLength / MaxCompletionTokens for each returned id.
func TestProviderFetcher_FetchAndMergeSetsContextLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("request path = %q, want /v1/models", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization header = %q, want Bearer test-key", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(providerBody(
			providerItem{ID: "openrouter-fetch-1", ContextLength: 200000, MaxCompletionTokens: 64000},
		)))
	}))
	defer srv.Close()

	reg := GetGlobalRegistry()
	reg.RegisterClient("test-fetch-u7", "openrouter", []*ModelInfo{
		{ID: "openrouter-fetch-1"},
	})
	t.Cleanup(func() { reg.UnregisterClient("test-fetch-u7") })

	f := NewProviderModelsFetcher(reg)
	if err := f.FetchAndMerge(context.Background(), ProviderConfig{
		Name:    "openrouter",
		BaseURL: srv.URL,
		APIKey:  "test-key",
	}); err != nil {
		t.Fatalf("FetchAndMerge error = %v, want nil", err)
	}

	info := reg.GetModelInfo("openrouter-fetch-1", "")
	if info == nil {
		t.Fatal("GetModelInfo(openrouter-fetch-1) = nil")
	}
	if info.ContextLength != 200000 {
		t.Errorf("ContextLength = %d, want 200000", info.ContextLength)
	}
	if info.MaxCompletionTokens != 64000 {
		t.Errorf("MaxCompletionTokens = %d, want 64000", info.MaxCompletionTokens)
	}
}

// U8: merges keyed by the exact provider-returned id, preserving namespaces.
func TestProviderFetcher_MergesByExactID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(providerBody(
			providerItem{ID: "ocd/deepseek-v4-flash-free", ContextLength: 1000000},
		)))
	}))
	defer srv.Close()

	reg := GetGlobalRegistry()
	reg.RegisterClient("test-fetch-u8", "openrouter", []*ModelInfo{
		{ID: "ocd/deepseek-v4-flash-free"},
	})
	t.Cleanup(func() { reg.UnregisterClient("test-fetch-u8") })

	f := NewProviderModelsFetcher(reg)
	if err := f.FetchAndMerge(context.Background(), ProviderConfig{
		Name:    "openrouter",
		BaseURL: srv.URL,
		APIKey:  "k",
	}); err != nil {
		t.Fatalf("FetchAndMerge error = %v, want nil", err)
	}

	info := reg.GetModelInfo("ocd/deepseek-v4-flash-free", "")
	if info == nil {
		t.Fatal("GetModelInfo(ocd/deepseek-v4-flash-free) = nil")
	}
	if info.ContextLength != 1000000 {
		t.Errorf("ContextLength = %d, want 1000000", info.ContextLength)
	}
}

// U9: a fetched context_length > 0 overrides the existing static-table value.
func TestProviderFetcher_OverridesStaticWhenPositive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(providerBody(
			providerItem{ID: "openrouter-override-1", ContextLength: 200000, MaxCompletionTokens: 64000},
		)))
	}))
	defer srv.Close()

	reg := GetGlobalRegistry()
	reg.RegisterClient("test-fetch-u9", "openrouter", []*ModelInfo{
		{ID: "openrouter-override-1", ContextLength: 1000, MaxCompletionTokens: 100},
	})
	t.Cleanup(func() { reg.UnregisterClient("test-fetch-u9") })

	f := NewProviderModelsFetcher(reg)
	if err := f.FetchAndMerge(context.Background(), ProviderConfig{Name: "openrouter", BaseURL: srv.URL, APIKey: "k"}); err != nil {
		t.Fatalf("FetchAndMerge error = %v, want nil", err)
	}

	info := reg.GetModelInfo("openrouter-override-1", "")
	if info.ContextLength != 200000 {
		t.Errorf("ContextLength = %d, want 200000 (overridden)", info.ContextLength)
	}
	if info.MaxCompletionTokens != 64000 {
		t.Errorf("MaxCompletionTokens = %d, want 64000 (overridden)", info.MaxCompletionTokens)
	}
}

// U10: a fetched context_length of 0 (or missing) leaves the existing value,
// never zeroed or guessed.
func TestProviderFetcher_KeepsExistingWhenZeroOrMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(providerBody(
			providerItem{ID: "openrouter-keep-1", ContextLength: 0, MaxCompletionTokens: 0},
		)))
	}))
	defer srv.Close()

	reg := GetGlobalRegistry()
	reg.RegisterClient("test-fetch-u10", "openrouter", []*ModelInfo{
		{ID: "openrouter-keep-1", ContextLength: 1000, MaxCompletionTokens: 100},
	})
	t.Cleanup(func() { reg.UnregisterClient("test-fetch-u10") })

	f := NewProviderModelsFetcher(reg)
	if err := f.FetchAndMerge(context.Background(), ProviderConfig{Name: "openrouter", BaseURL: srv.URL, APIKey: "k"}); err != nil {
		t.Fatalf("FetchAndMerge error = %v, want nil", err)
	}

	info := reg.GetModelInfo("openrouter-keep-1", "")
	if info.ContextLength != 1000 {
		t.Errorf("ContextLength = %d, want 1000 (retained)", info.ContextLength)
	}
	if info.MaxCompletionTokens != 100 {
		t.Errorf("MaxCompletionTokens = %d, want 100 (retained)", info.MaxCompletionTokens)
	}
}

// U11: a fetch error (401) keeps last-known-good values and returns an error.
func TestProviderFetcher_ErrorKeepsLastKnownGood(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	reg := GetGlobalRegistry()
	reg.RegisterClient("test-fetch-u11", "openrouter", []*ModelInfo{
		{ID: "openrouter-err-1", ContextLength: 1000, MaxCompletionTokens: 100},
	})
	t.Cleanup(func() { reg.UnregisterClient("test-fetch-u11") })

	f := NewProviderModelsFetcher(reg)
	err := f.FetchAndMerge(context.Background(), ProviderConfig{Name: "openrouter", BaseURL: srv.URL, APIKey: "bad"})
	if err == nil {
		t.Fatal("FetchAndMerge error = nil, want error")
	}

	info := reg.GetModelInfo("openrouter-err-1", "")
	if info == nil {
		t.Fatal("GetModelInfo(openrouter-err-1) = nil after error")
	}
	if info.ContextLength != 1000 {
		t.Errorf("ContextLength = %d, want 1000 (last-known-good retained)", info.ContextLength)
	}
}

// U12: base URLs both with and without a trailing /v1 resolve to <base>/v1/models.
func TestProviderFetcher_NormalizesBaseURL(t *testing.T) {
	var gotPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		_, _ = w.Write([]byte(providerBody(providerItem{ID: "openrouter-norm-1", ContextLength: 50000})))
	}))
	defer srv.Close()

	reg := GetGlobalRegistry()
	reg.RegisterClient("test-fetch-u12", "openrouter", []*ModelInfo{
		{ID: "openrouter-norm-1"},
	})
	t.Cleanup(func() { reg.UnregisterClient("test-fetch-u12") })

	f := NewProviderModelsFetcher(reg)
	// Base URL WITHOUT trailing /v1
	if err := f.FetchAndMerge(context.Background(), ProviderConfig{Name: "openrouter", BaseURL: srv.URL, APIKey: "k"}); err != nil {
		t.Fatalf("FetchAndMerge (no /v1) error = %v, want nil", err)
	}
	// Base URL WITH trailing /v1
	if err := f.FetchAndMerge(context.Background(), ProviderConfig{Name: "openrouter", BaseURL: srv.URL + "/v1", APIKey: "k"}); err != nil {
		t.Fatalf("FetchAndMerge (with /v1) error = %v, want nil", err)
	}

	for _, p := range gotPaths {
		if p != "/v1/models" {
			t.Errorf("request path = %q, want /v1/models (got paths: %v)", p, gotPaths)
		}
	}
}

// opencodeBody builds an OpenCode-style /zen/v1/models payload (ids only).
func opencodeBody(ids ...string) string {
	type item struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		OwnedBy string `json:"owned_by"`
	}
	data := make([]item, 0, len(ids))
	for _, id := range ids {
		data = append(data, item{ID: id, Object: "model", OwnedBy: "opencode"})
	}
	b, _ := json.Marshal(struct {
		Object string `json:"object"`
		Data   []item `json:"data"`
	}{Object: "list", Data: data})
	return string(b)
}

// U20 (FR-002, FR-009): opencode hits its real /zen/v1/models path, registers
// each returned id under the opencode namespace, and applies the curated window
// (the endpoint carries no context_length).
func TestProviderFetcher_OpenCodeRegistersAndUsesCuratedTable(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(opencodeBody("claude-opus-5", "gemini-3.6-flash")))
	}))
	defer srv.Close()

	reg := GetGlobalRegistry()
	f := NewProviderModelsFetcher(reg)
	prov := ProviderConfig{
		Name:       "opencode",
		BaseURL:    srv.URL,
		APIKey:     "k",
		ModelsPath: "/zen/v1/models",
		ParseStyle: ParseStyleOpenCode,
	}
	if err := f.FetchAndMerge(context.Background(), prov); err != nil {
		t.Fatalf("FetchAndMerge error = %v, want nil", err)
	}
	if gotPath != "/zen/v1/models" {
		t.Fatalf("request path = %q, want /zen/v1/models", gotPath)
	}

	info := reg.GetModelInfo("claude-opus-5", "opencode")
	if info == nil {
		t.Fatal("GetModelInfo(claude-opus-5, opencode) = nil after fetch")
	}
	if info.OwnedBy != "opencode" {
		t.Errorf("OwnedBy = %q, want opencode", info.OwnedBy)
	}
	if info.ContextLength != 200000 {
		t.Errorf("ContextLength (curated) = %d, want 200000", info.ContextLength)
	}
	if info.MaxCompletionTokens != 64000 {
		t.Errorf("MaxCompletionTokens (curated) = %d, want 64000", info.MaxCompletionTokens)
	}

	info2 := reg.GetModelInfo("gemini-3.6-flash", "opencode")
	if info2 == nil || info2.ContextLength != 1048576 {
		t.Fatalf("gemini-3.6-flash curated ContextLength = %v, want 1048576", info2)
	}
}

// U21 (FR-002, FR-003): z.ai hits its real /api/v1/models (top-level shape) and
// applies the live context_length to a model not pre-registered (auto-registered).
func TestProviderFetcher_ZAiTopLevelRegistersNewModel(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(providerBody(providerItem{ID: "z-ai/glm-4.6", ContextLength: 128000, MaxCompletionTokens: 8000})))
	}))
	defer srv.Close()

	reg := GetGlobalRegistry()
	f := NewProviderModelsFetcher(reg)
	prov := ProviderConfig{
		Name:       "z-ai",
		BaseURL:    srv.URL,
		APIKey:     "k",
		ModelsPath: "/api/v1/models",
		ParseStyle: ParseStyleTopLevel,
	}
	if err := f.FetchAndMerge(context.Background(), prov); err != nil {
		t.Fatalf("FetchAndMerge error = %v, want nil", err)
	}
	if gotPath != "/api/v1/models" {
		t.Fatalf("request path = %q, want /api/v1/models", gotPath)
	}

	info := reg.GetModelInfo("z-ai/glm-4.6", "z-ai")
	if info == nil {
		t.Fatal("GetModelInfo(z-ai/glm-4.6, z-ai) = nil after fetch (auto-register)")
	}
	if info.ContextLength != 128000 {
		t.Errorf("ContextLength = %d, want 128000", info.ContextLength)
	}
	if info.MaxCompletionTokens != 8000 {
		t.Errorf("MaxCompletionTokens = %d, want 8000", info.MaxCompletionTokens)
	}
}

// U22 (FR-008, FR-006): a curated model with no live window gets the curated
// value; a model absent from both the endpoint and the curated table gets no
// window (never fabricated).
func TestProviderFetcher_OpenCodeCuratedFallbackAndOmission(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(opencodeBody("claude-opus-5", "unknown-model")))
	}))
	defer srv.Close()

	reg := GetGlobalRegistry()
	f := NewProviderModelsFetcher(reg)
	prov := ProviderConfig{Name: "opencode", BaseURL: srv.URL, APIKey: "k", ModelsPath: "/zen/v1/models", ParseStyle: ParseStyleOpenCode}
	if err := f.FetchAndMerge(context.Background(), prov); err != nil {
		t.Fatalf("FetchAndMerge error = %v, want nil", err)
	}

	curated := reg.GetModelInfo("claude-opus-5", "opencode")
	if curated == nil || curated.ContextLength != 200000 {
		t.Fatalf("claude-opus-5 ContextLength = %v, want 200000 (curated)", curated)
	}
	unknown := reg.GetModelInfo("unknown-model", "opencode")
	if unknown == nil {
		t.Fatal("unknown-model not registered")
	}
	if unknown.ContextLength != 0 {
		t.Errorf("unknown-model ContextLength = %d, want 0 (omitted, not fabricated)", unknown.ContextLength)
	}
}

// TestResolveModelsURL exercises the URL-building logic in isolation. It locks in
// the doubled-/v1 regression guard: when a base URL already contains /v1, the
// default provider path must append "/models" (not "/v1/models") so the result is
// not <base>/v1/v1/models (see defaultProviderModelsPath).
func TestResolveModelsURL(t *testing.T) {
	cases := []struct {
		base, path, want string
	}{
		{"https://api.example.com/v1", "", "https://api.example.com/v1/models"}, // legacy branch strips /v1 then re-adds
		{"https://api.example.com/v1", "/models", "https://api.example.com/v1/models"},
		{"https://api.example.com/v1", "/v1/models", "https://api.example.com/v1/v1/models"}, // documents the doubled behavior
		{"https://opencode.ai", "/zen/v1/models", "https://opencode.ai/zen/v1/models"},
		{"https://opencode.ai", "/zen/go/v1/models", "https://opencode.ai/zen/go/v1/models"},
		{"https://api.kimi.com/coding/v1", "/models", "https://api.kimi.com/coding/v1/models"},
	}
	for _, c := range cases {
		if got := resolveModelsURL(c.base, c.path); got != c.want {
			t.Errorf("resolveModelsURL(%q, %q) = %q, want %q", c.base, c.path, got, c.want)
		}
	}
}
