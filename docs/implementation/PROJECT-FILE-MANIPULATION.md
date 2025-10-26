# Project File Manipulation Research

## Overview

This document provides a comprehensive analysis of how dotnet CLI manipulates MSBuild project files (`.csproj`) and Central Package Management files (`Directory.Packages.props`) for package management operations. This research is critical for achieving 100% parity with dotnet in gonuget CLI implementation.

## Key Architecture Decisions

### dotnet vs nuget.exe Philosophy

**dotnet CLI** (modern approach):
- Project-centric workflow
- Packages declared in `.csproj` as `<PackageReference>` elements
- Global package cache (`~/.nuget/packages`)
- No local `packages/` folder
- MSBuild integration for restore
- Commands: `add`, `remove`, `list`, `restore`
- ❌ No `install` command

**nuget.exe** (legacy approach):
- Package-centric workflow
- Packages listed in `packages.config` XML file
- Local `packages/` folder in solution directory
- Standalone package extraction
- Commands: `install`, `update`, `restore`

**gonuget CLI must follow dotnet approach** for modern .NET compatibility.

## Command Delegation Architecture

### dotnet add package

**Call chain:**
```
dotnet add package
  ↓
PackageAddCommand.Execute() [sdk/src/Cli/dotnet/Commands/Package/Add/PackageAddCommand.cs]
  ↓
NuGetCommand.Run() [sdk/src/Cli/dotnet/Commands/NuGet/NuGetCommand.cs]
  ↓
NuGetForwardingApp.Execute() [sdk/src/Cli/dotnet/NuGetForwardingApp.cs]
  ↓
NuGet.CommandLine.XPlat.dll (AddPackageReferenceCommandRunner)
  ↓
MSBuildAPIUtility.AddPackageReference()
  ↓
Project file XML modification via Microsoft.Build.Evaluation API
```

**Key files:**
- Entry point: `PackageAddCommand.cs` - Generates dependency graph, calls NuGet command
- XML manipulation: `MSBuildAPIUtility.cs` - Core project file modification logic
- NuGet implementation: `AddPackageReferenceCommandRunner.cs` - Package resolution and compatibility checks

### dotnet restore

**Call chain:**
```
dotnet restore
  ↓
RestoreCommand.FromParseResult() [sdk/src/Cli/dotnet/Commands/Restore/RestoreCommand.cs]
  ↓
MSBuildForwardingApp.Execute()
  ↓
MSBuild Restore targets
  ↓
NuGet.Build.Tasks.dll
  ↓
NuGet.Commands.RestoreCommand
```

**Implementation:**
- Restore is entirely handled by MSBuild targets
- No custom C# restore logic in dotnet CLI
- MSBuild evaluates project, generates restore graph, downloads packages

## MSBuild API Usage

### Core API Types

dotnet uses the **Microsoft.Build.Evaluation** namespace for project manipulation:

```csharp
using Microsoft.Build.Construction;  // ProjectRootElement, ProjectItemGroupElement
using Microsoft.Build.Evaluation;    // Project, ProjectItem
using Microsoft.Build.Execution;     // ProjectInstance for build operations
```

### Opening a Project

```csharp
// MSBuildAPIUtility.cs:58
internal static Project GetProject(string projectCSProjPath)
{
    var projectRootElement = TryOpenProjectRootElement(projectCSProjPath);
    if (projectCSProjPath == null)
    {
        throw new InvalidOperationException($"Unable to open project: {projectCSProjPath}");
    }
    return new Project(projectRootElement);
}

private static ProjectRootElement TryOpenProjectRootElement(string filename)
{
    return ProjectRootElement.Open(
        filename,
        ProjectCollection.GlobalProjectCollection,
        preserveFormatting: true  // CRITICAL: Preserves formatting
    );
}
```

**Key points:**
- `preserveFormatting: true` maintains whitespace and indentation
- Uses `ProjectCollection.GlobalProjectCollection` for caching
- Must call `project.Save()` to persist changes
- Must call `ProjectCollection.GlobalProjectCollection.UnloadProject(project)` to release

### Adding PackageReference (non-CPM)

