# NuGet.Client MSBuild Integration Reference

**Purpose**: Reference guide for implementing MSBuild integration in gonuget CLI to achieve 100% parity with nuget.exe

**Last Updated**: 2025-10-25

---

## Overview

NuGet.Client has deep MSBuild integration for the `pack`, `restore`, and `update` commands. This document maps the MSBuild-related code in NuGet.Client to guide the gonuget implementation.

**Critical Finding**: NuGet.Client's MSBuild integration is **fully cross-platform** via:
- **Windows**: Visual Studio Setup API + MSBuild.exe (from VS or Build Tools)
- **Linux/macOS (Modern)**: MSBuild from .NET SDK (5.0+, 6.0, 7.0, 8.0, 9.0)
- **Linux/macOS (Legacy)**: Mono XBuild (older Mono installations)
- **All Platforms**: PATH environment variable + external process invocation

NuGet.Client does **NOT** reimplement the MSBuild evaluation engine. It requires MSBuild or XBuild to be installed and invokes it as an external process. gonuget should follow the EXACT same approach for 100% parity.

**Modern Cross-Platform MSBuild** (Recommended for gonuget):
- .NET SDK includes MSBuild on all platforms (Windows, Linux, macOS)
- Installed with .NET SDK: `dotnet --version` to check
- MSBuild location: `dotnet msbuild` or `$(dotnet --info | grep "Base Path")/MSBuild.dll`
- This is the PRIMARY cross-platform approach since .NET Core 1.0+

---

## Primary MSBuild Parsing Code Locations

### 1. **NuGet.CommandLine** (Main CLI MSBuild Integration)

**Location**: `/NuGet.Client/src/NuGet.Clients/NuGet.CommandLine/`

#### Key Files:

**`Commands/ProjectFactory.cs`** - ⭐ Main MSBuild project parsing and package creation
- **Line 30**: `public class ProjectFactory : IProjectFactory`
- **Line 35**: `private dynamic _project;` (type is `Microsoft.Build.Evaluation.Project`)
- **Responsibilities**:
  - Parse .csproj/.vbproj/.fsproj files via MSBuild API
  - Property substitution (`$id$`, `$version$`, `$author$`, etc.)
  - File collection from MSBuild items (Compile, Content, None, etc.)
  - Referenced project handling (`IncludeReferencedProjects`)
  - Multi-targeting support (TargetFrameworks)
  - Build invocation (when `-Build` flag specified)

**`MsBuildUtility.cs`** - MSBuild discovery and invocation
- **Responsibilities**:
  - Discover MSBuild installations via Visual Studio Setup API (`ISetupInstance`)
  - Locate MSBuild.exe path
  - Invoke MSBuild for building projects
  - Get project references for restore operations
  - Support for MSBuild 14.0 (VS2015), 15.0 (VS2017), 16.0 (VS2019), 17.0+ (VS2022)

**`MsBuildToolset.cs`** - MSBuild version management
- **Responsibilities**:
  - Wraps `ISetupInstance` (Visual Studio Setup Configuration API)
  - Version comparison and selection (picks latest compatible)
  - Installation date tracking
  - Validates installation (checks for "VisualStudio" in name)

**`Common/MSBuildProjectSystem.cs`** - Project file property reading
- **Responsibilities**:
  - Read TargetFramework properties:
    - `TargetFrameworks` (multi-targeting)
    - `TargetFramework` (single target)
    - `TargetFrameworkMoniker` (legacy)
    - `TargetPlatformIdentifier`, `TargetPlatformVersion`, `TargetPlatformMinVersion`
  - Framework detection and parsing
  - NuGetFramework extraction

**`Common/MSBuildAssemblyResolver.cs`** - Dynamic MSBuild assembly loading
- **Responsibilities**:
  - Dynamically load `Microsoft.Build.dll` from specified directory
  - Dynamically load `Microsoft.Build.Framework.dll`
  - Handle architecture-specific folders (amd64, arm64)
  - Assembly resolution for MSBuild dependencies
  - Resource assembly loading for localization

---

### 2. **NuGet.Build.Tasks** (MSBuild Task Integration)

**Location**: `/NuGet.Client/src/NuGet.Core/NuGet.Build.Tasks/`

**Purpose**: MSBuild tasks that run during project build

