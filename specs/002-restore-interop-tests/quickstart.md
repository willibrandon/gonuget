# Quickstart: Running Restore Transitive Dependency Interop Tests

**Date**: 2025-11-01
**Audience**: gonuget developers

## Overview

This guide explains how to run and develop C# interop tests for gonuget's restore transitive dependency resolution. These tests validate that gonuget behaves identically to NuGet.Client for package resolution, categorization, error messages, and lock file generation.

## Prerequisites

- .NET 9.0 SDK installed
- Go 1.21+ installed
- gonuget repository cloned
- NuGet.Client interop test infrastructure functional (existing 550 tests pass)

## Quick Start

### Run All Restore Transitive Tests

```bash
cd /Users/brandon/src/gonuget

# Build gonuget-interop-test binary
make build-interop

# Run restore transitive tests specifically
cd tests/nuget-client-interop/GonugetInterop.Tests
dotnet test --filter "FullyQualifiedName~RestoreTransitiveTests"
```

### Run Full Interop Test Suite (Including Restore)

```bash
# From repo root
make test-interop

# This runs all 491+ tests including the new restore transitive tests
```

### Run Specific Test

```bash
cd tests/nuget-client-interop/GonugetInterop.Tests

# Run single test by name
dotnet test --filter "FullyQualifiedName~Test_SimpleTransitiveResolution_Parity"
```

## Test Organization

### Test File Structure

```
tests/nuget-client-interop/GonugetInterop.Tests/
├── RestoreTransitiveTests.cs          # Main test file (20-30 tests)
├── TestHelpers/
│   ├── GonugetBridge.cs               # JSON-RPC bridge (extended)
│   ├── RestoreTransitiveResponse.cs   # Response type
│   ├── CompareProjectAssetsResponse.cs
│   ├── ValidateErrorMessagesResponse.cs
│   └── TestProject.cs                 # Test project builder
```

### Test Categories

Tests are organized into 4 categories:

1. **Transitive Resolution Parity** - Verify package resolution matches NuGet.Client
2. **Direct vs Transitive Categorization** - Validate package categorization
3. **Unresolved Package Error Messages** - Check NU1101/NU1102/NU1103 formatting
4. **Lock File Format Compatibility** - Ensure project.assets.json MSBuild compatibility

## Writing New Tests

### Basic Test Pattern

```csharp
[Fact]
public void Test_SimpleTransitiveResolution_Parity()
{
    // 1. Create test project
    var project = TestProject.Create("TestApp")
        .WithFramework("net9.0")
        .AddPackage("Serilog.Sinks.File", "5.0.0")
        .Build();

    // 2. Restore with gonuget
    var gonugetResult = GonugetBridge.RestoreTransitive(project.Path);

    // 3. Restore with NuGet.Client (for comparison)
    var nugetResult = NuGetClientRestore(project.Path);

    // 4. Assert parity
    Assert.Equal(nugetResult.DirectPackages.Count, gonugetResult.DirectPackages.Count);
    Assert.Equal(nugetResult.TransitivePackages.Count, gonugetResult.TransitivePackages.Count);

    // 5. Verify specific packages
    Assert.Contains(gonugetResult.DirectPackages, p => p.PackageId == "Serilog.Sinks.File");
    Assert.Contains(gonugetResult.TransitivePackages, p => p.PackageId == "Serilog");
}
```

### Testing Error Messages

```csharp
[Fact]
public void Test_PackageNotFound_NU1101_Parity()
{
    // Create project with non-existent package
    var project = TestProject.Create("TestApp")
        .WithFramework("net9.0")
        .AddPackage("NonExistentPackage", "1.0.0")
        .Build();

    // Restore with gonuget (should fail)
    var gonugetResult = GonugetBridge.RestoreTransitive(project.Path);
    Assert.False(gonugetResult.Success);
    Assert.Single(gonugetResult.UnresolvedPackages);

    // Validate error message format
    var validation = GonugetBridge.ValidateErrorMessages(
        gonugetResult.ErrorMessages[0],
        expectedNuGetClientMessage
    );

    Assert.True(validation.Match);
    Assert.Equal("NU1101", validation.ErrorCode);
}
```

### Testing Lock File Format