```csharp
// MSBuildAPIUtility.cs:441
private void AddPackageReferenceIntoItemGroup(
    ProjectItemGroupElement itemGroup,
    LibraryDependency libraryDependency)
{
    // Add <PackageReference Include="PackageName" />
    var item = itemGroup.AddItem(PACKAGE_REFERENCE_TYPE_TAG, libraryDependency.Name);

    // Add Version="1.0.0" as attribute
    var packageVersion = libraryDependency.LibraryRange.VersionRange.OriginalString ??
        libraryDependency.LibraryRange.VersionRange.MinVersion.ToString();
    item.AddMetadata(VERSION_TAG, packageVersion, expressAsAttribute: true);

    // Add optional metadata (IncludeAssets, PrivateAssets)
    AddExtraMetadataToProjectItemElement(libraryDependency, item);

    Logger.LogInformation($"Added {libraryDependency.Name} {packageVersion} to {itemGroup.ContainingProject.FullPath}");
}
```

**Result in .csproj:**
```xml
<ItemGroup>
  <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
</ItemGroup>
```

### Adding PackageReference (CPM)

For Central Package Management, version goes in `Directory.Packages.props`:

```csharp
// MSBuildAPIUtility.cs:457
internal void AddPackageReferenceIntoItemGroupCPM(
    Project project,
    ProjectItemGroupElement itemGroup,
    LibraryDependency libraryDependency)
{
    // Only add package name, NO version
    ProjectItemElement item = itemGroup.AddItem(PACKAGE_REFERENCE_TYPE_TAG, libraryDependency.Name);
    AddExtraMetadataToProjectItemElement(libraryDependency, item);
}

// MSBuildAPIUtility.cs:425
internal void AddPackageVersionIntoPropsItemGroup(
    ProjectItemGroupElement itemGroup,
    LibraryDependency libraryDependency)
{
    // Add <PackageVersion Include="PackageName" Version="1.0.0" />
    var item = itemGroup.AddItem(PACKAGE_VERSION_TYPE_TAG, libraryDependency.Name);
    var packageVersion = AddVersionMetadata(libraryDependency, item);
}
```

**Result in .csproj:**
```xml
<ItemGroup>
  <PackageReference Include="Newtonsoft.Json" />
</ItemGroup>
```

**Result in Directory.Packages.props:**
```xml
<ItemGroup>
  <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
</ItemGroup>
```

### Finding or Creating ItemGroup

```csharp
// MSBuildAPIUtility.cs:379
static ProjectItemGroupElement GetOrCreateItemGroup(string targetFrameworkAlias, Project project)
{
    var itemGroups = GetItemGroups(project);
    string condition = targetFrameworkAlias is null ? null : GetTargetFrameworkCondition(targetFrameworkAlias);
    var itemGroup = GetItemGroup(itemGroups, PACKAGE_REFERENCE_TYPE_TAG, condition)
                    ?? CreateItemGroup(project, condition);
    return itemGroup;
}

// MSBuildAPIUtility.cs:1050
private static string GetTargetFrameworkCondition(string targetFramework)
{
    return $"'$(TargetFramework)' == '{targetFramework}'";
}
```

**Result for TFM-specific reference:**
```xml
<ItemGroup Condition="'$(TargetFramework)' == 'net8.0'">
  <PackageReference Include="System.Text.Json" Version="8.0.0" />
</ItemGroup>
```

### Updating Existing PackageReference

```csharp
// MSBuildAPIUtility.cs:567
private void UpdatePackageReferenceItems(
    IEnumerable<ProjectItem> packageReferencesItems,
    LibraryDependency libraryDependency)
{
    // Validate no imported items are being updated
    ValidateNoImportedItemsAreUpdated(packageReferencesItems, libraryDependency, UPDATE_OPERATION);

    foreach (var packageReferenceItem in packageReferencesItems)
    {
        var packageVersion = libraryDependency.LibraryRange.VersionRange.OriginalString ??
            libraryDependency.LibraryRange.VersionRange.MinVersion.ToString();

        // Update Version metadata
        packageReferenceItem.SetMetadataValue(VERSION_TAG, packageVersion);
        UpdateExtraMetadataInProjectItem(libraryDependency, packageReferenceItem);
    }
}
```

### Removing PackageReference

```csharp
// MSBuildAPIUtility.cs:148
public int RemovePackageReference(string projectPath, LibraryDependency libraryDependency)
{
    var project = GetProject(projectPath);

    var existingPackageReferences = project.ItemsIgnoringCondition
        .Where(item => item.ItemType.Equals(PACKAGE_REFERENCE_TYPE_TAG, StringComparison.OrdinalIgnoreCase) &&
                       item.EvaluatedInclude.Equals(libraryDependency.Name, StringComparison.OrdinalIgnoreCase));

    if (existingPackageReferences.Any())
    {
        ValidateNoImportedItemsAreUpdated(existingPackageReferences, libraryDependency, REMOVE_OPERATION);
        project.RemoveItems(existingPackageReferences);
        project.Save();
        ProjectCollection.GlobalProjectCollection.UnloadProject(project);
        return 0;
    }
    else
    {
        Logger.LogError($"Package {libraryDependency.Name} not found in {project.FullPath}");
        ProjectCollection.GlobalProjectCollection.UnloadProject(project);
        return 1;
    }
}
```

