# Data Model: CLI Command Structure Restructure

**Feature**: CLI Command Structure Restructure
**Branch**: `001-cli-command-restructure`
**Date**: 2025-10-31

## Overview

This document defines the data structures and relationships for the CLI command restructure. Since this is a command structure refactoring (not a new feature), the data model focuses on **command metadata** and **validation rules** rather than business entities.

## Core Entities

### 1. Command Hierarchy

Represents the tree structure of CLI commands with parent-child relationships.

**Attributes**:
- `Use` (string): Command name (verb-only for subcommands, noun for parent commands)
- `Short` (string): One-line description starting with verb
- `Long` (string): Multi-paragraph detailed description
- `Example` (string): Usage examples with flags
- `Aliases` ([]string): **MUST BE EMPTY** (zero tolerance policy)
- `Parent` (Command reference): Parent command (nil for top-level)
- `Subcommands` ([]Command): Child commands
- `Flags` ([]Flag): Command-specific flags
- `PersistentFlags` ([]Flag): Flags inherited by subcommands

**Validation Rules**:
- **VR-001**: Parent commands (package, source) MUST have `Use` field as single noun
- **VR-002**: Subcommands MUST have `Use` field as single verb (no spaces)
- **VR-003**: Top-level commands (restore, config, version) MUST have `Use` field as single verb/noun
- **VR-004**: ALL commands MUST have empty `Aliases` field (zero tolerance)
- **VR-005**: `Short` description MUST start with verb (e.g., "Add", "List", "Manage")
- **VR-006**: `Example` MUST include both minimal and fully-flagged forms

**Relationships**:
- Parent command → Many subcommands (1:N)
- Subcommand → One parent (N:1)
- Command → Many flags (1:N)

**State Transitions**: N/A (commands are statically defined)

---

### 2. Command Flag

Represents command-line flags for configuration and behavior modification.

**Attributes**:
- `Name` (string): Flag name in kebab-case (e.g., "config-file", "format")
- `Shorthand` (string): Single-character alias (e.g., "c", "f")
- `Type` (string): Data type (string, bool, int, string-slice)
- `DefaultValue` (interface{}): Default when flag not provided
- `Usage` (string): Description of flag purpose
- `Required` (bool): Whether flag must be provided
- `ValidValues` ([]string): Enum of allowed values (empty if free-form)
- `Persistent` (bool): Whether flag is inherited by subcommands

**Validation Rules**:
- **VR-007**: `Name` MUST use kebab-case (--config-file, NOT --configFile)
- **VR-008**: `Shorthand` MUST be single character or empty
- **VR-009**: Flags with `ValidValues` MUST reject values not in list
- **VR-010**: `--format` flag MUST have ValidValues: ["console", "json"]
- **VR-011**: `--verbosity` flag MUST have ValidValues: ["quiet", "minimal", "normal", "detailed", "diagnostic"]

**Global Flags** (applicable to all commands):
- `--verbosity`: Output detail level
- `--help`: Show help text

**Common Flags** (shared across multiple commands):
- `--format`: Output format (list/search commands)
- `--configfile`: NuGet.config path (source/config/restore commands)
- `--what-if` / `--dry-run`: Preview changes (mutating commands)
- `--yes` / `-y`: Skip confirmations (mutating commands)

---

### 3. Error Pattern

Represents detectable verb-first command patterns for helpful error messages.

**Attributes**:
- `Pattern` (string): Verb-first pattern to detect (e.g., "add package", "list source")
- `Suggestion` (string): Correct noun-first syntax (e.g., "package add", "source list")
- `Verb` (string): The verb portion (add, list, remove, enable, disable, update)
- `Noun` (string): The noun portion (package, source)

**Validation Rules**:
- **VR-012**: Pattern detection MUST be case-insensitive
- **VR-013**: Suggestions MUST include full command path (e.g., "gonuget package add", not "package add")
- **VR-014**: Error message MUST display in <50ms (performance requirement)

**Verb-First Patterns to Detect**:

| Pattern | Correct Syntax | Error Message |
|---------|---------------|---------------|
| `gonuget add package` | `gonuget package add` | "Error: the verb-first form is not supported. Try: gonuget package add" |
| `gonuget list package` | `gonuget package list` | "Error: the verb-first form is not supported. Try: gonuget package list" |
| `gonuget remove package` | `gonuget package remove` | "Error: the verb-first form is not supported. Try: gonuget package remove" |
| `gonuget search package` | `gonuget package search` | "Error: the verb-first form is not supported. Try: gonuget package search" |
| `gonuget add source` | `gonuget source add` | "Error: the verb-first form is not supported. Try: gonuget source add" |
| `gonuget list source` | `gonuget source list` | "Error: the verb-first form is not supported. Try: gonuget source list" |
| `gonuget remove source` | `gonuget source remove` | "Error: the verb-first form is not supported. Try: gonuget source remove" |
| `gonuget enable <name>` | `gonuget source enable` | "Error: the verb-first form is not supported. Try: gonuget source enable" |
| `gonuget disable <name>` | `gonuget source disable` | "Error: the verb-first form is not supported. Try: gonuget source disable" |
| `gonuget update <name>` | `gonuget source update` | "Error: the verb-first form is not supported. Try: gonuget source update" |

---

### 4. JSON Output Schema

Represents the structure of JSON output for list and search commands.

