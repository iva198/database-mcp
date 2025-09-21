package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"database-mcp/internal/db"
)

// Server implements the MCP Handler interface
type Server struct {
	serverInfo ServerInfo
	dbManager  *db.Manager
	config     *Config
}

// Config holds server configuration
type Config struct {
	ReadOnly       bool
	MaxRows        int
	QueryTimeoutMs int
	PrimaryDBURL   string
	AnalyticsDBURL string
	TransportMode  string
}

// NewServer creates a new MCP server
func NewServer() (*Server, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	dbManager, err := db.NewManager(config.PrimaryDBURL, config.AnalyticsDBURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create database manager: %w", err)
	}

	server := &Server{
		serverInfo: ServerInfo{
			Name:    "Database MCP Server",
			Version: getVersion(),
		},
		dbManager: dbManager,
		config:    config,
	}

	return server, nil
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	slog.Info("Starting MCP server",
		"version", s.serverInfo.Version,
		"transport", s.config.TransportMode,
		"read_only", s.config.ReadOnly)

	// Initialize database connections
	if err := s.dbManager.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to databases: %w", err)
	}
	defer s.dbManager.Close()

	// Create transport based on mode
	var transport Transport
	switch s.config.TransportMode {
	case "stdio":
		transport = NewStdioTransport(s)
	case "http":
		// TODO: Implement HTTP transport in Phase 1.5
		return fmt.Errorf("HTTP transport not yet implemented")
	default:
		return fmt.Errorf("unsupported transport mode: %s", s.config.TransportMode)
	}

	// Start the transport
	return transport.Start(ctx)
}

// Initialize implements the MCP initialize method
func (s *Server) Initialize(ctx context.Context, params InitializeParams) (*InitializeResult, error) {
	slog.Info("Client initializing",
		"client", params.ClientInfo.Name,
		"version", params.ClientInfo.Version,
		"protocol", params.ProtocolVersion)

	// Validate protocol version
	if params.ProtocolVersion != "2024-11-05" {
		slog.Warn("Unsupported protocol version", "version", params.ProtocolVersion)
	}

	result := &InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &ServerToolsCapabilities{
				ListChanged: false, // We don't support dynamic tool changes yet
			},
		},
		ServerInfo: s.serverInfo,
	}

	return result, nil
}

// ListTools implements the tools/list method
func (s *Server) ListTools(ctx context.Context) (*ToolListResult, error) {
	tools := []Tool{
		{
			Name:        "list_schemas",
			Description: "List all available database schemas",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"database": map[string]interface{}{
						"type":        "string",
						"description": "Database to query (primary or analytics)",
						"enum":        []string{"primary", "analytics"},
						"default":     "primary",
					},
				},
			},
		},
		{
			Name:        "list_tables",
			Description: "List tables in a schema with metadata",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"database": map[string]interface{}{
						"type":        "string",
						"description": "Database to query (primary or analytics)",
						"enum":        []string{"primary", "analytics"},
						"default":     "primary",
					},
					"schema": map[string]interface{}{
						"type":        "string",
						"description": "Schema name to list tables from",
					},
				},
				"required": []string{"schema"},
			},
		},
		{
			Name:        "describe_table",
			Description: "Get detailed table structure and constraints",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"database": map[string]interface{}{
						"type":        "string",
						"description": "Database to query (primary or analytics)",
						"enum":        []string{"primary", "analytics"},
						"default":     "primary",
					},
					"schema": map[string]interface{}{
						"type":        "string",
						"description": "Schema name",
					},
					"table": map[string]interface{}{
						"type":        "string",
						"description": "Table name",
					},
				},
				"required": []string{"schema", "table"},
			},
		},
		{
			Name:        "run_sql",
			Description: "Execute SQL queries with safety checks",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"database": map[string]interface{}{
						"type":        "string",
						"description": "Database to query (primary or analytics)",
						"enum":        []string{"primary", "analytics"},
						"default":     "primary",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "SQL query to execute",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of rows to return",
						"minimum":     1,
						"maximum":     s.config.MaxRows,
						"default":     1000,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "explain_sql",
			Description: "Get query execution plans",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"database": map[string]interface{}{
						"type":        "string",
						"description": "Database to query (primary or analytics)",
						"enum":        []string{"primary", "analytics"},
						"default":     "primary",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "SQL query to explain",
					},
				},
				"required": []string{"query"},
			},
		},
	}

	return &ToolListResult{Tools: tools}, nil
}

// CallTool implements the tools/call method
func (s *Server) CallTool(ctx context.Context, params ToolCallParams) (*ToolCallResult, error) {
	slog.Debug("Tool call", "name", params.Name, "args", params.Arguments)

	// Set query timeout
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.QueryTimeoutMs)*time.Millisecond)
	defer cancel()

	switch params.Name {
	case "list_schemas":
		return s.handleListSchemas(queryCtx, params.Arguments)
	case "list_tables":
		return s.handleListTables(queryCtx, params.Arguments)
	case "describe_table":
		return s.handleDescribeTable(queryCtx, params.Arguments)
	case "run_sql":
		return s.handleRunSQL(queryCtx, params.Arguments)
	case "explain_sql":
		return s.handleExplainSQL(queryCtx, params.Arguments)
	default:
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Unknown tool: %s", params.Name)},
			},
			IsError: true,
		}, nil
	}
}

// Helper function to get string argument with default
func getStringArg(args map[string]interface{}, key, defaultValue string) string {
	if val, ok := args[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// Helper function to get int argument with default
func getIntArg(args map[string]interface{}, key string, defaultValue int) int {
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return defaultValue
}

// loadConfig loads configuration from environment variables
func loadConfig() (*Config, error) {
	config := &Config{
		ReadOnly:       getEnvBool("READ_ONLY", true),
		MaxRows:        getEnvInt("MAX_ROWS", 10000),
		QueryTimeoutMs: getEnvInt("QUERY_TIMEOUT_MS", 30000),
		PrimaryDBURL:   os.Getenv("DB_PRIMARY_URL"),
		AnalyticsDBURL: os.Getenv("DB_ANALYTICS_URL"),
		TransportMode:  getEnvString("MCP_MODE", "stdio"),
	}

	// Validate required config
	if config.PrimaryDBURL == "" {
		return nil, fmt.Errorf("DB_PRIMARY_URL is required")
	}

	return config, nil
}

// Helper functions for environment variables
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

// getVersion returns the version, trying to get it from build-time variables
func getVersion() string {
	// This will be set by build flags in main.go
	// For now, return a default
	return "0.1.0-dev"
}
