# CLI Interoperability Testing Guide

**Version**: 1.0
**Last Updated**: 2025-01-25
**Target**: `dotnet nuget` (cross-platform) validation

---

## Overview

This guide describes the CLI interoperability testing system for gonuget, which validates that `gonuget` CLI commands produce identical behavior to Microsoft's official `dotnet nuget` implementation.

**Architecture Pattern**: Follows the same JSON-RPC bridge pattern as library interop tests (`cmd/nuget-interop-test`), but adapted for CLI command execution and comparison.

---

## Architecture

The CLI interop test system uses a **JSON-RPC command execution bridge**:

```
┌──────────────────────────────────────────────────────────────┐
│  xUnit Test (C#)                                             │
│  ┌───────────────────────────────────────────────────────┐   │
│  │ // Execute via bridge and compare                     │   │
│  │ var result = GonugetCliBridge.Execute(new {           │   │
│  │     action = "execute_command_pair",                  │   │
│  │     dotnetCommand = "list source",                    │   │
│  │     gonugetCommand = "sources list",                  │   │
│  │     workingDir = "/tmp/test",                         │   │
│  │     configFile = "NuGet.config"                       │   │
│  │ });                                                   │   │
│  │                                                       │   │
│  │ Assert.Equal(result.DotnetExitCode, result.Gonug...); │   │
│  │ Assert.Equal(result.DotnetStdOut, result.GonugetS...);│   │
│  └─────────────────────────────────────────────────────┘     │
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐     │
│  │ GonugetCliBridge.cs                                 │     │
│  │ • Spawns gonuget-cli-interop-test process           │     │
│  │ • Sends JSON request via stdin                      │     │
│  │ • Receives JSON response via stdout                 │     │
│  └─────────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  gonuget-cli-interop-test (Go)                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ • Reads JSON request from stdin                     │    │
│  │ • Executes `dotnet nuget <command>` subprocess      │    │
│  │ • Executes `gonuget <command>` subprocess           │    │
│  │ • Captures stdout, stderr, exit code from both      │    │
│  │ • Returns JSON response via stdout                  │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

**Key Differences from Library Interop**:
- Library interop: Invokes Go library functions directly
- CLI interop: Executes CLI commands as subprocesses and compares outputs

---

## Go Binary: `cmd/gonuget-cli-interop-test`

### File Structure

```
cmd/gonuget-cli-interop-test/
├── main.go                     # Entry point, request router
├── protocol.go                 # Request/response types
├── handlers_sources.go         # Sources command handlers
├── handlers_config.go          # Config command handlers
├── handlers_version.go         # Version command handler
├── helpers.go                  # Shared utilities (output normalization)
└── executor.go                 # Command execution wrapper
```

---

### main.go

```go
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
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
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
	// Command execution actions
	case "execute_command_pair":
		handler = &ExecuteCommandPairHandler{}
	case "execute_sources_list":
		handler = &ExecuteSourcesListHandler{}
	case "execute_sources_add":
		handler = &ExecuteSourcesAddHandler{}
	case "execute_sources_remove":
		handler = &ExecuteSourcesRemoveHandler{}
	case "execute_sources_enable":
		handler = &ExecuteSourcesEnableHandler{}
	case "execute_sources_disable":
		handler = &ExecuteSourcesDisableHandler{}
	case "execute_sources_update":
		handler = &ExecuteSourcesUpdateHandler{}
	case "execute_config_get":
		handler = &ExecuteConfigGetHandler{}
	case "execute_config_set":
		handler = &ExecuteConfigSetHandler{}
	case "execute_config_list":
		handler = &ExecuteConfigListHandler{}
	case "execute_version":
		handler = &ExecuteVersionHandler{}

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
```

---

### protocol.go

```go
package main

// ExecuteCommandPairRequest executes both dotnet nuget and gonuget commands
type ExecuteCommandPairRequest struct {
	DotnetCommand  string `json:"dotnetCommand"`  // e.g., "list source"
	GonugetCommand string `json:"gonugetCommand"` // e.g., "sources list"
	WorkingDir     string `json:"workingDir"`
	ConfigFile     string `json:"configFile,omitempty"`
	Timeout        int    `json:"timeout,omitempty"` // seconds, default 30
}

