# Implementation Plan: C# Interop Tests for Restore Transitive Dependencies

**Branch**: `002-restore-interop-tests` | **Date**: 2025-11-01 | **Spec**: [spec.md](spec.md)

## Summary

Implement comprehensive C# interop tests to validate gonuget's restore transitive dependency resolution matches NuGet.Client behavior exactly. Tests will verify package resolution parity, direct vs transitive categorization, unresolved package error messages (NU1101/NU1102/NU1103), and project.assets.json lock file format compatibility with MSBuild.

## Technical Context

**Language/Version**: C# 9.0 (test suite), Go 1.21+ (tested implementation)
**Primary Dependencies**: NuGet.Client 6.x libraries, xUnit, existing GonugetBridge infrastructure
**Storage**: Temporary test projects and package caches (cleaned between tests)
**Testing**: xUnit test framework with GonugetBridge JSON-RPC communication
**Target Platform**: Cross-platform (tests run on Linux, macOS, Windows CI)
**Project Type**: Test suite (extends existing `tests/nuget-client-interop/` structure)
**Performance Goals**: Test suite execution <2 minutes, 100% pass rate
**Constraints**: Must achieve 90%+ code coverage of restore transitive resolution logic
**Scale/Scope**: Add 20-30 test cases to existing 491-test interop suite

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Constitution Compliance ✅

**Principle I - 100% NuGet.Client Parity**: ✅ PASS
- Feature directly supports parity validation via interop tests
- Tests will verify exact behavior match with NuGet.Client
- Aligns with constitution mandate: "491 C# interop tests MUST pass before any feature is considered complete"

**Principle IV - Interop Testing (Quality Gate)**: ✅ PASS
- Feature IS the interop testing for restore transitive dependencies
- Directly implements constitution requirement: "All new features MUST have corresponding C# interop tests"
- Extends existing 491-test suite with restore-specific validation

**Testing Requirements**: ✅ PASS
- Will achieve 90%+ coverage of restore transitive resolution (per FR-008)
- Uses xUnit framework (existing pattern in GonugetInterop.Tests/)
- Organized under `tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs`

**Performance Standards**: ✅ PASS
- Test execution target <2 minutes (SC-005)
- Zero regression failures in existing 491 tests (SC-006)
- Aligns with observability principle (tests validate behavior, not just performance)

**Code Quality**: ✅ PASS
- Tests use existing GonugetBridge infrastructure (no new complexity)
- Follows established C# interop test patterns
- JSON-RPC protocol already proven with 491 existing tests

### Gate Evaluation: **PASS** ✅

No constitution violations. Feature directly implements constitutional requirements for interop testing.

## Project Structure

### Documentation (this feature)

```text
specs/002-restore-interop-tests/
├── plan.md                      # This file
├── spec.md                      # Feature specification
├── research.md                  # Phase 0: Test patterns and approach
├── data-model.md                # Phase 1: Test data structures
├── contracts/                   # Phase 1: JSON-RPC request/response contracts
│   └── restore-interop.json     # OpenAPI schema for restore test actions
├── quickstart.md                # Phase 1: Developer guide for running tests
├── checklists/
│   └── requirements.md          # Spec quality checklist (complete)
└── tasks.md                     # Phase 2: Implementation tasks (NOT created yet)
```

### Source Code (repository root)

```text
tests/nuget-client-interop/
├── GonugetInterop.Tests/
│   ├── RestoreTransitiveTests.cs          # NEW: Main test file (20-30 tests)
│   ├── TestHelpers/
│   │   ├── GonugetBridge.cs               # EXISTING: Extend with new methods
│   │   ├── RestoreTransitiveResponse.cs   # NEW: Response type for transitive restore
│   │   ├── CompareProjectAssetsResponse.cs # NEW: Lock file comparison response
│   │   ├── ValidateErrorMessagesResponse.cs # NEW: Error message validation response
│   │   └── TestProject.cs                 # NEW: Helper to create test .csproj files
│   └── GonugetInterop.Tests.csproj        # MODIFY: Add new test file reference
│
└── gonuget-interop-test binary (cmd/nuget-interop-test/)
    ├── handlers_restore.go                # MODIFY: Add new handler methods
    │   ├── RestoreTransitive              # NEW: Full transitive restore handler
    │   ├── CompareProjectAssets           # NEW: Lock file comparison handler
    │   └── ValidateErrorMessages          # NEW: Error message validation handler
    ├── protocol.go                        # MODIFY: Add new request/response types
    └── main.go                            # MODIFY: Register new handlers
```

