package openai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/api/handlers"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

// catalogModel is one entry under cliproxy.models in the /api.json response.
type catalogModel struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	ContextLength       int            `json:"context_length"`
	MaxCompletionTokens int            `json:"max_completion_tokens"`
	Limit               map[string]any `json:"limit"`
}

// catalogResponse is the models.dev-format /api.json document.
type catalogResponse struct {
	Cliproxy struct {
		Models map[string]catalogModel `json:"models"`
	} `json:"cliproxy"`
}

func TestAPIJSON_ExposesLimitContextOutputForKnownWindows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := registry.GetGlobalRegistry()
	const clientID = "us4-api-json"
	reg.RegisterClient(clientID, "openai-compatibility", []*registry.ModelInfo{
		{ID: "us4-model-with-window", Object: "model", OwnedBy: "test", Type: "openai-compatibility", DisplayName: "With Window", ContextLength: 200000, MaxCompletionTokens: 64000},
		{ID: "us4-model-without-window", Object: "model", OwnedBy: "test", Type: "openai-compatibility", DisplayName: "No Window"},
		{ID: "ocd/deepseek-v4-flash-free", Object: "model", OwnedBy: "ocd", Type: "openai-compatibility", DisplayName: "DeepSeek V4 Flash Free", ContextLength: 1000000, MaxCompletionTokens: 64000},
	})
	t.Cleanup(func() { reg.UnregisterClient(clientID) })

	h := NewOpenAIAPIHandler(handlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, nil))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api.json", nil)
	h.APIJSON(c)

	var resp catalogResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal /api.json: %v", err)
	}

	// U16: known window exposes limit.context / limit.output.
	withWindow, ok := resp.Cliproxy.Models["us4-model-with-window"]
	if !ok {
		t.Fatalf("us4-model-with-window missing from catalog: %s", string(w.Body.Bytes()))
	}
	if withWindow.ContextLength != 200000 {
		t.Fatalf("context_length = %d, want 200000", withWindow.ContextLength)
	}
	if withWindow.Limit["context"] != float64(200000) {
		t.Fatalf("limit.context = %v, want 200000", withWindow.Limit["context"])
	}
	if withWindow.Limit["output"] != float64(64000) {
		t.Fatalf("limit.output = %v, want 64000", withWindow.Limit["output"])
	}

	// U17: model with no window omits limit rather than emitting 0.
	withoutWindow, ok := resp.Cliproxy.Models["us4-model-without-window"]
	if !ok {
		t.Fatalf("us4-model-without-window missing from catalog: %s", string(w.Body.Bytes()))
	}
	if withoutWindow.Limit != nil {
		t.Fatalf("without-window should omit limit, got %v", withoutWindow.Limit)
	}

	// U18: namespaced id preserved verbatim as the catalog key.
	if _, ok := resp.Cliproxy.Models["ocd/deepseek-v4-flash-free"]; !ok {
		t.Fatalf("namespaced id not preserved verbatim; keys: %v", modelKeys(resp.Cliproxy.Models))
	}
	if entry, ok := resp.Cliproxy.Models["ocd/deepseek-v4-flash-free"]; ok && entry.ContextLength != 1000000 {
		t.Fatalf("namespaced model context_length = %d, want 1000000", entry.ContextLength)
	}
}

// TestAPIJSON_GeminiNativeLimitsFallback verifies that a Gemini-family model which
// only carries inputTokenLimit/outputTokenLimit (never context_length) still
// exposes a context window on the OpenAI-compatible /api.json endpoint via the
// registry's openai-handler fallback.
func TestAPIJSON_GeminiNativeLimitsFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := registry.GetGlobalRegistry()
	const clientID = "us4-gemini-native"
	reg.RegisterClient(clientID, "gemini", []*registry.ModelInfo{
		{ID: "gemini-2.5-pro", Object: "model", OwnedBy: "google", Type: "gemini", DisplayName: "Gemini 2.5 Pro", InputTokenLimit: 1048576, OutputTokenLimit: 65536},
	})
	t.Cleanup(func() { reg.UnregisterClient(clientID) })

	h := NewOpenAIAPIHandler(handlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, nil))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api.json", nil)
	h.APIJSON(c)

	var resp catalogResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal /api.json: %v", err)
	}

	entry, ok := resp.Cliproxy.Models["gemini-2.5-pro"]
	if !ok {
		t.Fatalf("gemini-2.5-pro missing from catalog: %s", string(w.Body.Bytes()))
	}
	if entry.ContextLength != 1048576 {
		t.Fatalf("context_length = %d, want 1048576 (from inputTokenLimit fallback)", entry.ContextLength)
	}
	if entry.Limit["context"] != float64(1048576) {
		t.Fatalf("limit.context = %v, want 1048576", entry.Limit["context"])
	}
	if entry.Limit["output"] != float64(65536) {
		t.Fatalf("limit.output = %v, want 65536 (from outputTokenLimit fallback)", entry.Limit["output"])
	}
}

func modelKeys(m map[string]catalogModel) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
