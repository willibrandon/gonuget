# Transitive Dependency Resolution Implementation Guide

**Status**: Implementation Required
**Target**: 100% parity with NuGet.Client RestoreCommand
**Test Coverage**: 90% minimum, library interop tests required
**Reference**: `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/`

---

## Executive Summary

gonuget's restore currently downloads ONLY direct dependencies (packages listed in .csproj). NuGet.Client downloads ALL transitive dependencies (packages required by direct dependencies). This guide implements full transitive resolution matching RestoreCommand.cs line-by-line behavior.

**Current State**: `restore/restorer.go:130` - "direct dependencies only (Chunk 5 - simplified)"

**Required State**: Full transitive resolution with DependencyGraphResolver + RemoteDependencyWalker

---

## Architecture Overview

### NuGet.Client RestoreCommand Flow

```
RestoreRunner.ExecuteAsync()
  └─> RestoreCommand.ExecuteAsync()
        ├─> GenerateRestoreGraphsAsync()
        │     └─> ExecuteRestoreAsync()  // NEW PATH (.NET 10+)
        │           └─> DependencyGraphResolver.ResolveAsync()
        │                 ├─> Creates queue of DependencyGraphItem
        │                 ├─> Processes dependencies breadth-first
        │                 ├─> Detects cycles and conflicts
        │                 └─> Returns List<RestoreTargetGraph>
        │     └─> ExecuteLegacyRestoreAsync()  // LEGACY PATH
        │           └─> RemoteDependencyWalker.WalkAsync()
        │                 ├─> Creates GraphNode<RemoteResolveResult>
        │                 ├─> Recursively walks dependencies
        │                 └─> Returns dependency graph
        ├─> ProjectRestoreCommand.InstallPackagesAsync()
        │     └─> Downloads all resolved packages to global cache
        └─> BuildAssetsFile()
              └─> Generates project.assets.json with full Libraries map
```

**Key Facts**:
1. RestoreCommand line 583: Uses `DependencyGraphResolver` (new path) or `RemoteDependencyWalker` (legacy)
2. Both produce `RestoreTargetGraph` with transitive dependencies
3. `ProjectRestoreCommand.InstallPackagesAsync()` downloads ALL packages from graph
4. `BuildAssetsFile()` populates Libraries map with ALL packages (direct + transitive)

---

## Current gonuget Implementation

### File: `restore/restorer.go`

**Line 130-186**: Restore() method
```go
// Restore executes the restore operation for direct dependencies only (Chunk 5 - simplified).
func (r *Restorer) Restore(ctx context.Context, proj *project.Project, packageRefs []project.PackageReference) (*Result, error) {
	result := &Result{
		Packages: make([]PackageInfo, 0, len(packageRefs)),
	}

	// PROBLEM: Only iterates packageRefs (direct dependencies)
	for _, pkgRef := range packageRefs {
		r.console.Printf("  Restoring %s %s...\n", pkgRef.Include, pkgRef.Version)

		// Downloads only this package, not its dependencies
		if err := r.downloadPackage(ctx, normalizedPackageID, pkgRef.Version, packagePath); err != nil {
			return nil, fmt.Errorf("failed to download package %s %s: %w", pkgRef.Include, pkgRef.Version, err)
		}

		result.Packages = append(result.Packages, PackageInfo{
			ID:      pkgRef.Include,
			Version: pkgRef.Version,
			Path:    packagePath,
		})
	}

	return result, nil
}
```

**Result**: Only direct dependencies in `result.Packages`

---

## Required Implementation

### Phase 1: Integrate core/resolver into restore

gonuget already has `core/resolver` package with `DependencyWalker` (lines 11-356 in walker.go). This matches RemoteDependencyWalker functionality. Integration required:

#### Step 1.1: Modify Result Structure

**File**: `restore/restorer.go`

**Current** (line 118-128):
```go
// Result holds restore results.
type Result struct {
	Packages []PackageInfo
}

// PackageInfo holds package information.
type PackageInfo struct {
	ID      string
	Version string
	Path    string
}
```

