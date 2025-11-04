# Research: Solution File Parsing Library Refactor

**Feature**: 004-solution-lib-refactor
**Date**: 2025-11-03
**Status**: Complete

## Overview

This research document addresses technical decisions and best practices for refactoring the solution file parsing code from a CLI-embedded package to a standalone library package.

## Research Questions

### Q1: Go Package Relocation Best Practices

**Research Task**: Identify best practices for moving Go packages between directories while maintaining backward compatibility and avoiding breaking changes.

**Decision**: Use direct file copy + import path update + old directory deletion strategy

**Rationale**:
- Go's module system treats package paths as immutable identifiers
- Moving a package creates a new import path, which is inherently a breaking change for external consumers
- Since `cmd/gonuget/solution` is CLI-internal (no external consumers yet), we can safely break this path
- The new `solution/` path becomes the stable, public API going forward
- Copying verbatim (not using `git mv`) preserves exact file contents and avoids merge conflicts

**Alternatives Considered**:
1. **Gradual deprecation with compatibility shim**: Create `solution/` and leave `cmd/gonuget/solution` as a re-export wrapper
   - **Rejected**: Adds complexity, violates "no new code" constraint, creates confusing dual import paths
2. **Git history preservation via `git mv`**: Use `git mv` to preserve file history across the move
   - **Rejected**: Git mv can cause merge conflicts, and file history is still accessible via `git log --follow`
3. **Monorepo tool (e.g., gomod-replace)**: Use replace directives to alias old path to new
   - **Rejected**: Over-engineered for internal refactor, adds build complexity

**References**:
- Go Wiki: [Go Modules - When should I use replace directives?](https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive)
- Go Blog: [Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors) (discusses API evolution)

---

### Q2: Import Path Update Automation

**Research Task**: Determine the safest method to update import paths across the codebase without missing references or breaking builds.

**Decision**: Manual import path replacement with build verification at each step

**Rationale**:
- Only 3 files need updates (`package_add.go`, `package_list.go`, `package_remove.go`)
- Manual replacement with editor find-replace ensures precision
- `go build ./...` provides immediate feedback if any imports are missed
- Tooling like `goimports` can auto-fix missing imports but may choose wrong paths

**Alternatives Considered**:
1. **Automated refactoring tools (e.g., gorename, gomvpkg)**: Use Go refactoring tools
   - **Rejected**: These tools are deprecated/unmaintained, and manual is safer for 3 files
2. **sed/awk batch replacement**: Script the replacement with Unix text tools
   - **Rejected**: Risk of false positives (e.g., replacing in comments or strings), not IDE-safe
3. **IDE refactoring (VSCode/GoLand)**: Use IDE's "rename symbol" feature
   - **Rejected**: Not applicable to package paths, only works for symbols within packages

**Implementation Strategy**:
1. Copy files to `solution/` directory
2. Run `go build ./...` to establish baseline
3. Update import in `package_add.go`, verify build succeeds
4. Update import in `package_list.go`, verify build succeeds
5. Update import in `package_remove.go`, verify build succeeds
6. Delete old `cmd/gonuget/solution/` directory, verify build still succeeds
7. Run full test suite

---

### Q3: Test Coverage Preservation Strategy

**Research Task**: Identify methods to ensure test coverage remains identical after the refactor.

**Decision**: Rely on existing CLI integration tests as validation; no test relocation needed

**Rationale**:
- The solution package has **no dedicated unit tests** in `cmd/gonuget/solution/*_test.go` (verified by codebase inspection)
- Test coverage comes from CLI integration tests that exercise solution parsing via commands
- These tests automatically validate the refactored package by importing it through CLI commands
- Coverage percentage remains identical because the same code is being tested through the same entry points

**Verification Steps**:
1. Run `go test ./cmd/gonuget/commands -v` before refactor (baseline)
2. Perform refactor with import path updates
3. Run `go test ./cmd/gonuget/commands -v` after refactor (must match baseline results)
4. Run `go test -cover ./solution` to verify new package has coverage (via CLI tests)

**Alternatives Considered**:
1. **Create dedicated unit tests for solution package**: Write new tests directly against `solution/` APIs
   - **Rejected**: Violates "no new code" constraint, would increase coverage (not preserve it)
