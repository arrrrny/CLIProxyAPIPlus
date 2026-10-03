// Package registry provides model definitions and lookup helpers for various AI providers.
// Static model metadata is loaded from the embedded models.json file and can be refreshed from network.
package registry

import (
	"strings"
)

const (
	codexBuiltinImage15ModelID      = "gpt-image-1.5"
	codexBuiltinImageModelID        = "gpt-image-2"
	xaiBuiltinImageModelID          = "grok-imagine-image"
	xaiBuiltinImageQualityModelID   = "grok-imagine-image-quality"
	xaiBuiltinVideoModelID          = "grok-imagine-video"
	xaiBuiltinVideo15PreviewModelID = "grok-imagine-video-1.5-preview"
)

// staticModelsJSON mirrors the top-level structure of models.json.
type staticModelsJSON struct {
	Claude      []*ModelInfo `json:"claude"`
	Gemini      []*ModelInfo `json:"gemini"`
	Vertex      []*ModelInfo `json:"vertex"`
	AIStudio    []*ModelInfo `json:"aistudio"`
	CodexFree   []*ModelInfo `json:"codex-free"`
	CodexTeam   []*ModelInfo `json:"codex-team"`
	CodexPlus   []*ModelInfo `json:"codex-plus"`
	CodexPro    []*ModelInfo `json:"codex-pro"`
	Kimi        []*ModelInfo `json:"kimi"`
	Antigravity []*ModelInfo `json:"antigravity"`
	XAI         []*ModelInfo `json:"xai"`
}

// GetClaudeModels returns the standard Claude model definitions.
func GetClaudeModels() []*ModelInfo {
	return cloneModelInfos(getModels().Claude)
}

// GetGeminiModels returns the standard Gemini model definitions.
func GetGeminiModels() []*ModelInfo {
	return cloneModelInfos(getModels().Gemini)
}

// GetGeminiVertexModels returns Gemini model definitions for Vertex AI.
func GetGeminiVertexModels() []*ModelInfo {
	return cloneModelInfos(getModels().Vertex)
}

// GetAIStudioModels returns model definitions for AI Studio.
func GetAIStudioModels() []*ModelInfo {
	return cloneModelInfos(getModels().AIStudio)
}

// GetCodexFreeModels returns model definitions for the Codex free plan tier.
func GetCodexFreeModels() []*ModelInfo {
	return WithCodexBuiltins(cloneModelInfos(getModels().CodexFree))
}

// GetCodexTeamModels returns model definitions for the Codex team plan tier.
func GetCodexTeamModels() []*ModelInfo {
	return WithCodexBuiltins(cloneModelInfos(getModels().CodexTeam))
}

// GetCodexPlusModels returns model definitions for the Codex plus plan tier.
func GetCodexPlusModels() []*ModelInfo {
	return WithCodexBuiltins(cloneModelInfos(getModels().CodexPlus))
}

// GetCodexProModels returns model definitions for the Codex pro plan tier.
func GetCodexProModels() []*ModelInfo {
	return WithCodexBuiltins(cloneModelInfos(getModels().CodexPro))
}

// GetKimiModels returns the standard Kimi (Moonshot AI) model definitions.
func GetKimiModels() []*ModelInfo {
	return cloneModelInfos(getModels().Kimi)
}

// antigravityLocalModels are Antigravity models this build keeps in the catalog
// no matter what the shared registry (router-for-me/models) publishes.
//
// The registry omits models that Antigravity does serve, which strands requests
// for them with "unknown provider for model". Every entry below was confirmed
// present in v1internal:fetchAvailableModels for all 11 Antigravity accounts and
// answering 200 on v1internal:generateContent, so they are reinstated here.
//
// The Claude pair is also the floor of the Claude line on this provider:
// Antigravity publishes no successor to them.
var antigravityLocalModels = []*ModelInfo{
	{
		ID:                  "claude-opus-4-6-thinking",
		Object:              "model",
		OwnedBy:             "antigravity",
		Type:                "antigravity",
		DisplayName:         "Claude Opus 4.6 (Thinking)",
		ContextLength:       200000,
		MaxCompletionTokens: 64000,
		Thinking: &ThinkingSupport{
			Min:            1024,
			Max:            64000,
			ZeroAllowed:    true,
			DynamicAllowed: true,
		},
	},
	{
		ID:                  "claude-sonnet-4-6",
		Object:              "model",
		OwnedBy:             "antigravity",
		Type:                "antigravity",
		DisplayName:         "Claude Sonnet 4.6 (Thinking)",
		ContextLength:       200000,
		MaxCompletionTokens: 64000,
		Thinking: &ThinkingSupport{
			Min:            1024,
			Max:            64000,
			ZeroAllowed:    true,
			DynamicAllowed: true,
		},
	},
	{
		ID:                  "gemini-3-flash-agent",
		Object:              "model",
		OwnedBy:             "antigravity",
		Type:                "antigravity",
		DisplayName:         "Gemini 3.5 Flash (High)",
		ContextLength:       1048576,
		MaxCompletionTokens: 65536,
		Thinking: &ThinkingSupport{
			Min:            128,
			Max:            32768,
			DynamicAllowed: true,
			Levels:         []string{"minimal", "low", "medium", "high"},
		},
	},
	{
		ID:                  "gemini-3.5-flash-extra-low",
		Object:              "model",
		OwnedBy:             "antigravity",
		Type:                "antigravity",
		DisplayName:         "Gemini 3.5 Flash (Low)",
		ContextLength:       1048576,
		MaxCompletionTokens: 65535,
		Thinking: &ThinkingSupport{
			Min:            1,
			Max:            65535,
			DynamicAllowed: true,
			Levels:         []string{"low", "medium", "high"},
		},
	},
	{
		ID:                  "gemini-3.5-flash-low",
		Object:              "model",
		OwnedBy:             "antigravity",
		Type:                "antigravity",
		DisplayName:         "Gemini 3.5 Flash (Medium)",
		ContextLength:       1048576,
		MaxCompletionTokens: 65535,
		Thinking: &ThinkingSupport{
			Min:            1,
			Max:            65535,
			DynamicAllowed: true,
			Levels:         []string{"low", "medium", "high"},
		},
	},
}

