# CLI M2: Restore Command

**Status:** Phase 2 - Package Management
**Command:** `gonuget restore`
**Parity Target:** `dotnet restore`

## Overview

Implements the `restore` command that downloads and installs packages based on PackageReference elements in the project file. This command reads the `.csproj` file, resolves dependencies, downloads packages to the global package cache, and generates the `project.assets.json` file.

**Critical:** This follows the MSBuild restore model, NOT the legacy packages.config restore.

## Command Specification

### Syntax

```bash
gonuget restore [<PROJECT|SOLUTION>] [options]
```

### Arguments

- `<PROJECT|SOLUTION>` - Optional path to project file (.csproj) or solution file (.sln). If not specified, searches current directory.

### Options

- `--source <SOURCE>` - Package source(s) to use
- `--packages <PATH>` - Custom global packages folder (default: ~/.nuget/packages)
- `--configfile <FILE>` - NuGet configuration file to use
- `--force` - Force re-download even if packages exist
- `--no-cache` - Don't use HTTP cache
- `--no-dependencies` - Only restore direct references, skip transitive
- `--verbosity <LEVEL>` - Set output verbosity (quiet, minimal, normal, detailed, diagnostic)
- `-h|--help` - Show help

### Examples

```bash
# Restore current project
gonuget restore

# Restore specific project
gonuget restore MyApp.csproj

# Restore with custom packages folder
gonuget restore --packages /custom/path/packages

# Restore from specific source
gonuget restore --source https://api.nuget.org/v3/index.json

# Force re-download
gonuget restore --force

# Minimal output
gonuget restore --verbosity quiet
```

## Implementation Phases

### Phase 1: Basic Restore (M2.1)

**Scope:**
- Read PackageReference elements from .csproj
- Download packages to global package cache
- Generate basic project.assets.json
- Single project only (no solution support)
- Direct dependencies only (no transitive)
- Single target framework

**Chunks:**
1. Project file reading and PackageReference extraction
2. Package download to global cache
3. project.assets.json generation

### Phase 2: Advanced Restore (M2.2)

**Scope:**
- Transitive dependency resolution
- Multi-TFM support
- Solution file support
- Lock file support (packages.lock.json)

**Deferred to Future:**
- Floating version resolution (1.* , [1.0,2.0))
- Central Package Management restore
- RID-specific assets

## Workflow

### High-Level Flow

```
1. Parse command arguments
2. Locate project/solution file
3. Load and parse project(s)
4. Extract PackageReference elements
5. Resolve dependencies (transitive)
6. Download packages to global cache
7. Extract package assets
8. Generate project.assets.json
9. Report restore summary
```

### Detailed Flow

```
┌─────────────────────────────────────┐
│ Parse arguments                     │
│ - Project/solution path (optional)  │
│ - Source, packages folder, flags    │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Locate project/solution file        │
│ - Use provided path OR              │
│ - Search current dir                │
│ - Error if 0 or >1 found            │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Load project file(s)                │
│ - Parse XML                         │
│ - Extract TargetFramework(s)        │
│ - Extract PackageReference elements │
│ - Extract package sources           │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Resolve dependencies                │
│ - For each PackageReference:        │
│   - Download .nuspec                │
│   - Parse dependencies              │
│   - Recurse for transitive deps     │
│ - Build dependency graph            │
│ - Detect conflicts                  │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Download packages                   │
│ - Check global cache first          │
│ - Download missing .nupkg files     │
│ - Extract to global cache           │
│ - Verify package hash (if available)│
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Select package assets               │
│ - For each package:                 │
│   - Read .nuspec                    │
│   - Select assemblies for TFM       │
│   - Select analyzers, build files   │
│   - Resolve RID-specific assets     │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Generate project.assets.json        │
│ - Write dependency graph            │
│ - Write resolved versions           │
│ - Write asset selections            │
│ - Write restore metadata            │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Report restore summary              │
│ - Packages restored count           │
│ - Errors, warnings                  │
│ - Total time elapsed                │
└─────────────────────────────────────┘
```

## dotnet restore Architecture

From research, `dotnet restore` delegates entirely to MSBuild:

```
dotnet restore
  ↓
RestoreCommand.FromParseResult() [sdk/src/Cli/dotnet/Commands/Restore/RestoreCommand.cs]
  ↓
MSBuildForwardingApp.Execute()
  ↓
MSBuild /t:Restore
  ↓
NuGet.Build.Tasks.dll (RestoreTask)
  ↓
NuGet.Commands.RestoreCommand
  ↓
Dependency resolution + package download
```

**Key insight:** There is NO custom restore logic in dotnet CLI itself. Everything is MSBuild targets.