**Structure Decision**: Test suite structure follows existing C# interop test pattern with tests in `GonugetInterop.Tests/` and handlers in `cmd/nuget-interop-test/`. This maintains consistency with the 491 existing tests and leverages proven GonugetBridge infrastructure.

## Complexity Tracking

> Feature adds no complexity - extends existing proven interop test pattern. No violations to justify.

## Phase 0: Research & Design Validation

### Research Topics

1. **Existing Interop Test Patterns**
   - **Decision**: Follow ResolverAdvancedTests.cs pattern for complex multi-package scenarios
   - **Rationale**: Existing resolver tests already validate transitive resolution at algorithm level; restore tests validate end-to-end with project.assets.json generation
   - **Pattern**: Each test creates test project → calls GonugetBridge → compares with NuGet.Client behavior

2. **NuGet.Client Error Message Format**
   - **Decision**: Use exact string matching for NU1101/NU1102/NU1103 error codes with tolerance for minor formatting (line endings, spacing)
   - **Rationale**: Error messages are user-facing and must match exactly for consistent UX
   - **Reference**: `NuGet.Commands/RestoreCommand/UnresolvedMessages.cs` in NuGet.Client source

3. **project.assets.json Comparison Strategy**
   - **Decision**: Deserialize JSON to object graph and compare semantically (not string comparison)
   - **Rationale**: JSON key order may vary; semantic comparison is more robust and maintainable
   - **Approach**: Compare Libraries map keys, ProjectFileDependencyGroups contents, package versions, and paths

4. **Test Project Generation**
   - **Decision**: Create minimal .csproj files in temp directories with PackageReference entries
   - **Rationale**: Lightweight, fast, no external dependencies, full control over test scenarios
   - **Pattern**: Reuse existing TestHelpers pattern from other interop tests

5. **GonugetBridge Extension Strategy**
   - **Decision**: Add new static methods to GonugetBridge.cs following existing naming conventions
   - **Rationale**: Maintains consistency with 491 existing tests, proven architecture
   - **Methods**: `RestoreTransitive()`, `CompareProjectAssets()`, `ValidateErrorMessages()`

### Alternatives Considered

**Alternative 1**: Use real .NET projects from GitHub
- **Rejected**: External dependencies, network flakiness, version drift, slow test execution

**Alternative 2**: String comparison for project.assets.json
- **Rejected**: Brittle (fails on key reordering), hard to debug, doesn't validate semantic equivalence

**Alternative 3**: Create separate test binary instead of extending gonuget-interop-test
- **Rejected**: Duplicates infrastructure, increases maintenance burden, breaks from proven pattern

## Phase 1: Detailed Design

### Test Data Model

See [data-model.md](data-model.md) for:
- `TestProject` entity (represents .csproj with PackageReferences)
- `RestoreResult` entity (captures restore outcome, packages, errors)
- `LockFileComparison` entity (semantic diff of project.assets.json files)
- `ErrorMessageValidation` entity (NU1101/NU1102/NU1103 message comparison)

### JSON-RPC Contracts

See [contracts/restore-interop.json](contracts/restore-interop.json) for:
- `restore_transitive` action (full transitive restore with result validation)
- `compare_project_assets` action (semantic lock file comparison)
- `validate_error_messages` action (error message format validation)

### Test Organization

**Test File**: `RestoreTransitiveTests.cs`

**Test Categories**:

1. **Transitive Resolution Parity** (8-10 tests)
   - Simple transitive (1-2 levels deep)
   - Moderate transitive (5-10 packages)
   - Complex transitive (10+ packages, shared dependencies)
   - Diamond dependencies (multiple paths to same package)
   - Framework-specific dependencies

2. **Direct vs Transitive Categorization** (5-7 tests)
   - Pure direct dependencies (no transitive)
   - Pure transitive dependencies (pulled by direct)
   - Mixed (package is both direct and transitive)
   - Categorization in ProjectFileDependencyGroups vs Libraries

3. **Unresolved Package Error Messages** (5-7 tests)
   - NU1101: Package doesn't exist
   - NU1102: Version doesn't exist (with available versions list)
   - NU1103: Only prerelease available when stable requested
   - Error message format matching (spacing, punctuation, sources list)

4. **Lock File Format Compatibility** (5-7 tests)
   - Libraries map structure (lowercase paths, metadata)
   - ProjectFileDependencyGroups (direct only, not transitive)
   - Multi-framework projects
   - MSBuild compatibility (dotnet build succeeds after gonuget restore)

### Test Execution Flow

```
[C# Test]
    ↓
  Create TestProject (.csproj with PackageReferences)
    ↓
  GonugetBridge.RestoreTransitive(projectPath)
    ↓
  [gonuget-interop-test binary]
    ↓
  restore.Run() [gonuget restore implementation]
    ↓
  Return RestoreTransitiveResponse (packages, lock file path, errors)
    ↓
  [C# Test]
    ↓
  NuGet.Client restore (for comparison)
    ↓
  Compare results (packages, categorization, errors, lock file)
    ↓
  Assert parity (100% match required)
```

### Coverage Strategy

**Target**: 90% coverage of `restore/restorer.go` transitive resolution logic

**Covered Code Paths**:
- Transitive graph walking (`walker.Walk()` with `recursive=true`)
- Direct vs transitive categorization logic
- Unresolved package handling and error generation
- Lock file builder (Libraries map, ProjectFileDependencyGroups)
- Local-first metadata client (cache-then-network pattern)

**Uncovered (Intentional)**:
- Performance benchmarking (separate test suite)
- Cache file format validation (existing tests)
- CLI output formatting (separate tests)

## Phase 2: Implementation Tasks

*Tasks will be generated by `/speckit.tasks` command (NOT part of `/speckit.plan`).*

## Dependencies

- ✅ Existing `GonugetBridge` infrastructure (functional)
- ✅ `gonuget-interop-test` binary with JSON-RPC protocol (operational)
- ✅ Restore implementation complete (per RESTORE-TRANSITIVE-DEPS.md)
- ✅ NuGet.Client libraries available in test project
- ✅ xUnit test framework (existing dependency)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Flaky tests due to nuget.org network | Test suite unreliable in CI | Use package cache, add retry logic, mark network tests appropriately |
| NuGet.Client behavior changes in updates | Tests break on NuGet.Client version upgrades | Pin NuGet.Client version, document upgrade process |
| Test execution time >2 minutes | CI slowdown, developer friction | Parallelize tests, optimize test project sizes, use package cache |
| project.assets.json format differences | False positive test failures | Semantic comparison (not string), focus on MSBuild-critical fields |

## Success Metrics

From specification success criteria:

- **SC-001**: 100% of test cases pass (gonuget matches NuGet.Client) ✅
- **SC-002**: 90%+ code coverage of restore transitive resolution ✅
- **SC-003**: Identical error messages (NU1101/NU1102/NU1103) ✅
- **SC-004**: 100% dotnet build success after gonuget restore ✅
- **SC-005**: Test suite execution <2 minutes ✅
- **SC-006**: Zero regressions in existing 491 tests ✅

## Next Steps

1. **Phase 0 Complete**: Research captured in this section
2. **Phase 1 Next**: Generate `data-model.md`, `contracts/`, `quickstart.md`
3. **Phase 2 Future**: Run `/speckit.tasks` to generate implementation tasks
