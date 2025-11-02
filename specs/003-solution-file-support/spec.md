# Feature Specification: Solution File Support for gonuget CLI

**Feature Branch**: `003-solution-file-support`
**Created**: 2025-11-01
**Status**: Draft
**Input**: User description: "Solution File Support for gonuget CLI"

## Clarifications

### Session 2025-11-01

- Q: How should warnings and non-critical issues be output during solution file processing? → A: Output warnings to stderr with visual formatting (yellow "Warning:" prefix)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - List Packages from Solution Files (Priority: P1)

As a developer maintaining a solution with multiple projects, I want to see all package dependencies across all projects at once, so I can identify duplicate dependencies and version inconsistencies.

**Why this priority**: This is the most common operation developers perform with solution files. It provides immediate value by showing a complete dependency overview, helping developers identify version conflicts and redundant packages across their entire solution.

**Independent Test**: Can be fully tested by running `gonuget package list MySolution.sln` on a test solution file and verifying that all projects' packages are listed with correct formatting matching dotnet CLI output exactly.

**Acceptance Scenarios**:

1. **Given** a solution file with 3 projects each having different packages, **When** a user runs `gonuget package list MySolution.sln`, **Then** the tool displays all packages from all projects with project names, framework versions, and package tables formatted exactly like dotnet CLI output
2. **Given** a solution with both .NET and C++ projects, **When** listing packages, **Then** only .NET project packages are shown (C++ projects are skipped)
3. **Given** a solution file referencing a missing project file, **When** listing packages, **Then** the tool silently skips the missing project and continues processing other projects (matching dotnet behavior)

---

### User Story 2 - Reject Solution Files for Package Add (Priority: P2)

As a developer who occasionally tries commands that don't work, I want clear error messages that match what I'm used to from the dotnet CLI when I accidentally try to add packages to a solution file instead of a project file.

**Why this priority**: Error handling is critical for user experience. Developers expect consistent behavior with dotnet CLI, and clear error messages prevent confusion and wasted time.

**Independent Test**: Can be tested by running `gonuget package add MySolution.sln Newtonsoft.Json` and verifying the exact error message matches dotnet CLI's message format.

**Acceptance Scenarios**:

1. **Given** a valid solution file, **When** a user runs `gonuget package add MySolution.sln Newtonsoft.Json`, **Then** the tool returns error "Couldn't find a project to run. Ensure a project exists in [directory], or pass the path to the project using --project"
2. **Given** any of the three solution file formats (.sln, .slnx, .slnf), **When** attempting package add, **Then** the same error message is shown consistently

---

### User Story 3 - Reject Solution Files for Package Remove (Priority: P2)

As a developer trying to remove a package, if I accidentally specify a solution file instead of a project file, I want a clear error message that matches the dotnet CLI behavior.

**Why this priority**: Consistent error handling across all package operations ensures developers can quickly understand and correct their mistakes without consulting documentation.

**Independent Test**: Can be tested by running `gonuget package remove MySolution.sln Moq` and verifying the error message exactly matches dotnet CLI output.

**Acceptance Scenarios**:

1. **Given** a valid solution file, **When** a user runs `gonuget package remove MySolution.sln Moq`, **Then** the tool returns error "Missing or invalid project file: [path]"
2. **Given** any solution file format, **When** attempting package remove, **Then** no changes are made to any project files

---

### User Story 4 - Cross-Platform Path Handling (Priority: P3)

As a developer working in mixed Windows/Linux/Mac teams, I want solution files created on Windows (with backslash paths) to work correctly on Unix systems and vice versa.

**Why this priority**: Cross-platform support is essential for team collaboration, but it's a quality-of-life improvement rather than core functionality.

**Independent Test**: Can be tested by using a Windows-created solution file with backslash paths on Linux/Mac and verifying all projects are found and processed correctly.

**Acceptance Scenarios**:

1. **Given** a solution file with Windows-style paths (backslashes), **When** processed on Linux/Mac, **Then** all project files are found correctly
2. **Given** a solution file with relative paths, **When** processed from different directories, **Then** paths are resolved correctly relative to the solution file location

---

### Edge Cases

- What happens when a solution file contains 100+ projects? (System completes within 30 seconds with linear memory scaling)
- How does system handle malformed solution files? (Parse error with clear message indicating the problem)
- What happens with circular references in solution filters? (Not our concern - MSBuild handles this)
- How are unrecognized project type GUIDs handled? (Include the project anyway - might be new project type)
- What happens when solution folders are nested? (Correctly exclude all solution folders regardless of nesting)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST detect solution files based on extension (.sln, .slnx, .slnf) using case-insensitive matching
- **FR-002**: System MUST parse .sln files following MSBuild format specification (version 11.00+)
- **FR-003**: System MUST parse .slnx files (XML-based format introduced in .NET 9)
- **FR-004**: System MUST parse .slnf files (JSON-based solution filter format)
- **FR-005**: System MUST exclude solution folders (GUID {2150E333-8FDC-42A3-9474-1A3956D46DE8}) from project enumeration
- **FR-006**: System MUST list packages from all .NET projects (.csproj, .fsproj, .vbproj) in a solution
- **FR-007**: System MUST format package list output to match dotnet CLI exactly (character-for-character)
- **FR-008**: System MUST reject solution files for `package add` command with specific error message
- **FR-009**: System MUST reject solution files for `package remove` command with specific error message
- **FR-010**: System MUST handle missing project files gracefully (silently skip for package list, report errors for restore/build operations)
- **FR-011**: System MUST convert Windows backslash paths to work on Unix systems
- **FR-012**: System MUST process projects in the order they appear in the solution file
- **FR-013**: System MUST support UTF-8 encoding (with or without BOM)
- **FR-014**: System MUST complete solution parsing with 10 projects in under 1 second
- **FR-015**: System MUST distinguish solution files from project files (files ending in "proj")
- **FR-016**: System MUST output warnings to stderr with "Warning:" prefix and visual formatting when terminal supports it

### Key Entities *(include if feature involves data)*

- **Solution File**: A configuration file that groups multiple projects (.sln, .slnx, or .slnf format)
- **Project Reference**: A pointer to a project file within a solution (contains name, path, GUID, type GUID)
- **Solution Folder**: Virtual organizational container (not a real project, identified by specific GUID)
- **Package List Output**: Formatted text showing packages per project with framework versions

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Package list output from solution files matches dotnet CLI output character-for-character 100% of the time
- **SC-002**: Error messages for invalid operations match dotnet CLI text exactly 100% of the time
- **SC-003**: Solution files with 10 projects parse in under 1 second
- **SC-004**: Solution files with 100+ projects complete processing in under 30 seconds
- **SC-005**: Cross-platform path resolution succeeds 100% of the time (Windows paths work on Unix, Unix paths work on Windows)
- **SC-006**: All three solution file formats (.sln, .slnx, .slnf) are successfully parsed and processed

## Dependencies & Assumptions *(optional)*

### Dependencies

- Existing gonuget package list functionality for individual projects
- Existing gonuget package add/remove commands for error handling integration

### Assumptions

- Solution files follow MSBuild format specification (Visual Studio 2022 format version 12.00 and later)
- Projects are displayed in the order they appear in the solution file (matches dotnet behavior)
- If one project fails to load, processing continues with other projects (matches dotnet behavior)
- Solution files use UTF-8 encoding (with or without BOM)
- Memory usage scales linearly with project count
- Shared projects (.shproj) don't have packages and are skipped

## Security & Privacy *(optional)*

### Security Considerations

- Solution files are treated as read-only - never modified by gonuget
- Path traversal attacks prevented by resolving all paths relative to solution file directory
- No execution of scripts or code from solution files
- File access limited to reading solution and project files only

## Open Questions *(optional)*

*All requirements have been clearly specified based on dotnet CLI parity requirements. No open questions remain.*