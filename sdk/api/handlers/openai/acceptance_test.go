package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/api/handlers"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

// newAcceptanceServer wires the real /v1/models and /api.json routes (the same
// handler methods the production router calls) so acceptance criteria are
// exercised through the actual HTTP entry points.
func newAcceptanceServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewOpenAIAPIHandler(handlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, nil))
	engine := gin.New()
	engine.GET("/v1/models", h.OpenAIModels)
	engine.GET("/api.json", h.APIJSON)
	return httptest.NewServer(engine)
}

// apiJSONDoc is the models.dev-format /api.json document.
type apiJSONDoc struct {
	Cliproxy struct {
		Models map[string]struct {
			ID           string `json:"id"`
			Limit        map[string]any `json:"limit"`
			ContextLength any `json:"context_length"`
		} `json:"models"`
	} `json:"cliproxy"`
}

func apiJSONModel(t *testing.T, body []byte, id string) map[string]any {
	t.Helper()
	var doc apiJSONDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("unmarshal /api.json: %v", err)
	}
	m, ok := doc.Cliproxy.Models[id]
	if !ok {
		t.Fatalf("model %q not found in /api.json: %s", id, string(body))
	}
	out := map[string]any{"id": m.ID}
	for k, v := range m.Limit {
		out[k] = v
	}
	if m.ContextLength != nil {
		out["context_length"] = m.ContextLength
	}
	return out
}

// A1: after a provider is connected, GET /v1/models returns the provider's
// reported context_length (non-zero) for a model it serves.
func TestAcceptance_A1_ContextLengthReturned(t *testing.T) {
	srv := newAcceptanceServer(t)
	defer srv.Close()

	reg := registry.GetGlobalRegistry()
	const clientID = "acc-a1"
	reg.RegisterClient(clientID, "openai-compatibility", []*registry.ModelInfo{
		{ID: "acc-a1-model", Object: "model", OwnedBy: "openrouter", Type: "openai-compatibility", ContextLength: 200000, MaxCompletionTokens: 64000},
	})
	t.Cleanup(func() { reg.UnregisterClient(clientID) })

	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil {
		t.Fatalf("GET /v1/models: %v", err)
	}
	defer resp.Body.Close()
	body := readAll(t, resp)
	m := modelByID(t, body, "acc-a1-model")
	if got, ok := m["context_length"]; !ok || toFloat64(got) != 200000 {
		t.Fatalf("context_length = %v (present=%v), want 200000", got, ok)
	}
}

// A2: a free-model variant returns its real (large) provider context_length, not
// a static-table placeholder.
func TestAcceptance_A2_FreeModelRealWindow(t *testing.T) {
	srv := newAcceptanceServer(t)
	defer srv.Close()

	reg := registry.GetGlobalRegistry()
	const clientID = "acc-a2"
	reg.RegisterClient(clientID, "openai-compatibility", []*registry.ModelInfo{
		{ID: "ocd/deepseek-v4-flash-free", Object: "model", OwnedBy: "openrouter", Type: "openai-compatibility", ContextLength: 1000000, MaxCompletionTokens: 8000},
	})
	t.Cleanup(func() { reg.UnregisterClient(clientID) })

	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil {
		t.Fatalf("GET /v1/models: %v", err)
	}
	defer resp.Body.Close()
	m := modelByID(t, readAll(t, resp), "ocd/deepseek-v4-flash-free")
	if got, ok := m["context_length"]; !ok || toFloat64(got) != 1000000 {
		t.Fatalf("free-model context_length = %v (present=%v), want 1000000", got, ok)
	}
}

// A3: namespaced ids are preserved verbatim on both /v1/models and /api.json.
func TestAcceptance_A3_NamespacedIDsPreserved(t *testing.T) {
	srv := newAcceptanceServer(t)
	defer srv.Close()

	reg := registry.GetGlobalRegistry()
	const clientID = "acc-a3"
	const nsID = "ocd/deepseek-v4-flash-free"
	reg.RegisterClient(clientID, "openai-compatibility", []*registry.ModelInfo{
		{ID: nsID, Object: "model", OwnedBy: "openrouter", Type: "openai-compatibility", ContextLength: 128000, MaxCompletionTokens: 8000},
	})
	t.Cleanup(func() { reg.UnregisterClient(clientID) })

	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil {
		t.Fatalf("GET /v1/models: %v", err)
	}
	defer resp.Body.Close()
	m := modelByID(t, readAll(t, resp), nsID)
	if got, _ := m["id"].(string); got != nsID {
		t.Fatalf("/v1/models id = %q, want verbatim %q", got, nsID)
	}

	aj, err := http.Get(srv.URL + "/api.json")
	if err != nil {
		t.Fatalf("GET /api.json: %v", err)
	}
	defer aj.Body.Close()
	am := apiJSONModel(t, readAll(t, aj), nsID)
	if got, _ := am["id"].(string); got != nsID {
		t.Fatalf("/api.json id = %q, want verbatim %q", got, nsID)
	}
}