// antigravityFlashTierModels are the per-tier Gemini 3.6/3.7/3.8 Flash variants.
// Antigravity ships them alongside each family's -high entry and lists them in
// tieredModelIds; only the -high member of each family reaches the shared
// registry. They are generated from a shared template because the families are
// identical apart from version and tier.
func antigravityFlashTierModels() []*ModelInfo {
	tiers := []struct {
		suffix      string
		displayTier string
	}{
		{"low", "Low"},
		{"medium", "Medium"},
		{"tiered", "Tiered"},
	}
	versions := []string{"3.6", "3.7", "3.8"}

	models := make([]*ModelInfo, 0, len(versions)*len(tiers))
	for _, version := range versions {
		for _, tier := range tiers {
			models = append(models, &ModelInfo{
				ID:                  "gemini-" + version + "-flash-" + tier.suffix,
				Object:              "model",
				OwnedBy:             "antigravity",
				Type:                "antigravity",
				DisplayName:         "Gemini " + version + " Flash (" + tier.displayTier + ")",
				ContextLength:       1048576,
				MaxCompletionTokens: 65536,
				Thinking: &ThinkingSupport{
					Min:            1,
					Max:            65535,
					DynamicAllowed: true,
					Levels:         []string{"minimal", "low", "medium", "high"},
				},
				SupportedInputModalities:  []string{"text", "image", "audio", "video"},
				SupportedOutputModalities: []string{"text"},
			})
		}
	}
	return models
}

// antigravityUnsupportedModelIDs are catalog entries no Antigravity account can
// execute. The shared registry advertises them, so without this filter they are
// published through /v1/models and the models.dev catalog while every request
// against them fails upstream.
//
// Verified 2026-10-03 against Antigravity Hub 2.19.1, including a Google AI
// subscription account: v1internal:fetchAvailableModels lists 33 models whose
// only Claude entries are claude-opus-4-6-thinking and claude-sonnet-4-6 (on
// cloudcode-pa, daily-cloudcode-pa and the daily sandbox alike), and
// v1internal:generateContent answers 404 NOT_FOUND for the IDs below while the
// two IDs above return 200. Remove an entry once the upstream catalog serves it.
var antigravityUnsupportedModelIDs = map[string]struct{}{
	"claude-opus-5-5-high":   {},
	"claude-sonnet-5-5-high": {},
}

// antigravityExcludedModelIDs are models this build deliberately does not
// publish. They are served upstream, so this is an operator preference rather
// than a capability gap; keep the two lists separate so the distinction stays
// obvious when the catalog is next revisited.
var antigravityExcludedModelIDs = map[string]struct{}{
	"gemini-2.5-flash":          {},
	"gemini-2.5-flash-lite":     {},
	"gemini-2.5-flash-thinking": {},
	"gemini-2.5-pro":            {},
}

// GetAntigravityModels returns the standard Antigravity model definitions.
func GetAntigravityModels() []*ModelInfo {
	models := cloneModelInfos(getModels().Antigravity)
	models = dropModelInfos(models, antigravityUnsupportedModelIDs)
	models = dropModelInfos(models, antigravityExcludedModelIDs)

	// Reinstates the Antigravity models the shared registry omits.
	models = upsertModelInfos(models, antigravityLocalModels...)
	models = upsertModelInfos(models, antigravityFlashTierModels()...)

	return models
}