**Key Tasks**:
- `GetProjectTargetFrameworksTask.cs` - Extract all target frameworks from project
- `GetRestoreProjectReferencesTask.cs` - Get project-to-project references
- `GetRestoreProjectStyleTask.cs` - Determine project style (PackageReference vs packages.config)
- `GetRestoreSolutionProjectsTask.cs` - Parse solution files
- `WarnForInvalidProjectsTask.cs` - Validate project structure

**Utilities**:
- `Common/MSBuildUtility.cs` - Shared utilities
- `Common/MSBuildTaskItem.cs` - MSBuild `ITaskItem` wrapper
- `Common/MSBuildLogger.cs` - MSBuild logger implementation

---

### 3. **NuGet.Build.Tasks.Pack** (Pack-specific MSBuild Tasks)

**Location**: `/NuGet.Client/src/NuGet.Core/NuGet.Build.Tasks.Pack/`

**Purpose**: MSBuild integration for `dotnet pack` (SDK-style projects)

**Key Files**:
- `PackTask.cs` - Main pack task for SDK-style projects
- `GetProjectReferencesFromAssetsFileTask.cs` - Resolve dependencies from assets file

---

### 4. **NuGet.Build.Tasks.Console** (Static Graph Restore)

**Location**: `/NuGet.Client/src/NuGet.Core/NuGet.Build.Tasks.Console/`

**Purpose**: MSBuild static graph restore (modern, efficient restore)

**Key Files**:
- `MSBuildStaticGraphRestore.cs` - Static graph restore implementation
- `MSBuildProjectInstance.cs` - Project instance wrapper
- `MSBuildProjectItemInstance.cs` - Project item wrapper
- `IMSBuildProject.cs` - MSBuild project abstraction

---

### 5. **NuGet.PackageManagement** (MSBuild Project Management)

**Location**: `/NuGet.Client/src/NuGet.Core/NuGet.PackageManagement/`

**Key Files**:
- `Projects/MSBuildNuGetProject.cs` - MSBuild-based NuGet project representation
- `Projects/IMSBuildProjectSystem.cs` - Interface for MSBuild project operations
- `Utility/MSBuildNuGetProjectSystemUtility.cs` - Utilities for MSBuild projects

---

### 6. **NuGet.Commands** (Command Implementation)

**Location**: `/NuGet.Client/src/NuGet.Core/NuGet.Commands/`

**Key Files**:
- `RestoreCommand/MSBuildRestoreItemGroup.cs` - MSBuild item groups for restore
- `RestoreCommand/MSBuildOutputFile.cs` - MSBuild output file generation
- `MSBuildPackTargetArgs.cs` - Pack arguments from MSBuild properties

---

## Architecture Details

### MSBuild Discovery Flow

**Windows (Primary)**:
1. Check `-MSBuildPath` flag if provided → use explicit path
2. Check `-MSBuildVersion` flag if provided → filter by version
3. Query Visual Studio Setup API (`ISetupInstance`):
   ```csharp
   var query = new SetupConfiguration() as ISetupConfiguration2;
   var enumInstances = query.EnumAllInstances();
   ```
4. Registry fallback for pre-VS2017 (MSBuild 14.0 and earlier):
   - `HKLM\SOFTWARE\Microsoft\MSBuild\ToolsVersions\{version}`
5. Environment variable fallback: `MSBUILD_EXE_PATH`

**Cross-Platform**:
- Look for MSBuild in PATH
- Check Mono MSBuild locations
- Check .NET SDK installations

### MSBuild Loading Flow

```
1. MSBuildAssemblyResolver(msbuildDirectory)
   ↓
2. Load Microsoft.Build.dll dynamically
   ↓
3. Load Microsoft.Build.Framework.dll dynamically
   ↓
4. Get Type: Microsoft.Build.Evaluation.Project (via reflection)
   ↓
5. Create Project instance: new Project(path, properties, toolsVersion)
   ↓
6. MSBuild evaluates project:
   - Processes <Import> elements
   - Evaluates conditions
   - Applies property values
   - Expands item groups
```

### Project Parsing Flow (pack command)

