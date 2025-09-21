# Database MCP Server

A lightweight, cross-platform MCP (Model Context Protocol) server written in **Go** that enables AI assistants to safely query and analyze databases.

## Features

- **Multi-Database Support**: PostgreSQL (with PostGIS) and ClickHouse
- **Security First**: Read-only by default with configurable safety policies
- **MCP Protocol**: Standard JSON-RPC 2.0 interface for AI assistants
- **Production Ready**: Observability, metrics, and audit logging
- **Developer Friendly**: Easy setup with Docker Compose

## Quick Start

### Prerequisites

- Go 1.21 or later
- Docker and Docker Compose (for local development)

### Development Setup

```bash
# Clone the repository
git clone <repository-url>
cd database-mcp

# Build the binary
make build

# Run with Docker Compose (includes PostgreSQL + ClickHouse)
make dev

# Run tests
make test
```

### Usage

```bash
# Start the MCP server (stdio mode)
./bin/database-mcp

# Or with environment configuration
DB_PRIMARY_URL="postgres://user:pass@localhost/db" \
DB_ANALYTICS_URL="clickhouse://localhost:9000/default" \
./bin/database-mcp
```

## Architecture

```
AI Assistant (Claude, GPT, etc.) ←→ MCP Server ←→ Database
```

The MCP server provides these tools to AI assistants:
- `list_schemas` - List all available database schemas
- `list_tables` - List tables in a schema with metadata
- `describe_table` - Get detailed table structure and constraints
- `run_sql` - Execute SQL queries with safety checks
- `explain_sql` - Get query execution plans

## Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `DB_PRIMARY_URL` | PostgreSQL connection string | Required |
| `DB_ANALYTICS_URL` | ClickHouse connection string | Optional |
| `MCP_MODE` | Transport mode (stdio/http) | `stdio` |
| `READ_ONLY` | Enable read-only mode | `true` |
| `MAX_ROWS` | Maximum rows per query | `10000` |
| `QUERY_TIMEOUT_MS` | Query timeout in milliseconds | `30000` |

## Development

See [PRD.md](PRD.md) for detailed project roadmap and implementation phases.

## License

MIT License - see [LICENSE](LICENSE) file for details.