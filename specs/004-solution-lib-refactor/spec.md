# Feature Specification: Solution File Parsing Library Refactor

**Feature Branch**: `004-solution-lib-refactor`
**Created**: 2025-11-03
**Status**: Draft
**Input**: User description: "Refactor solution file parsing (`.sln` and `.slnx`) from `cmd/gonuget/solution` package to a new library package `solution/` at the gonuget root level, making it reusable by external programs. **HARD REQUIREMENT**: All code, tests, documentation, and logic must be copied verbatim (copy-paste) from the existing implementation - do NOT rewrite, refactor, or modify any logic. The ONLY changes allowed are those strictly necessary for the package relocation: package declarations, import paths, and file paths. All other code (algorithms, error handling, variable names, comments, test cases, benchmarks) must remain byte-for-byte identical. Must preserve all existing functionality: auto-detection, parallel project processing, both format parsers (text-based .sln and XML .slnx), and solution folder filtering. Update CLI code to import and use the new library package. Ensure zero breaking changes to existing CLI behavior and maintain all test coverage."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - External Tool Integration (Priority: P1)

A developer building a third-party Go tool (e.g., project analyzer, build orchestrator, or IDE plugin) needs to parse .NET solution files to discover project references. They can import the `solution` package from gonuget as a library dependency and use its parsers directly.

**Why this priority**: This is the primary value proposition - making solution file parsing available as a reusable library. Without this, the feature has no purpose.

**Independent Test**: Can be fully tested by creating a minimal Go program that imports `github.com/willibrandon/gonuget/solution`, calls `solution.GetParser()`, and successfully parses a .sln file. Delivers immediate value by enabling external tool development.

**Acceptance Scenarios**:

1. **Given** a Go program outside the gonuget CLI codebase, **When** it imports `github.com/willibrandon/gonuget/solution` and calls parser functions, **Then** the import succeeds and all public APIs are accessible
2. **Given** an external tool using the solution library, **When** it parses a .sln file using the library's public API, **Then** it receives the same parsed data structure that the CLI would receive
3. **Given** a developer browsing pkg.go.dev documentation, **When** they search for gonuget solution parsing, **Then** they find the `solution` package with complete API documentation

---

### User Story 2 - CLI Backward Compatibility (Priority: P1)

An existing gonuget CLI user runs package commands (`package add`, `package list`, `package remove`) that operate on solution files. After the refactor, these commands continue to work identically with no behavior changes, errors, or performance degradation.

**Why this priority**: Equally critical as P1 - breaking existing CLI functionality would violate the zero-breaking-changes requirement. Both library availability and CLI stability are mandatory for feature success.

**Independent Test**: Can be fully tested by running the existing CLI test suite against the refactored code. All tests must pass without modification. Delivers value by ensuring production stability.

**Acceptance Scenarios**:

1. **Given** a project with a .sln file, **When** a user runs `gonuget package list --solution MySolution.sln`, **Then** the command produces identical output to the pre-refactor version
2. **Given** the gonuget CLI test suite, **When** tests run against the refactored code, **Then** 100% of existing tests pass without modification
3. **Given** CLI commands that parse solution files, **When** they execute after the refactor, **Then** performance (execution time and memory usage) remains within 5% of pre-refactor measurements

---

### User Story 3 - Library Documentation and Examples (Priority: P2)

A developer discovers the solution parsing library through pkg.go.dev or GitHub. They can read package documentation, understand the API surface, and find example code showing how to parse .sln, .slnx, and .slnf files.

**Why this priority**: Important for adoption but not blocking - the library works without perfect documentation. Can be improved iteratively after the core refactor is complete.

**Independent Test**: Can be tested by checking that godoc comments exist for all exported types and functions, and that at least one runnable example appears in pkg.go.dev. Delivers value by reducing adoption friction.

**Acceptance Scenarios**:

1. **Given** the refactored solution package, **When** godoc processes the code, **Then** all exported types, functions, and constants have documentation comments
2. **Given** a developer reading pkg.go.dev, **When** they view the solution package page, **Then** they see working code examples for common use cases (parsing .sln, auto-detection, parallel processing)
3. **Given** the README or package documentation, **When** a developer reads it, **Then** they understand the difference between .sln, .slnx, and .slnf formats and know which parser to use

---

### Edge Cases

