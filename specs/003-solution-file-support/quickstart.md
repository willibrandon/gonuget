# Quick Start: Solution File Support Implementation

**Feature**: Solution File Support for gonuget CLI
**Estimated Time**: 5 minutes to understand, 2-3 hours to implement core

## 5-Minute Implementation Guide

### Step 1: Create Core Types (2 min)

Create `cmd/gonuget/solution/types.go`:

```go
package solution

// Solution represents a parsed solution file
type Solution struct {
    FilePath            string
    FormatVersion       string
    VisualStudioVersion string
    Projects            []Project
    SolutionDir         string
}

// Project represents a project in a solution
type Project struct {
    Name     string
    Path     string
    GUID     string
    TypeGUID string
}

// IsNETProject returns true for .NET project types
func (p *Project) IsNETProject() bool {
    return p.TypeGUID == "{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}" || // C#
           p.TypeGUID == "{F184B08F-C81C-45F6-A57F-5ABD9991F28F}" || // VB.NET
           p.TypeGUID == "{F2A71F9B-5D33-465A-A702-920D77279786}"    // F#
}

// ParseError represents a solution parsing error
type ParseError struct {
    FilePath string
    Line     int
    Message  string
}

func (e *ParseError) Error() string {
    if e.Line > 0 {
        return fmt.Sprintf("%s:%d: %s", e.FilePath, e.Line, e.Message)
    }
    return fmt.Sprintf("%s: %s", e.FilePath, e.Message)
}
```

### Step 2: Create Parser Interface (1 min)

Create `cmd/gonuget/solution/parser.go`:

```go
package solution

import "strings"

// Parser interface for solution file parsing
type Parser interface {
    Parse(path string) (*Solution, error)
    CanParse(path string) bool
}

// GetParser returns appropriate parser for file type
func GetParser(path string) (Parser, error) {
    ext := strings.ToLower(filepath.Ext(path))
    switch ext {
    case ".sln":
        return NewSlnParser(), nil
    case ".slnx":
        return NewSlnxParser(), nil
    case ".slnf":
        return NewSlnfParser(), nil
    default:
        return nil, fmt.Errorf("unsupported solution format: %s", ext)
    }
}
```

### Step 3: Implement Basic .sln Parser (2 min)

Create `cmd/gonuget/solution/sln_parser.go`:

```go
package solution

import (
    "bufio"
    "os"
    "path/filepath"
    "regexp"
    "strings"
)

type SlnParser struct{}

func NewSlnParser() *SlnParser {
    return &SlnParser{}
}

func (p *SlnParser) CanParse(path string) bool {
    return strings.ToLower(filepath.Ext(path)) == ".sln"
}

func (p *SlnParser) Parse(path string) (*Solution, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    absPath, _ := filepath.Abs(path)
    sol := &Solution{
        FilePath:    absPath,
        SolutionDir: filepath.Dir(absPath),
        Projects:    []Project{},
    }

    // Regex for project lines
    projectRegex := regexp.MustCompile(
        `^Project\("\{([A-F0-9-]+)\}"\)\s*=\s*"([^"]+)",\s*"([^"]+)",\s*"\{([A-F0-9-]+)\}"`)

    scanner := bufio.NewScanner(file)
    lineNum := 0

    for scanner.Scan() {
        lineNum++
        line := scanner.Text()

        // Parse format version
        if strings.Contains(line, "Format Version") {
            parts := strings.Split(line, "Format Version ")
            if len(parts) == 2 {
                sol.FormatVersion = strings.TrimSpace(parts[1])
            }
        }

        // Parse projects
        if matches := projectRegex.FindStringSubmatch(line); matches != nil {
            typeGUID := "{" + strings.ToUpper(matches[1]) + "}"

            // Skip solution folders
            if typeGUID == "{2150E333-8FDC-42A3-9474-1A3956D46DE8}" {
                continue
            }

            projectPath := filepath.ToSlash(matches[3]) // Convert backslashes
            if !filepath.IsAbs(projectPath) {
                projectPath = filepath.Join(sol.SolutionDir, projectPath)
            }

            sol.Projects = append(sol.Projects, Project{
                Name:     matches[2],
                Path:     projectPath,
                GUID:     "{" + strings.ToUpper(matches[4]) + "}",
                TypeGUID: typeGUID,
            })
        }
    }

    return sol, scanner.Err()
}
```

## Integration Points

### Modify Package List Command

In `cmd/gonuget/commands/package_list.go`:

```go
func (cmd *PackageListCmd) Run() error {
    // Check if input is solution file
    if IsSolutionFile(cmd.Path) {
        return cmd.listSolutionPackages()
    }

    // Existing project logic...
    return cmd.listProjectPackages()
}

func (cmd *PackageListCmd) listSolutionPackages() error {
    parser, err := solution.GetParser(cmd.Path)
    if err != nil {
        return err
    }

    sol, err := parser.Parse(cmd.Path)
    if err != nil {
        return err
    }

    for _, project := range sol.Projects {
        if !project.IsNETProject() {
            continue
        }

        fmt.Printf("Project '%s' has the following package references\n", project.Name)

        // Use existing package list logic for this project
        packages, err := cmd.getProjectPackages(project.Path)
        if err != nil {
            // Silently skip missing projects (dotnet CLI behavior)
            continue
        }

        cmd.formatPackageOutput(packages)
        fmt.Println() // Blank line between projects
    }

    return nil
}
```

### Add Error Handling for Add/Remove

In `cmd/gonuget/commands/package_add.go`:

```go
func (cmd *PackageAddCmd) Validate() error {
    if IsSolutionFile(cmd.ProjectPath) {
        dir := filepath.Dir(cmd.ProjectPath)
        return fmt.Errorf("Couldn't find a project to run. Ensure a project exists in %s, or pass the path to the project using --project", dir)
    }
    // Existing validation...
}
```

## Testing

### Create Test Solution File

Create `tests/cmd/gonuget/solution/testdata/simple.sln`:

```
Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "WebApi", "src\WebApi\WebApi.csproj", "{11111111-1111-1111-1111-111111111111}"
EndProject
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "DataLayer", "src\DataLayer\DataLayer.csproj", "{22222222-2222-2222-2222-222222222222}"
EndProject
Global
    GlobalSection(SolutionConfigurationPlatforms) = preSolution
        Debug|Any CPU = Debug|Any CPU
        Release|Any CPU = Release|Any CPU
    EndGlobalSection
EndGlobal
```

### Write Basic Test

```go
func TestSlnParser(t *testing.T) {
    parser := NewSlnParser()

    sol, err := parser.Parse("testdata/simple.sln")
    if err != nil {
        t.Fatal(err)
    }

    if len(sol.Projects) != 2 {
        t.Errorf("Expected 2 projects, got %d", len(sol.Projects))
    }

    if sol.Projects[0].Name != "WebApi" {
        t.Errorf("Expected first project to be WebApi")
    }
}
```

## Validation Checklist

- [ ] Can parse .sln file with multiple projects
- [ ] Filters out solution folders
- [ ] Handles cross-platform paths correctly
- [ ] Package list shows all projects
- [ ] Package add rejects solution with correct error
- [ ] Package remove rejects solution with correct error
- [ ] Output matches dotnet CLI format

## Common Pitfalls to Avoid

1. **Don't forget UTF-8 BOM**: Some .sln files start with BOM bytes
2. **Handle backslashes**: Windows paths use backslashes, convert them
3. **Skip solution folders**: GUID {2150E333-8FDC-42A3-9474-1A3956D46DE8}
4. **Silent skip**: Don't warn for missing projects in package list
5. **Exact errors**: Error messages must match dotnet CLI exactly

## Next Steps

After basic implementation works:

1. Add .slnx parser (XML-based)
2. Add .slnf parser (JSON filter)
3. Add comprehensive tests
4. Add performance benchmarks
5. Integration test against real dotnet CLI output

## Troubleshooting

**Problem**: Paths not resolving correctly
**Solution**: Use `filepath.ToSlash()` for Windows paths, `filepath.Join()` for building paths

**Problem**: Solution folders appearing in project list
**Solution**: Check TypeGUID equals {2150E333-8FDC-42A3-9474-1A3956D46DE8}

**Problem**: Package list output doesn't match dotnet
**Solution**: Check spacing, indentation, and blank lines between projects

**Problem**: Error messages don't match
**Solution**: Use exact string constants from dotnet CLI, including punctuation