**Required**:
```go
// Result holds restore results.
type Result struct {
	// DirectPackages contains packages explicitly listed in project file
	DirectPackages []PackageInfo

	// TransitivePackages contains packages pulled in as dependencies
	TransitivePackages []PackageInfo

	// Graph contains full dependency graph (optional, for debugging)
	Graph *resolver.GraphNode
}

// PackageInfo holds package information.
type PackageInfo struct {
	ID      string
	Version string
	Path    string

	// IsDirect indicates if this is a direct dependency
	IsDirect bool

	// Parents lists packages that depend on this (for transitive deps)
	Parents []string
}

// AllPackages returns all packages (direct + transitive)
func (r *Result) AllPackages() []PackageInfo {
	all := make([]PackageInfo, 0, len(r.DirectPackages)+len(r.TransitivePackages))
	all = append(all, r.DirectPackages...)
	all = append(all, r.TransitivePackages...)
	return all
}
```

**Rationale**: NuGet.Client's RestoreTargetGraph contains ALL packages. project.assets.json Libraries map includes both direct and transitive.

#### Step 1.2: Create Dependency Walker Adapter

**File**: `restore/dependency_walker_adapter.go` (NEW FILE)

```go
package restore

import (
	"context"
	"fmt"

	"github.com/willibrandon/gonuget/core"
	"github.com/willibrandon/gonuget/core/resolver"
	"github.com/willibrandon/gonuget/packaging"
	"github.com/willibrandon/gonuget/version"
)

// DependencyWalkerAdapter adapts core.Client to resolver.PackageMetadataClient
type DependencyWalkerAdapter struct {
	client *core.Client
}

// NewDependencyWalkerAdapter creates adapter
func NewDependencyWalkerAdapter(client *core.Client) *DependencyWalkerAdapter {
	return &DependencyWalkerAdapter{client: client}
}

// GetPackageMetadata fetches package metadata from sources
func (a *DependencyWalkerAdapter) GetPackageMetadata(
	ctx context.Context,
	source string,
	packageID string,
) ([]*resolver.PackageDependencyInfo, error) {
	// Get all versions of package
	versions, err := a.client.GetPackageVersions(ctx, packageID)
	if err != nil {
		return nil, fmt.Errorf("get package versions: %w", err)
	}

	// Fetch metadata for each version
	infos := make([]*resolver.PackageDependencyInfo, 0, len(versions))
	for _, ver := range versions {
		metadata, err := a.client.GetPackageMetadata(ctx, packageID, ver)
		if err != nil {
			continue // Skip unavailable versions
		}

		// Parse dependencies from nuspec
		deps := make([]resolver.PackageDependency, 0)
		for _, depGroup := range metadata.DependencyGroups {
			for _, dep := range depGroup.Dependencies {
				deps = append(deps, resolver.PackageDependency{
					ID:           dep.ID,
					VersionRange: dep.VersionRange,
				})
			}
		}

		info := &resolver.PackageDependencyInfo{
			ID:               packageID,
			Version:          ver,
			Dependencies:     deps,
			DependencyGroups: metadata.DependencyGroups,
		}
		infos = append(infos, info)
	}

	return infos, nil
}
```

**Rationale**: NuGet.Client's RemoteWalkContext provides package metadata to RemoteDependencyWalker. This adapter bridges gonuget's core.Client to resolver.PackageMetadataClient interface.

#### Step 1.3: Rewrite Restore() Method

**File**: `restore/restorer.go`

**Replace** lines 130-186 with:

```go
// Restore executes the restore operation with full transitive dependency resolution.
// Matches NuGet.Client RestoreCommand behavior (line 572-616 GenerateRestoreGraphsAsync).
func (r *Restorer) Restore(
	ctx context.Context,
	proj *project.Project,
	packageRefs []project.PackageReference,
) (*Result, error) {
	result := &Result{
		DirectPackages:     make([]PackageInfo, 0, len(packageRefs)),
		TransitivePackages: make([]PackageInfo, 0),
	}

	// Get global packages folder
	packagesFolder := r.opts.PackagesFolder
	if packagesFolder == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		packagesFolder = filepath.Join(home, ".nuget", "packages")
	}

	// Ensure packages folder exists
	if err := os.MkdirAll(packagesFolder, 0755); err != nil {
		return nil, fmt.Errorf("failed to create packages folder: %w", err)
	}

	// Get target framework
	targetFrameworks := proj.GetTargetFrameworks()
	if len(targetFrameworks) == 0 {
		return nil, fmt.Errorf("project has no target frameworks")
	}
	targetFramework := targetFrameworks[0] // Use first TFM

	// Track all resolved packages (direct + transitive)
	allResolvedPackages := make(map[string]*resolver.PackageDependencyInfo)

	// Phase 1: Walk dependency graph for each direct dependency
	adapter := NewDependencyWalkerAdapter(r.client)
	walker := resolver.NewDependencyWalker(adapter, r.sources, targetFramework)

	for _, pkgRef := range packageRefs {
		r.console.Printf("  Resolving %s %s...\n", pkgRef.Include, pkgRef.Version)

		// Walk dependency graph (matches RemoteDependencyWalker.WalkAsync line 28)
		graphNode, err := walker.Walk(
			ctx,
			pkgRef.Include,
			pkgRef.Version,
			targetFramework,
			true, // recursive=true for transitive resolution
		)
		if err != nil {
			return nil, fmt.Errorf("failed to walk dependencies for %s: %w", pkgRef.Include, err)
		}

		// Collect all packages from graph (breadth-first)
		if err := r.collectPackagesFromGraph(graphNode, allResolvedPackages); err != nil {
			return nil, err
		}
	}

	// Phase 2: Download all resolved packages (direct + transitive)
	// Matches ProjectRestoreCommand.InstallPackagesAsync behavior
	for packageKey, pkgInfo := range allResolvedPackages {
		normalizedID := strings.ToLower(pkgInfo.ID)
		packagePath := filepath.Join(packagesFolder, normalizedID, pkgInfo.Version)

		// Check if package already exists in cache
		if !r.opts.Force {
			if _, err := os.Stat(packagePath); err == nil {
				r.console.Printf("    Package %s %s already cached\n", pkgInfo.ID, pkgInfo.Version)
				continue
			}
		}

		r.console.Printf("  Downloading %s %s...\n", pkgInfo.ID, pkgInfo.Version)

		// Download package
		if err := r.downloadPackage(ctx, normalizedID, pkgInfo.Version, packagePath); err != nil {
			return nil, fmt.Errorf("failed to download package %s %s: %w", pkgInfo.ID, pkgInfo.Version, err)
		}
	}

	// Phase 3: Categorize packages as direct vs transitive
	directPackageIDs := make(map[string]bool)
	for _, pkgRef := range packageRefs {
		directPackageIDs[strings.ToLower(pkgRef.Include)] = true
	}

	for packageKey, pkgInfo := range allResolvedPackages {
		normalizedID := strings.ToLower(pkgInfo.ID)
		packagePath := filepath.Join(packagesFolder, normalizedID, pkgInfo.Version)

		info := PackageInfo{
			ID:       pkgInfo.ID,
			Version:  pkgInfo.Version,
			Path:     packagePath,
			IsDirect: directPackageIDs[normalizedID],
			Parents:  []string{}, // TODO: Collect from graph
		}

		if info.IsDirect {
			result.DirectPackages = append(result.DirectPackages, info)
		} else {
			result.TransitivePackages = append(result.TransitivePackages, info)
		}
	}

	return result, nil
}

// collectPackagesFromGraph traverses graph and collects all packages.
// Matches NuGet.Client's graph flattening in BuildAssetsFile.
func (r *Restorer) collectPackagesFromGraph(
	node *resolver.GraphNode,
	collected map[string]*resolver.PackageDependencyInfo,
) error {
	if node == nil || node.Item == nil {
		return nil
	}

	// Add this node's package
	key := node.Key
	if _, exists := collected[key]; !exists {
		collected[key] = node.Item
	}

	// Recursively collect children (depth-first)
	for _, child := range node.InnerNodes {
		if err := r.collectPackagesFromGraph(child, collected); err != nil {
			return err
		}
	}

	return nil
}
```

