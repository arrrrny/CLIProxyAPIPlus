package openai

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIJSON serves a models.dev-format catalog built from the enriched registry
// (US4 / FR-009). It exposes per-model limit.context / limit.output derived from
// the registry's context_length / max_completion_tokens, so downstream consumers
// (e.g. Quotio's ProxyBridge) read correct limits by construction.
// Provider visibility is governed by the config.propagate_in_api allowlist: only
// providers mapped to true are published. An empty/unset allowlist publishes every
// provider (legacy behavior). This never changes request routing or /v1/models.
func (h *OpenAIAPIHandler) APIJSON(c *gin.Context) {
	models := h.Models()
	if h.Cfg != nil && len(h.Cfg.PropagateInAPI) > 0 {
		models = filterForAPIPropagation(models, h.Cfg.PropagateInAPI)
	}
	c.JSON(http.StatusOK, buildCatalog(models, apiBaseURL(c.Request)))
}

// apiBaseURL derives the proxy's public base URL (models.dev catalog shape) from the
// incoming request so the published /api.json is self-describing. Downstream consumers
// use it as the OpenAI-compatible endpoint, which always points at /v1.
func apiBaseURL(r *http.Request) string {
	scheme := "http"
	if r != nil && r.TLS != nil {
		scheme = "https"
	}
	host := ""
	if r != nil {
		host = r.Host
	}
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host + "/v1"
}

// buildCatalog assembles the models.dev-format document from the available models.
// limit is omitted entirely for a model with no known context window (U17) so a
// downstream consumer never reads a fabricated zero. apiBaseURL is the proxy's
// public base URL (+/v1) published for downstream OpenAI-compatible consumers.
func buildCatalog(models []map[string]any, apiBaseURL string) map[string]any {
	entries := make(map[string]any, len(models))
	for _, m := range models {
		id, _ := m["id"].(string)
		if id == "" {
			continue
		}
		entry := map[string]any{
			"id":   id,
			"name": modelName(m),
		}
		if cl, ok := m["context_length"]; ok && positiveInt(cl) {
			clVal := toIntValue(cl)
			entry["context_length"] = clVal
			limit := map[string]any{"context": clVal}
			if mc, ok := m["max_completion_tokens"]; ok && positiveInt(mc) {
				limit["output"] = toIntValue(mc)
			}
			entry["limit"] = limit
		}
		if mc, ok := m["max_completion_tokens"]; ok && positiveInt(mc) {
			entry["max_completion_tokens"] = toIntValue(mc)
		}
		entries[id] = entry
	}
	return map[string]any{
		"quotio": map[string]any{
			"id":     "cliproxy",
			"name":   "CLIProxyAPI",
			"api":    apiBaseURL,
			"type":   "openai",
			"env":    []string{"CLIPROXY_API_KEY"},
			"models": entries,
		},
	}
}

// modelName returns the display_name, falling back to the id.
func modelName(m map[string]any) string {
	if name, ok := m["display_name"].(string); ok && name != "" {
		return name
	}
	if id, ok := m["id"].(string); ok {
		return id
	}
	return ""
}

// toIntValue normalizes int / float64 registry values to int for the catalog.
func toIntValue(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}

// filterForAPIPropagation removes models whose provider (the model map "type"
// field, e.g. "kiro", "claude", "opencode") is not enabled in allowlist.
// An empty/nil allowlist publishes everything (legacy behavior). When an allowlist
// is set, only providers explicitly mapped to true are published; a model with no
// resolvable provider is hidden because it cannot match any allowlisted entry.
func filterForAPIPropagation(models []map[string]any, allowlist map[string]bool) []map[string]any {
	if len(allowlist) == 0 {
		return models
	}
	out := make([]map[string]any, 0, len(models))
	for _, m := range models {
		provider, _ := m["type"].(string)
		if provider == "" {
			continue
		}
		if allowlist[provider] {
			out = append(out, m)
		}
	}
	return out
}
