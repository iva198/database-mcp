#!/usr/bin/env python3
"""
Interactive MCP Database Server Tester
=====================================

This script provides an interactive way to test the MCP Database Server
by sending JSON-RPC commands and displaying formatted responses.
"""

import json
import subprocess
import sys
import time
import threading
import queue
import os
from typing import Dict, Any, Optional


class MCPTester:
    def __init__(self):
        self.process = None
        self.request_id = 1

    def start_server(self) -> bool:
        """Start the MCP server process"""
        try:
            env = os.environ.copy()
            env["DB_PRIMARY_URL"] = (
                "postgres://postgres:password@localhost:5433/postgres?sslmode=disable"
            )
            env["DB_ANALYTICS_URL"] = "clickhouse://default:@localhost:9001/default"

            self.process = subprocess.Popen(
                ["./bin/database-mcp"],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                env=env,
                bufsize=0,
            )

            # Wait a moment for startup
            time.sleep(1)

            if self.process.poll() is None:
                print("✅ MCP Server started successfully")
                return True
            else:
                print("❌ Failed to start MCP Server")
                return False

        except Exception as e:
            print(f"❌ Error starting server: {e}")
            return False

    def send_command(
        self, method: str, params: Dict[str, Any] = None
    ) -> Optional[Dict]:
        """Send a JSON-RPC command to the server"""
        if not self.process:
            print("❌ Server not running")
            return None

        command = {
            "jsonrpc": "2.0",
            "id": self.request_id,
            "method": method,
            "params": params or {},
        }

        try:
            command_str = json.dumps(command) + "\n"
            self.process.stdin.write(command_str)
            self.process.stdin.flush()

            # Read response
            response_line = self.process.stdout.readline()
            if response_line:
                response = json.loads(response_line.strip())
                self.request_id += 1
                return response
            else:
                print("❌ No response from server")
                return None

        except Exception as e:
            print(f"❌ Error sending command: {e}")
            return None

    def print_response(self, response: Dict):
        """Pretty print the response"""
        if "error" in response:
            print(f"❌ Error: {response['error']}")
        elif "result" in response:
            print("✅ Success:")
            print(json.dumps(response["result"], indent=2))
        else:
            print(f"📄 Raw response: {json.dumps(response, indent=2)}")
        print("-" * 50)

    def initialize(self):
        """Initialize the MCP session"""
        print("🔧 Initializing MCP session...")
        response = self.send_command(
            "initialize",
            {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {"name": "manual-tester", "version": "1.0.0"},
            },
        )
        if response:
            self.print_response(response)
            return True
        return False

    def test_list_schemas(self, connection: str):
        """Test listing schemas"""
        print(f"📋 Listing schemas for {connection}...")
        response = self.send_command(
            "tools/call",
            {"name": "list_schemas", "arguments": {"database": connection}},
        )
        if response:
            self.print_response(response)

    def test_list_tables(self, connection: str, schema: str):
        """Test listing tables"""
        print(f"📋 Listing tables for {connection}.{schema}...")
        response = self.send_command(
            "tools/call",
            {
                "name": "list_tables",
                "arguments": {"database": connection, "schema": schema},
            },
        )
        if response:
            self.print_response(response)

    def test_describe_table(self, connection: str, schema: str, table: str):
        """Test describing a table"""
        print(f"🔍 Describing table {connection}.{schema}.{table}...")
        response = self.send_command(
            "tools/call",
            {
                "name": "describe_table",
                "arguments": {"database": connection, "schema": schema, "table": table},
            },
        )
        if response:
            self.print_response(response)

    def test_run_sql(self, connection: str, query: str, limit: int = 10):
        """Test running SQL"""
        print(f"⚡ Running SQL on {connection}: {query[:50]}...")
        response = self.send_command(
            "tools/call",
            {
                "name": "run_sql",
                "arguments": {"database": connection, "query": query, "limit": limit},
            },
        )
        if response:
            self.print_response(response)

    def test_explain_sql(self, connection: str, query: str):
        """Test explaining SQL"""
        print(f"📊 Explaining SQL on {connection}: {query[:50]}...")
        response = self.send_command(
            "tools/call",
            {
                "name": "explain_sql",
                "arguments": {"database": connection, "query": query},
            },
        )
        if response:
            self.print_response(response)

    def run_comprehensive_test(self):
        """Run a comprehensive test suite"""
        print("🚀 Running Comprehensive MCP Database Server Test")
        print("=" * 60)

        if not self.initialize():
            print("❌ Failed to initialize. Stopping tests.")
            return

        # Test PostgreSQL
        print("\n🐘 Testing PostgreSQL Connection...")
        self.test_list_schemas("primary")
        self.test_list_tables("primary", "public")
        self.test_run_sql("primary", "SELECT current_database(), version()", 1)
        self.test_run_sql(
            "primary",
            "SELECT schemaname, tablename FROM pg_tables WHERE schemaname = 'public'",
            5,
        )
        self.test_explain_sql("primary", "SELECT * FROM pg_tables LIMIT 5")

        # Test ClickHouse
        print("\n🏠 Testing ClickHouse Connection...")
        self.test_list_schemas("analytics")
        self.test_list_tables("analytics", "default")
        self.test_describe_table("analytics", "default", "events")
        self.test_run_sql("analytics", "SELECT count(*) as total_events FROM events", 1)
        self.test_run_sql(
            "analytics",
            "SELECT event_type, count(*) as count FROM events GROUP BY event_type",
            10,
        )
        self.test_explain_sql(
            "analytics", "SELECT event_type, count(*) FROM events GROUP BY event_type"
        )

        print("\n✅ Comprehensive test completed!")

    def interactive_mode(self):
        """Run in interactive mode"""
        print("🎯 Interactive MCP Database Server Tester")
        print("=" * 50)

        if not self.initialize():
            print("❌ Failed to initialize. Exiting.")
            return

        while True:
            print("\nAvailable commands:")
            print("1. List schemas (PostgreSQL)")
            print("2. List schemas (ClickHouse)")
            print("3. List tables")
            print("4. Describe table")
            print("5. Run SQL query")
            print("6. Explain SQL query")
            print("7. Run comprehensive test")
            print("8. Exit")

            choice = input("\nEnter your choice (1-8): ").strip()

            if choice == "1":
                self.test_list_schemas("primary")
            elif choice == "2":
                self.test_list_schemas("analytics")
            elif choice == "3":
                connection = input("Enter connection (primary/analytics): ").strip()
                schema = input("Enter schema name: ").strip()
                self.test_list_tables(connection, schema)
            elif choice == "4":
                connection = input("Enter connection (primary/analytics): ").strip()
                schema = input("Enter schema name: ").strip()
                table = input("Enter table name: ").strip()
                self.test_describe_table(connection, schema, table)
            elif choice == "5":
                connection = input("Enter connection (primary/analytics): ").strip()
                query = input("Enter SQL query: ").strip()
                limit = input("Enter limit (default 10): ").strip()
                limit = int(limit) if limit else 10
                self.test_run_sql(connection, query, limit)
            elif choice == "6":
                connection = input("Enter connection (primary/analytics): ").strip()
                query = input("Enter SQL query: ").strip()
                self.test_explain_sql(connection, query)
            elif choice == "7":
                self.run_comprehensive_test()
            elif choice == "8":
                break
            else:
                print("❌ Invalid choice. Please try again.")

    def cleanup(self):
        """Clean up resources"""
        if self.process:
            print("🧹 Stopping server...")
            self.process.terminate()
            try:
                self.process.wait(timeout=3)  # Wait max 3 seconds
            except subprocess.TimeoutExpired:
                print("🔪 Force killing server...")
                self.process.kill()
                self.process.wait()
            print("✅ Server stopped")


