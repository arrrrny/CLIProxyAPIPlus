This page provides comprehensive guidance for deploying CLIProxyAPI Plus using Docker, covering both standalone and cluster configurations. Docker deployment offers a containerized, reproducible environment that simplifies setup and management across different infrastructure platforms.

## Architecture Overview

The Docker deployment architecture follows a multi-stage build pattern optimized for production use. The build process compiles the Go application in a build stage, then copies only the necessary binary and configuration into a minimal runtime image.

```mermaid
graph TB
    subgraph "Build Stage"
        A[Go 1.26 Builder] --> B[Compile Binary]
        B --> C[CGO_ENABLED=0]
        C --> D[Static Binary]
    end
    
    subgraph "Runtime Stage"
        E[Debian Bookworm] --> F[Minimal Runtime]
        F --> G[CLIProxyAPIPlus Binary]
        F --> H[Configuration]
        F --> I[Certificate Store]
    end
    
    D --> G
    
    subgraph "Volume Mounts"
        J[config.yaml] --> H
        K[auths/] --> L[Authentication Tokens]
        M[logs/] --> N[Application Logs]
    end
    
    subgraph "Ports"
        O[8317 - Main API]
        P[8085 - Alternative]
        Q[1455 - Additional]
        R[54545 - WebSocket]
        S[51121 - Secondary]
        T[11451 - Tertiary]
    end
    
    G --> O
    G --> P
    G --> Q
    G --> R
    G --> S
    G --> T
```

The build process uses Go build flags to inject version information (`-ldflags="-s -w"`) for production optimization, creating a statically linked binary that runs without external dependencies. The runtime image includes only essential system packages (`tzdata`, `ca-certificates`) to minimize attack surface and image size.

