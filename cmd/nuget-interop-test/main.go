// Package main implements a JSON-RPC bridge for gonuget interop testing.
// It receives requests via stdin and returns responses via stdout.
// This enables C# test projects to validate gonuget against NuGet.Client.
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Request represents an incoming test request from C# tests.
// Action specifies which gonuget operation to perform.
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
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo contains structured error information for debugging.
// Code is a machine-readable error code (e.g., "SIGN_001").
// Message is a human-readable error description.
// Details contains additional context (e.g., file paths, stack traces).
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func main() {
	// Disable log output to avoid contaminating JSON response
	// (tests may need to enable logging to files for debugging)

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
	// Signature operations
	case "sign_package":
		handler = &SignPackageHandler{}
	case "parse_signature":
		handler = &ParseSignatureHandler{}
	case "verify_signature":
		handler = &VerifySignatureHandler{}

	// Version operations
	case "compare_versions":
		handler = &CompareVersionsHandler{}
	case "parse_version":
		handler = &ParseVersionHandler{}

	// Framework operations
	case "check_framework_compat":
		handler = &CheckFrameworkCompatHandler{}
	case "parse_framework":
		handler = &ParseFrameworkHandler{}
	case "format_framework":
		handler = &FormatFrameworkHandler{}

	// Package operations
	case "read_package":
		handler = &ReadPackageHandler{}
	case "build_package":
		handler = &BuildPackageHandler{}
	case "extract_package_v2":
		handler = &ExtractPackageV2Handler{}
	case "install_from_source_v3":
		handler = &InstallFromSourceV3Handler{}

	// Asset selection operations
	case "find_runtime_assemblies":
		handler = &FindRuntimeAssembliesHandler{}
	case "find_compile_assemblies":
		handler = &FindCompileAssembliesHandler{}
	case "parse_asset_path":
		handler = &ParseAssetPathHandler{}

	// RID (Runtime Identifier) operations
	case "expand_runtime":
		handler = &ExpandRuntimeHandler{}
	case "are_runtimes_compatible":
		handler = &AreRuntimesCompatibleHandler{}

	// Cache operations
	case "compute_cache_hash":
		handler = &ComputeCacheHashHandler{}
	case "sanitize_cache_filename":
		handler = &SanitizeCacheFilenameHandler{}
	case "generate_cache_paths":
		handler = &GenerateCachePathsHandler{}
	case "validate_cache_file":
		handler = &ValidateCacheFileHandler{}
	case "calculate_dgspec_hash":
		handler = &CalculateDgSpecHashHandler{}
	case "verify_project_cache_file":
		handler = &VerifyProjectCacheFileHandler{}

	// Resolver operations
	case "walk_graph":
		handler = &WalkGraphHandler{}
	case "resolve_conflicts":
		handler = &ResolveConflictsHandler{}
	case "analyze_cycles":
		handler = &AnalyzeCyclesHandler{}
	case "resolve_transitive":
		handler = &ResolveTransitiveHandler{}
	case "benchmark_cache":
		handler = &BenchmarkCacheHandler{}
	case "resolve_with_ttl":
		handler = &ResolveWithTTLHandler{}
	case "benchmark_parallel":
		handler = &BenchmarkParallelHandler{}
	case "resolve_with_worker_limit":
		handler = &ResolveWithWorkerLimitHandler{}

	// Restore operations
	case "resolve_latest_version":
		handler = &ResolveLatestVersionHandler{}
	case "parse_lock_file":
		handler = &ParseLockFileHandler{}
	case "restore_direct_dependencies":
		handler = &RestoreDirectDependenciesHandler{}
	case "restore_transitive":
		handler = &RestoreTransitiveHandler{}
	case "compare_project_assets":
		handler = &CompareProjectAssetsHandler{}
	case "validate_error_messages":
		handler = &ValidateErrorMessagesHandler{}

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
	Handle(data json.RawMessage) (interface{}, error)
	ErrorCode() string
}

// sendSuccess writes a successful response to stdout.
func sendSuccess(data interface{}) {
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
