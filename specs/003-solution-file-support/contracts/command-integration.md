# Contract: Command Integration

**Feature**: Solution File Support for gonuget CLI
**Date**: 2025-11-01

## Package List Command Enhancement

### Current Behavior
```go
// gonuget package list [PROJECT]
type PackageListCommand struct {
    ProjectPath string
    // ... other fields
}
```

### Enhanced Behavior
```go
// gonuget package list [PROJECT_OR_SOLUTION]
type PackageListCommand struct {
    Path string  // Can be project or solution file
    // ... other fields
}
```

**Contract**:
- If Path is a solution file (.sln, .slnx, .slnf):
  - Parse solution to get all projects
  - Call existing ListPackages for each project
  - Format output with project headers
  - Silently skip missing projects
- If Path is a project file:
  - Existing behavior unchanged
- If Path is empty:
  - Search current directory (existing behavior)

**Output Format** (Multiple Projects):
```
Project 'WebApi' has the following package references
   [net8.0]:
   Top-level Package      Requested   Resolved
   > Microsoft.AspNetCore 8.0.0       8.0.0
   > Newtonsoft.Json      13.0.3      13.0.3

Project 'DataLayer' has the following package references
   [net8.0]:
   Top-level Package      Requested   Resolved
   > EntityFramework      7.0.0       7.0.0

Project 'Tests' has the following package references
   [net8.0]:
   Top-level Package      Requested   Resolved
   > xUnit                2.5.0       2.5.0
   > Moq                  4.20.0      4.20.0
```

## Package Add Command Enhancement

### Current Behavior
```go
// gonuget package add <PACKAGE> [PROJECT]
type PackageAddCommand struct {
    PackageID   string
    ProjectPath string
    // ... other fields
}
```

### Enhanced Behavior
```go
// Detect and reject solution files with specific error
func (cmd *PackageAddCommand) ValidateInput() error {
    if IsSolutionFile(cmd.ProjectPath) {
        return &SolutionNotSupportedError{
            Message: "Couldn't find a project to run. Ensure a project exists in {dir}, or pass the path to the project using --project",
            Directory: filepath.Dir(cmd.ProjectPath),
        }
    }
    // ... existing validation
}
```

**Contract**:
- Solution files must be rejected before any processing
- Error message must match dotnet CLI exactly
- Exit code must be 1 (standard error exit)
- No partial operations performed

**Error Message Format**:
```
error: Couldn't find a project to run. Ensure a project exists in /path/to/solution, or pass the path to the project using --project
```

## Package Remove Command Enhancement

### Current Behavior
```go
// gonuget package remove <PACKAGE> [PROJECT]
type PackageRemoveCommand struct {
    PackageID   string
    ProjectPath string
    // ... other fields
}
```

### Enhanced Behavior
```go
// Detect and reject solution files with specific error
func (cmd *PackageRemoveCommand) ValidateInput() error {
    if IsSolutionFile(cmd.ProjectPath) {
        return &InvalidProjectFileError{
            Message: "Missing or invalid project file: {path}",
            Path: cmd.ProjectPath,
        }
    }
    // ... existing validation
}
```

**Contract**:
- Solution files must be rejected before any processing
- Error message must match dotnet CLI exactly
- Exit code must be 1 (standard error exit)
- No modifications to any files

**Error Message Format**:
```
error: Missing or invalid project file: /path/to/MySolution.sln
```

## Error Message Constants

```go
package commands

// Error messages matching dotnet CLI exactly
const (
    // Package add error for solution files
    ErrNoProjectForAdd = "Couldn't find a project to run. Ensure a project exists in %s, or pass the path to the project using --project"

    // Package remove error for solution files
    ErrInvalidProjectFile = "Missing or invalid project file: %s"

    // Generic solution operation not supported
    ErrSolutionNotSupported = "Operation not supported for solution files"
)
```

## Helper Functions

```go
package commands

import "github.com/gonuget/cmd/gonuget/solution"

// IsSolutionFile checks if path is a solution file
func IsSolutionFile(path string) bool {
    detector := solution.NewDetector()
    return detector.IsSolutionFile(path)
}

// IsProjectFile checks if path is a project file
func IsProjectFile(path string) bool {
    detector := solution.NewDetector()
    return detector.IsProjectFile(path)
}

// ParseSolutionProjects extracts all projects from a solution
func ParseSolutionProjects(solutionPath string) ([]string, error) {
    parser, err := solution.GetParser(solutionPath)
    if err != nil {
        return nil, err
    }

    sol, err := parser.Parse(solutionPath)
    if err != nil {
        return nil, err
    }

    var projectPaths []string
    for _, project := range sol.Projects {
        if project.IsNETProject {
            projectPaths = append(projectPaths, project.Path)
        }
    }

    return projectPaths, nil
}
```

## Testing Requirements

### Command Tests

1. **Package List with Solution**:
   - Verify multi-project output format
   - Test with missing projects (silent skip)
   - Cross-platform path handling
   - All three solution formats

2. **Package Add with Solution**:
   - Verify exact error message
   - Confirm exit code = 1
   - Ensure no files modified
   - Test all solution formats

3. **Package Remove with Solution**:
   - Verify exact error message
   - Confirm exit code = 1
   - Ensure no files modified
   - Test all solution formats

### Integration Tests

Compare with actual dotnet CLI:
```bash
# Capture dotnet output
dotnet list MySolution.sln package > expected.txt

# Run gonuget
gonuget package list MySolution.sln > actual.txt

# Compare byte-for-byte
diff expected.txt actual.txt
```

## Performance Requirements

- Detection (IsSolutionFile): < 1ms
- Solution parsing: < 1s for 10 projects
- Package list with solution: < 3s for 10 projects
- Error detection: Immediate (< 10ms)