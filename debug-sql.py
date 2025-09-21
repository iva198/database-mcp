#!/usr/bin/env python3
"""
Quick debug script to test the MCP server's SQL capabilities
"""

import json
import subprocess
import sys
import time
import os


def test_sql_debug():
    """Test SQL to debug what database we're really connected to"""

    env = os.environ.copy()
    env["DB_PRIMARY_URL"] = (
        "postgres://postgres:password@localhost:5433/postgres?sslmode=disable"
    )
    env["DB_ANALYTICS_URL"] = "clickhouse://default:@localhost:9001/default"

    process = subprocess.Popen(
        ["./bin/database-mcp"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        env=env,
        bufsize=0,
    )

    time.sleep(2)  # Wait for startup

    # Initialize
    init_cmd = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "debug-client", "version": "1.0.0"},
        },
    }

    process.stdin.write(json.dumps(init_cmd) + "\n")
    process.stdin.flush()
    response = process.stdout.readline()
    print("Init response:", response.strip())

    # Test SQL query to see what schemas exist
    sql_cmd = {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "tools/call",
        "params": {
            "name": "run_sql",
            "arguments": {
                "database": "primary",
                "query": "SELECT schema_name FROM information_schema.schemata WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast') ORDER BY schema_name",
                "limit": 10,
            },
        },
    }

    process.stdin.write(json.dumps(sql_cmd) + "\n")
    process.stdin.flush()
    response = process.stdout.readline()
    print("SQL query response:", response.strip())

    # Clean up
    process.terminate()
    process.wait()


if __name__ == "__main__":
    test_sql_debug()
