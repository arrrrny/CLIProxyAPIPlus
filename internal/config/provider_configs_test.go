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
	if got[0] != want {
		t.Fatalf("DedicatedProviderConfigs[0] = %+v, want %+v", got[0], want)
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