// ExecuteCommandPairResponse contains execution results from both commands
type ExecuteCommandPairResponse struct {
	// dotnet nuget results
	DotnetExitCode int    `json:"dotnetExitCode"`
	DotnetStdOut   string `json:"dotnetStdOut"`
	DotnetStdErr   string `json:"dotnetStdErr"`
	DotnetSuccess  bool   `json:"dotnetSuccess"`

	// gonuget results
	GonugetExitCode int    `json:"gonugetExitCode"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	GonugetSuccess  bool   `json:"gonugetSuccess"`

	// Normalized comparison
	NormalizedDotnetStdOut  string `json:"normalizedDotnetStdOut"`
	NormalizedGonugetStdOut string `json:"normalizedGonugetStdOut"`
	OutputMatches           bool   `json:"outputMatches"`
}

// ExecuteSourcesListRequest for sources list command
type ExecuteSourcesListRequest struct {
	ConfigFile string `json:"configFile,omitempty"`
	Format     string `json:"format,omitempty"` // "Detailed" or "Short"
	WorkingDir string `json:"workingDir"`
}

// ExecuteSourcesListResponse contains parsed sources list
type ExecuteSourcesListResponse struct {
	DotnetExitCode  int              `json:"dotnetExitCode"`
	GonugetExitCode int              `json:"gonugetExitCode"`
	DotnetSources   []PackageSource  `json:"dotnetSources"`
	GonugetSources  []PackageSource  `json:"gonugetSources"`
	SourcesMatch    bool             `json:"sourcesMatch"`
}

// PackageSource represents a parsed package source
type PackageSource struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// ExecuteSourcesAddRequest for sources add command
type ExecuteSourcesAddRequest struct {
	Name       string `json:"name"`
	Source     string `json:"source"`
	ConfigFile string `json:"configFile,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	WorkingDir string `json:"workingDir"`
}

// ExecuteConfigGetRequest for config get command
type ExecuteConfigGetRequest struct {
	Key        string `json:"key"`
	ConfigFile string `json:"configFile,omitempty"`
	WorkingDir string `json:"workingDir"`
}

// ExecuteConfigGetResponse contains config value
type ExecuteConfigGetResponse struct {
	DotnetExitCode  int    `json:"dotnetExitCode"`
	GonugetExitCode int    `json:"gonugetExitCode"`
	DotnetValue     string `json:"dotnetValue"`
	GonugetValue    string `json:"gonugetValue"`
	ValuesMatch     bool   `json:"valuesMatch"`
}
```

---

### executor.go

```go
package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// CommandResult holds execution results from a single command
type CommandResult struct {
	ExitCode int
	StdOut   string
	StdErr   string
	Success  bool
}

// ExecuteCommand executes a command and captures output
func ExecuteCommand(executable string, args []string, workingDir string, timeout int) (*CommandResult, error) {
	if timeout == 0 {
		timeout = 30 // default 30 seconds
	}

	cmd := exec.Command(executable, args...)
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return nil, fmt.Errorf("command failed: %w", err)
			}
		}

		return &CommandResult{
			ExitCode: exitCode,
			StdOut:   stdout.String(),
			StdErr:   stderr.String(),
			Success:  exitCode == 0,
		}, nil

	case <-time.After(time.Duration(timeout) * time.Second):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("command timed out after %d seconds", timeout)
	}
}

// ExecuteDotnetNuget executes a dotnet nuget command
func ExecuteDotnetNuget(command string, workingDir string, configFile string, timeout int) (*CommandResult, error) {
	dotnetExe := "dotnet"
	if runtime.GOOS == "windows" {
		dotnetExe = "dotnet.exe"
	}

	args := []string{"nuget"}
	args = append(args, strings.Fields(command)...)

	if configFile != "" {
		args = append(args, "--configfile", configFile)
	}

	return ExecuteCommand(dotnetExe, args, workingDir, timeout)
}

// ExecuteGonuget executes a gonuget command
func ExecuteGonuget(command string, workingDir string, configFile string, timeout int) (*CommandResult, error) {
	gonugetExe := findGonugetExecutable()

	args := strings.Fields(command)

	if configFile != "" {
		args = append(args, "--configfile", configFile)
	}

	return ExecuteCommand(gonugetExe, args, workingDir, timeout)
}

