# Restore Diagnostic Output Implementation Guide

## Overview

This document describes the implementation of enhanced diagnostic output for `gonuget restore` operations. Unlike MSBuild which shows internal build system details, gonuget's diagnostic mode will provide **NuGet-specific insights** that help users understand dependency resolution, version selection, and package operations.

## Table of Contents

- [Rationale](#rationale)
- [Design Principles](#design-principles)
- [Output Tiers](#output-tiers)
- [Implementation Plan](#implementation-plan)
- [Technical Design](#technical-design)
- [Examples](#examples)
- [Testing Strategy](#testing-strategy)

## Rationale

### Why Enhanced Diagnostic Output?

**Problem**: When `dotnet restore -v:diagnostic` runs, it shows MSBuild internals that don't apply to gonuget:
- Property evaluations (`$(MSBuildExtensionsPath)`)
- SDK resolution
- Target execution
- Import chains

These messages are **meaningless for gonuget** which bypasses MSBuild entirely.

**Solution**: Provide diagnostic output that shows **NuGet operations gonuget actually performs**:
- Dependency resolution logic (why versions are selected)
- Package discovery and version availability
- Network operations and caching
- Framework compatibility decisions
- Performance breakdowns

### Value Proposition

Enhanced diagnostic output provides:

1. **Troubleshooting**: Users can see exactly why a version was selected or rejected
2. **Performance Analysis**: Identify slow operations (network, cache, resolution)
3. **Learning Tool**: Understand how NuGet dependency resolution works
4. **Debugging Aid**: Trace through complex dependency graphs
5. **Transparency**: See what gonuget is doing under the hood

## Design Principles

### 1. NuGet-Specific, Not MSBuild-Specific

Show operations gonuget **actually performs**:
- ✅ Package version resolution
- ✅ Dependency graph walking
- ✅ Network requests and cache hits
- ❌ MSBuild property evaluation
- ❌ SDK resolution
- ❌ Target execution

### 2. Actionable Information

Every diagnostic message should help users:
- **Understand** what happened
- **Debug** when things go wrong
- **Optimize** performance

### 3. Progressive Disclosure

Use verbosity levels appropriately:
- **quiet**: Errors only
- **minimal/normal**: Summary only
- **detailed**: Key operations (what dotnet shows with Terminal Logger)
- **diagnostic**: Everything (resolution trace, network, timing, caching)

### 4. Performance First

Diagnostic output must not impact performance:
- Use conditional compilation where possible
- Avoid string allocations for disabled messages
- Lazy evaluation of expensive formatting

## Output Tiers

### Tier 1: Dependency Resolution Trace (Highest Value)

Show the **why** behind version selection:

```
Resolving dependencies for net8.0:
  Newtonsoft.Json 13.0.3 (direct reference)
    Constraint: >= 13.0.3
    Available versions: 13.0.1, 13.0.2, 13.0.3, 13.0.4
    Selected: 13.0.3 (satisfies >=  13.0.3, highest matching)
    Framework: compatible with net8.0

  Serilog 3.1.1 (direct reference)
    Constraint: >= 3.1.1
    Available versions: 3.1.0, 3.1.1, 3.1.2
    Selected: 3.1.1 (satisfies >= 3.1.1, exact match)
    Framework: compatible with net8.0
    Dependencies:
      → Serilog.Sinks.File >= 5.0.0

  Serilog.Sinks.File (transitive via Serilog)
    Constraint: >= 5.0.0
    Available versions: 5.0.0, 5.0.1, 6.0.0
    Selected: 6.0.0 (satisfies >= 5.0.0, highest matching)
    Framework: compatible with net8.0
    Parent chain: Serilog → Serilog.Sinks.File

Dependency graph resolved:
  3 packages total (2 direct, 1 transitive)
  Max depth: 1 (Serilog.Sinks.File)
```

**Implementation Point**: Hook into `DependencyWalker.Walk()` and `fetchDependency()` to capture resolution decisions.

### Tier 2: Network and Cache Operations (Current Implementation)

Already implemented:
- HTTP GET requests with URLs and timing
- Cache hits (CACHE) vs downloads (GET → OK)
- Lock acquisition for package installation

```
Acquiring lock for Newtonsoft.Json 13.0.3
  GET https://api.nuget.org/v3/flatcontainer/newtonsoft.json/13.0.3/newtonsoft.json.13.0.3.nupkg
  OK 200 (156ms, 712 KB)
  Extracting to ~/.nuget/packages/newtonsoft.json/13.0.3

Serilog 3.1.1
  CACHE (already in ~/.nuget/packages/serilog/3.1.1)
```

**Status**: ✅ Already implemented in `restorer.go` lines 956-1161

### Tier 3: Project and Configuration Discovery

Show how gonuget locates and parses configuration:

```
Project analysis:
  File: /Users/brandon/src/gonuget/tests/test-scenarios/simple/test.csproj
  SDK: Microsoft.NET.Sdk
  Target frameworks: net8.0
  Package references:
    - Newtonsoft.Json: 13.0.3
    - Serilog: 3.1.1
  Project references: (none)

Configuration discovery:
  Searching for NuGet.config...
    ✗ /Users/brandon/src/gonuget/tests/test-scenarios/simple/NuGet.config (not found)
    ✗ /Users/brandon/src/gonuget/tests/test-scenarios/NuGet.config (not found)
    ✗ /Users/brandon/src/gonuget/NuGet.config (not found)
    ✓ ~/.nuget/NuGet/NuGet.config (found)

  Active package sources:
    1. https://api.nuget.org/v3/index.json (enabled)
```

**Implementation Point**: Hook into `project.LoadProject()` and `config.GetEnabledSourcesOrDefault()`.

### Tier 4: Performance Breakdown

Time every major operation:

```
Performance breakdown:
  Project parsing:                    2ms
  NuGet.config discovery:             1ms
  Dependency resolution:
    - Newtonsoft.Json 13.0.3:        15ms (cache hit)
    - Serilog 3.1.1:                 18ms (cache hit)
    - Serilog.Sinks.File 6.0.0:      12ms (cache hit)
    - Total resolution:              45ms
  Package downloads:
    - Newtonsoft.Json (cache):        0ms
    - Serilog (cache):                0ms
    - Serilog.Sinks.File (download): 156ms
    - Total downloads:               156ms
  Assets generation:
    - project.assets.json:            8ms
    - nuget.g.props:                  2ms
    - nuget.g.targets:                2ms
    - Total generation:              12ms
  Total restore time:                216ms

Parallelization:
  Worker pool: 4/8 utilized
  Concurrent operations: 3 package resolutions
  Sequential operations: 1 download (cache hit), 1 download (network)
```

**Implementation Point**: Add timing hooks throughout `Restore()` function.

### Tier 5: Assets File Generation

Show what output files are created:

```
Writing restore outputs:
  ✓ /path/to/obj/project.assets.json (1,234 lines, 45 KB)
    - 3 packages
    - 1 target framework (net8.0)
    - 12 libraries referenced
  ✓ /path/to/obj/project.nuget.cache (dgspec hash: a1b2c3d4)
  ✓ /path/to/obj/test.csproj.nuget.g.props (23 lines)
  ✓ /path/to/obj/test.csproj.nuget.g.targets (18 lines)
```

**Implementation Point**: Hook into lock file generation and cache file writing.

## Implementation Plan

### Phase 1: Dependency Resolution Trace (Highest Value)

**Goal**: Show version selection logic and dependency graph construction

**Files to Modify**:
1. `restore/restorer.go`:
   - Add `DiagnosticTracer` interface
   - Implement tracing hooks in `Restore()` method
   - Capture resolution decisions

2. `core/resolver/walker.go`:
   - Add callbacks for resolution events
   - Emit trace messages for version selection
   - Report constraint evaluation

3. `restore/diagnostic.go` (NEW):
   - `DiagnosticTracer` interface
   - `ResolutionTracer` implementation
   - Formatting functions

**Implementation Steps**:

1. **Create diagnostic tracer interface** (`restore/diagnostic.go`):

```go
package restore

import (
	"github.com/willibrandon/gonuget/core/resolver"
	"github.com/willibrandon/gonuget/version"
)

// DiagnosticTracer captures restore operations for diagnostic output
type DiagnosticTracer interface {
	// TracePackageResolution logs package version selection
	TracePackageResolution(packageID, constraint string, available []string, selected string, reason string)

	// TraceFrameworkCheck logs framework compatibility check
	TraceFrameworkCheck(packageID, packageVersion string, framework string, compatible bool)

	// TraceDependencyDiscovered logs when a dependency is found
	TraceDependencyDiscovered(parentID, dependencyID, constraint string, isDirect bool)

	// TraceDependencyGraph logs the final resolved graph
	TraceDependencyGraph(graph *resolver.GraphNode)
}

// ResolutionTracer implements DiagnosticTracer for resolution tracing
type ResolutionTracer struct {
	console Console
	enabled bool
}

// NewResolutionTracer creates a new resolution tracer
func NewResolutionTracer(console Console, verbosity string) *ResolutionTracer {
	return &ResolutionTracer{
		console: console,
		enabled: verbosity == "diagnostic",
	}
}
```

2. **Add tracing to restorer** (`restore/restorer.go`):

```go
type Restorer struct {
	opts    *Options
	console Console
	client  *core.Client
	tracer  DiagnosticTracer  // NEW
}

func NewRestorer(opts *Options, console Console) *Restorer {
	// ... existing code ...

	return &Restorer{
		opts:    opts,
		console: console,
		client:  client,
		tracer:  NewResolutionTracer(console, opts.Verbosity),  // NEW
	}
}
```

3. **Hook into dependency resolution** (`restore/restorer.go` around line 425):

```go
// Before walking dependencies
if r.opts.Verbosity == "diagnostic" {
	r.console.Printf("\nResolving dependencies for %s:\n", targetFrameworkStr)
}

for _, pkgRef := range packageRefs {
	versionRange := pkgRef.Version
	if versionRange == "" {
		versionRange = "0.0.0"
	}

	// NEW: Trace resolution start
	if r.opts.Verbosity == "diagnostic" {
		r.console.Printf("  %s %s (direct reference)\n", pkgRef.Include, versionRange)
		r.console.Printf("    Constraint: %s\n", versionRange)
	}

	// Check version availability (already have this)
	versionInfos, allVersions, allSourceNames, canSatisfy := r.checkVersionAvailability(ctx, pkgRef.Include, versionRange)

	// NEW: Trace available versions
	if r.opts.Verbosity == "diagnostic" && len(allVersions) > 0 {
		// Show up to 10 most recent versions
		displayVersions := allVersions
		if len(displayVersions) > 10 {
			displayVersions = displayVersions[len(displayVersions)-10:]
		}
		r.console.Printf("    Available versions: %s\n", strings.Join(displayVersions, ", "))
	}

	// Walk dependency graph
	graphNode, err := walker.Walk(ctx, pkgRef.Include, versionRange, targetFrameworkStr, true)

	// NEW: Trace selected version
	if r.opts.Verbosity == "diagnostic" && graphNode != nil && graphNode.Item != nil {
		r.console.Printf("    Selected: %s (highest matching)\n", graphNode.Item.Version)
		r.console.Printf("    Framework: compatible with %s\n", targetFrameworkStr)

		// Show dependencies
		deps := r.getDependenciesForFramework(graphNode.Item, targetFrameworkStr)
		if len(deps) > 0 {
			r.console.Printf("    Dependencies:\n")
			for _, dep := range deps {
				r.console.Printf("      → %s %s\n", dep.ID, dep.VersionRange)
			}
		}
		r.console.Printf("\n")
	}

	// ... rest of existing code ...
}

// NEW: Trace final graph summary
if r.opts.Verbosity == "diagnostic" {
	directCount := len(packageRefs)
	transitiveCount := len(allResolvedPackages) - directCount
	r.console.Printf("Dependency graph resolved:\n")
	r.console.Printf("  %d packages total (%d direct, %d transitive)\n\n",
		len(allResolvedPackages), directCount, transitiveCount)
}
```

4. **Add helper for framework-specific dependencies**:

```go
// getDependenciesForFramework extracts dependencies for a specific target framework
// This is needed for diagnostic output to show which dependencies are active
func (r *Restorer) getDependenciesForFramework(info *resolver.PackageDependencyInfo, framework string) []resolver.PackageDependency {
	if info == nil {
		return nil
	}

	// Find matching dependency group
	for _, group := range info.DependencyGroups {
		if group.TargetFramework == framework {
			return group.Dependencies
		}
	}

	// Fallback: return first group if no exact match
	if len(info.DependencyGroups) > 0 {
		return info.DependencyGroups[0].Dependencies
	}

	return nil
}
```

### Phase 2: Project and Configuration Discovery

**Goal**: Show how gonuget finds and parses project files and NuGet.config

**Files to Modify**:
1. `restore/restorer.go` - Add project parsing trace
2. `cmd/gonuget/commands/restore.go` - Add config discovery trace

**Implementation**: Add diagnostic output before project load and source setup

### Phase 3: Performance Breakdown

**Goal**: Time every major operation and show breakdown

**Files to Modify**:
1. `restore/restorer.go` - Add timing instrumentation

**Implementation**: Capture timing for each phase using `time.Now()` and `time.Since()`

### Phase 4: Assets Generation Trace

**Goal**: Show what files are written

**Files to Modify**:
1. `restore/restorer.go` - Add trace after lock file generation

**Implementation**: Log file paths and sizes after successful writes

## Technical Design

### Verbosity Level Behavior

| Level | Shows |
|-------|-------|
| `quiet` | Nothing on success, errors only |
| `minimal` | Summary only ("Restore complete", "Restore succeeded") |
| `normal` | Same as minimal for restore |
| `detailed` | Terminal Logger clean output (determining projects, per-project timing) |
| `diagnostic` | **Everything**: resolution trace, network ops, timing, caching, graph |

### Output Formatting Standards

1. **Indentation**: Use 2 spaces for nested information
2. **Bullets**: Use `→` for dependencies, `✓` for success, `✗` for not found
3. **Timing**: Show milliseconds for < 1000ms, seconds for >= 1000ms
4. **Colors**: None in diagnostic messages (just data)
5. **Alignment**: Left-align all text, right-align timing

### Performance Considerations

**Zero Cost When Disabled**:
```go
if r.opts.Verbosity == "diagnostic" {
	// Expensive formatting only when enabled
	r.console.Printf("Available versions: %s\n", strings.Join(allVersions, ", "))
}
```

**Lazy Evaluation**:
```go
// DON'T do this (always allocates):
message := fmt.Sprintf("Available: %v", allVersions)
if diagnostic { print(message) }

// DO this (allocates only when needed):
if diagnostic {
	r.console.Printf("Available: %v\n", allVersions)
}
```

## Examples

### Example 1: Simple Restore (2 Packages)

**Command**: `gonuget restore tests/test-scenarios/simple/test.csproj -v:diagnostic`

**Output**:
```
Project analysis:
  File: tests/test-scenarios/simple/test.csproj
  SDK: Microsoft.NET.Sdk
  Target frameworks: net8.0

Resolving dependencies for net8.0:
  Newtonsoft.Json 13.0.3 (direct reference)
    Constraint: >= 13.0.3
    Available versions: 13.0.1, 13.0.2, 13.0.3, 13.0.4
    Selected: 13.0.3 (exact match)
    Framework: compatible with net8.0

  Serilog 3.1.1 (direct reference)
    Constraint: >= 3.1.1
    Available versions: 3.0.0, 3.1.0, 3.1.1
    Selected: 3.1.1 (exact match)
    Framework: compatible with net8.0

Dependency graph resolved:
  2 packages total (2 direct, 0 transitive)

         Acquiring lock for the installation of Newtonsoft.Json 13.0.3
         Acquired lock for the installation of Newtonsoft.Json 13.0.3
           GET https://api.nuget.org/v3/flatcontainer/newtonsoft.json/13.0.3/newtonsoft.json.13.0.3.nupkg
           OK https://api.nuget.org/v3/flatcontainer/newtonsoft.json/13.0.3/newtonsoft.json.13.0.3.nupkg 120ms
           CACHE https://api.nuget.org/v3/vulnerabilities/index.json

Writing restore outputs:
  ✓ tests/test-scenarios/simple/obj/project.assets.json
  ✓ tests/test-scenarios/simple/obj/project.nuget.cache
  ✓ tests/test-scenarios/simple/obj/test.csproj.nuget.g.props
  ✓ tests/test-scenarios/simple/obj/test.csproj.nuget.g.targets

Restore complete (0.3s)
    Determining projects to restore...
    Restored tests/test-scenarios/simple/test.csproj (in 260 ms).

Restore succeeded in 0.3s
```

### Example 2: Complex Restore with Transitive Dependencies

**Command**: `gonuget restore tests/test-scenarios/complex/test.csproj -v:diagnostic`

**Output**:
```
Project analysis:
  File: tests/test-scenarios/complex/test.csproj
  SDK: Microsoft.NET.Sdk
  Target frameworks: net8.0

Resolving dependencies for net8.0:
  Newtonsoft.Json 13.0.3 (direct reference)
    Constraint: >= 13.0.3
    Selected: 13.0.3 (exact match)
    Framework: compatible with net8.0

  Serilog 3.1.1 (direct reference)
    Constraint: >= 3.1.1
    Selected: 3.1.1 (exact match)
    Framework: compatible with net8.0
    Dependencies:
      → Serilog.Sinks.File >= 5.0.0

  Serilog.Sinks.File 6.0.0 (transitive via Serilog)
    Constraint: >= 5.0.0
    Available versions: 5.0.0, 5.0.1, 6.0.0
    Selected: 6.0.0 (highest matching)
    Framework: compatible with net8.0
    Parent chain: Serilog → Serilog.Sinks.File

  AutoMapper 12.0.1 (direct reference)
    Constraint: >= 12.0.1
    Selected: 12.0.1 (exact match)
    Framework: compatible with net8.0
    Dependencies:
      → Microsoft.CSharp >= 4.7.0

  Microsoft.CSharp 4.7.0 (transitive via AutoMapper)
    Constraint: >= 4.7.0
    Available versions: 4.0.1, 4.4.0, 4.5.0, 4.7.0
    Selected: 4.7.0 (highest matching)
    Framework: compatible with net8.0
    Parent chain: AutoMapper → Microsoft.CSharp

Dependency graph resolved:
  7 packages total (3 direct, 4 transitive)
  Max depth: 1

Performance breakdown:
  Dependency resolution: 45ms
  Package downloads: 156ms
  Assets generation: 12ms
  Total: 213ms

Restore complete (0.2s)
    Determining projects to restore...
    Restored tests/test-scenarios/complex/test.csproj (in 213 ms).

Restore succeeded in 0.2s
```

### Example 3: Error Case (Version Not Found)

**Command**: `gonuget restore tests/test-scenarios/nu1102/test.csproj -v:diagnostic`

**Output**:
```
Project analysis:
  File: tests/test-scenarios/nu1102/test.csproj
  SDK: Microsoft.NET.Sdk
  Target frameworks: net8.0

Resolving dependencies for net8.0:
  Newtonsoft.Json 99.99.99 (direct reference)
    Constraint: >= 99.99.99
    Available versions: 13.0.1, 13.0.2, 13.0.3, 13.0.4
    ERROR: No version satisfies constraint >= 99.99.99
    Nearest version: 13.0.4

tests/test-scenarios/nu1102/test.csproj : error NU1102: Unable to find package Newtonsoft.Json with version (>= 99.99.99)
tests/test-scenarios/nu1102/test.csproj : error NU1102:   - Found 326 version(s) in nuget.org [ Nearest version: 13.0.3 ]

Restore failed with 1 error(s) in 0.5s
```

## Testing Strategy

### Unit Tests

Test diagnostic output generation:

```go
// restore/diagnostic_test.go
func TestResolutionTracer_TracePackageResolution(t *testing.T) {
	var buf bytes.Buffer
	console := &testConsole{writer: &buf}
	tracer := NewResolutionTracer(console, "diagnostic")

	tracer.TracePackageResolution(
		"Newtonsoft.Json",
		">= 13.0.3",
		[]string{"13.0.1", "13.0.2", "13.0.3", "13.0.4"},
		"13.0.3",
		"exact match",
	)

	output := buf.String()
	assert.Contains(t, output, "Newtonsoft.Json")
	assert.Contains(t, output, "13.0.3")
	assert.Contains(t, output, "Available versions")
}
```

### Integration Tests

Compare diagnostic output format against dotnet:

```bash
# Test diagnostic output matches expected format
./gonuget restore tests/test-scenarios/simple/test.csproj -v:diagnostic > gonuget.log
# Verify format
grep "Resolving dependencies" gonuget.log
grep "Selected:" gonuget.log
grep "Dependency graph resolved" gonuget.log
```

### Regression Tests

Ensure diagnostic output doesn't break:
- Normal/detailed/quiet modes
- Error scenarios
- Performance (diagnostic shouldn't slow down by >10%)

## Success Criteria

1. ✅ Diagnostic mode shows dependency resolution decisions
2. ✅ Users can trace why specific versions were selected
3. ✅ Output is readable and actionable
4. ✅ Performance impact < 5% for diagnostic mode
5. ✅ No performance impact for other verbosity levels
6. ✅ All existing tests still pass

## Future Enhancements

1. **Conflict Resolution Trace**: Show version conflicts and how they're resolved
2. **Graph Visualization**: ASCII art dependency tree
3. **JSON Output Mode**: Machine-readable diagnostic data for tools
4. **Flame Graphs**: Performance profiling visualization
5. **Detailed HTTP Trace**: Show request/response headers, retry attempts
6. **Cache Statistics**: Hit rate, size, eviction info

## References

- **NuGet.Client**: RestoreCommand diagnostic logging patterns
- **MSBuild**: Terminal Logger message filtering approach
- **NPM**: Dependency resolution trace format inspiration
- **Cargo**: Verbose output best practices