**Rationale**:
- Line 28-64 in RemoteDependencyWalker.WalkAsync: Creates root node and walks recursively
- RestoreCommand line 585: Calls ExecuteRestoreAsync which resolves full graph
- Line 165 in RestoreCommand: InstallPackagesAsync downloads all packages from graphs
- Line 265 in RestoreCommand: BuildAssetsFile populates Libraries with all packages

#### Step 1.4: Update Lock File Generation

**File**: `restore/lock_file_format.go`

**Current**: Libraries map only has direct dependencies

**Required**: Populate Libraries with ALL packages (direct + transitive)

**Modify** `LockFileBuilder.Build()` (line ~100):

```go
// Build creates lock file from restore result
func (b *LockFileBuilder) Build(proj *project.Project, result *Result) *LockFile {
	lockFile := &LockFile{
		Version:                     3,
		Targets:                     make(map[string]Target),
		Libraries:                   make(map[string]Library),
		ProjectFileDependencyGroups: make(map[string][]string),
		PackageFolders:              make(map[string]PackageFolder),
		Project:                     b.buildProjectInfo(proj),
	}

	// Add ALL packages to Libraries (direct + transitive)
	// This matches NuGet.Client BuildAssetsFile line 265
	allPackages := result.AllPackages()
	for _, pkg := range allPackages {
		key := fmt.Sprintf("%s/%s", pkg.ID, pkg.Version)

		lockFile.Libraries[key] = Library{
			Type:  "package",
			Path:  pkg.Path,
			Files: []string{}, // TODO: Read from .nupkg
		}
	}

	// ProjectFileDependencyGroups contains ONLY direct dependencies
	// This matches NuGet.Client behavior
	targetFrameworks := proj.GetTargetFrameworks()
	for _, tfm := range targetFrameworks {
		deps := make([]string, 0, len(result.DirectPackages))
		for _, pkg := range result.DirectPackages {
			deps = append(deps, fmt.Sprintf("%s >= %s", pkg.ID, pkg.Version))
		}
		lockFile.ProjectFileDependencyGroups[tfm] = deps
	}

	// Add global packages folder
	home, _ := os.UserHomeDir()
	packagesFolder := filepath.Join(home, ".nuget", "packages")
	lockFile.PackageFolders[packagesFolder] = PackageFolder{}

	return lockFile
}
```

**Key Changes**:
1. Libraries map includes ALL packages (direct + transitive)
2. ProjectFileDependencyGroups includes ONLY direct dependencies
3. Matches NuGet.Client's LockFileBuilder behavior exactly

---

## Phase 2: Add core.Client Methods

gonuget's `core/client.go` is missing methods required by adapter.

**File**: `core/client.go`

**Add**:

```go
// GetPackageVersions returns all available versions for a package
func (c *Client) GetPackageVersions(ctx context.Context, packageID string) ([]string, error) {
	repos := c.repoManager.ListRepositories()
	if len(repos) == 0 {
		return nil, fmt.Errorf("no package sources configured")
	}

	repo := repos[0]
	provider, err := repo.GetProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("get provider: %w", err)
	}

	return provider.ListVersions(ctx, packageID)
}

// GetPackageMetadata returns metadata for specific package version
func (c *Client) GetPackageMetadata(ctx context.Context, packageID, version string) (*Metadata, error) {
	repos := c.repoManager.ListRepositories()
	if len(repos) == 0 {
		return nil, fmt.Errorf("no package sources configured")
	}

	repo := repos[0]
	provider, err := repo.GetProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("get provider: %w", err)
	}

	return provider.GetMetadata(ctx, packageID, version)
}

// Metadata represents package metadata
type Metadata struct {
	ID               string
	Version          string
	Description      string
	Authors          []string
	DependencyGroups []DependencyGroup
}

// DependencyGroup represents framework-specific dependencies
type DependencyGroup struct {
	TargetFramework string
	Dependencies    []Dependency
}

// Dependency represents a package dependency
type Dependency struct {
	ID           string
	VersionRange string
}
```

