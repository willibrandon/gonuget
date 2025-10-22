# gonuget Interoperability Tests

Comprehensive test suite that validates gonuget's implementation against the official NuGet.Client libraries.

## Overview

This test suite ensures that gonuget produces identical behavior to Microsoft's official NuGet.Client implementation across all major subsystems. The tests use NuGet.Client as the source of truth, comparing gonuget's outputs against the official implementation for correctness.

## Architecture

The test suite uses a **JSON-RPC bridge** pattern:

1. **C# Tests** (xUnit) - Define test cases using official NuGet.Client APIs
2. **GonugetBridge** - C# helper that spawns the Go executable via `System.Diagnostics.Process`
3. **gonuget-interop-test** - Go CLI that exposes gonuget functionality via stdin/stdout JSON-RPC
4. **Comparison** - Tests compare Go results against C# results to validate behavior

```
┌─────────────────────────────────────────────────────────────┐
│  xUnit Test (C#)                                            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ var nugetResult = NuGetVersion.Parse("1.2.3");      │   │
│  │ var gonugetResult = GonugetBridge.ParseVersion(...);│   │
│  │ Assert.Equal(nugetResult.Major, gonugetResult.M...);│   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│                           ▼                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ GonugetBridge.cs                                    │   │
│  │ • Spawns gonuget-interop-test process              │   │
│  │ • Sends JSON request via stdin                     │   │
│  │ • Receives JSON response via stdout                │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  gonuget-interop-test (Go)                                  │
│  • Reads JSON from stdin                                    │
│  • Invokes gonuget implementation                           │
│  • Returns JSON result via stdout                           │
└─────────────────────────────────────────────────────────────┘
```

## Test Coverage

**8 test classes** with **327 total tests** covering all major gonuget subsystems:

