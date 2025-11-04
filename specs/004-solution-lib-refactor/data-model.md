# Data Model: Solution File Parsing Library Refactor

**Feature**: 004-solution-lib-refactor
**Date**: 2025-11-03
**Status**: Complete

## Overview

This document describes the data model for the solution file parsing library. Since this is a **pure refactoring task**, the data model is **not changing** - we are simply documenting the existing model that will be relocated from `cmd/gonuget/solution` to `solution/`.

## Model Stability

**IMPORTANT**: This refactor does **NOT** modify any data structures, field names, types, or relationships. All entities below are copied verbatim from the existing implementation.

---

## Core Entities

### Solution

**Purpose**: Represents a parsed .NET solution file (.sln, .slnx, or .slnf)

**Fields**:
- `FilePath` (string): Absolute path to the solution file
- `FormatVersion` (string): Solution file format version (e.g., "12.00" for VS 2013+)
- `VisualStudioVersion` (string): Visual Studio version that created the file
- `MinimumVisualStudioVersion` (string): Minimum VS version required to open the solution
- `Projects` ([]Project): All projects in the solution (excludes solution folders)
- `SolutionFolders` ([]SolutionFolder): Virtual folders for organizing projects
- `SolutionDir` (string): Directory containing the solution file

**Methods**:
- `GetProjects() []string`: Returns absolute paths to all .NET projects (filters by `IsNETProject()`)
- `GetProjectByName(name string) (*Project, bool)`: Finds project by display name (case-insensitive)
- `GetProjectByPath(path string) (*Project, bool)`: Finds project by file path (supports relative/absolute)

**Validation Rules**:
- `FilePath` must be absolute path to existing file
- `Projects` must not contain solution folders (filtered by GUID)
- Project paths must be resolvable relative to `SolutionDir`

**State Transitions**: None (immutable after parsing)

---

### Project

**Purpose**: Represents a project reference within a solution