**gonuget approach:** Implement restore natively in Go using existing gonuget packages.

## Go Implementation

### Command Entry Point

```go
// cmd/gonuget/commands/restore.go

package commands

import (
    "context"
    "fmt"
    "time"

    "github.com/spf13/cobra"
    "github.com/willibrandon/gonuget/cmd/gonuget/output"
    "github.com/willibrandon/gonuget/cmd/gonuget/project"
    "github.com/willibrandon/gonuget/core"
)

type restoreOptions struct {
    source         []string
    packagesFolder string
    configFile     string
    force          bool
    noCache        bool
    noDependencies bool
    verbosity      string
}

func NewRestoreCommand(console *output.Console) *cobra.Command {
    opts := &restoreOptions{}

    cmd := &cobra.Command{
        Use:   "restore [<PROJECT|SOLUTION>]",
        Short: "Restore NuGet packages",
        Long: `Restores packages based on PackageReference elements in the project file.

Downloads packages to the global package cache and generates project.assets.json.

Examples:
  gonuget restore
  gonuget restore MyApp.csproj
  gonuget restore --packages /custom/packages
  gonuget restore --force`,
        Args: cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            projectPath := ""
            if len(args) > 0 {
                projectPath = args[0]
            }
            return runRestore(cmd.Context(), console, projectPath, opts)
        },
    }

    cmd.Flags().StringSliceVar(&opts.source, "source", nil, "Package source(s)")
    cmd.Flags().StringVar(&opts.packagesFolder, "packages", "", "Global packages folder")
    cmd.Flags().StringVar(&opts.configFile, "configfile", "", "NuGet configuration file")
    cmd.Flags().BoolVar(&opts.force, "force", false, "Force re-download")
    cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "Don't use HTTP cache")
    cmd.Flags().BoolVar(&opts.noDependencies, "no-dependencies", false, "Skip transitive dependencies")
    cmd.Flags().StringVar(&opts.verbosity, "verbosity", "normal", "Output verbosity")

    return cmd
}

func runRestore(ctx context.Context, console *output.Console, projectPath string, opts *restoreOptions) error {
    start := time.Now()

    // 1. Locate project file
    if projectPath == "" {
        var err error
        projectPath, err = project.FindProjectFile(".")
        if err != nil {
            return fmt.Errorf("failed to locate project file: %w", err)
        }
    }

    console.Printf("  Restoring packages for %s...\n", projectPath)

    // 2. Load project
    proj, err := project.LoadProject(projectPath)
    if err != nil {
        return fmt.Errorf("failed to load project: %w", err)
    }

    // 3. Extract package references
    packageRefs := proj.GetPackageReferences()
    if len(packageRefs) == 0 {
        console.Println("  Nothing to restore")
        return nil
    }

    // 4. Create restore context
    restoreCtx := &RestoreContext{
        Project:        proj,
        Console:        console,
        PackagesFolder: opts.packagesFolder,
        Sources:        opts.source,
        Force:          opts.force,
        NoCache:        opts.noCache,
        NoDependencies: opts.noDependencies,
    }

    // 5. Resolve and download packages
    restorer := NewRestorer(restoreCtx)
    result, err := restorer.Restore(ctx, packageRefs)
    if err != nil {
        return fmt.Errorf("restore failed: %w", err)
    }

    // 6. Generate project.assets.json
    assetsFile := &AssetsFile{
        Version:   3,
        Targets:   result.Targets,
        Libraries: result.Libraries,
        ProjectFileDependencyGroups: result.DependencyGroups,
        PackageFolders: map[string]interface{}{
            result.PackagesFolder: map[string]interface{}{},
        },
        Project: &AssetsProject{
            Version: proj.GetVersion(),
            Restore: &AssetsRestore{
                ProjectUniqueName: projectPath,
                ProjectName:       proj.GetProjectName(),
                ProjectPath:       projectPath,
                PackagesPath:      result.PackagesFolder,
                OutputPath:        proj.GetOutputPath(),
                ProjectStyle:      "PackageReference",
                Frameworks:        result.Frameworks,
            },
        },
    }

    assetsPath := proj.GetAssetsFilePath()
    if err := assetsFile.Save(assetsPath); err != nil {
        return fmt.Errorf("failed to save project.assets.json: %w", err)
    }

    // 7. Report summary
    elapsed := time.Since(start)
    console.Printf("  Restored %s (in %d ms)\n", projectPath, elapsed.Milliseconds())

    return nil
}
```

### Restore Engine

