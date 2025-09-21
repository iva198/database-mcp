package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// Tool handler methods for the MCP server

// handleListSchemas handles the list_schemas tool call
func (s *Server) handleListSchemas(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
	database := getStringArg(args, "database", "primary")

	startTime := time.Now()
	schemas, err := s.dbManager.ListSchemas(ctx, database)
	duration := time.Since(startTime)

	if err != nil {
		slog.Error("Failed to list schemas", "database", database, "error", err)
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Error listing schemas from %s database: %v", database, err)},
			},
			IsError: true,
		}, nil
	}

	// Format result as JSON
	result := map[string]interface{}{
		"database":        database,
		"schemas":         schemas,
		"count":           len(schemas),
		"executionTimeMs": duration.Milliseconds(),
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Error formatting result: %v", err)},
			},
			IsError: true,
		}, nil
	}

	slog.Info("Listed schemas", "database", database, "count", len(schemas), "duration_ms", duration.Milliseconds())

	return &ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: string(jsonResult)},
		},
	}, nil
}

// handleListTables handles the list_tables tool call
func (s *Server) handleListTables(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
	database := getStringArg(args, "database", "primary")
	schema := getStringArg(args, "schema", "")

	if schema == "" {
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "Error: schema parameter is required"},
			},
			IsError: true,
		}, nil
	}

	startTime := time.Now()
	tables, err := s.dbManager.ListTables(ctx, database, schema)
	duration := time.Since(startTime)

	if err != nil {
		slog.Error("Failed to list tables", "database", database, "schema", schema, "error", err)
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Error listing tables from %s.%s: %v", database, schema, err)},
			},
			IsError: true,
		}, nil
	}

	// Format result as JSON
	result := map[string]interface{}{
		"database":        database,
		"schema":          schema,
		"tables":          tables,
		"count":           len(tables),
		"executionTimeMs": duration.Milliseconds(),
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Error formatting result: %v", err)},
			},
			IsError: true,
		}, nil
	}

	slog.Info("Listed tables", "database", database, "schema", schema, "count", len(tables), "duration_ms", duration.Milliseconds())

	return &ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: string(jsonResult)},
		},
	}, nil
}

// handleDescribeTable handles the describe_table tool call
func (s *Server) handleDescribeTable(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
	database := getStringArg(args, "database", "primary")
	schema := getStringArg(args, "schema", "")
	table := getStringArg(args, "table", "")

	if schema == "" || table == "" {
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "Error: schema and table parameters are required"},
			},
			IsError: true,
		}, nil
	}

	startTime := time.Now()
	description, err := s.dbManager.DescribeTable(ctx, database, schema, table)
	duration := time.Since(startTime)

	if err != nil {
		slog.Error("Failed to describe table", "database", database, "schema", schema, "table", table, "error", err)
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Error describing table %s.%s.%s: %v", database, schema, table, err)},
			},
			IsError: true,
		}, nil
	}

	// Format result as JSON
	result := map[string]interface{}{
		"database":        database,
		"table":           description,
		"executionTimeMs": duration.Milliseconds(),
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Error formatting result: %v", err)},
			},
			IsError: true,
		}, nil
	}

	slog.Info("Described table", "database", database, "schema", schema, "table", table, "columns", len(description.Columns), "duration_ms", duration.Milliseconds())

	return &ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: string(jsonResult)},
		},
	}, nil
}

// handleRunSQL handles the run_sql tool call
func (s *Server) handleRunSQL(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
	database := getStringArg(args, "database", "primary")
	query := getStringArg(args, "query", "")
	limit := getIntArg(args, "limit", 1000)

	if query == "" {
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "Error: query parameter is required"},
			},
			IsError: true,
		}, nil
	}

	// Enforce max limit from config
	if limit > s.config.MaxRows {
		limit = s.config.MaxRows
	}

	// TODO: Add safety layer validation here (Phase 3)
	// For now, we'll just log the query attempt
	slog.Info("Executing SQL query", "database", database, "query_length", len(query), "limit", limit)

	startTime := time.Now()
	result, err := s.dbManager.RunSQL(ctx, database, query, limit)
	duration := time.Since(startTime)

	if err != nil {
		slog.Error("Failed to execute SQL", "database", database, "error", err)
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Error executing SQL on %s database: %v", database, err)},
			},
			IsError: true,
		}, nil
	}

	// Format result as JSON
	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Error formatting result: %v", err)},
			},
			IsError: true,
		}, nil
	}

	slog.Info("Executed SQL query", "database", database, "rows", result.RowCount, "duration_ms", duration.Milliseconds())

	return &ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: string(jsonResult)},
		},
	}, nil
}

// handleExplainSQL handles the explain_sql tool call
func (s *Server) handleExplainSQL(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
	database := getStringArg(args, "database", "primary")
	query := getStringArg(args, "query", "")

	if query == "" {
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "Error: query parameter is required"},
			},
			IsError: true,
		}, nil
	}

	startTime := time.Now()
	result, err := s.dbManager.ExplainQuery(ctx, database, query)
	duration := time.Since(startTime)

	if err != nil {
		slog.Error("Failed to explain SQL", "database", database, "error", err)
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Error explaining SQL on %s database: %v", database, err)},
			},
			IsError: true,
		}, nil
	}

	// Format result as JSON
	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Error formatting result: %v", err)},
			},
			IsError: true,
		}, nil
	}

	slog.Info("Explained SQL query", "database", database, "duration_ms", duration.Milliseconds())

	return &ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: string(jsonResult)},
		},
	}, nil
}
