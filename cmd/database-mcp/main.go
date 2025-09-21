package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"database-mcp/internal/mcp"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var (
		versionFlag = flag.Bool("version", false, "show version information")
		helpFlag    = flag.Bool("help", false, "show help")
	)
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Database MCP Server %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	if *helpFlag {
		fmt.Println("Database MCP Server - A secure database interface for AI assistants")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  database-mcp [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Environment Variables:")
		fmt.Println("  DB_PRIMARY_URL      PostgreSQL connection string (required)")
		fmt.Println("  DB_ANALYTICS_URL    ClickHouse connection string (optional)")
		fmt.Println("  MCP_MODE           Transport mode: stdio or http (default: stdio)")
		fmt.Println("  READ_ONLY          Enable read-only mode (default: true)")
		fmt.Println("  MAX_ROWS           Maximum rows per query (default: 10000)")
		fmt.Println("  QUERY_TIMEOUT_MS   Query timeout in milliseconds (default: 30000)")
		fmt.Println("  LOG_LEVEL          Log level: debug, info, warn, error (default: info)")
		os.Exit(0)
	}

	// Set up structured logging
	logLevel := getEnvWithDefault("LOG_LEVEL", "info")
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Database MCP Server",
		"version", version,
		"commit", commit,
		"build_date", date)

	// Create context that cancels on interrupt
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize and start the MCP server
	server, err := mcp.NewServer()
	if err != nil {
		slog.Error("Failed to create MCP server", "error", err)
		os.Exit(1)
	}

	if err := server.Start(ctx); err != nil {
		slog.Error("MCP server failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Database MCP Server stopped")
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