```csharp
[Fact]
public void Test_ProjectAssets_LibrariesMap_Parity()
{
    // Create and restore project
    var project = TestProject.Create("TestApp")
        .WithFramework("net9.0")
        .AddPackage("Newtonsoft.Json", "13.0.4")
        .Build();

    var gonugetResult = GonugetBridge.RestoreTransitive(project.Path);
    var nugetResult = NuGetClientRestore(project.Path);

    // Compare project.assets.json files
    var comparison = GonugetBridge.CompareProjectAssets(
        gonugetResult.LockFilePath,
        nugetResult.LockFilePath
    );

    Assert.True(comparison.AreEqual, string.Join("\n", comparison.Differences));
    Assert.True(comparison.LibrariesMatch);
    Assert.True(comparison.PathsMatch); // Validates lowercase paths
}
```

## Debugging Tests

### View gonuget-interop-test JSON-RPC Communication

```bash
# Run test with detailed output
dotnet test --filter "FullyQualifiedName~Test_SimpleTransitiveResolution_Parity" --logger "console;verbosity=detailed"

# Check gonuget-interop-test binary output (if debugging handler)
# The binary is executed as a subprocess by GonugetBridge
```

### Inspect Generated project.assets.json

```csharp
[Fact]
public void Test_InspectLockFile()
{
    var project = TestProject.Create("TestApp")
        .WithFramework("net9.0")
        .AddPackage("Newtonsoft.Json", "13.0.4")
        .Build();

    var result = GonugetBridge.RestoreTransitive(project.Path);

    // Read and inspect lock file manually
    var lockFileContent = File.ReadAllText(result.LockFilePath);
    Console.WriteLine(lockFileContent); // Output in test results
}
```

### Compare Against NuGet.Client Behavior

```bash
# Create test project manually
mkdir /tmp/test-nuget
cd /tmp/test-nuget

# Create .csproj
cat > test.csproj <<EOF
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net9.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Serilog.Sinks.File" Version="5.0.0" />
  </ItemGroup>
</Project>
EOF

# Run dotnet restore
dotnet restore

# Inspect project.assets.json
cat obj/project.assets.json | jq .libraries
```

## Common Issues

### Test Fails: Package Cache Hit

**Problem**: Test expects network call but package is cached

**Solution**: Use `noCache: true` or `force: true` in restore options:

```csharp
var result = GonugetBridge.RestoreTransitive(project.Path, noCache: true);
```

### Test Fails: Lock File Comparison Mismatch

**Problem**: JSON key order differs between gonuget and NuGet.Client

**Solution**: Use semantic comparison (already implemented in CompareProjectAssets)

### Test Flaky: Network Timeout

**Problem**: nuget.org is slow or unreliable

**Solution**: Mark test with [Trait("Category", "Network")] and run with package cache pre-populated:

```csharp
[Fact]
[Trait("Category", "Network")]
public void Test_RequiresNetwork() { ... }
```

## Performance Tips

### Minimize Test Execution Time

1. **Use Package Cache**: Tests hit cache after first run (much faster)
2. **Parallelize Tests**: xUnit runs tests in parallel by default
3. **Minimize PackageReferences**: Use minimal test projects (1-2 packages for simple tests)
4. **Clean Up Temp Directories**: Dispose test projects promptly

### Optimize for CI

```bash
# Pre-populate package cache before running tests
dotnet restore tests/nuget-client-interop/GonugetInterop.Tests/GonugetInterop.Tests.csproj

# Run tests with parallelization
dotnet test --parallel
```

## Test Coverage

### Verify Coverage

```bash
# Run tests with coverage
cd tests/nuget-client-interop/GonugetInterop.Tests
dotnet test --collect:"XPlat Code Coverage"

# Check coverage report
# Target: 90% of restore/restorer.go transitive resolution logic
```

### Coverage Targets

- **Transitive graph walking**: 100% (walker.Walk with recursive=true)
- **Direct vs transitive categorization**: 100%
- **Unresolved package handling**: 90%+
- **Lock file builder**: 90%+ (Libraries, ProjectFileDependencyGroups)

## Next Steps

- **Read spec.md**: Understand feature requirements and success criteria
- **Read data-model.md**: Learn data structures used in tests
- **Read contracts/restore-interop.json**: Understand JSON-RPC API
- **Run `/speckit.tasks`**: Generate implementation tasks for Phase 2

## Resources

- **NuGet.Client Source**: `/Users/brandon/src/NuGet.Client`
- **Existing Interop Tests**: `tests/nuget-client-interop/GonugetInterop.Tests/ResolverAdvancedTests.cs`
- **GonugetBridge**: `tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/GonugetBridge.cs`
- **Restore Implementation**: `restore/restorer.go`
- **Lock File Builder**: `restore/lock_file_builder.go`
