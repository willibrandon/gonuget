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

**Library Package (restore/) - All NuGet.Client Logic**:
- `restore/restorer.go` - Core restore engine (port of NuGet.Commands/RestoreCommand/RestoreCommand.cs)
- `restore/lock_file_builder.go` - Builds project.assets.json (port of NuGet.Commands/RestoreCommand/LockFileBuilder.cs)
- `restore/lock_file_format.go` - LockFile types and JSON serialization (port of NuGet.ProjectModel/LockFile*.cs)
- `restore/restorer_test.go` - Unit tests for restore logic

**CLI Commands (cmd/gonuget/commands/) - Thin Wrappers Only**:
- `cmd/gonuget/commands/restore.go` - CLI flags and argument parsing, calls restore.Run()
- `cmd/gonuget/commands/restore_test.go` - CLI integration tests

**Files to Modify:**
- `cmd/gonuget/cli/app.go` - Register restore command
- `cmd/gonuget/commands/add_package.go` - Refactor to call library instead of having resolveLatestVersion() logic in CLI

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

### Command Entry Point (CLI - Thin Wrapper)

The CLI command is a thin wrapper that parses flags and calls the library.

```go
// cmd/gonuget/commands/restore.go

package commands

import (
    "github.com/spf13/cobra"
    "github.com/willibrandon/gonuget/cmd/gonuget/output"
    "github.com/willibrandon/gonuget/restore"
)

// NewRestoreCommand creates the restore command.
func NewRestoreCommand(console *output.Console) *cobra.Command {
    opts := &restore.Options{}

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
            // CLI just calls library function
            return restore.Run(cmd.Context(), args, opts, console)
        },
    }

    // Flag binding
    cmd.Flags().StringSliceVarP(&opts.Sources, "source", "s", nil, "Package source(s) to use")
    cmd.Flags().StringVar(&opts.PackagesFolder, "packages", "", "Custom global packages folder")
    cmd.Flags().StringVar(&opts.ConfigFile, "configfile", "", "NuGet configuration file")
    cmd.Flags().BoolVar(&opts.Force, "force", false, "Force re-download even if packages exist")
    cmd.Flags().BoolVar(&opts.NoCache, "no-cache", false, "Don't use HTTP cache")
    cmd.Flags().BoolVar(&opts.NoDependencies, "no-dependencies", false, "Only restore direct references")
    cmd.Flags().StringVar(&opts.Verbosity, "verbosity", "normal", "Verbosity level")

    return cmd
}
```

**Key Points**:
- CLI has NO business logic
- Just flag parsing and calling `restore.Run()`
- All NuGet logic is in the library

---

### Library Implementation (restore/ package)

All restore logic goes in the library package, ported from NuGet.Client.

#### restore/restorer.go

Core restore engine ported from NuGet.Commands/RestoreCommand/RestoreCommand.cs (1,400+ lines).

