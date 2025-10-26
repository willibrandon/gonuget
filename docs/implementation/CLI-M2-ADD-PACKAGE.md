# CLI M2: Add Package Command

**Status:** Phase 2 - Package Management
**Command:** `gonuget add package <PACKAGE_ID>`
**Parity Target:** `dotnet add package`

## Overview

Implements the `add package` command that adds a NuGet package reference to an MSBuild project file. This command modifies the `.csproj` file by adding or updating a `<PackageReference>` element and optionally triggers restore.

**Critical:** This follows the modern dotnet approach (PackageReference in .csproj), NOT the legacy nuget.exe approach (packages.config + packages/ folder).

## Command Specification

### Syntax

```bash
gonuget add [<PROJECT>] package <PACKAGE_ID> [options]
```

### Arguments

- `<PROJECT>` - Optional path to project file. If not specified, searches current directory for a single .csproj file.
- `<PACKAGE_ID>` - Package identifier (e.g., `Newtonsoft.Json`)

### Options

- `--version <VERSION>` - Specific version to add (e.g., `13.0.3`, `13.*`, `[13.0.0,14.0.0)`)
- `--framework <FRAMEWORK>` - Add package only for specific target framework(s)
- `--no-restore` - Don't restore packages after adding reference
- `--source <SOURCE>` - Package source to use for version resolution
- `--package-directory <PATH>` - Custom package directory (overrides global packages folder)
- `--prerelease` - Include prerelease versions when resolving latest
- `--interactive` - Allow user interaction for authentication
- `-h|--help` - Show help

### Examples

```bash
# Add latest stable version
gonuget add package Newtonsoft.Json

# Add specific version
gonuget add package Newtonsoft.Json --version 13.0.3

# Add to specific project
gonuget add MyApp.csproj package Newtonsoft.Json

# Add framework-specific reference
gonuget add package System.Drawing.Common --framework net8.0

# Add without restore
gonuget add package Newtonsoft.Json --no-restore

# Add latest prerelease
gonuget add package Newtonsoft.Json --prerelease

# Add from specific source
gonuget add package Newtonsoft.Json --source https://api.nuget.org/v3/index.json
```

## Implementation Phases

### Phase 1: Basic Add (M2.1)

**Scope:**
- Add unconditional `<PackageReference>` to .csproj
- Version resolution (latest stable or specified)
- Automatic restore after add
- Single target framework projects only
- CPM detection with error (full CPM support in M2.2 Chunks 11-13)

**Chunks:**
1. Project file detection and loading
2. XML manipulation for PackageReference
3. Version resolution from package sources
4. Project file saving with formatting preservation

### Phase 2: Advanced Features (M2.2)

**Scope:**
- **Central Package Management (CPM)** - Full support (Chunks 11-13)
  - Directory.Packages.props detection and manipulation
  - PackageVersion management
  - VersionOverride support
- Framework-specific references (conditional ItemGroups) (Chunk 14)
- Transitive dependency resolution integration (Chunk 15)
- Multi-TFM project support (Chunk 16)
- Solution file support (Chunk 17)
- Package compatibility verification
- `--no-restore` flag support (already in M2.1)

## Workflow

### High-Level Flow

```
1. Parse command arguments
2. Locate project file
3. Load and parse project XML
4. Resolve package version
5. Check if package already exists
6. Add or update PackageReference
7. Save project file
8. Restore packages (unless --no-restore)
9. Report result
```

### Detailed Flow (dotnet parity)