**Fields**:
- `Name` (string): Display name of the project
- `Path` (string): File system path to the project file (relative or absolute)
- `GUID` (string): Unique identifier for this project instance in the solution
- `TypeGUID` (string): Project type identifier (C#, VB.NET, F#, etc.)
- `ParentFolderGUID` (string): GUID of the containing solution folder (if nested)

**Methods**:
- `IsNETProject() bool`: Returns true if project is a .NET project (C#/VB/F# based on TypeGUID)
- `IsProjectFile() bool`: Returns true if path has .csproj/.vbproj/.fsproj extension
- `GetAbsolutePath(solutionDir string) string`: Resolves project path to absolute path

**Validation Rules**:
- `GUID` must be unique within solution
- `TypeGUID` must be a valid project type GUID (see constants below)
- `Path` extension must match `TypeGUID` for .NET projects

**Relationships**:
- Belongs to one `Solution`
- May be contained in zero or one `SolutionFolder` (via `ParentFolderGUID`)

---

### SolutionFolder

**Purpose**: Represents a virtual folder in the solution (used for organization, not physical directories)

**Fields**:
- `Name` (string): Display name of the folder
- `GUID` (string): Unique identifier for this folder
- `ParentFolderGUID` (string): GUID of parent folder for nested folders (empty if root-level)
- `Items` ([]string): File references in SolutionItems folders (e.g., README, config files)

**Validation Rules**:
- `GUID` must equal `ProjectTypeSolutionFolder` constant (`{2150E333-8FDC-42A3-9474-1A3956D46DE8}`)
- `ParentFolderGUID` must reference an existing folder or be empty
- Folders can be nested arbitrarily deep

**Relationships**:
- Belongs to one `Solution`
- Can contain zero or more `Project` entries (via their `ParentFolderGUID`)
- Can contain zero or more child `SolutionFolder` entries (via `ParentFolderGUID`)

---

### SolutionFilter

**Purpose**: Represents a .slnf filter file (subset of projects from a parent .sln)

**Fields**:
- `SolutionPath` (string): Path to the parent .sln file
- `Projects` ([]string): List of project paths to include in the filter

**Validation Rules**:
- `SolutionPath` must reference an existing .sln file
- `Projects` must be valid paths that exist in the parent solution
- Filter files use JSON format

**Relationships**:
- References one parent `Solution` (via `SolutionPath`)
- Contains subset of `Project` references from parent solution

---

### ParseError

**Purpose**: Represents an error during solution file parsing with location context

**Fields**:
- `FilePath` (string): Path to the file being parsed
- `Line` (int): Line number where error occurred (0 if not line-specific)
- `Column` (int): Column number where error occurred (0 if not column-specific)
- `Message` (string): Human-readable error description

**Methods**:
- `Error() string`: Implements error interface, formats as "file:line:col: message"

**Usage**: Returned by parser functions when syntax errors or invalid data detected

---

## Constants (Project Type GUIDs)

```go
const (
    ProjectTypeCSProject        = "{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}" // Classic C#
    ProjectTypeCSProjectSDK     = "{9A19103F-16F7-4668-BE54-9A1E7A4F7556}" // SDK-style C#
    ProjectTypeVBProject        = "{F184B08F-C81C-45F6-A57F-5ABD9991F28F}" // VB.NET
    ProjectTypeFSProject        = "{F2A71F9B-5D33-465A-A702-920D77279786}" // F#
    ProjectTypeSolutionFolder   = "{2150E333-8FDC-42A3-9474-1A3956D46DE8}" // Virtual folder
    ProjectTypeSharedProject    = "{D954291E-2A0B-460D-934E-DC6B0785DB48}" // Shared project
    ProjectTypeWebSite          = "{E24C65DC-7377-472B-9ABA-BC803B73C61A}" // Website
)
```

---

## Supporting Types

### Detector

**Purpose**: Searches directories for solution files and resolves ambiguities

**Fields**:
- `SearchDir` (string): Directory to search for solution files

**Methods**:
- `DetectSolution() (*DetectionResult, error)`: Searches for .sln/.slnx/.slnf files

**Behavior**:
- Skips hidden directories (`.git`, `.vscode`, etc.)
- Skips build output directories (`bin`, `obj`, `node_modules`)
- Returns ambiguous result if multiple solution files found

---

### DetectionResult

**Purpose**: Contains results of solution file auto-detection

**Fields**:
- `Found` (bool): True if at least one solution file found
- `Ambiguous` (bool): True if multiple solution files found
- `SolutionPath` (string): Path to the found solution file (if unambiguous)
- `FoundFiles` ([]string): All solution files found during search
- `Format` (string): Detected format ("sln", "slnx", or "slnf")

**State Logic**:
- 0 files found: `Found=false, Ambiguous=false`
- 1 file found: `Found=true, Ambiguous=false, SolutionPath=<path>`
- 2+ files found: `Found=true, Ambiguous=true, FoundFiles=[...]`

---

## Parser Interface

### Parser

**Purpose**: Abstract interface for parsing different solution file formats

**Methods**:
- `Parse(filePath string) (*Solution, error)`: Parses a solution file and returns structured data

**Implementations**:
- **SlnParser**: Parses text-based .sln files (VS 7.0-12.0 format)
- **SlnxParser**: Parses XML-based .slnx files (modern format)
- **SlnfParser**: Parses JSON-based .slnf filter files

**Factory Function**:
- `GetParser(filePath string) (Parser, error)`: Returns appropriate parser based on file extension

---

## File Format Notes

### .sln Format (Text-based)
- Line-oriented format with UTF-8/UTF-8 BOM support
- Uses GUID-based project references
- Solution folders have special GUID: `{2150E333-8FDC-42A3-9474-1A3956D46DE8}`
- Paths use backslashes (Windows-style), normalized to forward slashes on read

### .slnx Format (XML-based)
- Modern XML format introduced in recent Visual Studio versions
- Nested folder structure (folders contain projects as child elements)
- UTF-8 encoding required
- Flattened into same `Solution` structure as .sln for consistency

### .slnf Format (JSON-based)
- Filter file references parent .sln file
- Contains subset of project paths
- JSON structure with `solution.path` and `solution.projects[]` keys

---

## Entity Relationship Diagram

```
┌─────────────────┐
│    Solution     │
│─────────────────│
│ FilePath        │
│ FormatVersion   │
│ Projects[]      │◄─────┐
│ SolutionFolders│       │
│ SolutionDir     │       │
└─────────────────┘       │
                          │
         ┌────────────────┴────────────────┐
         │                                 │
    ┌────▼──────┐              ┌──────────▼───────┐
    │  Project  │              │ SolutionFolder   │
    │───────────│              │──────────────────│
    │ Name      │              │ Name             │
    │ Path      │              │ GUID             │
    │ GUID      │              │ ParentFolderGUID │
    │ TypeGUID  │              │ Items[]          │
    │ ParentFGUID──────────────►                  │
    └───────────┘              └──────────────────┘

    ┌───────────────┐
    │ SolutionFilter│
    │───────────────│
    │ SolutionPath  │──────► references .sln file
    │ Projects[]    │
    └───────────────┘
```

---

## Data Model Validation

### Pre-Refactor Validation
- [x] All entities documented match existing `cmd/gonuget/solution/types.go`
- [x] All field types and relationships verified in source code
- [x] Constants and GUIDs confirmed against existing implementation
- [x] Parser interface matches `cmd/gonuget/solution/parser.go`

### Post-Refactor Invariants
- All types must remain in `solution/types.go` (exact copy)
- All method signatures must remain unchanged
- All field names and types must remain unchanged
- All constants must retain exact GUID values
- No new fields, methods, or types added

---

## Summary

This data model document serves as a **reference snapshot** of the existing solution parsing data structures. The refactor task relocates these entities from `cmd/gonuget/solution` to `solution/` **without any modifications**.

**Key Points**:
- ✅ No new entities introduced
- ✅ No fields added/removed/renamed
- ✅ No validation rules changed
- ✅ No relationships modified
- ✅ Complete behavioral preservation

**Verification**: After refactor, run `diff -u cmd/gonuget/solution/types.go solution/types.go` should show **only package declaration line** difference (line 2: `package solution` remains unchanged).

**Status**: ✅ **Phase 1 Data Model Documentation Complete**
