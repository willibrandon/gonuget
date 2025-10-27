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

## M2.1 Chunks

This guide covers chunks 5-7 and 9 from the CLI implementation roadmap.

---

## Chunk 5: Restore Command - Direct Dependencies Only

**Objective:** Implement basic `restore` command that downloads direct PackageReference dependencies.

**Prerequisites:** Chunks 1-4 complete

**Files to Create:**
- `cmd/gonuget/commands/restore.go`
- `cmd/gonuget/commands/restore_test.go`

**Files to Modify:**
- `cmd/gonuget/cli/app.go` - Register command

**Scope:**
- Read PackageReference elements from .csproj
- Download packages to global package cache (~/.nuget/packages)
- Generate basic project.assets.json
- Single project only (no solution support)
- Direct dependencies only (no transitive)
- Single target framework

**Implementation:** See "Chunk 5 Implementation Details" section below.

---

## Chunk 6: project.assets.json Generation

**Objective:** Generate project.assets.json file after restore.

**Prerequisites:** Chunk 5 complete

**Implementation:** This is part of Chunk 5 - the restore command generates project.assets.json.

**Verification:** After restore, verify project.assets.json exists in obj/ directory.

---

## Chunk 7: Global Package Cache Integration

**Objective:** Integrate with global package cache (~/.nuget/packages).

**Prerequisites:** Chunks 5-6 complete

**Implementation:** This is part of Chunk 5 - packages are downloaded to global cache.

**Verification:** After restore, verify packages exist in ~/.nuget/packages/{packageid}/{version}/

---

## Chunk 9: CLI Interop Tests for Restore

**Objective:** Create CLI interop tests comparing gonuget restore vs dotnet restore.

**Prerequisites:** Chunks 5-7 complete

**Files to Create:**
- `tests/cli-interop/GonugetCliInterop.Tests/RestoreTests.cs`

**Implementation:** See Chunk 9 of CLI-M2-ADD-PACKAGE.md for interop test examples.

---

## M2.2 - Advanced Restore Features

**Chunks 15-17** implement advanced features:
- Chunk 15: Transitive dependency resolution
- Chunk 16: Multi-TFM support
- Chunk 17: Solution file support

**Deferred to Future:**
- Floating version resolution (1.* , [1.0,2.0))
- Central Package Management restore
- RID-specific assets

---

# Chunk 5 Implementation Details

This section provides the detailed implementation for Chunk 5: Restore Command.

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
    "io"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "github.com/willibrandon/gonuget/cmd/gonuget/output"
    "github.com/willibrandon/gonuget/cmd/gonuget/project"
    "github.com/willibrandon/gonuget/core"
    "github.com/willibrandon/gonuget/frameworks"
    "github.com/willibrandon/gonuget/packaging"
    "github.com/willibrandon/gonuget/packaging/assets"
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
    // Use existing resolver via core.Client
    // core.Client has ResolvePackageDependencies which uses resolver internally

    var allPackages []*ResolvedPackage
    seen := make(map[string]bool)

    for _, ref := range packageRefs {
        // Resolve each package and its dependencies
        result, err := r.client.ResolvePackageDependencies(ctx, ref.Include, ref.Version)
        if err != nil {
            return nil, fmt.Errorf("failed to resolve %s: %w", ref.Include, err)
        }

        // Add all resolved packages (including transitive)
        for _, pkg := range result.Packages {
            key := fmt.Sprintf("%s/%s", pkg.ID, pkg.Version)
            if !seen[key] {
                seen[key] = true
                allPackages = append(allPackages, &ResolvedPackage{
                    ID:      pkg.ID,
                    Version: pkg.Version,
                    Type:    "package",
                })
            }
        }
    }

    return allPackages, nil
}