```
┌─────────────────────────────────────┐
│ Parse arguments                     │
│ - Project path (optional)           │
│ - Package ID (required)             │
│ - Version, framework, flags         │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Locate project file                 │
│ - Use provided path OR              │
│ - Search current dir for .csproj    │
│ - Error if 0 or >1 project found    │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Load project file                   │
│ - Parse XML with encoding/xml       │
│ - Detect SDK-style vs legacy        │
│ - Extract target frameworks         │
│ - Detect CPM (error if enabled)     │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Resolve package version             │
│ - If --version specified: use it    │
│ - Else: query sources for latest    │
│ - Apply --prerelease filter         │
│ - Error if no version found         │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Check existing reference            │
│ - Search ItemGroups for package     │
│ - If found: prepare update          │
│ - If not found: prepare add         │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Modify project XML                  │
│ - Find/create appropriate ItemGroup │
│ - Add PackageReference element      │
│ - Set Include, Version attributes   │
│ - Preserve formatting and comments  │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Save project file                   │
│ - Marshal XML with indentation      │
│ - Write UTF-8 BOM                   │
│ - Write XML declaration             │
│ - Preserve original formatting      │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Restore packages (if enabled)       │
│ - Run gonuget restore               │
│ - Download package to global cache  │
│ - Generate assets file              │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│ Report result                       │
│ - Success: Package added/updated    │
│ - Error: Specific failure reason    │
└─────────────────────────────────────┘
```

## Project File Manipulation

### XML Structure

**Input .csproj:**
```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>
```

**Output .csproj (after add):**
```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>
```

### Go Implementation

```go
// cmd/gonuget/commands/add_package.go

package commands

import (
    "context"
    "encoding/xml"
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/willibrandon/gonuget/cmd/gonuget/output"
    "github.com/willibrandon/gonuget/cmd/gonuget/project"
    "github.com/willibrandon/gonuget/core"
    "github.com/willibrandon/gonuget/version"
)

type addPackageOptions struct {
    projectPath      string
    version          string
    framework        []string
    noRestore        bool
    source           []string
    packageDirectory string
    prerelease       bool
    interactive      bool
}

func NewAddPackageCommand(console *output.Console) *cobra.Command {
    opts := &addPackageOptions{}

    cmd := &cobra.Command{
        Use:   "package <PACKAGE_ID>",
        Short: "Add a NuGet package reference to the project",
        Long: `Adds a NuGet package reference to an MSBuild project file.

This command modifies the .csproj file by adding a PackageReference element.
After adding the reference, packages are automatically restored unless --no-restore is specified.

