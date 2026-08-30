package registry

import (
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/constant"
)

// openCodeCuratedWindows is the curated per-model context window table for the
// opencode / opencode-go endpoints. Those endpoints return model ids only (no
// context_length), so windows are sourced from this table keyed by model id
// (FR-008). Values reflect the underlying model family's known context window and
// are best-effort; extend as the provider's catalog changes. Models absent here
// intentionally receive no window (FR-006) rather than a fabricated value.
var openCodeCuratedWindows = map[string]ProviderModelLimit{
	// Claude family
	"claude-fable-5":    {ContextLength: 200000, MaxCompletionTokens: 64000},
	"claude-opus-5":     {ContextLength: 200000, MaxCompletionTokens: 64000},
	"claude-opus-4-8":   {ContextLength: 200000, MaxCompletionTokens: 32000},
	"claude-opus-4-7":   {ContextLength: 200000, MaxCompletionTokens: 32000},
	"claude-opus-4-6":   {ContextLength: 200000, MaxCompletionTokens: 32000},
	"claude-opus-4-5":   {ContextLength: 200000, MaxCompletionTokens: 32000},
	"claude-sonnet-5":   {ContextLength: 200000, MaxCompletionTokens: 64000},
	"claude-sonnet-4-6": {ContextLength: 200000, MaxCompletionTokens: 64000},
	"claude-sonnet-4-5": {ContextLength: 200000, MaxCompletionTokens: 64000},
	"claude-sonnet-4":   {ContextLength: 200000, MaxCompletionTokens: 64000},
	"claude-haiku-4-5":  {ContextLength: 200000, MaxCompletionTokens: 8000},
	// Gemini family (1M context)
	"gemini-3.6-flash":      {ContextLength: 1048576, MaxCompletionTokens: 65536},
	"gemini-3.7-flash":      {ContextLength: 1048576, MaxCompletionTokens: 65536},
	"gemini-3.5-flash":      {ContextLength: 1048576, MaxCompletionTokens: 65536},
	"gemini-3.5-flash-lite": {ContextLength: 1048576, MaxCompletionTokens: 65536},
	"gemini-3.1-pro":        {ContextLength: 1048576, MaxCompletionTokens: 65536},
	"gemini-3-flash":        {ContextLength: 1048576, MaxCompletionTokens: 65536},
	// GPT-5 family
	"gpt-5.6-sol":         {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.6-terra":       {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.6-luna":        {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.5":             {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.5-pro":         {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.4":             {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.4-pro":         {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.4-mini":        {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.4-nano":        {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.3-codex-spark": {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.3-codex":       {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.2":             {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.2-codex":       {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.1":             {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.1-codex-max":   {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.1-codex":       {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5.1-codex-mini":  {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5":               {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5-codex":         {ContextLength: 400000, MaxCompletionTokens: 100000},
	"gpt-5-nano":          {ContextLength: 400000, MaxCompletionTokens: 100000},
	// Grok family
	"grok-build-0.1": {ContextLength: 256000, MaxCompletionTokens: 100000},
	"grok-4.6":       {ContextLength: 256000, MaxCompletionTokens: 100000},
	"grok-4.5":       {ContextLength: 256000, MaxCompletionTokens: 100000},
	// DeepSeek family
	"deepseek-v4-pro":   {ContextLength: 128000, MaxCompletionTokens: 8000},
	"deepseek-v4-flash": {ContextLength: 128000, MaxCompletionTokens: 8000},
	// GLM family
	"glm-5.2": {ContextLength: 128000, MaxCompletionTokens: 8000},
	"glm-5.1": {ContextLength: 128000, MaxCompletionTokens: 8000},
	"glm-5":   {ContextLength: 128000, MaxCompletionTokens: 8000},
	// MiniMax family
	"minimax-m3":   {ContextLength: 256000, MaxCompletionTokens: 8000},
	"minimax-m2.7": {ContextLength: 256000, MaxCompletionTokens: 8000},
	"minimax-m2.5": {ContextLength: 256000, MaxCompletionTokens: 8000},
	// Kimi family
	"kimi-k3":        {ContextLength: 200000, MaxCompletionTokens: 32000},
	"kimi-k2.7-code": {ContextLength: 200000, MaxCompletionTokens: 32000},
	"kimi-k2.6":      {ContextLength: 200000, MaxCompletionTokens: 32000},
	"kimi-k2.5":      {ContextLength: 200000, MaxCompletionTokens: 32000},
	// Qwen family
	"qwen3.6-plus": {ContextLength: 256000, MaxCompletionTokens: 8000},
	"qwen3.5-plus": {ContextLength: 256000, MaxCompletionTokens: 8000},
	// Muse
	"muse-spark-1.2": {ContextLength: 200000, MaxCompletionTokens: 8000},
}

// curatedProviderWindows maps a dedicated provider name to its curated per-model
// context window table (FR-008). Used when the live endpoint omits context_length.
var curatedProviderWindows = map[string]map[string]ProviderModelLimit{
	constant.OpenCode:   openCodeCuratedWindows,
	constant.OpenCodeGo: openCodeCuratedWindows,
}

// curatedWindowsFor returns the curated window table for a provider, or nil when
// none is configured (e.g. openrouter / z.ai, whose endpoints carry live windows).
func curatedWindowsFor(provider string) map[string]ProviderModelLimit {
	return curatedProviderWindows[strings.ToLower(provider)]
}
