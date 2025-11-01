# Research: CLI Command Structure Restructure

**Feature**: CLI Command Structure Restructure
**Branch**: `001-cli-command-restructure`
**Date**: 2025-10-31

## Overview

This document captures research findings and design decisions for restructuring gonuget's CLI command hierarchy from mixed verb-first structure to consistent noun-first namespaces matching modern CLI standards (dotnet, kubectl, docker, aws).

## Key Design Decisions

### Decision 1: Noun-First Only (Zero Aliases)

**Decision**: Implement strict noun-first command structure with ZERO tolerance for verb-first aliases.

**Rationale**:
- **Principle V Mandate**: Constitution explicitly requires `gonuget package` and `gonuget source` namespaces
- **dotnet CLI Parity**: Modern dotnet uses `dotnet package add` (not `dotnet add package`)
- **Consistency with Industry**: kubectl, docker, aws all use noun-first (`kubectl get pod`, `docker container run`)
- **Migration Clarity**: Allowing aliases creates confusion about "correct" form and delays learning
- **Constitution Alignment**: "NO mixed command hierarchies" rule prohibits mixed patterns

**Alternatives Considered**:
1. **Temporary aliases during transition** - Rejected because:
   - Creates technical debt
   - Delays user adoption of new patterns
   - Violates constitution's zero tolerance policy
   - Users need clear "one way" to run commands

2. **Keep verb-first for backward compatibility** - Rejected because:
   - Perpetuates non-standard CLI patterns
   - Contradicts Constitution Principle V requirements
   - gonuget is early enough in lifecycle to make breaking changes
   - Constitution prioritizes long-term correctness over short-term convenience

**Supporting Evidence**:
- Constitution Principle V lines 82-96: Explicit command structure requirements
- Industry standards: kubectl 1.0 (2015), docker CLI (2013), aws CLI v2 (2020) all noun-first
- dotnet CLI evolution: Moved from mixed patterns to consistent noun-first in .NET 5+

---

### Decision 2: Top-Level Exceptions (restore, config, version)

**Decision**: Keep `restore`, `config`, and `version` as top-level commands (not under namespaces).

**Rationale**:
- **dotnet CLI Parity**: dotnet has `dotnet restore`, `dotnet --version` at top level
- **Frequency of Use**: `restore` is the most common operation - shorter is better
- **Semantic Clarity**: These are global operations, not scoped to packages or sources
- **Constitution Mandate**: Principle V line 95 explicitly requires this structure

**Alternatives Considered**:
1. **Pure noun-first with gonuget project restore** - Rejected because:
   - Breaks dotnet CLI parity (constitution requirement)
   - Makes most common operation longer
   - "project" namespace not semantically valuable (gonuget always operates on projects)

2. **gonuget tool restore / gonuget tool config** - Rejected because:
   - No precedent in dotnet CLI
   - Adds unnecessary nesting for global operations
   - "tool" namespace ambiguous (could mean NuGet tool packages)

**Supporting Evidence**:
- dotnet CLI: `dotnet restore`, `dotnet --version`, `dotnet nuget add source`
- Constitution Principle V line 95: "Top-level commands: restore, config, version"

---

### Decision 3: Cobra Parent Commands for Namespaces

**Decision**: Use Cobra parent commands (`package.go`, `source.go`) with subcommands registered via `AddCommand()`.

**Rationale**:
- **Cobra Best Practice**: Parent commands provide namespace grouping with automatic help generation
- **Subcommand Discoverability**: `gonuget package --help` shows all package operations
- **Clean Help Output**: Root help shows 5 top-level items instead of 15 flat commands
- **Go Idioms**: Cobra's parent/child pattern is Go-idiomatic for CLI hierarchies

**Implementation Pattern**:
```go
// package.go - Parent command
var packageCmd = &cobra.Command{
    Use:   "package",
    Short: "Manage package references",
    Long:  "...",
}

// package_add.go - Subcommand
var addCmd = &cobra.Command{
    Use:   "add <PACKAGE_ID>",  // Verb-only in Use field
    Short: "Add a package reference",
    Run:   runAddPackage,
}

func init() {
    packageCmd.AddCommand(addCmd)  // Register subcommand
}
```

**Alternatives Considered**:
1. **Flat command structure with prefixed names** - Rejected because:
   - Loses namespace grouping in help output
   - Harder to discover related commands
   - Doesn't leverage Cobra's parent command features