Examples:
  gonuget add package Newtonsoft.Json
  gonuget add package Newtonsoft.Json --version 13.0.3
  gonuget add MyApp.csproj package System.Text.Json --framework net8.0
  gonuget add package Newtonsoft.Json --no-restore`,
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return runAddPackage(cmd.Context(), console, args[0], opts)
        },
    }

    cmd.Flags().StringVar(&opts.version, "version", "", "Package version to add")
    cmd.Flags().StringSliceVar(&opts.framework, "framework", nil, "Target framework(s)")
    cmd.Flags().BoolVar(&opts.noRestore, "no-restore", false, "Don't restore after adding")
    cmd.Flags().StringSliceVar(&opts.source, "source", nil, "Package source URL")
    cmd.Flags().StringVar(&opts.packageDirectory, "package-directory", "", "Custom package directory")
    cmd.Flags().BoolVar(&opts.prerelease, "prerelease", false, "Include prerelease versions")
    cmd.Flags().BoolVar(&opts.interactive, "interactive", false, "Allow user interaction")

    return cmd
}

func runAddPackage(ctx context.Context, console *output.Console, packageID string, opts *addPackageOptions) error {
    // 1. Locate project file
    projectPath := opts.projectPath
    if projectPath == "" {
        var err error
        projectPath, err = project.FindProjectFile(".")
        if err != nil {
            return fmt.Errorf("failed to locate project file: %w", err)
        }
    }

    console.Printf("info : Adding package '%s' to project '%s'\n", packageID, projectPath)

    // 2. Load project
    proj, err := project.LoadProject(projectPath)
    if err != nil {
        return fmt.Errorf("failed to load project: %w", err)
    }

    // 3. Check for CPM (M2.1: error, M2.2: full support - see Chunks 11-13)
    if proj.IsCentralPackageManagementEnabled() {
        return fmt.Errorf("Central Package Management detected. CPM support is in M2.2 (Chunks 11-13). Use 'dotnet add package' for now.")
    }

    // 4. Resolve version
    packageVersion := opts.version
    if packageVersion == "" {
        console.Printf("info : Resolving latest version for '%s'\n", packageID)

        client := createClientFromOptions(opts)
        latestVersion, err := resolveLatestVersion(ctx, client, packageID, opts.prerelease)
        if err != nil {
            return fmt.Errorf("failed to resolve version: %w", err)
        }

        packageVersion = latestVersion.String()
        console.Printf("info : Resolved version: %s\n", packageVersion)
    }

    // 5. Add or update package reference
    updated, err := proj.AddOrUpdatePackageReference(packageID, packageVersion, opts.framework)
    if err != nil {
        return fmt.Errorf("failed to add package reference: %w", err)
    }

    // 6. Save project
    if err := proj.Save(); err != nil {
        return fmt.Errorf("failed to save project: %w", err)
    }

    if updated {
        console.Printf("info : Updated package '%s' to version '%s'\n", packageID, packageVersion)
    } else {
        console.Printf("info : Added package '%s' version '%s'\n", packageID, packageVersion)
    }

    // 7. Restore (unless --no-restore)
    if !opts.noRestore {
        console.Println("info : Restoring packages...")

        if err := runRestore(ctx, console, projectPath); err != nil {
            console.Printf("warn : Restore failed: %v\n", err)
            console.Println("warn : Run 'gonuget restore' manually to complete installation")
        } else {
            console.Println("info : Package added successfully")
        }
    } else {
        console.Println("info : Package reference added. Run 'gonuget restore' to download packages.")
    }

    return nil
}

func resolveLatestVersion(ctx context.Context, client *core.Client, packageID string, includePrerelease bool) (*version.NuGetVersion, error) {
    // Query package sources for latest version
    versions, err := client.ListVersions(ctx, packageID)
    if err != nil {
        return nil, err
    }

    if len(versions) == 0 {
        return nil, fmt.Errorf("no versions found for package '%s'", packageID)
    }

    // Filter to stable only if prerelease not requested
    var candidates []*version.NuGetVersion
    for _, v := range versions {
        if includePrerelease || !v.IsPrerelease {
            candidates = append(candidates, v)
        }
    }

    if len(candidates) == 0 {
        return nil, fmt.Errorf("no stable versions found for package '%s'. Use --prerelease to include prerelease versions", packageID)
    }

    // Return latest (versions already sorted descending)
    return candidates[0], nil
}
```

### Project File Abstraction

