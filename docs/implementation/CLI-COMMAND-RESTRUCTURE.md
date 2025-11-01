# CLI Command Structure Restructure - Modern CLI Standards

**Status:** Planning
**Goal:** Adopt modern noun-first CLI command structure from day one
**Approach:** Noun-first commands only (no verb-first aliases)

## Current State

```bash
gonuget
├── add [PARENT]
│   ├── source
│   └── package
├── config
├── disable
├── enable
├── list
├── remove
├── restore
└── update
```

**Problems:**
- Mixed verb-first structure (doesn't align with modern CLI standards)
- Partial namespace grouping (package under `add` parent, but source commands flat)
- No clear noun-first namespacing for source operations
- Inconsistent with dotnet's modern noun-first pattern (`dotnet package`, `dotnet nuget source`)

## Target State (Simplified dotnet-style Structure)

### Package Namespace (`gonuget package`)

```bash
gonuget package add <PACKAGE_NAME>
gonuget package list [PROJECT]
gonuget package remove <PACKAGE_NAME>
gonuget package search <SEARCH_TERM>
```

**Matches:** `dotnet package` exactly

### Source Namespace (`gonuget source`)

```bash
gonuget source add <SOURCE_URL>
gonuget source list
gonuget source remove <NAME>
gonuget source enable <NAME>
gonuget source disable <NAME>
gonuget source update <NAME>
```

**Aligns with the semantics of** `dotnet nuget [verb] source` (order differs by design); we use noun-first order (`source add`) for consistency with the package namespace and modern CLI standards.

### Top-Level Commands (Exceptions)

```bash
gonuget config get|set|list
gonuget restore [PROJECT]
gonuget version
```

**Dotnet parity:**
- `gonuget restore` → `dotnet restore` (exact match)
- `gonuget version` → functional equivalent to `dotnet --version`
- `gonuget config` → maps to `dotnet nuget config` (elevated at top level for ergonomics)

**Why top-level exceptions?**
- `restore` is ubiquitous and fast to type (matches dotnet exactly)
- `version` is a universal CLI convention
- `config` is elevated from `dotnet nuget config` for better ergonomics; behavior maps to `dotnet nuget config`, though structurally different

## Design Decision: Noun-First Only (No Aliases)

### Why Noun-First Commands?

Starting in .NET 10, Microsoft introduced noun-first command aliases (`dotnet package add`) alongside the original verb-first commands (`dotnet add package`). Both forms work identically in dotnet CLI.

**gonuget adopts ONLY the noun-first form** for package commands. Here's why:

### Rationale

1. **No Backward Compatibility Burden**: gonuget has not been released yet. There are no existing users or scripts to break, so we can adopt modern standards immediately without maintaining legacy command structures.

2. **Aligns with General CLI Standards**: Noun-first commands are the industry standard used by modern CLI tools:
   - `git branch create` (noun → verb)
   - `kubectl get pods` (noun → verb)
   - `docker container run` (noun → verb)
   - `aws s3 cp` (noun → verb)

3. **Simpler Implementation**: Supporting only one command form means:
   - No alias handling logic
   - Cleaner command registration
   - Less test surface area
   - Easier to maintain and document

4. **Clearer Intent**: Noun-first commands group related operations naturally:
   ```
   gonuget package add     # All package operations
   gonuget package list    # are clearly under the
   gonuget package remove  # 'package' namespace

   gonuget source add      # All source operations
   gonuget source list     # are clearly under the
   gonuget source remove   # 'source' namespace

   gonuget restore         # Exceptions: top-level commands
   gonuget config          # that match dotnet CLI exactly
   ```

5. **Future-Proof**: As Microsoft recommends the noun-first form for new code and documentation, starting with this pattern ensures gonuget stays aligned with evolving .NET CLI conventions.

### Policy: No Hidden/Legacy Aliases

**Aliases:** none
**Deprecated:** none
**Forward compatibility:** Verb-first forms will never be added as aliases
**Breaking change process:** Adding verb-first aliases is out of scope for v1.x

This is a hard policy. There are no hidden backward-compatibility layers, no soft deprecations, and no future plans to support verb-first aliases. The command structure is noun-first from day one.

### What This Means

**Package commands (noun-first)**:
- ✅ **Supported**: `gonuget package add`, `gonuget package list`, `gonuget package remove`, `gonuget package search`
- ❌ **Not Supported**: `gonuget add package`, `gonuget list package`, `gonuget remove package`

**Source commands (noun-first)**:
- ✅ **Supported**: `gonuget source add`, `gonuget source list`, `gonuget source remove`, `gonuget source enable`, `gonuget source disable`, `gonuget source update`
- ❌ **Not Supported**: `gonuget add source`, `gonuget list source`, `gonuget remove source`, etc.

**Top-level commands (exceptions to noun-first pattern)**:
- ✅ **Supported**: `gonuget restore`, `gonuget config`, `gonuget version`
- **Rationale**: These match dotnet CLI exactly (`dotnet restore`, `dotnet --version`) and are not namespaced operations

### Error Hints for Verb-First Attempts

To guide users away from verb-first patterns, implement helpful error suggestions in Cobra:

```go
// In root command setup
rootCmd.SuggestionsMinimumDistance = 1

// In main.go or command setup, add custom error handling:
// If user types: gonuget add package <name>
//   Detect pattern and show: "The verb-first form is not supported. Try: gonuget package add <name>"
// If user types: gonuget list source
//   Detect pattern and show: "The verb-first form is not supported. Try: gonuget source list"
```

**Example error output:**
```
$ gonuget add package Newtonsoft.Json
Error: unknown command "add" for "gonuget"

The verb-first form is not supported.
Try: gonuget package add Newtonsoft.Json

Run 'gonuget --help' for usage.
```

### Microsoft's Guidance

From .NET 10 documentation:
> "The new noun-first forms align with general CLI standards, making the dotnet CLI more consistent with other tools. While the verb-first forms continue to work, **it's better to use the noun-first forms for improved readability and consistency** in scripts and documentation."

gonuget follows this guidance from inception by implementing only the recommended noun-first form.

## Flag Consistency & Standards

### Standard Flags (All Commands)

Apply consistently across all commands where applicable:

| Flag | Type | Default | Usage | Commands |
|------|------|---------|-------|----------|
| `--format` | string | `console` | Output format: `console` or `json` | list, search, config list |
| `--verbosity` | string | `normal` | Verbosity: `quiet`, `minimal`, `normal`, `detailed`, `diagnostic` | all commands |
| `--configfile` | string | auto-detect | Path to NuGet.config file | source commands, config, restore |
| `--what-if` / `--dry-run` | bool | `false` | Show planned changes without executing | mutating commands |
| `--yes` / `-y` | bool | `false` | Suppress interactive confirmations | mutating commands |

### Positional vs Flag Conventions

**Package commands:**
- `package add <PACKAGE_NAME>` - positional package name (required)
- `package list [PROJECT]` - positional project path (optional)
- `package remove <PACKAGE_NAME>` - positional package name (required)
- `package search <SEARCH_TERM>` - positional search term (required)
- Use `--project` flag for mutating ops when not using positional arg

**Source commands:**
- `source add <SOURCE_URL>` - positional URL (required)
- `source remove <NAME>` - positional name (required)
- `source enable <NAME>` - positional name (required)
- `source disable <NAME>` - positional name (required)
- `source update <NAME>` - positional name (required)
- Use `--name` flag consistently (never `--source-name`)

### Verbosity Level Mapping

Maps to dotnet verbosity levels:

| Level | Behavior | Use Case |
|-------|----------|----------|
| `quiet` | Errors only | CI/CD scripts |
| `minimal` | Errors + warnings | Default for scripts |
| `normal` | Info + errors + warnings | Default interactive |
| `detailed` | Verbose operation details | Troubleshooting |
| `diagnostic` | Full diagnostic output + timestamps + operation IDs | Deep debugging |

### JSON Output Contract

When `--format json` is specified:

**Requirements:**
- Never print non-JSON to stdout (send all human text to stderr)
- Always include `schemaVersion` field
- Use stable, documented JSON shapes
- Return 0 for empty results (e.g., search with no matches)

**Standard JSON structure:**
```json
{
  "schemaVersion": "1.0",
  "items": [],
  "total": 0,
  "warnings": [],
  "errors": [],
  "elapsedMs": 123
}
```

**Package list JSON:**
```json
{
  "schemaVersion": "1.0",
  "project": "/path/to/project.csproj",
  "framework": "net8.0",
  "packages": [
    {
      "id": "Newtonsoft.Json",
      "requestedVersion": "13.0.3",
      "resolvedVersion": "13.0.3",
      "type": "direct",
      "dependencies": []
    }
  ],
  "warnings": [],
  "elapsedMs": 45
}
```

**Package search JSON:**
```json
{
  "schemaVersion": "1.0",
  "searchTerm": "Serilog",
  "sources": ["https://api.nuget.org/v3/index.json"],
  "items": [
    {
      "id": "Serilog",
      "latestVersion": "3.1.1",
      "description": "Simple .NET logging with fully-structured events",
      "downloadCount": 500000000,
      "authors": ["Serilog Contributors"],
      "projectUrl": "https://serilog.net/",
      "tags": ["logging", "serilog"]
    }
  ],
  "total": 1,
  "elapsedMs": 234
}
```

## Exit Codes

Standardized exit codes for consistent error handling:

| Code | Meaning | Example |
|------|---------|---------|
| `0` | Success | Command completed without errors |
| `1` | Generic error | Unexpected failure, internal error |
| `2` | Invalid arguments | Unknown command, invalid flags, parse errors |
| `3` | Not found | Package not referenced, source doesn't exist |
| `4` | Network/temporary failure | Source unreachable, timeout, transient error |

**Usage in code:**
```go
const (
    ExitSuccess         = 0
    ExitGenericError    = 1
    ExitInvalidArgs     = 2
    ExitNotFound        = 3
    ExitNetworkFailure  = 4
)
```

**Test coverage:** Each exit code must have dedicated test cases.

## Configuration Precedence

Configuration values are resolved in this order (highest to lowest precedence):

1. **CLI flags** - `--configfile /path/to/nuget.config`
2. **Environment variables** - `GONUGET_*` prefixed vars
3. **NuGet.config files** - Hierarchical merge (project → user → machine)
4. **Defaults** - Built-in defaults

### Environment Variables

| Variable | Type | Purpose | Dotnet Equivalent |
|----------|------|---------|-------------------|
| `GONUGET_SOURCES` | string | Semicolon-separated source URLs | N/A |
| `GONUGET_NUGET_CONFIG_PATH` | string | Path to NuGet.config | N/A |
| `GONUGET_VERBOSITY` | string | Default verbosity level | N/A |
| `NUGET_PACKAGES` | string | Global packages folder (read-only parity) | `NUGET_PACKAGES` |

**Note:** Read `NUGET_PACKAGES` for parity, but prefix gonuget-specific vars with `GONUGET_`.

## Shell Completion

Provide dynamic completions for nouns, verbs, and context-aware arguments:

**Namespace completion:**
```bash
$ gonuget <TAB>
config   package   restore   source   version

$ gonuget package <TAB>
add   list   remove   search
```

**Dynamic completions:**
- Package IDs: Complete from configured sources (cache recent searches)
- Source names: Complete from NuGet.config
- Project paths: Complete from filesystem (*.csproj, *.fsproj, *.vbproj)

**Test coverage:**
- `gonuget __complete package ''` → suggests `add list remove search`
- `gonuget __complete source ''` → suggests `add disable enable list remove update`
- Bash/Zsh/PowerShell completion script generation tests

## Error UX & Messaging

### Error Message Guidelines

**Format:**
- Single-line, no trailing punctuation
- Start with context, end with action
- Use tokens for names (avoid embedding in localized strings)

**Examples:**
```
Error: package 'Newtonsoft.Json' is not referenced in this project
Try: gonuget package list to see referenced packages

Error: source 'nuget.org' not found in NuGet.config
Try: gonuget source list to see configured sources
```

### Verbosity-Gated Details

**Normal verbosity:**
```
Error: failed to restore packages
```

**Diagnostic verbosity:**
```
[2025-10-31T12:34:56Z] [op:restore-abc123] Error: failed to restore packages
  Source: https://api.nuget.org/v3/index.json
  HTTP Status: 503 Service Unavailable
  Retry-After: 60 seconds
  Stack trace: ...
```

### Operation-Specific Error Handling

**package remove (not referenced):**
- Exit code: `3` (Not found)
- Message: `Error: package 'X' is not referenced in this project`
- Suggestion: `Try: gonuget package list`

**package search (zero results):**
- Exit code: `0` (Success - empty is valid)
- Console output: `No packages found matching 'X'`
- JSON output: `{"items": [], "total": 0}`

**source add (network failure):**
- Exit code: `4` (Network failure)
- Message: `Error: failed to verify source 'https://example.com/v3/index.json'`
- Detail (diagnostic): Include HTTP status, retry info, timeout details

## Command Mapping

### Package Operations

| Old Command | New Command | Action |
|-------------|-------------|--------|
| `gonuget add package <PKG>` | `gonuget package add <PKG>` | MOVE |
| N/A | `gonuget package list [PROJECT]` | NEW |
| N/A | `gonuget package search <TERM>` | NEW |
| N/A | `gonuget package remove <PKG>` | NEW |

### Source Operations

| Old Command | New Command | Action |
|-------------|-------------|--------|
| `gonuget add source <URL>` | `gonuget source add <URL>` | CHANGE |
| `gonuget list` | `gonuget source list` | CHANGE |
| `gonuget remove <NAME>` | `gonuget source remove <NAME>` | CHANGE |
| `gonuget enable <NAME>` | `gonuget source enable <NAME>` | CHANGE |
| `gonuget disable <NAME>` | `gonuget source disable <NAME>` | CHANGE |
| `gonuget update <NAME>` | `gonuget source update <NAME>` | CHANGE |

### Top-Level Operations

| Command | Action |
|---------|--------|
| `gonuget config get/set/list` | KEEP |
| `gonuget restore [PROJECT]` | KEEP |

## File Structure Changes

```
cmd/gonuget/commands/
├── package.go              # NEW: Parent command for package operations
├── package_add.go          # MOVE from add_package.go
├── package_list.go         # NEW
├── package_remove.go       # NEW
├── package_search.go       # NEW
├── source.go               # NEW: Parent command for source operations
├── source_add.go           # MOVE from add_source.go
├── source_list.go          # MOVE from list.go
├── source_remove.go        # MOVE from remove.go
├── source_enable.go        # MOVE from enable.go
├── source_disable.go       # MOVE from disable.go
├── source_update.go        # MOVE from update.go
├── config.go               # KEEP
├── restore.go              # KEEP
└── version.go              # KEEP
```

## Implementation Steps

### Step 1: Create Parent Commands

**File:** `cmd/gonuget/commands/package.go`

```go
package commands

import (
    "github.com/spf13/cobra"
    "github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewPackageCommand creates the package parent command
func NewPackageCommand(console output.Console) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "package",
        Short: "Manage NuGet package references",
        Long: `Manage NuGet package references for a project.
Subcommands allow you to add, remove, list, and search for packages.`,
        Example: `  gonuget package add Newtonsoft.Json
  gonuget package list --include-transitive
  gonuget package search Serilog
  gonuget package remove Newtonsoft.Json`,
    }

    // Add subcommands
    cmd.AddCommand(NewPackageAddCommand(console))
    cmd.AddCommand(NewPackageListCommand(console))
    cmd.AddCommand(NewPackageSearchCommand(console))
    cmd.AddCommand(NewPackageRemoveCommand(console))

    return cmd
}
```

**File:** `cmd/gonuget/commands/source.go`

```go
package commands

import (
    "github.com/spf13/cobra"
    "github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewSourceCommand creates the source parent command
func NewSourceCommand(console output.Console) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "source",
        Short: "Manage NuGet package sources",
        Long: `Manage NuGet package sources in NuGet.config files.
Subcommands allow you to add, remove, list, enable, disable, and update sources.`,
        Example: `  gonuget source add https://api.nuget.org/v3/index.json --name nuget.org
  gonuget source list
  gonuget source enable nuget.org
  gonuget source disable nuget.org
  gonuget source remove nuget.org`,
    }

    // Add subcommands
    cmd.AddCommand(NewSourceAddCommand(console))
    cmd.AddCommand(NewSourceListCommand(console))
    cmd.AddCommand(NewSourceRemoveCommand(console))
    cmd.AddCommand(NewSourceEnableCommand(console))
    cmd.AddCommand(NewSourceDisableCommand(console))
    cmd.AddCommand(NewSourceUpdateCommand(console))

    return cmd
}
```


### Step 2: Rename Command Files

```bash
# Rename package command files
mv cmd/gonuget/commands/add_package.go cmd/gonuget/commands/package_add.go
mv cmd/gonuget/commands/add_package_test.go cmd/gonuget/commands/package_add_test.go

# Rename source command files
mv cmd/gonuget/commands/add.go cmd/gonuget/commands/source_add.go  # (if not already named source_add.go)
mv cmd/gonuget/commands/list.go cmd/gonuget/commands/source_list.go
mv cmd/gonuget/commands/remove.go cmd/gonuget/commands/source_remove.go
mv cmd/gonuget/commands/enable.go cmd/gonuget/commands/source_enable.go
mv cmd/gonuget/commands/disable.go cmd/gonuget/commands/source_disable.go
mv cmd/gonuget/commands/update.go cmd/gonuget/commands/source_update.go
```

Update `package_add.go`:
- Rename `NewAddPackageCommand` → `NewPackageAddCommand`
- Update `Use: "package"` → `Use: "add"`
- **CRITICAL**: Ensure `Use: "add"` is verb-only (not `Use: "add package"`)
- **CRITICAL**: Do NOT set `Aliases` field (hard policy: no aliases)

Update all source command files:
- Rename functions: `NewAddCommand` → `NewSourceAddCommand`, `NewListCommand` → `NewSourceListCommand`, etc.
- Update `Use:` fields to be verb-only: `Use: "add"`, `Use: "list"`, `Use: "remove"`, etc.
- **CRITICAL**: Ensure `Use:` is verb-only (not `Use: "add source"`)
- **CRITICAL**: Do NOT set `Aliases` field on any command

### Step 3: Create New Package Commands

### Step 4: Implement New Commands

**File:** `cmd/gonuget/commands/package_list.go`

```go
package commands

import (
    "context"
    "fmt"
    "github.com/spf13/cobra"
    "github.com/willibrandon/gonuget/cmd/gonuget/output"
    "github.com/willibrandon/gonuget/restore"
)

// NewPackageListCommand creates the package list command
func NewPackageListCommand(console output.Console) *cobra.Command {
    var (
        includeTransitive bool
        framework         string
        format            string
        projectPath       string
    )

    cmd := &cobra.Command{
        Use:   "list [PROJECT]",
        Short: "List all package references of the project",
        Long:  `List all package references of the project or solution.`,
        Args:  cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            // Get project path
            if len(args) > 0 {
                projectPath = args[0]
            }

            // Read obj/project.assets.json and display packages
            // Implementation matches dotnet package list behavior
            return runPackageList(cmd.Context(), projectPath, includeTransitive, framework, format, console)
        },
    }

    cmd.Flags().BoolVar(&includeTransitive, "include-transitive", false, "Lists transitive and top-level packages")
    cmd.Flags().StringVarP(&framework, "framework", "f", "", "Chooses a framework to show its packages")
    cmd.Flags().StringVar(&format, "format", "console", "Output format: console or json")

    return cmd
}

func runPackageList(ctx context.Context, projectPath string, includeTransitive bool, framework string, format string, console output.Console) error {
    // TODO: Implement lock file reading and package listing
    return fmt.Errorf("not implemented")
}
```

**File:** `cmd/gonuget/commands/package_search.go`

```go
package commands

import (
    "context"
    "github.com/spf13/cobra"
    "github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewPackageSearchCommand creates the package search command
func NewPackageSearchCommand(console output.Console) *cobra.Command {
    var (
        source      []string
        prerelease  bool
        exactMatch  bool
        format      string
        take        int
    )

    cmd := &cobra.Command{
        Use:   "search <SEARCH_TERM>",
        Short: "Search for NuGet packages",
        Long:  `Searches one or more package sources for packages that match a search term.`,
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            searchTerm := args[0]
            return runPackageSearch(cmd.Context(), searchTerm, source, prerelease, exactMatch, take, format, console)
        },
    }

    cmd.Flags().StringSliceVarP(&source, "source", "s", nil, "NuGet source(s) to search")
    cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Include prerelease packages")
    cmd.Flags().BoolVar(&exactMatch, "exact-match", false, "Use exact match")
    cmd.Flags().StringVar(&format, "format", "console", "Output format: console or json")
    cmd.Flags().IntVar(&take, "take", 20, "Number of results to return")

    return cmd
}

func runPackageSearch(ctx context.Context, searchTerm string, sources []string, prerelease bool, exactMatch bool, take int, format string, console output.Console) error {
    // TODO: Implement package search
    return nil
}
```

**File:** `cmd/gonuget/commands/package_remove.go`

```go
package commands

import (
    "github.com/spf13/cobra"
    "github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewPackageRemoveCommand creates the package remove command
func NewPackageRemoveCommand(console output.Console) *cobra.Command {
    var projectPath string

    cmd := &cobra.Command{
        Use:   "remove <PACKAGE_NAME>",
        Short: "Remove a NuGet package reference from the project",
        Long:  `Remove a NuGet package reference from the project file.`,
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            packageID := args[0]
            return runPackageRemove(packageID, projectPath, console)
        },
    }

    cmd.Flags().StringVar(&projectPath, "project", "", "The project file to operate on")

    return cmd
}

func runPackageRemove(packageID string, projectPath string, console output.Console) error {
    // TODO: Implement package removal
    return nil
}
```

### Step 5: Update Main Registration

**File:** `cmd/gonuget/main.go`

```go
func main() {
    // ... setup code ...

    // Register parent commands
    cli.AddCommand(commands.NewPackageCommand(cli.Console))
    cli.AddCommand(commands.NewSourceCommand(cli.Console))

    // Top-level commands
    cli.AddCommand(commands.NewConfigCommand(cli.Console))
    cli.AddCommand(commands.NewRestoreCommand(cli.Console))
    cli.AddCommand(commands.NewVersionCommand(cli.Console))

    // Configure error suggestions
    cli.SuggestionsMinimumDistance = 1

    // Add custom error handler for verb-first detection
    // (implementation details in error handling section)

    // ... execute ...
}
```

### Step 6: Implement Error Hints

Create custom error handler to detect verb-first patterns and suggest noun-first alternatives.

**Note:** `SetFlagErrorFunc` only fires for flag errors. To catch command-not-found cases (e.g., `gonuget add package`), handle errors from `Execute()`:

```go
// In main.go:
func main() {
    // ... setup code ...

    // Configure error suggestions
    cli.SilenceErrors = true  // Suppress default Cobra error output
    cli.SuggestionsMinimumDistance = 1

    // Execute and handle errors
    err := cli.Execute()
    if err != nil {
        // Check for verb-first patterns
        args := os.Args[1:]
        if len(args) >= 2 {
            verb := args[0]
            noun := args[1]

            // Check for verb-first package commands
            if verb == "add" && noun == "package" {
                fmt.Fprintf(os.Stderr, "Error: the verb-first form is not supported\nTry: gonuget package add\n\nRun 'gonuget --help' for usage.\n")
                os.Exit(1)
            }
            if verb == "list" && noun == "package" {
                fmt.Fprintf(os.Stderr, "Error: the verb-first form is not supported\nTry: gonuget package list\n\nRun 'gonuget --help' for usage.\n")
                os.Exit(1)
            }
            if verb == "remove" && noun == "package" {
                fmt.Fprintf(os.Stderr, "Error: the verb-first form is not supported\nTry: gonuget package remove\n\nRun 'gonuget --help' for usage.\n")
                os.Exit(1)
            }

            // Check for verb-first source commands
            if verb == "add" && noun == "source" {
                fmt.Fprintf(os.Stderr, "Error: the verb-first form is not supported\nTry: gonuget source add\n\nRun 'gonuget --help' for usage.\n")
                os.Exit(1)
            }
            if verb == "list" && noun == "source" {
                fmt.Fprintf(os.Stderr, "Error: the verb-first form is not supported\nTry: gonuget source list\n\nRun 'gonuget --help' for usage.\n")
                os.Exit(1)
            }
            if verb == "remove" && noun == "source" {
                fmt.Fprintf(os.Stderr, "Error: the verb-first form is not supported\nTry: gonuget source remove\n\nRun 'gonuget --help' for usage.\n")
                os.Exit(1)
            }
            if verb == "enable" && noun == "source" {
                fmt.Fprintf(os.Stderr, "Error: the verb-first form is not supported\nTry: gonuget source enable\n\nRun 'gonuget --help' for usage.\n")
                os.Exit(1)
            }
            if verb == "disable" && noun == "source" {
                fmt.Fprintf(os.Stderr, "Error: the verb-first form is not supported\nTry: gonuget source disable\n\nRun 'gonuget --help' for usage.\n")
                os.Exit(1)
            }
            if verb == "update" && noun == "source" {
                fmt.Fprintf(os.Stderr, "Error: the verb-first form is not supported\nTry: gonuget source update\n\nRun 'gonuget --help' for usage.\n")
                os.Exit(1)
            }
        }

        // For other errors, print the default Cobra error
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

**Alternative approach:** Create a custom `RunE` wrapper or `PersistentPreRunE` hook to intercept and check args before command execution.

### Step 7: Update Tests

Update test invocations:
```go
// Package commands
cmd := exec.Command("gonuget", "package", "add", "Newtonsoft.Json")
cmd := exec.Command("gonuget", "package", "list")
cmd := exec.Command("gonuget", "package", "remove", "Newtonsoft.Json")
cmd := exec.Command("gonuget", "package", "search", "Serilog")

// Source commands
cmd := exec.Command("gonuget", "source", "add", "https://api.nuget.org/v3/index.json")
cmd := exec.Command("gonuget", "source", "list")
cmd := exec.Command("gonuget", "source", "remove", "nuget.org")
cmd := exec.Command("gonuget", "source", "enable", "nuget.org")
cmd := exec.Command("gonuget", "source", "disable", "nuget.org")
cmd := exec.Command("gonuget", "source", "update", "nuget.org")

// Top-level commands
cmd := exec.Command("gonuget", "restore")
cmd := exec.Command("gonuget", "config", "get", "packageSources")
cmd := exec.Command("gonuget", "version")
```

Add test for help output structure:
```go
func TestHelpOutputShowsExpectedCommands(t *testing.T) {
    // Test that 'gonuget help' shows exactly the expected top-level commands
    cmd := exec.Command("gonuget", "help")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    helpText := string(output)

    // Assert that help output contains all expected commands
    expectedCommands := []string{"package", "source", "restore", "config", "version"}
    for _, cmd := range expectedCommands {
        assert.Contains(t, helpText, cmd, "Help output should include '%s' command", cmd)
    }

    // Guard against regressions: ensure no verb-first commands appear
    unexpectedCommands := []string{"add package", "list package", "add source", "list source"}
    for _, cmd := range unexpectedCommands {
        assert.NotContains(t, helpText, cmd, "Help output should NOT include verb-first pattern '%s'", cmd)
    }
}
```

Add tests for verb-first error messages:
```go
func TestVerbFirstCommandsProduceHelpfulErrors(t *testing.T) {
    tests := []struct {
        name           string
        args           []string
        expectedError  string
        expectedSuggestion string
    }{
        {
            name:           "add package",
            args:           []string{"add", "package", "Newtonsoft.Json"},
            expectedError:  "verb-first form is not supported",
            expectedSuggestion: "gonuget package add",
        },
        {
            name:           "list source",
            args:           []string{"list", "source"},
            expectedError:  "verb-first form is not supported",
            expectedSuggestion: "gonuget source list",
        },
        // Add more cases for all verb-first patterns
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := exec.Command("gonuget", tt.args...)
            output, err := cmd.CombinedOutput()

            assert.Error(t, err, "Verb-first command should fail")
            assert.Contains(t, string(output), tt.expectedError)
            assert.Contains(t, string(output), tt.expectedSuggestion)
        })
    }
}
```

## Testing Strategy

### Golden Tests

Capture and version control expected outputs for regression detection:

**Help output golden tests:**
```go
func TestGoldenHelpOutput_TopLevel(t *testing.T) {
    cmd := exec.Command("gonuget", "help")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    goldenFile := "testdata/golden/help_toplevel.txt"
    if *update {
        os.WriteFile(goldenFile, output, 0644)
    }

    expected, err := os.ReadFile(goldenFile)
    require.NoError(t, err)
    assert.Equal(t, string(expected), string(output))
}

