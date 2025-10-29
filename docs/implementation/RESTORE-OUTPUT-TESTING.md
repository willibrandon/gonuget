# Restore Output Testing Guide

This document provides comprehensive testing commands for validating gonuget restore output against dotnet restore across all verbosity levels and scenarios.

## Table of Contents

- [Overview](#overview)
- [Verbosity Levels](#verbosity-levels)
- [Terminal Logger Behavior](#terminal-logger-behavior)
- [Testing Strategy](#testing-strategy)
- [Test Scenarios](#test-scenarios)
- [Cache Management](#cache-management)
- [Complete Test Script](#complete-test-script)

## Overview

gonuget restore output must match dotnet restore behavior exactly. This includes:
- Success/error messages
- Timing format (milliseconds vs seconds)
- ANSI color codes (green for success, red for errors)
- Verbosity-appropriate message filtering

## Verbosity Levels

| Level | Alias | Output |
|-------|-------|--------|
| `quiet` | `q` | Minimal output, errors only |
| `minimal` | `m` | Standard output with restore summary and success message (default) |
| `normal` | `n` | Same as minimal for restore operations |
| `detailed` | `d` | Adds "Determining projects to restore..." and per-project timing breakdown (clean Terminal Logger output) |
| `diagnostic` | `diag` | Shows all download messages (Acquiring lock, GET, OK, CACHE) plus detailed breakdown |

## Terminal Logger Behavior

**Critical Understanding**: .NET 8+ uses Terminal Logger by default, which provides clean output:

### Terminal Logger (default in .NET 8+)
- **Enabled when**: Output goes directly to terminal (TTY)
- **Behavior**: Clean output, hides MSBuild internals even in `detailed` mode
- **This is what users see**: `dotnet restore -v:detailed`

### Classic Console Logger (legacy)
- **Enabled when**:
  - Output is redirected/piped (`dotnet restore 2>&1 | head`)
  - Explicitly disabled with `--tl:off`
- **Behavior**: Shows full MSBuild internals in `detailed` and `diagnostic` modes
- **Use for diagnostic comparison**: `dotnet restore -v:diagnostic --tl:off`

### What This Means for Testing

1. **For minimal/normal/detailed**: Compare against Terminal Logger (no redirection)
2. **For diagnostic**: Use `--tl:off` to get comparable download messages
3. **gonuget always matches Terminal Logger behavior** (clean output is the modern standard)

## Testing Strategy

### Phase 1: Quick Validation (Cached Packages)
Test all verbosity levels with cached packages for fast feedback on formatting:

```bash
# Normal
./gonuget restore tests/test-scenarios/simple/test.csproj
dotnet restore tests/test-scenarios/simple/test.csproj

# Piped output
./gonuget restore tests/test-scenarios/simple/test.csproj 2>&1 | cat
dotnet restore tests/test-scenarios/simple/test.csproj 2>&1 | cat

# LGTM ✅

# Detailed
./gonuget restore tests/test-scenarios/simple/test.csproj -v:detailed
dotnet restore tests/test-scenarios/simple/test.csproj -v:detailed

# Piped output
./gonuget restore tests/test-scenarios/simple/test.csproj -v:detailed 2>&1 | cat
dotnet restore tests/test-scenarios/simple/test.csproj -v:detailed 2>&1 | cat

# LGTM ✅

# Quiet
./gonuget restore tests/test-scenarios/simple/test.csproj -v:quiet
dotnet restore tests/test-scenarios/simple/test.csproj -v:quiet

# Piped output
./gonuget restore tests/test-scenarios/simple/test.csproj -v:quiet 2>&1 | cat
dotnet restore tests/test-scenarios/simple/test.csproj -v:quiet 2>&1 | cat

# LGTM ✅

```

### Phase 2: Diagnostic Testing (Selective Cache Clear)
Test download messages with selective package clearing:

```bash
# Clear just Newtonsoft.Json to see download messages
rm -rf ~/.nuget/packages/newtonsoft.json

./gonuget restore tests/test-scenarios/simple/test.csproj -v:diagnostic
dotnet restore tests/test-scenarios/simple/test.csproj -v:diagnostic --tl:off

# Piped output
./gonuget restore tests/test-scenarios/simple/test.csproj -v:diagnostic 2>&1 | cat
dotnet restore tests/test-scenarios/simple/test.csproj -v:diagnostic 2>&1 | cat
```

### Phase 3: Full Cache Clear (Complete Flow)
Test complete restore flow with all packages fresh:

```bash
rm -rf ~/.nuget/packages

./gonuget restore tests/test-scenarios/complex/test.csproj -v:diagnostic
dotnet restore tests/test-scenarios/complex/test.csproj -v:diagnostic --tl:off

# Piped output
./gonuget restore tests/test-scenarios/complex/test.csproj -v:diagnostic 2>&1 | cat
dotnet restore tests/test-scenarios/complex/test.csproj -v:diagnostic 2>&1 | cat
```

## Test Scenarios

All test scenarios are located in `tests/test-scenarios/` within the gonuget repository.

### Success Scenarios

1. **Simple (2 packages)**: `tests/test-scenarios/simple/test.csproj`
   - Newtonsoft.Json 13.0.3
   - Serilog 3.1.1

2. **Multi-targeting (net6.0 + net8.0)**: `tests/test-scenarios/multitarget/test.csproj`
   - Tests framework-specific dependency resolution

3. **Complex (7 packages with deep dependencies)**: `tests/test-scenarios/complex/test.csproj`
   - Multiple transitive dependencies
   - Tests dependency resolution performance

### Error Scenarios

1. **NU1101 - Package not found**: `tests/test-scenarios/nu1101/test.csproj`
   - Package ID doesn't exist

2. **NU1102 - Version not found**: `tests/test-scenarios/nu1102/test.csproj`
   - Package exists, but requested version doesn't

3. **NU1103 - Only prerelease available**: `tests/test-scenarios/nu1103/test.csproj`
   - Stable version requested (1.1.4), only prerelease exists (1.1.4-alpha)

## Cache Management

### Clear Global Package Cache
```bash
rm -rf ~/.nuget/packages
```

### Clear Specific Package
```bash
rm -rf ~/.nuget/packages/newtonsoft.json
rm -rf ~/.nuget/packages/serilog
```

### Clear Project Build Artifacts
```bash
rm -rf tests/test-scenarios/simple/obj tests/test-scenarios/simple/bin
rm -rf tests/test-scenarios/multitarget/obj tests/test-scenarios/multitarget/bin
rm -rf tests/test-scenarios/complex/obj tests/test-scenarios/complex/bin
```

## Complete Test Script

### Normal/Minimal Verbosity (Default)

```bash
# ============================================
# NORMAL VERBOSITY (default)
# ============================================

# Scenario 1 - Simple (2 packages)
./gonuget restore tests/test-scenarios/simple/test.csproj
dotnet restore tests/test-scenarios/simple/test.csproj

# Piped output
./gonuget restore tests/test-scenarios/simple/test.csproj 2>&1 | cat
dotnet restore tests/test-scenarios/simple/test.csproj 2>&1 | cat

# Scenario 2 - Multi-targeting (net6.0 + net8.0)
./gonuget restore tests/test-scenarios/multitarget/test.csproj
dotnet restore tests/test-scenarios/multitarget/test.csproj

# Piped output
./gonuget restore tests/test-scenarios/multitarget/test.csproj 2>&1 | cat
dotnet restore tests/test-scenarios/multitarget/test.csproj 2>&1 | cat

# Scenario 3 - Complex (7 packages with deep dependencies)
./gonuget restore tests/test-scenarios/complex/test.csproj
dotnet restore tests/test-scenarios/complex/test.csproj

# Piped output
./gonuget restore tests/test-scenarios/complex/test.csproj 2>&1 | cat
dotnet restore tests/test-scenarios/complex/test.csproj 2>&1 | cat

# Error scenarios
# NU1101 - Package not found
./gonuget restore tests/test-scenarios/nu1101/test.csproj
dotnet restore tests/test-scenarios/nu1101/test.csproj

# Piped output
./gonuget restore tests/test-scenarios/nu1101/test.csproj 2>&1 | cat
dotnet restore tests/test-scenarios/nu1101/test.csproj 2>&1 | cat

# NU1102 - Version not found
./gonuget restore tests/test-scenarios/nu1102/test.csproj
dotnet restore tests/test-scenarios/nu1102/test.csproj

# Piped output
./gonuget restore tests/test-scenarios/nu1102/test.csproj 2>&1 | cat
dotnet restore tests/test-scenarios/nu1102/test.csproj 2>&1 | cat

# NU1103 - Only prerelease available
./gonuget restore tests/test-scenarios/nu1103/test.csproj
dotnet restore tests/test-scenarios/nu1103/test.csproj

# Piped output
./gonuget restore tests/test-scenarios/nu1103/test.csproj 2>&1 | cat
dotnet restore tests/test-scenarios/nu1103/test.csproj 2>&1 | cat
```

### Detailed Verbosity (Terminal Logger - Clean Output)

```bash
# ============================================
# DETAILED VERBOSITY (Terminal Logger)
# ============================================
# Note: Run without redirection to see Terminal Logger clean output
# Expected: "Restore complete", "Determining projects", "Restored /path (in X ms)", "Restore succeeded"

# Scenario 1 - Simple (2 packages) - DETAILED
./gonuget restore tests/test-scenarios/simple/test.csproj -v:detailed
dotnet restore tests/test-scenarios/simple/test.csproj -v:detailed

# Piped output
./gonuget restore tests/test-scenarios/simple/test.csproj -v:detailed 2>&1 | cat
dotnet restore tests/test-scenarios/simple/test.csproj -v:detailed 2>&1 | cat

# Scenario 2 - Multi-targeting (net6.0 + net8.0) - DETAILED
./gonuget restore tests/test-scenarios/multitarget/test.csproj -v:detailed
dotnet restore tests/test-scenarios/multitarget/test.csproj -v:detailed

# Piped output
./gonuget restore tests/test-scenarios/multitarget/test.csproj -v:detailed 2>&1 | cat
dotnet restore tests/test-scenarios/multitarget/test.csproj -v:detailed 2>&1 | cat

# Scenario 3 - Complex (7 packages with deep dependencies) - DETAILED
./gonuget restore tests/test-scenarios/complex/test.csproj -v:detailed
dotnet restore tests/test-scenarios/complex/test.csproj -v:detailed

# Piped output
./gonuget restore tests/test-scenarios/complex/test.csproj -v:detailed 2>&1 | cat
dotnet restore tests/test-scenarios/complex/test.csproj -v:detailed 2>&1 | cat

# Error scenarios - DETAILED
# NU1101 - Package not found - DETAILED
./gonuget restore tests/test-scenarios/nu1101/test.csproj -v:detailed
dotnet restore tests/test-scenarios/nu1101/test.csproj -v:detailed

# Piped output
./gonuget restore tests/test-scenarios/nu1101/test.csproj -v:detailed 2>&1 | cat
dotnet restore tests/test-scenarios/nu1101/test.csproj -v:detailed 2>&1 | cat

# NU1102 - Version not found - DETAILED
./gonuget restore tests/test-scenarios/nu1102/test.csproj -v:detailed
dotnet restore tests/test-scenarios/nu1102/test.csproj -v:detailed

# Piped output
./gonuget restore tests/test-scenarios/nu1102/test.csproj -v:detailed 2>&1 | cat
dotnet restore tests/test-scenarios/nu1102/test.csproj -v:detailed 2>&1 | cat

# NU1103 - Only prerelease available - DETAILED
./gonuget restore tests/test-scenarios/nu1103/test.csproj -v:detailed
dotnet restore tests/test-scenarios/nu1103/test.csproj -v:detailed

# Piped output
./gonuget restore tests/test-scenarios/nu1103/test.csproj -v:detailed 2>&1 | cat
dotnet restore tests/test-scenarios/nu1103/test.csproj -v:detailed 2>&1 | cat
```

### Diagnostic Verbosity (Shows Download Messages)

```bash
# ============================================
# DIAGNOSTIC VERBOSITY (shows download messages)
# ============================================
# Note: Use --tl:off with dotnet to disable Terminal Logger and see download messages
# Expected: "Acquiring lock", "GET", "OK", "CACHE" messages plus detailed summary

# Clear cache for one scenario to see download messages
rm -rf ~/.nuget/packages/newtonsoft.json ~/.nuget/packages/serilog

# Scenario 1 - Simple (2 packages) - DIAGNOSTIC (with fresh download)
./gonuget restore tests/test-scenarios/simple/test.csproj -v:diagnostic
dotnet restore tests/test-scenarios/simple/test.csproj -v:diagnostic --tl:off 2>&1 | grep -E "(Acquiring|GET|OK|CACHE|Restore|Restored|succeeded|failed)" | head -20

# Piped output
./gonuget restore tests/test-scenarios/simple/test.csproj -v:diagnostic 2>&1 | cat
dotnet restore tests/test-scenarios/simple/test.csproj -v:diagnostic 2>&1 | cat

# Scenario 2 - Multi-targeting (net6.0 + net8.0) - DIAGNOSTIC
./gonuget restore tests/test-scenarios/multitarget/test.csproj -v:diagnostic
dotnet restore tests/test-scenarios/multitarget/test.csproj -v:diagnostic --tl:off 2>&1 | grep -E "(Acquiring|GET|OK|CACHE|Restore|Restored|succeeded|failed)" | head -30

# Piped output
./gonuget restore tests/test-scenarios/multitarget/test.csproj -v:diagnostic 2>&1 | cat
dotnet restore tests/test-scenarios/multitarget/test.csproj -v:diagnostic 2>&1 | cat

# Clear full cache for complex scenario
rm -rf ~/.nuget/packages

# Scenario 3 - Complex (7 packages with deep dependencies) - DIAGNOSTIC (with full fresh download)
./gonuget restore tests/test-scenarios/complex/test.csproj -v:diagnostic
dotnet restore tests/test-scenarios/complex/test.csproj -v:diagnostic --tl:off 2>&1 | grep -E "(Acquiring|GET|OK|CACHE|Restore|Restored|succeeded|failed)" | head -50

# Piped output
./gonuget restore tests/test-scenarios/complex/test.csproj -v:diagnostic 2>&1 | cat
dotnet restore tests/test-scenarios/complex/test.csproj -v:diagnostic 2>&1 | cat

# Error scenarios - DIAGNOSTIC
# NU1101 - Package not found - DIAGNOSTIC
./gonuget restore tests/test-scenarios/nu1101/test.csproj -v:diagnostic
dotnet restore tests/test-scenarios/nu1101/test.csproj -v:diagnostic --tl:off 2>&1 | grep -E "(NU1101|error|Restore|failed)" | head -20

# Piped output
./gonuget restore tests/test-scenarios/nu1101/test.csproj -v:diagnostic 2>&1 | cat
dotnet restore tests/test-scenarios/nu1101/test.csproj -v:diagnostic 2>&1 | cat

# NU1102 - Version not found - DIAGNOSTIC
./gonuget restore tests/test-scenarios/nu1102/test.csproj -v:diagnostic
dotnet restore tests/test-scenarios/nu1102/test.csproj -v:diagnostic --tl:off 2>&1 | grep -E "(NU1102|error|Restore|failed)" | head -20

# Piped output
./gonuget restore tests/test-scenarios/nu1102/test.csproj -v:diagnostic 2>&1 | cat
dotnet restore tests/test-scenarios/nu1102/test.csproj -v:diagnostic 2>&1 | cat

# NU1103 - Only prerelease available - DIAGNOSTIC
./gonuget restore tests/test-scenarios/nu1103/test.csproj -v:diagnostic
dotnet restore tests/test-scenarios/nu1103/test.csproj -v:diagnostic --tl:off 2>&1 | grep -E "(NU1103|error|Restore|failed)" | head -20

# Piped output
./gonuget restore tests/test-scenarios/nu1103/test.csproj -v:diagnostic 2>&1 | cat
dotnet restore tests/test-scenarios/nu1103/test.csproj -v:diagnostic 2>&1 | cat
```

### Quiet Verbosity

```bash
# ============================================
# QUIET VERBOSITY (minimal output, errors only)
# ============================================

# Scenario 1 - Simple (2 packages) - QUIET
./gonuget restore tests/test-scenarios/simple/test.csproj -v:quiet
dotnet restore tests/test-scenarios/simple/test.csproj -v:quiet

# Piped output
./gonuget restore tests/test-scenarios/simple/test.csproj -v:quiet 2>&1 | cat
dotnet restore tests/test-scenarios/simple/test.csproj -v:quiet 2>&1 | cat

# Scenario 2 - Multi-targeting (net6.0 + net8.0) - QUIET
./gonuget restore tests/test-scenarios/multitarget/test.csproj -v:quiet
dotnet restore tests/test-scenarios/multitarget/test.csproj -v:quiet

# Piped output
./gonuget restore tests/test-scenarios/multitarget/test.csproj -v:quiet 2>&1 | cat
dotnet restore tests/test-scenarios/multitarget/test.csproj -v:quiet 2>&1 | cat

# Scenario 3 - Complex (7 packages with deep dependencies) - QUIET
./gonuget restore tests/test-scenarios/complex/test.csproj -v:quiet
dotnet restore tests/test-scenarios/complex/test.csproj -v:quiet

# Piped output
./gonuget restore tests/test-scenarios/complex/test.csproj -v:quiet 2>&1 | cat
dotnet restore tests/test-scenarios/complex/test.csproj -v:quiet 2>&1 | cat

# Error scenarios - QUIET (should still show errors)
# NU1101 - Package not found - QUIET
./gonuget restore tests/test-scenarios/nu1101/test.csproj -v:quiet
dotnet restore tests/test-scenarios/nu1101/test.csproj -v:quiet

# Piped output
./gonuget restore tests/test-scenarios/nu1101/test.csproj -v:quiet 2>&1 | cat
dotnet restore tests/test-scenarios/nu1101/test.csproj -v:quiet 2>&1 | cat

# NU1102 - Version not found - QUIET
./gonuget restore tests/test-scenarios/nu1102/test.csproj -v:quiet
dotnet restore tests/test-scenarios/nu1102/test.csproj -v:quiet

# Piped output
./gonuget restore tests/test-scenarios/nu1102/test.csproj -v:quiet 2>&1 | cat
dotnet restore tests/test-scenarios/nu1102/test.csproj -v:quiet 2>&1 | cat

# NU1103 - Only prerelease available - QUIET
./gonuget restore tests/test-scenarios/nu1103/test.csproj -v:quiet
dotnet restore tests/test-scenarios/nu1103/test.csproj -v:quiet

# Piped output
./gonuget restore tests/test-scenarios/nu1103/test.csproj -v:quiet 2>&1 | cat
dotnet restore tests/test-scenarios/nu1103/test.csproj -v:quiet 2>&1 | cat
```

## Expected Output Examples

### Minimal/Normal Verbosity (Success)
```
Restore complete (0.0s)

Restore succeeded in 0.0s
```

### Detailed Verbosity (Success)
```
Restore complete (0.3s)
    Determining projects to restore...
    Restored tests/test-scenarios/simple/test.csproj (in 260 ms).

Restore succeeded in 0.3s
```

### Diagnostic Verbosity (Success with Downloads)
```
         Acquiring lock for the installation of Newtonsoft.Json 13.0.3
         Acquired lock for the installation of Newtonsoft.Json 13.0.3
           GET https://api.nuget.org/v3/flatcontainer/newtonsoft.json/13.0.3/newtonsoft.json.13.0.3.nupkg
           OK https://api.nuget.org/v3/flatcontainer/newtonsoft.json/13.0.3/newtonsoft.json.13.0.3.nupkg 120ms
           CACHE https://api.nuget.org/v3/vulnerabilities/index.json
Restore complete (0.3s)
    Determining projects to restore...
    Restored tests/test-scenarios/simple/test.csproj (in 260 ms).

Restore succeeded in 0.3s
```

### Error Output (NU1102)
```
tests/test-scenarios/nu1102/test.csproj : error NU1102: Unable to find package Newtonsoft.Json with version (>= 99.99.99)
tests/test-scenarios/nu1102/test.csproj : error NU1102:   - Found 326 version(s) in nuget.org [ Nearest version: 13.0.3 ]

Restore failed with 1 error(s) in 0.5s
```

## Validation Checklist

- [ ] All success scenarios show correct timing (ms when < 1s, s for total)
- [ ] "succeeded" appears in green in success messages
- [ ] "failed" and error codes appear in red in error messages
- [ ] Detailed mode shows clean output (no download spam)
- [ ] Diagnostic mode shows download messages (Acquiring, GET, OK, CACHE)
- [ ] Quiet mode shows minimal output
- [ ] Error messages match dotnet format exactly (NU1101, NU1102, NU1103)
- [ ] Multi-line error formatting matches dotnet (version lists, per-source info)
- [ ] Timing accuracy: restore operation time vs total time are both correct

## Notes

1. **Terminal Logger is the reference**: Modern dotnet uses Terminal Logger by default, so gonuget matches that behavior
2. **Diagnostic mode is for debugging**: Shows download messages that users don't normally see
3. **Timing precision matters**: Use `elapsed.Seconds()` not `elapsed.Milliseconds() / 1000.0` to avoid precision loss
4. **ANSI colors required**: Success messages use green (`\033[32m`), errors use bright red (`\033[1;31m`)
5. **Message ordering matters**: "Restore complete" before detailed breakdown, blank line before "Restore succeeded/failed"