// findGonugetExecutable locates the gonuget binary
func findGonugetExecutable() string {
	exeName := "gonuget"
	if runtime.GOOS == "windows" {
		exeName = "gonuget.exe"
	}

	// Check current directory
	if _, err := exec.LookPath("./" + exeName); err == nil {
		return "./" + exeName
	}

	// Check parent directories (repository root)
	for i := 0; i < 5; i++ {
		prefix := strings.Repeat("../", i)
		path := prefix + exeName
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}

	// Check PATH
	if path, err := exec.LookPath(exeName); err == nil {
		return path
	}

	// Default to just the name (will fail if not in PATH)
	return exeName
}
```

---

### handlers_sources.go

```go
package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ExecuteCommandPairHandler executes both dotnet nuget and gonuget commands
type ExecuteCommandPairHandler struct{}

func (h *ExecuteCommandPairHandler) ErrorCode() string { return "CLI_EXEC_001" }

func (h *ExecuteCommandPairHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ExecuteCommandPairRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Execute dotnet nuget command
	dotnetResult, err := ExecuteDotnetNuget(req.DotnetCommand, req.WorkingDir, req.ConfigFile, req.Timeout)
	if err != nil {
		return nil, fmt.Errorf("execute dotnet nuget: %w", err)
	}

	// Execute gonuget command
	gonugetResult, err := ExecuteGonuget(req.GonugetCommand, req.WorkingDir, req.ConfigFile, req.Timeout)
	if err != nil {
		return nil, fmt.Errorf("execute gonuget: %w", err)
	}

	// Normalize outputs for comparison
	normalizedDotnet := NormalizeOutput(dotnetResult.StdOut)
	normalizedGonuget := NormalizeOutput(gonugetResult.StdOut)

	return ExecuteCommandPairResponse{
		DotnetExitCode:          dotnetResult.ExitCode,
		DotnetStdOut:            dotnetResult.StdOut,
		DotnetStdErr:            dotnetResult.StdErr,
		DotnetSuccess:           dotnetResult.Success,
		GonugetExitCode:         gonugetResult.ExitCode,
		GonugetStdOut:           gonugetResult.StdOut,
		GonugetStdErr:           gonugetResult.StdErr,
		GonugetSuccess:          gonugetResult.Success,
		NormalizedDotnetStdOut:  normalizedDotnet,
		NormalizedGonugetStdOut: normalizedGonuget,
		OutputMatches:           normalizedDotnet == normalizedGonuget,
	}, nil
}

// ExecuteSourcesListHandler handles sources list command
type ExecuteSourcesListHandler struct{}

func (h *ExecuteSourcesListHandler) ErrorCode() string { return "CLI_SRC_LIST_001" }

func (h *ExecuteSourcesListHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ExecuteSourcesListRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Build command
	dotnetCmd := "list source"
	gonugetCmd := "sources list"

	if req.Format != "" {
		dotnetCmd += " --format " + req.Format
		gonugetCmd += " --format " + req.Format
	}

	// Execute both commands
	dotnetResult, err := ExecuteDotnetNuget(dotnetCmd, req.WorkingDir, req.ConfigFile, 30)
	if err != nil {
		return nil, fmt.Errorf("execute dotnet nuget: %w", err)
	}

	gonugetResult, err := ExecuteGonuget(gonugetCmd, req.WorkingDir, req.ConfigFile, 30)
	if err != nil {
		return nil, fmt.Errorf("execute gonuget: %w", err)
	}

	// Parse outputs
	dotnetSources := parseSourcesList(dotnetResult.StdOut)
	gonugetSources := parseSourcesList(gonugetResult.StdOut)

	// Compare
	sourcesMatch := compareSourceLists(dotnetSources, gonugetSources)

	return ExecuteSourcesListResponse{
		DotnetExitCode:  dotnetResult.ExitCode,
		GonugetExitCode: gonugetResult.ExitCode,
		DotnetSources:   dotnetSources,
		GonugetSources:  gonugetSources,
		SourcesMatch:    sourcesMatch,
	}, nil
}

// parseSourcesList parses sources list output into structured format
func parseSourcesList(output string) []PackageSource {
	var sources []PackageSource
	lines := strings.Split(output, "\n")

	// Match pattern: "  1.  nuget.org [Enabled]"
	// Next line contains URL
	re := regexp.MustCompile(`^\s*\d+\.\s+(.+?)\s+\[(Enabled|Disabled)\]`)

	for i := 0; i < len(lines); i++ {
		match := re.FindStringSubmatch(lines[i])
		if match != nil {
			name := strings.TrimSpace(match[1])
			enabled := match[2] == "Enabled"

			// Next line should contain URL
			if i+1 < len(lines) {
				url := strings.TrimSpace(lines[i+1])
				if url != "" {
					sources = append(sources, PackageSource{
						Name:    name,
						URL:     url,
						Enabled: enabled,
					})
				}
			}
		}
	}

	return sources
}