2. **Custom command routing** - Rejected because:
   - Reinvents Cobra's built-in functionality
   - More complex to test and maintain
   - Non-idiomatic Go/Cobra usage

---

### Decision 4: Verb-First Error Detection

**Decision**: Implement custom error handler to detect verb-first patterns and suggest noun-first alternatives.

**Rationale**:
- **Migration UX**: Users familiar with old syntax get immediate, helpful feedback
- **Learning Acceleration**: Clear suggestions reduce trial-and-error learning time
- **Constitution Requirement**: FR-007, FR-008 mandate helpful error messages for migration
- **Cobra Integration**: `SilenceErrors = true` + custom handler allows pattern inspection before error display

**Implementation Pattern**:
```go
// errors.go
func customErrorHandler(cmd *cobra.Command, err error) error {
    if err == nil {
        return nil
    }

    // Detect verb-first patterns (add package, list source, etc.)
    if isVerbFirstPattern(cmd) {
        return fmt.Errorf("the verb-first form is not supported. Try: %s",
            suggestNounFirst(cmd))
    }

    return err  // Default error handling
}

// Patterns to detect:
// - gonuget add package -> suggest: gonuget package add
// - gonuget list source -> suggest: gonuget source list
// - gonuget enable nuget.org -> suggest: gonuget source enable
```

**Alternatives Considered**:
1. **Generic "unknown command" errors** - Rejected because:
   - Poor migration UX - users don't learn correct syntax
   - Violates FR-008 requirement for specific suggestions
   - Increases support burden

2. **Alias redirects with deprecation warnings** - Rejected because:
   - Violates zero aliases policy (constitution + FR-006)
   - Creates technical debt
   - Users rely on deprecated form instead of learning new syntax

---

### Decision 5: Golden Tests for Help Output

**Decision**: Use golden file testing to validate help text format, command structure, and flag consistency.

**Rationale**:
- **Snapshot Validation**: Catches unintended help text changes across refactors
- **Constitution Compliance**: Validates FR-024 through FR-028 (help text requirements)
- **Regression Prevention**: Ensures help output remains stable and consistent
- **dotnet Parity Verification**: Golden files can be diffed against dotnet help output

**Implementation Pattern**:
```go
// golden_test.go
func TestHelpOutput(t *testing.T) {
    tests := []struct {
        name    string
        args    []string
        golden  string
    }{
        {"root help", []string{"--help"}, "help_root.golden"},
        {"package help", []string{"package", "--help"}, "help_package.golden"},
        {"source help", []string{"source", "--help"}, "help_source.golden"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            output := captureOutput(tt.args)
            compareGolden(t, tt.golden, output)
        })
    }
}
```

**Alternatives Considered**:
1. **Manual help text inspection** - Rejected because:
   - Error-prone for subtle formatting changes
   - Doesn't scale as commands grow
   - Hard to catch regressions

2. **Regex-based validation** - Rejected because:
   - Fragile - breaks on minor formatting changes
   - Doesn't validate full output structure
   - Harder to debug failures

---

### Decision 6: Reflection Tests for Policy Enforcement

**Decision**: Use Go reflection to programmatically validate Use fields are verb-only and Aliases fields are empty.

**Rationale**:
- **Zero Tolerance Enforcement**: Catches policy violations at test time (FR-005, FR-006)
- **Automated Validation**: No manual review needed for every command addition
- **Constitution Compliance**: Enforces "hard policy: zero tolerance for aliases"
- **Fail Fast**: Tests fail immediately if new command violates structure rules

**Implementation Pattern**:
```go
// reflection_test.go
func TestCommandStructurePolicy(t *testing.T) {
    rootCmd := getRootCommand()

    // Validate all commands recursively
    validateCommand(t, rootCmd)
}

func validateCommand(t *testing.T, cmd *cobra.Command) {
    // FR-006: Zero aliases policy
    if len(cmd.Aliases) > 0 {
        t.Errorf("Command %s has aliases %v - aliases are forbidden",
            cmd.Use, cmd.Aliases)
    }

    // FR-005: Subcommands must have verb-only Use fields
    if cmd.Parent() != nil && isParentCommand(cmd.Parent()) {
        if containsSpace(cmd.Use) {
            t.Errorf("Subcommand %s has multi-word Use field - must be verb-only",
                cmd.Use)
        }
    }

    // Recurse to subcommands
    for _, child := range cmd.Commands() {
        validateCommand(t, child)
    }
}
```

