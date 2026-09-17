// Fork-owned dedicated model provider helpers (spec 002).

package config

import (
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/constant"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
)

// DedicatedProviderConfigs returns the OpenAI-compatible providers that should be
// refreshed from their live /v1/models endpoints for accurate context windows
// (FR-002, FR-004, FR-007). Each entry maps an openai-compatibility block — keyed
// by its Name — to a registry.ProviderConfig carrying the base URL and the first
// API key. Disabled blocks and those missing a Name or BaseURL are skipped (the
// latter are also dropped by SanitizeOpenAICompatibility). The provider Name is
// preserved verbatim so the registry override is distinguishable from generic
// "openai" (U13, A6).
func (cfg *Config) DedicatedProviderConfigs() []registry.ProviderConfig {
	out := make([]registry.ProviderConfig, 0, len(cfg.OpenAICompatibility))
	for _, c := range cfg.OpenAICompatibility {
		if c.Disabled || c.Name == "" {
			continue
		}
		baseURL := c.BaseURL
		if baseURL == "" {
			baseURL = defaultProviderBaseURL(c.Name)
		}
		if baseURL == "" {
			// No base URL and no known default: cannot reach a models endpoint.
			continue
		}
		pc := registry.ProviderConfig{
			Name:       c.Name,
			BaseURL:    baseURL,
			ModelsPath: defaultProviderModelsPath(c.Name),
			ParseStyle: defaultProviderParseStyle(c.Name),
		}
		if c.ModelsPath != "" {
			pc.ModelsPath = c.ModelsPath
		}
		if c.ParseStyle != "" {
			pc.ParseStyle = registry.ProviderParseStyle(c.ParseStyle)
		}
		if len(c.APIKeyEntries) > 0 {
			pc.APIKey = c.APIKeyEntries[0].APIKey
		}
		for _, m := range c.Models {
			clientID := m.Alias
			if clientID == "" {
				clientID = m.Name
			}
			if clientID == "" {
				continue
			}
			pc.ConfiguredModels = append(pc.ConfiguredModels, registry.ConfiguredModel{
				ClientID:     clientID,
				UpstreamName: m.Name,
			})
		}
		out = append(out, pc)
	}
	return out
}

// defaultProviderBaseURL returns the well-known base URL for a dedicated provider
// name when the user omitted base-url in their openai-compatibility block (FR-002).
// Returns "" for unknown provider names so blocks without an explicit base URL are
// skipped rather than pointed at a guessed endpoint.
func defaultProviderBaseURL(name string) string {
	switch strings.ToLower(name) {
	case constant.OpenCode:
		return "https://opencode.ai"
	case constant.OpenCodeGo:
		return "https://opencode.ai"
	case constant.OpenRouter:
		return "https://openrouter.ai/api/v1"
	case constant.ZAi:
		return "https://api.z.ai/v1"
	default:
		return ""
	}
}

// defaultProviderModelsPath returns the provider-specific /models endpoint path
// appended to BaseURL when fetching live context windows (FR-002).
func defaultProviderModelsPath(name string) string {
	switch strings.ToLower(name) {
	// OpenCode Go (mimo) and OpenCode (Zen) use distinct endpoints:
	//   go  -> https://opencode.ai/zen/go/v1/models
	//   zen -> https://opencode.ai/zen/v1/models
	case constant.OpenCodeGo:
		return "/zen/go/v1/models"
	case constant.OpenCode:
		return "/zen/v1/models"
	case constant.OpenRouter, constant.ZAi:
		return "/models"
	default:
		// Documented base URLs already include /v1 (e.g. openrouter
		// "https://openrouter.ai/api/v1"), so the suffix is just "/models".
		// Returning "/v1/models" here would double the /v1 segment.
		return "/models"
	}
}

// defaultProviderParseStyle returns how the provider's /models response is parsed
// for context windows. "opencode" endpoints return ids only; windows are sourced
// from the curated table. All others use the top-level shape.
func defaultProviderParseStyle(name string) registry.ProviderParseStyle {
	switch strings.ToLower(name) {
	case constant.OpenCode, constant.OpenCodeGo:
		return registry.ParseStyleOpenCode
	default:
		return registry.ParseStyleTopLevel
	}
}

// ---- Kiro (AWS CodeWhisperer) configuration types (fork-owned) ----

// KiroKey represents the configuration for Kiro (AWS CodeWhisperer) authentication.
type KiroKey struct {
	// TokenFile is the path to the Kiro token file (default: ~/.aws/sso/cache/kiro-auth-token.json)
	TokenFile string `yaml:"token-file,omitempty" json:"token-file,omitempty"`

	// AccessToken is the OAuth access token for direct configuration.
	AccessToken string `yaml:"access-token,omitempty" json:"access-token,omitempty"`

	// RefreshToken is the OAuth refresh token for token renewal.
	RefreshToken string `yaml:"refresh-token,omitempty" json:"refresh-token,omitempty"`

	// ProfileArn is the AWS CodeWhisperer profile ARN.
	ProfileArn string `yaml:"profile-arn,omitempty" json:"profile-arn,omitempty"`

	// Region is the AWS region (default: us-east-1).
	Region string `yaml:"region,omitempty" json:"region,omitempty"`

	// StartURL is the IAM Identity Center (IDC) start URL for SSO login.
	StartURL string `yaml:"start-url,omitempty" json:"start-url,omitempty"`

	// ProxyURL optionally overrides the global proxy for this configuration.
	ProxyURL string `yaml:"proxy-url,omitempty" json:"proxy-url,omitempty"`

	// AgentTaskType sets the Kiro API task type. Known values: "vibe", "dev", "chat".
	// Leave empty to let API use defaults. Different values may inject different system prompts.
	AgentTaskType string `yaml:"agent-task-type,omitempty" json:"agent-task-type,omitempty"`

	// PreferredEndpoint sets the preferred Kiro API endpoint/quota.
	// Values: "codewhisperer" (default, IDE quota) or "amazonq" (CLI quota).
	PreferredEndpoint string `yaml:"preferred-endpoint,omitempty" json:"preferred-endpoint,omitempty"`
}

// KiroFingerprintConfig defines a global fingerprint configuration for Kiro requests.
// When configured, all Kiro requests will use this fixed fingerprint instead of random generation.
// Empty fields will fall back to random selection from built-in pools.
type KiroFingerprintConfig struct {
	OIDCSDKVersion      string `yaml:"oidc-sdk-version,omitempty" json:"oidc-sdk-version,omitempty"`
	RuntimeSDKVersion   string `yaml:"runtime-sdk-version,omitempty" json:"runtime-sdk-version,omitempty"`
	StreamingSDKVersion string `yaml:"streaming-sdk-version,omitempty" json:"streaming-sdk-version,omitempty"`
	OSType              string `yaml:"os-type,omitempty" json:"os-type,omitempty"`
	OSVersion           string `yaml:"os-version,omitempty" json:"os-version,omitempty"`
	NodeVersion         string `yaml:"node-version,omitempty" json:"node-version,omitempty"`
	KiroVersion         string `yaml:"kiro-version,omitempty" json:"kiro-version,omitempty"`
	KiroHash            string `yaml:"kiro-hash,omitempty" json:"kiro-hash,omitempty"`
}
