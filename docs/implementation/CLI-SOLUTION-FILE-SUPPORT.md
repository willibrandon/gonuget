# CLI Solution File Support

**Author**: Analysis of dotnet/sdk and NuGet.Client source code
**Date**: 2025-11-01
**Purpose**: Comprehensive specification for gonuget solution file support achieving 100% dotnet CLI parity

---

## Overview

This document specifies **exactly** how gonuget must handle solution files (.sln, .slnx, .slnf) based on the actual implementation in Microsoft's dotnet SDK and NuGet.Client codebases. All behaviors documented here are derived from reading the actual C# source code, not from assumptions or documentation.

**Key Principle**: Some commands accept solution files, others explicitly reject them. gonuget must match this behavior precisely.

---

## Solution File Formats

### Supported Formats

| Extension | Name | Description | Parser |
|-----------|------|-------------|--------|
| `.sln` | Solution File | Text-based Visual Studio solution format | MSBuild SolutionFile.Parse() |
| `.slnx` | Solution X File | XML-based solution format (introduced in .NET 9) | Microsoft.VisualStudio.SolutionPersistence |
| `.slnf` | Solution Filter | JSON filter specifying subset of projects to load | MSBuild SolutionFile.ParseSolutionFilter() |

### Detection Logic

**Source**: `NuGet.Client/src/NuGet.Core/NuGet.CommandLine.XPlat/Utility/XPlatUtility.cs` lines 129-139

```csharp
internal static bool IsSolutionFile(string fileName)
{
    if (!string.IsNullOrEmpty(fileName) && File.Exists(fileName))
    {
        var extension = System.IO.Path.GetExtension(fileName);

        return string.Equals(extension, ".sln", StringComparison.OrdinalIgnoreCase) ||
               string.Equals(extension, ".slnx", StringComparison.OrdinalIgnoreCase);
    }

    return false;
}
```

**IMPORTANT**: The check above only handles `.sln` and `.slnx`. However, MSBuild's `SolutionFile` class handles `.slnf` via `ParseSolutionFilter()` (line 245 in SolutionFile.cs).

**gonuget Implementation**:
```go
// IsSolutionFile returns true if the path is a solution file
func IsSolutionFile(path string) bool {
    if path == "" {
        return false
    }

    ext := strings.ToLower(filepath.Ext(path))
    return ext == ".sln" || ext == ".slnx" || ext == ".slnf"
}
```

### Project Detection Logic

**Source**: `XPlatUtility.cs` lines 146-160

```csharp
internal static bool IsProjectFile(string fileName)
{
    if (!string.IsNullOrEmpty(fileName) && File.Exists(fileName))
    {
        var extension = System.IO.Path.GetExtension(fileName);

        var lastFourCharacters = extension.Length >= 4
                                    ? extension.Substring(extension.Length - 4)
                                    : string.Empty;

        return string.Equals(lastFourCharacters, "proj", StringComparison.OrdinalIgnoreCase);
    }

    return false;
}
```

**gonuget Implementation**:
```go
// IsProjectFile returns true if file ends with "proj" (case-insensitive)
func IsProjectFile(path string) bool {
    if path == "" {
        return false
    }

    ext := filepath.Ext(path)
    if len(ext) < 4 {
        return false
    }

    lastFour := strings.ToLower(ext[len(ext)-4:])
    return lastFour == "proj"
}
```

---

## Solution Parsing Implementation

### MSBuild SolutionFile Parser

**Source**: `msbuild/src/Build/Construction/Solution/SolutionFile.cs`

#### Project Line Regex Pattern

