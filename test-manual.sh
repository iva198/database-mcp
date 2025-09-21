#!/bin/bash

# Manual MCP Database Server Testing Script
# This script helps you test the MCP server interactively

echo "🔧 MCP Database Server - Manual Testing"
echo "======================================="

# Check if databases are running
echo "1. Checking database containers..."
if ! docker ps | grep -q "database-mcp-postgres"; then
    echo "❌ PostgreSQL container not running. Starting databases..."
    docker compose up -d postgres clickhouse
    echo "⏱️  Waiting for databases to be ready..."
    sleep 15
else
    echo "✅ Databases are running"
fi

echo ""
echo "2. Available test commands:"
echo ""
echo "🔹 Test MCP Server Startup:"
echo 'DB_PRIMARY_URL="postgres://postgres:password@localhost:5433/postgres?sslmode=disable" DB_ANALYTICS_URL="clickhouse://default:@localhost:9001/default" ./bin/database-mcp'
echo ""
echo "🔹 Send MCP Commands (copy and paste these into the running server):"
echo ""
echo "Initialize:"
echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test-client", "version": "1.0.0"}}}'
echo ""
echo "List Schemas (PostgreSQL):"
echo '{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": "list_schemas", "arguments": {"database": "primary"}}}'
echo ""
echo "List Schemas (ClickHouse):"
echo '{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": {"name": "list_schemas", "arguments": {"database": "analytics"}}}'
echo ""
echo "List Tables (PostgreSQL public schema):"
echo '{"jsonrpc": "2.0", "id": 4, "method": "tools/call", "params": {"name": "list_tables", "arguments": {"database": "primary", "schema": "public"}}}'
echo ""
echo "List Tables (ClickHouse default schema):"
echo '{"jsonrpc": "2.0", "id": 5, "method": "tools/call", "params": {"name": "list_tables", "arguments": {"database": "analytics", "schema": "default"}}}'
echo ""
echo "Describe Table (ClickHouse events table):"
echo '{"jsonrpc": "2.0", "id": 6, "method": "tools/call", "params": {"name": "describe_table", "arguments": {"database": "analytics", "schema": "default", "table": "events"}}}'
echo ""
echo "Run SQL (PostgreSQL):"
echo '{"jsonrpc": "2.0", "id": 7, "method": "tools/call", "params": {"name": "run_sql", "arguments": {"database": "primary", "query": "SELECT current_database(), version()", "limit": 1}}}'
echo ""
echo "Run SQL (ClickHouse):"
echo '{"jsonrpc": "2.0", "id": 8, "method": "tools/call", "params": {"name": "run_sql", "arguments": {"database": "analytics", "query": "SELECT count(*) as total_events FROM events", "limit": 10}}}'
echo ""
echo "Explain SQL (PostgreSQL):"
echo '{"jsonrpc": "2.0", "id": 9, "method": "tools/call", "params": {"name": "explain_sql", "arguments": {"database": "primary", "query": "SELECT * FROM pg_tables LIMIT 5"}}}'
echo ""
echo "Explain SQL (ClickHouse):"
echo '{"jsonrpc": "2.0", "id": 10, "method": "tools/call", "params": {"name": "explain_sql", "arguments": {"database": "analytics", "query": "SELECT event_type, count(*) FROM events GROUP BY event_type"}}}'
echo ""
echo "🔹 To exit the server, press Ctrl+C"
echo ""
echo "Ready to test! Run the commands above step by step."