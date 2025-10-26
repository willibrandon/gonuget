# gonuget CLI Interoperability Tests

Comprehensive test suite that validates gonuget's CLI behavior against the official `dotnet nuget` CLI.

## Overview

This test suite ensures that `gonuget` produces identical command-line behavior to `dotnet nuget` across all implemented commands. The tests use `dotnet nuget` as the source of truth, comparing gonuget's exit codes, stdout, stderr, and overall UX against the official .NET CLI.

## Architecture Difference: CLI vs Library Testing

This test suite is **different** from `tests/nuget-client-interop`:

| Aspect | Library Interop Tests | CLI Interop Tests |
|--------|----------------------|-------------------|
| **What's tested** | Go library API | CLI binary behavior |
| **Comparison** | Function return values | stdout/stderr/exit codes |
| **Execution** | In-process function calls | Subprocess execution |
| **Source of truth** | NuGet.Client C# libraries | `dotnet nuget` CLI |
| **Test count** | 491 tests | 14 tests |
| **Purpose** | Validate core logic | Validate user experience |

Both ensure 100% parity with NuGet, but at different layers.

## Architecture

The test suite uses a **JSON-RPC bridge** pattern:

1. **C# Tests** (xUnit) - Define test cases using `dotnet nuget` as reference
2. **GonugetCliBridge** - C# helper that spawns the Go executable via `System.Diagnostics.Process`
3. **gonuget-cli-interop-test** - Go binary that executes both `dotnet nuget` and `gonuget` commands
4. **Comparison** - Tests compare exit codes, stdout, stderr from both CLIs

```
┌─────────────────────────────────────────────────────────────┐
│  xUnit Test (C#)                                            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ var result = _bridge.ExecuteConfigGet("key", ...);  │   │
│  │ Assert.Equal(0, result.DotnetExitCode);             │   │
│  │ Assert.Equal(0, result.GonugetExitCode);            │   │
│  │ Assert.True(result.OutputMatches);                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│                           ▼                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ GonugetCliBridge.cs                                 │   │
│  │ • Spawns gonuget-cli-interop-test process          │   │
│  │ • Sends JSON request via stdin                     │   │
│  │ • Receives JSON response via stdout                │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  gonuget-cli-interop-test (Go)                              │
│  • Reads JSON from stdin                                    │
│  • Spawns TWO processes:                                    │
│    1. dotnet nuget <command>                                │
│    2. gonuget <command>                                     │
│  • Captures stdout/stderr/exit codes from both              │
│  • Returns comparison JSON via stdout                       │
└─────────────────────────────────────────────────────────────┘
```

## Test Coverage

**2 test classes** with **14 total tests** covering CLI foundation commands:

| Test Class | Tests | Coverage |
|------------|-------|----------|
| `VersionCommandTests.cs` | 3 | Version command behavior |
| `ConfigCommandTests.cs` | 11 | Config command operations |

### Test Categories

#### 1. Version Command Tests (`VersionCommandTests.cs`)
Validates `gonuget version` vs `dotnet nuget --version`:
- Exit code 0 on success
- Version information in output
- Output format similarity (both show version numbers)

**Example Test:**
```csharp
[Fact]
public void VersionCommand_ReturnsVersionInfo()
{
    var result = _bridge.ExecuteVersion(env.TestDirectory);

    Assert.True(result.ExitCodesMatch, "Exit codes should match");
    Assert.Equal(0, result.DotnetExitCode);
    Assert.Equal(0, result.GonugetExitCode);
    Assert.True(result.OutputFormatSimilar, "Both should contain version info");
}
```

#### 2. Config Command Tests (`ConfigCommandTests.cs`)
Validates `gonuget config` vs `dotnet nuget config`:

**Config Get:**
- Simple value retrieval
- Exit code 2 for missing keys (not found)
- `--show-path` flag output format: `<value>\tfile: <path>`
- Relative path preservation

**Config Set:**
- Setting new values
- Updating existing values
- Requires existing NuGet.config file
- Invalid key validation

**Config Unset:**
- Removing existing keys
- Exit code 2 for missing keys

**Example Test:**
```csharp
[Fact]
public void ConfigGet_ShowPathFlag_MatchesDotnet()
{
    using var env = new TestEnvironment();
    env.CreateTestConfigWithValues(new Dictionary<string, string>
    {
        { "relativePath", "./packages" }
    });

    var result = _bridge.ExecuteConfigGet(
        "relativePath",
        env.TestDirectory,
        showPath: true);

    Assert.Equal(0, result.DotnetExitCode);
    Assert.Equal(0, result.GonugetExitCode);

    // Both should return format: <value><TAB>file: <path>
    Assert.Contains("./packages", result.DotnetStdOut);
    Assert.Contains("./packages", result.GonugetStdOut);
    Assert.Contains("\tfile: ", result.DotnetStdOut);
    Assert.Contains("\tfile: ", result.GonugetStdOut);
}
```