**Rationale**: RemoteWalkContext in NuGet.Client provides these methods. Current core.Client only has Search and DownloadPackage.

---

## Phase 3: Update CLI Integration

**File**: `cmd/gonuget/commands/add_package.go`

**Modify** lines 147-184 (restore section):

```go
// Perform restore if needed
if !opts.NoRestore {
	restoreOpts := &restore.Options{
		PackagesFolder: opts.PackageDirectory,
		Sources:        []string{},
	}

	if opts.Source != "" {
		restoreOpts.Sources = []string{opts.Source}
	} else {
		projectDir := filepath.Dir(projectPath)
		sources := config.GetEnabledSourcesOrDefault(projectDir)
		for _, source := range sources {
			restoreOpts.Sources = append(restoreOpts.Sources, source.Value)
		}
	}

	console := &cliConsole{}
	restorer := restore.NewRestorer(restoreOpts, console)

	packageRefs := proj.GetPackageReferences()
	result, err := restorer.Restore(ctx, proj, packageRefs)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	// Generate project.assets.json (matches dotnet add package behavior)
	lockFile := restore.NewLockFileBuilder().Build(proj, result)
	objDir := filepath.Join(filepath.Dir(projectPath), "obj")
	assetsPath := filepath.Join(objDir, "project.assets.json")
	if err := lockFile.Save(assetsPath); err != nil {
		return fmt.Errorf("failed to save project.assets.json: %w", err)
	}

	// Show summary of restored packages
	fmt.Printf("  Restored %d package(s) to %s\n",
		len(result.DirectPackages)+len(result.TransitivePackages),
		projectPath)

	if len(result.DirectPackages) > 0 {
		fmt.Printf("\nDirect packages:\n")
		for _, pkg := range result.DirectPackages {
			fmt.Printf("  - %s %s\n", pkg.ID, pkg.Version)
		}
	}

	if len(result.TransitivePackages) > 0 {
		fmt.Printf("\nTransitive packages:\n")
		for _, pkg := range result.TransitivePackages {
			fmt.Printf("  - %s %s\n", pkg.ID, pkg.Version)
		}
	}
}
```

**Rationale**: Matches `dotnet add package` output which shows both direct and transitive dependencies.

---

## Testing Requirements

### Unit Tests (90% coverage target)

**File**: `restore/restorer_test.go`

**Required Tests**:

1. **TestRestore_DirectDependenciesOnly**
   - Package with NO transitive dependencies
   - Verify only direct package in DirectPackages
   - Verify TransitivePackages is empty

2. **TestRestore_SingleTransitiveDependency**
   - Package A depends on B
   - Verify A in DirectPackages
   - Verify B in TransitivePackages

3. **TestRestore_DeepTransitiveChain**
   - A → B → C → D (4-level chain)
   - Verify A in DirectPackages
   - Verify B, C, D in TransitivePackages

4. **TestRestore_DiamondDependency**
   - A → B, A → C, B → D, C → D
   - Verify D appears once in TransitivePackages
   - Verify no duplicates

5. **TestRestore_MultipleDirectDependencies**
   - Project depends on A and B
   - A → C, B → D
   - Verify A, B in DirectPackages
   - Verify C, D in TransitivePackages

6. **TestRestore_CyclicDependency**
   - A → B → C → A
   - Verify cycle detection
   - Verify no infinite loop

7. **TestRestore_VersionConflict**
   - A → C v1.0, B → C v2.0
   - Verify "nearest wins" resolution
   - Verify only one version of C in result

8. **TestRestore_FrameworkSpecificDependencies**
   - Package has different deps for net6.0 vs net8.0
   - Verify correct deps selected for target framework

### Integration Tests (with nuget.org)

**File**: `restore/restorer_integration_test.go`