```go
// restore/restorer.go

package restore

import (
    "context"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"

    "github.com/willibrandon/gonuget/cmd/gonuget/project"
    "github.com/willibrandon/gonuget/core"
    "github.com/willibrandon/gonuget/frameworks"
    "github.com/willibrandon/gonuget/packaging"
    "github.com/willibrandon/gonuget/version"
)

// Console interface for output (injected from CLI)
type Console interface {
    Printf(format string, args ...any)
    Error(msg string)
    Warning(msg string)
}

// Options holds restore configuration
type Options struct {
    Sources        []string
    PackagesFolder string
    ConfigFile     string
    Force          bool
    NoCache        bool
    NoDependencies bool
    Verbosity      string
}

// Run executes the restore operation (entry point called from CLI)
func Run(ctx context.Context, args []string, opts *Options, console Console) error {
    start := time.Now()

    // 1. Find project file
    projectPath, err := findProjectFile(args)
    if err != nil {
        return err
    }

    console.Printf("Restoring packages for %s...\n", projectPath)

    // 2. Load project
    proj, err := project.LoadProject(projectPath)
    if err != nil {
        return fmt.Errorf("failed to load project: %w", err)
    }

    // 3. Get package references
    packageRefs, err := proj.GetPackageReferences()
    if err != nil {
        return fmt.Errorf("failed to get package references: %w", err)
    }

    if len(packageRefs) == 0 {
        console.Printf("Nothing to restore\n")
        return nil
    }

    // 4. Create restorer
    restorer := NewRestorer(opts, console)

    // 5. Execute restore
    result, err := restorer.Restore(ctx, proj, packageRefs)
    if err != nil {
        return fmt.Errorf("restore failed: %w", err)
    }

    // 6. Generate lock file (project.assets.json)
    lockFile := NewLockFileBuilder().Build(proj, result)
    assetsPath := proj.GetAssetsFilePath()
    if err := lockFile.Save(assetsPath); err != nil {
        return fmt.Errorf("failed to save project.assets.json: %w", err)
    }

    // 7. Report summary
    elapsed := time.Since(start)
    console.Printf("  Restored %s (in %d ms)\n", projectPath, elapsed.Milliseconds())

    return nil
}

func findProjectFile(args []string) (string, error) {
    if len(args) > 0 {
        return args[0], nil
    }

    cwd, err := os.Getwd()
    if err != nil {
        return "", err
    }

    return project.FindProjectFile(cwd)
}

// Restorer executes restore operations
type Restorer struct {
    opts    *Options
    console Console
    client  *core.Client
}

// NewRestorer creates a new restorer
func NewRestorer(opts *Options, console Console) *Restorer {
    // Create client
    repoManager := core.NewRepositoryManager()

    sources := opts.Sources
    if len(sources) == 0 {
        sources = []string{"https://api.nuget.org/v3/index.json"}
    }

    for _, sourceURL := range sources {
        repo := core.NewSourceRepository(core.RepositoryConfig{
            SourceURL: sourceURL,
        })
        repoManager.AddRepository(repo)
    }

    return &Restorer{
        opts:    opts,
        console: console,
        client: core.NewClient(core.ClientConfig{
            RepositoryManager: repoManager,
        }),
    }
}

// RestoreResult holds restore operation results
type RestoreResult struct {
    Packages       []*ResolvedPackage
    PackagesFolder string
    TargetFramework *frameworks.NuGetFramework
}

// ResolvedPackage represents a resolved package
type ResolvedPackage struct {
    ID      string
    Version string
}

// Restore performs the restore operation
func (r *Restorer) Restore(ctx context.Context, proj *project.Project, packageRefs []project.PackageReference) (*RestoreResult, error) {
    // Get target framework
    tfm, err := proj.GetTargetFramework()
    if err != nil {
        return nil, err
    }

    r.console.Printf("  Target framework: %s\n", tfm)

    // Resolve packages (direct only for Chunk 5)
    packages := r.resolveDirectOnly(packageRefs)

    r.console.Printf("  Resolved %d package(s)\n", len(packages))

    // Download packages
    packagesFolder := r.getPackagesFolder()
    for _, pkg := range packages {
        if err := r.downloadPackage(ctx, pkg, packagesFolder); err != nil {
            return nil, fmt.Errorf("failed to download %s: %w", pkg.ID, err)
        }
    }

    return &RestoreResult{
        Packages:        packages,
        PackagesFolder:  packagesFolder,
        TargetFramework: tfm,
    }, nil
}

func (r *Restorer) resolveDirectOnly(packageRefs []project.PackageReference) []*ResolvedPackage {
    var packages []*ResolvedPackage
    for _, ref := range packageRefs {
        packages = append(packages, &ResolvedPackage{
            ID:      ref.Include,
            Version: ref.Version,
        })
    }
    return packages
}

func (r *Restorer) downloadPackage(ctx context.Context, pkg *ResolvedPackage, packagesFolder string) error {
    pkgVer, err := version.Parse(pkg.Version)
    if err != nil {
        return err
    }

    // Check if already installed
    pathResolver := packaging.NewVersionFolderPathResolver(packagesFolder, true)
    metadataPath := pathResolver.GetNupkgMetadataPath(pkg.ID, pkg.Version)
    if !r.opts.Force {
        if _, err := os.Stat(metadataPath); err == nil {
            return nil // Already installed
        }
    }

    r.console.Printf("  Downloading %s %s\n", pkg.ID, pkg.Version)

    // Download and extract using library API
    packageIdentity := &packaging.PackageIdentity{
        ID:      pkg.ID,
        Version: pkgVer,
    }

    extractionContext := &packaging.PackageExtractionContext{
        PackageSaveMode:    packaging.PackageSaveModeNupkg | packaging.PackageSaveModeNuspec | packaging.PackageSaveModeFiles,
        XMLDocFileSaveMode: packaging.XMLDocFileSaveModeNone,
    }

    copyToAsync := func(targetPath string) error {
        stream, err := r.client.DownloadPackage(ctx, pkg.ID, pkg.Version)
        if err != nil {
            return err
        }
        defer stream.Close()

        outFile, err := os.Create(targetPath)
        if err != nil {
            return err
        }
        defer outFile.Close()

        _, err = io.Copy(outFile, stream)
        return err
    }

    repos := r.client.GetRepositoryManager().ListRepositories()
    var sourceURL string
    if len(repos) > 0 {
        sourceURL = repos[0].SourceURL()
    }

    _, err = packaging.InstallFromSourceV3(
        ctx,
        sourceURL,
        packageIdentity,
        copyToAsync,
        pathResolver,
        extractionContext,
    )

    return err
}

func (r *Restorer) getPackagesFolder() string {
    if r.opts.PackagesFolder != "" {
        return r.opts.PackagesFolder
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".nuget", "packages")
}
```

