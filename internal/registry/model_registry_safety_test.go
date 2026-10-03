package registry

import (
	"strings"
	"testing"
	"time"
)

func TestGetModelInfoReturnsClone(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "gemini", []*ModelInfo{{
		ID:          "m1",
		DisplayName: "Model One",
		Thinking:    &ThinkingSupport{Min: 1, Max: 2, Levels: []string{"low", "high"}},
	}})

	first := r.GetModelInfo("m1", "gemini")
	if first == nil {
		t.Fatal("expected model info")
	}
	first.DisplayName = "mutated"
	first.Thinking.Levels[0] = "mutated"

	second := r.GetModelInfo("m1", "gemini")
	if second.DisplayName != "Model One" {
		t.Fatalf("expected cloned display name, got %q", second.DisplayName)
	}
	if second.Thinking == nil || len(second.Thinking.Levels) == 0 || second.Thinking.Levels[0] != "low" {
		t.Fatalf("expected cloned thinking levels, got %+v", second.Thinking)
	}
}

func TestGetModelsForClientReturnsClones(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "gemini", []*ModelInfo{{
		ID:          "m1",
		DisplayName: "Model One",
		Thinking:    &ThinkingSupport{Levels: []string{"low", "high"}},
	}})

	first := r.GetModelsForClient("client-1")
	if len(first) != 1 || first[0] == nil {
		t.Fatalf("expected one model, got %+v", first)
	}
	first[0].DisplayName = "mutated"
	first[0].Thinking.Levels[0] = "mutated"

	second := r.GetModelsForClient("client-1")
	if len(second) != 1 || second[0] == nil {
		t.Fatalf("expected one model on second fetch, got %+v", second)
	}
	if second[0].DisplayName != "Model One" {
		t.Fatalf("expected cloned display name, got %q", second[0].DisplayName)
	}
	if second[0].Thinking == nil || len(second[0].Thinking.Levels) == 0 || second[0].Thinking.Levels[0] != "low" {
		t.Fatalf("expected cloned thinking levels, got %+v", second[0].Thinking)
	}
}

func TestGetAvailableModelsByProviderReturnsClones(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "gemini", []*ModelInfo{{
		ID:          "m1",
		DisplayName: "Model One",
		Thinking:    &ThinkingSupport{Levels: []string{"low", "high"}},
	}})

	first := r.GetAvailableModelsByProvider("gemini")
	if len(first) != 1 || first[0] == nil {
		t.Fatalf("expected one model, got %+v", first)
	}
	first[0].DisplayName = "mutated"
	first[0].Thinking.Levels[0] = "mutated"

	second := r.GetAvailableModelsByProvider("gemini")
	if len(second) != 1 || second[0] == nil {
		t.Fatalf("expected one model on second fetch, got %+v", second)
	}
	if second[0].DisplayName != "Model One" {
		t.Fatalf("expected cloned display name, got %q", second[0].DisplayName)
	}
	if second[0].Thinking == nil || len(second[0].Thinking.Levels) == 0 || second[0].Thinking.Levels[0] != "low" {
		t.Fatalf("expected cloned thinking levels, got %+v", second[0].Thinking)
	}
}

func TestCleanupExpiredQuotasInvalidatesAvailableModelsCache(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "openai", []*ModelInfo{{ID: "m1", Created: 1}})
	r.SetModelQuotaExceeded("client-1", "m1")
	if models := r.GetAvailableModels("openai"); len(models) != 1 {
		t.Fatalf("expected cooldown model to remain listed before cleanup, got %d", len(models))
	}

	r.mutex.Lock()
	quotaTime := time.Now().Add(-6 * time.Minute)
	r.models["m1"].QuotaExceededClients["client-1"] = &quotaTime
	r.mutex.Unlock()

	r.CleanupExpiredQuotas()

	if count := r.GetModelCount("m1"); count != 1 {
		t.Fatalf("expected model count 1 after cleanup, got %d", count)
	}
	models := r.GetAvailableModels("openai")
	if len(models) != 1 {
		t.Fatalf("expected model to stay available after cleanup, got %d", len(models))
	}
	if got := models[0]["id"]; got != "m1" {
		t.Fatalf("expected model id m1, got %v", got)
	}
}

func TestGetAvailableModelsReturnsClonedSupportedParameters(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "openai", []*ModelInfo{{
		ID:                  "m1",
		DisplayName:         "Model One",
		SupportedParameters: []string{"temperature", "top_p"},
	}})

	first := r.GetAvailableModels("openai")
	if len(first) != 1 {
		t.Fatalf("expected one model, got %d", len(first))
	}
	params, ok := first[0]["supported_parameters"].([]string)
	if !ok || len(params) != 2 {
		t.Fatalf("expected supported_parameters slice, got %#v", first[0]["supported_parameters"])
	}
	params[0] = "mutated"

	second := r.GetAvailableModels("openai")
	params, ok = second[0]["supported_parameters"].([]string)
	if !ok || len(params) != 2 || params[0] != "temperature" {
		t.Fatalf("expected cloned supported_parameters, got %#v", second[0]["supported_parameters"])
	}
}