// A4: GET /api.json exposes limit.context and limit.output for a model with a
// known window.
func TestAcceptance_A4_APIJSONLimitFields(t *testing.T) {
	srv := newAcceptanceServer(t)
	defer srv.Close()

	reg := registry.GetGlobalRegistry()
	const clientID = "acc-a4"
	reg.RegisterClient(clientID, "openai-compatibility", []*registry.ModelInfo{
		{ID: "acc-a4-model", Object: "model", OwnedBy: "openrouter", Type: "openai-compatibility", ContextLength: 200000, MaxCompletionTokens: 64000},
	})
	t.Cleanup(func() { reg.UnregisterClient(clientID) })

	aj, err := http.Get(srv.URL + "/api.json")
	if err != nil {
		t.Fatalf("GET /api.json: %v", err)
	}
	defer aj.Body.Close()
	am := apiJSONModel(t, readAll(t, aj), "acc-a4-model")
	if got, ok := am["context"]; !ok || toFloat64(got) != 200000 {
		t.Fatalf("limit.context = %v (present=%v), want 200000", got, ok)
	}
	if got, ok := am["output"]; !ok || toFloat64(got) != 64000 {
		t.Fatalf("limit.output = %v (present=%v), want 64000", got, ok)
	}
}

// A5: after a provider fetch error, /v1/models and /api.json still return the
// last-known-good context_length (no zeroing).
func TestAcceptance_A5_FetchErrorKeepsLastKnownGood(t *testing.T) {
	srv := newAcceptanceServer(t)
	defer srv.Close()

	reg := registry.GetGlobalRegistry()
	const clientID = "acc-a5"
	const id = "acc-a5-model"
	reg.RegisterClient(clientID, "openai-compatibility", []*registry.ModelInfo{
		{ID: id, Object: "model", OwnedBy: "openrouter", Type: "openai-compatibility", ContextLength: 200000, MaxCompletionTokens: 64000},
	})
	t.Cleanup(func() { reg.UnregisterClient(clientID) })

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer bad.Close()

	f := registry.NewProviderModelsFetcher(reg)
	if err := f.FetchAndMerge(context.Background(), registry.ProviderConfig{Name: "openrouter", BaseURL: bad.URL, APIKey: "bad"}); err == nil {
		t.Fatal("FetchAndMerge error = nil, want error")
	}

	// /v1/models keeps the value.
	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil {
		t.Fatalf("GET /v1/models: %v", err)
	}
	b1 := readAll(t, resp)
	resp.Body.Close()
	m := modelByID(t, b1, id)
	if got, ok := m["context_length"]; !ok || toFloat64(got) != 200000 {
		t.Fatalf("after error /v1/models context_length = %v (present=%v), want 200000 (last-known-good)", got, ok)
	}

	// /api.json keeps the limit.
	aj, err := http.Get(srv.URL + "/api.json")
	if err != nil {
		t.Fatalf("GET /api.json: %v", err)
	}
	b2 := readAll(t, aj)
	aj.Body.Close()
	am := apiJSONModel(t, b2, id)
	if got, ok := am["context"]; !ok || toFloat64(got) != 200000 {
		t.Fatalf("after error /api.json limit.context = %v (present=%v), want 200000 (last-known-good)", got, ok)
	}
}

// A6: onboarding a dedicated provider via openai-compatibility config surfaces its
// live context_length with no code change — the refresher merges the mock
// provider's reported window into /v1/models.
func TestAcceptance_A6_OnboardingSurfacesLiveWindow(t *testing.T) {
	srv := newAcceptanceServer(t)
	defer srv.Close()

	reg := registry.GetGlobalRegistry()
	const clientID = "acc-a6"
	const id = "acc-a6-model"
	// Pre-register with a stale/zero window, as if from a static table.
	reg.RegisterClient(clientID, "openrouter", []*registry.ModelInfo{
		{ID: id, Object: "model", OwnedBy: "openrouter", Type: "openrouter", ContextLength: 0, MaxCompletionTokens: 0},
	})
	t.Cleanup(func() { reg.UnregisterClient(clientID) })

	// Mock provider with a live /v1/models reporting the real window.
	prov := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"acc-a6-model","context_length":300000,"max_completion_tokens":32000}]}`))
	}))
	defer prov.Close()

	// Onboarding = build provider config from openai-compatibility and refresh once.
	f := registry.NewProviderModelsFetcher(reg)
	if err := f.FetchAndMerge(context.Background(), registry.ProviderConfig{Name: "openrouter", BaseURL: prov.URL, APIKey: "sk-live"}); err != nil {
		t.Fatalf("FetchAndMerge: %v", err)
	}

	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil {
		t.Fatalf("GET /v1/models: %v", err)
	}
	defer resp.Body.Close()
	m := modelByID(t, readAll(t, resp), id)
	if got, ok := m["context_length"]; !ok || toFloat64(got) != 300000 {
		t.Fatalf("onboarded context_length = %v (present=%v), want 300000 (live value)", got, ok)
	}
}

// readAll reads a response body and fails the test on error.
func readAll(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return b
}
