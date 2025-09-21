package mcp

import (
	"context"
	"encoding/json"
)

// MCP Protocol Types and Interfaces

// MCPRequest represents an MCP JSON-RPC request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP JSON-RPC error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard MCP error codes
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
)

// MCP Protocol Methods
const (
	MethodInitialize    = "initialize"
	MethodListTools     = "tools/list"
	MethodCallTool      = "tools/call"
	MethodListSchemas   = "list_schemas"
	MethodListTables    = "list_tables"
	MethodDescribeTable = "describe_table"
	MethodRunSQL        = "run_sql"
	MethodExplainSQL    = "explain_sql"
)

// InitializeParams represents the parameters for the initialize method
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    ClientCapabilities     `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
	Extra           map[string]interface{} `json:"extra,omitempty"`
}

// InitializeResult represents the result of the initialize method
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ClientCapabilities represents what the client can do
type ClientCapabilities struct {
	Tools *ClientToolsCapabilities `json:"tools,omitempty"`
}

// ClientToolsCapabilities represents client tool capabilities
type ClientToolsCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerCapabilities represents what the server can do
type ServerCapabilities struct {
	Tools *ServerToolsCapabilities `json:"tools,omitempty"`
}

// ServerToolsCapabilities represents server tool capabilities
type ServerToolsCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ClientInfo represents information about the client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo represents information about the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// ToolListResult represents the result of tools/list
type ToolListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolCallParams represents the parameters for tools/call
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResult represents the result of tools/call
type ToolCallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem represents a piece of content in a tool result
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Note: Database-specific types moved to internal/types package to avoid import cycles

// Handler interface for MCP methods
type Handler interface {
	Initialize(ctx context.Context, params InitializeParams) (*InitializeResult, error)
	ListTools(ctx context.Context) (*ToolListResult, error)
	CallTool(ctx context.Context, params ToolCallParams) (*ToolCallResult, error)
}
