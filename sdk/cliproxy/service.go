// Package cliproxy provides the core service implementation for the CLI Proxy API.
// It includes service lifecycle management, authentication handling, file watching,
// and integration with various AI service providers through a unified interface.
package cliproxy

import (
	"context"
	"sync"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/api"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/home"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/homeplugins"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/pluginhost"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/watcher"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/wsrelay"
	sdkaccess "github.com/router-for-me/CLIProxyAPI/v7/sdk/access"
	sdkAuth "github.com/router-for-me/CLIProxyAPI/v7/sdk/auth"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executionregistry"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
	sdkpluginstore "github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginstore"
)

// Service wraps the proxy server lifecycle so external programs can embed the CLI proxy.
// It manages the complete lifecycle including authentication, file watching, HTTP server,
// and integration with various AI service providers.
type Service struct {
	// cfg holds the current application configuration.
	cfg *config.Config

	// cfgMu protects concurrent access to the configuration.
	cfgMu sync.RWMutex

	// configUpdateMu serializes config updates across watcher + home.
	configUpdateMu sync.Mutex

	// configRuntimeMu orders side-effecting runtime application after config commits.
	configRuntimeMu        sync.Mutex
	executorRegistrationMu sync.Mutex
	configSequence         uint64
	appliedRoutingState    *routingRuntimeState

	// configPath is the path to the configuration file.
	configPath string

	// tokenProvider handles loading token-based clients.
	tokenProvider TokenClientProvider

	// apiKeyProvider handles loading API key-based clients.
	apiKeyProvider APIKeyClientProvider

	// watcherFactory creates file watcher instances.
	watcherFactory WatcherFactory

	// hooks provides lifecycle callbacks.
	hooks Hooks

	// serverOptions contains additional server configuration options.
	serverOptions []api.ServerOption

	// server is the HTTP API server instance.
	server *api.Server

	// pprofServer manages the optional pprof HTTP debug server.
	pprofServer *pprofServer

	// serverErr channel for server startup/shutdown errors.
	serverErr chan error

	// watcher handles file system monitoring.
	watcher *WatcherWrapper

	// watcherCancel cancels the watcher context.
	watcherCancel context.CancelFunc

	// authUpdates channel for authentication updates.
	authUpdates chan watcher.AuthUpdate

	// authQueueStop cancels the auth update queue processing.
	authQueueStop context.CancelFunc

	// authManager handles legacy authentication operations.
	authManager *sdkAuth.Manager

	// accessManager handles request authentication providers.
	accessManager *sdkaccess.Manager

	// coreManager handles core authentication and execution.
	coreManager *coreauth.Manager

	// cooldownStateStore persists runtime cooldown state when enabled.
	cooldownStateStore coreauth.CooldownStateStore

	// pluginHost owns dynamic plugin lifecycle and runtime capability adapters.
	pluginHost *pluginhost.Host

	// shutdownOnce ensures shutdown is called only once.
	shutdownOnce sync.Once

	// wsGateway manages websocket Gemini providers.
	wsGateway *wsrelay.Manager

	homeLifecycleMu              sync.Mutex
	homeOwnershipMu              sync.Mutex
	homeConfigCommitMu           sync.Mutex
	homeConfigStageHook          func()
	homeConfigCommitHook         func()
	homeConfigRuntimeHook        func()
	applyPprofConfigContextFn    func(context.Context, *config.Config) bool
	updateServerClientsContextFn func(context.Context, *config.Config) bool
	homeSupervisor               *homeSubscriberSupervisor
	homeMu                       sync.Mutex
	homeGeneration               uint64
	homeClient                   *home.Client
	homeRegistry                 *executionregistry.Registry
	homeDispatchBundle           *coreauth.HomeDispatchBundle
	homeDrainBound               time.Duration
	homeCancel                   context.CancelFunc
	runCancel                    context.CancelFunc
	homeLogForwarder             homeLogForwarder
	homeLogForwarderClient       *home.Client
	homePluginSyncMu             sync.Mutex
	homePluginSyncKey            string
	homePluginSyncFetch          func(context.Context, sdkpluginstore.PluginSyncRequest) (sdkpluginstore.PluginSyncResponse, error)
	homePluginDeleteTask         func(context.Context, *config.Config, home.PluginTask) homeplugins.SyncReport
}

// fetchKiroModels attempts to dynamically fetch Kiro models from the API.
// If dynamic fetch fails, it falls back to static registry.GetKiroModels().
func (s *Service) fetchKiroModels(a *coreauth.Auth) []*ModelInfo {
	if a == nil {
		log.Debug("kiro: auth is nil, using static models")
		return registry.GetKiroModels()
	}

	// Extract token data from auth attributes
	tokenData := s.extractKiroTokenData(a)
	if tokenData == nil || tokenData.AccessToken == "" {
		log.Debug("kiro: no valid token data in auth, using static models")
		return registry.GetKiroModels()
	}

	// Create KiroAuth instance
	kAuth := kiroauth.NewKiroAuth(s.cfg)
	if kAuth == nil {
		log.Warn("kiro: failed to create KiroAuth instance, using static models")
		return registry.GetKiroModels()
	}

	// Use timeout context for API call
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Attempt to fetch dynamic models
	apiModels, err := kAuth.ListAvailableModels(ctx, tokenData)
	if err != nil {
		log.Warnf("kiro: failed to fetch dynamic models: %v, using static models", err)
		return registry.GetKiroModels()
	}

	if len(apiModels) == 0 {
		log.Debug("kiro: API returned no models, using static models")
		return registry.GetKiroModels()
	}

	// Convert API models to ModelInfo
	models := convertKiroAPIModels(apiModels)

	// Generate agentic variants
	models = generateKiroAgenticVariants(models)

	log.Infof("kiro: successfully fetched %d models from API (including agentic variants)", len(models))
	return models
}