```go
// cmd/gonuget/project/project.go

package project

import (
    "encoding/xml"
    "fmt"
    "os"
    "path/filepath"
)

// Project represents an MSBuild project file
type Project struct {
    Path     string
    Root     *ProjectRootElement
    modified bool
}

// ProjectRootElement is the root <Project> element
type ProjectRootElement struct {
    XMLName    xml.Name             `xml:"Project"`
    Sdk        string               `xml:"Sdk,attr,omitempty"`
    Properties []*PropertyGroup     `xml:"PropertyGroup"`
    ItemGroups []*ItemGroup         `xml:"ItemGroup"`
    RawContent []byte               `xml:",innerxml"` // Preserve unknown content
}

// ItemGroup represents <ItemGroup>
type ItemGroup struct {
    XMLName           xml.Name            `xml:"ItemGroup"`
    Condition         string              `xml:"Condition,attr,omitempty"`
    PackageReferences []*PackageReference `xml:"PackageReference"`
    RawContent        []byte              `xml:",innerxml"`
}

// PackageReference represents <PackageReference>
type PackageReference struct {
    XMLName       xml.Name `xml:"PackageReference"`
    Include       string   `xml:"Include,attr"`
    Version       string   `xml:"Version,attr,omitempty"`
    IncludeAssets string   `xml:"IncludeAssets,omitempty"`
    PrivateAssets string   `xml:"PrivateAssets,omitempty"`
}

// PropertyGroup represents <PropertyGroup>
type PropertyGroup struct {
    XMLName    xml.Name `xml:"PropertyGroup"`
    RawContent []byte   `xml:",innerxml"`
}

// LoadProject loads an MSBuild project file
func LoadProject(path string) (*Project, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read project file: %w", err)
    }

    var root ProjectRootElement
    if err := xml.Unmarshal(data, &root); err != nil {
        return nil, fmt.Errorf("failed to parse project XML: %w", err)
    }

    return &Project{
        Path: path,
        Root: &root,
    }, nil
}

// AddOrUpdatePackageReference adds or updates a package reference
func (p *Project) AddOrUpdatePackageReference(packageID, version string, frameworks []string) (updated bool, err error) {
    // Find existing reference
    for _, ig := range p.Root.ItemGroups {
        for _, pr := range ig.PackageReferences {
            if pr.Include == packageID {
                // Update version
                pr.Version = version
                p.modified = true
                return true, nil
            }
        }
    }

    // Not found, add new reference
    itemGroup := p.findOrCreateItemGroup(frameworks)
    itemGroup.PackageReferences = append(itemGroup.PackageReferences, &PackageReference{
        Include: packageID,
        Version: version,
    })

    p.modified = true
    return false, nil
}

// findOrCreateItemGroup finds or creates an ItemGroup with the given condition
func (p *Project) findOrCreateItemGroup(frameworks []string) *ItemGroup {
    condition := ""
    if len(frameworks) > 0 {
        // M2.1: Error on framework-specific references
        // M2.2: Support conditional ItemGroups
        panic("framework-specific references not yet supported")
    }

    // Find existing unconditional ItemGroup
    for _, ig := range p.Root.ItemGroups {
        if ig.Condition == condition {
            return ig
        }
    }

    // Create new ItemGroup
    itemGroup := &ItemGroup{
        Condition:         condition,
        PackageReferences: []*PackageReference{},
    }
    p.Root.ItemGroups = append(p.Root.ItemGroups, itemGroup)
    return itemGroup
}

// Save saves the project file
func (p *Project) Save() error {
    if !p.modified {
        return nil
    }

    // Marshal with indentation
    output, err := xml.MarshalIndent(p.Root, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal project: %w", err)
    }

    // Open file for writing
    file, err := os.Create(p.Path)
    if err != nil {
        return fmt.Errorf("failed to create file: %w", err)
    }
    defer file.Close()

    // Write UTF-8 BOM (required for .NET compatibility)
    if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
        return err
    }

    // Write XML declaration
    if _, err := file.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"); err != nil {
        return err
    }

    // Write project XML
    if _, err := file.Write(output); err != nil {
        return err
    }

    return nil
}

// IsCentralPackageManagementEnabled checks if CPM is enabled
func (p *Project) IsCentralPackageManagementEnabled() bool {
    // M2.1: Simple detection - check for Directory.Packages.props
    dir := filepath.Dir(p.Path)
    cpmPath := filepath.Join(dir, "Directory.Packages.props")
    _, err := os.Stat(cpmPath)
    return err == nil
}

// FindProjectFile finds a single .csproj file in the directory
func FindProjectFile(dir string) (string, error) {
    matches, err := filepath.Glob(filepath.Join(dir, "*.csproj"))
    if err != nil {
        return "", err
    }

    if len(matches) == 0 {
        return "", fmt.Errorf("no project file found in directory: %s", dir)
    }

    if len(matches) > 1 {
        return "", fmt.Errorf("multiple project files found in directory: %s. Specify which project to use.", dir)
    }

    return matches[0], nil
}
```

## Testing Strategy

### Unit Tests