func (r *Restorer) downloadPackage(ctx context.Context, pkg *ResolvedPackage, packagesFolder string) error {
    // Create package identity
    pkgVer, err := version.Parse(pkg.Version)
    if err != nil {
        return fmt.Errorf("invalid version: %w", err)
    }

    packageIdentity := &packaging.PackageIdentity{
        ID:      pkg.ID,
        Version: pkgVer,
    }

    // Create version folder path resolver
    versionFolderResolver := packaging.NewVersionFolderPathResolver(packagesFolder)

    // Check if package already exists (via completion marker .nupkg.metadata)
    metadataPath := versionFolderResolver.GetNupkgMetadataPath(pkg.ID, pkg.Version)
    if !r.ctx.Force {
        if _, err := os.Stat(metadataPath); err == nil {
            return nil // Already installed
        }
    }

    r.ctx.Console.Printf("  Downloading %s %s\n", pkg.ID, pkg.Version)

    // Create extraction context
    extractionContext := &packaging.PackageExtractionContext{
        PackageSaveMode:    packaging.PackageSaveModeNupkg | packaging.PackageSaveModeNuspec | packaging.PackageSaveModeFiles,
        XMLDocFileSaveMode: packaging.XMLDocFileSaveModeNone,
        SignatureVerifier:  nil, // No signature verification in M2.1
        Logger:             nil,
    }

    // Download callback - downloads .nupkg to temp location
    copyToAsync := func(targetPath string) error {
        // Download package stream
        stream, err := r.client.DownloadPackage(ctx, pkg.ID, pkg.Version)
        if err != nil {
            return err
        }
        defer stream.Close()

        // Write to temp file
        outFile, err := os.Create(targetPath)
        if err != nil {
            return err
        }
        defer outFile.Close()

        _, err = io.Copy(outFile, stream)
        return err
    }

    // Get source URL (from first configured repository)
    repos := r.client.GetRepositoryManager().ListRepositories()
    var sourceURL string
    if len(repos) > 0 {
        sourceURL = repos[0].SourceURL()
    }

    // Install package using V3 extraction (with file locking for concurrent safety)
    _, err = packaging.InstallFromSourceV3(
        ctx,
        sourceURL,
        packageIdentity,
        copyToAsync,
        versionFolderResolver,
        extractionContext,
    )

    return err
}

func (r *Restorer) getPackagesFolder() string {
    if r.ctx.PackagesFolder != "" {
        return r.ctx.PackagesFolder
    }

    // Default: ~/.nuget/packages
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".nuget", "packages")
}

// buildTargets builds the targets section of project.assets.json
func (r *Restorer) buildTargets(tfm *frameworks.NuGetFramework, packages []*ResolvedPackage) map[string]any {
    targets := make(map[string]any)

    // Target key format: ".NETCoreApp,Version=v8.0"
    provider := frameworks.DefaultFrameworkNameProvider()
    targetKey := tfm.DotNetFrameworkName(provider)

    // Create per-package entries
    targetLibraries := make(map[string]any)

    packagesFolder := r.getPackagesFolder()
    pathResolver := packaging.NewVersionFolderPathResolver(packagesFolder, true)

    for _, pkg := range packages {
        // Parse version
        pkgVer, err := version.Parse(pkg.Version)
        if err != nil {
            continue
        }

        // Select assets and dependencies
        compile, runtime, dependencies := r.selectAssets(pkg.ID, pkgVer, tfm, pathResolver)

        // Build target library entry
        targetLib := map[string]any{
            "type": "package",
        }

        if len(dependencies) > 0 {
            targetLib["dependencies"] = dependencies
        }
        if len(compile) > 0 {
            targetLib["compile"] = compile
        }
        if len(runtime) > 0 {
            targetLib["runtime"] = runtime
        }

        targetLibraries[fmt.Sprintf("%s/%s", pkg.ID, pkg.Version)] = targetLib
    }

    targets[targetKey] = targetLibraries
    return targets
}

// selectAssets selects compile/runtime assets and dependencies for the target framework
func (r *Restorer) selectAssets(packageID string, pkgVer *version.NuGetVersion, tfm *frameworks.NuGetFramework, pathResolver *packaging.VersionFolderPathResolver) (map[string]any, map[string]any, map[string]string) {
    compile := make(map[string]any)
    runtime := make(map[string]any)
    dependencies := make(map[string]string)

    // Open package from global cache
    nupkgPath := pathResolver.GetPackageFilePath(packageID, pkgVer)
    pkgReader, err := packaging.OpenPackage(nupkgPath)
    if err != nil {
        return compile, runtime, dependencies
    }
    defer pkgReader.Close()

    // Read nuspec for dependencies
    nuspec, err := pkgReader.GetNuspec()
    if err == nil {
        depGroups, err := nuspec.GetDependencyGroups()
        if err == nil {
            // Find nearest compatible dependency group
            reducer := frameworks.NewFrameworkReducer()
            var compatible []*frameworks.NuGetFramework
            for i := range depGroups {
                if tfm.IsCompatibleWith(depGroups[i].TargetFramework) {
                    compatible = append(compatible, depGroups[i].TargetFramework)
                }
            }
            if len(compatible) > 0 {
                nearest := reducer.GetNearest(tfm, compatible)
                for i := range depGroups {
                    if depGroups[i].TargetFramework.Equals(nearest) {
                        for _, dep := range depGroups[i].Dependencies {
                            versionStr := ">= 1.0.0"
                            if dep.VersionRange != nil {
                                versionStr = dep.VersionRange.String()
                            }
                            dependencies[dep.ID] = versionStr
                        }
                        break
                    }
                }
            }
        }
    }

    // Use asset selector to find compile/runtime assemblies
    conventions := assets.NewManagedCodeConventions()
    criteria := assets.ForFramework(tfm, conventions.Properties)
    collection := assets.NewContentItemCollection()

    var filePaths []string
    for _, f := range pkgReader.Files() {
        if !strings.HasSuffix(f.Name, "/") {
            filePaths = append(filePaths, f.Name)
        }
    }
    collection.Load(filePaths)

    // Select compile assets (ref/ or lib/)
    compileGroup := collection.FindBestItemGroup(criteria, conventions.CompileRefAssemblies, conventions.CompileLibAssemblies)
    if compileGroup != nil {
        for _, item := range compileGroup.Items {
            compile[item.Path] = map[string]any{}
        }
    }

    // Select runtime assets (lib/)
    runtimeGroup := collection.FindBestItemGroup(criteria, conventions.RuntimeAssemblies)
    if runtimeGroup != nil {
        for _, item := range runtimeGroup.Items {
            runtime[item.Path] = map[string]any{}
        }
    }

    return compile, runtime, dependencies
}