**Required Tests** (skip with `-short`):

1. **TestRestore_NewtonsoftJson**
   - Newtonsoft.Json 13.0.3 has zero dependencies
   - Verify DirectPackages contains only Newtonsoft.Json

2. **TestRestore_SerilogSinksFile**
   - Serilog.Sinks.File 5.0.0 depends on Serilog
   - Verify DirectPackages contains Serilog.Sinks.File
   - Verify TransitivePackages contains Serilog

3. **TestRestore_MicrosoftExtensionsLogging**
   - Microsoft.Extensions.Logging has multiple transitive deps
   - Verify all transitive dependencies resolved

### Library Interop Tests

**File**: `tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs` (NEW)

**Required Tests**:

```csharp
[Fact]
public async Task RestoreWithTransitiveDependencies_MatchesNuGetClient()
{
    // Setup: Create test project with Serilog.Sinks.File reference
    var projectContent = @"
<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Serilog.Sinks.File"" Version=""5.0.0"" />
  </ItemGroup>
</Project>";

    var projectPath = Path.Combine(TestOutputPath, "test.csproj");
    File.WriteAllText(projectPath, projectContent);

    // Execute: Run NuGet.Client restore
    var nugetResult = await RestoreWithNuGetClient(projectPath);

    // Execute: Run gonuget restore
    var gonugetResult = await RestoreWithGonuget(projectPath);

    // Verify: Library count matches
    Assert.Equal(nugetResult.LibraryCount, gonugetResult.LibraryCount);

    // Verify: All libraries match
    foreach (var lib in nugetResult.Libraries)
    {
        Assert.Contains(lib, gonugetResult.Libraries);
    }

    // Verify: ProjectFileDependencyGroups matches (only direct deps)
    Assert.Equal(
        nugetResult.ProjectFileDependencyGroups["net8.0"],
        gonugetResult.ProjectFileDependencyGroups["net8.0"]
    );
}

[Fact]
public async Task RestoreWithDiamondDependency_MatchesNuGetClient()
{
    // Create project with diamond dependency pattern
    // Verify gonuget resolves identically to NuGet.Client
}

[Fact]
public async Task RestoreWithVersionConflict_MatchesNuGetClient()
{
    // Create project with version conflict
    // Verify gonuget uses "nearest wins" like NuGet.Client
}
```

**Target**: 491 existing interop tests MUST continue passing + 10+ new tests for transitive resolution

---

## Performance Requirements

**Benchmark Target**: Match or exceed NuGet.Client performance

**File**: `restore/restorer_benchmark_test.go`

```go
func BenchmarkRestore_10Packages(b *testing.B) {
    // Project with 10 direct dependencies, ~50 transitive
    // Target: <500ms per restore
}

func BenchmarkRestore_100Packages(b *testing.B) {
    // Project with 100 direct dependencies, ~500 transitive
    // Target: <5s per restore
}
```

**Parallel Resolution**: Use `DependencyWalker` goroutine-based parallelism (walker.go line 178-181)

---

## Verification Against dotnet

**Manual Verification Steps**:

1. Create test project:
```bash
dotnet new console -n TestTransitive
cd TestTransitive
dotnet add package Serilog.Sinks.File --version 5.0.0
```

2. Run dotnet restore:
```bash
dotnet restore -v detailed > dotnet-restore.log
```

3. Run gonuget restore:
```bash
gonuget restore > gonuget-restore.log
```

4. Compare project.assets.json:
```bash
jq '.libraries | keys' obj/project.assets.json > dotnet-libs.txt
# Run gonuget
jq '.libraries | keys' obj/project.assets.json > gonuget-libs.txt
diff dotnet-libs.txt gonuget-libs.txt
```

5. Verify ZERO differences in Libraries map

---

## Implementation Checklist