## Central Package Management (CPM)

### Detection

```csharp
// Check if project uses CPM
var isCentralPackageManagementEnabled =
    project.RestoreMetadata.CentralPackageVersionsEnabled;

// Get Directory.Packages.props path
string directoryPackagesPropsPath =
    project.GetPropertyValue("DirectoryPackagesPropsPath");

// Load Directory.Packages.props
ProjectRootElement directoryBuildPropsRootElement =
    project.Imports.FirstOrDefault(i =>
        i.ImportedProject.FullPath.Equals(directoryPackagesPropsPath, StringComparison.OrdinalIgnoreCase))
    .ImportedProject;
```

### Validation Rules (AreCentralVersionRequirementsSatisfied)

CPM enforces strict validation:

1. **PackageReference must NOT have Version attribute** in `.csproj`
   - ❌ `<PackageReference Include="Foo" Version="1.0.0" />`
   - ✅ `<PackageReference Include="Foo" />`

2. **PackageVersion must be in Directory.Packages.props**, not in `.csproj`
   - ❌ `<PackageVersion>` in project file
   - ✅ `<PackageVersion>` in `Directory.Packages.props`

3. **PackageReference must NOT be in Directory.Packages.props**
   - ❌ `<PackageReference>` in props file
   - ✅ `<PackageReference>` in project file

4. **VersionOverride support** (if not disabled)
   - ✅ `<PackageReference Include="Foo" VersionOverride="2.0.0" />`

5. **Floating versions disabled by default**
   - ❌ `<PackageVersion Include="Foo" Version="1.*" />` (unless enabled)

### VersionOverride

Allows project-level version override in CPM:

```csharp
// MSBuildAPIUtility.cs:597
internal static void UpdateVersionOverride(
    Project project,
    ProjectItem packageReference,
    string versionCLIArgument)
{
    ProjectItemElement packageReferenceItemElement =
        project.GetItemProvenance(packageReference).LastOrDefault()?.ItemElement;

    ProjectMetadataElement versionOverrideAttribute =
        packageReferenceItemElement.Metadata.FirstOrDefault(i => i.Name.Equals("VersionOverride"));

    versionOverrideAttribute.Value = versionCLIArgument;
    packageReferenceItemElement.ContainingProject.Save();
}
```

**Result:**
```xml
<!-- Directory.Packages.props -->
<ItemGroup>
  <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
</ItemGroup>

<!-- MyProject.csproj -->
<ItemGroup>
  <!-- Override to 12.0.3 for this project only -->
  <PackageReference Include="Newtonsoft.Json" VersionOverride="12.0.3" />
</ItemGroup>
```

## Dependency Graph Generation

### MSBuild Target Invocation

```csharp
// PackageAddCommand.cs:175
private static void GetProjectDependencyGraph(string projectFilePath, string dgFilePath)
{
    List<string> args =
    [
        projectFilePath,
        "-target:GenerateRestoreGraphFile",
        $"-property:RestoreGraphOutputPath=\"{dgFilePath}\"",
        $"-property:RestoreRecursive=false",
        $"-property:RestoreDotnetCliToolReferences=false",
        "-nologo",
        "-v:quiet"
    ];

    var result = new MSBuildForwardingApp(args).Execute();
    if (result != 0)
    {
        throw new InvalidOperationException("Failed to generate dependency graph");
    }
}
```

**Output:** Creates `.dg` file (JSON) containing:
- All project references
- All package references
- Target frameworks
- Package sources
- Restore metadata

### DependencyGraphSpec Format

```json
{
  "format": 1,
  "restore": {
    "C:\\MyProject\\MyProject.csproj": {}
  },
  "projects": {
    "C:\\MyProject\\MyProject.csproj": {
      "version": "1.0.0",
      "frameworks": {
        "net8.0": {
          "dependencies": {
            "Newtonsoft.Json": {
              "target": "Package",
              "version": "[13.0.3, )"
            }
          }
        }
      },
      "restore": {
        "projectUniqueName": "C:\\MyProject\\MyProject.csproj",
        "projectName": "MyProject",
        "projectPath": "C:\\MyProject\\MyProject.csproj",
        "projectStyle": "PackageReference",
        "sources": {
          "https://api.nuget.org/v3/index.json": {}
        }
      }
    }
  }
}
```

