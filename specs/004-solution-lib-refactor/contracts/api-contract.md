# API Contract: Solution File Parsing Library

**Feature**: 004-solution-lib-refactor
**Date**: 2025-11-03
**Package**: `github.com/willibrandon/gonuget/solution`

## Overview

This document defines the public API contract for the solution file parsing library. Since this is a **pure refactoring task**, the API contract is **not changing** - we are documenting the existing API that will be exposed at the new import path.

---

## API Stability Guarantee

**HARD REQUIREMENT**: All public APIs (exported types, functions, methods, constants) MUST remain **byte-for-byte identical** after refactoring. The only change is the import path:

**Before**: `github.com/willibrandon/gonuget/cmd/gonuget/solution`
**After**: `github.com/willibrandon/gonuget/solution`

---

## Public API Surface

### Types

#### Solution
```go
type Solution struct {
    FilePath                   string
    FormatVersion              string
    VisualStudioVersion        string
    MinimumVisualStudioVersion string
    Projects                   []Project
    SolutionFolders            []SolutionFolder
    SolutionDir                string
}

func (s *Solution) GetProjects() []string
func (s *Solution) GetProjectByName(name string) (*Project, bool)
func (s *Solution) GetProjectByPath(path string) (*Project, bool)
```

#### Project
```go
type Project struct {
    Name             string
    Path             string
    GUID             string
    TypeGUID         string
    ParentFolderGUID string
}

func (p *Project) IsNETProject() bool
func (p *Project) IsProjectFile() bool
func (p *Project) GetAbsolutePath(solutionDir string) string
```

#### SolutionFolder
```go
type SolutionFolder struct {
    Name             string
    GUID             string
    ParentFolderGUID string
    Items            []string
}
```

#### SolutionFilter
```go
type SolutionFilter struct {
    SolutionPath string
    Projects     []string
}
```

#### ParseError
```go
type ParseError struct {
    FilePath string
    Line     int
    Column   int
    Message  string
}

func (e *ParseError) Error() string
```

---

### Constants

```go
const (
    ProjectTypeCSProject      = "{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}"
    ProjectTypeCSProjectSDK   = "{9A19103F-16F7-4668-BE54-9A1E7A4F7556}"
    ProjectTypeVBProject      = "{F184B08F-C81C-45F6-A57F-5ABD9991F28F}"
    ProjectTypeFSProject      = "{F2A71F9B-5D33-465A-A702-920D77279786}"
    ProjectTypeSolutionFolder = "{2150E333-8FDC-42A3-9474-1A3956D46DE8}"
    ProjectTypeSharedProject  = "{D954291E-2A0B-460D-934E-DC6B0785DB48}"
    ProjectTypeWebSite        = "{E24C65DC-7377-472B-9ABA-BC803B73C61A}"
)
```

---

### Detector API

```go
type Detector struct {
    SearchDir string
}

func NewDetector(searchDir string) *Detector

type DetectionResult struct {
    Found        bool
    Ambiguous    bool
    SolutionPath string
    FoundFiles   []string
    Format       string
}

func (d *Detector) DetectSolution() (*DetectionResult, error)
```

---

### Helper Functions

```go
func IsSolutionFile(path string) bool
func IsProjectFile(path string) bool
func GetSolutionFormat(path string) string
func ValidateSolutionFile(path string) error
```

---

### Parser Interface

```go
type Parser interface {
    Parse(filePath string) (*Solution, error)
}

func GetParser(filePath string) (Parser, error)
```

**Note**: Concrete parser implementations (SlnParser, SlnxParser, SlnfParser) are **not exported** - users interact via the `GetParser` factory function.

---

## Usage Examples

### Example 1: Parse a Solution File

```go
package main

import (
    "fmt"
    "log"

    "github.com/willibrandon/gonuget/solution"
)

func main() {
    // Get appropriate parser for file type
    parser, err := solution.GetParser("MyApp.sln")
    if err != nil {
        log.Fatal(err)
    }

    // Parse the solution file
    sol, err := parser.Parse("MyApp.sln")
    if err != nil {
        log.Fatal(err)
    }

    // List all .NET projects
    for _, projectPath := range sol.GetProjects() {
        fmt.Println(projectPath)
    }
}
```

### Example 2: Auto-Detect Solution File

```go
package main

import (
    "fmt"
    "log"

    "github.com/willibrandon/gonuget/solution"
)

func main() {
    detector := solution.NewDetector(".")
    result, err := detector.DetectSolution()
    if err != nil {
        log.Fatal(err)
    }

    if !result.Found {
        log.Fatal("No solution file found")
    }

    if result.Ambiguous {
        fmt.Println("Multiple solution files found:")
        for _, file := range result.FoundFiles {
            fmt.Println("  -", file)
        }
        log.Fatal("Please specify which solution file to use")
    }

    fmt.Printf("Found solution: %s (format: %s)\n",
        result.SolutionPath, result.Format)
}
```