```go
// cmd/gonuget/project/project_test.go

func TestAddPackageReference_New(t *testing.T) {
    // Create minimal project
    proj := &Project{
        Path: "test.csproj",
        Root: &ProjectRootElement{
            Sdk: "Microsoft.NET.Sdk",
            ItemGroups: []*ItemGroup{},
        },
    }

    // Add package reference
    updated, err := proj.AddOrUpdatePackageReference("Newtonsoft.Json", "13.0.3", nil)
    require.NoError(t, err)
    assert.False(t, updated) // Not an update, it's new

    // Verify reference was added
    require.Len(t, proj.Root.ItemGroups, 1)
    require.Len(t, proj.Root.ItemGroups[0].PackageReferences, 1)

    ref := proj.Root.ItemGroups[0].PackageReferences[0]
    assert.Equal(t, "Newtonsoft.Json", ref.Include)
    assert.Equal(t, "13.0.3", ref.Version)
}

func TestAddPackageReference_Update(t *testing.T) {
    // Create project with existing reference
    proj := &Project{
        Path: "test.csproj",
        Root: &ProjectRootElement{
            Sdk: "Microsoft.NET.Sdk",
            ItemGroups: []*ItemGroup{
                {
                    PackageReferences: []*PackageReference{
                        {Include: "Newtonsoft.Json", Version: "12.0.0"},
                    },
                },
            },
        },
    }

    // Update version
    updated, err := proj.AddOrUpdatePackageReference("Newtonsoft.Json", "13.0.3", nil)
    require.NoError(t, err)
    assert.True(t, updated) // This is an update

    // Verify version was updated
    ref := proj.Root.ItemGroups[0].PackageReferences[0]
    assert.Equal(t, "13.0.3", ref.Version)
}
```

### CLI Interop Tests

```csharp
// tests/cli-interop/GonugetCliInterop.Tests/AddPackageTests.cs

[Fact]
public async Task AddPackage_Simple_MatchesDotnet()
{
    // Create test project
    var projectPath = CreateTestProject("net8.0");

    // Run dotnet add package
    await RunDotnetCommand($"add {projectPath} package Newtonsoft.Json --version 13.0.3 --no-restore");
    var dotnetProjectXml = await File.ReadAllTextAsync(projectPath);

    // Reset project
    ResetTestProject(projectPath);

    // Run gonuget add package
    var result = await ExecuteGonugetCommand(new AddPackageRequest
    {
        ProjectPath = projectPath,
        PackageId = "Newtonsoft.Json",
        Version = "13.0.3",
        NoRestore = true
    });
    var gonugetProjectXml = await File.ReadAllTextAsync(projectPath);

    // Compare XML (normalize whitespace)
    AssertXmlEqual(dotnetProjectXml, gonugetProjectXml);
}

[Fact]
public async Task AddPackage_LatestVersion_MatchesDotnet()
{
    var projectPath = CreateTestProject("net8.0");

    // Both should resolve to same latest version
    await RunDotnetCommand($"add {projectPath} package Newtonsoft.Json --no-restore");
    var dotnetVersion = ExtractPackageVersion(projectPath, "Newtonsoft.Json");

    ResetTestProject(projectPath);

    var result = await ExecuteGonugetCommand(new AddPackageRequest
    {
        ProjectPath = projectPath,
        PackageId = "Newtonsoft.Json",
        NoRestore = true
    });
    var gonugetVersion = ExtractPackageVersion(projectPath, "Newtonsoft.Json");

    Assert.Equal(dotnetVersion, gonugetVersion);
}
```

## Error Handling

### Error Scenarios

1. **No project file found:**
```
error: No project file found in current directory.
Specify a project file path or run from a directory containing a .csproj file.
```

2. **Multiple project files:**
```
error: Multiple project files found in directory.
Specify which project to use:
  gonuget add MyApp.csproj package Newtonsoft.Json
```

3. **Package not found:**
```
error: Package 'InvalidPackageName' not found in configured sources.
Check the package ID and try again.
```

4. **No stable versions:**
```
error: No stable versions found for package 'Newtonsoft.Json.Bson'.
Use --prerelease to include prerelease versions.
```

5. **CPM detected in M2.1 (full support in M2.2 Chunks 11-13):**
```
error: Central Package Management detected (Directory.Packages.props exists).
CPM support is implemented in M2.2 (Chunks 11-13). Use 'dotnet add package' for now.
```

6. **Invalid project file:**
```
error: Failed to parse project file: /path/to/project.csproj
The file does not appear to be a valid MSBuild project file.
```