## Package Compatibility Verification

### Restore Preview Workflow

```csharp
// AddPackageReferenceCommandRunner.cs:193
var restorePreviewResult = await PreviewAddPackageReferenceAsync(
    packageReferenceArgs,
    updatedDgSpec);

// Check which frameworks are compatible
var compatibleFrameworks = new HashSet<NuGetFramework>(
    restorePreviewResult
    .Result
    .CompatibilityCheckResults
    .Where(t => t.Success)
    .Select(t => t.Graph.Framework),
    NuGetFrameworkFullComparer.Instance);
```

**Three outcomes:**

1. **Compatible with ALL frameworks** → Unconditional `<PackageReference>`
```csharp
// AddPackageReferenceCommandRunner.cs:247
msBuild.AddPackageReference(packageReferenceArgs.ProjectPath, libraryDependency, packageReferenceArgs.NoVersion);
```

2. **Compatible with SOME frameworks** → Conditional `<PackageReference>` per TFM
```csharp
// AddPackageReferenceCommandRunner.cs:268
msBuild.AddPackageReferencePerTFM(
    packageReferenceArgs.ProjectPath,
    libraryDependency,
    compatibleOriginalFrameworks,
    packageReferenceArgs.NoVersion);
```

3. **Compatible with NONE** → Error, no modification
```csharp
// AddPackageReferenceCommandRunner.cs:220
packageReferenceArgs.Logger.LogError(
    $"Package {packageReferenceArgs.PackageId} is not compatible with any target frameworks");
return 1;
```

## Version Resolution

### No Version Specified

```csharp
// AddPackageReferenceCommandRunner.cs:148-166
if (packageDependency == null)
{
    var latestVersion = await GetLatestVersionAsync(
        originalPackageSpec,
        packageReferenceArgs.PackageId,
        packageReferenceArgs.Logger,
        packageReferenceArgs.Prerelease);

    if (latestVersion == null)
    {
        if (!packageReferenceArgs.Prerelease)
        {
            latestVersion = await GetLatestVersionAsync(..., !packageReferenceArgs.Prerelease);
            if (latestVersion != null)
            {
                throw new CommandException($"Prerelease versions available: {latestVersion}");
            }
        }
        throw new CommandException($"No versions available for {packageReferenceArgs.PackageId}");
    }

    packageDependency = new PackageDependency(
        packageReferenceArgs.PackageId,
        new VersionRange(minVersion: latestVersion, includeMinVersion: true));
}
```

**Logic:**
1. Try to find latest stable version
2. If none found and `--prerelease` not specified, check for prerelease and error
3. If `--prerelease` specified, include prerelease versions

### CPM with No Version

```csharp
// AddPackageReferenceCommandRunner.cs:136-146
if (originalPackageSpec.RestoreMetadata.CentralPackageVersionsEnabled)
{
    var centralVersion = originalPackageSpec
        .TargetFrameworks
        .Where(tf => tf.CentralPackageVersions.ContainsKey(packageReferenceArgs.PackageId))
        .Select(tf => tf.CentralPackageVersions[packageReferenceArgs.PackageId])
        .FirstOrDefault();
    if (centralVersion != null)
    {
        packageDependency = new PackageDependency(packageReferenceArgs.PackageId, centralVersion.VersionRange);
    }
}
```

If CPM enabled and version exists in `Directory.Packages.props`, use that version.

## Implementation Recommendations for gonuget

### Option 1: Use Microsoft.Build NuGet Packages

**Approach:** Reference Microsoft.Build.* packages in Go via CGO or external process

**Packages needed:**
- `Microsoft.Build` (16.0.0+)
- `Microsoft.Build.Framework`
- `NuGet.ProjectModel`
- `NuGet.Commands`

**Pros:**
- 100% parity with dotnet (same code)
- Handles all edge cases
- CPM support automatic
- Condition evaluation automatic

**Cons:**
- Requires .NET runtime on user machine
- Complex CGO integration or subprocess communication
- Large dependency surface

### Option 2: Pure Go XML Manipulation

**Approach:** Parse and modify `.csproj` XML using Go's `encoding/xml`

**Required capabilities:**
- Parse MSBuild XML with namespaces
- Preserve formatting, whitespace, comments
- Handle conditional ItemGroups (`Condition` attribute)
- Evaluate MSBuild properties for CPM detection
- Generate dependency graph JSON for restore

