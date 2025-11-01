# Implementation Plan: CLI Command Structure Restructure

**Branch**: `001-cli-command-restructure` | **Date**: 2025-10-31 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-cli-command-restructure/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Restructure gonuget CLI to adopt modern noun-first command hierarchy matching dotnet CLI standards. This involves reorganizing commands from mixed verb-first structure (gonuget add package) to consistent noun-first namespaces (gonuget package add, gonuget source add) with zero tolerance for aliases. The restructure maintains 100% functional parity while improving discoverability, consistency with modern CLI tools, and migration user experience through helpful error messages.

## Technical Context

**Language/Version**: Go 1.21+ (existing gonuget project)
**Primary Dependencies**: Cobra CLI framework, existing gonuget core packages (version, frameworks, protocol, packaging, resolver)
**Storage**: N/A (command structure refactoring only)
**Testing**: Go testing (table-driven tests, golden tests for help output, reflection tests for policy enforcement)
**Target Platform**: Cross-platform CLI (Linux, macOS, Windows)
**Project Type**: Single project (CLI tool restructure within existing gonuget codebase)
**Performance Goals**: Command execution <100ms, help output <50ms, error detection <50ms (matching Constitution Principle V)
**Constraints**: Zero aliases policy (hard requirement), 100% dotnet CLI parity for command structure, no breaking changes to underlying functionality
**Scale/Scope**: 5 top-level commands (config, package, restore, source, version), 10 subcommands total (package: add/list/remove/search, source: add/disable/enable/list/remove/update)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: 100% NuGet.Client Parity (NON-NEGOTIABLE)
**Status**: ✅ **PASS**
- This feature restructures CLI command hierarchy WITHOUT changing underlying functionality
- All NuGet protocol implementations remain unchanged
- Existing interop tests continue to validate behavioral parity
- Command structure change is purely presentational/UX - does not affect NuGet.Client compatibility

### Principle V: CLI Performance Target & Structure
**Status**: ✅ **PASS** (Direct alignment with constitution)
- Feature DIRECTLY implements command structure requirements from Principle V
- Implements mandatory `gonuget package` namespace (matches `dotnet package`)
- Implements mandatory `gonuget source` namespace (matches `dotnet nuget source`)
- Enforces "NO mixed command hierarchies" rule
- Maintains top-level commands: config, restore, version (matches dotnet)
- Performance targets preserved: <1/15th dotnet execution time, 30-35% less memory
- Flag naming uses kebab-case per constitution
- Output format/help text/error messages match dotnet patterns

### Principle VI: Go-Idiomatic API
**Status**: ✅ **PASS**
- Cobra framework usage follows Go CLI conventions
- Command registration uses standard Cobra patterns
- Error handling remains explicit (no exceptions)
- Reflection tests enforce policy (zero aliases) in Go-idiomatic way