## Output Compatibility

Match dotnet CLI output format exactly:

**dotnet add package output:**
```
  Determining projects to restore...
  Writing /tmp/tmpXXXXXX.csproj.dgspec.json
  Restored /path/to/MyProject.csproj (in 234 ms).
info : Added package 'Newtonsoft.Json' version '13.0.3' to project '/path/to/MyProject.csproj'.
```

**gonuget add package output:**
```
info : Adding package 'Newtonsoft.Json' to project '/path/to/MyProject.csproj'
info : Resolved version: 13.0.3
info : Added package 'Newtonsoft.Json' version '13.0.3'
info : Restoring packages...
info : Package added successfully
```

## Integration with Existing Commands

### gonuget restore

The `add package` command calls `gonuget restore` internally (unless `--no-restore`). Restore implementation:

1. Read project file
2. Extract all PackageReference elements
3. Download packages to global cache
4. Generate project.assets.json

See `CLI-M2-RESTORE.md` for restore implementation details.

## Performance Considerations

### Optimization Targets

- Project file parsing: <10ms
- Version resolution: <500ms (network dependent)
- Project file saving: <10ms
- Total add operation (no restore): <1s
- With restore: Depends on package size and network

### Caching

Reuse existing caching infrastructure:
- Package metadata cache (version lists)
- HTTP response cache
- Downloaded package cache

## M2.2 - Advanced Features

### Chunk 11: Central Package Management (CPM) - Detection and Error Handling

**Objective**: Detect CPM-enabled projects and handle them appropriately.

**Reference**: `../sdk/src/Cli/dotnet/Commands/Package/Add/PackageAddCommand.cs:SetCentralVersion()`

**Implementation**:
```go
// cmd/gonuget/project/project.go

// IsCentralPackageManagementEnabled checks if CPM is enabled
func (p *Project) IsCentralPackageManagementEnabled() bool {
    // Check ManagePackageVersionsCentrally property
    for _, pg := range p.Root.Properties {
        if pg.ManagePackageVersionsCentrally == "true" {
            return true
        }
    }
    return false
}

// GetDirectoryPackagesPropsPath returns path to Directory.Packages.props
func (p *Project) GetDirectoryPackagesPropsPath() string {
    dir := filepath.Dir(p.Path)

    // Check DirectoryPackagesPropsPath property
    for _, pg := range p.Root.Properties {
        if pg.DirectoryPackagesPropsPath != "" {
            return pg.DirectoryPackagesPropsPath
        }
    }

    // Default location
    return filepath.Join(dir, "Directory.Packages.props")
}
```

**PropertyGroup Extension**:
```go
type PropertyGroup struct {
    XMLName                           xml.Name `xml:"PropertyGroup"`
    TargetFramework                   string   `xml:"TargetFramework,omitempty"`
    TargetFrameworks                  string   `xml:"TargetFrameworks,omitempty"`
    ManagePackageVersionsCentrally    string   `xml:"ManagePackageVersionsCentrally,omitempty"`
    DirectoryPackagesPropsPath        string   `xml:"DirectoryPackagesPropsPath,omitempty"`
    OutputType                        string   `xml:"OutputType,omitempty"`
    RootNamespace                     string   `xml:"RootNamespace,omitempty"`
}
```

**Time Estimate**: 2 hours

---

### Chunk 12: Central Package Management - Directory.Packages.props Manipulation

**Objective**: Load, parse, and modify Directory.Packages.props files.

**Reference**: `../sdk/src/Cli/dotnet/Commands/Package/Add/PackageAddCommand.cs`