**Reference**: Port from `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/RestoreCommand.cs`

---

#### restore/lock_file_builder.go

Builds project.assets.json, ported from NuGet.Commands/RestoreCommand/LockFileBuilder.cs (682 lines).

```go
// restore/lock_file_builder.go

package restore

import (
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "github.com/willibrandon/gonuget/cmd/gonuget/project"
    "github.com/willibrandon/gonuget/frameworks"
    "github.com/willibrandon/gonuget/packaging"
    "github.com/willibrandon/gonuget/packaging/assets"
    "github.com/willibrandon/gonuget/version"
)

// LockFileBuilder builds project.assets.json from restore results
type LockFileBuilder struct {
    pathResolver *packaging.VersionFolderPathResolver
}

// NewLockFileBuilder creates a new lock file builder
func NewLockFileBuilder() *LockFileBuilder {
    return &LockFileBuilder{}
}

// Build creates a LockFile from restore results
func (b *LockFileBuilder) Build(proj *project.Project, result *RestoreResult) *LockFile {
    b.pathResolver = packaging.NewVersionFolderPathResolver(result.PackagesFolder, true)

    projectPath, _ := filepath.Abs(proj.FilePath)
    projectName := filepath.Base(filepath.Dir(projectPath))

    return &LockFile{
        Version:                     3,
        Targets:                     b.buildTargets(result.TargetFramework, result.Packages),
        Libraries:                   b.buildLibraries(result.Packages),
        ProjectFileDependencyGroups: b.buildDependencyGroups(result.TargetFramework, result.Packages),
        PackageFolders: map[string]any{
            result.PackagesFolder: map[string]any{},
        },
        Project: &LockFileProject{
            Version: "1.0.0",
            Restore: &LockFileRestore{
                ProjectUniqueName: projectPath,
                ProjectName:       projectName,
                ProjectPath:       projectPath,
                PackagesPath:      result.PackagesFolder,
                OutputPath:        proj.GetOutputPath(),
                ProjectStyle:      "PackageReference",
                Frameworks:        b.buildFrameworks(result.TargetFramework),
            },
        },
    }
}

// buildTargets builds the targets section (ported from LockFileBuilder.CreateTargetSection)
func (b *LockFileBuilder) buildTargets(tfm *frameworks.NuGetFramework, packages []*ResolvedPackage) map[string]any {
    targets := make(map[string]any)
    provider := frameworks.DefaultFrameworkNameProvider()
    targetKey := tfm.DotNetFrameworkName(provider)

    targetLibraries := make(map[string]any)

    for _, pkg := range packages {
        pkgVer, err := version.Parse(pkg.Version)
        if err != nil {
            continue
        }

        // Select assets and dependencies for this package
        compile, runtime, dependencies := b.selectAssets(pkg.ID, pkgVer, tfm)

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

// selectAssets selects compile/runtime assets (ported from LockFileBuilder)
func (b *LockFileBuilder) selectAssets(packageID string, pkgVer *version.NuGetVersion, tfm *frameworks.NuGetFramework) (map[string]any, map[string]any, map[string]string) {
    compile := make(map[string]any)
    runtime := make(map[string]any)
    dependencies := make(map[string]string)

    nupkgPath := b.pathResolver.GetPackageFilePath(packageID, pkgVer)
    pkgReader, err := packaging.OpenPackage(nupkgPath)
    if err != nil {
        return compile, runtime, dependencies
    }
    defer pkgReader.Close()

    // Get dependencies from nuspec
    nuspec, _ := pkgReader.GetNuspec()
    if nuspec != nil {
        depGroups, _ := nuspec.GetDependencyGroups()
        if depGroups != nil {
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

    // Select assets using ManagedCodeConventions
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

    compileGroup := collection.FindBestItemGroup(criteria, conventions.CompileRefAssemblies, conventions.CompileLibAssemblies)
    if compileGroup != nil {
        for _, item := range compileGroup.Items {
            compile[item.Path] = map[string]any{}
        }
    }

    runtimeGroup := collection.FindBestItemGroup(criteria, conventions.RuntimeAssemblies)
    if runtimeGroup != nil {
        for _, item := range runtimeGroup.Items {
            runtime[item.Path] = map[string]any{}
        }
    }

    return compile, runtime, dependencies
}

// buildLibraries builds the libraries section (ported from LockFileBuilder.CreateLibrarySection)
func (b *LockFileBuilder) buildLibraries(packages []*ResolvedPackage) map[string]any {
    libraries := make(map[string]any)

    for _, pkg := range packages {
        pkgVer, err := version.Parse(pkg.Version)
        if err != nil {
            continue
        }

        lib := map[string]any{
            "type": "package",
        }

        // Add SHA512
        hashPath := b.pathResolver.GetHashPath(pkg.ID, pkgVer)
        if hashData, err := os.ReadFile(hashPath); err == nil {
            lib["sha512"] = strings.TrimSpace(string(hashData))
        }

        // Add normalized path
        normalizedID := strings.ToLower(pkg.ID)
        normalizedVer := strings.ToLower(pkgVer.ToNormalizedString())
        lib["path"] = fmt.Sprintf("%s/%s", normalizedID, normalizedVer)

        // Add sorted file list
        nupkgPath := b.pathResolver.GetPackageFilePath(pkg.ID, pkgVer)
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

// buildDependencyGroups builds projectFileDependencyGroups section
func (b *LockFileBuilder) buildDependencyGroups(tfm *frameworks.NuGetFramework, packages []*ResolvedPackage) map[string][]string {
    groups := make(map[string][]string)
    provider := frameworks.DefaultFrameworkNameProvider()
    targetKey := tfm.DotNetFrameworkName(provider)

    var deps []string
    for _, pkg := range packages {
        pkgVer, err := version.Parse(pkg.Version)
        if err != nil {
            continue
        }
        deps = append(deps, fmt.Sprintf("%s >= %s", pkg.ID, pkgVer.ToNormalizedString()))
    }

    sort.Strings(deps)
    groups[targetKey] = deps
    return groups
}

// buildFrameworks builds the frameworks section
func (b *LockFileBuilder) buildFrameworks(tfm *frameworks.NuGetFramework) map[string]any {
    frameworks := make(map[string]any)
    provider := frameworks.DefaultFrameworkNameProvider()
    shortName := tfm.GetShortFolderName(provider)

    frameworks[shortName] = map[string]any{
        "targetAlias":       shortName,
        "projectReferences": map[string]any{},
    }
    return frameworks
}
```

