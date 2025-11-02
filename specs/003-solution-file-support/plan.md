# Implementation Plan: Solution File Support for gonuget CLI

**Branch**: `003-solution-file-support` | **Date**: 2025-11-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-solution-file-support/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add support for .NET solution files (.sln, .slnx, .slnf) to the gonuget CLI, enabling developers to list packages across multiple projects while rejecting invalid operations with exact dotnet CLI error messages. The implementation will parse all three solution file formats, enumerate projects, and integrate with existing gonuget package commands to achieve 100% parity with dotnet CLI behavior.

## Technical Context

**Language/Version**: Go 1.21+ (existing gonuget project)
**Primary Dependencies**: Cobra (CLI framework), existing gonuget core packages (version, frameworks, protocol, packaging, resolver)
**Storage**: File system (read-only access to .sln, .slnx, .slnf files, read-write for project files via existing project package)
**Testing**: Go standard testing package, integration tests matching dotnet CLI output
**Target Platform**: Cross-platform CLI (Windows, macOS, Linux)
**Project Type**: CLI extension to existing gonuget project
**Performance Goals**: Parse 10-project solution in < 1 second, 100+ projects in < 30 seconds
**Constraints**: Character-for-character output parity with dotnet CLI, exact error message matching
**Scale/Scope**: Support solutions with up to 1000+ projects (typical enterprise scale)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: 100% NuGet.Client Parity ✅ PASS
- Feature explicitly requires character-for-character output matching with dotnet CLI
- Error messages must match exactly
- Behavioral parity is the core requirement

### Principle II: Zero-Allocation Hot Paths ⚠️ NOT APPLICABLE
- Solution file parsing is not a hot path (one-time operation)
- Performance targets are reasonable (< 1s for 10 projects)
- No tight loops or repeated operations

### Principle III: Context Propagation ✅ PASS
- Package list operations will accept context.Context
- File I/O operations will be cancellable
- Follows existing gonuget patterns

### Principle IV: Interop Testing ✅ PASS
- Integration tests will validate exact output matching with dotnet CLI
- Error message parity tests required
- Cross-platform path handling tests needed

### Principle V: CLI Performance & Structure ✅ PASS
- Maintains noun-first command pattern (package list, package add, package remove)
- Performance targets align with 15-17x improvement goal
- Command structure unchanged, only behavior enhanced

### Principle VI: Go-Idiomatic API ✅ PASS
- Will follow Go conventions for error handling
- Package structure follows Go standards
- Interface-based design for parser extensibility

### Principle VII: Thread Safety ⚠️ NOT APPLICABLE
- Solution parsing is single-threaded operation
- No shared state between concurrent operations
- Each command invocation is isolated

### Principle VIII: Complete Implementation ✅ PASS
- All three formats (.sln, .slnx, .slnf) will be fully implemented
- No deferrals or "future" markers
- Complete feature as specified

**GATE RESULT**: ✅ ALL APPLICABLE PRINCIPLES PASS

## Post-Design Constitution Re-Check (Phase 1 Complete)

After completing the research and design phase, re-validating all principles:

### Principle I: 100% NuGet.Client Parity ✅ STILL PASS
- Contracts specify exact error message matching
- Output format contracts ensure character-for-character parity
- Integration test strategy validates against actual dotnet CLI

### Principle III: Context Propagation ✅ STILL PASS
- Parser interface design allows context propagation in future
- Command integration maintains context flow
- File operations designed to be cancellable

### Principle IV: Interop Testing ✅ STILL PASS
- Testing strategy includes byte-for-byte comparison with dotnet
- Error message validation tests specified
- Cross-platform tests defined

### Principle V: CLI Performance & Structure ✅ STILL PASS
- Commands remain noun-first (package list, not list package)
- Performance targets maintained (< 1s for 10 projects)
- No structural changes to command hierarchy

### Principle VI: Go-Idiomatic API ✅ STILL PASS
- Interface-based design for parsers
- Error types follow Go conventions
- Package structure follows Go standards

### Principle VIII: Complete Implementation ✅ STILL PASS
- All three formats have concrete parser implementations planned
- No deferrals in the design
- Complete feature implementation defined

**FINAL GATE RESULT**: ✅ ALL APPLICABLE PRINCIPLES STILL PASS

## Project Structure

### Documentation (this feature)

```text
specs/003-solution-file-support/
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
├── solution/                # NEW: Solution file parsing package
│   ├── parser.go            # Solution file parser interface
│   ├── sln_parser.go        # .sln format parser (text-based MSBuild format)
│   ├── slnx_parser.go       # .slnx format parser (XML-based)
│   ├── slnf_parser.go       # .slnf format parser (JSON filter)
│   ├── detector.go          # Solution file detection by extension
│   ├── types.go             # Solution, Project, SolutionFolder types
│   └── path.go              # Cross-platform path resolution utilities
│
├── commands/                # EXISTING: CLI commands
│   ├── package_list.go      # MODIFY: Add solution file support
│   ├── package_add.go       # MODIFY: Add solution file rejection
│   ├── package_remove.go    # MODIFY: Add solution file rejection
│   └── errors.go            # MODIFY: Add dotnet-compatible error messages
│
└── output/                  # EXISTING: Output formatting
    └── package_list.go      # MODIFY: Format multi-project output

tests/cmd/gonuget/
├── solution/                # NEW: Solution parser tests
│   ├── sln_parser_test.go
│   ├── slnx_parser_test.go
│   ├── slnf_parser_test.go
│   ├── detector_test.go
│   ├── path_test.go
│   └── testdata/            # Test solution files
│       ├── simple.sln
│       ├── large.sln
│       ├── malformed.sln
│       ├── simple.slnx
│       └── filter.slnf
│
└── commands/                # EXISTING: Command tests
    ├── package_list_solution_test.go  # NEW: Solution support tests
    └── package_errors_test.go         # NEW: Error message parity tests
```

**Structure Decision**: Extending the existing gonuget CLI structure by adding a new `solution` package under `cmd/gonuget/` for all solution file parsing logic. This keeps solution-specific code isolated while allowing easy integration with existing commands. The modular design with separate parsers for each format ensures maintainability and testability.

## Complexity Tracking

> No violations to justify - all applicable Constitution principles pass.