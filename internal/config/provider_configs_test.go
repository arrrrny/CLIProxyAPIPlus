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
		Name:    "openrouter",
		BaseURL: "https://openrouter.ai/api/v1",
		APIKey:  "sk-openrouter",
	}
	if got[0] != want {
		t.Fatalf("DedicatedProviderConfigs[0] = %+v, want %+v", got[0], want)
	}
}

func TestDedicatedProviderConfigs_EmptyWhenNone(t *testing.T) {
	cfg := &Config{}
	if got := cfg.DedicatedProviderConfigs(); len(got) != 0 {
		t.Fatalf("DedicatedProviderConfigs len = %d, want 0", len(got))
	}
}