## Prerequisites

- **.NET 9.0 SDK** - [Download](https://dotnet.microsoft.com/download/dotnet/9.0)
- **gonuget binary** - The CLI being tested
- **gonuget-cli-interop-test executable** - Must be built before running tests

## Building the Bridge

Before running tests, build the Go bridge executable.

### Using Make (Recommended)

From the repository root:
```bash
# Build the CLI interop bridge
make build-cli-interop

# Build gonuget CLI
make build

# Build bridge + .NET test project
make build-dotnet
```

### Manual Build

```bash
# Build gonuget CLI
cd cmd/gonuget
go build -o ../../gonuget .
cd ../..

# Build CLI interop test bridge
cd cmd/gonuget-cli-interop-test
go build -o ../../gonuget-cli-interop-test .
cd ../..
```

Both executables will be placed at the repository root where tests can find them.

## Running Tests

### Using Make (Recommended)

From the repository root:
```bash
# Run all tests (Go unit + integration + library interop + CLI interop)
make test

# Run only CLI interop tests
make test-cli-interop

# Quick rebuild + test (for development)
make quick-test
```

### Using dotnet CLI

```bash
# Run all tests
cd tests/cli-interop/GonugetCliInterop.Tests
dotnet test

# Run specific test class
dotnet test --filter "FullyQualifiedName~ConfigCommandTests"

# Run with verbose output
dotnet test --logger "console;verbosity=detailed"

# List all tests
dotnet test --list-tests
```

## Test Helpers

### GonugetCliBridge.cs
Central bridge helper that exposes CLI command execution methods:

```csharp
/// <summary>
/// Bridge for executing gonuget CLI commands via JSON-RPC protocol.
/// </summary>
public class GonugetCliBridge
{
    // Version command
    ExecuteVersionResponse ExecuteVersion(string workingDir)

    // Config commands
    ExecuteConfigGetResponse ExecuteConfigGet(string key, string workingDir,
        bool showPath = false, string? workingDirFlag = null)

    ExecuteConfigSetResponse ExecuteConfigSet(string key, string value,
        string workingDir)

    ExecuteConfigUnsetResponse ExecuteConfigUnset(string key,
        string workingDir)

    ExecuteConfigPathsResponse ExecuteConfigPaths(string workingDir,
        string? workingDirFlag = null)

    // Generic command pair execution
    ExecuteCommandPairResponse ExecuteCommandPair(string dotnetCommand,
        string gonugetCommand, string workingDir,
        string? configFile = null, int timeout = 30)
}
```

### TestEnvironment.cs
Provides disposable test environment for CLI tests:

```csharp
/// <summary>
/// Disposable test environment with temporary directories and config files.
/// </summary>
public class TestEnvironment : IDisposable
{
    public string TestDirectory { get; }
    public string ConfigFilePath { get; }

    // Create test config with specific values
    void CreateTestConfigWithValues(Dictionary<string, string> configValues)

    // Check config file contents
    bool ConfigContains(string key, string value)
    bool ConfigContainsKey(string key)

    // Cleanup on dispose
    void Dispose()
}
```

### Response Types
Strongly-typed C# response models for all bridge actions:

- **`ExecuteVersionResponse`** - Version command comparison
  - Properties: dotnetExitCode, gonugetExitCode, dotnetStdOut, gonugetStdOut, dotnetStdErr, gonugetStdErr, exitCodesMatch, outputFormatSimilar

- **`ExecuteConfigGetResponse`** - Config get comparison
  - Properties: dotnetExitCode, gonugetExitCode, dotnetStdOut, gonugetStdOut, dotnetStdErr, gonugetStdErr, outputMatches

- **`ExecuteConfigSetResponse`** - Config set comparison
- **`ExecuteConfigUnsetResponse`** - Config unset comparison
- **`ExecuteConfigPathsResponse`** - Config paths comparison
- **`ExecuteCommandPairResponse`** - Generic command pair comparison

All response types include:
- Exit codes from both commands
- Stdout and stderr from both commands
- Comparison result (do they match?)

## Test Discovery

The bridge automatically locates the `gonuget-cli-interop-test` executable in:
1. **Test output directory**: `bin/Debug/net9.0/gonuget-cli-interop-test`
2. **Repository root**: `../../../../../../gonuget-cli-interop-test` (relative from test DLL)

If the executable is not found, tests will fail with:
```
FileNotFoundException: gonuget-cli-interop-test executable not found.
Run 'make build-cli-interop' or 'go build -o gonuget-cli-interop-test ./cmd/gonuget-cli-interop-test' before running tests.
```

## Error Handling

The bridge throws `GonugetCliException` on failures:

```csharp
/// <summary>
/// Exception thrown when the CLI interop bridge returns an error.
/// </summary>
public class GonugetCliException : Exception
{
    public string Code { get; }      // e.g., "CLI_CFG_GET_001"
    public string Details { get; }   // Additional context
}
```

All bridge errors include:
- Error code (structured)
- Human-readable message
- Optional details for debugging

## Project Structure

```
tests/cli-interop/
├── GonugetCliInterop.Tests/
│   ├── Foundation/
│   │   ├── ConfigCommandTests.cs    # Config command tests (11 tests)
│   │   └── VersionCommandTests.cs   # Version command tests (3 tests)
│   └── TestHelpers/
│       ├── GonugetCliBridge.cs      # JSON-RPC bridge client
│       ├── TestEnvironment.cs       # Test environment helper
│       ├── BridgeResponse.cs        # JSON response wrapper
│       ├── BridgeErrorInfo.cs       # Error information
│       ├── GonugetCliException.cs   # Bridge exception
│       ├── ExecuteVersionResponse.cs
│       ├── ExecuteConfigGetResponse.cs
│       ├── ExecuteConfigSetResponse.cs
│       ├── ExecuteConfigUnsetResponse.cs
│       ├── ExecuteConfigPathsResponse.cs
│       └── ExecuteCommandPairResponse.cs
└── README.md
```

## CI/CD Integration

### Using Make
```bash
# From repository root - run all tests
make test

# Or run only CLI interop tests
make test-cli-interop
```

### Manual
```bash
# Build CLI and bridge
go build -o gonuget ./cmd/gonuget
go build -o gonuget-cli-interop-test ./cmd/gonuget-cli-interop-test

# Run tests
cd tests/cli-interop/GonugetCliInterop.Tests
dotnet test
```

## Makefile Targets

### Root Makefile (repository root)
- `make build` - Build all Go packages, CLI, interop bridges, and .NET tests
- `make build-cli` - Build only the gonuget CLI executable
- `make build-cli-interop` - Build only the gonuget-cli-interop-test binary
- `make test` - Run all tests (Go unit + integration + library interop + CLI interop)
- `make test-cli-interop` - Run only CLI interop tests
- `make clean` - Clean all build artifacts

### Test Directory Makefile (tests/cli-interop)
- `make build` - Build gonuget CLI and gonuget-cli-interop-test executables
- `make restore` - Restore C# project dependencies
- `make test` - Build binaries and run CLI interop tests
- `make test-verbose` - Run tests with detailed output
- `make clean` - Clean CLI interop test artifacts

## Comparison with Library Interop Tests

| Feature | Library Interop | CLI Interop |
|---------|----------------|-------------|
| **Location** | `tests/nuget-client-interop/` | `tests/cli-interop/` |
| **Binary** | `nuget-interop-test` | `gonuget-cli-interop-test` |
| **Tests** | 491 tests | 14 tests |
| **What's compared** | Library API results | CLI stdout/stderr/exit codes |
| **Execution** | Direct function calls | Process spawning |
| **Source of truth** | NuGet.Client libraries | `dotnet nuget` CLI |
| **Example** | `version.Parse("1.2.3")` | `gonuget version` vs `dotnet nuget --version` |

## Test Philosophy

These tests follow the principle of **CLI behavior validation**:

1. **Source of Truth**: `dotnet nuget` CLI is always correct
2. **User Experience**: We test what users see (stdout/stderr/exit codes)
3. **Exit Code Parity**: Different exit codes for different error types (0=success, 1=error, 2=not found)
4. **Output Format Parity**: Output must match exactly (including tabs, newlines, formatting)
5. **Cross-Platform**: Tests run on Windows, macOS, and Linux
6. **Continuous Validation**: CI ensures gonuget CLI stays compatible with dotnet nuget

If a test fails, gonuget's CLI has a bug - not `dotnet nuget`.

## Related Components

- **Bridge Executable**: `cmd/gonuget-cli-interop-test/` - Go binary that executes both CLIs
- **CLI Implementation**: `cmd/gonuget/` - The gonuget CLI being tested
- **CLI Design**: `docs/design/CLI-DESIGN.md` - CLI architecture and design principles
- **CLI Implementation Guide**: `docs/implementation/CLI-M1-FOUNDATION.md` - CLI implementation roadmap

## Future Test Coverage

As more commands are implemented, tests will be added for:
- `push` - Package publishing
- `delete` - Package deletion
- `list` - Package listing
- `search` - Package search
- `install` - Package installation
- `restore` - Dependency restoration
- `pack` - Package creation
- `sign` - Package signing
- `verify` - Signature verification

Each command will have comprehensive tests validating exact CLI behavior parity with `dotnet nuget`.
