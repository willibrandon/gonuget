# CLI Output Formatting Specification

**Status**: BLOCKING - Required for production
**Priority**: Critical
**Reference**: Exact parity with `dotnet restore` output

---

## Executive Summary

gonuget CLI output MUST match `dotnet restore` output exactly (100% parity). This includes:
- Normal verbosity output (default)
- All verbosity levels (minimal, normal, detailed, diagnostic)
- Error message formatting
- Whitespace, indentation, and punctuation

This document specifies the required output format based on testing against dotnet 9.0.306.

**Current Status**: 🟡 PARTIAL - Basic output works, missing exact format match and verbosity modes

---

## Test Results: dotnet vs gonuget

### Test Setup
```bash
# Created test project with Serilog.Sinks.File 5.0.0
# This package has 1 transitive dependency: Serilog 2.12.0
cd /tmp/test-dotnet-restore/TestRestore
dotnet add package Serilog.Sinks.File --version 5.0.0
```

### First Run (Clean Restore)

**dotnet output:**
```
  Determining projects to restore...
  Restored /private/tmp/test-dotnet-restore/TestRestore/TestRestore.csproj (in 78 ms).
```

**gonuget output (CURRENT - WRONG):**
```
Restoring packages for /tmp/test-dotnet-restore/TestRestore/TestRestore.csproj...
  Restored /tmp/test-dotnet-restore/TestRestore/TestRestore.csproj (in 2 ms)
```

**Issues**:
1. ❌ First line format differs: "Determining projects to restore..." vs "Restoring packages for [path]..."
2. ❌ First line indentation differs: 2 spaces vs no indent
3. ❌ Missing period after time: "(in 78 ms)." vs "(in 2 ms)"

### Cached Restore (No-Op)

**dotnet output:**
```
  Determining projects to restore...
  All projects are up-to-date for restore.
```

**gonuget output (CURRENT - MATCHES):**
```
Restoring packages for /tmp/test-dotnet-restore/TestRestore/TestRestore.csproj...
  All projects are up-to-date for restore.
```

**Issues**:
1. ❌ First line format differs (same issue as above)
2. ✅ Second line matches exactly

---

## Required Output Format

### Normal Verbosity (Default)

#### First Run (Packages Downloaded)
```
  Determining projects to restore...
  Restored <PROJECT_PATH> (in <TIME> ms).
```

#### Cached Run (No Downloads)
```
  Determining projects to restore...
  All projects are up-to-date for restore.
```

### Format Rules

1. **Line 1**: `"  Determining projects to restore..."`
   - Always shown
   - 2-space indent
   - Ends with `...`
   - No project path

2. **Line 2 (first run)**: `"  Restored <PATH> (in <TIME> ms)."`
   - 2-space indent
   - Uses absolute project path
   - Time in milliseconds
   - **CRITICAL**: Period after closing parenthesis

3. **Line 2 (cached)**: `"  All projects are up-to-date for restore."`
   - 2-space indent
   - Fixed message
   - Period at end

---

## Current gonuget Implementation Issues

**File**: `restore/run.go` (lines 21-76)

**Current code** (lines 31-32):
```go
console.Printf("Restoring packages for %s...\n", projectPath)
```

**Should be**:
```go
console.Printf("  Determining projects to restore...\n")
```

**Current code** (lines 67-72):
```go
elapsed := time.Since(start)
if result.CacheHit {
    console.Printf("  All projects are up-to-date for restore.\n")
} else {
    console.Printf("  Restored %s (in %d ms)\n", projectPath, elapsed.Milliseconds())
}
```

**Should be**:
```go
elapsed := time.Since(start)
if result.CacheHit {
    console.Printf("  All projects are up-to-date for restore.\n")
} else {
    console.Printf("  Restored %s (in %d ms).\n", projectPath, elapsed.Milliseconds()) // Add period!
}
```

---

## Missing Features (BLOCKING)

### 1. No Direct vs Transitive Package Indication