### Example 3: Filter by Project Type

```go
package main

import (
    "fmt"

    "github.com/willibrandon/gonuget/solution"
)

func main() {
    parser, _ := solution.GetParser("MyApp.sln")
    sol, _ := parser.Parse("MyApp.sln")

    // Find all C# projects (SDK-style)
    for _, proj := range sol.Projects {
        if proj.TypeGUID == solution.ProjectTypeCSProjectSDK {
            fmt.Printf("SDK-style C# project: %s\n", proj.Name)
        }
    }
}
```

---

## Error Handling Contract

### ParseError Usage

When parsing fails, implementations return `*ParseError` with location context:

```go
parser, _ := solution.GetParser("invalid.sln")
sol, err := parser.Parse("invalid.sln")
if err != nil {
    if parseErr, ok := err.(*solution.ParseError); ok {
        fmt.Printf("Parse error at %s:%d:%d: %s\n",
            parseErr.FilePath,
            parseErr.Line,
            parseErr.Column,
            parseErr.Message)
    }
}
```

### Validation Errors

Helper functions return standard Go errors:

```go
if err := solution.ValidateSolutionFile("test.txt"); err != nil {
    // Returns: "not a solution file (must have .sln, .slnx, or .slnf extension): test.txt"
}
```

---

## Behavioral Contracts

### Path Handling

1. **Input Paths**: Accept both relative and absolute paths
2. **Output Paths**: Always return absolute paths via `GetAbsolutePath()` or `GetProjects()`
3. **Path Separators**: Normalize backslashes to forward slashes on read

### File Format Support

| Extension | Parser | Format | Notes |
|-----------|--------|--------|-------|
| `.sln` | SlnParser | Text-based | VS 7.0-12.0, UTF-8/UTF-8 BOM |
| `.slnx` | SlnxParser | XML | Modern format, nested folders |
| `.slnf` | SlnfParser | JSON | Filter file, references parent .sln |

### Solution Folder Filtering

- `GetProjects()` returns **only** .NET projects (excludes solution folders)
- Solution folders identified by GUID: `{2150E333-8FDC-42A3-9474-1A3956D46DE8}`
- Projects with `IsNETProject() == false` are excluded from `GetProjects()`

---

## Thread Safety

**All types are immutable after creation** - safe for concurrent read access.

- ✅ Safe: Multiple goroutines reading same `Solution` instance
- ✅ Safe: Multiple goroutines calling `GetParser()` concurrently
- ✅ Safe: Multiple `Detector` instances searching different directories
- ❌ Unsafe: Modifying `Solution` fields after creation (don't do this)

---

## Performance Characteristics

### Parser Performance
- `.sln` parsing: O(n) where n = number of lines
- `.slnx` parsing: O(n) where n = number of XML nodes
- `.slnf` parsing: O(n) where n = number of projects in filter

### Detector Performance
- `DetectSolution()`: O(m) where m = number of files in directory tree
- Skips hidden directories and build output folders (`.git`, `bin`, `obj`, `node_modules`)
- Non-recursive by default (walks from `SearchDir` downward)

---

## API Compatibility Checklist

### Pre-Refactor Validation
- [x] All exported types documented
- [x] All exported functions documented
- [x] All exported constants documented
- [x] All exported methods documented
- [x] Usage examples verified against current implementation

### Post-Refactor Invariants
- All public symbols remain exported
- All method signatures unchanged
- All constant values unchanged
- All godoc comments preserved verbatim
- No new public APIs introduced
- No existing public APIs removed

---

## Migration Guide (for CLI code)

**Old Import**:
```go
import "github.com/willibrandon/gonuget/cmd/gonuget/solution"
```

**New Import**:
```go
import "github.com/willibrandon/gonuget/solution"
```

**Usage**: Identical - all `solution.` prefixed calls work the same way.

---

## Verification

### API Surface Verification
```bash
# Before refactor
go doc -all github.com/willibrandon/gonuget/cmd/gonuget/solution > api-before.txt

# After refactor
go doc -all github.com/willibrandon/gonuget/solution > api-after.txt

# Compare (should differ only in package path line)
diff api-before.txt api-after.txt
```

### Signature Verification
```bash
# Ensure no method signatures changed
go test -run=XXX -v ./solution  # Should compile with no errors
```

---

## Summary

This API contract documents the **existing, stable API** of the solution parsing library. The refactor task preserves this contract **exactly**, changing only the import path. No new APIs are introduced, no existing APIs are removed or modified.

**Contract Guarantee**: Any code that worked with `cmd/gonuget/solution` will work identically with `solution` after updating import paths.

**Status**: ✅ **API Contract Documented - No Changes Required**
