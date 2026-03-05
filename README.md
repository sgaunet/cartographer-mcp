[![License](https://img.shields.io/github/license/sgaunet/cartographer-mcp.svg)](LICENSE)

# Cartographer MCP Server

A Model Context Protocol (MCP) server that crawls GitLab organizations to build a service catalog with dependency graphs. Give your AI coding assistant instant knowledge of your entire architecture — no more explaining your services at the start of every session.

## Features

- **Service Discovery**: Automatically crawl GitLab groups to discover all projects and their metadata
- **Dependency Graphs**: Query forward and reverse dependencies between services, including transitive chains
- **Rich Metadata**: Collect README content, latest versions, descriptions, and lifecycle status
- **`.cartographer.yaml` Support**: Enrich auto-discovered data with human-defined service types, owners, dependencies, and outputs
- **Impact Analysis**: Answer "what would break if I change this service?" instantly
- **Offline Queries**: All queries served from local cache — zero network calls during query handling
- **Rate-Limit Resilient**: Automatic retry with exponential backoff for GitLab API rate limits
- **MCP Architecture**: Seamless integration with Claude Code via stdio communication

## Quick Start

### Prerequisites

- Go 1.23+
- GitLab personal access token with `read_api` scope
- At least one GitLab group to crawl

### Build

```bash
go build -o cartographer ./cmd/cartographer/
```

### Configuration

```bash
# Required: GitLab token
export GITLAB_TOKEN="glpat-xxxxxxxxxxxx"

# Optional: For self-hosted GitLab (defaults to https://gitlab.com/)
export GITLAB_URI="https://gitlab.example.com/"

# Optional: Groups via environment variable (comma-separated)
export CARTOGRAPHER_GROUPS="myorg/platform,myorg/product"
```

Or create a config file at `$HOME/.config/cartographer/config.yaml`:

```yaml
groups:
  - "myorg/platform"
  - "myorg/product"
```

### Add to Claude Code

```bash
claude mcp add cartographer -s user -- /path/to/cartographer
```

Or add to your `.mcp.json`:

```json
{
  "mcpServers": {
    "cartographer": {
      "command": "/path/to/cartographer",
      "env": {
        "GITLAB_TOKEN": "glpat-xxxxxxxxxxxx"
      }
    }
  }
}
```

### First Usage

Trigger an initial cache refresh:

```
Refresh the cartographer cache.
```

Then query your architecture:

```
What services do we have?
```

```
Tell me about the payment service.
```

```
What depends on the user service?
```

```
Which services send emails?
```

## Available Tools

| Tool | Description |
|------|-------------|
| `list_services` | List all discovered services with filtering by type, lifecycle, or tag |
| `get_service` | Get full details for a specific service including dependencies and outputs |
| `get_dependencies` | Get forward dependencies of a service (what it depends on) |
| `get_dependents` | Get reverse dependencies (what depends on this service) |
| `search_services` | Search services by keyword across names, descriptions, tags, and outputs |
| `refresh_cache` | Trigger a full cache refresh from GitLab |

## `.cartographer.yaml`

Place at the root of any GitLab repository to enrich auto-discovered metadata:

```yaml
schema_version: 1
service:
  name: "payment-service"
  type: "api"
  lifecycle: "production"
  owner: "payments-team"
  tags:
    - "payments"
    - "stripe"
dependencies:
  - service: "user-service"
    type: "api"
  - service: "notification-service"
    type: "events"
outputs:
  - name: "payment-events"
    type: "events"
    description: "Payment lifecycle events"
  - name: "payments-api"
    type: "api"
    description: "REST API for payment operations"
```

Projects without this file are still discovered with auto-discovered metadata (name, description, README, version).

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GITLAB_TOKEN` | Yes | — | GitLab personal access token |
| `GITLAB_URI` | No | `https://gitlab.com/` | GitLab instance URL |
| `CARTOGRAPHER_GROUPS` | No | — | Comma-separated group paths (overrides config file) |
| `CARTOGRAPHER_CACHE_DIR` | No | `~/.config/cartographer/cache` | Cache directory path |
| `CARTOGRAPHER_CONFIG` | No | `~/.config/cartographer/config.yaml` | Config file path |

## Development

```bash
# Build
go build -o cartographer ./cmd/cartographer/

# Run tests
go test ./...

# Lint
golangci-lint run

# Vet
go vet ./...
```

## License

MIT License. See [LICENSE](LICENSE) for details.
