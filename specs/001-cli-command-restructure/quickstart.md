# Quickstart Guide: CLI Command Structure Restructure

**Feature**: CLI Command Structure Restructure
**Branch**: `001-cli-command-restructure`
**Date**: 2025-10-31

## Overview

This guide provides a quick reference for developers implementing the CLI command restructure. It includes code patterns, testing strategies, and validation checklists.

## Table of Contents

1. [Command Implementation Pattern](#command-implementation-pattern)
2. [Parent Command Pattern](#parent-command-pattern)
3. [Subcommand Pattern](#subcommand-pattern)
4. [Error Handler Pattern](#error-handler-pattern)
5. [Testing Strategy](#testing-strategy)
6. [Validation Checklist](#validation-checklist)
7. [Common Pitfalls](#common-pitfalls)

---

## Command Implementation Pattern

### File Naming Convention

```text
Parent commands:     {noun}.go           (e.g., package.go, source.go)
Subcommands:         {parent}_{verb}.go  (e.g., package_add.go, source_list.go)
Top-level commands:  {verb}.go           (e.g., restore.go, config.go, version.go)
```

### Directory Structure

```text
cmd/gonuget/commands/
├── package.go              # Parent: gonuget package
├── package_add.go          # Subcommand: gonuget package add
├── package_list.go         # Subcommand: gonuget package list
├── package_remove.go       # Subcommand: gonuget package remove
├── package_search.go       # Subcommand: gonuget package search
├── source.go               # Parent: gonuget source
├── source_add.go           # Subcommand: gonuget source add
├── source_list.go          # Subcommand: gonuget source list
├── source_remove.go        # Subcommand: gonuget source remove
├── source_enable.go        # Subcommand: gonuget source enable
├── source_disable.go       # Subcommand: gonuget source disable
├── source_update.go        # Subcommand: gonuget source update
├── restore.go              # Top-level: gonuget restore
├── config.go               # Top-level: gonuget config
├── version.go              # Top-level: gonuget version
├── root.go                 # Root command registration
└── errors.go               # Custom error handler
```

---

## Parent Command Pattern

**Purpose**: Parent commands provide namespace grouping without executing logic themselves.

### Template: `package.go`

```go
package commands

import "github.com/spf13/cobra"

var packageCmd = &cobra.Command{
    Use:   "package",  // MUST be single noun
    Short: "Manage package references",  // MUST start with verb
    Long: `Manage NuGet package references in .NET project files.

This command provides operations for adding, listing, removing, and searching
packages. All operations modify or query .NET project files (.csproj, .fsproj, .vbproj).`,
    // NO Run function - parent commands are containers only
    // NO Aliases field - zero tolerance policy
}

func init() {
    // Add persistent flags (inherited by all subcommands)
    packageCmd.PersistentFlags().StringP("project", "p", "",
        "Path to .NET project file")

    // Subcommands are registered in their own init() functions via:
    // packageCmd.AddCommand(addCmd)
}

// Exported for root.go to register
func GetPackageCommand() *cobra.Command {
    return packageCmd
}
```

### Key Rules for Parent Commands

✅ **DO**:
- Use single noun in `Use` field (package, source)
- Start `Short` description with verb (Manage, Configure)
- Export getter function for root registration
- Define persistent flags shared by all subcommands
- Omit `Run` function (parent commands don't execute)

❌ **DON'T**:
- Set `Aliases` field (forbidden by VR-004)
- Include verb in `Use` field ("manage-package" is wrong)
- Implement `Run` function (logic belongs in subcommands)
- Register subcommands directly in parent (use init() in subcommand file)

---

## Subcommand Pattern

**Purpose**: Subcommands implement actual command logic under parent namespaces.

### Template: `package_add.go`

```go
package commands

import (
    "fmt"
    "github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
    Use:   "add <PACKAGE_ID>",  // MUST be verb-only (no "add package")
    Short: "Add a package reference to the project",  // MUST start with verb
    Long: `Add a NuGet package reference to a .NET project file.

This command modifies the .csproj (or .fsproj, .vbproj) file by adding a
<PackageReference> element with the specified package ID and version.`,
    Example: `  # Add latest version
  gonuget package add Newtonsoft.Json

  # Add specific version
  gonuget package add Serilog --version 3.1.1

  # Add to specific project
  gonuget package add Microsoft.Extensions.Logging --project ./src/MyApp/MyApp.csproj`,
    Args: cobra.ExactArgs(1),  // Require exactly 1 positional arg
    RunE: runAddPackage,  // Use RunE for error handling
    // NO Aliases field - zero tolerance policy
}

func init() {
    // Register subcommand with parent
    packageCmd.AddCommand(addCmd)

    // Add command-specific flags
    addCmd.Flags().StringP("version", "v", "", "Package version")
    addCmd.Flags().StringP("framework", "f", "", "Target framework moniker")
    addCmd.Flags().Bool("prerelease", false, "Include prerelease versions")
    addCmd.Flags().StringP("source", "s", "", "Package source URL or name")
}

func runAddPackage(cmd *cobra.Command, args []string) error {
    packageID := args[0]

    // Get flags
    version, _ := cmd.Flags().GetString("version")
    project, _ := cmd.Flags().GetString("project")
    framework, _ := cmd.Flags().GetString("framework")
    prerelease, _ := cmd.Flags().GetBool("prerelease")
    source, _ := cmd.Flags().GetString("source")

    // Implementation logic here
    fmt.Printf("Adding package %s (version: %s) to project %s\n",
        packageID, version, project)

    return nil
}
```

### Key Rules for Subcommands

✅ **DO**:
- Use single verb in `Use` field (add, list, remove, search)
- Include positional args in `Use` (e.g., "add <PACKAGE_ID>")
- Start `Short` description with verb
- Use `RunE` for error handling (returns error)
- Register with parent in `init()` via `parentCmd.AddCommand()`
- Validate args with `Args: cobra.ExactArgs(n)`

❌ **DON'T**:
- Set `Aliases` field (forbidden by VR-004)
- Include parent noun in `Use` ("add package" is wrong, "add" is correct)
- Use `Run` function (prefer `RunE` for error handling)
- Forget to register with parent command
- Use spaces in `Use` field (violates VR-002)

---

## Error Handler Pattern

**Purpose**: Detect verb-first command attempts and suggest noun-first alternatives.

### Template: `errors.go`

```go
package commands

import (
    "fmt"
    "strings"

    "github.com/spf13/cobra"
)

// Verb-first patterns that should be detected and rejected
var verbFirstPatterns = map[string]string{
    // Package namespace
    "add package":    "gonuget package add",
    "list package":   "gonuget package list",
    "remove package": "gonuget package remove",
    "search package": "gonuget package search",

    // Source namespace
    "add source":    "gonuget source add",
    "list source":   "gonuget source list",
    "remove source": "gonuget source remove",

    // Top-level verbs that imply source (backward compatibility)
    "enable":  "gonuget source enable",
    "disable": "gonuget source disable",
    "update":  "gonuget source update",
}

// SetupCustomErrorHandler configures verb-first pattern detection
func SetupCustomErrorHandler(rootCmd *cobra.Command) {
    rootCmd.SilenceErrors = true  // Prevent Cobra's default error output

    // Set custom error handler
    rootCmd.SetFRErrorFunc(func(cmd *cobra.Command, err error) error {
        if err == nil {
            return nil
        }

        // Check if this looks like a verb-first pattern
        if suggestion := detectVerbFirstPattern(cmd); suggestion != "" {
            return fmt.Errorf("the verb-first form is not supported. Try: %s", suggestion)
        }

        // Default error handling
        return err
    })
}

// detectVerbFirstPattern checks if command looks like verb-first and suggests alternative
func detectVerbFirstPattern(cmd *cobra.Command) string {
    // Build command path (e.g., "add package")
    parts := []string{}
    for c := cmd; c.Parent() != nil; c = c.Parent() {
        parts = append([]string{c.Name()}, parts...)
    }
    commandPath := strings.Join(parts, " ")

    // Check against known patterns
    if suggestion, found := verbFirstPatterns[commandPath]; found {
        return suggestion
    }

    // Check if it's a top-level verb that should be under source
    firstArg := ""
    if len(parts) > 0 {
        firstArg = parts[0]
    }
    if suggestion, found := verbFirstPatterns[firstArg]; found {
        return suggestion
    }

    return ""
}
```

### Integration in `root.go`

```go
func Execute() {
    rootCmd := getRootCommand()

    // Setup custom error handler for verb-first detection
    SetupCustomErrorHandler(rootCmd)

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

---

## Testing Strategy

### 1. Reflection Tests (Policy Enforcement)

**File**: `tests/cmd/gonuget/commands/reflection_test.go`

```go
package commands_test

import (
    "strings"
    "testing"

    "github.com/yourusername/gonuget/cmd/gonuget/commands"
)

func TestCommandStructurePolicy(t *testing.T) {
    rootCmd := commands.GetRootCommand()
    validateCommand(t, rootCmd, nil)
}

func validateCommand(t *testing.T, cmd *cobra.Command, parent *cobra.Command) {
    t.Helper()

    // VR-004: Zero aliases policy
    if len(cmd.Aliases) > 0 {
        t.Errorf("Command '%s' has aliases %v - aliases are FORBIDDEN (VR-004)",
            cmd.Use, cmd.Aliases)
    }

    // VR-002: Subcommands must have verb-only Use fields
    if parent != nil && isParentCommand(parent) {
        useParts := strings.Fields(cmd.Use)
        if len(useParts) > 1 {
            // Allow args in angle brackets: "add <PACKAGE_ID>" is OK
            if !strings.HasPrefix(useParts[1], "<") {
                t.Errorf("Subcommand '%s' has multi-word Use field (VR-002) - must be verb-only",
                    cmd.Use)
            }
        }
    }

    // VR-005: Short description must start with verb (capital letter)
    if cmd.Short != "" {
        firstWord := strings.Fields(cmd.Short)[0]
        if !startsWithCapitalVerb(firstWord) {
            t.Errorf("Command '%s' Short description doesn't start with verb: '%s' (VR-005)",
                cmd.Use, cmd.Short)
        }
    }

    // Recurse to subcommands
    for _, child := range cmd.Commands() {
        validateCommand(t, child, cmd)
    }
}

func isParentCommand(cmd *cobra.Command) bool {
    // Parent commands are: package, source
    return cmd.Name() == "package" || cmd.Name() == "source"
}

func startsWithCapitalVerb(word string) bool {
    return len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z'
}
```

### 2. Golden Tests (Help Output)

**File**: `tests/cmd/gonuget/commands/golden_test.go`

```go
package commands_test

import (
    "bytes"
    "flag"
    "os"
    "path/filepath"
    "testing"

    "github.com/yourusername/gonuget/cmd/gonuget/commands"
)

var update = flag.Bool("update", false, "update golden files")

func TestHelpOutput(t *testing.T) {
    tests := []struct {
        name   string
        args   []string
        golden string
    }{
        {"root help", []string{"--help"}, "help_root.golden"},
        {"package help", []string{"package", "--help"}, "help_package.golden"},
        {"package add help", []string{"package", "add", "--help"}, "help_package_add.golden"},
        {"source help", []string{"source", "--help"}, "help_source.golden"},
        {"source list help", []string{"source", "list", "--help"}, "help_source_list.golden"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Capture output
            var buf bytes.Buffer
            rootCmd := commands.GetRootCommand()
            rootCmd.SetOut(&buf)
            rootCmd.SetArgs(tt.args)

            if err := rootCmd.Execute(); err != nil {
                t.Fatalf("command execution failed: %v", err)
            }

            goldenPath := filepath.Join("golden", tt.golden)

            if *update {
                // Update golden file
                if err := os.WriteFile(goldenPath, buf.Bytes(), 0644); err != nil {
                    t.Fatalf("failed to update golden file: %v", err)
                }
            } else {
                // Compare against golden file
                golden, err := os.ReadFile(goldenPath)
                if err != nil {
                    t.Fatalf("failed to read golden file: %v", err)
                }

                if !bytes.Equal(buf.Bytes(), golden) {
                    t.Errorf("output mismatch:\nGot:\n%s\nWant:\n%s",
                        buf.String(), string(golden))
                }
            }
        })
    }
}
```

**Running golden tests**:

```bash
# Normal test - compare against golden files
go test ./cmd/gonuget/commands -v

# Update golden files after intentional help text changes
go test ./cmd/gonuget/commands -update
```

### 3. Unit Tests (Command Logic)

**File**: `tests/cmd/gonuget/commands/package_add_test.go`

```go
package commands_test

import (
    "testing"

    "github.com/yourusername/gonuget/cmd/gonuget/commands"
)

func TestPackageAddCommand(t *testing.T) {
    tests := []struct {
        name        string
        args        []string
        wantErr     bool
        errContains string
    }{
        {
            name:    "valid add",
            args:    []string{"package", "add", "Newtonsoft.Json"},
            wantErr: false,
        },
        {
            name:        "missing package ID",
            args:        []string{"package", "add"},
            wantErr:     true,
            errContains: "requires exactly 1 arg",
        },
        {
            name:    "with version flag",
            args:    []string{"package", "add", "Serilog", "--version", "3.1.1"},
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            rootCmd := commands.GetRootCommand()
            rootCmd.SetArgs(tt.args)

            err := rootCmd.Execute()
            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
                t.Errorf("error message %q does not contain %q", err.Error(), tt.errContains)
            }
        })
    }
}
```

---

## Validation Checklist

Use this checklist when implementing or reviewing command changes:

### Command Structure

- [ ] Parent commands use single noun in `Use` field (VR-001)
- [ ] Subcommands use verb-only in `Use` field (VR-002)
- [ ] NO command has `Aliases` field set (VR-004)
- [ ] All `Short` descriptions start with verb (VR-005)
- [ ] Examples include both minimal and fully-flagged forms (VR-006)
- [ ] Subcommands registered with parent via `AddCommand()` in `init()`
- [ ] Parent commands have NO `Run` function
- [ ] Subcommands use `RunE` for error handling

### Flag Consistency

- [ ] Flag names use kebab-case (VR-007)
- [ ] Shorthands are single character (VR-008)
- [ ] `--format` flag has valid values: console, json (VR-010)
- [ ] `--verbosity` flag has valid values: quiet, minimal, normal, detailed, diagnostic (VR-011)
- [ ] Persistent flags defined on parent for inheritance

### JSON Output

- [ ] All JSON output includes `schemaVersion` field (VR-015)
- [ ] JSON goes to stdout, warnings/errors to stderr (VR-018)
- [ ] Empty results return exit code 0 with valid JSON (VR-019)
- [ ] JSON validates against schema in `contracts/json-schemas.json`

### Testing

- [ ] Reflection test passes (zero aliases, verb-only Use fields)
- [ ] Golden test exists for help output
- [ ] Unit tests cover command logic and error cases
- [ ] Integration tests verify end-to-end behavior

### Help Text

- [ ] Consistent formatting and alignment (VR-020)
- [ ] Available Commands sorted alphabetically (VR-021)
- [ ] Flags show both long and short forms (VR-022)
- [ ] Examples demonstrate real-world usage (VR-023)

---

## Common Pitfalls

### ❌ Pitfall 1: Including parent noun in subcommand Use field

**Wrong**:
```go
var addCmd = &cobra.Command{
    Use: "add package <PACKAGE_ID>",  // WRONG - includes "package"
}
```

**Correct**:
```go
var addCmd = &cobra.Command{
    Use: "add <PACKAGE_ID>",  // CORRECT - verb only
}
```

**Why**: The parent command (`package`) already provides the namespace. Including it again violates VR-002 and breaks help output.

---

### ❌ Pitfall 2: Setting Aliases field

**Wrong**:
```go
var packageCmd = &cobra.Command{
    Use:     "package",
    Aliases: []string{"pkg", "p"},  // WRONG - forbidden
}
```

**Correct**:
```go
var packageCmd = &cobra.Command{
    Use: "package",
    // NO Aliases field at all
}
```

**Why**: Constitution Principle V and VR-004 enforce zero tolerance for aliases. Reflection test will fail.

---

### ❌ Pitfall 3: Parent command with Run function

**Wrong**:
```go
var packageCmd = &cobra.Command{
    Use:   "package",
    Short: "Manage package references",
    Run: func(cmd *cobra.Command, args []string) {  // WRONG
        fmt.Println("Package command")
    },
}
```

**Correct**:
```go
var packageCmd = &cobra.Command{
    Use:   "package",
    Short: "Manage package references",
    // NO Run function - parent commands are containers only
}
```

**Why**: Parent commands provide namespace grouping. Logic belongs in subcommands.

---

### ❌ Pitfall 4: Forgetting to register subcommand with parent

**Wrong**:
```go
// package_add.go
var addCmd = &cobra.Command{
    Use: "add <PACKAGE_ID>",
    RunE: runAddPackage,
}

func init() {
    // Forgot to register with parent!
}
```

**Correct**:
```go
// package_add.go
var addCmd = &cobra.Command{
    Use: "add <PACKAGE_ID>",
    RunE: runAddPackage,
}

func init() {
    packageCmd.AddCommand(addCmd)  // REQUIRED
}
```

**Why**: Subcommands must be registered with parent to appear in help and execute correctly.

---

### ❌ Pitfall 5: Using camelCase for flags

**Wrong**:
```go
addCmd.Flags().String("configFile", "", "...")  // WRONG - camelCase
```

**Correct**:
```go
addCmd.Flags().String("configfile", "", "...")  // CORRECT - lowercase
// OR
addCmd.Flags().String("config-file", "", "...")  // CORRECT - kebab-case
```

**Why**: VR-007 requires kebab-case for multi-word flags. Constitution Principle V mandates `--configfile` specifically.

---

## Quick Commands Reference

### Build

```bash
# Build gonuget CLI
go build -o gonuget ./cmd/gonuget
```

### Test

```bash
# Run all CLI tests
go test ./cmd/gonuget/commands -v

# Run reflection tests only
go test ./cmd/gonuget/commands -run TestCommandStructurePolicy

# Run golden tests
go test ./cmd/gonuget/commands -run TestHelpOutput

# Update golden files
go test ./cmd/gonuget/commands -update

# Run with coverage
go test ./cmd/gonuget/commands -cover
```

### Manual Testing

```bash
# Test help output
./gonuget --help
./gonuget package --help
./gonuget package add --help

# Test verb-first detection
./gonuget add package Serilog  # Should error with suggestion

# Test command execution
./gonuget package add Newtonsoft.Json --version 13.0.3
./gonuget package list --format json
./gonuget source list
```

---

## Next Steps

After implementing commands following this guide:

1. **Run validation checklist** - Ensure all checkboxes pass
2. **Run all tests** - Reflection + golden + unit tests
3. **Update golden files** - If help text intentionally changed
4. **Manual testing** - Verify error messages and command execution
5. **Commit with proper message** - Use `feat(cli):` prefix (see constitution)

---

## References

- [Feature Specification](spec.md)
- [Implementation Plan](plan.md)
- [Research Findings](research.md)
- [Data Model](data-model.md)
- [Command Structure Contract](contracts/command-structure.yaml)
- [JSON Schemas](contracts/json-schemas.json)
- [Cobra Documentation](https://github.com/spf13/cobra)
- [Constitution](../../.specify/memory/constitution.md)
