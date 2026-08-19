CLIProxyAPIPlus is configured through a single YAML file (typically `config.yaml`) that defines server behavior, authentication credentials, provider integrations, and advanced routing rules. This reference covers every configuration section with practical examples and default values to help you get started quickly. The full annotated example lives in [config.example.yaml](config.example.yaml).

## Configuration File Location

The server searches for configuration in the following priority order:

1. **`--config` flag**: Explicit path passed via the command-line flag `--config /path/to/config.yaml`
2. **Working directory**: Falls back to `./config.yaml` in the current working directory
3. **Home control plane**: When `--home host:port` is passed, configuration is fetched from a remote Redis-compatible Home server and local config files are ignored entirely
4. **Cloud deploy mode**: When the `DEPLOY=cloud` environment variable is set and no config file exists, the server starts in standby mode without an API server

You can copy the example configuration as a starting point:

```bash
cp config.example.yaml config.yaml
```

The configuration file supports YAML comments, and the server preserves your comments when updating values (like hashed management keys) at runtime.

Sources: [cmd/server/main.go#L419-L468](cmd/server/main.go#L419-L468), [internal/config/config.go#L779-L818](internal/config/config.go#L779-L818)

## Server Basics

These fundamental settings control where and how the server listens for incoming requests.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `host` | string | `""` (all interfaces) | Network interface to bind. Use `"127.0.0.1"` for local-only access |
| `port` | integer | `8317` | TCP port the API server listens on |
| `debug` | boolean | `false` | Enables debug-level logging and verbose diagnostic output |

```yaml
host: ''        # Bind all interfaces (IPv4 + IPv6)
port: 8317      # Default API port
debug: false    # Set true for verbose logging
```

When `host` is empty, the server binds to `0.0.0.0` and `[::]`, accepting connections from any network interface. For development or security-sensitive deployments, set `host: "127.0.0.1"` to restrict access to the local machine only.

Sources: [internal/config/config.go#L33-L40](internal/config/config.go#L33-L40), [config.example.yaml#L1-L10](config.example.yaml#L1-L10)

## TLS (HTTPS)

The server supports native TLS termination without requiring a reverse proxy.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `tls.enable` | boolean | `false` | Enable HTTPS mode |
| `tls.cert` | string | `""` | Path to the TLS certificate file |
| `tls.key` | string | `""` | Path to the TLS private key file |

```yaml
tls:
  enable: true
  cert: '/etc/ssl/certs/server.crt'
  key: '/etc/ssl/private/server.key'
```

When TLS is disabled (the default), the server accepts plain HTTP connections. For production deployments behind a load balancer, you typically leave TLS disabled and let the load balancer handle termination.

Sources: [internal/config/config.go#L310-L319](internal/config/config.go#L310-L319), [config.example.yaml#L12-L17](config.example.yaml#L12-L17)

## Client Authentication

The proxy requires incoming clients to authenticate with API keys before accessing any provider endpoints.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `api-keys` | list of strings | `["your-api-key-1", ...]` | List of valid API keys for client authentication |

```yaml
api-keys:
  - 'sk-proxy-prod-abc123'
  - 'sk-proxy-dev-xyz789'
```

**Important security note:** The example configuration ships with template values (`your-api-key-1`, etc.). If these placeholder keys are detected at startup, the server displays a setup warning page instead of proxying requests. Always replace them with strong, random keys before use.

Clients pass their API key in the `Authorization: Bearer <key>` header on every request to the proxy.

Sources: [internal/safemode/example_api_keys.go#L7-L11](internal/safemode/example_api_keys.go#L7-L11), [internal/config/sdk_config.go#L40-L41](internal/config/sdk_config.go#L40-L41)

## Authentication Directory

The authentication directory stores OAuth tokens, API key files, and other credential material for all configured providers.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `auth-dir` | string | `~/.cli-proxy-api` | Directory for storing auth tokens and credential files |

```yaml
auth-dir: '~/.cli-proxy-api'
```

The `~` prefix expands to the current user's home directory. Each provider (Codex, Claude, Gemini, Antigravity, etc.) stores its OAuth tokens as JSON files within subdirectories of this location. When using the `--login`, `--codex-login`, or other login flags, the server writes credentials into this directory.

For Docker deployments, this directory is typically mounted as a volume to persist credentials across container restarts.

Sources: [internal/config/config.go#L58-L59](internal/config/config.go#L58-L59), [config.example.yaml#L45-L46](config.example.yaml#L45-L46)

## Provider API Keys

CLIProxyAPIPlus supports multiple AI provider backends simultaneously. Each provider section defines credentials, model mappings, and per-key routing overrides.

### Gemini API Keys

Google Gemini API keys for accessing Gemini models through the standard REST API.

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `gemini-api-key[].api-key` | string | Yes | The Gemini API key |
| `gemini-api-key[].prefix` | string | No | Model namespace prefix (e.g., `"test"` → `test/gemini-3-pro`) |
| `gemini-api-key[].base-url` | string | No | Override the default Gemini API endpoint |
| `gemini-api-key[].proxy-url` | string | No | Per-key proxy override (`"direct"` for no proxy) |
| `gemini-api-key[].headers` | map | No | Extra HTTP headers for requests |
| `gemini-api-key[].models` | list | No | Model name/alias mappings |
| `gemini-api-key[].excluded-models` | list | No | Models to exclude (supports wildcards) |
| `gemini-api-key[].disable-cooling` | boolean | No | Disable cooldown scheduling for this key |

```yaml
gemini-api-key:
  - api-key: "AIzaSy...01"
    base-url: "https://generativelanguage.googleapis.com"
    proxy-url: "socks5://proxy.example.com:1080"
    models:
      - name: "gemini-2.5-flash"
        alias: "gemini-flash"
        display-name: "Gemini Flash"
    excluded-models:
      - "gemini-2.5-*"
```

### Claude API Keys

Anthropic Claude API keys for direct Claude model access.

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `claude-api-key[].api-key` | string | Yes | The Claude API key |
| `claude-api-key[].prefix` | string | No | Model namespace prefix |
| `claude-api-key[].base-url` | string | No | Custom Claude API endpoint |
| `claude-api-key[].proxy-url` | string | No | Per-key proxy override |
| `claude-api-key[].headers` | map | No | Extra HTTP headers |
| `claude-api-key[].models` | list | No | Model name/alias mappings |
| `claude-api-key[].excluded-models` | list | No | Models to exclude |
| `claude-api-key[].rebuild-mid-system-message` | boolean | No | Move system-role messages to top-level system field |
| `claude-api-key[].cloak` | object | No | Request cloaking configuration |
| `claude-api-key[].experimental-cch-signing` | boolean | No | Enable experimental cch body signing |

```yaml
claude-api-key:
  - api-key: "sk-ant-..."
    prefix: "teamA"
    models:
      - name: "claude-sonnet-4-5-20250929"
        alias: "claude-sonnet-latest"
        display-name: "Claude Sonnet"
        force-mapping: true
    cloak:
      mode: "auto"         # "auto" | "always" | "never"
      strict-mode: false
      sensitive-words:
        - "API"
        - "proxy"
      cache-user-id: true
```

**Claude Cloaking** disguises API requests to appear as originating from the official Claude Code CLI. This is useful for avoiding detection by upstream providers. The `mode` field controls when cloaking is applied: `"auto"` (default) only cloaks non-Claude-Code clients, `"always"` applies cloaking universally, and `"never"` disables it.

Sources: [internal/config/config.go#L467-L528](internal/config/config.go#L467-L528), [config.example.yaml#L271-L322](config.example.yaml#L271-L322)

### Codex API Keys

OpenAI Codex API keys for GPT model access through the Codex protocol.

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `codex-api-key[].api-key` | string | Yes | The Codex API key |
| `codex-api-key[].prefix` | string | No | Model namespace prefix |
| `codex-api-key[].base-url` | string | No | Custom Codex API endpoint |
| `codex-api-key[].websockets` | boolean | No | Use WebSocket transport for this credential |
| `codex-api-key[].proxy-url` | string | No | Per-key proxy override |
| `codex-api-key[].models` | list | No | Model name/alias mappings |
| `codex-api-key[].excluded-models` | list | No | Models to exclude |

```yaml
codex-api-key:
  - api-key: "sk-atSM..."
    base-url: "https://www.example.com"
    websockets: true
    models:
      - name: "gpt-5-codex"
        alias: "codex-latest"
        display-name: "Codex Latest"
        force-mapping: true
```

### xAI API Keys

xAI (Grok) API keys using the native xAI executor with Responses namespace handling.

The structure mirrors `codex-api-key` exactly (both use `CodexKey` internally). Set `websockets: true` to enable upstream WebSocket transport.

```yaml
xai-api-key:
  - api-key: "xai-..."
    base-url: "https://api.x.ai/v1"
    websockets: true
    models:
      - name: "grok-4.5"
        alias: "grok-latest"
```

Sources: [internal/config/config.go#L592-L598](internal/config/config.go#L592-L598), [config.example.yaml#L236-L269](config.example.yaml#L236-L269)

### Vertex-Compatible API Keys

For third-party services that use Vertex AI-style endpoint paths with simple API key authentication.

```yaml
vertex-api-key:
  - api-key: "vk-123..."
    base-url: "https://example.com/api"
    models:
      - name: "gemini-2.5-flash"
        alias: "vertex-flash"
```

Sources: [internal/config/vertex_compat.go#L11-L46](internal/config/vertex_compat.go#L11-L46)

### OpenAI Compatibility Providers

For external providers that implement the OpenAI API format (e.g., OpenRouter, custom providers).

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `openai-compatibility[].name` | string | Yes | Provider identifier |
| `openai-compatibility[].base-url` | string | Yes | Base URL for the provider's API |
| `openai-compatibility[].disabled` | boolean | No | Disable this provider |
| `openai-compatibility[].prefix` | string | No | Model namespace prefix |
| `openai-compatibility[].api-key-entries` | list | No | API keys with optional per-key proxy |
| `openai-compatibility[].models` | list | Yes | Model definitions including aliases |
| `openai-compatibility[].headers` | map | No | Extra HTTP headers |
| `openai-compatibility[].disable-cooling` | boolean | No | Disable cooldown scheduling |

```yaml
openai-compatibility:
  - name: "openrouter"
    base-url: "https://openrouter.ai/api/v1"
    api-key-entries:
      - api-key: "sk-or-v1-..."
        proxy-url: "socks5://proxy.example.com:1080"
    models:
      - name: "moonshotai/kimi-k2:free"
        alias: "kimi-k2"
        image: false
        input-modalities: [text, image]
        thinking:
          levels: ["low", "medium", "high"]
```

OpenAI compatibility models support advanced features like model pooling (repeating the same alias with different upstream names for load balancing), thinking level configuration, and multimodal capability declarations.

Sources: [internal/config/config.go#L731-L785](internal/config/config.go#L731-L785), [config.example.yaml#L400-L450](config.example.yaml#L400-L450)

### Kiro (AWS CodeWhisperer) Configuration

Kiro uses AWS SSO tokens for authentication.

```yaml
kiro:
  - token-file: "~/.aws/sso/cache/kiro-auth-token.json"
    agent-task-type: ""
    start-url: "https://your-company.awsapps.com/start"
    region: "us-east-1"
  - access-token: "aoaAAAAA..."
    refresh-token: "aorAAAAA..."
    profile-arn: "arn:aws:codewhisperer:us-east-1:..."
```

Sources: [internal/config/config.go#L653-L686](internal/config/config.go#L653-L686)

### Model Mapping Reference

All provider API key sections share a common model mapping pattern:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Upstream model identifier used in API requests |
| `alias` | string | Client-facing model name that routes to `name` |
| `display-name` | string | Human-readable name shown in model catalogs |
| `force-mapping` | boolean | Rewrite upstream response model fields back to the alias |

Model exclusions support wildcards: `"gpt-5-*"` (prefix), `"*-preview"` (suffix), `"*flash*"` (substring).

## Model Prefixes and Routing

### Force Model Prefix

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `force-model-prefix` | boolean | `false` | Require explicit prefixes to target prefixed credentials |

When `force-model-prefix` is `false` (default), unprefixed model requests may use credentials that have a prefix set. Setting this to `true` enforces strict namespace isolation: only requests like `teamA/gemini-flash` will match a credential with `prefix: "teamA"`.

### Routing Strategy

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `routing.strategy` | string | `"round-robin"` | Credential selection strategy |
| `routing.session-affinity` | boolean | `false` | Pin sessions to specific credentials |
| `routing.session-affinity-ttl` | string | `"1h"` | How long session bindings persist |

```yaml
routing:
  strategy: "round-robin"   # or "fill-first"
  session-affinity: true
  session-affinity-ttl: "2h"
```

The `"round-robin"` strategy cycles through available credentials equally. `"fill-first"` prefers credentials with lower usage before spilling to others. Session affinity extracts session IDs from multiple sources: `X-Session-ID`, `Session_id`, `X-Client-Request-Id`, `metadata.user_id`, `conversation_id`, or a message hash.

Sources: [internal/config/config.go#L359-L378](internal/config/config.go#L359-L378), [config.example.yaml#L138-L153](config.example.yaml#L138-L153)

## Retry and Cooldown

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `request-retry` | integer | `3` | Number of retries on 403/408/500/502/503/504 |
| `max-retry-credentials` | integer | `0` | Max different credentials to try per request (0 = all) |
| `max-retry-interval` | integer | `30` | Max seconds to wait before retrying a cooled-down credential |
| `disable-cooling` | boolean | `false` | Globally disable auth/model cooldown scheduling |
| `save-cooldown-status` | boolean | `false` | Persist cooldown state as `.cds` files next to auth files |
| `transient-error-cooldown-seconds` | integer | `0` | Cooldown for 408/500/502/503/504 (0 = default 60s, -1 = disabled) |

When a credential hits a rate limit or error, the server temporarily removes it from rotation (cooldown). The `disable-cooling` flag disables this mechanism globally, while `save-cooldown-status` enables persistence so cooldowns survive server restarts.

Sources: [internal/config/config.go#L89-L102](internal/config/config.go#L89-L102), [config.example.yaml#L106-L125](config.example.yaml#L106-L125)

## Quota Exceeded Behavior

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `quota-exceeded.switch-project` | boolean | `true` | Auto-switch to another project on quota exceeded |
| `quota-exceeded.switch-preview-model` | boolean | `true` | Auto-switch to preview model on quota exceeded |
| `quota-exceeded.antigravity-credits` | boolean | `true` | Use credits as last-resort fallback for Claude models |

```yaml
quota-exceeded:
  switch-project: true
  switch-preview-model: true
  antigravity-credits: true
```

Sources: [internal/config/config.go#L348-L357](internal/config/config.go#L348-L357), [config.example.yaml#L155-L161](config.example.yaml#L155-L161)

## Management API

The Management API provides a web-based control panel and programmatic endpoint for runtime configuration.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `remote-management.allow-remote` | boolean | `false` | Allow non-localhost management access |
| `remote-management.secret-key` | string | `""` | Management authentication key (auto-hashed) |
| `remote-management.disable-control-panel` | boolean | `false` | Disable the bundled management UI |
| `remote-management.disable-auto-update-panel` | boolean | `false` | Disable auto-updating the panel from GitHub |
| `remote-management.panel-github-repository` | string | GitHub URL | GitHub repo for the management panel |

```yaml
remote-management:
  allow-remote: false
  secret-key: 'my-management-password'
  disable-control-panel: false
```

The `secret-key` is automatically hashed with bcrypt on first startup and the hashed value is persisted back to the config file. Leave `secret-key` empty to disable the Management API entirely (all `/v0/management` routes return 404).

Sources: [internal/config/config.go#L334-L347](internal/config/config.go#L334-L347), [config.example.yaml#L23-L43](config.example.yaml#L23-L43)

## Logging

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `logging-to-file` | boolean | `false` | Write logs to rotating files instead of stdout |
| `logs-max-total-size-mb` | integer | `0` | Max total log size in MB (0 = unlimited) |
| `error-logs-max-files` | integer | `10` | Max error log files retained (0 = disabled) |

```yaml
logging-to-file: false
logs-max-total-size-mb: 100
error-logs-max-files: 10
```

When `logging-to-file` is `false`, logs go to stdout. When `true`, logs are written to rotating files under a `logs/` directory. The `logs-max-total-size-mb` setting enforces a disk space cap by deleting the oldest files when exceeded.

Sources: [internal/config/config.go#L65-L73](internal/config/config.go#L65-L73)

## Proxy Configuration

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `proxy-url` | string | `""` | Global proxy for outbound requests (socks5/http/https) |

```yaml
proxy-url: "socks5://user:pass@192.168.1.1:1080/"
```

Each provider credential can also specify its own `proxy-url` to override the global setting. Use `"direct"` or `"none"` as the per-entry value to explicitly bypass both the global proxy and environment proxies.

Sources: [internal/config/sdk_config.go#L11-L12](internal/config/sdk_config.go#L11-L12), [config.example.yaml#L127-L129](config.example.yaml#L127-L129)

## Streaming Configuration

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `nonstream-keepalive-interval` | integer | `0` | Blank-line interval for non-streaming responses (seconds, 0 = disabled) |
| `streaming.keepalive-seconds` | integer | `0` | SSE heartbeat interval (0 = disabled) |
| `streaming.bootstrap-retries` | integer | `0` | Retry count before first byte is sent (0 = disabled) |

```yaml
nonstream-keepalive-interval: 30
streaming:
  keepalive-seconds: 15
  bootstrap-retries: 1
```

Keep-alive settings prevent idle timeouts during long operations. Bootstrap retries allow the server to attempt auth rotation before the first byte reaches the client, enabling transparent credential failover for streaming responses.

Sources: [internal/config/sdk_config.go#L60-L74](internal/config/sdk_config.go#L60-L74)

## Image Generation Control

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `disable-image-generation` | bool/string | `false` | Controls image_generation tool availability |
| `gpt-image-2-base-model` | string | `"gpt-5.4-mini"` | Base model for legacy image_generation path |

The `disable-image-generation` setting accepts multiple values:

| Value | Behavior |
|-------|----------|
| `false` | image_generation enabled everywhere (default) |
| `true` | Disabled everywhere including `/v1/images/*` endpoints |
| `"chat"` | Disabled on chat/responses endpoints, enabled on `/v1/images/*` |
| `"passthrough"` | Never inject or strip image_generation (forward client payload unchanged) |

Sources: [internal/config/sdk_config.go#L15-L33](internal/config/sdk_config.go#L15-L33), [internal/config/disable_image_generation_mode.go#L9-L19](internal/config/disable_image_generation_mode.go#L9-L19)

## WebSocket and Authentication

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `ws-auth` | boolean | `true` | Require authentication for WebSocket API (`/v1/ws`) |

```yaml
ws-auth: true
```

When enabled, WebSocket connections must authenticate using the same API key mechanism as HTTP endpoints.

Sources: [internal/config/config.go#L111-L112](internal/config/config.go#L111-L112)

## Claude and Codex Header Defaults

These sections configure fallback headers injected when the client omits them. They are particularly important for maintaining compatibility with upstream provider expectations.

### Claude Header Defaults

```yaml
claude-header-defaults:
  user-agent: "claude-cli/2.1.44 (external, sdk-cli)"
  package-version: "0.74.0"
  runtime-version: "v24.3.0"
  os: "MacOS"
  arch: "arm64"
  timeout: "600"
  stabilize-device-profile: false
```

| Key | Description |
|-----|-------------|
| `user-agent` | HTTP User-Agent header value |
| `package-version` | Claude Code package version identifier |
| `runtime-version` | Node.js runtime version |
| `os` / `arch` | Platform identifiers (pinned when `stabilize-device-profile` is true) |
| `timeout` | Request timeout in seconds |
| `stabilize-device-profile` | When true, pins OS/arch and generates upgradeable software fingerprints |

### Codex Header Defaults

```yaml
codex-header-defaults:
  user-agent: "codex_cli_rs/0.114.0 (Mac OS 14.2.0; x86_64) vscode/1.111.0"
  beta-features: "multi_agent"
```

These apply only to OAuth/file-backed Codex requests when the client does not send its own headers. The `beta-features` header applies only to WebSocket requests.

Sources: [internal/config/config.go#L263-L285](internal/config/config.go#L263-L285)

## OAuth Model Aliases

Global model name aliases allow you to remap upstream model IDs to shorter or more convenient client-facing names across all OAuth-based providers.

```yaml
oauth-model-alias:
  gemini-cli:
    - name: "gemini-2.5-pro"
      alias: "g2.5p"
  claude:
    - name: "claude-sonnet-4-5-20250929"
      alias: "cs4.5"
  codex:
    - name: "gpt-5"
      alias: "g5"
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Upstream model ID |
| `alias` | string | Client-visible model name |
| `fork` | boolean | Keep original and expose alias as a separate model |
| `display-name` | string | Override human-readable catalog name |
| `force-mapping` | boolean | Rewrite response model fields back to the alias |

Supported channels: `gemini-cli`, `vertex`, `aistudio`, `antigravity`, `claude`, `codex`, `kimi`, `kiro`, `github-copilot`.

Default aliases are automatically injected for `kiro` and `github-copilot` channels when not explicitly configured. These defaults expose standard Claude model IDs for Kiro-prefixed upstream models and dot-to-hyphen model ID mappings for GitHub Copilot.

Sources: [internal/config/config.go#L380-L391](internal/config/config.go#L380-L391), [internal/config/oauth_model_alias_defaults.go#L12-L46](internal/config/oauth_model_alias_defaults.go#L12-L46)

## OAuth Model Exclusions

Per-provider global model exclusions remove specific models from OAuth/file-backed authentication listings.

```yaml
oauth-excluded-models:
  vertex:
    - "gemini-3-pro-preview"
  claude:
    - "claude-3-5-haiku-20241022"
  codex:
    - "gpt-5-codex-mini"
  kiro:
    - "kiro-claude-haiku-4-5"
```

Exclusions support exact matches and wildcard patterns (same syntax as per-credential `excluded-models`).

Sources: [internal/config/config.go#L122-L128](internal/config/config.go#L122-L128), [config.example.yaml#L535-L560](config.example.yaml#L535-L560)

## Payload Configuration

Payload rules allow you to inject, override, or filter request parameters based on model name, protocol, and payload content.

```yaml
payload:
  default:
    - models:
        - name: "gemini-2.5-pro"
          protocol: "gemini"
      params:
        "generationConfig.thinkingConfig.thinkingBudget": 32768
  override:
    - models:
        - name: "gpt-5.4-fast"
          protocol: "codex"
      params:
        service_tier: priority
  filter:
    - models:
        - name: "gemini-2.5-pro"
          protocol: "gemini"
      params:
        - "generationConfig.thinkingConfig.thinkingBudget"
```

| Rule Type | Behavior |
|-----------|----------|
| `default` | Set parameters only when missing in the payload |
| `default-raw` | Set raw JSON parameters only when missing |
| `override` | Always set parameters, overwriting existing values |
| `override-raw` | Always set raw JSON parameters |
| `filter` | Remove specified parameters from the payload |

Model rules support wildcards (`"gemini-*"`, `"gpt-*"`) and can be restricted by protocol (`openai`, `gemini`, `claude`, `codex`, `antigravity`), headers, and payload content matching.

Sources: [internal/config/config.go#L392-L443](internal/config/config.go#L392-L443), [config.example.yaml#L497-L586](config.example.yaml#L497-L586)

## Plugin System

The plugin system enables dynamic extension of the proxy with Go shared libraries (`.so`/`.dylib`) implementing a C ABI interface.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `plugins.enabled` | boolean | `false` | Enable dynamic plugin loading |
| `plugins.dir` | string | `"plugins"` | Plugin discovery directory |
| `plugins.store-sources` | list | `[]` | Additional plugin store registries |
| `plugins.store-auth` | list | `[]` | Auth rules for plugin store requests |
| `plugins.configs.<id>.enabled` | boolean | `false` | Enable a specific plugin |
| `plugins.configs.<id>.priority` | integer | `0` | Plugin startup and routing order |

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    my-plugin:
      enabled: true
      priority: 1
      custom-option: "value"
```

The `dir` path supports `~` expansion for the home directory. Plugin configurations under `configs.<id>` are passed directly to the plugin's `ApplyConfig` callback.

Sources: [internal/config/config.go#L195-L210](internal/config/config.go#L195-L210), [config.example.yaml#L52-L82](config.example.yaml#L52-L82)

## Home Control Plane

The Home control plane provides centralized configuration management via a Redis-compatible protocol.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `home.enabled` | boolean | `false` | Enable Home integration |
| `home.host` | string | `"127.0.0.1"` | Home server host |
| `home.port` | integer | `6379` | Home server port |
| `home.password` | string | `""` | Redis AUTH password |

When Home integration is enabled, the server fetches its configuration from the Home server at startup instead of reading from a local YAML file. This is also the mode activated by the `--home host:port` command-line flag.

**Note:** `HomeConfig` fields (host, port, password) are intentionally stripped from local YAML config parsing. They can only be set via the `--home` and `--home-password` command-line flags.

Sources: [internal/config/home.go#L15-L24](internal/config/home.go#L15-L24), [config.example.yaml#L19-L23](config.example.yaml#L19-L23)

## Advanced Features

### Commercial Mode

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `commercial-mode` | boolean | `false` | Disable high-overhead logging and middleware for reduced memory |

### Incognito Browser

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `incognito-browser` | boolean | `false` | Open OAuth URLs in private browsing mode |

Useful for managing multiple accounts simultaneously without logging out of existing sessions. Kiro authentication defaults to incognito mode for multi-account support.

### Claude Cloak Mode

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `disable-claude-cloak-mode` | boolean | `false` | Globally disable Claude request cloaking |

When `true`, every Claude credential defaults to no cloaking. Individual credentials can still override this via their own `cloak.mode` setting.

### Passthrough Headers

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `passthrough-headers` | boolean | `false` | Forward filtered upstream response headers to clients |

### Usage Statistics

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `usage-statistics-enabled` | boolean | `false` | Enable in-memory usage data aggregation |
| `redis-usage-queue-retention-seconds` | integer | `60` | How long usage items are retained (max: 3600) |

### Video Result Cache

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `video-result-auth-cache-ttl` | string | `"3h"` | How long video IDs stay bound to the creating credential |

### Auth Auto-Refresh Workers

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `auth-auto-refresh-workers` | integer | `0` | Override core token refresh worker pool size (0 = default 16) |

### Pprof Debug Server

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `pprof.enable` | boolean | `false` | Enable pprof HTTP debug server |
| `pprof.addr` | string | `"127.0.0.1:8316"` | Bind address for pprof server |

Always keep the pprof server bound to localhost in production.

Sources: [config.example.yaml#L83-L105](config.example.yaml#L83-L105)

## Environment Variables

The server loads environment variables from a `.env` file in the working directory using `godotenv`. These are primarily used for remote storage backends and the Home control plane.

| Variable | Description |
|----------|-------------|
| `DEPLOY` | Set to `"cloud"` for cloud deploy standby mode |
| `HOME_JWT` | JWT token for Home control plane mTLS bootstrap |
| `MANAGEMENT_PASSWORD` | Management web UI password |
| `PGSTORE_DSN` | PostgreSQL connection string for token storage |
| `PGSTORE_SCHEMA` | PostgreSQL schema name |
| `PGSTORE_LOCAL_PATH` | Local spool path for PostgreSQL store |
| `GITSTORE_GIT_URL` | Git repository URL for config storage |
| `GITSTORE_GIT_USERNAME` | Git repository username |
| `GITSTORE_GIT_TOKEN` | Git personal access token |
| `GITSTORE_LOCAL_PATH` | Local clone path for git store |
| `GITSTORE_GIT_BRANCH` | Git branch to track |
| `OBJECTSTORE_ENDPOINT` | S3-compatible object store endpoint |
| `OBJECTSTORE_BUCKET` | Object store bucket name |
| `OBJECTSTORE_ACCESS_KEY` | Object store access key |
| `OBJECTSTORE_SECRET_KEY` | Object store secret key |
| `OBJECTSTORE_LOCAL_PATH` | Local cache path for object store |

Only one storage backend (PostgreSQL, Git, or Object Store) can be active at a time. When a storage backend is configured, it takes precedence over local YAML configuration files.

Sources: [cmd/server/main.go#L249-L319](cmd/server/main.go#L249-L319), [.env.example#L1-L35](.env.example#L1-L35)

## Command-Line Flags Reference

The server binary accepts numerous command-line flags for authentication flows, configuration overrides, and operational modes.

| Flag | Description |
|------|-------------|
| `-config <path>` | Path to configuration file |
| `-password <key>` | Local management password |
| `-home <host:port>` | Home control plane address (skips local config) |
| `-home-password <pwd>` | Home Redis AUTH password |
| `-home-jwt <jwt>` | Home control plane JWT |
| `-tui` | Start with terminal management UI |
| `-standalone` | Start embedded local server in TUI mode |
| `-local-model` | Use only embedded model catalogs (skip remote fetch) |
| `-incognito` | Force open OAuth URLs in incognito mode |
| `-no-incognito` | Force normal browser mode for OAuth |
| `-no-browser` | Don't open browser for OAuth flows |
| `-codex-login` | Login to Codex via OAuth |
| `-claude-login` | Login to Claude via OAuth |
| `-login` | Login Google Account |
| `-kilo-login` | Login to Kilo AI via device flow |
| `-antigravity-login` | Login to Antigravity via OAuth |
| `-kimi-login` | Login to Kimi via OAuth |
| `-cursor-login` | Login to Cursor via OAuth |
| `-kiro-login` | Login to Kiro via Google OAuth |
| `-kiro-aws-login` | Login to Kiro via AWS Builder ID (device code) |
| `-kiro-aws-authcode` | Login to Kiro via AWS Builder ID (auth code) |
| `-kiro-import` | Import Kiro token from IDE cache |
| `-kiro-idc-login` | Login to Kiro via IAM Identity Center |
| `-gitlab-login` | Login to GitLab Duo via OAuth |
| `-github-copilot-login` | Login to GitHub Copilot via device flow |
| `-codebuddy-login` | Login to CodeBuddy via browser OAuth |
| `-vertex-import <path>` | Import Vertex service account key JSON |

Sources: [cmd/server/main.go#L128-L170](cmd/server/main.go#L128-L170)

## Docker Configuration

The Docker Compose setup maps three critical volumes:

```yaml
volumes:
  - ${CLI_PROXY_CONFIG_PATH:-./config.yaml}:/CLIProxyAPI/config.yaml
  - ${CLI_PROXY_AUTH_PATH:-./auths}:/root/.cli-proxy-api
  - ${CLI_PROXY_LOG_PATH:-./logs}:/CLIProxyAPI/logs
```

| Volume | Purpose |
|--------|---------|
| `config.yaml` | Configuration file (read by the server) |
| `auths` → `~/.cli-proxy-api` | Authentication directory (OAuth tokens) |
| `logs` | Log files (when `logging-to-file: true`) |

Set the corresponding environment variables to customize paths. For Home control plane deployments, no local config file is needed — pass `-home host:port` via the `command` directive.

Sources: [docker-compose.yml#L17-L28](docker-compose.yml#L17-L28)

## Next Steps

Now that you understand the configuration options, continue your journey:

- Learn how the server starts and processes command-line flags in [Application Entry Point and CLI Flags](5-application-entry-point-and-cli-flags)
- Understand how the HTTP server multiplexes protocols in [HTTP Server and Protocol Multiplexing](6-http-server-and-protocol-multiplexing)
- Set up provider authentication flows in [Authentication System and Provider Login Flows](7-authentication-system-and-provider-login-flows)
- See how configuration changes are detected at runtime in [Configuration Hot-Reload and File Watching](15-configuration-hot-reload-and-file-watching)