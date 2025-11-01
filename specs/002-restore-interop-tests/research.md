# Research: C# Interop Tests for Restore Transitive Dependencies

**Date**: 2025-11-01
**Status**: Complete

## Purpose

Research existing interop test patterns, NuGet.Client error message formats, project.assets.json comparison strategies, and test infrastructure to guide implementation of comprehensive restore transitive dependency tests.

## Research Findings

### 1. Existing Interop Test Patterns

**Investigation**: Analyzed existing 491 interop tests to identify proven patterns for testing gonuget vs NuGet.Client parity.

**Key Files Reviewed**:
- `tests/nuget-client-interop/GonugetInterop.Tests/ResolverAdvancedTests.cs` - Complex multi-package scenarios
- `tests/nuget-client-interop/GonugetInterop.Tests/RestoreTests.cs` - Basic restore operations
- `tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/GonugetBridge.cs` - JSON-RPC bridge
- `cmd/nuget-interop-test/handlers_restore.go` - Existing restore handlers

**Findings**:
- Tests follow consistent pattern: Create test data → Call GonugetBridge → Compare with NuGet.Client → Assert parity
- `ResolverAdvancedTests.cs` uses `InMemoryDependencyProvider` for controlled package metadata (avoids network calls)
- Existing restore handlers (`RestoreDirectDependenciesHandler`, `ParseLockFileHandler`) provide foundation
- Tests use xUnit `[Fact]` and `[Theory]` attributes with descriptive names

**Decision**: Follow `ResolverAdvancedTests.cs` pattern for transitive resolution tests. Create test projects programmatically, use GonugetBridge for execution, compare results semantically.

**Alternatives Considered**:
- Use real GitHub projects: Rejected (external dependencies, network flakiness, version drift)
- Mock NuGet.Client: Rejected (defeats purpose of parity validation)

---

### 2. NuGet.Client Error Message Format

**Investigation**: Analyzed NuGet.Client source code to understand exact error message formatting for NU1101, NU1102, NU1103.

**Reference Files** (from `/Users/brandon/src/NuGet.Client`):
- `src/NuGet.Core/NuGet.Commands/RestoreCommand/UnresolvedMessages.cs:GetMessagesAsync()`
- `src/NuGet.Core/NuGet.Commands/RestoreCommand/Logging/RestoreLogMessages.cs`

**NU1101 Format** (Package not found):
```
NU1101: Unable to find package '{packageId}'. No packages exist with this id in source(s): {sources}
```

**NU1102 Format** (Version not found):
```
NU1102: Unable to find package '{packageId}' with version ({versionRange})
  - Found {count} version(s) in {source} [ Nearest version: {nearestVersion} ]
```

**NU1103 Format** (Prerelease only):
```
NU1103: Unable to find a stable package '{packageId}' with version ({versionRange})
  - Found {count} version(s) in {source} [ Nearest version: {nearestVersion} ]
  - This package may be released as a pre-release. To allow downloads of pre-release packages, select 'Include prerelease' when searching...
```

**Findings**:
- Error codes are prefixes to messages (NU1101, NU1102, NU1103)
- Messages include package ID, version range, source list, available versions
- Formatting includes specific punctuation and capitalization

**Decision**: Use exact string matching for error code and core message. Allow tolerance for minor formatting (line endings, spacing). Extract error code, package ID, version range, sources list as separate assertions.

**Rationale**: Error messages are user-facing. Exact parity ensures consistent UX. Tolerant comparison prevents brittle tests from whitespace differences.

---

### 3. project.assets.json Comparison Strategy

**Investigation**: Analyzed project.assets.json format and existing lock file parsing to determine comparison approach.

**Reference Files**:
- `cmd/nuget-interop-test/handlers_restore.go:ParseLockFileHandler` - Existing parser
- `restore/lock_file_builder.go` - Lock file generation
- `restore/lock_file_builder_test.go:232-394` - Lock file validation tests

**project.assets.json Key Sections**:
```json
{
  "version": 3,
  "targets": { "<framework>": { "<package>/<version>": {...} } },
  "libraries": { "<package>/<version>": { "type": "package", "path": "lowercase/path" } },
  "projectFileDependencyGroups": { "<framework>": ["Package >= Version"] },
  "packageFolders": { "/path/to/packages/": {} },
  "project": { "version": "1.0.0", "restore": {...}, "frameworks": {...} }
}
```

**Findings**:
- Existing `ParseLockFileHandler` deserializes entire lock file to Go struct
- `Libraries` map keys must be lowercase package IDs (verified by test at line 232-394)
- `ProjectFileDependencyGroups` contains ONLY direct dependencies (not transitive)
- `targets` section contains both direct and transitive packages
- JSON key order may vary between implementations

**Decision**: Deserialize project.assets.json to object graph and compare semantically:
1. Compare `Libraries` map keys (must match exactly, including lowercase paths)
2. Compare `ProjectFileDependencyGroups` contents (direct deps only)
3. Compare package versions in `Libraries` entries
4. Validate `targets` section structure (framework-specific)

**Rationale**: Semantic comparison is robust to key ordering differences. Validates MSBuild-critical fields (Libraries, ProjectFileDependencyGroups) without false positives from formatting.

**Alternatives Considered**:
- String comparison: Rejected (brittle, fails on key reordering)
- Byte-for-byte comparison: Rejected (too strict, whitespace sensitivity)

---

### 4. Test Project Generation

**Investigation**: Determined optimal approach for creating test .csproj files for restore validation.

