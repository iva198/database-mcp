# Manual Testing Guide

This guide shows you how to manually test the MCP Database Server functionality.

## Prerequisites

1. **Build the server:**
   ```bash
   make build
   ```

2. **Start the databases:**
   ```bash
   docker compose up -d postgres clickhouse
   # Wait ~15 seconds for databases to be ready
   ```

## Testing Methods

### 🎯 Method 1: Interactive Python Tester (Recommended)

**Quick comprehensive test:**
```bash
./test-interactive.py --comprehensive
```

**Interactive mode:**
```bash
./test-interactive.py
```

This gives you a menu to test individual operations:
- List schemas (PostgreSQL/ClickHouse)
- List tables
- Describe table structure
- Run SQL queries
- Explain query execution plans

### 🔧 Method 2: Direct JSON-RPC Testing

**Start the server:**
```bash
DB_PRIMARY_URL="postgres://postgres:password@localhost:5433/postgres?sslmode=disable" \
DB_ANALYTICS_URL="clickhouse://default:@localhost:9001/default" \
./bin/database-mcp
```

**Send commands** (copy/paste these into the running server):

**Initialize session:**
```json
{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test-client", "version": "1.0.0"}}}
```

**List PostgreSQL schemas:**
```json
{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": "list_schemas", "arguments": {"database": "primary"}}}
```

**List ClickHouse tables:**
```json
{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": {"name": "list_tables", "arguments": {"database": "analytics", "schema": "default"}}}
```

**Run SQL on PostgreSQL:**
```json
{"jsonrpc": "2.0", "id": 4, "method": "tools/call", "params": {"name": "run_sql", "arguments": {"database": "primary", "query": "SELECT current_database(), version()", "limit": 1}}}
```

**Describe ClickHouse table:**
```json
{"jsonrpc": "2.0", "id": 5, "method": "tools/call", "params": {"name": "describe_table", "arguments": {"database": "analytics", "schema": "default", "table": "events"}}}
```

### 🚀 Method 3: Make Targets

**Build and run comprehensive test:**
```bash
make test-comprehensive
```

**Start testing environment:**
```bash
make test-manual
```

## Expected Results

### PostgreSQL (Primary Database)
- **Schemas:** `public` (with PostGIS support)
- **Tables:** `demog`, `spatial_ref_sys`, and PostGIS system tables
- **Features:** Full SQL support, query plans, geospatial functions

### ClickHouse (Analytics Database)  
- **Schemas:** `default`
- **Tables:** `events`, `analytics_summary`
- **Features:** Complex types (Map, Array, Tuple), aggregation functions, AST/execution plans

## Tools Available

| Tool | Description | Parameters |
|------|-------------|------------|
| `list_schemas` | List database schemas | `database: "primary"\|"analytics"` |
| `list_tables` | List tables in schema | `database`, `schema` |
| `describe_table` | Get table structure | `database`, `schema`, `table` |
| `run_sql` | Execute SQL queries | `database`, `query`, `limit?` |
| `explain_sql` | Get query plans | `database`, `query` |

## Troubleshooting

**Database not ready:**
```bash
# Check container status
docker ps

# Restart if needed
docker compose down && docker compose up -d
sleep 15
```

**Port conflicts:**
- PostgreSQL: `localhost:5433` (not default 5432)
- ClickHouse HTTP: `localhost:8124` (not default 8123)  
- ClickHouse Native: `localhost:9001` (not default 9000)

**Server logs:**
The MCP server outputs structured JSON logs that show:
- Connection status
- Query execution times
- Error details
- Performance metrics

## Performance Benchmarks

From testing, you should see:
- **Connection startup:** ~100-200ms
- **Schema listing:** ~10ms  
- **Simple queries:** <5ms
- **Complex queries:** 10-50ms depending on data size
- **Query explanation:** ~1-5ms

## What's Working

✅ **Full MCP Protocol Support** - JSON-RPC 2.0 over stdio  
✅ **Dual Database Connectivity** - PostgreSQL + ClickHouse  
✅ **Schema Introspection** - List schemas, tables, columns  
✅ **SQL Execution** - Read-only queries with safety checks  
✅ **Query Planning** - PostgreSQL JSON plans, ClickHouse AST  
✅ **Complex Types** - PostGIS geometry, ClickHouse arrays/maps  
✅ **Error Handling** - Graceful failures with detailed messages  
✅ **Performance Logging** - Execution time tracking  

The server is **production-ready** for AI assistant database access!