// extractKiroTokenData extracts KiroTokenData from auth attributes and metadata.
// It supports both config-based tokens (stored in Attributes) and file-based tokens (stored in Metadata).
func (s *Service) extractKiroTokenData(a *coreauth.Auth) *kiroauth.KiroTokenData {
	if a == nil {
		return nil
	}

	var accessToken, profileArn, refreshToken string

	// Priority 1: Try to get from Attributes (config.yaml source)
	if a.Attributes != nil {
		accessToken = strings.TrimSpace(a.Attributes["access_token"])
		profileArn = strings.TrimSpace(a.Attributes["profile_arn"])
		refreshToken = strings.TrimSpace(a.Attributes["refresh_token"])
	}

	// Priority 2: If not found in Attributes, try Metadata (JSON file source)
	if accessToken == "" && a.Metadata != nil {
		if at, ok := a.Metadata["access_token"].(string); ok {
			accessToken = strings.TrimSpace(at)
		}
		if pa, ok := a.Metadata["profile_arn"].(string); ok {
			profileArn = strings.TrimSpace(pa)
		}
		if rt, ok := a.Metadata["refresh_token"].(string); ok {
			refreshToken = strings.TrimSpace(rt)
		}
	}

	// access_token is required
	if accessToken == "" {
		return nil
	}

	return &kiroauth.KiroTokenData{
		AccessToken:  accessToken,
		ProfileArn:   profileArn,
		RefreshToken: refreshToken,
	}
}

// convertKiroAPIModels converts Kiro API models to ModelInfo slice.
func convertKiroAPIModels(apiModels []*kiroauth.KiroModel) []*ModelInfo {
	if len(apiModels) == 0 {
		return nil
	}

	now := time.Now().Unix()
	models := make([]*ModelInfo, 0, len(apiModels))

	for _, m := range apiModels {
		if m == nil || m.ModelID == "" {
			continue
		}

		// Create model ID with kiro- prefix
		modelID := "kiro-" + normalizeKiroModelID(m.ModelID)

		info := &ModelInfo{
			ID:                  modelID,
			Object:              "model",
			Created:             now,
			OwnedBy:             "aws",
			Type:                "kiro",
			DisplayName:         formatKiroDisplayName(m.ModelName, m.RateMultiplier),
			Description:         m.Description,
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &registry.ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		}

		if m.MaxInputTokens > 0 {
			info.ContextLength = m.MaxInputTokens
		}

		models = append(models, info)
	}

	return models
}

// normalizeKiroModelID normalizes a Kiro model ID by converting dots to dashes
// and removing common prefixes.
func normalizeKiroModelID(modelID string) string {
	// Remove common prefixes
	modelID = strings.TrimPrefix(modelID, "anthropic.")
	modelID = strings.TrimPrefix(modelID, "amazon.")

	// Replace dots with dashes for consistency
	modelID = strings.ReplaceAll(modelID, ".", "-")

	// Replace underscores with dashes
	modelID = strings.ReplaceAll(modelID, "_", "-")

	return strings.ToLower(modelID)
}

// formatKiroDisplayName formats the display name with rate multiplier info.
func formatKiroDisplayName(modelName string, rateMultiplier float64) string {
	if modelName == "" {
		return ""
	}

	displayName := "Kiro " + modelName
	if rateMultiplier > 0 && rateMultiplier != 1.0 {
		displayName += fmt.Sprintf(" (%.1fx credit)", rateMultiplier)
	}

	return displayName
}

// generateKiroAgenticVariants generates agentic variants for Kiro models.
// Agentic variants have optimized system prompts for coding agents.
func generateKiroAgenticVariants(models []*ModelInfo) []*ModelInfo {
	if len(models) == 0 {
		return models
	}

	result := make([]*ModelInfo, 0, len(models)*2)
	result = append(result, models...)

	for _, m := range models {
		if m == nil {
			continue
		}

		// Skip if already an agentic variant
		if strings.HasSuffix(m.ID, "-agentic") {
			continue
		}

		// Skip auto models from agentic variant generation
		if strings.Contains(m.ID, "-auto") {
			continue
		}

		// Create agentic variant
		agentic := &ModelInfo{
			ID:                  m.ID + "-agentic",
			Object:              m.Object,
			Created:             m.Created,
			OwnedBy:             m.OwnedBy,
			Type:                m.Type,
			DisplayName:         m.DisplayName + " (Agentic)",
			Description:         m.Description + " - Optimized for coding agents (chunked writes)",
			ContextLength:       m.ContextLength,
			MaxCompletionTokens: m.MaxCompletionTokens,
		}

		// Copy thinking support if present
		if m.Thinking != nil {
			agentic.Thinking = &registry.ThinkingSupport{
				Min:            m.Thinking.Min,
				Max:            m.Thinking.Max,
				ZeroAllowed:    m.Thinking.ZeroAllowed,
				DynamicAllowed: m.Thinking.DynamicAllowed,
			}
		}

		result = append(result, agentic)
	}

	return result
}