**Package List Schema**:
```json
{
  "schemaVersion": "1.0.0",
  "project": "/path/to/project.csproj",
  "framework": "net8.0",
  "packages": [
    {
      "id": "Newtonsoft.Json",
      "version": "13.0.3",
      "type": "direct",
      "resolvedVersion": "13.0.3"
    }
  ],
  "warnings": [],
  "elapsedMs": 42
}
```

**Package Search Schema**:
```json
{
  "schemaVersion": "1.0.0",
  "searchTerm": "Serilog",
  "sources": ["https://api.nuget.org/v3/index.json"],
  "items": [
    {
      "id": "Serilog",
      "version": "3.1.1",
      "description": "Simple .NET logging with fully-structured events",
      "authors": "Serilog Contributors",
      "totalDownloads": 500000000
    }
  ],
  "total": 147,
  "elapsedMs": 156
}
```

**Validation Rules**:
- **VR-015**: ALL JSON output MUST include `schemaVersion` field
- **VR-016**: `schemaVersion` MUST follow semantic versioning (MAJOR.MINOR.PATCH)
- **VR-017**: Schema version increments:
  - MAJOR: Breaking changes (field removal, type change)
  - MINOR: Additive changes (new fields)
  - PATCH: Documentation/clarification only
- **VR-018**: JSON output MUST go to stdout, warnings/errors to stderr
- **VR-019**: Empty results MUST return valid JSON with empty arrays (not error)

---

### 5. Help Text Template

Represents the standardized structure for command help output.

**Structure**:
```
[Command description]

Usage:
  gonuget [parent] <verb> [arguments] [flags]

Examples:
  # Minimal form
  gonuget package add Newtonsoft.Json

  # With flags
  gonuget package add Serilog --version 3.1.1 --project ./src/MyApp/MyApp.csproj

Available Commands:
  add         Add a package reference
  list        List package references
  remove      Remove a package reference
  search      Search for packages

Flags:
      --configfile string     Path to NuGet.config file
  -h, --help                  help for package
      --verbosity string      Output verbosity (quiet|minimal|normal|detailed|diagnostic)

Use "gonuget [parent] [command] --help" for more information about a command.
```

**Validation Rules**:
- **VR-020**: Help text MUST use consistent formatting (alignment, capitalization)
- **VR-021**: Available Commands MUST be sorted alphabetically
- **VR-022**: Flags MUST show long form (--name) and shorthand (-n) when available
- **VR-023**: Examples MUST demonstrate real-world usage (not placeholder values)

---

## Relationships

### Command Hierarchy Tree

```
root
├── config (top-level)
├── package (parent)
│   ├── add (subcommand)
│   ├── list (subcommand)
│   ├── remove (subcommand)
│   └── search (subcommand)
├── restore (top-level)
├── source (parent)
│   ├── add (subcommand)
│   ├── disable (subcommand)
│   ├── enable (subcommand)
│   ├── list (subcommand)
│   ├── remove (subcommand)
│   └── update (subcommand)
└── version (top-level)
```

**Relationship Rules**:
- Parent commands MUST NOT have `Run` function (they are containers only)
- Subcommands MUST have `Run` function (they execute actual logic)
- Top-level commands MUST be registered directly on root command
- Subcommands MUST be registered via `parent.AddCommand(child)`

---

## Data Flow

### 1. Command Execution Flow

```
User Input → Cobra Parser → Command Lookup → Validation → Execution
    ↓
    ├─ Valid noun-first → Execute command logic
    └─ Invalid verb-first → Error handler → Suggest noun-first
```

### 2. Help Text Generation Flow

```
User runs --help → Cobra Help System → Template Rendering → Stdout
    ↓
    Golden Test captures output → Compare to .golden file
```

### 3. JSON Output Flow

```
Command Logic → Data Collection → Schema Serialization → Stdout (JSON only)
                                                        → Stderr (warnings/errors)
```

---

## Validation Matrix

| Entity | Validation Rule | Enforced By | Test Type |
|--------|----------------|-------------|-----------|
| Command | VR-001 to VR-006 | Reflection test | Unit |
| Flag | VR-007 to VR-011 | Unit tests | Unit |
| Error Pattern | VR-012 to VR-014 | Integration tests | Integration |
| JSON Schema | VR-015 to VR-019 | JSON validation tests | Unit |
| Help Text | VR-020 to VR-023 | Golden tests | Snapshot |

---

## Extension Points

### Future Command Additions

When adding new commands in the future:

1. **Determine namespace**: Does it belong under `package`, `source`, or as top-level?
2. **Create command file**: Follow naming pattern `{parent}_{verb}.go` or `{verb}.go`
3. **Implement validation rules**: VR-001 to VR-006 enforced automatically by reflection tests
4. **Add golden test**: Capture help output in `golden/help_{command}.golden`
5. **Update error patterns**: If command has verb that could be used verb-first

**Example** - Adding `gonuget package update`:
- File: `cmd/gonuget/commands/package_update.go`
- Use field: `"update"` (verb-only)
- Register: `packageCmd.AddCommand(updateCmd)`
- Golden: `tests/cmd/gonuget/commands/golden/help_package_update.golden`
- Error pattern: Detect `gonuget update package` → suggest `gonuget package update`

---

## References

- Feature Specification: [spec.md](spec.md)
- Implementation Plan: [plan.md](plan.md)
- Research Findings: [research.md](research.md)
- Cobra Command Documentation: https://pkg.go.dev/github.com/spf13/cobra#Command