**Alternatives Considered**:
1. **Manual code review** - Rejected because:
   - Human error likely
   - Doesn't scale
   - No automated regression prevention

2. **Linter custom rule** - Rejected because:
   - More complex to implement than reflection test
   - Requires linter infrastructure setup
   - Reflection test is Go-idiomatic and sufficient

---

### Decision 7: JSON Schema Versioning

**Decision**: All JSON output includes `schemaVersion` field for stability and backward compatibility.

**Rationale**:
- **API Contract Stability**: Consumers can detect breaking changes via version field
- **Constitution Requirement**: FR-020 mandates schemaVersion in all JSON output
- **Automation-Friendly**: Scripts can validate against expected schema version
- **Future-Proof**: Enables schema evolution without breaking existing consumers

**Schema Structure**:
```json
{
  "schemaVersion": "1.0.0",
  "packages": [...],
  "warnings": [],
  "elapsedMs": 42
}
```

**Alternatives Considered**:
1. **Unversioned JSON output** - Rejected because:
   - Breaking changes undetectable by consumers
   - Violates FR-020 requirement
   - Poor experience for automation scripts

2. **API version in URL/header** - Rejected because:
   - CLI doesn't use HTTP (no headers/URLs)
   - schemaVersion in payload is standard for CLI JSON output

---

## Technology Stack Validation

### Cobra CLI Framework

**Usage**: Existing dependency - no changes needed.

**Validation**:
- ✅ Supports parent commands with subcommands
- ✅ Automatic help generation
- ✅ Flag inheritance from parent to child commands
- ✅ Custom error handling via `SilenceErrors` + handler
- ✅ Shell completion generation (bash, zsh, PowerShell)

**References**:
- Cobra documentation: https://github.com/spf13/cobra
- Existing gonuget usage: `cmd/gonuget/commands/root.go`

---

### Go Testing Framework

**Usage**: Standard library `testing` package with golden file pattern.

**Validation**:
- ✅ Table-driven tests for parameterized validation
- ✅ Reflection API for command structure inspection
- ✅ `t.Helper()` for test helper functions
- ✅ `-update` flag pattern for golden file updates

**Golden Test Pattern**:
```bash
# Normal test - compare against golden files
go test ./cmd/gonuget/commands -v

# Update golden files after intentional help text changes
go test ./cmd/gonuget/commands -update
```

---

## Testing Strategy

### Test Categories

1. **Unit Tests** (`*_test.go`):
   - Command constructor validation
   - Flag registration
   - Argument parsing
   - Error message formatting

2. **Golden Tests** (`golden_test.go`):
   - Help output for all commands
   - Error message format
   - Shell completion output

3. **Reflection Tests** (`reflection_test.go`):
   - Zero aliases policy
   - Verb-only Use fields for subcommands
   - Parent command structure

4. **Integration Tests** (existing):
   - End-to-end command execution
   - NuGet.config modification
   - Project file updates

### Test Coverage Targets

- **Command structure**: 100% (critical path - reflection tests enforce policy)
- **Help text**: 100% (golden tests cover all commands)
- **Error handling**: 90%+ (table-driven tests for all error scenarios)
- **Overall CLI package**: 85%+ (per existing gonuget standards)

---

## Risk Mitigation

### Risk 1: Breaking Change for Existing Users

**Mitigation**:
- Clear error messages with exact new syntax when verb-first attempted
- Documentation update with migration guide
- gonuget is early in lifecycle (acceptable breaking change window)

### Risk 2: Help Text Maintenance Burden

**Mitigation**:
- Golden tests catch unintended changes
- `-update` flag simplifies intentional updates
- Help text follows standard template pattern

### Risk 3: Test Complexity

**Mitigation**:
- Reflection tests use simple recursive validation pattern
- Golden tests are straightforward snapshot comparisons
- Both patterns are well-documented in Go community

---

## Open Questions

**None** - All design decisions are resolved. Technical context is complete with zero "NEEDS CLARIFICATION" markers.

---

## References

- Feature Specification: [spec.md](spec.md)
- Constitution Principle V: `.specify/memory/constitution.md` lines 78-109
- Cobra Documentation: https://github.com/spf13/cobra
- dotnet CLI Reference: https://docs.microsoft.com/en-us/dotnet/core/tools/
- Golden Testing Pattern: https://ieftimov.com/posts/testing-in-go-golden-files/