**Reference**: Port from `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.Commands/RestoreCommand/LockFileBuilder.cs`

---

#### restore/lock_file_format.go

LockFile types and JSON serialization, ported from NuGet.ProjectModel.

```go
// restore/lock_file_format.go

package restore

import (
    "encoding/json"
    "os"
)

// LockFile represents project.assets.json structure (V3 format)
// Ported from NuGet.ProjectModel/LockFile.cs
type LockFile struct {
    Version                     int                    `json:"version"`
    Targets                     map[string]any         `json:"targets"`
    Libraries                   map[string]any         `json:"libraries"`
    ProjectFileDependencyGroups map[string][]string    `json:"projectFileDependencyGroups"`
    PackageFolders              map[string]any         `json:"packageFolders"`
    Project                     *LockFileProject       `json:"project"`
}

// LockFileProject represents the project section
type LockFileProject struct {
    Version string            `json:"version"`
    Restore *LockFileRestore  `json:"restore"`
}

// LockFileRestore represents the restore metadata
type LockFileRestore struct {
    ProjectUniqueName string         `json:"projectUniqueName"`
    ProjectName       string         `json:"projectName"`
    ProjectPath       string         `json:"projectPath"`
    PackagesPath      string         `json:"packagesPath"`
    OutputPath        string         `json:"outputPath"`
    ProjectStyle      string         `json:"projectStyle"`
    Frameworks        map[string]any `json:"frameworks"`
}

// Save writes the lock file to disk
func (lf *LockFile) Save(path string) error {
    data, err := json.MarshalIndent(lf, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}
```