def main():
    if len(sys.argv) > 1 and sys.argv[1] == "--comprehensive":
        mode = "comprehensive"
    else:
        mode = "interactive"

    # Check if server binary exists
    if not os.path.exists("./bin/database-mcp"):
        print("❌ Server binary not found. Please build first:")
        print("   make build")
        sys.exit(1)

    # Check if databases are running
    try:
        result = subprocess.run(["docker", "ps"], capture_output=True, text=True)
        if (
            "database-mcp-postgres" not in result.stdout
            or "database-mcp-clickhouse" not in result.stdout
        ):
            print("❌ Database containers not running. Starting them...")
            subprocess.run(
                ["docker", "compose", "up", "-d", "postgres", "clickhouse"], check=True
            )
            print("⏱️  Waiting for databases to be ready...")
            time.sleep(15)
    except Exception as e:
        print(f"❌ Error checking/starting databases: {e}")
        sys.exit(1)

    tester = MCPTester()

    try:
        if not tester.start_server():
            sys.exit(1)

        if mode == "comprehensive":
            tester.run_comprehensive_test()
        else:
            tester.interactive_mode()

    except KeyboardInterrupt:
        print("\n👋 Interrupted by user")
    finally:
        tester.cleanup()


if __name__ == "__main__":
    main()
