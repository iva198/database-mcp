package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Transport interface for different MCP transport methods
type Transport interface {
	Start(ctx context.Context) error
	Stop() error
}

// StdioTransport implements MCP over stdio (standard input/output)
type StdioTransport struct {
	handler Handler
	reader  *bufio.Reader
	writer  io.Writer
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(handler Handler) *StdioTransport {
	return &StdioTransport{
		handler: handler,
		reader:  bufio.NewReader(os.Stdin),
		writer:  os.Stdout,
	}
}

// Start begins processing MCP requests over stdio
func (t *StdioTransport) Start(ctx context.Context) error {
	slog.Info("Starting MCP stdio transport")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping MCP stdio transport")
			return ctx.Err()
		default:
			// Read and process one request
			if err := t.processRequest(ctx); err != nil {
				if err == io.EOF {
					slog.Info("EOF received, stopping transport")
					return nil
				}
				slog.Error("Error processing request", "error", err)
				// Continue processing other requests
			}
		}
	}
}

// Stop stops the transport
func (t *StdioTransport) Stop() error {
	// Stdio transport doesn't need explicit cleanup
	return nil
}

// processRequest reads and processes a single JSON-RPC request
func (t *StdioTransport) processRequest(ctx context.Context) error {
	// Read line from stdin
	line, err := t.reader.ReadString('\n')
	if err != nil {
		return err
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return nil // Skip empty lines
	}

	slog.Debug("Received request", "raw", line)

	// Parse JSON-RPC request
	var req MCPRequest
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		// Send parse error response
		response := MCPResponse{
			JSONRPC: "2.0",
			Error: &MCPError{
				Code:    ErrorCodeParseError,
				Message: "Parse error",
				Data:    err.Error(),
			},
		}
		return t.sendResponse(response)
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		response := MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    ErrorCodeInvalidRequest,
				Message: "Invalid JSON-RPC version",
			},
		}
		return t.sendResponse(response)
	}

	// Process the request
	response := t.handleRequest(ctx, req)
	return t.sendResponse(response)
}

// handleRequest processes an MCP request and returns a response
func (t *StdioTransport) handleRequest(ctx context.Context, req MCPRequest) MCPResponse {
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	slog.Debug("Processing request", "method", req.Method, "id", req.ID)

	switch req.Method {
	case MethodInitialize:
		var params InitializeParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			response.Error = &MCPError{
				Code:    ErrorCodeInvalidParams,
				Message: "Invalid parameters for initialize",
				Data:    err.Error(),
			}
			return response
		}

		result, err := t.handler.Initialize(ctx, params)
		if err != nil {
			response.Error = &MCPError{
				Code:    ErrorCodeInternalError,
				Message: "Initialize failed",
				Data:    err.Error(),
			}
		} else {
			response.Result = result
		}

	case MethodListTools:
		result, err := t.handler.ListTools(ctx)
		if err != nil {
			response.Error = &MCPError{
				Code:    ErrorCodeInternalError,
				Message: "List tools failed",
				Data:    err.Error(),
			}
		} else {
			response.Result = result
		}

	case MethodCallTool:
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			response.Error = &MCPError{
				Code:    ErrorCodeInvalidParams,
				Message: "Invalid parameters for tool call",
				Data:    err.Error(),
			}
			return response
		}

		result, err := t.handler.CallTool(ctx, params)
		if err != nil {
			response.Error = &MCPError{
				Code:    ErrorCodeInternalError,
				Message: "Tool call failed",
				Data:    err.Error(),
			}
		} else {
			response.Result = result
		}

	default:
		response.Error = &MCPError{
			Code:    ErrorCodeMethodNotFound,
			Message: fmt.Sprintf("Method '%s' not found", req.Method),
		}
	}

	return response
}

// sendResponse sends a JSON-RPC response to stdout
func (t *StdioTransport) sendResponse(response MCPResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		slog.Error("Failed to marshal response", "error", err)
		return err
	}

	slog.Debug("Sending response", "raw", string(data))

	// Write response followed by newline
	_, err = fmt.Fprintf(t.writer, "%s\n", data)
	if err != nil {
		slog.Error("Failed to write response", "error", err)
		return err
	}

	return nil
}