```
1. PackCommand receives arguments
   ↓
2. Create ProjectFactory with MSBuild directory
   ↓
3. ProjectFactory constructor:
   - Initialize MSBuildAssemblyResolver
   - Load Microsoft.Build.dll
   - Create Microsoft.Build.Evaluation.Project instance
   ↓
4. MSBuild evaluates project file:
   - Process all <Import> elements (SDK imports, .props, .targets)
   - Evaluate all conditions
   - Apply property values from command line (-Properties)
   - Expand item groups
   ↓
5. Extract metadata from evaluated project:
   - Package ID: $(PackageId) or $(AssemblyName)
   - Version: $(PackageVersion) or $(Version)
   - Authors: $(Authors)
   - Description: $(Description)
   - ... (all NuGet metadata properties)
   ↓
6. Collect files from MSBuild items:
   - Compile items (source files)
   - Content items (content files)
   - None items (misc files)
   - Apply file patterns and exclusions
   ↓
7. Handle IncludeReferencedProjects:
   - Recursively process <ProjectReference> items
   - Determine if reference should be dependency or included content
   ↓
8. Create .nupkg:
   - Generate .nuspec from metadata
   - Add collected files with correct structure
   - Apply OPC conventions
```

### Property Substitution

**Token Replacement in .nuspec**:
```xml
<package>
  <metadata>
    <id>$id$</id>
    <version>$version$</version>
    <authors>$author$</authors>
    <description>$description$</description>
  </metadata>
</package>
```

**Resolution**:
- `$id$` → `$(PackageId)` or `$(AssemblyName)`
- `$version$` → `$(PackageVersion)` or `$(Version)`
- `$author$` → `$(Authors)`
- `$description$` → `$(Description)`
- Plus many more...

**Property Sources** (in precedence order):
1. Command-line properties (`-Properties Configuration=Release`)
2. Project file properties
3. Imported .props files
4. SDK-provided properties
5. Default property values

---

## Key MSBuild APIs Used

From **Microsoft.Build.dll** (Microsoft.Build namespace):

```csharp
// Core types
Microsoft.Build.Evaluation.Project         // Represents an evaluated MSBuild project
Microsoft.Build.Evaluation.ProjectCollection // Manages project instances
Microsoft.Build.Evaluation.ProjectItem     // Represents a project item (file)
Microsoft.Build.Evaluation.ProjectProperty // Represents a project property

// Framework types
Microsoft.Build.Framework.ILogger          // Logger interface
Microsoft.Build.Framework.ITaskItem        // Task item interface
```

**Common Operations**:
```csharp
// Create project instance
var project = new Project(
    projectFile,           // Path to .csproj
    globalProperties,      // Dictionary<string, string> from -Properties
    toolsVersion: null     // Use default tools version
);

// Get property value
string packageId = project.GetPropertyValue("PackageId");

// Get all items of type
var compileItems = project.GetItems("Compile");

// Get evaluated property (with all conditions applied)
var targetFramework = project.GetProperty("TargetFramework")?.EvaluatedValue;
```

---

## For gonuget Implementation

### Required Capabilities

1. **MSBuild Discovery**
   - Visual Studio Setup API on Windows (COM interop)
   - PATH and environment variable fallback
   - Version selection and filtering

2. **Project File Parsing**
   - XML parsing of .csproj/.vbproj/.fsproj
   - SDK imports resolution (`<Project Sdk="Microsoft.NET.Sdk">`)
   - Property evaluation with conditions
   - Item group expansion

3. **Property Evaluation**
   - Variable substitution (`$(PropertyName)`)
   - Condition evaluation (`Condition="'$(Foo)'=='Bar'"`)
   - Property inheritance and precedence
   - Built-in properties (MSBuildProjectDirectory, etc.)

4. **File Collection**
   - Item group enumeration (Compile, Content, None)
   - Glob pattern expansion (`**/*.cs`)
   - File exclusion patterns
   - Metadata extraction from items

5. **Build Invocation** (for `-Build` flag)
   - Execute MSBuild.exe with appropriate arguments
   - Capture build output and errors
   - Detect build success/failure

### Implementation Options

**What NuGet.Client Actually Does** ⭐ (Verified from source code):

NuGet.Client uses **Option C: Hybrid approach with platform-specific MSBuild discovery**:

**Cross-Platform MSBuild Discovery Flow** (from `MsBuildUtility.cs:494`):
1. **Mono/Linux/macOS** (line 503):
   - Check well-known Mono paths: `/usr/lib/mono/msbuild/{version}/bin`
   - Supports MSBuild 15.0 and 14.1 on Mono
   - Uses XBuild for solution parsing on Mono (line 476)

2. **All Platforms** (line 512):
   - Check PATH environment variable for `msbuild` or `msbuild.exe`
   - Extract version from file metadata

3. **Windows** (line 522-548):
   - Load MSBuild from GAC (Global Assembly Cache)
   - Query Visual Studio Setup API for side-by-side installs (MSBuild 15.1+)
   - Enumerate all installed toolsets

4. **Platform-Specific Handling**:
   - **Mono**: Special argument escaping (`\\\"` vs `\"`) (line 315-349)
   - **Windows**: Architecture-specific folders (amd64, arm64) (line 585)

**NuGet.Client does NOT**:
- ❌ Reimplement MSBuild evaluation engine
- ❌ Parse .csproj as XML without MSBuild
- ❌ Work without MSBuild/XBuild installed

**NuGet.Client DOES**:
- ✅ Require MSBuild (Windows) or XBuild (Mono) to be installed
- ✅ Invoke MSBuild.exe as external process
- ✅ Use platform-specific discovery mechanisms
- ✅ Support cross-platform via Mono's XBuild on Linux/macOS

**Key Code References**:
- `MsBuildUtility.cs:676` - `GetMsBuildFromMonoPaths()` - Mono-specific discovery
- `MsBuildUtility.cs:494` - `GetMsBuildToolset()` - Main discovery logic
- `MsBuildUtility.cs:471` - `GetAllProjectFileNames()` - XBuild vs MSBuild routing
- `MsBuildUtility.cs:315` - `AddRestoreSources()` - Platform-specific escaping

### Recommended Approach for gonuget

**Follow NuGet.Client's Hybrid Approach** ⭐ (100% parity + cross-platform)

NuGet.Client is ALREADY cross-platform via Mono support. gonuget should follow the EXACT same approach:

**Phase 1: Basic .nuspec support** ✅ (Already done)

**Phase 2: MSBuild Discovery (Cross-Platform)**
- **Windows**:
  - Check `MSBUILD_EXE_PATH` environment variable
  - Check PATH for `MSBuild.exe`
  - Query Visual Studio Setup API (via COM interop or exec `vswhere.exe`)
  - Check registry for MSBuild 14.0 (VS2015) and earlier
- **Linux/macOS**:
  - Check PATH for `msbuild` (from Mono or .NET SDK)
  - Check well-known Mono paths: `/usr/lib/mono/msbuild/{version}/bin`
  - Check .NET SDK paths: `dotnet --list-sdks` to find MSBuild
- **All Platforms**:
  - Support `-MSBuildPath` flag (explicit path override)
  - Support `-MSBuildVersion` flag (version selection)

**Phase 3: MSBuild Invocation (Cross-Platform)**
- Execute MSBuild.exe (Windows) or msbuild (Linux/macOS) as external process
- Pass project file + properties via command-line arguments
- Use special escaping for Mono: `\\\"` instead of `\"`
- Parse MSBuild output for:
  - Package metadata (PackageId, Version, Authors, etc.)
  - File lists (Compile, Content, None items)
  - Dependency information
- Support `-Build` flag to build project before packing

**Phase 4: Advanced Features** (Full parity)
- `-IncludeReferencedProjects` support (recursive project references)
- Multi-targeting support (TargetFrameworks)
- Property substitution in .nuspec files
- Solution file parsing (both Windows MSBuild and XBuild on Mono)
- Custom MSBuild targets integration

**Why This Approach Works**:
- ✅ Same approach as NuGet.Client (100% parity)
- ✅ Cross-platform from day one (Windows + Linux + macOS)
- ✅ No need to reimplement MSBuild evaluation engine
- ✅ Leverages existing MSBuild installations
- ✅ Works with .NET SDK MSBuild (modern) and Mono MSBuild (legacy)

### Go Packages Needed

1. **XML Parsing**: `encoding/xml` (standard library) - for parsing MSBuild output
2. **Process Execution**: `os/exec` (standard library) - for invoking MSBuild
3. **Platform Detection**: `runtime` (standard library) - for Windows/Linux/macOS detection
4. **Path Handling**: `path/filepath` (standard library) - for path manipulation
5. **Windows COM** (optional): `github.com/go-ole/go-ole` - for Visual Studio Setup API (can also use `vswhere.exe`)