```go
// cmd/gonuget/commands/restore_engine.go

package commands

import (
    "context"
    "fmt"
    "path/filepath"

    "github.com/willibrandon/gonuget/core"
    "github.com/willibrandon/gonuget/core/resolver"
    "github.com/willibrandon/gonuget/frameworks"
    "github.com/willibrandon/gonuget/packaging"
    "github.com/willibrandon/gonuget/version"
)

type Restorer struct {
    ctx    *RestoreContext
    client *core.Client
}

type RestoreContext struct {
    Project        *project.Project
    Console        *output.Console
    PackagesFolder string
    Sources        []string
    Force          bool
    NoCache        bool
    NoDependencies bool
}

type RestoreResult struct {
    PackagesFolder  string
    Targets         map[string]interface{}
    Libraries       map[string]interface{}
    DependencyGroups map[string][]string
    Frameworks      map[string]interface{}
}

func NewRestorer(ctx *RestoreContext) *Restorer {
    // Create client from context
    client := createClient(ctx)

    return &Restorer{
        ctx:    ctx,
        client: client,
    }
}

func (r *Restorer) Restore(ctx context.Context, packageRefs []*project.PackageReference) (*RestoreResult, error) {
    // Get target framework
    tfm, err := r.ctx.Project.GetTargetFramework()
    if err != nil {
        return nil, err
    }

    r.ctx.Console.Printf("  Target framework: %s\n", tfm)

    // Resolve dependencies
    var allPackages []*ResolvedPackage
    if r.ctx.NoDependencies {
        // Direct only
        allPackages = r.resolveDirectOnly(ctx, packageRefs, tfm)
    } else {
        // Full transitive resolution
        allPackages, err = r.resolveTransitive(ctx, packageRefs, tfm)
        if err != nil {
            return nil, err
        }
    }

    r.ctx.Console.Printf("  Resolved %d package(s)\n", len(allPackages))

    // Download packages
    packagesFolder := r.getPackagesFolder()
    for _, pkg := range allPackages {
        if err := r.downloadPackage(ctx, pkg, packagesFolder); err != nil {
            return nil, fmt.Errorf("failed to download %s: %w", pkg.ID, err)
        }
    }

    // Build restore result
    result := &RestoreResult{
        PackagesFolder:   packagesFolder,
        Targets:          r.buildTargets(tfm, allPackages),
        Libraries:        r.buildLibraries(allPackages),
        DependencyGroups: r.buildDependencyGroups(tfm, packageRefs),
        Frameworks:       r.buildFrameworks(tfm),
    }

    return result, nil
}

func (r *Restorer) resolveDirectOnly(ctx context.Context, packageRefs []*project.PackageReference, tfm *frameworks.NuGetFramework) []*ResolvedPackage {
    var packages []*ResolvedPackage

    for _, ref := range packageRefs {
        pkg := &ResolvedPackage{
            ID:      ref.Include,
            Version: ref.Version,
            Type:    "package",
        }
        packages = append(packages, pkg)
    }

    return packages
}

func (r *Restorer) resolveTransitive(ctx context.Context, packageRefs []*project.PackageReference, tfm *frameworks.NuGetFramework) ([]*ResolvedPackage, error) {
    // Use existing resolver package
    walker := resolver.NewDependencyWalker(r.client)

    // Convert to resolver format
    roots := make([]*resolver.GraphNode, 0, len(packageRefs))
    for _, ref := range packageRefs {
        ver, err := version.Parse(ref.Version)
        if err != nil {
            return nil, fmt.Errorf("invalid version for %s: %w", ref.Include, err)
        }

        root := &resolver.GraphNode{
            Item: &resolver.LibraryIdentity{
                Name:    ref.Include,
                Version: ver,
            },
            OuterEdge: nil,
        }
        roots = append(roots, root)
    }

    // Walk dependency graph
    graph, err := walker.WalkAsync(ctx, roots)
    if err != nil {
        return nil, err
    }

    // Flatten to package list
    packages := r.flattenGraph(graph)
    return packages, nil
}

func (r *Restorer) downloadPackage(ctx context.Context, pkg *ResolvedPackage, packagesFolder string) error {
    // Check if package already exists
    pkgPath := filepath.Join(packagesFolder, pkg.ID, pkg.Version)
    if !r.ctx.Force && packageExists(pkgPath) {
        return nil // Already downloaded
    }

    r.ctx.Console.Printf("  Downloading %s %s\n", pkg.ID, pkg.Version)

    // Download .nupkg
    reader, err := r.client.DownloadPackage(ctx, pkg.ID, pkg.Version)
    if err != nil {
        return err
    }
    defer reader.Close()

    // Extract to global cache
    extractor := packaging.NewPackageExtractor()
    if err := extractor.ExtractToDirectory(reader, pkgPath); err != nil {
        return err
    }

    return nil
}

func (r *Restorer) getPackagesFolder() string {
    if r.ctx.PackagesFolder != "" {
        return r.ctx.PackagesFolder
    }

    // Default: ~/.nuget/packages
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".nuget", "packages")
}
```