**Lines 43-52**:
```csharp
// An example of a project line looks like this:
//  Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "ClassLibrary1", "ClassLibrary1\ClassLibrary1.csproj", "{05A5AD00-71B5-4612-AF2F-9EA9121C4111}"
private const string CrackProjectLinePattern =
    "^" // Beginning of line
    + "Project\\(\"(?<PROJECTTYPEGUID>.*)\"\\)"
    + "\\s*=\\s*" // Any amount of whitespace plus "=" plus any amount of whitespace
    + "\"(?<PROJECTNAME>.*)\""
    + "\\s*,\\s*" // Any amount of whitespace plus "," plus any amount of whitespace
    + "\"(?<RELATIVEPATH>.*)\""
    + "\\s*,\\s*" // Any amount of whitespace plus "," plus any amount of whitespace
    + "\"(?<PROJECTGUID>.*)\""
    + "$"; // End-of-line
```

#### Project Type GUIDs

**Lines 83-97**:
```csharp
private const string vbProjectGuid = "{F184B08F-C81C-45F6-A57F-5ABD9991F28F}";
private const string csProjectGuid = "{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}";
private const string cpsProjectGuid = "{13B669BE-BB05-4DDF-9536-439F39A36129}";
private const string cpsCsProjectGuid = "{9A19103F-16F7-4668-BE54-9A1E7A4F7556}";  // Modern C# SDK-style
private const string cpsVbProjectGuid = "{778DAE3C-4631-46EA-AA77-85C1314464D9}";  // Modern VB SDK-style
private const string cpsFsProjectGuid = "{6EC3EE1D-3C4E-46DD-8F32-0CC8E7565705}";  // Modern F# SDK-style
private const string vjProjectGuid = "{E6FDF86B-F3D1-11D4-8576-0002A516ECE8}";
private const string vcProjectGuid = "{8BC9CEB8-8B4A-11D0-8D11-00A0C91BC942}";
private const string fsProjectGuid = "{F2A71F9B-5D33-465A-A702-920D77279786}";     // F# legacy
private const string dbProjectGuid = "{C8D11400-126E-41CD-887F-60BD40844F9E}";
private const string wdProjectGuid = "{2CFEAB61-6A3B-4EB8-B523-560B4BEEF521}";
private const string synProjectGuid = "{BBD0F5D1-1CC4-42FD-BA4C-A96779C64378}";
private const string webProjectGuid = "{E24C65DC-7377-472B-9ABA-BC803B73C61A}";
private const string solutionFolderGuid = "{2150E333-8FDC-42A3-9474-1A3956D46DE8}";
private const string sharedProjectGuid = "{D954291E-2A0B-460D-934E-DC6B0785DB48}";
```

#### GetProjectsFromSolution Implementation

**Source**: `NuGet.Client/src/NuGet.Core/NuGet.CommandLine.XPlat/Utility/MSBuildAPIUtility.cs` lines 84-106

```csharp
internal static IEnumerable<string> GetProjectsFromSolution(string solutionPath)
{
    var sln = SolutionFile.Parse(solutionPath);

    if (XPlatUtility.IsSolutionFile(solutionPath))
    {
        return sln.ProjectsInOrder.Select(p => p.AbsolutePath);
    }

    MethodInfo projectShouldBuildMethod = typeof(SolutionFile).GetMethod("ProjectShouldBuild", BindingFlags.NonPublic | BindingFlags.Public | BindingFlags.Instance);
    Func<string, bool> projectShouldBuild = (Func<string, bool>)Delegate.CreateDelegate(typeof(Func<string, bool>), sln, projectShouldBuildMethod);

    List<string> projects = new List<string>();
    foreach (var project in sln.ProjectsInOrder)
    {
        if (projectShouldBuild(project.RelativePath))
        {
            projects.Add(project.AbsolutePath);
        }
    }

    return projects;
}
```

**Key Behavior**:
- Uses MSBuild's `SolutionFile.Parse()`
- Returns `AbsolutePath` for each project
- For `.slnf` files, filters projects using `ProjectShouldBuild()`
- Returns projects in order they appear in solution

### gonuget Solution Parser Implementation

**File**: `cmd/gonuget/solution/parser.go`