### Principle IV: Interop Testing (Quality Gate)
**Status**: ✅ **PASS**
- All 491 interop tests remain valid (CLI structure change doesn't affect library behavior)
- New CLI tests will validate command structure via golden tests and reflection tests
- No changes to library code that would invalidate existing interop tests

### Code Quality Standards
**Status**: ✅ **PASS**
- New tests required: golden tests (help output), reflection tests (Use fields, aliases policy)
- Commit conventions will use `feat(cli):` prefix per constitution guidelines
- No "Chunk" references or AI attribution in commit messages

**Gate Outcome**: ✅ **ALL GATES PASSED** - Proceed to Phase 0 research

---

### Phase 1 Re-Evaluation (Post-Design)

**Status**: ✅ **ALL GATES STILL PASS**

After completing Phase 1 design artifacts (research.md, data-model.md, contracts/, quickstart.md), all constitution checks remain valid:

- ✅ **Principle I (NuGet.Client Parity)**: Design confirms no changes to library behavior
- ✅ **Principle V (CLI Structure & Performance)**: Contracts validate exact alignment with constitution command structure requirements
- ✅ **Principle VI (Go Idioms)**: Quickstart guide demonstrates Go-idiomatic Cobra patterns
- ✅ **Principle IV (Interop Testing)**: Design confirms existing 491 tests remain valid
- ✅ **Code Quality**: Testing strategy includes reflection tests, golden tests, and unit tests per standards

**Design Validation**:
- Data model includes 23 validation rules (VR-001 to VR-023) enforcing all constitution requirements
- Contracts define exact command structure matching Principle V lines 82-96
- Research resolved all design decisions with zero "NEEDS CLARIFICATION" remaining
- Quickstart provides implementation patterns following Go best practices

**No new violations introduced** - Design maintains full constitution compliance.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/gonuget/
├── main.go                      # Entry point - register parent + top-level commands
├── commands/
│   ├── package.go               # NEW: Parent command for package namespace
│   ├── package_add.go           # RENAMED: from add_package.go
│   ├── package_list.go          # NEW: List packages command
│   ├── package_remove.go        # NEW: Remove package command
│   ├── package_search.go        # NEW: Search packages command
│   ├── source.go                # NEW: Parent command for source namespace
│   ├── source_add.go            # RENAMED: from add_source.go (+ restructure)
│   ├── source_list.go           # RENAMED: from list.go (+ restructure)
│   ├── source_remove.go         # RENAMED: from remove.go (+ restructure)
│   ├── source_enable.go         # RENAMED: from enable.go (+ restructure)
│   ├── source_disable.go        # RENAMED: from disable.go (+ restructure)
│   ├── source_update.go         # RENAMED: from update.go (+ restructure)
│   ├── restore.go               # UNCHANGED: top-level command
│   ├── config.go                # UNCHANGED: top-level command
│   ├── version.go               # UNCHANGED: top-level command
│   ├── root.go                  # MODIFIED: register only parent + top-level commands
│   └── errors.go                # NEW: Custom error handler for verb-first detection
├── config/                      # Existing NuGet.config handling
├── output/                      # Existing console output
└── project/                     # Existing .csproj manipulation

tests/
└── cmd/gonuget/
    └── commands/
        ├── package_test.go          # NEW: Package parent command tests
        ├── package_add_test.go      # MODIFIED: Update for new structure
        ├── package_list_test.go     # NEW: List command tests
        ├── package_remove_test.go   # NEW: Remove command tests
        ├── package_search_test.go   # NEW: Search command tests
        ├── source_test.go           # NEW: Source parent command tests
        ├── source_*_test.go         # MODIFIED: Update for new structure
        ├── golden/                  # NEW: Golden test fixtures for help output
        │   ├── help_root.golden
        │   ├── help_package.golden
        │   ├── help_source.golden
        │   └── ...
        └── reflection_test.go       # NEW: Enforce Use fields + zero aliases policy
```

**Structure Decision**: Single project structure (Option 1) with CLI-focused restructure. The existing gonuget CLI codebase uses `cmd/gonuget/` for CLI implementation. This feature reorganizes the `commands/` directory to adopt noun-first hierarchy while preserving existing infrastructure (config/, output/, project/ packages).

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**No violations detected** - All constitution checks passed. This feature directly implements requirements from Constitution Principle V and aligns with all other principles.

---

## Phase 0 & Phase 1 Completion Summary

**Status**: ✅ **PLANNING COMPLETE** - Ready for `/speckit.tasks`

### Artifacts Generated

**Phase 0 (Research)**:
- ✅ `research.md` - 7 design decisions documented with rationale and alternatives
  - Noun-first only (zero aliases)
  - Top-level exceptions (restore, config, version)
  - Cobra parent commands
  - Verb-first error detection
  - Golden tests for help output
  - Reflection tests for policy enforcement
  - JSON schema versioning

**Phase 1 (Design & Contracts)**:
- ✅ `data-model.md` - 5 core entities with 23 validation rules (VR-001 to VR-023)
  - Command Hierarchy (parent/child relationships)
  - Command Flags (kebab-case, valid values)
  - Error Patterns (10 verb-first patterns to detect)
  - JSON Output Schemas (package list, package search, source list)
  - Help Text Templates (consistent formatting)

- ✅ `contracts/command-structure.yaml` - Complete command specification
  - Root command with 5 top-level commands
  - Package parent with 4 subcommands (add, list, remove, search)
  - Source parent with 6 subcommands (add, disable, enable, list, remove, update)
  - All flags, args, validation rules, examples
  - Error pattern detection rules
  - Exit codes

- ✅ `contracts/json-schemas.json` - JSON Schema definitions
  - Package list output schema
  - Package search output schema
  - Source list output schema
  - Definitions for packageReference, searchResult, packageSource
  - Examples for all schemas including empty results

- ✅ `quickstart.md` - Developer implementation guide
  - Command implementation patterns (parent, subcommand, top-level)
  - Error handler pattern for verb-first detection
  - Testing strategy (reflection, golden, unit tests)
  - Validation checklist (28 checkboxes)
  - Common pitfalls with correct alternatives
  - Quick commands reference

- ✅ `CLAUDE.md` updated - Agent context synchronized
  - Added Go 1.21+ technology
  - Added Cobra framework
  - Added existing gonuget packages reference
  - Added database: N/A (refactoring only)

### Constitution Compliance

**Initial Gate Check**: ✅ ALL GATES PASSED
**Post-Design Re-evaluation**: ✅ ALL GATES STILL PASS

No violations introduced during design phase. All artifacts align with:
- Principle I: 100% NuGet.Client Parity (no library changes)
- Principle V: CLI Performance Target & Structure (exact match)
- Principle VI: Go-Idiomatic API (Cobra patterns validated)
- Principle IV: Interop Testing (existing tests remain valid)

### Design Highlights

**Command Structure**:
- 5 top-level commands (config, package, restore, source, version)
- 2 parent commands (package, source)
- 10 subcommands total (4 under package, 6 under source)
- 0 aliases (zero tolerance policy)
- 10 verb-first error patterns detected

**Validation Rules**: 23 rules (VR-001 to VR-023) covering:
- Command structure (parent/subcommand Use fields)
- Flag naming (kebab-case, valid values)
- Error detection (performance, messaging)
- JSON output (schema versioning, stdout/stderr separation)
- Help text (formatting, consistency)

**Testing Strategy**:
- Reflection tests: Enforce zero aliases and verb-only Use fields
- Golden tests: Snapshot validation for all help outputs
- Unit tests: Command logic and error scenarios
- Integration tests: End-to-end command execution

### Next Steps

**Command Output**: `/speckit.plan` execution complete. Use `/speckit.tasks` to generate actionable task list.

**Files Ready for Implementation**:
1. Research findings in `research.md`
2. Data model with validation rules in `data-model.md`
3. API contracts in `contracts/*.yaml` and `contracts/*.json`
4. Developer guide in `quickstart.md`
5. Implementation plan in `plan.md` (this file)

**Branch**: `001-cli-command-restructure`
**Plan Path**: `specs/001-cli-command-restructure/plan.md`