### project.assets.json Generation

```go
// cmd/gonuget/commands/assets_file.go

package commands

import (
    "encoding/json"
    "fmt"
    "os"
)

// AssetsFile represents project.assets.json structure
type AssetsFile struct {
    Version                     int                         `json:"version"`
    Targets                     map[string]interface{}      `json:"targets"`
    Libraries                   map[string]interface{}      `json:"libraries"`
    ProjectFileDependencyGroups map[string][]string         `json:"projectFileDependencyGroups"`
    PackageFolders              map[string]interface{}      `json:"packageFolders"`
    Project                     *AssetsProject              `json:"project"`
}

type AssetsProject struct {
    Version string        `json:"version"`
    Restore *AssetsRestore `json:"restore"`
}

type AssetsRestore struct {
    ProjectUniqueName string                 `json:"projectUniqueName"`
    ProjectName       string                 `json:"projectName"`
    ProjectPath       string                 `json:"projectPath"`
    PackagesPath      string                 `json:"packagesPath"`
    OutputPath        string                 `json:"outputPath"`
    ProjectStyle      string                 `json:"projectStyle"`
    Frameworks        map[string]interface{} `json:"frameworks"`
}

func (af *AssetsFile) Save(path string) error {
    data, err := json.MarshalIndent(af, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal assets file: %w", err)
    }

    if err := os.WriteFile(path, data, 0644); err != nil {
        return fmt.Errorf("failed to write assets file: %w", err)
    }

    return nil
}
```

## project.assets.json Format

### Minimal Example

```json
{
  "version": 3,
  "targets": {
    ".NETCoreApp,Version=v8.0": {
      "Newtonsoft.Json/13.0.3": {
        "type": "package",
        "compile": {
          "lib/net6.0/Newtonsoft.Json.dll": {}
        },
        "runtime": {
          "lib/net6.0/Newtonsoft.Json.dll": {}
        }
      }
    }
  },
  "libraries": {
    "Newtonsoft.Json/13.0.3": {
      "sha512": "...",
      "type": "package",
      "path": "newtonsoft.json/13.0.3",
      "files": [
        ".nupkg.metadata",
        "newtonsoft.json.13.0.3.nupkg.sha512",
        "newtonsoft.json.nuspec",
        "lib/net6.0/Newtonsoft.Json.dll",
        "lib/net6.0/Newtonsoft.Json.xml"
      ]
    }
  },
  "projectFileDependencyGroups": {
    ".NETCoreApp,Version=v8.0": [
      "Newtonsoft.Json >= 13.0.3"
    ]
  },
  "packageFolders": {
    "/Users/brandon/.nuget/packages/": {}
  },
  "project": {
    "version": "1.0.0",
    "restore": {
      "projectUniqueName": "/path/to/MyProject.csproj",
      "projectName": "MyProject",
      "projectPath": "/path/to/MyProject.csproj",
      "packagesPath": "/Users/brandon/.nuget/packages/",
      "outputPath": "/path/to/obj/",
      "projectStyle": "PackageReference",
      "frameworks": {
        "net8.0": {
          "targetAlias": "net8.0",
          "projectReferences": {}
        }
      }
    }
  }
}
```

## Testing Strategy

### Unit Tests

```go
// cmd/gonuget/commands/restore_test.go

func TestRestore_Simple(t *testing.T) {
    // Create test project with PackageReference
    proj := createTestProject(t, "net8.0", []*project.PackageReference{
        {Include: "Newtonsoft.Json", Version: "13.0.3"},
    })

    // Run restore
    ctx := context.Background()
    opts := &restoreOptions{}
    err := runRestore(ctx, &output.Console{}, proj.Path, opts)
    require.NoError(t, err)

    // Verify package downloaded
    packagesFolder := getPackagesFolder()
    pkgPath := filepath.Join(packagesFolder, "newtonsoft.json", "13.0.3")
    assert.DirExists(t, pkgPath)

    // Verify project.assets.json created
    assetsPath := filepath.Join(filepath.Dir(proj.Path), "obj", "project.assets.json")
    assert.FileExists(t, assetsPath)

    // Verify assets file content
    assets := loadAssetsFile(t, assetsPath)
    assert.Contains(t, assets.Libraries, "Newtonsoft.Json/13.0.3")
}
```