// GetCodeBuddyModels returns the available models for CodeBuddy (Tencent).
// These models are served through the copilot.tencent.com API.
func GetCodeBuddyModels() []*ModelInfo {
	now := int64(1748044800) // 2025-05-24
	return []*ModelInfo{
		{
			ID:                  "auto",
			Object:              "model",
			Created:             now,
			OwnedBy:             "tencent",
			Type:                "codebuddy",
			DisplayName:         "Auto",
			Description:         "Automatic model selection via CodeBuddy",
			ContextLength:       128000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "glm-5v-turbo",
			Object:              "model",
			Created:             now,
			OwnedBy:             "tencent",
			Type:                "codebuddy",
			DisplayName:         "GLM-5v Turbo",
			Description:         "GLM-5v Turbo via CodeBuddy",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "glm-5.1",
			Object:              "model",
			Created:             now,
			OwnedBy:             "tencent",
			Type:                "codebuddy",
			DisplayName:         "GLM-5.1",
			Description:         "GLM-5.1 via CodeBuddy",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "glm-5.0-turbo",
			Object:              "model",
			Created:             now,
			OwnedBy:             "tencent",
			Type:                "codebuddy",
			DisplayName:         "GLM-5.0 Turbo",
			Description:         "GLM-5.0 Turbo via CodeBuddy",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "glm-5.0",
			Object:              "model",
			Created:             now,
			OwnedBy:             "tencent",
			Type:                "codebuddy",
			DisplayName:         "GLM-5.0",
			Description:         "GLM-5.0 via CodeBuddy",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "glm-4.7",
			Object:              "model",
			Created:             now,
			OwnedBy:             "tencent",
			Type:                "codebuddy",
			DisplayName:         "GLM-4.7",
			Description:         "GLM-4.7 via CodeBuddy",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "minimax-m2.7",
			Object:              "model",
			Created:             now,
			OwnedBy:             "tencent",
			Type:                "codebuddy",
			DisplayName:         "MiniMax M2.7",
			Description:         "MiniMax M2.7 via CodeBuddy",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "kimi-k2.5",
			Object:              "model",
			Created:             now,
			OwnedBy:             "tencent",
			Type:                "codebuddy",
			DisplayName:         "Kimi K2.5",
			Description:         "Kimi K2.5 via CodeBuddy",
			ContextLength:       256000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "kimi-k2-thinking",
			Object:              "model",
			Created:             now,
			OwnedBy:             "tencent",
			Type:                "codebuddy",
			DisplayName:         "Kimi K2 Thinking",
			Description:         "Kimi K2 Thinking via CodeBuddy",
			ContextLength:       256000,
			MaxCompletionTokens: 32768,
			Thinking:            &ThinkingSupport{ZeroAllowed: true},
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "deepseek-v3-2-volc",
			Object:              "model",
			Created:             now,
			OwnedBy:             "tencent",
			Type:                "codebuddy",
			DisplayName:         "DeepSeek V3.2 (Volc)",
			Description:         "DeepSeek V3.2 via CodeBuddy",
			ContextLength:       128000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
	}
}

// AntigravityWebSearchModelFor returns the Antigravity model that should run a
// native web search request for modelID.
func AntigravityWebSearchModelFor(modelID string) string {
	modelID = normalizeAntigravityCapabilityModelID(modelID)
	if modelID == "" {
		return ""
	}
	for _, model := range GetGlobalRegistry().GetAvailableModelsByProvider("antigravity") {
		if model == nil {
			continue
		}
		currentModelID := normalizeAntigravityCapabilityModelID(model.ID)
		if currentModelID == "" {
			continue
		}
		if currentModelID == modelID {
			if model.SupportsWebSearch {
				return currentModelID
			}
			return ""
		}
	}
	return ""
}

// GetXAIModels returns the standard xAI Grok model definitions.
func GetXAIModels() []*ModelInfo {
	return WithXAIBuiltins(cloneModelInfos(getModels().XAI))
}

// WithCodexBuiltins injects hard-coded Codex-only model definitions that should
// not depend on remote models.json updates. Built-ins replace any matching IDs
// already present in the provided slice.
func WithCodexBuiltins(models []*ModelInfo) []*ModelInfo {
	return upsertModelInfos(models, codexBuiltinImage15ModelInfo(), codexBuiltinImageModelInfo())
}

// WithXAIBuiltins injects hard-coded xAI image/video model definitions that should
// not depend on remote models.json updates.
func WithXAIBuiltins(models []*ModelInfo) []*ModelInfo {
	return upsertModelInfos(models, xaiBuiltinImageModelInfo(), xaiBuiltinImageQualityModelInfo(), xaiBuiltinVideoModelInfo(), xaiBuiltinVideo15PreviewModelInfo())
}

func normalizeAntigravityCapabilityModelID(modelID string) string {
	modelID = strings.ToLower(strings.TrimSpace(modelID))
	if open := strings.LastIndex(modelID, "("); open >= 0 && strings.HasSuffix(modelID, ")") {
		modelID = strings.TrimSpace(modelID[:open])
	}
	return modelID
}

func codexBuiltinImage15ModelInfo() *ModelInfo {
	return &ModelInfo{
		ID:          codexBuiltinImage15ModelID,
		Object:      "model",
		Created:     1704067200, // 2024-01-01
		OwnedBy:     "openai",
		Type:        "openai",
		DisplayName: "GPT Image 1.5",
		Version:     codexBuiltinImage15ModelID,
	}
}

func codexBuiltinImageModelInfo() *ModelInfo {
	return &ModelInfo{
		ID:          codexBuiltinImageModelID,
		Object:      "model",
		Created:     1704067200, // 2024-01-01
		OwnedBy:     "openai",
		Type:        "openai",
		DisplayName: "GPT Image 2",
		Version:     codexBuiltinImageModelID,
	}
}

func xaiBuiltinImageModelInfo() *ModelInfo {
	return &ModelInfo{
		ID:          xaiBuiltinImageModelID,
		Object:      "model",
		Created:     1735689600, // 2025-01-01
		OwnedBy:     "xai",
		Type:        "xai",
		DisplayName: "Grok Imagine Image",
		Name:        xaiBuiltinImageModelID,
		Description: "xAI Grok image generation model.",
	}
}

func xaiBuiltinImageQualityModelInfo() *ModelInfo {
	return &ModelInfo{
		ID:          xaiBuiltinImageQualityModelID,
		Object:      "model",
		Created:     1735689600, // 2025-01-01
		OwnedBy:     "xai",
		Type:        "xai",
		DisplayName: "Grok Imagine Image Quality",
		Name:        xaiBuiltinImageQualityModelID,
		Description: "xAI Grok higher-fidelity image generation model.",
	}
}

func xaiBuiltinVideoModelInfo() *ModelInfo {
	return &ModelInfo{
		ID:          xaiBuiltinVideoModelID,
		Object:      "model",
		Created:     1735689600, // 2025-01-01
		OwnedBy:     "xai",
		Type:        "xai",
		DisplayName: "Grok Imagine Video",
		Name:        xaiBuiltinVideoModelID,
		Description: "xAI Grok video generation model.",
	}
}

func xaiBuiltinVideo15PreviewModelInfo() *ModelInfo {
	return &ModelInfo{
		ID:          xaiBuiltinVideo15PreviewModelID,
		Object:      "model",
		Created:     1735689600, // 2025-01-01
		OwnedBy:     "xai",
		Type:        "xai",
		DisplayName: "Grok Imagine Video 1.5 Preview",
		Name:        xaiBuiltinVideo15PreviewModelID,
		Description: "xAI Grok preview video generation model.",
	}
}

func upsertModelInfos(models []*ModelInfo, extras ...*ModelInfo) []*ModelInfo {
	if len(extras) == 0 {
		return models
	}

	extraIDs := make(map[string]struct{}, len(extras))
	extraList := make([]*ModelInfo, 0, len(extras))
	for _, extra := range extras {
		if extra == nil {
			continue
		}
		id := strings.TrimSpace(extra.ID)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, exists := extraIDs[key]; exists {
			continue
		}
		extraIDs[key] = struct{}{}
		extraList = append(extraList, cloneModelInfo(extra))
	}

	if len(extraList) == 0 {
		return models
	}

	filtered := make([]*ModelInfo, 0, len(models)+len(extraList))
	for _, model := range models {
		if model == nil {
			continue
		}
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		if _, exists := extraIDs[strings.ToLower(id)]; exists {
			continue
		}
		filtered = append(filtered, model)
	}

	filtered = append(filtered, extraList...)
	return filtered
}

// dropModelInfos returns models with the IDs in exclude removed. Comparison is
// case-insensitive and whitespace-trimmed, matching upsertModelInfos. A nil or
// empty exclude set returns models unchanged.
func dropModelInfos(models []*ModelInfo, exclude map[string]struct{}) []*ModelInfo {
	if len(models) == 0 || len(exclude) == 0 {
		return models
	}

	filtered := make([]*ModelInfo, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		if _, drop := exclude[strings.ToLower(id)]; drop {
			continue
		}
		filtered = append(filtered, model)
	}

	return filtered
}

// cloneModelInfos returns a shallow copy of the slice with each element deep-cloned.
func cloneModelInfos(models []*ModelInfo) []*ModelInfo {
	if len(models) == 0 {
		return nil
	}
	out := make([]*ModelInfo, len(models))
	for i, m := range models {
		out[i] = cloneModelInfo(m)
	}
	return out
}

// GetStaticModelDefinitionsByChannel returns static model definitions for a given channel/provider.
// It returns nil when the channel is unknown.
//
// Supported channels:
//   - claude
//   - gemini
//   - vertex
//   - aistudio
//   - codex
//   - kimi
//   - kilo
//   - github-copilot
//   - amazonq
//   - antigravity (returns static overrides only)
func GetStaticModelDefinitionsByChannel(channel string) []*ModelInfo {
	key := strings.ToLower(strings.TrimSpace(channel))
	switch key {
	case "claude":
		return GetClaudeModels()
	case "gemini":
		return GetGeminiModels()
	case "vertex":
		return GetGeminiVertexModels()
	case "aistudio":
		return GetAIStudioModels()
	case "codex":
		return GetCodexProModels()
	case "kimi":
		return GetKimiModels()
	case "github-copilot":
		return GetGitHubCopilotModels()
	case "kiro":
		return GetKiroModels()
	case "kilo":
		return GetKiloModels()
	case "amazonq":
		return GetAmazonQModels()
	case "antigravity":
		return GetAntigravityModels()
	case "xai", "x-ai", "grok":
		return GetXAIModels()
	case "codebuddy":
		return GetCodeBuddyModels()
	case "cursor":
		return GetCursorModels()
	default:
		return nil
	}
}

// GetCursorModels returns the fallback Cursor model definitions.
func GetCursorModels() []*ModelInfo {
	return []*ModelInfo{
		{ID: "composer-2", Object: "model", OwnedBy: "cursor", Type: "cursor", DisplayName: "Composer 2", ContextLength: 200000, MaxCompletionTokens: 64000, Thinking: &ThinkingSupport{Max: 50000, DynamicAllowed: true}},
		{ID: "claude-4-sonnet", Object: "model", OwnedBy: "cursor", Type: "cursor", DisplayName: "Claude 4 Sonnet", ContextLength: 200000, MaxCompletionTokens: 64000, Thinking: &ThinkingSupport{Max: 50000, DynamicAllowed: true}},
		{ID: "claude-3.5-sonnet", Object: "model", OwnedBy: "cursor", Type: "cursor", DisplayName: "Claude 3.5 Sonnet", ContextLength: 200000, MaxCompletionTokens: 8192},
		{ID: "gpt-4o", Object: "model", OwnedBy: "cursor", Type: "cursor", DisplayName: "GPT-4o", ContextLength: 128000, MaxCompletionTokens: 16384},
		{ID: "cursor-small", Object: "model", OwnedBy: "cursor", Type: "cursor", DisplayName: "Cursor Small", ContextLength: 200000, MaxCompletionTokens: 64000},
		{ID: "gemini-2.5-pro", Object: "model", OwnedBy: "cursor", Type: "cursor", DisplayName: "Gemini 2.5 Pro", ContextLength: 1000000, MaxCompletionTokens: 65536, Thinking: &ThinkingSupport{Max: 50000, DynamicAllowed: true}},
	}
}

// LookupStaticModelInfo searches all static model definitions for a model by ID.
// Returns nil if no matching model is found.
func LookupStaticModelInfo(modelID string) *ModelInfo {
	if modelID == "" {
		return nil
	}

	data := getModels()
	if data == nil {
		return nil
	}
	allModels := [][]*ModelInfo{
		data.Claude,
		data.Gemini,
		data.Vertex,
		data.AIStudio,
		data.CodexPro,
		data.Kimi,
		GetAntigravityModels(),
		data.XAI,
		GetGitHubCopilotModels(),
		GetKiroModels(),
		GetKiloModels(),
		GetAmazonQModels(),
		GetCodeBuddyModels(),
		GetCursorModels(),
	}
	for _, models := range allModels {
		for _, m := range models {
			if m != nil && m.ID == modelID {
				return cloneModelInfo(m)
			}
		}
	}

	return nil
}

// defaultCopilotClaudeContextLength is the conservative prompt token limit for
// Claude models accessed via the GitHub Copilot API. Individual accounts are
// capped at 128K; business accounts at 168K. When the dynamic /models API fetch
// succeeds, the real per-account limit overrides this value. This constant is
// only used as a safe fallback.
const defaultCopilotClaudeContextLength = 128000

// GetGitHubCopilotModels returns the available models for GitHub Copilot.
// These models are available through the GitHub Copilot API at api.githubcopilot.com.
func GetGitHubCopilotModels() []*ModelInfo {
	now := int64(1732752000) // 2024-11-27
	copilotClaudeEndpoints := []string{"/chat/completions", "/messages"}
	gpt4oEntries := []struct {
		ID          string
		DisplayName string
		Description string
	}{
		{ID: "gpt-4o-2024-11-20", DisplayName: "GPT-4o (2024-11-20)", Description: "OpenAI GPT-4o 2024-11-20 via GitHub Copilot"},
		{ID: "gpt-4o-2024-08-06", DisplayName: "GPT-4o (2024-08-06)", Description: "OpenAI GPT-4o 2024-08-06 via GitHub Copilot"},
		{ID: "gpt-4o-2024-05-13", DisplayName: "GPT-4o (2024-05-13)", Description: "OpenAI GPT-4o 2024-05-13 via GitHub Copilot"},
		{ID: "gpt-4o", DisplayName: "GPT-4o", Description: "OpenAI GPT-4o via GitHub Copilot"},
		{ID: "gpt-4-o-preview", DisplayName: "GPT-4-o Preview", Description: "OpenAI GPT-4-o Preview via GitHub Copilot"},
	}

	models := []*ModelInfo{
		{
			ID:                  "gpt-4.1",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-4.1",
			Description:         "OpenAI GPT-4.1 via GitHub Copilot",
			ContextLength:       128000,
			MaxCompletionTokens: 16384,
			SupportedEndpoints:  []string{"/chat/completions", "/responses"},
		},
	}

	for _, entry := range gpt4oEntries {
		models = append(models, &ModelInfo{
			ID:                  entry.ID,
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         entry.DisplayName,
			Description:         entry.Description,
			ContextLength:       128000,
			MaxCompletionTokens: 16384,
			SupportedEndpoints:  []string{"/chat/completions", "/responses"},
		})
	}

	return append(models, []*ModelInfo{
		{
			ID:                  "gpt-5",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5",
			Description:         "OpenAI GPT-5 via GitHub Copilot",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions", "/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"low", "medium", "high"}},
		},
		{
			ID:                  "gpt-5-mini",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5 Mini",
			Description:         "OpenAI GPT-5 Mini via GitHub Copilot",
			ContextLength:       128000,
			MaxCompletionTokens: 16384,
			SupportedEndpoints:  []string{"/chat/completions", "/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"low", "medium", "high"}},
		},
		{
			ID:                  "gpt-5-codex",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5 Codex",
			Description:         "OpenAI GPT-5 Codex via GitHub Copilot",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"low", "medium", "high"}},
		},
		{
			ID:                  "gpt-5.1",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5.1",
			Description:         "OpenAI GPT-5.1 via GitHub Copilot",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions", "/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"none", "low", "medium", "high"}},
		},
		{
			ID:                  "gpt-5.1-codex",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5.1 Codex",
			Description:         "OpenAI GPT-5.1 Codex via GitHub Copilot",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"none", "low", "medium", "high"}},
		},
		{
			ID:                  "gpt-5.1-codex-mini",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5.1 Codex Mini",
			Description:         "OpenAI GPT-5.1 Codex Mini via GitHub Copilot",
			ContextLength:       128000,
			MaxCompletionTokens: 16384,
			SupportedEndpoints:  []string{"/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"none", "low", "medium", "high"}},
		},
		{
			ID:                  "gpt-5.1-codex-max",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5.1 Codex Max",
			Description:         "OpenAI GPT-5.1 Codex Max via GitHub Copilot",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"none", "low", "medium", "high", "xhigh"}},
		},
		{
			ID:                  "gpt-5.2",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5.2",
			Description:         "OpenAI GPT-5.2 via GitHub Copilot",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/chat/completions", "/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"none", "low", "medium", "high", "xhigh"}},
		},
		{
			ID:                  "gpt-5.2-codex",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5.2 Codex",
			Description:         "OpenAI GPT-5.2 Codex via GitHub Copilot",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"none", "low", "medium", "high", "xhigh"}},
		},
		{
			ID:                  "gpt-5.3-codex",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5.3 Codex",
			Description:         "OpenAI GPT-5.3 Codex via GitHub Copilot",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"none", "low", "medium", "high", "xhigh"}},
		},
		{
			ID:                  "gpt-5.4",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5.4",
			Description:         "OpenAI GPT-5.4 via GitHub Copilot",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"none", "low", "medium", "high", "xhigh"}},
		},
		{
			ID:                  "gpt-5.4-mini",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "GPT-5.4 mini",
			Description:         "OpenAI GPT-5.4 mini via GitHub Copilot",
			ContextLength:       200000,
			MaxCompletionTokens: 32768,
			SupportedEndpoints:  []string{"/responses"},
			Thinking:            &ThinkingSupport{Levels: []string{"none", "low", "medium", "high", "xhigh"}},
		},
		{
			ID:                  "claude-haiku-4.5",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Claude Haiku 4.5",
			Description:         "Anthropic Claude Haiku 4.5 via GitHub Copilot",
			ContextLength:       defaultCopilotClaudeContextLength,
			MaxCompletionTokens: 64000,
			SupportedEndpoints:  copilotClaudeEndpoints,
		},
		{
			ID:                  "claude-opus-4.1",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Claude Opus 4.1",
			Description:         "Anthropic Claude Opus 4.1 via GitHub Copilot",
			ContextLength:       defaultCopilotClaudeContextLength,
			MaxCompletionTokens: 32000,
			SupportedEndpoints:  copilotClaudeEndpoints,
		},
		{
			ID:                  "claude-opus-4.5",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Claude Opus 4.5",
			Description:         "Anthropic Claude Opus 4.5 via GitHub Copilot",
			ContextLength:       defaultCopilotClaudeContextLength,
			MaxCompletionTokens: 64000,
			SupportedEndpoints:  copilotClaudeEndpoints,
			Thinking:            &ThinkingSupport{Levels: []string{"low", "medium", "high"}},
		},
		{
			ID:                  "claude-opus-4.6",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Claude Opus 4.6",
			Description:         "Anthropic Claude Opus 4.6 via GitHub Copilot",
			ContextLength:       defaultCopilotClaudeContextLength,
			MaxCompletionTokens: 64000,
			SupportedEndpoints:  copilotClaudeEndpoints,
			Thinking:            &ThinkingSupport{Levels: []string{"low", "medium", "high"}},
		},
		{
			ID:                  "claude-sonnet-4",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Claude Sonnet 4",
			Description:         "Anthropic Claude Sonnet 4 via GitHub Copilot",
			ContextLength:       defaultCopilotClaudeContextLength,
			MaxCompletionTokens: 64000,
			SupportedEndpoints:  copilotClaudeEndpoints,
			Thinking:            &ThinkingSupport{Levels: []string{"low", "medium", "high"}},
		},
		{
			ID:                  "claude-sonnet-4.5",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Claude Sonnet 4.5",
			Description:         "Anthropic Claude Sonnet 4.5 via GitHub Copilot",
			ContextLength:       defaultCopilotClaudeContextLength,
			MaxCompletionTokens: 64000,
			SupportedEndpoints:  copilotClaudeEndpoints,
			Thinking:            &ThinkingSupport{Levels: []string{"low", "medium", "high"}},
		},
		{
			ID:                  "claude-sonnet-4.6",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Claude Sonnet 4.6",
			Description:         "Anthropic Claude Sonnet 4.6 via GitHub Copilot",
			ContextLength:       defaultCopilotClaudeContextLength,
			MaxCompletionTokens: 64000,
			SupportedEndpoints:  copilotClaudeEndpoints,
			Thinking:            &ThinkingSupport{Levels: []string{"low", "medium", "high"}},
		},
		{
			ID:                  "gemini-2.5-pro",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Gemini 2.5 Pro",
			Description:         "Google Gemini 2.5 Pro via GitHub Copilot",
			ContextLength:       1048576,
			MaxCompletionTokens: 65536,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "gemini-3-pro-preview",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Gemini 3 Pro (Preview)",
			Description:         "Google Gemini 3 Pro Preview via GitHub Copilot",
			ContextLength:       1048576,
			MaxCompletionTokens: 65536,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "gemini-3.1-pro-preview",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Gemini 3.1 Pro (Preview)",
			Description:         "Google Gemini 3.1 Pro Preview via GitHub Copilot",
			ContextLength:       173000,
			MaxCompletionTokens: 65536,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "gemini-3-flash-preview",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Gemini 3 Flash (Preview)",
			Description:         "Google Gemini 3 Flash Preview via GitHub Copilot",
			ContextLength:       173000,
			MaxCompletionTokens: 65536,
			SupportedEndpoints:  []string{"/chat/completions"},
		},
		{
			ID:                  "grok-code-fast-1",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Grok Code Fast 1",
			Description:         "xAI Grok Code Fast 1 via GitHub Copilot",
			ContextLength:       128000,
			MaxCompletionTokens: 16384,
		},
		{
			ID:                  "oswe-vscode-prime",
			Object:              "model",
			Created:             now,
			OwnedBy:             "github-copilot",
			Type:                "github-copilot",
			DisplayName:         "Raptor mini (Preview)",
			Description:         "Raptor mini via GitHub Copilot",
			ContextLength:       128000,
			MaxCompletionTokens: 16384,
			SupportedEndpoints:  []string{"/chat/completions", "/responses"},
		},
	}...)
}