// Similar tests for:
// - gonuget package --help
// - gonuget source --help
// - gonuget package add --help
// etc.
```

**JSON schema golden tests:**
```go
func TestGoldenJSON_PackageList(t *testing.T) {
    cmd := exec.Command("gonuget", "package", "list", "--format", "json", testProjectPath)
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    var result map[string]interface{}
    err = json.Unmarshal(output, &result)
    require.NoError(t, err)

    // Validate schema version
    assert.Equal(t, "1.0", result["schemaVersion"])

    // Validate structure
    assert.Contains(t, result, "project")
    assert.Contains(t, result, "packages")
    assert.Contains(t, result, "warnings")
    assert.Contains(t, result, "elapsedMs")

    // Golden file comparison (normalize timestamps/paths)
    normalized := normalizeJSON(result)
    goldenFile := "testdata/golden/package_list.json"
    compareGolden(t, normalized, goldenFile)
}
```

**Exit code tests:**
```go
func TestExitCodes(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        wantCode int
    }{
        {"success", []string{"package", "list"}, 0},
        {"invalid args", []string{"invalid-command"}, 2},
        {"not found", []string{"package", "remove", "NonExistentPackage"}, 3},
        // Network failure test requires mock
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := exec.Command("gonuget", tt.args...)
            err := cmd.Run()

            if tt.wantCode == 0 {
                assert.NoError(t, err)
            } else {
                var exitErr *exec.ExitError
                require.ErrorAs(t, err, &exitErr)
                assert.Equal(t, tt.wantCode, exitErr.ExitCode())
            }
        })
    }
}
```

**Reflection test for Use: verb-only convention:**
```go
func TestCommandUseFieldsAreVerbOnly(t *testing.T) {
    // Scan all registered commands and ensure Use fields don't contain spaces
    // (which would indicate "verb noun" instead of just "verb")

    rootCmd := commands.NewRootCommand(console)

    // Recursively check all commands
    var check func(*cobra.Command)
    check = func(cmd *cobra.Command) {
        // Skip root and parent commands (package, source)
        if cmd.Use == "" || cmd.Parent() == nil {
            return
        }

        // For subcommands, ensure Use is verb-only
        if cmd.Parent().Use == "package" || cmd.Parent().Use == "source" {
            assert.NotContains(t, cmd.Use, " ",
                "Command %s has space in Use field - should be verb-only, got: %s",
                cmd.Name(), cmd.Use)
        }

        // Check all subcommands
        for _, subCmd := range cmd.Commands() {
            check(subCmd)
        }
    }

    check(rootCmd)
}

