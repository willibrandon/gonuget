# Research: Solution File Support Implementation

**Feature**: Solution File Support for gonuget CLI
**Date**: 2025-11-01

## Key Decisions

### 1. Solution File Parsing Strategy

**Decision**: Text-based scanner with regex for .sln, XML parser for .slnx, JSON parser for .slnf

**Rationale**:
- .sln files use a custom text format that's not standard XML/JSON
- Go's standard library provides robust XML (encoding/xml) and JSON (encoding/json) parsers
- Regex patterns can reliably extract project information from .sln format
- MSBuild's own parser uses similar text scanning approach

**Alternatives Considered**:
- Using MSBuild libraries via interop: Rejected due to .NET dependency requirement
- Generic parser library: Unnecessary complexity for well-defined formats
- Line-by-line state machine: More complex than regex for this use case

### 2. Cross-Platform Path Resolution

**Decision**: Use filepath.ToSlash() for Windows→Unix conversion, filepath.Join() for building paths

**Rationale**:
- Go's filepath package handles platform differences automatically
- ToSlash() converts backslashes to forward slashes reliably
- Works correctly with both absolute and relative paths
- Matches how other Go tools handle cross-platform paths

**Alternatives Considered**:
- Manual string replacement: Error-prone and doesn't handle edge cases
- Platform-specific code paths: Unnecessary complexity
- External path library: Go's standard library is sufficient

### 3. Error Message Formatting

**Decision**: Store exact dotnet CLI error messages as constants

**Rationale**:
- Ensures 100% parity with dotnet CLI output
- Makes it easy to update if dotnet changes messages
- Centralizes all error messages for maintainability
- Allows for easy testing of message format

**Alternatives Considered**:
- Dynamic message generation: Risk of format differences
- Resource files: Overkill for a few error messages
- Embedding in code: Makes maintenance harder

### 4. Solution Format Detection

**Decision**: Extension-based detection with case-insensitive matching

**Rationale**:
- File extensions are standardized (.sln, .slnx, .slnf)
- Fast detection without file I/O
- Matches how dotnet CLI identifies solution files
- Simple and reliable

**Alternatives Considered**:
- Content-based detection: Slower, requires file reading
- Magic bytes: Solution files don't have consistent headers
- Hybrid approach: Unnecessary complexity

### 5. Parser Interface Design

**Decision**: Common Parser interface with format-specific implementations

**Rationale**:
- Allows clean separation of parsing logic
- Easy to add new formats in the future
- Testable in isolation
- Follows Go interface best practices

**Alternatives Considered**:
- Single parser with format switches: Would become complex
- Separate packages per format: Too much overhead
- Factory pattern only: Need interface for polymorphism

### 6. Project Type Filtering

**Decision**: Filter by project type GUID, include only .NET project types

**Rationale**:
- GUIDs are standardized across all Visual Studio versions
- Reliable way to identify project capabilities
- C++ projects ({8BC9CEB8...}) don't support NuGet packages
- Solution folders ({2150E333...}) are not real projects

**Alternatives Considered**:
- File extension filtering: Less reliable, custom project types exist
- Attempting to parse all projects: Would fail on incompatible types
- Content inspection: Too slow and complex

### 7. Missing Project Handling

**Decision**: Silently skip for package list, error for restore/build operations

**Rationale**:
- Matches exact dotnet CLI behavior per user research
- Package list is informational, shouldn't fail
- Restore/build require all projects present
- Provides best user experience

**Alternatives Considered**:
- Always warn: Doesn't match dotnet behavior
- Always error: Too strict for list operations
- Configuration option: Unnecessary complexity

### 8. .slnx Format Support (XML)

**Decision**: Use encoding/xml with struct tags for unmarshaling

**Rationale**:
- .slnx is standard XML format introduced in .NET 9
- Go's XML parser is mature and reliable
- Struct tags provide clean mapping
- Similar to how we handle other XML formats in gonuget

**Alternatives Considered**:
- DOM parsing: More complex for simple structure
- SAX parsing: Overkill for small files
- Third-party XML library: Standard library sufficient

### 9. .slnf Format Support (JSON)

**Decision**: Use encoding/json for parsing solution filters

**Rationale**:
- .slnf files are standard JSON
- Solution filters reference a parent .sln file
- Need to parse filter then load referenced solution
- Apply project filtering based on filter specification

**Alternatives Considered**:
- Treat as separate solution type: Doesn't match actual behavior
- Ignore filters: Would miss user intent
- Manual JSON parsing: Unnecessary when standard library works

### 10. Output Formatting

**Decision**: Reuse existing package list output formatter with multi-project wrapper

**Rationale**:
- Maintains consistency with single-project output
- Existing formatter already matches dotnet CLI
- Just need to add project name headers
- Minimizes code duplication

**Alternatives Considered**:
- New formatter for solutions: Would duplicate logic
- Template-based output: Overkill for this use case
- Direct string building: Harder to maintain

## Implementation Order

1. **Core Types** (types.go): Solution, Project, and error types
2. **Parser Interface** (parser.go): Common interface definition
3. **.sln Parser** (sln_parser.go): Most common format
4. **Path Resolution** (path.go): Cross-platform support
5. **Detector** (detector.go): Format detection
6. **.slnx Parser** (slnx_parser.go): XML format
7. **.slnf Parser** (slnf_parser.go): JSON filter format
8. **Command Integration**: Modify existing commands
9. **Output Formatting**: Multi-project support
10. **Testing**: Integration tests for parity validation

## Performance Considerations

- **Sequential Parsing**: Parse projects one at a time (I/O bound, not CPU bound)
- **Memory Usage**: Stream parsing where possible, don't load entire file for large solutions
- **Caching**: No caching needed (one-time operations)
- **Benchmarking**: Measure against performance targets (< 1s for 10 projects)

## Testing Strategy

1. **Unit Tests**: Each parser tested independently
2. **Integration Tests**: End-to-end command tests
3. **Parity Tests**: Compare output with dotnet CLI byte-for-byte
4. **Cross-Platform Tests**: Windows paths on Unix, Unix paths on Windows
5. **Error Tests**: Verify exact error message matching
6. **Performance Tests**: Validate < 1s for 10 projects target

## References

- MSBuild Solution File Format: https://docs.microsoft.com/en-us/visualstudio/msbuild/solution-dot-sln-file
- Project Type GUIDs: https://www.codeproject.com/Reference/720512/List-of-Visual-Studio-Project-Type-GUIDs
- Solution Filter Format: https://docs.microsoft.com/en-us/visualstudio/ide/filtered-solutions
- .NET 9 .slnx Format: https://devblogs.microsoft.com/dotnet/introducing-dotnet-9/