### Example Cross-Platform Implementation

```go
// packaging/msbuild/discovery.go
package msbuild

import (
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
)

// FindMSBuild discovers MSBuild on the current platform
func FindMSBuild() (string, error) {
    // 1. Check explicit path from flag/env var
    if path := os.Getenv("MSBUILD_EXE_PATH"); path != "" {
        return path, nil
    }

    // 2. Check PATH environment variable (all platforms)
    if path := findInPath(); path != "" {
        return path, nil
    }

    // 3. Platform-specific discovery
    switch runtime.GOOS {
    case "windows":
        return findMSBuildWindows()
    case "linux", "darwin":
        return findMSBuildUnix()
    default:
        return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
    }
}

// findInPath checks PATH for msbuild
func findInPath() string {
    // Windows: MSBuild.exe
    // Linux/macOS: msbuild
    executable := "MSBuild.exe"
    if runtime.GOOS != "windows" {
        executable = "msbuild"
    }

    path, _ := exec.LookPath(executable)
    return path
}

// findMSBuildWindows discovers MSBuild on Windows
func findMSBuildWindows() (string, error) {
    // Option 1: Use vswhere.exe (easiest)
    vswhere := filepath.Join(
        os.Getenv("ProgramFiles(x86)"),
        "Microsoft Visual Studio", "Installer", "vswhere.exe",
    )

    if _, err := os.Stat(vswhere); err == nil {
        // Execute: vswhere.exe -latest -requires Microsoft.Component.MSBuild -find MSBuild\**\Bin\MSBuild.exe
        cmd := exec.Command(vswhere,
            "-latest",
            "-requires", "Microsoft.Component.MSBuild",
            "-find", "MSBuild\\**\\Bin\\MSBuild.exe",
        )
        output, err := cmd.Output()
        if err == nil {
            return strings.TrimSpace(string(output)), nil
        }
    }

    // Option 2: Check registry (MSBuild 14.0 and earlier)
    // Option 3: Check well-known paths

    return "", fmt.Errorf("MSBuild not found on Windows")
}

// findMSBuildUnix discovers MSBuild on Linux/macOS
func findMSBuildUnix() (string, error) {
    // Option 1: .NET SDK MSBuild (modern, preferred)
    // Execute: dotnet --info to find SDK path
    cmd := exec.Command("dotnet", "--info")
    output, err := cmd.Output()
    if err == nil {
        // Parse output to find "Base Path: /usr/share/dotnet/sdk/..."
        // MSBuild.dll is in the same directory
        // Use: dotnet <path>/MSBuild.dll
    }

    // Option 2: Mono MSBuild (legacy)
    monoPaths := []string{
        "/usr/lib/mono/msbuild/15.0/bin/MSBuild.dll",
        "/usr/lib/mono/msbuild/14.1/bin/MSBuild.dll",
        "/Library/Frameworks/Mono.framework/Versions/Current/lib/mono/msbuild/15.0/bin/MSBuild.dll",
    }

    for _, path := range monoPaths {
        if _, err := os.Stat(path); err == nil {
            return path, nil
        }
    }

    return "", fmt.Errorf("MSBuild not found on Linux/macOS")
}

// InvokeMSBuild executes MSBuild with project file and properties
func InvokeMSBuild(msbuildPath, projectPath string, properties map[string]string) ([]byte, error) {
    args := []string{projectPath, "/t:GetPackageMetadata"}

    // Add properties
    for key, value := range properties {
        // Platform-specific escaping
        if runtime.GOOS != "windows" {
            // Mono needs extra escaping
            args = append(args, fmt.Sprintf("/p:%s=\\\"%s\\\"", key, value))
        } else {
            args = append(args, fmt.Sprintf("/p:%s=\"%s\"", key, value))
        }
    }

    cmd := exec.Command(msbuildPath, args...)
    return cmd.CombinedOutput()
}
```

**Key Implementation Notes**:
- On Windows: Use `vswhere.exe` (shipped with VS 2017+) to find MSBuild
- On Linux/macOS: Prefer .NET SDK MSBuild (`dotnet msbuild`) over legacy Mono XBuild
- Handle platform-specific escaping: Mono requires `\\\"` instead of `\"`
- Support `-MSBuildPath` flag to override discovery

