# Implementation Plan: Solution File Parsing Library Refactor

**Branch**: `004-solution-lib-refactor` | **Date**: 2025-11-03 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-solution-lib-refactor/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

**Primary Requirement**: Relocate solution file parsing functionality from `cmd/gonuget/solution` to a root-level `solution/` library package, making it reusable by external Go programs. All code must be copied verbatim with only minimal changes for package relocation (import paths, file paths). The old package will be deleted after successful migration.

**Technical Approach**: This is a pure refactoring task with zero new functionality. The implementation consists of:
1. Copy all source files from `cmd/gonuget/solution/*.go` to `solution/*.go` (verbatim)
2. Update import paths in CLI commands (`package_add.go`, `package_list.go`, `package_remove.go`)
3. Delete old `cmd/gonuget/solution` directory
4. Validate with existing test suite and build checks

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: Go standard library only (encoding/xml, bufio, path/filepath, os, strings, fmt)
**Storage**: N/A (reads .sln/.slnx/.slnf files from filesystem, no persistence)
**Testing**: Go testing framework (`go test`), existing CLI test suite
**Target Platform**: Cross-platform (Linux, macOS, Windows - Go's standard build targets)
**Project Type**: Library package refactor (moving from CLI-embedded to standalone library)
**Performance Goals**: Maintain existing performance within 5% of baseline (no optimization required)
**Constraints**:
  - HARD REQUIREMENT: Verbatim copy-paste only (no logic changes)
  - Zero new dependencies allowed
  - 100% test pass rate after refactor
  - No circular dependencies introduced
**Scale/Scope**:
  - 7 source files to relocate (detector.go, types.go, sln_parser.go, slnx_parser.go, slnf_parser.go, parser.go, path.go)
  - 3 CLI command files to update (package_add.go, package_list.go, package_remove.go)
  - Existing test coverage must be preserved

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| **I. 100% NuGet.Client Parity** | ✅ **PASS** | Refactor maintains existing parity - no behavior changes allowed |
| **II. Zero-Allocation Hot Paths** | ✅ **PASS** | No hot paths affected - solution parsing is not performance-critical |
| **III. Context Propagation** | ✅ **PASS** | Existing code already follows context patterns (no changes allowed) |
| **IV. Interop Testing** | ✅ **PASS** | No interop tests needed - pure Go code refactor, CLI tests validate |
| **V. CLI Performance Target** | ✅ **PASS** | Performance must stay within 5% of baseline (SC-004) |
| **VI. Go-Idiomatic API** | ✅ **PASS** | Preserving existing idiomatic API - no changes allowed |
| **VII. Thread Safety** | ✅ **PASS** | Solution parsing is stateless - no shared state to protect |
| **VIII. Complete Implementation** | ✅ **PASS** | User confirmed: old package will be deleted (no deferral) |

**Gate Result**: ✅ **ALL GATES PASS** - Proceed to Phase 0

**Rationale**: This is a pure refactoring task with zero logic changes. All constitution principles are satisfied by maintaining existing implementation exactly as-is. The verbatim copy requirement ensures no accidental violations during relocation.

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

**Before Refactor**:
```text
cmd/gonuget/
└── solution/               # ← TO BE MOVED
    ├── detector.go         # Solution file auto-detection
    ├── types.go            # Core types (Solution, Project, SolutionFolder, etc.)
    ├── sln_parser.go       # Text-based .sln parser
    ├── slnx_parser.go      # XML-based .slnx parser
    ├── slnf_parser.go      # Solution filter (.slnf) parser
    ├── parser.go           # Parser factory and interface
    └── path.go             # Path utilities

cmd/gonuget/commands/
├── package_add.go          # ← IMPORT PATH UPDATE NEEDED
├── package_list.go         # ← IMPORT PATH UPDATE NEEDED
└── package_remove.go       # ← IMPORT PATH UPDATE NEEDED
```

**After Refactor**:
```text
solution/                   # ← NEW LIBRARY PACKAGE (root level)
├── detector.go             # (copied verbatim)
├── types.go                # (copied verbatim)
├── sln_parser.go           # (copied verbatim)
├── slnx_parser.go          # (copied verbatim)
├── slnf_parser.go          # (copied verbatim)
├── parser.go               # (copied verbatim)
└── path.go                 # (copied verbatim)

cmd/gonuget/commands/
├── package_add.go          # Import: github.com/willibrandon/gonuget/solution
├── package_list.go         # Import: github.com/willibrandon/gonuget/solution
└── package_remove.go       # Import: github.com/willibrandon/gonuget/solution

# Old directory deleted:
# cmd/gonuget/solution/     # ← DELETED after migration complete
```

**Structure Decision**: Gonuget uses a single-project structure with library packages at root level (`version/`, `frameworks/`, `protocol/`, etc.) and CLI under `cmd/gonuget/`. The solution package follows this established pattern by moving to `solution/` at repository root, making it accessible to external Go programs via standard Go module imports.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**N/A** - No constitution violations. All gates pass.