// compareSourceLists compares two lists of package sources
func compareSourceLists(a, b []PackageSource) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].Name != b[i].Name ||
			a[i].URL != b[i].URL ||
			a[i].Enabled != b[i].Enabled {
			return false
		}
	}

	return true
}
```

---

### helpers.go

```go
package main

import (
	"regexp"
	"strings"
)

// NormalizeOutput normalizes command output for comparison
func NormalizeOutput(output string) string {
	if output == "" {
		return ""
	}

	normalized := output

	// Normalize line endings
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")

	// Normalize path separators
	normalized = strings.ReplaceAll(normalized, "\\", "/")

	// Remove timestamps (e.g., "2025-01-25 14:30:45")
	re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}`)
	normalized = re.ReplaceAllString(normalized, "<TIMESTAMP>")

	// Remove version numbers from tool output
	re = regexp.MustCompile(`(NuGet|gonuget)\s+\d+\.\d+\.\d+(\.\d+)?`)
	normalized = re.ReplaceAllString(normalized, "$1 <VERSION>")

	// Normalize absolute paths to relative
	normalized = normalizePaths(normalized)

	// Trim trailing whitespace from each line
	lines := strings.Split(normalized, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	normalized = strings.Join(lines, "\n")

	return strings.TrimSpace(normalized)
}

// normalizePaths normalizes absolute paths to relative paths
func normalizePaths(output string) string {
	// Remove drive letters (Windows)
	re := regexp.MustCompile(`[A-Z]:[/\\]`)
	output = re.ReplaceAllString(output, "")

	// Replace home directory patterns
	re = regexp.MustCompile(`/Users/[^/]+/`)
	output = re.ReplaceAllString(output, "~/")

	re = regexp.MustCompile(`/home/[^/]+/`)
	output = re.ReplaceAllString(output, "~/")

	re = regexp.MustCompile(`C:\\Users\\[^\\]+\\`)
	output = re.ReplaceAllString(output, "~/")

	return output
}
```

---

## C# Test Project: `tests/cli-interop`

### File Structure

```
tests/cli-interop/
├── GonugetCliInterop.Tests/
│   ├── GonugetCliInterop.Tests.csproj
│   ├── TestHelpers/
│   │   ├── GonugetCliBridge.cs       # Bridge to Go executable
│   │   ├── CommandResult.cs          # Response models
│   │   └── TestEnvironment.cs        # Test setup/cleanup
│   ├── Foundation/
│   │   ├── VersionCommandTests.cs
│   │   ├── ConfigCommandTests.cs
│   │   └── SourcesCommandTests.cs
│   └── ... (more test categories)
└── Makefile
```

---

### GonugetCliBridge.cs