**Reference**: Port from `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.ProjectModel/LockFile*.cs`

---

### Refactor Existing CLI Code

#### cmd/gonuget/commands/add_package.go

Move `resolveLatestVersion()` logic to the library.

**Before (WRONG - logic in CLI)**:
```go
func resolveLatestVersion(ctx context.Context, packageID string, opts *AddPackageOptions) (string, error) {
    // 70 lines of NuGet logic in CLI...
}
```

**After (CORRECT - call library)**:
```go
func resolveLatestVersion(ctx context.Context, packageID string, opts *AddPackageOptions) (string, error) {
    return restore.ResolveLatestVersion(ctx, &restore.ResolveOptions{
        PackageID:  packageID,
        Source:     opts.Source,
        Prerelease: opts.Prerelease,
    })
}
```

Then add to `restore/package_resolver.go`:

```go
// restore/package_resolver.go

package restore

import (
    "context"
    "fmt"
    "time"

    "github.com/willibrandon/gonuget/core"
    "github.com/willibrandon/gonuget/version"
)

// ResolveOptions holds version resolution options
type ResolveOptions struct {
    PackageID  string
    Source     string
    Prerelease bool
}

// ResolveLatestVersion resolves the latest version of a package
func ResolveLatestVersion(ctx context.Context, opts *ResolveOptions) (string, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    source := opts.Source
    if source == "" {
        source = "https://api.nuget.org/v3/index.json"
    }

    repoManager := core.NewRepositoryManager()
    repo := core.NewSourceRepository(core.RepositoryConfig{
        SourceURL: source,
    })
    repoManager.AddRepository(repo)

    versions, err := repo.ListVersions(ctx, nil, opts.PackageID)
    if err != nil {
        return "", err
    }

    if len(versions) == 0 {
        return "", fmt.Errorf("package '%s' not found", opts.PackageID)
    }

    var latestStable, latestPrerelease *version.NuGetVersion
    for _, v := range versions {
        parsed, err := version.Parse(v)
        if err != nil {
            continue
        }

        if parsed.IsPrerelease() {
            if latestPrerelease == nil || parsed.Compare(latestPrerelease) > 0 {
                latestPrerelease = parsed
            }
        } else {
            if latestStable == nil || parsed.Compare(latestStable) > 0 {
                latestStable = parsed
            }
        }
    }

    if opts.Prerelease {
        if latestPrerelease != nil {
            return latestPrerelease.String(), nil
        }
        if latestStable != nil {
            return latestStable.String(), nil
        }
    } else {
        if latestStable != nil {
            return latestStable.String(), nil
        }
        return "", fmt.Errorf("no stable version found for '%s'", opts.PackageID)
    }

    return "", fmt.Errorf("no versions found")
}
```

---

## Testing Strategy

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