**Implementation**:
```go
// cmd/gonuget/project/directory_packages.go

// DirectoryPackagesProps represents a Directory.Packages.props file
type DirectoryPackagesProps struct {
    Path     string
    Root     *DirectoryPackagesRootElement
    modified bool
}

// DirectoryPackagesRootElement is the root <Project> element
type DirectoryPackagesRootElement struct {
    XMLName    xml.Name              `xml:"Project"`
    Properties []*PropertyGroup      `xml:"PropertyGroup"`
    ItemGroups []*PackageVersionGroup `xml:"ItemGroup"`
}

// PackageVersionGroup represents an ItemGroup containing PackageVersion elements
type PackageVersionGroup struct {
    XMLName         xml.Name          `xml:"ItemGroup"`
    PackageVersions []*PackageVersion `xml:"PackageVersion"`
}

// PackageVersion represents a <PackageVersion> element in Directory.Packages.props
type PackageVersion struct {
    XMLName xml.Name `xml:"PackageVersion"`
    Include string   `xml:"Include,attr"`
    Version string   `xml:"Version,attr"`
}

// LoadDirectoryPackagesProps loads a Directory.Packages.props file
func LoadDirectoryPackagesProps(path string) (*DirectoryPackagesProps, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read Directory.Packages.props: %w", err)
    }

    var root DirectoryPackagesRootElement
    if err := xml.Unmarshal(data, &root); err != nil {
        return nil, fmt.Errorf("failed to parse Directory.Packages.props: %w", err)
    }

    return &DirectoryPackagesProps{
        Path: path,
        Root: &root,
    }, nil
}

// AddOrUpdatePackageVersion adds or updates a package version
func (dp *DirectoryPackagesProps) AddOrUpdatePackageVersion(packageID, version string) (updated bool, err error) {
    // Find existing PackageVersion
    for _, ig := range dp.Root.ItemGroups {
        for _, pv := range ig.PackageVersions {
            if strings.EqualFold(pv.Include, packageID) {
                pv.Version = version
                dp.modified = true
                return true, nil
            }
        }
    }

    // Not found, add new PackageVersion
    itemGroup := dp.findOrCreateItemGroup()
    itemGroup.PackageVersions = append(itemGroup.PackageVersions, &PackageVersion{
        Include: packageID,
        Version: version,
    })

    dp.modified = true
    return false, nil
}

// Save saves the Directory.Packages.props file
func (dp *DirectoryPackagesProps) Save() error {
    if !dp.modified {
        return nil
    }

    // Marshal XML with indentation
    output, err := xml.MarshalIndent(dp.Root, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal Directory.Packages.props: %w", err)
    }

    // Open file for writing
    file, err := os.Create(dp.Path)
    if err != nil {
        return fmt.Errorf("failed to create file: %w", err)
    }
    defer file.Close()

    // Write UTF-8 BOM
    if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
        return err
    }

    // Write XML declaration
    if _, err := file.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"); err != nil {
        return err
    }

    // Write XML
    if _, err := file.Write(output); err != nil {
        return err
    }

    return nil
}
```

**Time Estimate**: 3 hours

---

### Chunk 13: Central Package Management - PackageVersion Management

**Objective**: Integrate CPM into add package command.

**Reference**: `../sdk/src/Cli/dotnet/Commands/Package/Add/PackageAddCommand.cs:Execute()`

