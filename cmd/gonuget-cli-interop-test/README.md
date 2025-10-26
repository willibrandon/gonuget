# gonuget-cli-interop-test

A JSON-RPC bridge that enables CLI behavior comparison testing between `gonuget` and `dotnet nuget` commands.

## Overview

This executable provides a command-line interface for executing both `dotnet nuget` and `gonuget` CLI commands in parallel and comparing their behavior. It acts as a bridge to validate that gonuget's CLI produces identical output, exit codes, and behavior to the official `dotnet nuget` CLI.

## Architecture Difference: CLI vs Library Interop

This binary is **different** from `nuget-interop-test`:

| Aspect | Library Interop (`nuget-interop-test`) | CLI Interop (`gonuget-cli-interop-test`) |
|--------|----------------------------------------|------------------------------------------|
| **What's tested** | Go library code | CLI binary behavior |
| **Execution** | In-process function calls | Subprocess execution via `exec.Command()` |
| **Comparison** | Return values, objects | stdout/stderr/exit codes |
| **Purpose** | API parity with NuGet.Client | UX parity with `dotnet nuget` |

Both ensure 100% parity with NuGet, but at different layers.

## JSON-RPC Protocol

The bridge uses a simple JSON-RPC protocol over stdin/stdout:

**Request Format:**
```json
{
  "action": "action_name",
  "data": { /* action-specific parameters */ }
}
```

**Response Format:**
```json
{
  "success": true,
  "data": { /* action-specific results */ }
}
```

**Error Response:**
```json
{
  "success": false,
  "error": {
    "code": "error_code",
    "message": "Human-readable message",
    "details": "Additional context (optional)"
  }
}
```

## Supported Actions

The bridge exposes 6 actions for CLI behavior comparison:

### Version Command
- **`execute_version`** - Execute version command on both CLIs
  - Input: workingDir
  - Output: dotnetExitCode, gonugetExitCode, dotnetStdOut, gonugetStdOut, dotnetStdErr, gonugetStdErr, exitCodesMatch, outputFormatSimilar

### Config Get Command
- **`execute_config_get`** - Execute config get on both CLIs
  - Inputs: key, workingDir, showPath (optional), workingDirFlag (optional)
  - Output: dotnetExitCode, gonugetExitCode, dotnetStdOut, gonugetStdOut, dotnetStdErr, gonugetStdErr, outputMatches

### Config Set Command
- **`execute_config_set`** - Execute config set on both CLIs
  - Inputs: key, value, workingDir
  - Output: dotnetExitCode, gonugetExitCode, dotnetStdOut, gonugetStdOut, dotnetStdErr, gonugetStdErr, outputMatches

### Config Unset Command
- **`execute_config_unset`** - Execute config unset on both CLIs
  - Inputs: key, workingDir
  - Output: dotnetExitCode, gonugetExitCode, dotnetStdOut, gonugetStdOut, dotnetStdErr, gonugetStdErr, outputMatches

### Config Paths Command
- **`execute_config_paths`** - Execute config paths on both CLIs
  - Inputs: workingDir, workingDirFlag (optional)
  - Output: dotnetExitCode, gonugetExitCode, dotnetStdOut, gonugetStdOut, dotnetStdErr, gonugetStdErr, outputMatches

### Generic Command Pair
- **`execute_command_pair`** - Execute arbitrary command pair
  - Inputs: dotnetCommand, gonugetCommand, workingDir, configFile (optional), timeout
  - Output: dotnetExitCode, gonugetExitCode, dotnetStdOut, gonugetStdOut, dotnetStdErr, gonugetStdErr, normalizedDotnetStdOut, normalizedGonugetStdOut, outputMatches, dotnetSuccess, gonugetSuccess

## Building

### Using Make (Recommended)

```bash
# From repository root
make build-cli-interop

# Or build everything
make build
```

### Manual Build

```bash
# From repository root
go build -o gonuget-cli-interop-test ./cmd/gonuget-cli-interop-test

# Or from cmd/gonuget-cli-interop-test directory
cd cmd/gonuget-cli-interop-test
go build -o ../../gonuget-cli-interop-test .
```

## Usage

The bridge is designed to be invoked by automated tests, not used interactively. However, for manual testing:

```bash
# Echo JSON request to stdin
echo '{"action":"execute_version","data":{"workingDir":"/tmp"}}' | ./gonuget-cli-interop-test

# Output:
# {
#   "success": true,
#   "data": {
#     "dotnetExitCode": 0,
#     "gonugetExitCode": 0,
#     "dotnetStdOut": "NuGet Command Line\n6.x.x.x\n",
#     "gonugetStdOut": "gonuget version 0.0.0-dev\n",
#     "dotnetStdErr": "",
#     "gonugetStdErr": "",
#     "exitCodesMatch": true,
#     "outputFormatSimilar": true
#   }
# }
```

