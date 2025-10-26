// Package main implements a JSON-RPC bridge for gonuget CLI interop testing.
// It executes both dotnet nuget and gonuget commands and returns comparison results.
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Request represents an incoming CLI test request from C# tests.
// Action specifies which command comparison to perform.
// Data contains action-specific parameters in JSON format.
type Request struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

// Response represents the standard response format sent back to C#.
// Success indicates whether the operation completed without errors.
// Data contains action-specific results (only present on success).
// Error contains detailed error information (only present on failure).
type Response struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrorInfo `json:"error,omitempty"`
}

// ErrorInfo contains structured error information for debugging.
// Code is a machine-readable error code (e.g., "CLI_001").
// Message is a human-readable error description.
// Details contains additional context (e.g., file paths, stderr output).
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func main() {
	// Read request from stdin
	var req Request
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&req); err != nil {
		sendError("REQ_001", "Failed to parse request JSON", err.Error())
		os.Exit(1)
	}

	// Route to appropriate handler based on action
	var handler Handler
	switch req.Action {
	// Generic command execution
	case "execute_command_pair":
		handler = &ExecuteCommandPairHandler{}

	// Config command actions
	case "execute_config_get":
		handler = &ExecuteConfigGetHandler{}
	case "execute_config_set":
		handler = &ExecuteConfigSetHandler{}
	case "execute_config_unset":
		handler = &ExecuteConfigUnsetHandler{}
	case "execute_config_paths":
		handler = &ExecuteConfigPathsHandler{}

	// Version command action
	case "execute_version":
		handler = &ExecuteVersionHandler{}

	// Source command actions
	case "execute_source_list":
		handler = &ExecuteSourceListHandler{}
	case "execute_source_add":
		handler = &ExecuteSourceAddHandler{}
	case "execute_source_remove":
		handler = &ExecuteSourceRemoveHandler{}
	case "execute_source_enable":
		handler = &ExecuteSourceEnableHandler{}
	case "execute_source_disable":
		handler = &ExecuteSourceDisableHandler{}
	case "execute_source_update":
		handler = &ExecuteSourceUpdateHandler{}

	// Help command action
	case "execute_help":
		handler = &ExecuteHelpHandler{}

	default:
		sendError("ACT_001", "Unknown action", fmt.Sprintf("action=%s", req.Action))
		os.Exit(1)
	}

	// Execute handler
	result, err := handler.Handle(req.Data)
	if err != nil {
		sendError(handler.ErrorCode(), err.Error(), "")
		os.Exit(1)
	}

	// Send success response
	sendSuccess(result)
}

// Handler interface for all request handlers.
// Handle processes the request data and returns a result or error.
// ErrorCode returns the error code prefix for this handler.
type Handler interface {
	Handle(data json.RawMessage) (any, error)
	ErrorCode() string
}

// sendSuccess writes a successful response to stdout.
func sendSuccess(data any) {
	resp := Response{
		Success: true,
		Data:    data,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ") // Pretty print for debugging
	_ = encoder.Encode(resp)
}

// sendError writes an error response to stdout.
func sendError(code, message, details string) {
	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ") // Pretty print for debugging
	_ = encoder.Encode(resp)
}