| Test Class | Tests | Coverage |
|------------|-------|----------|
| `BridgeSmokeTests.cs` | 5 | Basic bridge communication validation |
| `SignatureTests.cs` | ~50 | Package signing (PKCS#7, Author/Repository, timestamps) |
| `VersionTests.cs` | ~80 | SemVer parsing, comparison, normalization |
| `FrameworkTests.cs` | ~60 | TFM parsing, compatibility checks |
| `PackageReaderTests.cs` | ~40 | Package reading, building, metadata extraction |
| `AssetSelectionInteropTests.cs` | ~45 | Asset selection (lib/, ref/, runtime/, native/) |
| `ContentModelTests.cs` | ~15 | Pattern matching, property extraction |
| `RuntimeIdentifierTests.cs` | 32 | RID expansion, compatibility graph |

### Test Categories

#### 1. Signature Tests (`SignatureTests.cs`)
Validates PKCS#7 signature creation, parsing, and verification:
- Author vs Repository signature types
- SHA256, SHA384, SHA512 hash algorithms
- Timestamp validation (RFC 3161)
- Certificate chain verification
- Self-signed certificate handling
- Signature metadata extraction (signer, issuer, dates)

#### 2. Version Tests (`VersionTests.cs`)
Validates NuGet version semantics:
- Parsing: `1.2.3`, `1.2.3-beta.1`, `1.2.3+metadata`
- Comparison: Stable vs prerelease ordering
- Normalization: `1.0` → `1.0.0`, `1.2.3.0` → `1.2.3`
- Edge cases: Leading zeros, long version numbers
- Prerelease labels: Lexicographic comparison
- Metadata: Should be ignored in comparisons

#### 3. Framework Tests (`FrameworkTests.cs`)
Validates Target Framework Moniker (TFM) handling:
- Parsing: `.NETFramework`, `.NETCoreApp`, `.NETStandard`
- Compatibility: Forward compatibility rules (e.g., net6.0 package on net8.0 project)
- Versioning: `net45`, `net6.0`, `netstandard2.1`, `netcoreapp3.1`
- Portable profiles: `portable-net45+win8`
- Platform-specific: `net6.0-windows`, `net7.0-android`

#### 4. Package Reader Tests (`PackageReaderTests.cs`)
Validates package reading and building:
- ZIP structure validation
- `.nuspec` parsing
- Dependency extraction
- File enumeration
- Signature detection
- Minimal package building

#### 5. Asset Selection Tests (`AssetSelectionInteropTests.cs`)
Validates NuGet content model asset selection:
- `lib/` - Runtime assemblies
- `ref/` - Compile-time reference assemblies
- `runtime/` - RID-specific native libraries
- `native/` - Architecture-specific binaries
- Best framework selection (nearest compatible)
- RID-specific fallback chains

#### 6. Content Model Tests (`ContentModelTests.cs`)
Validates pattern matching and property extraction:
- Pattern expressions: `lib/{tfm}/{assembly}`
- Token parsing: `{tfm}`, `{rid}`, `{locale}`
- Pattern sets: RuntimeAssemblies, CompileRefAssemblies
- Property defaults and overrides

#### 7. Runtime Identifier Tests (`RuntimeIdentifierTests.cs`)
Validates RID graph and compatibility:
- **Expansion**: `win10-x64` → `[win10-x64, win10, win81-x64, win81, win8-x64, win8, win7-x64, win7, win-x64, win, any, base]`
- **Compatibility**: `win10-x64` target can use `win-x64` package (true), `linux-x64` package (false)
- **Platform families**: Windows, Linux, macOS
- **Architecture**: x86, x64, ARM, ARM64
- **OS versions**: win7 → win8 → win81 → win10, ubuntu.20.04 → ubuntu.22.04 → ubuntu.24.04

#### 8. Bridge Smoke Tests (`BridgeSmokeTests.cs`)
Validates basic bridge functionality:
- Process spawning and communication
- JSON serialization/deserialization
- Error handling
- Timeout behavior

## Prerequisites

- **.NET 9.0 SDK** - [Download](https://dotnet.microsoft.com/download/dotnet/9.0)
- **gonuget-interop-test executable** - Must be built before running tests

## Building the Bridge

Before running tests, build the Go bridge executable.

### Using Make (Recommended)

From the repository root:
```bash
# Build just the interop bridge
make build-interop

# Build bridge + .NET test project
make build-dotnet

# Build everything (Go packages + bridge + .NET tests)
make build
```

Or from the test directory:
```bash
cd tests/nuget-client-interop
make build    # Builds gonuget-interop-test executable
```

### Manual Build

```bash
# From repository root
cd cmd/nuget-interop-test
go build -o ../../gonuget-interop-test .
cd ../..
```

The executable will be placed at the repository root where tests can find it.

## Running Tests

### Using Make (Recommended)

From the repository root:
```bash
# Run all tests (Go unit + integration + .NET interop)
make test

# Run only .NET interop tests
make test-interop

# Run specific test categories
make test-smoke      # Only BridgeSmokeTests
make test-version    # Only VersionTests
make test-signature  # Only SignatureTests

# Quick rebuild + test (for development)
make quick-test

# Show test count by category
make test-count
```

From the test directory:
```bash
cd tests/nuget-client-interop
make test          # Build bridge + run tests
make test-verbose  # Run with detailed output
```

### Using dotnet CLI

```bash
# Run all tests
cd tests/nuget-client-interop/GonugetInterop.Tests
dotnet test

# Run specific test class
dotnet test --filter "FullyQualifiedName~RuntimeIdentifierTests"

# Run with verbose output
dotnet test --logger "console;verbosity=detailed"

# List all tests
dotnet test --list-tests

# Run tests with coverage
dotnet test --collect:"XPlat Code Coverage"
```

## Dependencies

From `GonugetInterop.Tests.csproj`:

```xml
<ItemGroup>
  <!-- Test Framework -->
  <PackageReference Include="xunit" Version="2.6.1" />
  <PackageReference Include="xunit.runner.visualstudio" Version="2.8.2" />

  <!-- NuGet Client Libraries (Source of Truth) -->
  <PackageReference Include="NuGet.Packaging" Version="6.12.1" />
  <PackageReference Include="NuGet.Versioning" Version="6.12.1" />
  <PackageReference Include="NuGet.Frameworks" Version="6.12.1" />
</ItemGroup>
```

**NuGet.Client 6.12.1** is used as the reference implementation for all validation.

## Test Helpers

### GonugetBridge.cs
Central bridge helper that exposes 15 methods corresponding to the 15 bridge actions:

```csharp
// Signature operations
byte[] SignPackage(byte[] packageHash, string certPath, ...)
ParseSignatureResponse ParseSignature(byte[] signature)
VerifySignatureResponse VerifySignature(byte[] signature, ...)

// Version operations
CompareVersionsResponse CompareVersions(string version1, string version2)
ParseVersionResponse ParseVersion(string version)

// Framework operations
CheckFrameworkCompatResponse CheckFrameworkCompat(string packageFramework, string projectFramework)
ParseFrameworkResponse ParseFramework(string framework)

// Package operations
ReadPackageResponse ReadPackage(byte[] packageBytes)
BuildPackageResponse BuildPackage(string id, string version, ...)

// Asset selection operations
FindAssembliesResponse FindRuntimeAssemblies(string[] paths, string targetFramework)
FindAssembliesResponse FindCompileAssemblies(string[] paths, string targetFramework)
ParseAssetPathResponse ParseAssetPath(string path)

// RID operations
ExpandRuntimeResponse ExpandRuntime(string rid)
AreRuntimesCompatibleResponse AreRuntimesCompatible(string targetRid, string packageRid)
```

### TestCertificates.cs
Helper for generating test certificates:
- Creates X.509 code signing certificates
- Exports to PEM format (certificate + private key)
- Handles RSA key generation
- Supports self-signed certificates

### Response Types
Strongly-typed C# response models for all bridge actions:
- `SignPackageResponse`
- `ParseSignatureResponse`
- `VerifySignatureResponse`
- `CompareVersionsResponse`
- `ParseVersionResponse`
- And more...

## Example Test

```csharp
[Fact]
public void ExpandRuntime_Win10x64_ShouldMatchNuGetClient()
{
    // NuGet.Client (source of truth)
    var runtimeGraph = CreateDefaultRuntimeGraph();
    var nugetExpanded = runtimeGraph.ExpandRuntime("win10-x64").ToArray();

    // Expected chain
    var expected = new[]
    {
        "win10-x64", "win10", "win81-x64", "win81", "win8-x64", "win8",
        "win7-x64", "win7", "win-x64", "win", "any", "base"
    };
    Assert.Equal(expected, nugetExpanded);

    // Gonuget (under test)
    var gonugetResponse = GonugetBridge.ExpandRuntime("win10-x64");

    // Validation
    Assert.Equal(expected, gonugetResponse.ExpandedRuntimes);
}
```

## Test Discovery

The bridge automatically locates the `gonuget-interop-test` executable in:
1. **Test output directory**: `bin/Debug/net9.0/gonuget-interop-test`
2. **Repository root**: `../../../../../../gonuget-interop-test` (relative from test DLL)

If the executable is not found, tests will fail with:
```
FileNotFoundException: gonuget-interop-test executable not found.
Run 'go build -o gonuget-interop-test ./cmd/nuget-interop-test' before running tests.
```

## Error Handling

The bridge throws `GonugetException` on failures:

```csharp
public class GonugetException : Exception
{
    public string Code { get; }      // e.g., "parse_error", "signature_error"
    public string Details { get; }   // Additional context
}
```

All bridge errors include:
- Error code (structured)
- Human-readable message
- Optional details for debugging

## CI/CD Integration

### Using Make
```bash
# From repository root - run all tests
make test

# Or run only interop tests
make test-interop
```

### Manual
```bash
go build -o gonuget-interop-test ./cmd/nuget-interop-test
cd tests/nuget-client-interop/GonugetInterop.Tests
dotnet test
```

## Makefile Targets

### Root Makefile (repository root)
- `make build` - Build all Go packages, interop bridge, and .NET tests
- `make build-interop` - Build only the gonuget-interop-test binary
- `make build-dotnet` - Build .NET test project (requires interop binary)
- `make test` - Run all tests (Go unit + integration + .NET interop)
- `make test-interop` - Run only .NET interop tests
- `make test-go-unit` - Run only Go unit tests (skip integration)
- `make test-smoke` - Run only BridgeSmokeTests
- `make test-version` - Run only VersionTests
- `make test-signature` - Run only SignatureTests
- `make quick-test` - Quick rebuild and test (no clean)
- `make test-count` - Show test count by category
- `make clean` - Clean all build artifacts

### Test Directory Makefile (tests/nuget-client-interop)
- `make build` - Build gonuget-interop-test executable
- `make restore` - Restore C# project dependencies
- `make test` - Build bridge and run interop tests
- `make test-verbose` - Run tests with detailed output
- `make clean` - Clean interop test artifacts

## Related Components

- **Bridge Executable**: `cmd/nuget-interop-test/` - Go CLI that exposes gonuget via JSON-RPC
- **Implementation**: `packaging/`, `version/`, `frameworks/`, `core/` - gonuget subsystems under test
- **Documentation**: See implementation guides in `docs/implementation/` for architecture details

## Test Philosophy

These tests follow the principle of **reference implementation validation**:

1. **Source of Truth**: NuGet.Client 6.12.1 is always correct
2. **Behavioral Testing**: We test behavior, not implementation details
3. **Comprehensive Coverage**: 327 tests cover all major code paths
4. **Cross-Language**: JSON bridge eliminates language-specific quirks
5. **Continuous Validation**: CI ensures gonuget stays compatible with NuGet ecosystem

If a test fails, gonuget has a bug - not NuGet.Client.
