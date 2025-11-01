# CLI Command Structure Restructure - Simplified dotnet-style Commands

**Status:** Planning
**Goal:** Achieve functional parity with dotnet CLI using simplified command structure
**Approach:** Big bang restructure (immediate breaking changes)

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
- Mixed command hierarchy (parent `add` vs flat `list/enable/disable`)
- Source commands scattered at top-level
- No clear namespace separation
- Inconsistent with dotnet pattern

## Target State (Simplified dotnet-style Structure)

### Package Namespace (`gonuget package`)

```bash
gonuget package add <PACKAGE_NAME>
gonuget package list [PROJECT]
gonuget package remove <PACKAGE_NAME>
gonuget package search <SEARCH_TERM>
```

**Matches:** `dotnet package` exactly

### Source Commands

```bash
gonuget add source <SOURCE_URL>
gonuget list source
gonuget remove source <NAME>
gonuget enable source <NAME>
gonuget disable source <NAME>
gonuget update source <NAME>
```

**Matches:** `dotnet nuget [command] source` (removes "nuget" parent, keeps same structure)

### Top-Level Commands

```bash
gonuget config get|set|list
gonuget restore [PROJECT]
```

**Matches:** `dotnet restore`, `dotnet nuget config`

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
| `gonuget add source <URL>` | `gonuget add source <URL>` | KEEP |
| `gonuget list` | `gonuget list source` | CHANGE |
| `gonuget remove <NAME>` | `gonuget remove source <NAME>` | CHANGE |
| `gonuget enable <NAME>` | `gonuget enable source <NAME>` | CHANGE |
| `gonuget disable <NAME>` | `gonuget disable source <NAME>` | CHANGE |
| `gonuget update <NAME>` | `gonuget update source <NAME>` | CHANGE |

### Top-Level Operations

| Command | Action |
|---------|--------|
| `gonuget config get/set/list` | KEEP |
| `gonuget restore [PROJECT]` | KEEP |

## File Structure Changes

```
cmd/gonuget/commands/
├── package.go              # NEW: Parent command
├── package_add.go          # MOVE from add_package.go
├── package_list.go         # NEW
├── package_remove.go       # NEW
├── package_search.go       # NEW
├── add.go                  # KEEP: Parent for "add source"
├── source_add.go           # KEEP (already exists)
├── list.go                 # UPDATE: Change from "list" to "list source"
├── source_list.go          # KEEP
├── remove.go               # UPDATE: Change to "remove source"
├── source_remove.go        # KEEP
├── enable.go               # UPDATE: Change to "enable source"
├── source_enable.go        # KEEP
├── disable.go              # UPDATE: Change to "disable source"
├── source_disable.go       # KEEP
├── update.go               # UPDATE: Change to "update source"
├── source_update.go        # KEEP
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


### Step 2: Rename Package Command Files

```bash
# Rename add_package.go to package_add.go
mv cmd/gonuget/commands/add_package.go cmd/gonuget/commands/package_add.go
mv cmd/gonuget/commands/add_package_test.go cmd/gonuget/commands/package_add_test.go
```

Update `package_add.go`:
- Rename `NewAddPackageCommand` → `NewPackageAddCommand`
- Update `Use: "package"` → `Use: "add"`

### Step 3: Update Source Command Files

Update `Use:` fields to include "source" argument:
- `list.go`: `Use: "list"` → `Use: "list source"`
- `remove.go`: `Use: "remove <name>"` → `Use: "remove source <NAME>"`
- `enable.go`: `Use: "enable <name>"` → `Use: "enable source <NAME>"`
- `disable.go`: `Use: "disable <name>"` → `Use: "disable source <NAME>"`
- `update.go`: `Use: "update <name>"` → `Use: "update source <NAME>"`

Keep `add source` as-is (already correct).

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

    // Register new command structure
    cli.AddCommand(commands.NewPackageCommand(cli.Console))
    cli.AddCommand(commands.NewAddCommand(cli.Console))        // Keep: add source
    cli.AddCommand(commands.NewListCommand(cli.Console))       // Update: list source
    cli.AddCommand(commands.NewRemoveCommand(cli.Console))     // Update: remove source
    cli.AddCommand(commands.NewEnableCommand(cli.Console))     // Update: enable source
    cli.AddCommand(commands.NewDisableCommand(cli.Console))    // Update: disable source
    cli.AddCommand(commands.NewUpdateCommand(cli.Console))     // Update: update source

    // Top-level commands
    cli.AddCommand(commands.NewConfigCommand(cli.Console))
    cli.AddCommand(commands.NewRestoreCommand(cli.Console))
    cli.AddCommand(commands.NewVersionCommand(cli.Console))

    // ... execute ...
}
```

### Step 6: Update Tests

Update test invocations:
```go
// Package commands
cmd := exec.Command("gonuget", "package", "add", "Newtonsoft.Json")

// Source commands (no changes needed)
cmd := exec.Command("gonuget", "add", "source", "https://api.nuget.org/v3/index.json")
cmd := exec.Command("gonuget", "list", "source")
cmd := exec.Command("gonuget", "remove", "source", "nuget.org")
```

## Testing Strategy

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
            gonugetArgs: []string{"list", "source"},
            dotnetArgs:  []string{"nuget", "list", "source"},
        },
        {
            name:        "source add",
            gonugetArgs: []string{"add", "source", "https://api.nuget.org/v3/index.json"},
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

```bash
# Package operations
gonuget package add Newtonsoft.Json
gonuget package list --include-transitive
gonuget package search Serilog
gonuget package remove Newtonsoft.Json

# Source operations
gonuget add source https://api.nuget.org/v3/index.json --name nuget.org
gonuget list source
gonuget enable source nuget.org
gonuget disable source nuget.org
gonuget remove source nuget.org
gonuget update source nuget.org --source https://new-url.com

# Project operations
gonuget restore
gonuget config get packageSources
```
```

## Success Criteria

- [ ] `gonuget package` matches `dotnet package` exactly
- [ ] `gonuget [verb] source` matches `dotnet nuget [verb] source` (removes "nuget" parent)
- [ ] All flags use kebab-case matching dotnet
- [ ] All output formats match dotnet
- [ ] All tests pass
- [ ] Interop tests correctly handle different command structures

## References

- dotnet CLI source: https://github.com/dotnet/sdk
- dotnet nuget: https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-nuget
- dotnet package: https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-package