### Core Implementation
- [ ] Add Result.DirectPackages and Result.TransitivePackages fields
- [ ] Create DependencyWalkerAdapter in restore/dependency_walker_adapter.go
- [ ] Rewrite Restore() method to use DependencyWalker
- [ ] Add collectPackagesFromGraph() helper
- [ ] Add core.Client.GetPackageVersions() method
- [ ] Add core.Client.GetPackageMetadata() method
- [ ] Update LockFileBuilder.Build() to include transitive packages

### CLI Integration
- [ ] Update add_package.go to show direct vs transitive packages
- [ ] Update restore output formatting

### Testing
- [ ] Write 8 unit tests in restorer_test.go
- [ ] Write 3 integration tests in restorer_integration_test.go
- [ ] Write 10+ interop tests in RestoreTransitiveTests.cs
- [ ] Verify 491 existing interop tests still pass
- [ ] Add benchmark tests
- [ ] Achieve 90% test coverage

### Verification
- [ ] Manual comparison with dotnet restore output
- [ ] project.assets.json Libraries match 100%
- [ ] ProjectFileDependencyGroups contains only direct deps
- [ ] Performance benchmarks meet targets

---

## Expected Behavior Examples

### Example 1: Serilog.Sinks.File

**Input**: `dotnet add package Serilog.Sinks.File --version 5.0.0`

**Expected project.assets.json Libraries**:
```json
{
  "libraries": {
    "Serilog/2.12.0": {
      "type": "package",
      "path": "serilog/2.12.0"
    },
    "Serilog.Sinks.File/5.0.0": {
      "type": "package",
      "path": "serilog.sinks.file/5.0.0"
    }
  },
  "projectFileDependencyGroups": {
    "net8.0": [
      "Serilog.Sinks.File >= 5.0.0"
    ]
  }
}
```

**Expected CLI Output**:
```
info : Added package 'Serilog.Sinks.File' version '5.0.0' to project
  Resolving Serilog.Sinks.File 5.0.0...
  Downloading Serilog.Sinks.File 5.0.0...
  Downloading Serilog 2.12.0...
  Restored 2 package(s)

Direct packages:
  - Serilog.Sinks.File 5.0.0

Transitive packages:
  - Serilog 2.12.0
```

### Example 2: Package with No Dependencies

**Input**: `dotnet add package Newtonsoft.Json --version 13.0.3`

**Expected project.assets.json Libraries**:
```json
{
  "libraries": {
    "Newtonsoft.Json/13.0.3": {
      "type": "package",
      "path": "newtonsoft.json/13.0.3"
    }
  },
  "projectFileDependencyGroups": {
    "net8.0": [
      "Newtonsoft.Json >= 13.0.3"
    ]
  }
}
```

**Expected CLI Output**:
```
info : Added package 'Newtonsoft.Json' version '13.0.3' to project
  Resolving Newtonsoft.Json 13.0.3...
  Downloading Newtonsoft.Json 13.0.3...
  Restored 1 package(s)

Direct packages:
  - Newtonsoft.Json 13.0.3
```

---

## Reference Files

**NuGet.Client Source**:
- `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/RestoreCommand.cs`
- `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/RestoreRunner.cs`
- `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/DependencyGraphResolver.cs`
- `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.DependencyResolver.Core/Remote/RemoteDependencyWalker.cs`
- `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/RestoreTargetGraph.cs`

**gonuget Source**:
- `/Users/brandon/src/gonuget/restore/restorer.go` (line 130-186)
- `/Users/brandon/src/gonuget/core/resolver/walker.go` (line 11-356)
- `/Users/brandon/src/gonuget/restore/lock_file_format.go`
- `/Users/brandon/src/gonuget/cmd/gonuget/commands/add_package.go` (line 147-184)

---

## Completion Criteria

**Implementation is complete when**:
1. All 8 unit tests pass
2. All 3 integration tests pass
3. All 10+ interop tests pass
4. 491 existing interop tests still pass
5. Test coverage ≥90%
6. Manual verification: `project.assets.json` Libraries matches dotnet 100%
7. Benchmarks meet performance targets
8. `gonuget add package Serilog.Sinks.File` shows transitive dependency Serilog in output

---

**END OF IMPLEMENTATION GUIDE**
