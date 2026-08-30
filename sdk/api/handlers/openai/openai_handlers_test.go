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

// modelsResponse is the OpenAI /v1/models list shape.
type modelsResponse struct {
	Data []map[string]any `json:"data"`
}

// modelByID returns the first entry in the /v1/models response whose id matches.
func modelByID(t *testing.T, body []byte, id string) map[string]any {
	t.Helper()
	var resp modelsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal /v1/models response: %v", err)
	}
	for _, m := range resp.Data {
		if got, _ := m["id"].(string); got == id {
			return m
		}
	}
	t.Fatalf("model %q not found in /v1/models response: %s", id, string(body))
	return nil
}

// TestOpenAIModels_EmitsContextLengthWhenPositiveAndOmitsWhenZero covers U1 and
// U3: OpenAIModels must include the registry's context_length when it is > 0 and
// must NOT emit a context_length key when the model has none.
func TestOpenAIModels_EmitsContextLengthWhenPositiveAndOmitsWhenZero(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := registry.GetGlobalRegistry()
	const clientID = "us1-context-length"
	reg.RegisterClient(clientID, "openai-compatibility", []*registry.ModelInfo{
		{ID: "us1-model-with-window", Object: "model", OwnedBy: "test", Type: "openai-compatibility", ContextLength: 200000, MaxCompletionTokens: 64000},
		{ID: "us1-model-without-window", Object: "model", OwnedBy: "test", Type: "openai-compatibility"},
	})
	t.Cleanup(func() { reg.UnregisterClient(clientID) })

	h := NewOpenAIAPIHandler(handlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, nil))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	h.OpenAIModels(c)

	withWindow := modelByID(t, w.Body.Bytes(), "us1-model-with-window")
	if got, ok := withWindow["context_length"]; !ok || toFloat64(got) != 200000 {
		t.Fatalf("with-window context_length = %v (present=%v), want 200000", got, ok)
	}
	// U2: max_completion_tokens is included when > 0.
	if got, ok := withWindow["max_completion_tokens"]; !ok || toFloat64(got) != 64000 {
		t.Fatalf("with-window max_completion_tokens = %v (present=%v), want 64000", got, ok)
	}

	withoutWindow := modelByID(t, w.Body.Bytes(), "us1-model-without-window")
	if _, ok := withoutWindow["context_length"]; ok {
		t.Fatalf("without-window should omit context_length, got %v", withoutWindow["context_length"])
	}
	// U4: max_completion_tokens is omitted when the model has none.
	if _, ok := withoutWindow["max_completion_tokens"]; ok {
		t.Fatalf("without-window should omit max_completion_tokens, got %v", withoutWindow["max_completion_tokens"])
	}
}

// toFloat64 normalizes JSON numbers (float64) for assertions.
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return 0
	}
}
