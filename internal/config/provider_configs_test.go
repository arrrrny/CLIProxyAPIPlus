package config

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
)

// U13 / A6: openai-compatibility blocks surface as dedicated provider configs,
// keyed by their Name, so live context windows can be fetched with no code change.
func TestDedicatedProviderConfigs_MapsEnabledProviders(t *testing.T) {
	cfg := &Config{
		OpenAICompatibility: []OpenAICompatibility{
			{
				Name:    "openrouter",
				BaseURL: "https://openrouter.ai/api/v1",
				APIKeyEntries: []OpenAICompatibilityAPIKey{
					{APIKey: "sk-openrouter"},
				},
			},
			{Name: "disabled-prov", BaseURL: "https://example/v1", Disabled: true},
			{Name: "no-url", BaseURL: ""},
			{Name: "", BaseURL: "https://noname/v1"},
		},
	}

	got := cfg.DedicatedProviderConfigs()
	if len(got) != 1 {
		t.Fatalf("DedicatedProviderConfigs len = %d, want 1 (only openrouter)", len(got))
	}
	want := registry.ProviderConfig{
		Name:       "openrouter",
		BaseURL:    "https://openrouter.ai/api/v1",
		APIKey:     "sk-openrouter",
		ModelsPath: "/models",
		ParseStyle: registry.ParseStyleTopLevel,
	}
	if got[0].Name != want.Name || got[0].BaseURL != want.BaseURL || got[0].APIKey != want.APIKey ||
		got[0].ModelsPath != want.ModelsPath || got[0].ParseStyle != want.ParseStyle {
		t.Fatalf("DedicatedProviderConfigs[0] = %+v, want %+v", got[0], want)
	}
}

// U13 / A6: an openai-compatibility block for a provider with a well-known base
// URL (e.g. opencode) is usable without an explicit base-url. Sanitize keeps it and
// assigns the default endpoint, while a block with neither an explicit nor a known
// base URL is dropped (FR-002).
func TestSanitizeOpenAICompatibility_KeepsWellKnownEndpointWithoutBaseURL(t *testing.T) {
	cfg := &Config{
		OpenAICompatibility: []OpenAICompatibility{
			{Name: "opencode"},                             // known default, no base-url
			{Name: "z-ai"},                                 // known default, no base-url
			{Name: "mystery", BaseURL: ""},                 // unknown, no base-url -> dropped
			{Name: "openrouter", BaseURL: "https://ex/v1"}, // explicit base URL kept as-is
		},
	}
	cfg.SanitizeOpenAICompatibility()

	byName := make(map[string]OpenAICompatibility, len(cfg.OpenAICompatibility))
	for _, e := range cfg.OpenAICompatibility {
		byName[e.Name] = e
	}
	if len(cfg.OpenAICompatibility) != 3 {
		t.Fatalf("len = %d, want 3 (opencode, z-ai, openrouter kept; mystery dropped)", len(cfg.OpenAICompatibility))
	}
	if e, ok := byName["opencode"]; !ok || e.BaseURL != "https://opencode.ai" {
		t.Fatalf("opencode BaseURL = %q, want https://opencode.ai", e.BaseURL)
	}
	if e, ok := byName["z-ai"]; !ok || e.BaseURL != "https://api.z.ai/v1" {
		t.Fatalf("z-ai BaseURL = %q, want https://api.z.ai/v1", e.BaseURL)
	}
	if _, ok := byName["mystery"]; ok {
		t.Fatal("mystery block should have been dropped (no base-url, no known default)")
	}
	if e, ok := byName["openrouter"]; !ok || e.BaseURL != "https://ex/v1" {
		t.Fatalf("openrouter BaseURL = %q, want explicit https://ex/v1 preserved", e.BaseURL)
	}
}

// U19 (FR-002, FR-003): opencode infers its dedicated /zen/v1/models path and the
// opencode parse style; an explicit models-path / parse-style override wins.
func TestDedicatedProviderConfigs_OpenCodeInfersEndpoint(t *testing.T) {
	cfg := &Config{
		OpenAICompatibility: []OpenAICompatibility{
			{Name: "opencode", BaseURL: "https://opencode.ai", APIKeyEntries: []OpenAICompatibilityAPIKey{{APIKey: "k"}}},
			{Name: "opencode-go", BaseURL: "https://opencode.ai", APIKeyEntries: []OpenAICompatibilityAPIKey{{APIKey: "k"}}},
			{Name: "z-ai", BaseURL: "https://api.z.ai/api/v1", APIKeyEntries: []OpenAICompatibilityAPIKey{{APIKey: "k"}}},
			// explicit override
			{Name: "custom", BaseURL: "https://custom/v1", ModelsPath: "/custom/models", ParseStyle: "opencode"},
		},
	}

	got := cfg.DedicatedProviderConfigs()
	byName := make(map[string]registry.ProviderConfig, len(got))
	for _, p := range got {
		byName[p.Name] = p
	}

	oc := byName["opencode"]
	if oc.ModelsPath != "/zen/v1/models" || oc.ParseStyle != registry.ParseStyleOpenCode {
		t.Fatalf("opencode = %+v, want path /zen/v1/models + style opencode", oc)
	}
	ocg := byName["opencode-go"]
	if ocg.ModelsPath != "/zen/go/v1/models" || ocg.ParseStyle != registry.ParseStyleOpenCode {
		t.Fatalf("opencode-go = %+v, want path /zen/go/v1/models + style opencode", ocg)
	}
	zai := byName["z-ai"]
	if zai.ModelsPath != "/models" || zai.ParseStyle != registry.ParseStyleTopLevel {
		t.Fatalf("z-ai = %+v, want path /models + style top-level", zai)
	}
	custom := byName["custom"]
	if custom.ModelsPath != "/custom/models" || custom.ParseStyle != registry.ParseStyleOpenCode {
		t.Fatalf("custom = %+v, want explicit override /custom/models + opencode", custom)
	}
}

func TestDedicatedProviderConfigs_EmptyWhenNone(t *testing.T) {
	cfg := &Config{}
	if got := cfg.DedicatedProviderConfigs(); len(got) != 0 {
		t.Fatalf("DedicatedProviderConfigs len = %d, want 0", len(got))
	}
}
