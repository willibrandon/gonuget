# Data Model: Solution File Support

**Feature**: Solution File Support for gonuget CLI
**Date**: 2025-11-01

## Core Entities

### 1. Solution

Represents a parsed solution file containing multiple projects.

**Fields**:
- `FilePath` (string): Absolute path to the solution file
- `Format` (enum): File format - SLN, SLNX, or SLNF
- `FormatVersion` (string): MSBuild format version (e.g., "12.00" for .sln)
- `VisualStudioVersion` (string): Visual Studio version identifier
- `Projects` ([]Project): List of projects in the solution (excludes solution folders)
- `SolutionDir` (string): Directory containing the solution file

**Validation Rules**:
- FilePath must exist and be readable
- Format must be one of the supported types
- Projects list can be empty (valid empty solution)
- SolutionDir must be a valid directory path

**Relationships**:
- Contains multiple Project entities
- Referenced by SolutionFilter (for .slnf files)

### 2. Project

Represents a project reference within a solution.

**Fields**:
- `Name` (string): Project display name
- `Path` (string): Absolute path to the project file
- `RelativePath` (string): Original relative path from solution file
- `GUID` (string): Project's unique identifier
- `TypeGUID` (string): Project type identifier
- `IsNETProject` (bool): Whether project supports NuGet packages

**Validation Rules**:
- Name must not be empty
- Path must be a valid file path (may not exist)
- GUID must be valid GUID format
- TypeGUID must be valid GUID format

**Relationships**:
- Belongs to a Solution
- May reference PackageReferences (via existing project parser)

### 3. SolutionFolder

Virtual organizational container in solution (not a real project).

**Fields**:
- `Name` (string): Folder display name
- `GUID` (string): Folder's unique identifier
- `Items` ([]string): GUIDs of contained projects/folders

**Validation Rules**:
- TypeGUID must equal {2150E333-8FDC-42A3-9474-1A3956D46DE8}
- Not included in Projects list
- Used only for organizational hierarchy

**Relationships**:
- Can contain Projects or other SolutionFolders
- Excluded from package operations

### 4. SolutionFilter

Represents a .slnf file that filters projects from a parent solution.

**Fields**:
- `Solution` (string): Path to parent .sln file
- `Projects` ([]string): List of project paths to include

**Validation Rules**:
- Solution path must reference existing .sln file
- Project paths must exist in parent solution
- Can be empty (no projects selected)

**Relationships**:
- References a parent Solution
- Filters the Projects list from parent

### 5. ParseError

Represents errors during solution file parsing.

**Fields**:
- `FilePath` (string): Path to file being parsed
- `Line` (int): Line number where error occurred (0 if N/A)
- `Column` (int): Column number where error occurred (0 if N/A)
- `Message` (string): Error description
- `ErrorCode` (string): Optional error code for specific failures

**Validation Rules**:
- FilePath must not be empty
- Message must not be empty
- Line/Column are optional (0 means not applicable)

**Relationships**:
- Returned by parser operations
- No persistence needed

### 6. ProjectType

Enumeration of known project type GUIDs.

**Values**:
- `CSharpProject`: {FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}
- `VBNetProject`: {F184B08F-C81C-45F6-A57F-5ABD9991F28F}
- `FSharpProject`: {F2A71F9B-5D33-465A-A702-920D77279786}
- `CppProject`: {8BC9CEB8-8B4A-11D0-8D11-00A0C91BC942}
- `SolutionFolder`: {2150E333-8FDC-42A3-9474-1A3956D46DE8}
- `SharedProject`: {D954291E-2A0B-460D-934E-DC6B0785DB48}

**Usage**:
- Identifies project capabilities
- Determines NuGet package support
- Filters non-.NET projects

## State Transitions

### Solution Loading States

```
NotLoaded -> Parsing -> Loaded
         |-> ParseError
```

### Project Discovery Flow

```
SolutionFile -> Detect Format -> Parse Headers
             -> Extract Projects -> Filter by Type
             -> Resolve Paths -> Return Projects
```

## Operations

### Read Operations

1. **ParseSolution(path string) -> (Solution, error)**
   - Detects format
   - Parses file
   - Returns Solution with all projects

2. **ListProjects(solution Solution) -> []Project**
   - Returns only .NET projects
   - Excludes solution folders
   - Maintains solution order

3. **GetProjectType(typeGUID string) -> ProjectType**
   - Maps GUID to known type
   - Returns Unknown for unrecognized GUIDs

### Validation Operations

1. **ValidateSolutionFile(path string) -> error**
   - Checks file exists
   - Verifies format supported
   - Basic structure validation

2. **IsNETProject(project Project) -> bool**
   - Checks TypeGUID against known .NET types
   - Returns true for C#, VB.NET, F#
   - Returns false for C++, folders, etc.

### Path Operations

1. **ResolvePath(solutionDir, projectPath string) -> string**
   - Converts relative to absolute
   - Handles cross-platform separators
   - Normalizes path format

2. **NormalizePath(path string) -> string**
   - Converts backslashes to forward slashes
   - Removes redundant separators
   - Handles UNC paths on Windows

## Constraints

### Performance Constraints

- Solution parsing must complete in < 1 second for 10 projects
- Memory usage should scale linearly with project count
- No caching required (one-time operations)

### Data Constraints

- Maximum solution file size: 10 MB (reasonable for 1000+ projects)
- Maximum path length: OS-dependent (260 chars on Windows)
- GUID format: Standard 8-4-4-4-12 format with hyphens

### Compatibility Constraints

- Support MSBuild format version 11.00+ (.sln)
- Support .NET 9 XML format (.slnx)
- Support JSON solution filters (.slnf)
- UTF-8 encoding with or without BOM

## Error Handling

### Parse Errors

- **Malformed Structure**: Clear error with line number
- **Invalid GUID Format**: Report specific GUID and location
- **Missing Closing Tags**: Report expected vs found
- **Encoding Issues**: Attempt UTF-8, fall back with error

### File System Errors

- **Solution Not Found**: Standard file not found error
- **Permission Denied**: Report file path and permission issue
- **Path Too Long**: Platform-specific path length error

### Validation Errors

- **Unsupported Format**: List supported formats in error
- **Version Too Old**: Specify minimum version required
- **Corrupted File**: Generic parse error with details

## Security Considerations

- **Path Traversal**: Validate all paths resolve within solution directory or are absolute
- **File Size Limits**: Reject files > 10 MB to prevent DoS
- **No Code Execution**: Never execute scripts or code from solution files
- **Read-Only Access**: Never modify solution files