```csharp
using System.Diagnostics;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Text;
using System.Text.Json;

namespace GonugetCliInterop.Tests.TestHelpers;

public class GonugetCliBridge
{
    private readonly string _executablePath;

    public GonugetCliBridge()
    {
        _executablePath = FindExecutable();
    }

    /// <summary>
    /// Execute a command pair (dotnet nuget vs gonuget)
    /// </summary>
    public ExecuteCommandPairResponse ExecuteCommandPair(
        string dotnetCommand,
        string gonugetCommand,
        string workingDir,
        string? configFile = null,
        int timeout = 30)
    {
        var request = new
        {
            action = "execute_command_pair",
            data = new
            {
                dotnetCommand,
                gonugetCommand,
                workingDir,
                configFile,
                timeout
            }
        };

        return Execute<ExecuteCommandPairResponse>(request);
    }

    /// <summary>
    /// Execute sources list command
    /// </summary>
    public ExecuteSourcesListResponse ExecuteSourcesList(
        string workingDir,
        string? configFile = null,
        string? format = null)
    {
        var request = new
        {
            action = "execute_sources_list",
            data = new
            {
                workingDir,
                configFile,
                format
            }
        };

        return Execute<ExecuteSourcesListResponse>(request);
    }

    private T Execute<T>(object request)
    {
        var requestJson = JsonSerializer.Serialize(request);

        var startInfo = new ProcessStartInfo
        {
            FileName = _executablePath,
            RedirectStandardInput = true,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
            CreateNoWindow = true
        };

        using var process = new Process { StartInfo = startInfo };
        process.Start();

        // Write request to stdin
        process.StandardInput.WriteLine(requestJson);
        process.StandardInput.Close();

        // Read response from stdout
        var responseJson = process.StandardOutput.ReadToEnd();
        var stderr = process.StandardError.ReadToEnd();

        if (!process.WaitForExit(30000))
        {
            process.Kill();
            throw new TimeoutException("Bridge process timed out");
        }

        // Parse response
        var response = JsonSerializer.Deserialize<BridgeResponse<T>>(responseJson);

        if (response == null)
        {
            throw new Exception($"Failed to parse bridge response. StdErr: {stderr}");
        }

        if (!response.Success)
        {
            throw new GonugetCliException(
                response.Error?.Code ?? "UNKNOWN",
                response.Error?.Message ?? "Unknown error",
                response.Error?.Details);
        }

        return response.Data!;
    }

    private string FindExecutable()
    {
        var exeName = RuntimeInformation.IsOSPlatform(OSPlatform.Windows)
            ? "gonuget-cli-interop-test.exe"
            : "gonuget-cli-interop-test";

        // Check test output directory
        var testDir = Path.GetDirectoryName(Assembly.GetExecutingAssembly().Location);
        var testPath = Path.Combine(testDir!, exeName);
        if (File.Exists(testPath))
            return testPath;

        // Check repository root (6 levels up)
        var repoRoot = Path.GetFullPath(Path.Combine(testDir!, "../../../../../.."));
        var repoPath = Path.Combine(repoRoot, exeName);
        if (File.Exists(repoPath))
            return repoPath;

        throw new FileNotFoundException(
            $"gonuget-cli-interop-test executable not found. " +
            $"Build it with: go build -o {exeName} ./cmd/gonuget-cli-interop-test");
    }

    private class BridgeResponse<T>
    {
        public bool Success { get; set; }
        public T? Data { get; set; }
        public ErrorInfo? Error { get; set; }
    }

    private class ErrorInfo
    {
        public string Code { get; set; } = string.Empty;
        public string Message { get; set; } = string.Empty;
        public string? Details { get; set; }
    }
}

public class GonugetCliException : Exception
{
    public string Code { get; }
    public string? Details { get; }

    public GonugetCliException(string code, string message, string? details = null)
        : base(message)
    {
        Code = code;
        Details = details;
    }
}
```

---

## Building and Running

### Build Go CLI Interop Bridge

```bash
# From repository root
go build -o gonuget-cli-interop-test ./cmd/gonuget-cli-interop-test

# Or use Makefile
make build-cli-interop
```

### Build gonuget CLI (required for tests)

```bash
# From repository root
go build -o gonuget ./cmd/gonuget
```

### Run Tests

```bash
# From repository root
cd tests/cli-interop/GonugetCliInterop.Tests
dotnet test

# Or use Makefile
make test-cli-interop
```

---

## Makefile Integration

### Root Makefile Updates

```makefile
# Build CLI interop test bridge
build-cli-interop: ## Build the gonuget-cli-interop-test binary
	@echo "Building gonuget-cli-interop-test binary..."
	@cd cmd/gonuget-cli-interop-test && go build -o ../../gonuget-cli-interop-test .

# Build gonuget CLI
build-cli: ## Build the gonuget CLI executable
	@echo "Building gonuget CLI..."
	@go build -o gonuget ./cmd/gonuget

# Run CLI interop tests
test-cli-interop: build-cli build-cli-interop ## Run CLI interop tests
	@echo "Running CLI interop tests..."
	@cd tests/cli-interop/GonugetCliInterop.Tests && dotnet test
```

---

## Test Philosophy

These tests follow the same principle as library interop tests:

1. **Source of Truth**: `dotnet nuget` commands are always correct
2. **Behavioral Testing**: We test command-line behavior, not implementation
3. **Comprehensive Coverage**: All commands with all flag combinations
4. **Cross-Platform**: Tests run on Windows, macOS, and Linux
5. **JSON-RPC Bridge**: Same pattern as library interop for consistency
6. **Continuous Validation**: CI ensures gonuget CLI stays compatible

**If a CLI interop test fails, gonuget has a bug - not dotnet nuget.**

---

## Related Documentation

- **Library Interop**: `cmd/nuget-interop-test/` and `tests/nuget-client-interop/`
- **CLI Design**: `docs/design/CLI-DESIGN.md`
- **CLI PRD**: `docs/requirements/CLI-PRD.md`
- **Implementation Roadmap**: `docs/implementation/CLI-MILESTONES-INDEX.md`