---

## Critical MSBuild Features for Parity

### Must-Have (P0):
- [x] Parse .csproj/.vbproj/.fsproj files
- [x] Extract NuGet metadata properties
- [x] Collect files from item groups
- [x] Property substitution in .nuspec
- [x] `-MSBuildPath` flag support
- [x] `-Build` flag support
- [x] `-IncludeReferencedProjects` flag

### Should-Have (P1):
- [ ] SDK-style project support (`<Project Sdk="Microsoft.NET.Sdk">`)
- [ ] Condition evaluation in properties and items
- [ ] Glob pattern expansion (`**/*.cs`)
- [ ] Multi-targeting (TargetFrameworks)
- [ ] Custom MSBuild targets

### Nice-to-Have (P2):
- [ ] Full MSBuild evaluation engine
- [ ] Import resolution (`.props`, `.targets`)
- [ ] Property functions (`$([System.IO.Path]::GetFileName(...))`)

---

## Testing Strategy

### Interop Tests:
1. Create test projects (.csproj with various patterns)
2. Pack with nuget.exe → extract metadata from .nupkg
3. Pack with gonuget → extract metadata from .nupkg
4. Assert: Metadata matches exactly

### Test Cases:
- Simple project (no MSBuild magic)
- SDK-style project
- Multi-targeting project (TargetFrameworks)
- Project with referenced projects
- Project with conditions
- Project with glob patterns
- Project with custom metadata

---

## References

**NuGet.Client Source Code**:
- ProjectFactory: `NuGet.Client/src/NuGet.Clients/NuGet.CommandLine/Commands/ProjectFactory.cs`
- MsBuildUtility: `NuGet.Client/src/NuGet.Clients/NuGet.CommandLine/MsBuildUtility.cs`
- MSBuildAssemblyResolver: `NuGet.Client/src/NuGet.Clients/NuGet.CommandLine/Common/MSBuildAssemblyResolver.cs`

**MSBuild Documentation**:
- [MSBuild API](https://learn.microsoft.com/dotnet/api/microsoft.build)
- [MSBuild Project File Schema](https://learn.microsoft.com/visualstudio/msbuild/msbuild-project-file-schema-reference)
- [MSBuild Properties](https://learn.microsoft.com/visualstudio/msbuild/msbuild-properties)

**Visual Studio Setup API**:
- [ISetupConfiguration](https://learn.microsoft.com/dotnet/api/microsoft.visualstudio.setup.configuration.isetupconfiguration)
- [Finding MSBuild](https://learn.microsoft.com/visualstudio/msbuild/find-and-use-msbuild-versions)

---

## Summary: Cross-Platform MSBuild Strategy

**Answer to "What does NuGet.Client do?"**

NuGet.Client uses **Option C: Hybrid approach with platform-specific discovery** (NOT Option A Windows-only, NOT Option B XML parsing):

1. **Cross-Platform Architecture**:
   - Windows: Visual Studio Setup API → MSBuild.exe
   - Linux/macOS: .NET SDK → `dotnet msbuild` or Mono → `/usr/lib/mono/msbuild/.../MSBuild.dll`
   - All platforms: PATH environment variable fallback

2. **External Process Invocation**:
   - Does NOT reimplement MSBuild evaluation engine
   - Requires MSBuild to be installed on the system
   - Invokes MSBuild as external process with project file + properties
   - Parses MSBuild output for metadata and file lists

3. **Platform-Specific Handling**:
   - Windows: Uses .exe, normal escaping
   - Linux/macOS: Uses .dll (with `mono` or `dotnet`), special escaping (`\\\"` vs `\"`)
   - Architecture handling: amd64, arm64 folders on Windows

**For gonuget**: Follow NuGet.Client's exact approach for 100% parity + cross-platform support from day one.

---

**Status**: Reference document for gonuget MSBuild integration implementation (CROSS-PLATFORM)
**Next Steps**: Create implementation guide (milestone) for MSBuild integration in gonuget CLI

**Key Takeaway**: MSBuild integration is NOT Windows-only. NuGet.Client has FULL cross-platform support via Mono (legacy) and .NET SDK (modern).