### CLI Interop Tests

```csharp
// tests/cli-interop/GonugetCliInterop.Tests/RestoreTests.cs

[Fact]
public async Task Restore_Simple_MatchesDotnet()
{
    // Create test project with PackageReference
    var projectPath = CreateTestProjectWithPackageReference("net8.0",
        "Newtonsoft.Json", "13.0.3");

    // Run dotnet restore
    await RunDotnetCommand($"restore {projectPath}");
    var dotnetAssets = LoadAssetsFile(projectPath);

    // Clean obj/ folder
    CleanObjFolder(projectPath);

    // Run gonuget restore
    var result = await ExecuteGonugetCommand(new RestoreRequest
    {
        ProjectPath = projectPath
    });
    var gonugetAssets = LoadAssetsFile(projectPath);

    // Compare key fields in project.assets.json
    AssertAssetsEqual(dotnetAssets, gonugetAssets);
}
```

## Integration with gonuget Packages

Restore leverages existing gonuget infrastructure:

### Used Packages

- `core.Client` - Package download and metadata
- `core/resolver.DependencyWalker` - Transitive dependency resolution
- `packaging.PackageExtractor` - .nupkg extraction
- `frameworks.NuGetFramework` - TFM parsing and compatibility
- `version.NuGetVersion` - Version parsing and comparison
- `cache` - HTTP and metadata caching

### New Code Required

- Project file parsing (`cmd/gonuget/project/`)
- PackageReference extraction
- project.assets.json generation
- CLI command implementation

## Error Handling

### Error Scenarios

1. **No project file found:**
```
error: No project file found in current directory.
```

2. **Package not found:**
```
error: Package 'InvalidPackage' not found in configured sources.
Restore failed.
```

3. **Version conflict:**
```
error: Package version conflict detected:
  Project requires Newtonsoft.Json 13.0.3
  Transitive dependency requires Newtonsoft.Json 12.0.0
Resolve the conflict and try again.
```

4. **Download failure:**
```
error: Failed to download Newtonsoft.Json 13.0.3
  Network error: connection timeout
Restore failed.
```

5. **Corrupted package:**
```
error: Package hash mismatch for Newtonsoft.Json 13.0.3
  Expected: abc123...
  Actual: def456...
Delete the package from cache and try again with --force.
```

## Output Compatibility

Match dotnet restore output format:

**dotnet restore output:**
```
  Determining projects to restore...
  Restored /path/to/MyProject.csproj (in 234 ms).
```

**gonuget restore output:**
```
  Restoring packages for /path/to/MyProject.csproj...
  Target framework: net8.0
  Resolved 5 package(s)
  Downloading Newtonsoft.Json 13.0.3
  Restored /path/to/MyProject.csproj (in 234 ms)
```

## Performance Considerations

### Optimization Targets

- Project file parsing: <10ms
- Dependency resolution: <1s (cached metadata)
- Package download: Network dependent
- Asset extraction: <100ms per package
- project.assets.json generation: <50ms
- Total restore (cached): <2s
- Total restore (uncached): Depends on package count and size

### Parallel Downloads

Download multiple packages concurrently:

```go
func (r *Restorer) downloadPackages(ctx context.Context, packages []*ResolvedPackage, packagesFolder string) error {
    // Create worker pool
    const maxConcurrent = 8
    sem := make(chan struct{}, maxConcurrent)
    errChan := make(chan error, len(packages))
    var wg sync.WaitGroup

    for _, pkg := range packages {
        wg.Add(1)
        go func(pkg *ResolvedPackage) {
            defer wg.Done()
            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release

            if err := r.downloadPackage(ctx, pkg, packagesFolder); err != nil {
                errChan <- err
            }
        }(pkg)
    }

    wg.Wait()
    close(errChan)

    // Check for errors
    if err := <-errChan; err != nil {
        return err
    }

    return nil
}
```

## Future Enhancements (Post-M2)

### Solution Restore

- Parse .sln file
- Restore all projects in solution
- Respect solution configuration

### Lock File Support

- Generate packages.lock.json
- Locked mode restore (exact versions)
- Lock file validation

### RID-Specific Restore

- Parse RuntimeIdentifier property
- Download native libraries for RID
- Select RID-specific assets

## Success Criteria

- [ ] Restore packages from .csproj
- [ ] Download to global package cache
- [ ] Generate valid project.assets.json
- [ ] Transitive dependency resolution
- [ ] CLI interop tests pass
- [ ] Performance targets met
- [ ] Error handling comprehensive
- [ ] Integration with existing gonuget packages
