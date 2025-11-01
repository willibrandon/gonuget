# Feature Specification: C# Interop Tests for Restore Transitive Dependencies

**Feature Branch**: `002-restore-interop-tests`
**Created**: 2025-11-01
**Status**: Draft
**Input**: User description: "C# interop tests for restore transitive dependency resolution to verify 100% NuGet.Client parity"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Transitive Dependency Resolution Parity (Priority: P1)

As a gonuget developer, I need automated tests that verify gonuget's transitive dependency resolution behaves identically to NuGet.Client so that I can confidently ship the restore feature knowing it will work correctly with all .NET projects.

**Why this priority**: This is the foundational test that validates the core restore functionality matches NuGet.Client. Without this, we cannot claim NuGet.Client parity and risk breaking user builds.

**Independent Test**: Can be fully tested by running C# interop tests that compare gonuget restore results against NuGet.Client restore results for the same project with transitive dependencies. Delivers confidence in core restore parity.

**Acceptance Scenarios**:

1. **Given** a .NET project with a package that has transitive dependencies, **When** gonuget restore is executed, **Then** all direct and transitive packages are downloaded and match NuGet.Client's resolution exactly
2. **Given** a project with multiple direct dependencies that share transitive dependencies, **When** both gonuget and NuGet.Client restore, **Then** both resolve to the same set of packages with identical versions
3. **Given** a project with complex dependency trees (10+ transitive packages), **When** restore completes, **Then** gonuget and NuGet.Client produce identical dependency graphs

---

### User Story 2 - Direct vs Transitive Categorization Verification (Priority: P1)

As a gonuget developer, I need tests that validate packages are correctly categorized as direct or transitive so that future commands like `gonuget list package --include-transitive` will display accurate information to users.

**Why this priority**: Incorrect categorization would break future package listing commands and confuse users about their project's dependencies. This is critical for the restore feature to be production-ready.

**Independent Test**: Can be fully tested by comparing the direct/transitive categorization in project.assets.json between gonuget and NuGet.Client for various project configurations. Delivers accurate dependency categorization.

**Acceptance Scenarios**:

1. **Given** a project with 3 direct dependencies, **When** restore completes, **Then** project.assets.json correctly marks exactly those 3 packages as direct dependencies
2. **Given** a project where a transitive dependency appears multiple times in the dependency tree, **When** restore completes, **Then** the package is categorized as transitive (not direct) if it's only referenced through other packages
3. **Given** a project where the same package is both a direct dependency and a transitive dependency, **When** restore completes, **Then** the package is correctly categorized as direct

---

### User Story 3 - Unresolved Package Error Message Parity (Priority: P2)

As a gonuget user who encounters missing packages, I need to see the same helpful error messages (NU1101, NU1102, NU1103) as NuGet.Client provides so that I can quickly diagnose and fix dependency problems.

**Why this priority**: Error messages are critical for user experience when things go wrong. While not blocking basic restore functionality, consistent error messages ensure users can troubleshoot issues effectively.

**Independent Test**: Can be fully tested by creating projects with intentionally missing or incompatible packages and comparing error messages between gonuget and NuGet.Client. Delivers user-friendly error diagnostics.

**Acceptance Scenarios**:

1. **Given** a project referencing a non-existent package, **When** restore fails, **Then** gonuget returns NU1101 error matching NuGet.Client's message format
2. **Given** a project requesting a version that doesn't exist (but package exists), **When** restore fails, **Then** gonuget returns NU1102 error with available versions, matching NuGet.Client's output
3. **Given** a project requesting a stable version when only prerelease exists, **When** restore fails, **Then** gonuget returns NU1103 error matching NuGet.Client's guidance

---

### User Story 4 - Lock File Format Compatibility (Priority: P1)

As a .NET developer using gonuget, I need the generated project.assets.json file to be 100% compatible with MSBuild and dotnet build so that my projects can compile successfully after gonuget restore.

**Why this priority**: If project.assets.json format differs from NuGet.Client's output, MSBuild will fail and the restore feature is unusable. This is a blocker for production release.

**Independent Test**: Can be fully tested by running gonuget restore, then dotnet build, and verifying the build succeeds with identical behavior to a NuGet.Client restore + build. Delivers MSBuild compatibility.

**Acceptance Scenarios**:

1. **Given** a restored project with gonuget, **When** dotnet build is executed, **Then** the build succeeds without errors or warnings related to project.assets.json
2. **Given** identical projects restored with gonuget and NuGet.Client, **When** comparing project.assets.json files, **Then** the Libraries map contains identical entries with matching lowercase package ID paths
3. **Given** a project with framework-specific dependencies, **When** gonuget restore completes, **Then** ProjectFileDependencyGroups contains only direct dependencies (not transitive) matching NuGet.Client's format

---

### Edge Cases

- What happens when gonuget and NuGet.Client resolve to different versions for the same package (version conflict scenarios)?
- How does the test system handle packages that have been delisted or removed from nuget.org between test runs?
- What happens when a transitive dependency has platform-specific variations (different DLLs for different frameworks)?
- How does the system validate project.assets.json when packages have complex dependency graphs with cycles (prevented by NuGet.Client)?
- What happens when testing against packages with very large dependency trees (50+ transitive dependencies)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Test suite MUST execute gonuget restore and NuGet.Client restore on identical test projects and compare results
- **FR-002**: Test suite MUST verify that all packages resolved by NuGet.Client are also resolved by gonuget with identical versions
- **FR-003**: Test suite MUST validate direct vs transitive categorization matches between gonuget and NuGet.Client
- **FR-004**: Test suite MUST verify NU1101, NU1102, and NU1103 error messages match NuGet.Client's format and content
- **FR-005**: Test suite MUST validate project.assets.json Libraries map structure matches NuGet.Client exactly
- **FR-006**: Test suite MUST verify ProjectFileDependencyGroups contains only direct dependencies (not transitive)
- **FR-007**: Test suite MUST use the existing GonugetBridge infrastructure to execute gonuget commands from C# tests
- **FR-008**: Test suite MUST achieve 90% code coverage of restore transitive dependency resolution logic
- **FR-009**: Test suite MUST include test cases for simple dependencies (1-2 transitive), moderate complexity (5-10 transitive), and complex scenarios (10+ transitive)
- **FR-010**: Test suite MUST validate package path casing in project.assets.json matches NuGet.Client (lowercase package IDs)

### Key Entities *(include if feature involves data)*

- **Test Project**: A .NET project file (.csproj) with PackageReference entries used to test restore behavior
- **Restore Result**: The outcome of running restore, including resolved packages, categorization, and generated project.assets.json
- **Unresolved Package**: A package that could not be found or resolved, with associated error code (NU1101/NU1102/NU1103)
- **Libraries Map**: The section of project.assets.json containing all packages (direct + transitive) with their metadata
- **ProjectFileDependencyGroups**: The section of project.assets.json listing only direct dependencies per target framework

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of test cases comparing gonuget restore results against NuGet.Client restore results pass without discrepancies
- **SC-002**: Test suite achieves minimum 90% code coverage of restore transitive dependency resolution implementation
- **SC-003**: All error message tests (NU1101, NU1102, NU1103) produce identical output between gonuget and NuGet.Client
- **SC-004**: 100% of dotnet build commands succeed after gonuget restore on test projects, matching NuGet.Client behavior
- **SC-005**: Test suite execution completes in under 2 minutes on CI/CD infrastructure
- **SC-006**: Zero regression failures when running the full 491-test interop suite after adding restore tests

## Assumptions

- The existing GonugetBridge infrastructure (`tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/GonugetBridge.cs`) is functional and can be reused
- Test projects can use packages from nuget.org without authentication
- The gonuget-interop-test binary supports restore operations via JSON-RPC
- Test execution environment has both gonuget and dotnet CLI available
- Project.assets.json format comparison can be done via JSON deserialization and object comparison
- Error message comparison can use string matching with tolerance for minor formatting differences (e.g., line endings, spacing)

## Dependencies

- Existing restore implementation in gonuget (complete per RESTORE-TRANSITIVE-DEPS.md)
- GonugetBridge C# infrastructure for executing gonuget commands
- gonuget-interop-test binary with restore request handlers
- NuGet.Client libraries for running reference restore operations
- Test projects with varying dependency complexity levels

## Out of Scope

- Performance benchmarking between gonuget and NuGet.Client restore (covered by existing benchmarks)
- Testing package installation to global packages folder (covered by existing integration tests)
- Validating cache file format (dgSpecHash validation - covered by existing tests)
- Testing UI/CLI output formatting (covered by separate CLI tests)
- Testing `gonuget list package` command (future enhancement, not part of restore)
- Testing `gonuget nuget why` command (future enhancement, not part of restore)