func TestNoCommandsHaveAliases(t *testing.T) {
    rootCmd := commands.NewRootCommand(console)

    var check func(*cobra.Command)
    check = func(cmd *cobra.Command) {
        assert.Empty(t, cmd.Aliases,
            "Command %s has aliases - policy is no aliases: %v",
            cmd.Name(), cmd.Aliases)

        for _, subCmd := range cmd.Commands() {
            check(subCmd)
        }
    }

    check(rootCmd)
}
```

### Dotnet Parity Tests

```go
func TestCommandStructure_MatchesDotnet(t *testing.T) {
    tests := []struct {
        name           string
        gonugetArgs    []string
        dotnetArgs     []string
        compareOutput  bool
    }{
        {
            name:        "package add",
            gonugetArgs: []string{"package", "add", "Newtonsoft.Json"},
            dotnetArgs:  []string{"package", "add", "Newtonsoft.Json"},
        },
        {
            name:        "source list",
            gonugetArgs: []string{"source", "list"},
            dotnetArgs:  []string{"nuget", "list", "source"},
        },
        {
            name:        "source add",
            gonugetArgs: []string{"source", "add", "https://api.nuget.org/v3/index.json"},
            dotnetArgs:  []string{"nuget", "add", "source", "https://api.nuget.org/v3/index.json"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Verify command structure exists
            // Don't compare exact output, just structure
        })
    }
}
```

## Documentation Updates

### README.md

```markdown
# gonuget

