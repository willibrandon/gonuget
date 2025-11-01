# CLI Command Structure Restructure - Full dotnet Parity

**Status:** Planning
**Goal:** Achieve 100% command structure parity with dotnet CLI
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

## Target State (100% dotnet Parity)

### Package Namespace (`gonuget package`)

```bash
gonuget package add <PACKAGE_NAME>
gonuget package list [PROJECT]
gonuget package remove <PACKAGE_NAME>
gonuget package search <SEARCH_TERM>
```

**Matches:** `dotnet package`

### Source Namespace (`gonuget source`)

```bash
gonuget source add <SOURCE_URL>
gonuget source list
gonuget source remove <NAME>
gonuget source enable <NAME>
gonuget source disable <NAME>
gonuget source update <NAME>
```

**Matches:** `dotnet nuget` source commands (grouped under `source` for clarity)

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
| `gonuget add source <URL>` | `gonuget source add <URL>` | MOVE |
| `gonuget list` | `gonuget source list` | MOVE |
| `gonuget remove <NAME>` | `gonuget source remove <NAME>` | MOVE |
| `gonuget enable <NAME>` | `gonuget source enable <NAME>` | MOVE |
| `gonuget disable <NAME>` | `gonuget source disable <NAME>` | MOVE |
| `gonuget update <NAME>` | `gonuget source update <NAME>` | MOVE |

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
├── source.go               # NEW: Parent command
├── source_add.go           # MOVE from source_add.go
├── source_list.go          # MOVE from source_list.go
├── source_remove.go        # MOVE from source_remove.go
├── source_enable.go        # MOVE from source_enable.go
├── source_disable.go       # MOVE from source_disable.go
├── source_update.go        # MOVE from source_update.go
├── config.go               # KEEP
├── restore.go              # KEEP
├── version.go              # KEEP
└── [DELETE] add.go         # REMOVE parent command
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
        Long: `Manage NuGet package sources in NuGet.config.
Subcommands allow you to add, remove, list, enable, disable, and update sources.`,
        Example: `  gonuget source add https://api.nuget.org/v3/index.json --name nuget.org
  gonuget source list
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

### Step 2: Rename Package Command Files

```bash
# Rename add_package.go to package_add.go
mv cmd/gonuget/commands/add_package.go cmd/gonuget/commands/package_add.go
mv cmd/gonuget/commands/add_package_test.go cmd/gonuget/commands/package_add_test.go
```

Update `package_add.go`:
- Rename `NewAddPackageCommand` → `NewPackageAddCommand`
- Update `Use: "package"` → `Use: "add"`

### Step 3: Rename Source Command Files

Already named correctly, just update function names:
- `NewListCommand` → `NewSourceListCommand`
- `NewRemoveCommand` → `NewSourceRemoveCommand`
- `NewEnableCommand` → `NewSourceEnableCommand`
- `NewDisableCommand` → `NewSourceDisableCommand`
- `NewUpdateCommand` → `NewSourceUpdateCommand`
- `NewAddSourceCommand` → `NewSourceAddCommand`

Update each file's `Use:` field to remove "source" prefix since it's now a subcommand.

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
    cli.AddCommand(commands.NewSourceCommand(cli.Console))

    // Top-level commands
    cli.AddCommand(commands.NewConfigCommand(cli.Console))
    cli.AddCommand(commands.NewRestoreCommand(cli.Console))
    cli.AddCommand(commands.NewVersionCommand(cli.Console))

    // ... execute ...
}
```

### Step 6: Delete Old Files

```bash
rm cmd/gonuget/commands/add.go
rm cmd/gonuget/commands/add_test.go  # if exists
```

### Step 7: Update Tests

Rename all test functions to match new command names:
- `TestAddPackageCommand` → `TestPackageAddCommand`
- `TestListCommand` → `TestSourceListCommand`
- etc.

Update test invocations:
```go
// Old
cmd := exec.Command("gonuget", "add", "package", "Newtonsoft.Json")

// New
cmd := exec.Command("gonuget", "package", "add", "Newtonsoft.Json")
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
            gonugetArgs: []string{"source", "list"},
            dotnetArgs:  []string{"nuget", "list"},
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
gonuget source add https://api.nuget.org/v3/index.json --name nuget.org
gonuget source list
gonuget source enable nuget.org
gonuget source disable nuget.org

# Project operations
gonuget restore
gonuget config get packageSources
```
```

## Success Criteria

- [ ] `gonuget package` matches `dotnet package` exactly
- [ ] `gonuget source` matches `dotnet nuget` source commands exactly
- [ ] All flags use kebab-case matching dotnet
- [ ] All output formats match dotnet
- [ ] All tests pass
- [ ] No old command structure remains

## References

- dotnet CLI source: https://github.com/dotnet/sdk
- dotnet nuget: https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-nuget
- dotnet package: https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-package