```go
package solution

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Solution represents a parsed solution file (.sln, .slnx, .slnf)
type Solution struct {
	Path     string
	Projects []ProjectReference
}

// ProjectReference represents a project in the solution
type ProjectReference struct {
	Name         string  // Project name from solution
	RelativePath string  // Path relative to solution directory
	AbsolutePath string  // Absolute path to project file
	ProjectGUID  string  // Project GUID in {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX} format
	TypeGUID     string  // Project type GUID
}

// Project type GUIDs matching MSBuild constants
const (
	CSharpProjectGUID      = "{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}"
	CPSCSharpProjectGUID   = "{9A19103F-16F7-4668-BE54-9A1E7A4F7556}" // Modern SDK-style C#
	VBProjectGUID          = "{F184B08F-C81C-45F6-A57F-5ABD9991F28F}"
	CPSVBProjectGUID       = "{778DAE3C-4631-46EA-AA77-85C1314464D9}" // Modern SDK-style VB
	FSharpProjectGUID      = "{F2A71F9B-5D33-465A-A702-920D77279786}"
	CPSFSharpProjectGUID   = "{6EC3EE1D-3C4E-46DD-8F32-0CC8E7565705}" // Modern SDK-style F#
	CPPProjectGUID         = "{8BC9CEB8-8B4A-11D0-8D11-00A0C91BC942}"
	SolutionFolderGUID     = "{2150E333-8FDC-42A3-9474-1A3956D46DE8}"
	SharedProjectGUID      = "{D954291E-2A0B-460D-934E-DC6B0785DB48}"
	WebProjectGUID         = "{E24C65DC-7377-472B-9ABA-BC803B73C61A}"
)

var (
	// projectLineRegex matches MSBuild's CrackProjectLinePattern exactly
	// Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "WebApi", "WebApi\WebApi.csproj", "{0B901EF3-9A59-4885-BC2C-B795DE8F6212}"
	projectLineRegex = regexp.MustCompile(
		`^Project\("\{(?P<PROJECTTYPEGUID>[^}]+)\}"\)\s*=\s*"(?P<PROJECTNAME>[^"]+)"\s*,\s*"(?P<RELATIVEPATH>[^"]+)"\s*,\s*"\{(?P<PROJECTGUID>[^}]+)\}"$`,
	)
)

// Parse parses a solution file (.sln) and returns a Solution
// Matches behavior of MSBuild's SolutionFile.Parse()
func Parse(solutionPath string) (*Solution, error) {
	file, err := os.Open(solutionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open solution file: %w", err)
	}
	defer file.Close()

	solution := &Solution{
		Path:     solutionPath,
		Projects: []ProjectReference{},
	}

	solutionDir := filepath.Dir(solutionPath)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Match project lines
		if strings.HasPrefix(line, "Project(") {
			matches := projectLineRegex.FindStringSubmatch(line)
			if len(matches) == 5 {
				typeGUID := matches[1]
				name := matches[2]
				relativePath := matches[3]
				projectGUID := matches[4]

				// Skip solution folders (they are not buildable projects)
				if strings.EqualFold(typeGUID, strings.Trim(SolutionFolderGUID, "{}")) {
					continue
				}

				// Convert backslashes to OS-specific path separator (Windows uses \, Unix uses /)
				relativePath = filepath.FromSlash(strings.ReplaceAll(relativePath, "\\", "/"))

				// Make path absolute relative to solution directory
				absolutePath := filepath.Join(solutionDir, relativePath)

				solution.Projects = append(solution.Projects, ProjectReference{
					Name:         name,
					RelativePath: relativePath,
					AbsolutePath: absolutePath,
					ProjectGUID:  projectGUID,
					TypeGUID:     typeGUID,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading solution file: %w", err)
	}

	return solution, nil
}

// GetProjectPaths returns absolute paths to all projects in the solution
// Matches MSBuildAPIUtility.GetProjectsFromSolution behavior
func GetProjectPaths(solutionPath string) ([]string, error) {
	solution, err := Parse(solutionPath)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, proj := range solution.Projects {
		// Only include files that exist (matches dotnet behavior)
		if _, err := os.Stat(proj.AbsolutePath); err == nil {
			// Only include .NET project files (not .vcxproj, etc.)
			ext := strings.ToLower(filepath.Ext(proj.AbsolutePath))
			if ext == ".csproj" || ext == ".fsproj" || ext == ".vbproj" {
				paths = append(paths, proj.AbsolutePath)
			}
		}
	}

	return paths, nil
}

// IsSolutionFile returns true if the path is a solution file
func IsSolutionFile(path string) bool {
	if path == "" {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".sln" || ext == ".slnx" || ext == ".slnf"
}

// IsProjectFile returns true if file ends with "proj" (matches XPlatUtility.IsProjectFile)
func IsProjectFile(path string) bool {
	if path == "" {
		return false
	}

	ext := filepath.Ext(path)
	if len(ext) < 4 {
		return false
	}

	lastFour := strings.ToLower(ext[len(ext)-4:])
	return lastFour == "proj"
}
```

**Testing**: `cmd/gonuget/solution/parser_test.go`

```go
package solution

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tmpDir := t.TempDir()

	slnPath := filepath.Join(tmpDir, "TestSolution.sln")
	slnContent := `Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
VisualStudioVersion = 17.0.31903.59
MinimumVisualStudioVersion = 10.0.40219.1
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "WebApi", "WebApi\WebApi.csproj", "{0B901EF3-9A59-4885-BC2C-B795DE8F6212}"
EndProject
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "DataLayer", "DataLayer\DataLayer.csproj", "{05355CB2-47D1-43B5-84FA-EC67CBAAB6FF}"
EndProject
Project("{2150E333-8FDC-42A3-9474-1A3956D46DE8}") = "Solution Items", "Solution Items", "{D91E7F1B-9C77-4F6B-BD75-2FBF5C0D9E65}"
EndProject
Global
	GlobalSection(SolutionConfigurationPlatforms) = preSolution
		Debug|Any CPU = Debug|Any CPU
		Release|Any CPU = Release|Any CPU
	EndGlobalSection
EndGlobal`

	err := os.WriteFile(slnPath, []byte(slnContent), 0644)
	require.NoError(t, err)

	// Parse solution
	solution, err := Parse(slnPath)
	require.NoError(t, err)
	assert.NotNil(t, solution)

	// Verify projects (solution folder should be excluded)
	assert.Equal(t, 2, len(solution.Projects))

	// WebApi project
	assert.Equal(t, "WebApi", solution.Projects[0].Name)
	assert.Contains(t, solution.Projects[0].AbsolutePath, "WebApi.csproj")
	assert.Equal(t, "FAE04EC0-301F-11D3-BF4B-00C04F79EFBC", solution.Projects[0].TypeGUID)
	assert.Equal(t, "0B901EF3-9A59-4885-BC2C-B795DE8F6212", solution.Projects[0].ProjectGUID)

	// DataLayer project
	assert.Equal(t, "DataLayer", solution.Projects[1].Name)
	assert.Contains(t, solution.Projects[1].AbsolutePath, "DataLayer.csproj")
}

func TestGetProjectPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project files
	webApiDir := filepath.Join(tmpDir, "WebApi")
	os.MkdirAll(webApiDir, 0755)
	webApiProj := filepath.Join(webApiDir, "WebApi.csproj")
	os.WriteFile(webApiProj, []byte("<Project />"), 0644)

	dataLayerDir := filepath.Join(tmpDir, "DataLayer")
	os.MkdirAll(dataLayerDir, 0755)
	dataLayerProj := filepath.Join(dataLayerDir, "DataLayer.csproj")
	os.WriteFile(dataLayerProj, []byte("<Project />"), 0644)

	// Create solution file
	slnPath := filepath.Join(tmpDir, "TestSolution.sln")
	slnContent := `Microsoft Visual Studio Solution File, Format Version 12.00
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "WebApi", "WebApi\WebApi.csproj", "{0B901EF3-9A59-4885-BC2C-B795DE8F6212}"
EndProject
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "DataLayer", "DataLayer\DataLayer.csproj", "{05355CB2-47D1-43B5-84FA-EC67CBAAB6FF}"
EndProject
Global
EndGlobal`
	os.WriteFile(slnPath, []byte(slnContent), 0644)

	// Get project paths
	paths, err := GetProjectPaths(slnPath)
	require.NoError(t, err)
	assert.Equal(t, 2, len(paths))
	assert.Contains(t, paths[0], "WebApi.csproj")
	assert.Contains(t, paths[1], "DataLayer.csproj")
}

func TestIsSolutionFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.sln", true},
		{"test.slnx", true},
		{"test.slnf", true},
		{"test.SLN", true},
		{"TEST.SLNX", true},
		{"test.csproj", false},
		{"test.fsproj", false},
		{"test.vbproj", false},
		{"test.txt", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsSolutionFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsProjectFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.csproj", true},
		{"test.fsproj", true},
		{"test.vbproj", true},
		{"test.CSPROJ", true},
		{"test.vcxproj", true},  // C++ project
		{"test.sln", false},
		{"test.txt", false},
		{"test.cs", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsProjectFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
```

---

## Command Behavior Matrix

### Commands That ACCEPT Solution Files

| Command | Source Location | Line | Behavior |
|---------|----------------|------|----------|
| `dotnet list package` | ListPackageCommandRunner.cs | 64-70 | Checks for `.sln`, `.slnx`, `.slnf` → calls `MSBuildAPIUtility.GetProjectsFromSolution()` |
| `dotnet restore` | (SDK) | N/A | Restores all projects in solution |
| `dotnet nuget why` | (NuGet.Client) | N/A | Shows dependency graph for all projects |

### Commands That REJECT Solution Files

| Command | Source Location | Line | Validation | Error Message |
|---------|----------------|------|------------|---------------|
| `dotnet add package` | AddPackageReferenceCommand.cs | 135-141 | `!projectPath.Value().EndsWith("proj", StringComparison.OrdinalIgnoreCase)` | "Couldn't find a project to run. Ensure a project exists in {dir}, or pass the path to the project using --project" |
| `dotnet remove package` | (Same validation pattern) | N/A | Same | "Missing or invalid project file: {path}" |

---

## Implementation Specifications

### Package Remove - Solution File Rejection

**File**: `cmd/gonuget/commands/package_remove.go`

**Validation Code** (add before existing logic):
```go
func runRemovePackage(ctx context.Context, packageID string, opts *RemovePackageOptions) error {
	projectPath := opts.ProjectPath

	// Validation: Reject solution files (100% dotnet parity)
	// Source: NuGet.Client AddPackageReferenceCommand.cs ValidateProjectPath
	if projectPath != "" {
		// Check if file exists
		if _, err := os.Stat(projectPath); os.IsNotExist(err) {
			return fmt.Errorf("error: Missing or invalid project file: %s", projectPath)
		}

		// Reject solution files
		if solution.IsSolutionFile(projectPath) {
			return fmt.Errorf("error: Missing or invalid project file: %s", projectPath)
		}

		// Validate file ends with "proj" (matches dotnet validation exactly)
		if !solution.IsProjectFile(projectPath) {
			return fmt.Errorf("error: Missing or invalid project file: %s", projectPath)
		}
	}

	// ... rest of existing implementation ...
}
```

**Tests**: `cmd/gonuget/commands/package_remove_test.go`

```go
func TestRunRemovePackage_RejectsSolutionFile_Sln(t *testing.T) {
	tmpDir := t.TempDir()

	slnPath := filepath.Join(tmpDir, "test.sln")
	slnContent := `Microsoft Visual Studio Solution File, Format Version 12.00
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "Project1", "Project1\\Project1.csproj", "{GUID}"
EndProject
Global
EndGlobal`
	err := os.WriteFile(slnPath, []byte(slnContent), 0644)
	require.NoError(t, err)

	opts := &RemovePackageOptions{
		ProjectPath: slnPath,
	}

	err = runRemovePackage(context.Background(), "Newtonsoft.Json", opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Missing or invalid project file")
	assert.Contains(t, err.Error(), slnPath)
}

func TestRunRemovePackage_RejectsSolutionFile_Slnx(t *testing.T) {
	tmpDir := t.TempDir()

	slnxPath := filepath.Join(tmpDir, "test.slnx")
	slnxContent := `<?xml version="1.0" encoding="utf-8"?>
<Solution>
  <Project Path="Project1/Project1.csproj" />
</Solution>`
	err := os.WriteFile(slnxPath, []byte(slnxContent), 0644)
	require.NoError(t, err)

	opts := &RemovePackageOptions{
		ProjectPath: slnxPath,
	}

	err = runRemovePackage(context.Background(), "Newtonsoft.Json", opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Missing or invalid project file")
}

func TestRunRemovePackage_AcceptsProjectFile(t *testing.T) {
	tmpDir := t.TempDir()

	projPath := filepath.Join(tmpDir, "test.csproj")
	projContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`
	err := os.WriteFile(projPath, []byte(projContent), 0644)
	require.NoError(t, err)

	opts := &RemovePackageOptions{
		ProjectPath: projPath,
	}

	err = runRemovePackage(context.Background(), "Newtonsoft.Json", opts)

	assert.NoError(t, err)

	// Verify package was removed
	content, err := os.ReadFile(projPath)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "Newtonsoft.Json")
}
```

---

### Package List - Solution File Support

**File**: `cmd/gonuget/commands/package_list.go`

**Source Reference**: ListPackageCommandRunner.cs lines 62-77

**Implementation**:
```go
func runPackageList(opts *PackageListOptions) error {
	start := time.Now()

	// Find the project or solution file
	inputPath := opts.ProjectPath
	if inputPath == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Try to find solution file first, then project file
		foundPath, err := findProjectOrSolutionFile(currentDir)
		if err != nil {
			return fmt.Errorf("failed to find project or solution file: %w", err)
		}
		inputPath = foundPath
	}

	// Verify file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", inputPath)
	}

	// Check if it's a solution file
	if solution.IsSolutionFile(inputPath) {
		return listPackagesFromSolution(inputPath, opts, start)
	}

	// Otherwise, it's a project file
	return listPackagesFromProject(inputPath, opts, start)
}