**Pros:**
- No external dependencies
- Fast, native Go
- Full control

**Cons:**
- Must implement MSBuild property evaluation
- Must implement CPM validation logic
- Risk of missing edge cases
- Must maintain parity manually

### Option 3: Hybrid - External .NET Tool for Complex Operations

**Approach:** Pure Go for simple operations, delegate to .NET tool for complex ones

**Go handles:**
- Simple unconditional `<PackageReference>` add/remove
- XML parsing and formatting
- `gonuget config` commands

**.NET tool handles:**
- CPM operations
- Conditional references
- Dependency graph generation
- Restore preview

**Pros:**
- Best of both worlds
- Parity for complex scenarios
- Simple Go code for common cases

**Cons:**
- Split implementation
- Requires .NET for advanced features
- Coordination complexity

### Recommended: Option 2 (Pure Go) with Constraints

**Phase 1: Basic PackageReference Support**
- No CPM support initially
- No conditional ItemGroups
- Simple add/remove/update operations
- Document CPM limitation clearly

**Phase 2: Full Parity**
- Implement MSBuild property evaluation
- CPM support
- Conditional references
- Full validation

**Rationale:**
- Matches gonuget philosophy (pure Go, no runtime deps)
- Phase 1 covers 80% of use cases
- Can validate against dotnet via interop tests
- Clear upgrade path

## Go Implementation Strategy

### Project File Parser

```go
// ProjectFile represents an MSBuild project
type ProjectFile struct {
    Path     string
    Root     *ProjectRootElement
    modified bool
}

// ProjectRootElement is the root XML element
type ProjectRootElement struct {
    XMLName    xml.Name            `xml:"Project"`
    Sdk        string              `xml:"Sdk,attr,omitempty"`
    ItemGroups []*ProjectItemGroup `xml:"ItemGroup"`
    // Preserve unknown elements
    InnerXML []byte `xml:",innerxml"`
}

// ProjectItemGroup represents <ItemGroup>
type ProjectItemGroup struct {
    Condition         string               `xml:"Condition,attr,omitempty"`
    PackageReferences []*PackageReference  `xml:"PackageReference"`
    PackageVersions   []*PackageVersion    `xml:"PackageVersion"`
    // Preserve unknown items
    Items []interface{} `xml:",any"`
}

// PackageReference represents <PackageReference>
type PackageReference struct {
    Include       string `xml:"Include,attr"`
    Version       string `xml:"Version,attr,omitempty"`
    VersionOverride string `xml:"VersionOverride,attr,omitempty"`
    IncludeAssets string `xml:"IncludeAssets,omitempty"`
    PrivateAssets string `xml:"PrivateAssets,omitempty"`
}

// PackageVersion represents <PackageVersion> in Directory.Packages.props
type PackageVersion struct {
    Include string `xml:"Include,attr"`
    Version string `xml:"Version,attr"`
}
```

### Add Package Reference

```go
func (pf *ProjectFile) AddPackageReference(packageID, version string, frameworks []string) error {
    // Find or create ItemGroup
    itemGroup := pf.FindOrCreateItemGroup(frameworks)

    // Check if package already exists
    existing := itemGroup.FindPackageReference(packageID)
    if existing != nil {
        // Update version
        existing.Version = version
    } else {
        // Add new reference
        itemGroup.PackageReferences = append(itemGroup.PackageReferences, &PackageReference{
            Include: packageID,
            Version: version,
        })
    }

    pf.modified = true
    return nil
}

func (pf *ProjectFile) FindOrCreateItemGroup(frameworks []string) *ProjectItemGroup {
    condition := ""
    if len(frameworks) > 0 {
        condition = fmt.Sprintf("'$(TargetFramework)' == '%s'", frameworks[0])
    }

    // Find existing ItemGroup with matching condition
    for _, ig := range pf.Root.ItemGroups {
        if ig.Condition == condition {
            return ig
        }
    }

    // Create new ItemGroup
    itemGroup := &ProjectItemGroup{
        Condition: condition,
        PackageReferences: []*PackageReference{},
    }
    pf.Root.ItemGroups = append(pf.Root.ItemGroups, itemGroup)
    return itemGroup
}

func (pf *ProjectFile) Save() error {
    if !pf.modified {
        return nil
    }

    // Marshal to XML with indentation
    output, err := xml.MarshalIndent(pf.Root, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal project file: %w", err)
    }

    // Write to file with UTF-8 BOM for .NET compatibility
    file, err := os.Create(pf.Path)
    if err != nil {
        return fmt.Errorf("failed to create file: %w", err)
    }
    defer file.Close()

    // Write UTF-8 BOM
    file.Write([]byte{0xEF, 0xBB, 0xBF})

    // Write XML declaration
    file.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")

    // Write project XML
    file.Write(output)

    return nil
}
```

