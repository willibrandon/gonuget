# Contract: Solution Parser Interface

**Feature**: Solution File Support for gonuget CLI
**Date**: 2025-11-01

## Interface Definition

```go
// Parser defines the interface for parsing solution files
type Parser interface {
    // Parse reads and parses a solution file
    // Returns a Solution structure with all projects
    // Returns ParseError if file is malformed or cannot be read
    Parse(path string) (*Solution, error)

    // CanParse checks if this parser supports the given file
    // Uses extension-based detection for performance
    // Does not perform file I/O
    CanParse(path string) bool
}
```

## Concrete Implementations

### SlnParser

Parses text-based .sln files (MSBuild format).

**Contract**:
- Must support format version 11.00+ (Visual Studio 2010+)
- Must handle UTF-8 encoding with or without BOM
- Must extract all Project entries
- Must exclude SolutionFolder entries (GUID {2150E333-8FDC-42A3-9474-1A3956D46DE8})
- Must preserve project order from file

**Error Cases**:
- Malformed project lines return ParseError with line number
- Invalid GUIDs return ParseError with specific GUID
- Missing EndProject tags return ParseError
- Unsupported format version returns error

### SlnxParser

Parses XML-based .slnx files (introduced in .NET 9).

**Contract**:
- Must use standard XML parsing
- Must handle XML namespaces correctly
- Must validate against expected schema
- Must extract Project elements
- Must handle missing optional elements gracefully

**Error Cases**:
- Malformed XML returns parse error with position
- Missing required elements return specific error
- Invalid schema returns validation error

### SlnfParser

Parses JSON-based .slnf solution filter files.

**Contract**:
- Must parse standard JSON format
- Must resolve parent solution file
- Must apply project filtering to parent solution
- Must handle relative paths in filter
- Must validate filtered projects exist in parent

**Error Cases**:
- Invalid JSON returns parse error with position
- Missing parent solution returns specific error
- Invalid project references return list of invalid projects

## Parser Factory

```go
// GetParser returns the appropriate parser for a solution file
func GetParser(path string) (Parser, error)
```

**Contract**:
- Returns SlnParser for .sln files
- Returns SlnxParser for .slnx files
- Returns SlnfParser for .slnf files
- Returns error for unsupported extensions
- Case-insensitive extension matching

## Usage Pattern

```go
// 1. Get appropriate parser
parser, err := GetParser("MySolution.sln")
if err != nil {
    return err
}

// 2. Parse the solution file
solution, err := parser.Parse("MySolution.sln")
if err != nil {
    if parseErr, ok := err.(*ParseError); ok {
        // Handle parse error with line/column info
        fmt.Printf("Parse error at line %d: %s\n",
                   parseErr.Line, parseErr.Message)
    }
    return err
}

// 3. Process projects
for _, project := range solution.Projects {
    if project.IsNETProject {
        // Process .NET project
    }
}
```

## Testing Requirements

### Unit Tests

Each parser implementation must have tests for:
- Valid file parsing
- Malformed file handling
- Edge cases (empty, very large)
- Character encoding handling
- Cross-platform path resolution

### Integration Tests

- Parser factory selection
- End-to-end parsing with real files
- Error message format validation
- Performance benchmarks

### Contract Tests

Verify all parsers conform to interface:
- Return consistent Solution structure
- Error types are consistent
- CanParse behaves identically
- Null/empty handling matches

## Performance Requirements

- CanParse: < 1ms (no file I/O)
- Parse 10 projects: < 1 second
- Parse 100 projects: < 10 seconds
- Parse 1000 projects: < 30 seconds
- Memory usage: O(n) where n = number of projects

## Thread Safety

- Parsers must be stateless
- Safe for concurrent use
- No shared mutable state
- Each Parse call independent

## Extensibility

Interface allows future formats:
- Could add .slnk parser for Kubernetes solutions
- Could add .sln2 for next-gen format
- Could add custom format parsers
- No breaking changes to interface needed