Sources: [Dockerfile](Dockerfile#L1-L37)

## Quick Start with Pre-built Images

The fastest way to deploy CLIProxyAPI Plus is using the pre-built Docker image from the official registry. This approach requires no build tools or Go installation.

### Prerequisites

Before deploying, ensure you have:
- Docker Engine 20.10+ or Docker Desktop
- Docker Compose v2.0+
- A configuration file (copy from `config.example.yaml`)
- Authentication tokens for your chosen AI providers

### Single Command Deployment

Execute the build script which provides an interactive menu:

```bash
# For Linux/macOS
./docker-build.sh

# For Windows PowerShell
.\docker-build.ps1
```

Select option **1** to run using the pre-built image. The script will:
1. Pull the latest image from `eceasy/cli-proxy-api-plus:latest`
2. Create necessary volume mounts
3. Start the service in detached mode

Sources: [docker-build.sh](docker-build.sh#L1-L64)

### Manual Docker Compose Deployment

For more control, use Docker Compose directly:

```bash
# Create your configuration
cp config.example.yaml config.yaml
# Edit config.yaml with your settings

# Create authentication directory
mkdir -p auths

# Start the service
docker compose up -d
```

The `docker-compose.yml` file defines the service with appropriate volume mounts, port mappings, and restart policies. The service automatically pulls the latest image unless you specify a local build.

Sources: [docker-compose.yml](docker-compose.yml#L1-L29)

## Building from Source

For developers who need custom builds or want to contribute to the project, building from source provides full control over the binary and dependencies.

### Build Arguments

The Dockerfile accepts several build arguments that inject version information into the binary:

| Argument | Default | Description |
|----------|---------|-------------|
| `VERSION` | `dev` | Git tag or version string |
| `COMMIT` | `none` | Short git commit hash |
| `BUILD_DATE` | `unknown` | UTC build timestamp (ISO 8601) |

These arguments enable version tracking and debugging in production deployments. The build process automatically extracts version information from git when using the build scripts.

Sources: [Dockerfile](Dockerfile#L12-L14)

### Development Build Process

When selecting option **2** in the build script, it performs these steps:

1. **Extracts version information** from git repository
2. **Sets local image tag** (`cli-proxy-api:local`) to avoid conflicts with production images
3. **Builds the Docker image** with version injection
4. **Starts the service** with `--pull never` to use the locally built image

The build script uses `git describe --tags --always --dirty` for version strings, ensuring consistent versioning across environments.

```mermaid
flowchart TD
    A[Start Build Script] --> B{Select Option}
    B -->|Option 1| C[Pull Pre-built Image]
    B -->|Option 2| D[Extract Git Version Info]
    
    D --> E[Set Local Image Tag]
    E --> F[Build Docker Image]
    F --> G[Start Service with Local Image]
    
    C --> H[Start Service with Remote Image]
    
    G --> I[Monitor with docker compose logs -f]
    H --> I
```

Sources: [docker-build.sh](docker-build.sh#L25-L64)

## Configuration Management

Docker deployment requires proper configuration handling for both the application settings and authentication credentials.

### Volume Mounts

The Docker Compose configuration mounts three critical directories:

| Mount Point | Host Path | Purpose |
|-------------|-----------|---------|
| `/CLIProxyAPI/config.yaml` | `./config.yaml` | Application configuration |
| `/root/.cli-proxy-api` | `./auths` | Authentication tokens |
| `/CLIProxyAPI/logs` | `./logs` | Application logs |

These mounts ensure configuration persistence across container restarts and enable easy updates without rebuilding the image.

### Environment Variables

The `.env.example` file provides templates for optional environment variables:

```bash
# Management Web UI password
MANAGEMENT_PASSWORD=change-me-to-a-strong-password

# Postgres Token Store (optional)
PGSTORE_DSN=postgresql://user:pass@localhost:5432/cliproxy

# Git-Backed Config Store (optional)
GITSTORE_GIT_URL=https://github.com/your-org/cli-proxy-config.git
```

Environment variables are loaded automatically from `.env` files in the working directory. Only configure the storage backends you plan to use.

Sources: [.env.example](.env.example#L1-L35)

### Configuration File Structure

The main configuration file (`config.yaml`) controls all application behavior. Key sections include:

- **Server settings**: `host`, `port`, `tls`
- **Authentication**: `auth-dir`, `api-keys`
- **Provider configurations**: `gemini-api-key`, `claude-api-key`, `codex-api-key`
- **Management API**: `remote-management`
- **Plugins**: `plugins`

The application supports hot-reloading for many configuration changes without requiring container restarts.

Sources: [config.example.yaml](config.example.yaml#L1-L200)

## Cluster Deployment

For production environments requiring high availability and centralized management, CLIProxyAPI Plus supports cluster deployment with the Home control plane.

### Cluster Architecture

```mermaid
graph TB
    subgraph "Home Control Plane"
        A[Home Service] --> B[Configuration Management]
        A --> C[Certificate Authority]
        A --> D[Plugin Distribution]
    end
    
    subgraph "CLIProxyAPI Cluster"
        E[Node 1] --> F[Local Auth Cache]
        G[Node 2] --> H[Local Auth Cache]
        I[Node 3] --> J[Local Auth Cache]
    end
    
    A --> E
    A --> G
    A --> I
    
    K[JWT Authentication] --> E
    K --> G
    K --> I
    
    subgraph "Shared Storage"
        L[Redis] --> M[Configuration Sync]
        L --> N[Usage Statistics]
    end
    
    E --> L
    G --> L
    I --> L
```

### Cluster Configuration

The `docker-compose.cluster.yml` file defines the cluster deployment pattern:

```yaml
command: >
  sh -eu -c '
    if [ -z "$$HOME_JWT" ]; then
      echo "HOME_JWT is required" >&2
      exit 1
    fi
    exec ./CLIProxyAPI -home-jwt "$$HOME_JWT"
  '
```

This command ensures the `HOME_JWT` environment variable is present before starting the service. The JWT token authenticates the node with the Home control plane.

### Obtaining the JWT Token

After deploying the Home control plane, obtain the JWT token:

```bash
curl -sS -X POST \
  "http://<home-host>:8327/v0/management/certificates/clients" \
  -H "X-MANAGEMENT-KEY: <management-key>" | jq -r '.home_jwt'
```

Store this token in your `.env` file or export it before starting the cluster.

Sources: [.env.cluster.example](.env.cluster.example#L1-L6)

### Cluster Deployment Steps

1. **Deploy Home Control Plane** first and obtain the JWT token
2. **Configure cluster nodes** with the JWT token
3. **Start cluster services**:
   ```bash
   # Set the JWT token
   export HOME_JWT=your-jwt-token-here
   
   # Start with cluster configuration
   docker compose -f docker-compose.cluster.yml up -d
   ```

The cluster configuration automatically handles:
- Configuration synchronization from Home
- Certificate management
- Plugin distribution and updates
- Usage statistics aggregation

Sources: [docker-compose.cluster.yml](docker-compose.cluster.yml#L1-L29)

## Port Configuration

CLIProxyAPI Plus exposes multiple ports for different services and protocols:

| Port | Protocol | Purpose |
|------|----------|---------|
| 8317 | HTTP/HTTPS | Main API endpoint |
| 8085 | HTTP | Alternative API endpoint |
| 1455 | HTTP | Additional service |
| 54545 | WebSocket | Real-time communication |
| 51121 | HTTP | Secondary endpoints |
| 11451 | HTTP | Tertiary endpoints |

The main API port (8317) handles all standard AI API requests, while additional ports support specific use cases like WebSocket connections for streaming responses.

### Port Customization

To change the exposed ports, modify the `docker-compose.yml` file:

```yaml
ports:
  - "9000:8317"  # Map host port 9000 to container port 8317
```

Ensure your configuration file (`config.yaml`) matches the internal port configuration if you change the container's internal port mapping.

## Security Considerations

### Authentication Directory Security

The authentication directory (`auths`) contains sensitive OAuth tokens and API keys. Ensure proper file permissions:

```bash
# Set restrictive permissions on auth directory
chmod 700 auths
chown -R $(id -u):$(id -g) auths
```

### TLS Configuration

Enable TLS in your configuration file for production deployments:

```yaml
tls:
  enable: true
  cert: /path/to/certificate.pem
  key: /path/to/private-key.pem
```

Mount the certificate files as additional volumes in your Docker Compose configuration.

### Network Security

Restrict network access to necessary ports only. In production environments, consider using a reverse proxy (Nginx, Traefik) for TLS termination and access control.

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| Container exits immediately | Check logs with `docker compose logs` |
| Configuration not loading | Verify volume mount paths and file permissions |
| Authentication failures | Ensure `auths` directory contains valid tokens |
| Port conflicts | Change host port mappings in `docker-compose.yml` |
| Memory issues | Increase Docker memory limits or enable `commercial-mode` |

### Log Analysis

Access container logs for debugging:

```bash
# Follow real-time logs
docker compose logs -f

# View recent logs
docker compose logs --tail=100

# View specific service logs
docker compose logs cli-proxy-api
```

### Health Checks

Monitor the service health:

```bash
# Check if the API is responding
curl http://localhost:8317/v1/models

# Check container status
docker compose ps

# Inspect container resources
docker stats cli-proxy-api-plus
```

## Next Steps

After successful Docker deployment, explore these related topics:

- [Configuration Reference](3-configuration-reference) - Detailed configuration options
- [Application Entry Point and CLI Flags](5-application-entry-point-and-cli-flags) - Advanced startup options
- [Home Control Plane Integration](16-home-control-plane-integration) - Cluster management details
- [Management API](17-management-api) - Remote administration capabilities

For production deployments, consider implementing monitoring, log aggregation, and backup strategies for your configuration and authentication data.