// buildLibraries builds the libraries section with package metadata
func (r *Restorer) buildLibraries(packages []*ResolvedPackage) map[string]any {
    libraries := make(map[string]any)

    packagesFolder := r.getPackagesFolder()
    pathResolver := packaging.NewVersionFolderPathResolver(packagesFolder, true)

    for _, pkg := range packages {
        pkgVer, err := version.Parse(pkg.Version)
        if err != nil {
            continue
        }

        lib := map[string]any{
            "type": "package",
        }

        // Add SHA512
        hashPath := pathResolver.GetHashPath(pkg.ID, pkgVer)
        if hashData, err := os.ReadFile(hashPath); err == nil {
            lib["sha512"] = strings.TrimSpace(string(hashData))
        }

        // Add path (lowercase)
        normalizedID := strings.ToLower(pkg.ID)
        normalizedVer := strings.ToLower(pkgVer.ToNormalizedString())
        lib["path"] = fmt.Sprintf("%s/%s", normalizedID, normalizedVer)

        // Add files list
        nupkgPath := pathResolver.GetPackageFilePath(pkg.ID, pkgVer)
        pkgReader, err := packaging.OpenPackage(nupkgPath)
        if err == nil {
            var files []string
            for _, f := range pkgReader.Files() {
                if !strings.HasSuffix(f.Name, "/") {
                    files = append(files, strings.ToLower(strings.ReplaceAll(f.Name, "\\", "/")))
                }
            }
            pkgReader.Close()
            sort.Strings(files)
            lib["files"] = files
        }

        libraries[fmt.Sprintf("%s/%s", pkg.ID, pkg.Version)] = lib
    }

    return libraries
}

// buildDependencyGroups builds the projectFileDependencyGroups section
func (r *Restorer) buildDependencyGroups(tfm *frameworks.NuGetFramework, packageRefs []project.PackageReference) map[string][]string {
    groups := make(map[string][]string)
    provider := frameworks.DefaultFrameworkNameProvider()
    targetKey := tfm.DotNetFrameworkName(provider)

    var deps []string
    for _, ref := range packageRefs {
        pkgVer, err := version.Parse(ref.Version)
        if err != nil {
            continue
        }
        deps = append(deps, fmt.Sprintf("%s >= %s", ref.Include, pkgVer.ToNormalizedString()))
    }

    sort.Strings(deps)
    groups[targetKey] = deps
    return groups
}

// buildFrameworks builds the frameworks section
func (r *Restorer) buildFrameworks(tfm *frameworks.NuGetFramework) map[string]any {
    frameworks := make(map[string]any)
    provider := frameworks.DefaultFrameworkNameProvider()
    shortName := tfm.GetShortFolderName(provider)

    frameworks[shortName] = map[string]any{
        "targetAlias":       shortName,
        "projectReferences": map[string]any{},
    }
    return frameworks
}

func createClient(ctx *RestoreContext) *core.Client {
    repoManager := core.NewRepositoryManager()

    sources := ctx.Sources
    if len(sources) == 0 {
        sources = []string{"https://api.nuget.org/v3/index.json"}
    }

    for _, sourceURL := range sources {
        repoManager.AddRepository(sourceURL)
    }

    tfm, _ := ctx.Project.GetTargetFramework()

    return core.NewClient(core.ClientConfig{
        RepositoryManager: repoManager,
        TargetFramework:   tfm,
    })
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