## How Commands Are Executed

The bridge spawns **two separate processes** for each request:

### 1. dotnet nuget Command (`executor.go:69-84`)
```go
func ExecuteDotnetNuget(command string, workingDir string, configFile string, timeout int) (*CommandResult, error) {
    dotnetExe := "dotnet"
    args := []string{"nuget"}
    args = append(args, strings.Fields(command)...)

    return ExecuteCommand(dotnetExe, args, workingDir, timeout)
}
```

Example: `dotnet nuget config get repositoryPath`

### 2. gonuget Command (`executor.go:86-97`)
```go
func ExecuteGonuget(command string, workingDir string, configFile string, timeout int) (*CommandResult, error) {
    gonugetExe := findGonugetExecutable()
    args := strings.Fields(command)

    return ExecuteCommand(gonugetExe, args, workingDir, timeout)
}
```

Example: `gonuget config get repositoryPath`

### 3. Generic Executor (`executor.go:22-67`)
Uses Go's `os/exec.Command()` to:
- Spawn process with arguments
- Set working directory
- Capture stdout and stderr
- Wait with timeout
- Return exit code and output

## Integration with C# Tests

The C# test suite (`tests/cli-interop/GonugetCliInterop.Tests`) uses `GonugetCliBridge.cs` to invoke this executable via `System.Diagnostics.Process`:

```csharp
// C# test helper spawns the bridge process
var result = _bridge.ExecuteConfigGet("repositoryPath", env.TestDirectory);
Assert.Equal(0, result.DotnetExitCode);
Assert.Equal(0, result.GonugetExitCode);
Assert.True(result.OutputMatches);
```

The bridge automatically finds the executable in:
1. Test output directory (`bin/Debug/net9.0/`)
2. Repository root (for local development)

## Error Handling

All errors are returned as structured JSON with error codes:

| Error Code Prefix | Meaning |
|-------------------|---------|
| `REQ_001` | Failed to parse request JSON |
| `ACT_001` | Unknown action name |
| `CLI_EXEC_001` | Command pair execution failed |
| `CLI_VER_001` | Version command execution failed |
| `CLI_CFG_GET_001` | Config get command failed |
| `CLI_CFG_SET_001` | Config set command failed |
| `CLI_CFG_UNSET_001` | Config unset command failed |
| `CLI_CFG_PATHS_001` | Config paths command failed |

Example error response:
```json
{
  "success": false,
  "error": {
    "code": "CLI_CFG_GET_001",
    "message": "execute dotnet nuget: command timed out after 30 seconds",
    "details": ""
  }
}
```

## Implementation Notes

- **Handler Pattern**: Each action is implemented by a handler that implements the `Handler` interface
- **JSON Serialization**: Uses Go's `encoding/json` with camelCase property names
- **Process Execution**: Uses `os/exec.Command()` to spawn `dotnet` and `gonuget` processes
- **Output Normalization**: Normalizes whitespace and line endings for comparison
- **Process Isolation**: Each request spawns new processes (C# side controls lifecycle)
- **Timeout**: Default 30-second timeout per command execution
- **Exit Code Handling**: Preserves exact exit codes from both commands

## File Structure

```
cmd/gonuget-cli-interop-test/
├── main.go              # Request routing and main entry point
├── protocol.go          # Request/response type definitions
├── executor.go          # Command execution via os/exec
├── helpers.go           # Output normalization utilities
├── handlers_version.go  # Version command handler
└── handlers_config.go   # Config command handlers (get/set/unset/paths)
```

## Related Components

- **C# Test Suite**: `tests/cli-interop/GonugetCliInterop.Tests/`
- **Bridge Client**: `tests/cli-interop/GonugetCliInterop.Tests/TestHelpers/GonugetCliBridge.cs`
- **CLI Implementation**: `cmd/gonuget/` - The gonuget CLI being tested
- **Test Coverage**: 14 tests validating CLI behavior parity

## Comparison with Library Interop

| Feature | Library Interop | CLI Interop |
|---------|----------------|-------------|
| **Binary** | `nuget-interop-test` | `gonuget-cli-interop-test` |
| **Tests** | 491 tests | 14 tests |
| **What's compared** | Library API results | CLI stdout/stderr/exit codes |
| **Execution** | Direct Go function calls | Process spawning via `exec.Command()` |
| **Use case** | Validate core library logic | Validate user-facing CLI behavior |
| **Example** | Compare `version.Parse()` | Compare `gonuget version` vs `dotnet nuget --version` |

Both are essential for 100% NuGet parity - library interop ensures correctness, CLI interop ensures user experience matches.