2. **Copy internal CLI tests to solution package**: Move integration tests to library package
   - **Rejected**: Tests belong with their consumers (CLI commands), not the library

---

### Q4: Dependency Cycle Detection

**Research Task**: Best practices for detecting and preventing circular dependencies during package refactoring.

**Decision**: Use `go build` as primary validation tool with explicit dependency graph verification

**Rationale**:
- Go's build system natively detects circular dependencies and fails with clear error messages
- The solution package uses only standard library imports (no gonuget internal packages)
- CLI commands import solution package (unidirectional dependency: CLI → solution)
- Build failure would immediately reveal any accidental reverse dependency

**Verification Command**:
```bash
go build ./...  # Fails immediately if circular dependency exists
```

**Alternatives Considered**:
1. **Dependency visualization tools (e.g., godepgraph)**: Generate visual dependency graph
   - **Rejected**: Overkill for simple refactor, `go build` is sufficient
2. **Static analysis (e.g., go list -deps)**: Manually inspect dependency tree
   - **Rejected**: More complex than needed, build verification is definitive

**Expected Dependency Flow (After Refactor)**:
```
solution/ (pure library, no internal deps)
    ↑
    └── imports only std lib (encoding/xml, os, filepath, etc.)

cmd/gonuget/commands/
    ↑
    └── imports solution/ (unidirectional)
```

---

### Q5: Performance Baseline Establishment

**Research Task**: Methods to capture pre-refactor performance baseline for the 5% threshold validation (SC-004).

**Decision**: Use Go's built-in benchmarking for programmatic measurement; manual timing for CLI commands

**Rationale**:
- Go's `testing.B` provides precise nanosecond-level benchmarking for functions
- CLI command timing needs wall-clock measurement (includes startup overhead, file I/O)
- The 5% threshold is measured against total execution time, not individual function performance

**Baseline Capture Strategy**:

**For Library Functions** (if benchmarks exist):
```bash
# Run before refactor
go test -bench=. -benchmem ./cmd/gonuget/solution > baseline-bench.txt

# Run after refactor
go test -bench=. -benchmem ./solution > refactor-bench.txt

# Compare with benchstat
benchstat baseline-bench.txt refactor-bench.txt
```

**For CLI Commands** (primary validation):
```bash
# Create test project with .sln file
# Run command multiple times and average

# Before refactor:
time gonuget package list --solution Test.sln
# Record: execution time, max memory (via /usr/bin/time -l on macOS)

# After refactor:
time gonuget package list --solution Test.sln
# Verify: within 5% of baseline
```

**Alternatives Considered**:
1. **Continuous benchmarking system**: Integrate with CI/CD for automated performance tracking
   - **Rejected**: Out of scope for single refactor task
2. **Profiling with pprof**: Use CPU/memory profiling for detailed analysis
   - **Rejected**: Unnecessary detail for "should be identical" refactor

**Validation Criteria**:
- Execution time: ±5% of baseline
- Memory usage: ±5% of baseline
- If benchmarks show >5% regression, investigate (likely indicates accidental logic change)

---

## Summary of Decisions

| Question | Decision | Confidence |
|----------|----------|------------|
| Package relocation method | Direct copy + import update + delete old | ✅ High |
| Import path update | Manual replacement with incremental builds | ✅ High |
| Test coverage strategy | Rely on existing CLI integration tests | ✅ High |
| Circular dependency detection | Go build validation | ✅ High |
| Performance baseline | Manual CLI timing + benchstat (if applicable) | ✅ High |

## Open Questions

**None** - All technical decisions resolved. Ready for Phase 1 design.

---

## Phase 0 Completion Checklist

- [x] All NEEDS CLARIFICATION items from Technical Context resolved
- [x] Best practices for Go package relocation documented
- [x] Import path update strategy defined
- [x] Test coverage preservation approach validated
- [x] Circular dependency detection method confirmed
- [x] Performance baseline capture strategy documented
- [x] No outstanding research questions
- [x] Ready to proceed to Phase 1 (Design & Contracts)

**Status**: ✅ **Phase 0 Complete** - Proceed to Phase 1