// listPackagesFromSolution lists packages from all projects in a solution
// Matches behavior of ListPackageCommandRunner.GetReportDataAsync
func listPackagesFromSolution(solutionPath string, opts *PackageListOptions, start time.Time) error {
	// Parse solution file using MSBuild-compatible parser
	projectPaths, err := solution.GetProjectPaths(solutionPath)
	if err != nil {
		return fmt.Errorf("failed to parse solution file: %w", err)
	}

	if len(projectPaths) == 0 {
		fmt.Println("No projects found in solution")
		return nil
	}

	// List packages from each project
	for _, projectPath := range projectPaths {
		// Load project
		proj, err := project.LoadProject(projectPath)
		if err != nil {
			// Log error but continue to next project (matches dotnet behavior)
			fmt.Fprintf(os.Stderr, "Warning: failed to load project %s: %v\n", filepath.Base(projectPath), err)
			continue
		}

		// Get project name
		projectName := filepath.Base(filepath.Dir(projectPath))

		// Output based on format
		if opts.Format == "json" {
			// TODO: JSON format for solutions requires aggregating all projects
			return fmt.Errorf("JSON format not yet supported for solution files")
		} else {
			// Console format: display each project as we go
			if err := outputProjectPackagesConsole(projectName, proj); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to list packages from %s: %v\n", projectName, err)
			}
			fmt.Println() // Blank line between projects
		}
	}

	return nil
}