**Implementation**:
```go
// cmd/gonuget/commands/add_package.go (modify runAddPackage)

func runAddPackage(ctx context.Context, console *output.Console, packageID string, opts *addPackageOptions) error {
    // ... (existing project loading code)

    // Check for CPM
    if proj.IsCentralPackageManagementEnabled() {
        console.Printf("info : Central Package Management detected\n")
        return addPackageWithCPM(ctx, console, proj, packageID, opts)
    }

    // ... (existing non-CPM code)
}

func addPackageWithCPM(ctx context.Context, console *output.Console, proj *project.Project, packageID string, opts *addPackageOptions) error {
    // 1. Load Directory.Packages.props
    propsPath := proj.GetDirectoryPackagesPropsPath()
    if _, err := os.Stat(propsPath); os.IsNotExist(err) {
        return fmt.Errorf("Directory.Packages.props not found at %s. CPM requires this file.", propsPath)
    }

    props, err := project.LoadDirectoryPackagesProps(propsPath)
    if err != nil {
        return fmt.Errorf("failed to load Directory.Packages.props: %w", err)
    }

    // 2. Resolve version
    packageVersion := opts.version
    if packageVersion == "" {
        console.Printf("info : Resolving latest version for '%s'\n", packageID)
        client := createClientFromOptions(opts)
        latestVersion, err := resolveLatestVersion(ctx, client, packageID, opts.prerelease)
        if err != nil {
            return fmt.Errorf("failed to resolve version: %w", err)
        }
        packageVersion = latestVersion.String()
        console.Printf("info : Resolved version: %s\n", packageVersion)
    }

    // 3. Add/update PackageVersion in Directory.Packages.props
    updated, err := props.AddOrUpdatePackageVersion(packageID, packageVersion)
    if err != nil {
        return fmt.Errorf("failed to add package version: %w", err)
    }

    if err := props.Save(); err != nil {
        return fmt.Errorf("failed to save Directory.Packages.props: %w", err)
    }

    // 4. Add PackageReference WITHOUT version to .csproj
    _, err = proj.AddOrUpdatePackageReference(packageID, "", opts.framework)
    if err != nil {
        return fmt.Errorf("failed to add package reference: %w", err)
    }

    if err := proj.Save(); err != nil {
        return fmt.Errorf("failed to save project: %w", err)
    }

    if updated {
        console.Printf("info : Updated package '%s' to version '%s' in Directory.Packages.props\n", packageID, packageVersion)
    } else {
        console.Printf("info : Added package '%s' version '%s' to Directory.Packages.props\n", packageID, packageVersion)
    }

    // 5. Restore (unless --no-restore)
    if !opts.noRestore {
        console.Println("info : Restoring packages...")
        if err := runRestore(ctx, console, proj.Path); err != nil {
            console.Printf("warn : Restore failed: %v\n", err)
        } else {
            console.Println("info : Package added successfully")
        }
    }

    return nil
}
```

**Time Estimate**: 4 hours

---

### Chunk 14: Framework-Specific References

**Objective**: Support conditional ItemGroups based on target framework.

**Implementation**: Parse `<TargetFrameworks>` (plural), create conditional ItemGroups with `Condition` attribute, verify package compatibility per TFM.

**Time Estimate**: 3 hours

---

### Chunk 15: Transitive Dependency Resolution

**Objective**: Integrate existing `core/resolver` for transitive deps during add package.

**Time Estimate**: 2 hours

---

### Chunk 16: Multi-TFM Project Support

**Objective**: Handle projects with multiple target frameworks.

**Time Estimate**: 2 hours

---

### Chunk 17: Solution File Support

**Objective**: Support adding packages to solution-level projects.

**Time Estimate**: 4 hours

---

### Chunk 18: CLI Interop Tests for CPM

**Objective**: Test CPM functionality against dotnet nuget behavior.

**Time Estimate**: 3 hours

---

### Chunk 19: CLI Interop Tests for Advanced Features

**Objective**: Test all M2.2 features for parity.

**Time Estimate**: 3 hours

## Success Criteria

**M2.1 (Basic Add):**
- [ ] Add package with version specified
- [ ] Add package with latest version resolution
- [ ] Update existing package reference
- [ ] Preserve project file formatting
- [ ] CPM detection with error message
- [ ] CLI interop tests pass (100% XML parity with dotnet)
- [ ] Error handling for all scenarios
- [ ] Help documentation complete
- [ ] Integration tests with real NuGet.org packages

**M2.2 (Advanced Features - 100% dotnet parity):**
- [ ] Central Package Management (CPM) full support
  - [ ] Detect ManagePackageVersionsCentrally property
  - [ ] Load and parse Directory.Packages.props
  - [ ] Add/update PackageVersion entries
  - [ ] Add PackageReference without version to .csproj
  - [ ] VersionOverride support
- [ ] Framework-specific references (conditional ItemGroups)
- [ ] Transitive dependency resolution
- [ ] Multi-TFM project support
- [ ] Solution file support
- [ ] CPM CLI interop tests pass (100% parity with dotnet)
- [ ] All advanced features interop tests pass