// GetKiroModels returns the Kiro (AWS CodeWhisperer) model definitions
func GetKiroModels() []*ModelInfo {
	return []*ModelInfo{
		// --- Base Models ---
		{
			ID:                  "kiro-auto",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Auto",
			Description:         "Automatic model selection by Kiro",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-claude-opus-4-6",
			Object:              "model",
			Created:             1736899200, // 2025-01-15
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Opus 4.6",
			Description:         "Claude Opus 4.6 via Kiro (2.2x credit)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-claude-sonnet-4-6",
			Object:              "model",
			Created:             1739836800, // 2025-02-18
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Sonnet 4.6",
			Description:         "Claude Sonnet 4.6 via Kiro (1.3x credit)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-claude-opus-4-5",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Opus 4.5",
			Description:         "Claude Opus 4.5 via Kiro (2.2x credit)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-claude-sonnet-4-5",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Sonnet 4.5",
			Description:         "Claude Sonnet 4.5 via Kiro (1.3x credit)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-claude-sonnet-4",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Sonnet 4",
			Description:         "Claude Sonnet 4 via Kiro (1.3x credit)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-claude-haiku-4-5",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Haiku 4.5",
			Description:         "Claude Haiku 4.5 via Kiro (0.4x credit)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		// --- 第三方模型 (通过 Kiro 接入) ---
		{
			ID:                  "kiro-deepseek-3-2",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro DeepSeek 3.2",
			Description:         "DeepSeek 3.2 via Kiro",
			ContextLength:       128000,
			MaxCompletionTokens: 32768,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-minimax-m2-1",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro MiniMax M2.1",
			Description:         "MiniMax M2.1 via Kiro",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-qwen3-coder-next",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Qwen3 Coder Next",
			Description:         "Qwen3 Coder Next via Kiro",
			ContextLength:       128000,
			MaxCompletionTokens: 32768,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-gpt-4o",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro GPT-4o",
			Description:         "OpenAI GPT-4o via Kiro",
			ContextLength:       128000,
			MaxCompletionTokens: 16384,
		},
		{
			ID:                  "kiro-gpt-4",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro GPT-4",
			Description:         "OpenAI GPT-4 via Kiro",
			ContextLength:       128000,
			MaxCompletionTokens: 8192,
		},
		{
			ID:                  "kiro-gpt-4-turbo",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro GPT-4 Turbo",
			Description:         "OpenAI GPT-4 Turbo via Kiro",
			ContextLength:       128000,
			MaxCompletionTokens: 16384,
		},
		{
			ID:                  "kiro-gpt-3-5-turbo",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro GPT-3.5 Turbo",
			Description:         "OpenAI GPT-3.5 Turbo via Kiro",
			ContextLength:       16384,
			MaxCompletionTokens: 4096,
		},
		// --- Agentic Variants (Optimized for coding agents with chunked writes) ---
		{
			ID:                  "kiro-claude-opus-4-6-agentic",
			Object:              "model",
			Created:             1736899200, // 2025-01-15
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Opus 4.6 (Agentic)",
			Description:         "Claude Opus 4.6 optimized for coding agents (chunked writes)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-claude-sonnet-4-6-agentic",
			Object:              "model",
			Created:             1739836800, // 2025-02-18
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Sonnet 4.6 (Agentic)",
			Description:         "Claude Sonnet 4.6 optimized for coding agents (chunked writes)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-claude-opus-4-5-agentic",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Opus 4.5 (Agentic)",
			Description:         "Claude Opus 4.5 optimized for coding agents (chunked writes)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-claude-sonnet-4-5-agentic",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Sonnet 4.5 (Agentic)",
			Description:         "Claude Sonnet 4.5 optimized for coding agents (chunked writes)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-claude-sonnet-4-agentic",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Sonnet 4 (Agentic)",
			Description:         "Claude Sonnet 4 optimized for coding agents (chunked writes)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-claude-haiku-4-5-agentic",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Claude Haiku 4.5 (Agentic)",
			Description:         "Claude Haiku 4.5 optimized for coding agents (chunked writes)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-deepseek-3-2-agentic",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro DeepSeek 3.2 (Agentic)",
			Description:         "DeepSeek 3.2 optimized for coding agents (chunked writes)",
			ContextLength:       128000,
			MaxCompletionTokens: 32768,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-minimax-m2-1-agentic",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro MiniMax M2.1 (Agentic)",
			Description:         "MiniMax M2.1 optimized for coding agents (chunked writes)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:                  "kiro-qwen3-coder-next-agentic",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Kiro Qwen3 Coder Next (Agentic)",
			Description:         "Qwen3 Coder Next optimized for coding agents (chunked writes)",
			ContextLength:       128000,
			MaxCompletionTokens: 32768,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
	}
}

// GetAmazonQModels returns the Amazon Q (AWS CodeWhisperer) model definitions.
// These models use the same API as Kiro and share the same executor.
func GetAmazonQModels() []*ModelInfo {
	return []*ModelInfo{
		{
			ID:                  "amazonq-auto",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro", // Uses Kiro executor - same API
			DisplayName:         "Amazon Q Auto",
			Description:         "Automatic model selection by Amazon Q",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
		},
		{
			ID:                  "amazonq-claude-opus-4.5",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Amazon Q Claude Opus 4.5",
			Description:         "Claude Opus 4.5 via Amazon Q (2.2x credit)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
		},
		{
			ID:                  "amazonq-claude-sonnet-4.5",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Amazon Q Claude Sonnet 4.5",
			Description:         "Claude Sonnet 4.5 via Amazon Q (1.3x credit)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
		},
		{
			ID:                  "amazonq-claude-sonnet-4",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Amazon Q Claude Sonnet 4",
			Description:         "Claude Sonnet 4 via Amazon Q (1.3x credit)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
		},
		{
			ID:                  "amazonq-claude-haiku-4.5",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         "Amazon Q Claude Haiku 4.5",
			Description:         "Claude Haiku 4.5 via Amazon Q (0.4x credit)",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
		},
	}
}