**Findings**:
- Minimal .csproj format:
  ```xml
  <Project Sdk="Microsoft.NET.Sdk">
    <PropertyGroup>
      <TargetFramework>net9.0</TargetFramework>
    </PropertyGroup>
    <ItemGroup>
      <PackageReference Include="Newtonsoft.Json" Version="13.0.4" />
    </ItemGroup>
  </Project>
  ```
- Can create in temp directories, no build required for restore testing
- Framework can be parameterized (net6.0, net8.0, net9.0)
- PackageReferences can be added programmatically

**Decision**: Create minimal .csproj files in temp directories with programmatic PackageReference generation. Create TestProject helper class with fluent API:
```csharp
var project = TestProject.Create("TestApp")
    .WithFramework("net9.0")
    .AddPackage("Newtonsoft.Json", "13.0.4")
    .AddPackage("Serilog.Sinks.File", "5.0.0")
    .Build();
```

**Rationale**: Lightweight, fast, full control over test scenarios, no external dependencies.

**Alternatives Considered**:
- Use dotnet new: Rejected (creates unnecessary files, slower)
- Hardcode .csproj strings: Rejected (unmaintainable, inflexible)

---

### 5. GonugetBridge Extension Strategy

**Investigation**: Analyzed existing GonugetBridge methods to determine extension pattern.

**Reference**: `tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/GonugetBridge.cs`

**Existing Methods Pattern**:
```csharp
public static CompareVersionsResponse CompareVersions(string version1, string version2)
{
    var request = new { action = "compare_versions", data = new { version1, version2 } };
    return Execute<CompareVersionsResponse>(request);
}
```

**Findings**:
- Static methods with descriptive names
- Request format: `{ action: "...", data: {...} }`
- Type-safe response objects
- Execute<T>() handles JSON-RPC communication

**Decision**: Add new static methods following existing pattern:
- `RestoreTransitive(string projectPath, ...options)` - Full transitive restore
- `CompareProjectAssets(string gonugetLockFile, string nugetLockFile)` - Semantic comparison
- `ValidateErrorMessages(string gonugetError, string nugetError)` - Error format validation

**Rationale**: Maintains consistency with 491 existing tests, proven architecture, type-safe API.

**Alternatives Considered**:
- Create separate BridgeV2 class: Rejected (unnecessary abstraction, breaks from pattern)
- Use dynamic typing: Rejected (loses type safety, harder to maintain)

---

## Recommendations

### Test Implementation Approach

1. **Create TestProject Helper** (`tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/TestProject.cs`)
   - Fluent API for building .csproj files
   - Support for multi-framework projects
   - Automatic temp directory management

2. **Extend GonugetBridge** (`GonugetBridge.cs`)
   - Add `RestoreTransitive()` method
   - Add `CompareProjectAssets()` method
   - Add `ValidateErrorMessages()` method

3. **Add Go Handlers** (`cmd/nuget-interop-test/handlers_restore.go`)
   - `RestoreTransitiveHandler` - Full transitive restore with categorization
   - `CompareProjectAssetsHandler` - Semantic lock file comparison
   - `ValidateErrorMessagesHandler` - Error message format validation

4. **Create Test File** (`RestoreTransitiveTests.cs`)
   - 20-30 test methods organized by category
   - Use [Theory] with InlineData for parameterized tests
   - Follow existing naming convention: `Test_{Feature}_{Scenario}_Parity()`

### Test Organization

```
Transitive Resolution Parity (8-10 tests)
├── Simple transitive (Newtonsoft.Json → no deps)
├── Moderate transitive (Serilog.Sinks.File → Serilog)
├── Complex transitive (ASP.NET Core packages, 10+ deps)
├── Diamond dependencies (multiple paths to same package)
└── Framework-specific deps (net6.0 vs net8.0 vs net9.0)

Direct vs Transitive Categorization (5-7 tests)
├── Pure direct (no transitive)
├── Pure transitive (only pulled by direct)
├── Mixed (package is both direct and transitive)
└── ProjectFileDependencyGroups validation

Unresolved Package Error Messages (5-7 tests)
├── NU1101 (package doesn't exist)
├── NU1102 (version doesn't exist)
├── NU1103 (prerelease only)
└── Error format matching

Lock File Format Compatibility (5-7 tests)
├── Libraries map structure
├── ProjectFileDependencyGroups (direct only)
├── Multi-framework projects
└── MSBuild compatibility (dotnet build succeeds)
```

### Performance Considerations

- Use package cache to minimize network calls (tests should hit cache after first run)
- Parallelize test execution with xUnit (tests are independent)
- Clean up temp directories in test Dispose() or finally blocks
- Target <2 minutes total execution time (per SC-005)

### Risk Mitigations

1. **Network Flakiness**: Use cached packages, mark tests that require network with [Trait("Category", "Network")]
2. **NuGet.Client Version Drift**: Pin NuGet.Client version in test .csproj, document upgrade process
3. **Test Execution Time**: Optimize test project sizes (minimal PackageReferences), use parallelization
4. **False Positives**: Semantic comparison (not string), focus on MSBuild-critical fields

## Conclusion

Research complete. All technical unknowns resolved. Ready to proceed with Phase 1 (data model, contracts, quickstart) and Phase 2 (implementation tasks).

**Key Decisions**:
1. Follow ResolverAdvancedTests.cs pattern for test structure
2. Use exact string matching for error codes with tolerance for formatting
3. Deserialize and semantically compare project.assets.json
4. Create minimal .csproj files programmatically with TestProject helper
5. Extend GonugetBridge following existing static method pattern

**No remaining NEEDS CLARIFICATION items**.