**Issue**: gonuget does not show which packages were downloaded or distinguish between direct and transitive packages.

**dotnet verbose output** (not shown in normal mode):
- Does NOT show individual package downloads in normal mode
- Verbose mode (`-v detailed`) shows detailed MSBuild and package resolution information

**Required for gonuget**:
- Normal mode: Match dotnet exactly (no package listing)
- Verbose mode: Match dotnet exactly (100% parity with dotnet verbose output)

**Current Status**: ❌ Not implemented

### 2. No Verbose Mode

**Issue**: gonuget has no verbose output mode.

**dotnet verbose** (`-v detailed`):
- Shows MSBuild internals
- Shows package resolution details
- Shows download progress

**Required for gonuget**:
- `--verbosity minimal` - Match dotnet minimal output exactly
- `--verbosity normal` - Match dotnet normal output exactly (default)
- `--verbosity detailed` - Match dotnet detailed output exactly
- `--verbosity diagnostic` - Match dotnet diagnostic output exactly

**Current Status**: ❌ Not implemented

### 3. No Error Message Formatting

**Issue**: gonuget error messages don't match dotnet format.

**dotnet error output** (for unresolved packages):
```
/path/to/project.csproj : error NU1101: Unable to find package 'NonExistent'. No packages exist with this id in source(s): nuget.org
```

**Required format**:
```
<PROJECT_PATH> : error <ERROR_CODE>: <MESSAGE>
```

**Current gonuget**: Returns Go errors, not NuGet error codes

**Current Status**: ❌ Not implemented

---

## Implementation Plan

### Phase 1: Fix Basic Output Format (IMMEDIATE)

**Files to modify**:
- `restore/run.go` (lines 31, 72)

**Changes**:
1. Change line 31 from `"Restoring packages for %s..."` to `"  Determining projects to restore..."`
2. Add period to line 72: `"(in %d ms).\n"`

**Testing**:
```bash
# Should match dotnet exactly
cd /tmp/test-dotnet-restore/TestRestore
rm -rf obj bin

# Test first run
gonuget restore > gonuget-out.txt
dotnet restore > dotnet-out.txt
diff gonuget-out.txt dotnet-out.txt  # Should be empty

# Test cached run
gonuget restore > gonuget-cached.txt
dotnet restore > dotnet-cached.txt
diff gonuget-cached.txt dotnet-cached.txt  # Should be empty
```

**Estimated time**: 5 minutes

### Phase 2: Add Verbose Mode (BLOCKING)

**Files to modify**:
- `restore/options.go` - Add `Verbosity` field
- `restore/run.go` - Check verbosity level
- `restore/restorer.go` - Add console output for package downloads

**Requirement**: gonuget verbose output MUST match dotnet verbose output exactly (100% parity).

**Testing**:
```bash
# Test each verbosity level
cd /tmp/test-dotnet-restore/TestRestore
rm -rf obj bin

# Test minimal verbosity
dotnet restore --verbosity minimal > dotnet-minimal.txt 2>&1
gonuget restore --verbosity minimal > gonuget-minimal.txt 2>&1
diff dotnet-minimal.txt gonuget-minimal.txt  # Must be empty

# Test normal verbosity (default)
dotnet restore --verbosity normal > dotnet-normal.txt 2>&1
gonuget restore --verbosity normal > gonuget-normal.txt 2>&1
diff dotnet-normal.txt gonuget-normal.txt  # Must be empty

# Test detailed verbosity
dotnet restore --verbosity detailed > dotnet-detailed.txt 2>&1
gonuget restore --verbosity detailed > gonuget-detailed.txt 2>&1
diff dotnet-detailed.txt gonuget-detailed.txt  # Must be empty

# Test diagnostic verbosity
dotnet restore --verbosity diagnostic > dotnet-diagnostic.txt 2>&1
gonuget restore --verbosity diagnostic > gonuget-diagnostic.txt 2>&1
diff dotnet-diagnostic.txt gonuget-diagnostic.txt  # Must be empty
```