func TestLookupModelInfoReturnsCloneForStaticDefinitions(t *testing.T) {
	first := LookupModelInfo("claude-sonnet-4-6")
	if first == nil || first.Thinking == nil || len(first.Thinking.Levels) == 0 {
		t.Fatalf("expected static model with thinking levels, got %+v", first)
	}
	first.Thinking.Levels[0] = "mutated"

	second := LookupModelInfo("claude-sonnet-4-6")
	if second == nil || second.Thinking == nil || len(second.Thinking.Levels) == 0 || second.Thinking.Levels[0] == "mutated" {
		t.Fatalf("expected static lookup clone, got %+v", second)
	}
}

func TestLookupModelInfoIncludesClaudeSonnet5(t *testing.T) {
	model := LookupModelInfo("claude-sonnet-5")
	if model == nil {
		t.Fatal("expected Claude Sonnet 5 static model")
	}
	if model.Type != "claude" {
		t.Fatalf("Claude Sonnet 5 type = %q, want claude", model.Type)
	}
	if model.ContextLength != 1000000 {
		t.Fatalf("Claude Sonnet 5 context length = %d, want 1000000", model.ContextLength)
	}
	if model.MaxCompletionTokens != 128000 {
		t.Fatalf("Claude Sonnet 5 max completion tokens = %d, want 128000", model.MaxCompletionTokens)
	}
	if model.Thinking == nil || !model.Thinking.ZeroAllowed || !model.Thinking.DynamicAllowed || model.Thinking.Min != 0 || model.Thinking.Max != 0 {
		t.Fatalf("expected Claude Sonnet 5 dynamic level-only thinking with zero allowed, got %+v", model.Thinking)
	}
	expectedLevels := []string{"low", "medium", "high", "xhigh", "max"}
	if len(model.Thinking.Levels) != len(expectedLevels) {
		t.Fatalf("Claude Sonnet 5 thinking levels = %+v, want %+v", model.Thinking.Levels, expectedLevels)
	}
	for i, level := range expectedLevels {
		if model.Thinking.Levels[i] != level {
			t.Fatalf("Claude Sonnet 5 thinking levels = %+v, want %+v", model.Thinking.Levels, expectedLevels)
		}
	}
}

// The shared registry (router-for-me/models) omits models Antigravity does serve
// and advertises claude-opus-5-5-high / claude-sonnet-5-5-high, which no
// Antigravity account can execute. A periodic refresh used to therefore strand
// requests for those models with "unknown provider for model". These tests pin
// the overrides that keep the catalog honest regardless of what is fetched.
func TestAntigravityCatalogSurvivesRegistryRefresh(t *testing.T) {
	// What the shared registry actually publishes today.
	registry := []*ModelInfo{
		{ID: "claude-opus-5-5-high", OwnedBy: "antigravity", Type: "antigravity"},
		{ID: "claude-sonnet-5-5-high", OwnedBy: "antigravity", Type: "antigravity"},
		{ID: "gemini-3.8-flash-high", OwnedBy: "antigravity", Type: "antigravity"},
		{ID: "gemini-pro-agent", OwnedBy: "antigravity", Type: "antigravity"},
	}

	got := make(map[string]bool)
	rebuilt := upsertModelInfos(dropModelInfos(registry, antigravityUnsupportedModelIDs), antigravityLocalModels...)
	for _, model := range upsertModelInfos(rebuilt, antigravityFlashTierModels()...) {
		got[model.ID] = true
	}

	// Served upstream, dropped by the registry: must be reinstated.
	for _, model := range append(append([]*ModelInfo{}, antigravityLocalModels...), antigravityFlashTierModels()...) {
		if !got[model.ID] {
			t.Errorf("Antigravity model %q is served upstream and must survive a registry refresh", model.ID)
		}
	}
	for _, id := range []string{"claude-opus-5-5-high", "claude-sonnet-5-5-high"} {
		if got[id] {
			t.Errorf("Antigravity model %q is not served upstream and must not be advertised", id)
		}
	}
	for _, id := range []string{"gemini-3.8-flash-high", "gemini-pro-agent"} {
		if !got[id] {
			t.Errorf("registry model %q must be left alone", id)
		}
	}
}