- What happens when CLI code imports the old `cmd/gonuget/solution` path after the refactor? (Compile-time error expected - import paths must be updated)
- How does the system handle existing third-party tools that might have vendored the old CLI-embedded solution code? (They continue working - we don't control external vendors)
- What happens if package-level constants or unexported helpers are used internally by the CLI? (They must remain accessible after relocation)
- How does the refactor affect test coverage metrics? (Coverage percentage must remain identical or higher)
- What happens if the solution package has circular dependencies with other gonuget packages? (Refactor must not introduce cycles - validate with `go build`)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The solution parsing library MUST be relocated from `cmd/gonuget/solution` to `solution/` at the repository root level
- **FR-002**: All code (algorithms, error handling, variable names, comments) MUST be copied verbatim with zero logic changes
- **FR-003**: Package declarations MUST be updated from `package solution` to `package solution` (remains same - already correct)
- **FR-004**: Import paths in CLI code MUST be updated from `github.com/willibrandon/gonuget/cmd/gonuget/solution` to `github.com/willibrandon/gonuget/solution`
- **FR-005**: All existing functionality MUST be preserved: auto-detection (`Detector`), parallel project processing (`ProcessProjectsInParallel`), .sln parser, .slnx parser, .slnf parser, solution folder filtering
- **FR-006**: All exported types, functions, and methods MUST maintain identical signatures and behavior
- **FR-007**: The refactored library MUST be importable by external Go programs without requiring the gonuget CLI
- **FR-008**: CLI commands (`package add`, `package list`, `package remove`) MUST continue functioning with zero behavior changes
- **FR-009**: All existing test coverage MUST be preserved (tests relocated and import paths updated)
- **FR-010**: No new dependencies MUST be introduced during the refactor
- **FR-011**: The refactor MUST not break existing builds or introduce circular dependencies
- **FR-012**: Documentation comments (if present) MUST be preserved byte-for-byte
- **FR-013**: Any file paths hardcoded in the original code MUST be updated to work from the new package location
- **FR-014**: The solution package MUST remain self-contained with no CLI-specific dependencies

### Key Entities *(include if feature involves data)*

- **Solution Package**: The library package at `solution/` containing all solution file parsing logic
- **CLI Integration**: Updated import paths in `cmd/gonuget/commands/package_add.go`, `package_list.go`, and `package_remove.go`
- **File Artifacts**: Source files (detector.go, types.go, sln_parser.go, slnx_parser.go, slnf_parser.go, parser.go, path.go), test files, and any documentation files
- **Import Graph**: Dependency relationships between the solution package, CLI commands, and other gonuget library packages

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: External Go programs can successfully import and use `github.com/willibrandon/gonuget/solution` as a library dependency
- **SC-002**: 100% of existing CLI tests pass without modification after updating import paths
- **SC-003**: Test coverage percentage for solution parsing logic remains identical or increases (no coverage loss)
- **SC-004**: CLI command execution time for solution-related operations remains within 5% of baseline measurements
- **SC-005**: `go build ./...` succeeds with zero errors and zero new warnings
- **SC-006**: All exported APIs from the original `cmd/gonuget/solution` package remain accessible at the new `solution` package location
- **SC-007**: Code diff between old and new package shows ONLY: package path changes in imports, file path adjustments (if any), and no logic modifications
- **SC-008**: At least one external program (test harness or example) successfully demonstrates importing and using the library

### Assumptions

- The existing `cmd/gonuget/solution` package has no circular dependencies with other CLI-specific packages
- All solution package code is self-contained and does not rely on CLI-specific initialization or global state
- The current test suite provides adequate coverage and does not need new tests for the refactor
- No breaking changes to the solution file formats (.sln, .slnx, .slnf) are expected during the refactor timeframe
- The gonuget module path (`github.com/willibrandon/gonuget`) will not change
- Performance characteristics of solution parsing are already acceptable and do not require optimization
- The package relocation will not trigger semantic versioning implications (internal refactor, not API change)

## Constraints

### Technical Constraints

- **HARD REQUIREMENT**: All code must be copied verbatim (copy-paste) - no rewrites, no refactoring, no logic modifications
- **ALLOWED CHANGES ONLY**: Package declarations (if needed), import path updates, file path corrections
- **FORBIDDEN CHANGES**: Algorithm modifications, error handling changes, variable renaming, comment edits, test logic rewrites
- **Build Requirement**: `go build ./...` must succeed at every commit
- **Test Requirement**: All existing tests must pass after import path updates
- **Dependency Requirement**: No new third-party dependencies allowed
- **Compatibility Requirement**: No breaking changes to CLI behavior or public APIs

### Out of Scope

- Adding new features to the solution parsing library
- Performance optimizations or algorithmic improvements
- Expanding test coverage beyond the existing baseline
- Creating comprehensive package documentation (beyond preserving existing comments)
- Supporting additional solution file formats beyond .sln, .slnx, .slnf
- Refactoring or improving code quality issues in the original implementation
- Updating CLI commands to use new APIs (they must work with existing APIs)