**Estimated time**: TBD - requires analyzing exact dotnet output format

### Phase 3: Add Error Formatting (HIGH PRIORITY)

**Files to modify**:
- `restore/run.go` - Format errors with NuGet error codes
- `restore/restorer.go` - Return structured errors

**Example**:
```go
if len(unresolvedPackages) > 0 {
    for _, unresolved := range unresolvedPackages {
        console.Error("%s : error %s: %s\n",
            projectPath,
            unresolved.ErrorCode,
            unresolved.Message)
    }
    return fmt.Errorf("restore failed")
}
```

**Estimated time**: 2 hours

---

## Verification Test Cases

### Test 1: Basic Output Format

```bash
# Setup
cd /tmp && rm -rf test-output && mkdir test-output && cd test-output
dotnet new console -n OutputTest
cd OutputTest
dotnet add package Newtonsoft.Json --version 13.0.3

# Test gonuget
rm -rf obj bin
gonuget restore > gonuget.txt 2>&1
dotnet restore > dotnet.txt 2>&1

# Verify
diff gonuget.txt dotnet.txt
# Expected: No differences
```

### Test 2: Cached Restore

```bash
# After Test 1
gonuget restore > gonuget-cached.txt 2>&1
dotnet restore > dotnet-cached.txt 2>&1

# Verify
diff gonuget-cached.txt dotnet-cached.txt
# Expected: No differences
```

### Test 3: Error Messages

```bash
# Create project with non-existent package
cat > test.csproj <<'EOF'
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net9.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="NonExistentPackage999999" Version="1.0.0" />
  </ItemGroup>
</Project>
EOF

# Test error output
gonuget restore > gonuget-error.txt 2>&1
dotnet restore > dotnet-error.txt 2>&1

# Verify error format matches
grep "error NU1101" gonuget-error.txt
grep "error NU1101" dotnet-error.txt
# Both should contain NU1101 error
```

---

## Success Criteria

### Phase 1 (BLOCKING)
- [x] First line: "  Determining projects to restore..."
- [ ] Second line (first run): "  Restored <PATH> (in X ms)." with period
- [x] Second line (cached): "  All projects are up-to-date for restore."
- [ ] Output matches `dotnet restore` byte-for-byte

### Phase 2 (BLOCKING)
- [ ] `--verbosity minimal` flag implemented - output matches dotnet 100%
- [ ] `--verbosity normal` flag implemented (default) - output matches dotnet 100%
- [ ] `--verbosity detailed` flag implemented - output matches dotnet 100%
- [ ] `--verbosity diagnostic` flag implemented - output matches dotnet 100%
- [ ] All verbosity levels verified with diff tests (must be byte-for-byte identical)

### Phase 3 (BLOCKING)
- [ ] Error messages use NuGet error codes (NU1101, NU1102, NU1103)
- [ ] Error format matches: `<PROJECT> : error <CODE>: <MESSAGE>`
- [ ] Unresolved package errors show all packages
- [ ] Error messages include source URLs

---

## References

**NuGet.Client Source**:
- `RestoreCommand.cs` (line 823-871) - Output formatting
- `MSBuildLogger.cs` - Error message formatting
- `RestoreSummary.cs` - Summary output

**gonuget Files**:
- `restore/run.go` - Main restore entry point
- `restore/console.go` - Console interface
- `cmd/gonuget/commands/restore.go` - CLI command

**Testing**:
- dotnet SDK 9.0.306
- Test project: Serilog.Sinks.File 5.0.0 (1 transitive dependency)

---

## Notes

1. **Whitespace is critical** - dotnet uses 2-space indentation consistently
2. **Punctuation matters** - Period after time "(in X ms)." not "(in X ms)"
3. **Path format** - dotnet uses absolute paths (macOS resolves symlinks)
4. **Time precision** - dotnet uses milliseconds, no decimal places
5. **Error codes are mandatory** - Users rely on NU1101/NU1102/NU1103 for diagnostics