## Quick Start

**Note:** gonuget uses noun-first command structure (`gonuget package add`, `gonuget source list`) exclusively. Top-level commands (`restore`, `config`, `version`) are exceptions that match dotnet CLI exactly.

```bash
# Package operations (noun-first)
gonuget package add Newtonsoft.Json
gonuget package list --include-transitive
gonuget package search Serilog
gonuget package remove Newtonsoft.Json

# Source operations (noun-first)
gonuget source add https://api.nuget.org/v3/index.json --name nuget.org
gonuget source list
gonuget source enable nuget.org
gonuget source disable nuget.org
gonuget source remove nuget.org
gonuget source update nuget.org --source https://new-url.com

# Top-level operations (exceptions - match dotnet exactly)
gonuget restore
gonuget config get packageSources
gonuget version
```
```

## Code Hygiene Checklist

Before marking implementation complete, verify:

### Command Registration

- [ ] All subcommand `Use:` fields are **verb-only** (`Use: "add"`, not `Use: "add package"`)
- [ ] No `Aliases` fields are set on any command (hard policy: no aliases)
- [ ] Parent commands (`package.go`, `source.go`) correctly register all subcommands
- [ ] `version.go` is registered at top-level and appears in `gonuget help`

### Error Handling

- [ ] Custom error handler detects verb-first patterns (`add package`, `list source`)
- [ ] Error messages suggest correct noun-first syntax
- [ ] `SuggestionsMinimumDistance` is set appropriately for typo suggestions

### Help Text

- [ ] All commands have clear `Short` and `Long` descriptions
- [ ] `Short` descriptions start with verbs ("Add a package reference...", "List package references...")
- [ ] Examples show both minimal and fully-flagged forms
- [ ] Top-level help shows commands in logical order: `config, package, restore, source, version` (or custom order)
- [ ] Exceptions (restore, config, version) are documented in help output

### Testing

- [ ] Integration tests cover noun-first commands
- [ ] Tests verify verb-first commands fail with appropriate errors
- [ ] Help output tests validate command structure visibility
- [ ] Test asserts `gonuget help` shows exactly `{package, source, restore, config, version}` (guard against regressions)

## Success Criteria

### Command Structure
- [ ] `gonuget package [subcommand]` matches `dotnet package [subcommand]` exactly (noun-first form only)
- [ ] `gonuget source [subcommand]` uses fully noun-first structure (noun before verb)
- [ ] Old verb-first commands (`gonuget add package`, `gonuget add source`) are NOT supported (noun-first only)
- [ ] Verb-first attempts produce helpful error messages with correct noun-first syntax suggestions
- [ ] All subcommand `Use:` strings are verb-only (`"add"`, `"list"`) with no `Aliases` set
- [ ] Command structure follows modern CLI standards (fully noun-first across all namespaces, with documented exceptions)

### Flag Consistency
- [ ] All flags use kebab-case naming
- [ ] `--format console|json` implemented consistently on list/search/config commands
- [ ] `--verbosity quiet|minimal|normal|detailed|diagnostic` works on all commands
- [ ] `--name` used consistently (never `--source-name`)
- [ ] Positional args follow documented pattern (positional for required, flags for optional)
- [ ] `--what-if` / `--dry-run` implemented on all mutating commands
- [ ] `--yes` / `-y` available for confirmation suppression

### JSON Output
- [ ] JSON output never prints non-JSON to stdout (human text goes to stderr)
- [ ] All JSON responses include `schemaVersion` field
- [ ] Package list JSON matches documented contract
- [ ] Package search JSON matches documented contract
- [ ] Schema version is tested and validated

### Exit Codes
- [ ] Exit code 0 for success
- [ ] Exit code 1 for generic errors
- [ ] Exit code 2 for invalid arguments/unknown commands
- [ ] Exit code 3 for not found errors
- [ ] Exit code 4 for network/temporary failures
- [ ] Each exit code has dedicated test coverage

### Configuration & Environment
- [ ] Precedence order implemented: CLI flags > env vars > config files > defaults
- [ ] `GONUGET_*` environment variables supported
- [ ] `NUGET_PACKAGES` read for parity (read-only)
- [ ] Configuration precedence tested

### Shell Completion
- [ ] Namespace completion works (`gonuget <TAB>` → `config package restore source version`)
- [ ] Verb completion works (`gonuget package <TAB>` → `add list remove search`)
- [ ] Dynamic completion for source names from NuGet.config
- [ ] Completion scripts tested for bash/zsh/PowerShell

### Error UX
- [ ] Error messages are single-line, no trailing punctuation
- [ ] Error messages use token-based formatting (not embedded names)
- [ ] Package remove (not found) returns exit code 3 with helpful message
- [ ] Package search (zero results) returns exit code 0 with empty JSON
- [ ] Network failures return exit code 4 with diagnostic details at `--verbosity diagnostic`

### Help & Documentation
- [ ] `gonuget version` documented as approved exception and appears in help
- [ ] Help text clearly documents noun-first command structure for both namespaces
- [ ] Help text explicitly calls out top-level exceptions (restore, config, version) with rationale
- [ ] `Short` descriptions start with verbs
- [ ] Examples show both minimal and fully-flagged forms
- [ ] Top-level help shows commands in logical order

### Testing
- [ ] All tests pass
- [ ] Golden tests for help output (top-level + subcommands)
- [ ] Golden tests for `--format json` payloads (schema snapshots)
- [ ] Exit code table tests (one test per code)
- [ ] Shell completion suggestion tests
- [ ] Reflection test enforces `Use:` verb-only convention
- [ ] Verb-first suggestion tests
- [ ] Help output regression guard test (`{package, source, restore, config, version}` exactly)

## References

- dotnet CLI source: https://github.com/dotnet/sdk
- dotnet nuget: https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-nuget
- dotnet package: https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-package
- .NET 10 Command Aliases: https://learn.microsoft.com/en-us/dotnet/core/whats-new/dotnet-10/sdk#more-consistent-command-order