### Testing Against dotnet CLI

All project manipulation must be validated via CLI interop tests:

```csharp
[Fact]
public void AddPackageReference_Simple_MatchesDotnet()
{
    // Create test project
    var projectPath = CreateTestProject("net8.0");

    // Run dotnet add package
    var dotnetResult = RunDotnetCommand($"add {projectPath} package Newtonsoft.Json --version 13.0.3");
    var dotnetProject = LoadProject(projectPath);

    // Reset project
    ResetProject(projectPath);

    // Run gonuget add package
    var gonugetResult = RunGonugetCommand($"add {projectPath} package Newtonsoft.Json --version 13.0.3");
    var gonugetProject = LoadProject(projectPath);

    // Compare XML (whitespace-insensitive)
    AssertProjectsEqual(dotnetProject, gonugetProject);
}
```

## Key Constants

```csharp
// MSBuildAPIUtility.cs:26-44
private const string PACKAGE_REFERENCE_TYPE_TAG = "PackageReference";
private const string PACKAGE_VERSION_TYPE_TAG = "PackageVersion";
private const string VERSION_TAG = "Version";
private const string FRAMEWORK_TAG = "TargetFramework";
private const string FRAMEWORKS_TAG = "TargetFrameworks";
private const string RESTORE_STYLE_TAG = "RestoreProjectStyle";
private const string NUGET_STYLE_TAG = "NuGetProjectStyle";
private const string ASSETS_FILE_PATH_TAG = "ProjectAssetsFile";
private const string UPDATE_OPERATION = "Update";
private const string REMOVE_OPERATION = "Remove";
private const string IncludeAssets = "IncludeAssets";
private const string PrivateAssets = "PrivateAssets";
private const string DirectoryPackagesPropsPathPropertyName = "DirectoryPackagesPropsPath";
```

## Summary

### Must Implement for M2

1. **Basic PackageReference manipulation:**
   - Add unconditional `<PackageReference>`
   - Update existing `<PackageReference>`
   - Remove `<PackageReference>`

2. **XML preservation:**
   - Preserve formatting
   - Preserve comments
   - Preserve unknown elements
   - UTF-8 BOM for .NET compatibility

3. **Validation:**
   - CLI interop tests comparing gonuget output vs dotnet output
   - Test multi-TFM projects
   - Test existing vs new package references

### Defer to Future Milestones

1. **Central Package Management (CPM):**
   - `Directory.Packages.props` manipulation
   - CPM validation rules
   - VersionOverride support

2. **Conditional references:**
   - TFM-specific `<PackageReference>`
   - Condition evaluation

3. **Restore preview:**
   - Compatibility verification
   - Dependency graph generation

### Critical for Parity

- Use XML libraries that preserve formatting exactly
- Test against dotnet CLI output for every operation
- Document limitations clearly (e.g., "CPM not supported in M2")
- Provide clear error messages when hitting limitations

## References

### Source Files Analyzed

- `dotnet/sdk`: src/Cli/dotnet/Commands/Package/Add/PackageAddCommand.cs
- `dotnet/sdk`: src/Cli/dotnet/Commands/NuGet/NuGetCommand.cs
- `dotnet/sdk`: src/Cli/dotnet/NuGetForwardingApp.cs
- `dotnet/sdk`: src/Cli/dotnet/Commands/Restore/RestoreCommand.cs
- `NuGet/NuGet.Client`: src/NuGet.Core/NuGet.CommandLine.XPlat/Commands/PackageReferenceCommands/AddPackageReferenceCommandRunner.cs
- `NuGet/NuGet.Client`: src/NuGet.Core/NuGet.CommandLine.XPlat/Utility/MSBuildAPIUtility.cs

### External Documentation

- [NuGet PackageReference documentation](https://learn.microsoft.com/en-us/nuget/consume-packages/package-references-in-project-files)
- [Central Package Management](https://learn.microsoft.com/en-us/nuget/consume-packages/central-package-management)
- [MSBuild Project File Schema](https://learn.microsoft.com/en-us/visualstudio/msbuild/msbuild-project-file-schema-reference)