func TestAntigravityFlashTierModelsAreWellFormed(t *testing.T) {
	tiers := antigravityFlashTierModels()
	if len(tiers) != 9 {
		t.Fatalf("expected 3 versions x 3 tiers, got %d", len(tiers))
	}

	seen := make(map[string]bool, len(tiers))
	for _, model := range tiers {
		if seen[model.ID] {
			t.Errorf("duplicate generated model id %q", model.ID)
		}
		seen[model.ID] = true

		if !strings.HasPrefix(model.ID, "gemini-3.") || !strings.HasSuffix(model.ID, "-flash-tiered") &&
			!strings.HasSuffix(model.ID, "-flash-low") && !strings.HasSuffix(model.ID, "-flash-medium") {
			t.Errorf("unexpected generated id %q", model.ID)
		}
		if model.OwnedBy != "antigravity" || model.Type != "antigravity" {
			t.Errorf("%q generated as %s/%s, want antigravity/antigravity", model.ID, model.OwnedBy, model.Type)
		}
		if !strings.Contains(model.DisplayName, "(") {
			t.Errorf("%q needs a tier suffix in its display name, got %q", model.ID, model.DisplayName)
		}
		if model.Thinking == nil || len(model.Thinking.Levels) == 0 {
			t.Errorf("%q must advertise thinking levels", model.ID)
		}
		if len(model.SupportedInputModalities) == 0 || len(model.SupportedOutputModalities) == 0 {
			t.Errorf("%q must advertise modalities", model.ID)
		}
		if model.ContextLength != 1048576 || model.MaxCompletionTokens != 65536 {
			t.Errorf("%q has unexpected limits %d/%d", model.ID, model.ContextLength, model.MaxCompletionTokens)
		}
	}
}

// The embedded catalog must already carry every model the overrides reinstate,
// so a build that never reaches the network is not missing them.
func TestAntigravityLocalModelsAreEmbedded(t *testing.T) {
	embedded := make(map[string]bool)
	for _, model := range getModels().Antigravity {
		embedded[model.ID] = true
	}

	for _, model := range append(append([]*ModelInfo{}, antigravityLocalModels...), antigravityFlashTierModels()...) {
		if !embedded[model.ID] {
			t.Errorf("%q is missing from the embedded models.json catalog", model.ID)
		}
	}
}

func TestAntigravityLocalModelsAreComplete(t *testing.T) {
	published := make(map[string]*ModelInfo)
	for _, model := range GetAntigravityModels() {
		published[model.ID] = model
	}

	all := append(append([]*ModelInfo{}, antigravityLocalModels...), antigravityFlashTierModels()...)
	for _, model := range all {
		info, ok := published[model.ID]
		if !ok {
			t.Errorf("%q missing from the Antigravity catalog", model.ID)
			continue
		}
		if info.OwnedBy != "antigravity" || info.Type != "antigravity" {
			t.Errorf("%q published as %s/%s, want antigravity/antigravity", model.ID, info.OwnedBy, info.Type)
		}
		if info.DisplayName == "" {
			t.Errorf("%q has no display name", model.ID)
		}
		if info.ContextLength <= 0 || info.MaxCompletionTokens <= 0 {
			t.Errorf("%q has no usable token limits: %+v", model.ID, info)
		}
		if info.Thinking == nil {
			t.Errorf("%q must advertise thinking support", model.ID)
		}
	}

	for id := range antigravityUnsupportedModelIDs {
		if _, ok := published[id]; ok {
			t.Errorf("%q is not served upstream and must not be published", id)
		}
	}
	for id := range antigravityExcludedModelIDs {
		if _, ok := published[id]; ok {
			t.Errorf("%q is excluded by operator preference and must not be published", id)
		}
	}
}

// The Gemini 2.x Antigravity models are intentionally unpublished. Guard against
// a registry refresh quietly reintroducing them.
func TestAntigravityExcludedModelsStayExcluded(t *testing.T) {
	registry := []*ModelInfo{
		{ID: "gemini-2.5-flash", OwnedBy: "antigravity", Type: "antigravity"},
		{ID: "gemini-2.5-pro", OwnedBy: "antigravity", Type: "antigravity"},
		{ID: "gemini-3.8-flash-high", OwnedBy: "antigravity", Type: "antigravity"},
	}

	got := make(map[string]bool)
	for _, model := range upsertModelInfos(dropModelInfos(dropModelInfos(registry, antigravityUnsupportedModelIDs), antigravityExcludedModelIDs), antigravityLocalModels...) {
		got[model.ID] = true
	}

	for id := range antigravityExcludedModelIDs {
		if got[id] {
			t.Errorf("%q must stay excluded after a registry refresh", id)
		}
	}
	if !got["gemini-3.8-flash-high"] {
		t.Error("filtering must not touch models outside the exclusion lists")
	}
}