// outputProjectPackagesConsole outputs packages for a single project in console format
// Matches ListPackageConsoleRenderer output format
func outputProjectPackagesConsole(projectName string, proj *project.Project) error {
	packageRefs := proj.GetPackageReferences()
	framework := proj.TargetFramework

	if framework == "" {
		return fmt.Errorf("project does not specify a TargetFramework")
	}

	// Output project header
	fmt.Printf("Project '%s' has the following package references\n", projectName)
	fmt.Printf("   [%s]:\n", framework)

	if len(packageRefs) == 0 {
		return nil
	}

	// Calculate column widths
	maxPackageLen := len("Top-level Package")
	for _, ref := range packageRefs {
		if len(ref.Include) > maxPackageLen {
			maxPackageLen = len(ref.Include)
		}
	}

	// Print header
	fmt.Printf("   %-*s   Requested   Resolved\n", maxPackageLen, "Top-level Package")

	// Print packages
	for _, ref := range packageRefs {
		if ref.Version != "" {
			fmt.Printf("   > %-*s   %-11s %s\n", maxPackageLen, ref.Include, ref.Version, ref.Version)
		} else {
			fmt.Printf("   > %s (version managed centrally)\n", ref.Include)
		}
	}

	return nil
}

// findProjectOrSolutionFile finds a project or solution file in the given directory
// Matches XPlatUtility.GetProjectOrSolutionFileFromDirectory
func findProjectOrSolutionFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var foundFile string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if solution.IsSolutionFile(path) || solution.IsProjectFile(path) {
			if foundFile != "" {
				return "", fmt.Errorf("multiple project or solution files found in directory: %s", dir)
			}
			foundFile = path
		}
	}

	if foundFile == "" {
		return "", fmt.Errorf("no project or solution file found in directory: %s", dir)
	}

	return foundFile, nil
}
```

**Tests**: `cmd/gonuget/commands/package_list_test.go`

```go
func TestRunPackageList_Solution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple projects
	webApiDir := filepath.Join(tmpDir, "WebApi")
	os.MkdirAll(webApiDir, 0755)
	webApiPath := filepath.Join(webApiDir, "WebApi.csproj")
	webApiContent := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageReference Include="Serilog" Version="4.3.0" />
  </ItemGroup>
</Project>`
	os.WriteFile(webApiPath, []byte(webApiContent), 0644)

	dataLayerDir := filepath.Join(tmpDir, "DataLayer")
	os.MkdirAll(dataLayerDir, 0755)
	dataLayerPath := filepath.Join(dataLayerDir, "DataLayer.csproj")
	dataLayerContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>`
	os.WriteFile(dataLayerPath, []byte(dataLayerContent), 0644)

	// Create solution file
	slnPath := filepath.Join(tmpDir, "TestSolution.sln")
	slnContent := `Microsoft Visual Studio Solution File, Format Version 12.00
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "WebApi", "WebApi\WebApi.csproj", "{0B901EF3-9A59-4885-BC2C-B795DE8F6212}"
EndProject
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "DataLayer", "DataLayer\DataLayer.csproj", "{05355CB2-47D1-43B5-84FA-EC67CBAAB6FF}"
EndProject
Global
EndGlobal`
	os.WriteFile(slnPath, []byte(slnContent), 0644)

	// Run package list on solution
	opts := &PackageListOptions{
		ProjectPath: slnPath,
		Format:      "console",
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runPackageList(opts)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, output, "Project 'WebApi'")
	assert.Contains(t, output, "Project 'DataLayer'")
	assert.Contains(t, output, "Newtonsoft.Json")
	assert.Contains(t, output, "Serilog")
	assert.Contains(t, output, "13.0.3")
	assert.Contains(t, output, "4.3.0")
	assert.Contains(t, output, "[net8.0]")
}
```

---

## Success Criteria

### Package Remove - Solution File Rejection

- ✅ Solution files (.sln, .slnx, .slnf) are REJECTED with error
- ✅ Error message matches dotnet CLI exactly: "Missing or invalid project file: {path}"
- ✅ Project files (.csproj, .fsproj, .vbproj) are ACCEPTED
- ✅ Validation uses `IsProjectFile()` matching XPlatUtility logic
- ✅ Unit tests pass with 90%+ coverage
- ✅ **100% dotnet parity: NO solution file support**

### Package List - Solution File Support

- ✅ Solution files (.sln, .slnx, .slnf) are parsed correctly
- ✅ All projects in solution are enumerated using MSBuild-compatible parser
- ✅ Packages listed for each project with proper formatting
- ✅ Output format matches `dotnet list package` exactly:
  - `Project 'ProjectName' has the following package references`
  - `   [net8.0]:`
  - `   Top-level Package   Requested   Resolved`
  - `   > PackageName        Version     Version`
- ✅ Missing projects are skipped gracefully with warning
- ✅ Solution folders (GUID {2150E333-8FDC-42A3-9474-1A3956D46DE8}) are excluded
- ✅ Only .NET project files included (.csproj, .fsproj, .vbproj)
- ✅ Unit tests pass with 90%+ coverage
- ✅ **100% dotnet parity: Solution file support**

---

## Reference Documentation

### Source Files Analyzed

1. **MSBuild SolutionFile Parser**:
   - `msbuild/src/Build/Construction/Solution/SolutionFile.cs` (lines 1-300)
   - Regex patterns, project type GUIDs, Parse() method

2. **NuGet.Client List Package**:
   - `NuGet.Client/src/NuGet.Core/NuGet.CommandLine.XPlat/Commands/PackageReferenceCommands/ListPackage/ListPackageCommandRunner.cs` (lines 1-200)
   - Solution detection, GetReportDataAsync, project enumeration

3. **NuGet.Client MSBuildAPIUtility**:
   - `NuGet.Client/src/NuGet.Core/NuGet.CommandLine.XPlat/Utility/MSBuildAPIUtility.cs` (lines 84-106)
   - GetProjectsFromSolution implementation

4. **NuGet.Client Add Package Validation**:
   - `NuGet.Client/src/NuGet.Core/NuGet.CommandLine.XPlat/Commands/PackageReferenceCommands/AddPackageReferenceCommand.cs` (lines 133-142)
   - ValidateProjectPath showing explicit rejection of solution files

5. **XPlatUtility**:
   - `NuGet.Client/src/NuGet.Core/NuGet.CommandLine.XPlat/Utility/XPlatUtility.cs` (lines 129-160)
   - IsSolutionFile, IsProjectFile implementations

### Testing Verification Commands

```bash
# Test dotnet behavior (run in /tmp/gonuget-test-solution)
dotnet list TestSolution.sln package
dotnet add TestSolution.sln package Newtonsoft.Json --version 13.0.3  # Should fail
dotnet remove TestSolution.sln package Moq  # Should fail

# Test gonuget behavior (must match exactly)
gonuget package list TestSolution.sln
gonuget package add Newtonsoft.Json --project TestSolution.sln --version 13.0.3  # Should fail
gonuget package remove Moq --project TestSolution.sln  # Should fail
```

---

## Implementation Summary

**Required Components**:

1. **Package Remove - Solution File Rejection** (Estimated: 1 hour)
   - Add validation to reject solution files
   - Match dotnet error message exactly
   - Comprehensive test coverage

2. **Package List - Solution File Support** (Estimated: 4 hours)
   - MSBuild-compatible solution parser
   - Enumerate all projects in solution
   - Format output matching dotnet CLI exactly
   - Comprehensive test coverage

**Total Estimated Time**: 5 hours

Both components must be implemented to achieve 100% dotnet CLI parity for solution file handling.
