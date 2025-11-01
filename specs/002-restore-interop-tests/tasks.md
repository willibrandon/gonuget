# Tasks: C# Interop Tests for Restore Transitive Dependencies

**Input**: Design documents from `/specs/002-restore-interop-tests/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/restore-interop.json

**Tests**: Interop tests ARE the feature - all tasks create test infrastructure and test cases.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Test suite structure**: `tests/nuget-client-interop/GonugetInterop.Tests/`
- **Go handlers**: `cmd/nuget-interop-test/`
- **Test helpers**: `tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create test helpers and extend GonugetBridge infrastructure for restore testing

- [x] T001 Create TestProject helper class in tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/TestProject.cs
- [x] T002 [P] Create RestoreTransitiveResponse type in tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/RestoreTransitiveResponse.cs
- [x] T003 [P] Create CompareProjectAssetsResponse type in tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/CompareProjectAssetsResponse.cs
- [x] T004 [P] Create ValidateErrorMessagesResponse type in tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/ValidateErrorMessagesResponse.cs

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core GonugetBridge methods and Go handlers that ALL user stories depend on

**⚠️ CRITICAL**: No user story test work can begin until this phase is complete

- [x] T005 Extend GonugetBridge with RestoreTransitive method in tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/GonugetBridge.cs
- [x] T006 [P] Extend GonugetBridge with CompareProjectAssets method in tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/GonugetBridge.cs
- [x] T007 [P] Extend GonugetBridge with ValidateErrorMessages method in tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/GonugetBridge.cs
- [x] T008 Create RestoreTransitiveHandler in cmd/nuget-interop-test/handlers_restore.go
- [x] T009 [P] Create CompareProjectAssetsHandler in cmd/nuget-interop-test/handlers_restore.go
- [x] T010 [P] Create ValidateErrorMessagesHandler in cmd/nuget-interop-test/handlers_restore.go
- [x] T011 Register new handlers in cmd/nuget-interop-test/main.go
- [x] T012 Add request/response types to cmd/nuget-interop-test/protocol.go
- [x] T013 Build and test gonuget-interop-test binary with new handlers (make build-interop)

**Checkpoint**: Foundation ready - user story test implementation can now begin in parallel

---

## Phase 3: User Story 1 - Transitive Dependency Resolution Parity (Priority: P1) 🎯 MVP

**Goal**: Verify gonuget's transitive dependency resolution behaves identically to NuGet.Client for all direct and transitive packages

**Independent Test**: Create test project with packages that have transitive dependencies (e.g., Serilog.Sinks.File → Serilog). Run gonuget restore and NuGet.Client restore. Compare resolved package lists. Test passes if all packages match exactly.

### Implementation for User Story 1

- [x] T014 [P] [US1] Create test for simple transitive resolution (1-2 levels) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T015 [P] [US1] Create test for moderate transitive resolution (5-10 packages) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T016 [P] [US1] Create test for complex transitive resolution (10+ packages) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T017 [P] [US1] Create test for diamond dependencies (multiple paths to same package) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T018 [P] [US1] Create test for framework-specific dependencies (net6.0 vs net8.0 vs net9.0) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T019 [P] [US1] Create test for shared transitive dependencies (multiple direct deps share same transitive) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T020 [P] [US1] Create test for version resolution in transitive chain in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T021 [P] [US1] Create test for transitive resolution with version ranges in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs

**Checkpoint**: At this point, transitive resolution parity should be fully validated. User Story 1 delivers core confidence in restore behavior matching NuGet.Client.

---

## Phase 4: User Story 2 - Direct vs Transitive Categorization Verification (Priority: P1)

**Goal**: Validate packages are correctly categorized as direct or transitive in project.assets.json for future package listing commands

**Independent Test**: Create test project with known direct dependencies and verify project.assets.json correctly marks only those packages as direct. Compare ProjectFileDependencyGroups (should contain only direct) vs Libraries map (should contain all packages). Test passes if categorization matches NuGet.Client exactly.

### Implementation for User Story 2

- [x] T022 [P] [US2] Create test for pure direct dependencies (no transitive) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T023 [P] [US2] Create test for pure transitive dependencies (only pulled by direct) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T024 [P] [US2] Create test for mixed scenario (package is both direct and transitive) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T025 [P] [US2] Create test for ProjectFileDependencyGroups contains only direct in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T026 [P] [US2] Create test for Libraries map contains all packages (direct + transitive) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T027 [P] [US2] Create test for multi-framework project categorization in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T028 [P] [US2] Create test for framework-specific transitive dependencies categorization in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs

**Checkpoint**: At this point, direct vs transitive categorization should be fully validated. User Stories 1 AND 2 should both work independently.

---

## Phase 5: User Story 3 - Unresolved Package Error Message Parity (Priority: P2)

**Goal**: Ensure gonuget returns identical error messages (NU1101, NU1102, NU1103) as NuGet.Client for missing or incompatible packages

**Independent Test**: Create test project with non-existent package. Run gonuget restore (should fail). Compare error message format and content with NuGet.Client error. Test passes if error codes and messages match exactly (allowing for minor formatting tolerance).

### Implementation for User Story 3

- [x] T029 [P] [US3] Create test for NU1101 error (package doesn't exist) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T030 [P] [US3] Create test for NU1102 error (version doesn't exist) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T031 [P] [US3] Create test for NU1103 error (only prerelease available) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T032 [P] [US3] Create test for error message format matching (spacing, punctuation) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T033 [P] [US3] Create test for error message sources list accuracy in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T034 [P] [US3] Create test for NU1102 available versions list in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T035 [P] [US3] Create test for NU1102 nearest version suggestion in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs

**Checkpoint**: At this point, unresolved package error messages should be fully validated. All user stories 1, 2, AND 3 should work independently.

---

## Phase 6: User Story 4 - Lock File Format Compatibility (Priority: P1)

**Goal**: Ensure generated project.assets.json is 100% compatible with MSBuild and dotnet build

**Independent Test**: Run gonuget restore on test project, then run dotnet build. Test passes if build succeeds without errors or warnings. Additionally, compare project.assets.json structure with NuGet.Client output to verify semantic equivalence.

### Implementation for User Story 4

- [x] T036 [P] [US4] Create test for Libraries map structure (lowercase paths, metadata) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T037 [P] [US4] Create test for ProjectFileDependencyGroups (direct only, not transitive) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T038 [P] [US4] Create test for multi-framework project lock file structure in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T039 [P] [US4] Create test for MSBuild compatibility (dotnet build succeeds after gonuget restore) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T040 [P] [US4] Create test for package path casing (lowercase package IDs) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T041 [P] [US4] Create test for targets section structure (framework-specific packages) in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs
- [x] T042 [P] [US4] Create test for lock file version and format compatibility in tests/nuget-client-interop/GonugetInterop.Tests/RestoreTransitiveTests.cs

**Checkpoint**: All user stories should now be independently functional. The test suite comprehensively validates restore transitive dependency resolution parity.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories and overall test quality

- [ ] T043 [P] Add summary comments to RestoreTransitiveTests.cs explaining test categories
- [ ] T044 [P] Add XML documentation to TestProject helper class
- [ ] T045 [P] Add XML documentation to GonugetBridge new methods
- [ ] T046 [P] Add error handling and diagnostic output to test helpers
- [ ] T047 Verify test execution time <2 minutes (SC-005 requirement)
- [ ] T048 Verify zero regression failures in existing 491 tests (SC-006 requirement)
- [ ] T049 Run coverage analysis and verify 90%+ coverage of restore transitive resolution (SC-002 requirement)
- [ ] T050 Update CLAUDE.md with test usage examples if needed
- [ ] T051 Validate quickstart.md instructions match actual implementation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (US1 → US2 → US4 → US3)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Independently testable, uses same infrastructure as US1
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Independently testable, tests error cases
- **User Story 4 (P1)**: Can start after Foundational (Phase 2) - Independently testable, validates output format

### Within Each User Story

- All tests within a user story marked [P] can run in parallel (different test methods, independent scenarios)
- Test creation is independent - tests don't depend on each other
- Each test is self-contained with its own test project creation and cleanup

### Parallel Opportunities

- All Setup tasks (T001-T004) marked [P] can run in parallel
- All Foundational tasks (T006-T007, T009-T010, T012) marked [P] can run in parallel within Phase 2
- Once Foundational phase completes, all user stories (Phase 3-6) can start in parallel
- All tests within each user story marked [P] can be created in parallel
- All Polish tasks marked [P] can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together (8 independent test cases):
Task: T014 "Create test for simple transitive resolution"
Task: T015 "Create test for moderate transitive resolution"
Task: T016 "Create test for complex transitive resolution"
Task: T017 "Create test for diamond dependencies"
Task: T018 "Create test for framework-specific dependencies"
Task: T019 "Create test for shared transitive dependencies"
Task: T020 "Create test for version resolution in transitive chain"
Task: T021 "Create test for transitive resolution with version ranges"
```

## Parallel Example: All User Stories After Foundation

```bash
# After Phase 2 completes, launch all user stories in parallel:
Task Group: US1 (T014-T021) - Transitive Resolution Parity tests
Task Group: US2 (T022-T028) - Categorization tests
Task Group: US3 (T029-T035) - Error Message tests
Task Group: US4 (T036-T042) - Lock File Format tests
```

---

## Implementation Strategy

### MVP First (User Stories 1 and 2 Only)

1. Complete Phase 1: Setup (T001-T004)
2. Complete Phase 2: Foundational (T005-T013) - CRITICAL - blocks all stories
3. Complete Phase 3: User Story 1 (T014-T021) - Core transitive resolution
4. Complete Phase 4: User Story 2 (T022-T028) - Categorization
5. **STOP and VALIDATE**: Run test suite, verify 100% pass rate for US1 and US2
6. Achieve minimum viable test coverage (basic transitive resolution + categorization)

### Incremental Delivery

1. Complete Setup + Foundational → Test infrastructure ready
2. Add User Story 1 → Test independently → Validate transitive resolution parity (MVP!)
3. Add User Story 2 → Test independently → Validate categorization
4. Add User Story 4 → Test independently → Validate lock file format
5. Add User Story 3 → Test independently → Validate error messages
6. Each story adds validation coverage without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (T001-T013)
2. Once Foundational is done:
   - Developer A: User Story 1 (T014-T021) - 8 tests
   - Developer B: User Story 2 (T022-T028) - 7 tests
   - Developer C: User Story 3 (T029-T035) - 7 tests
   - Developer D: User Story 4 (T036-T042) - 7 tests
3. Stories complete and integrate independently (no conflicts - different test methods)

---

## Task Summary

**Total Tasks**: 51 tasks

**By Phase**:
- Phase 1 (Setup): 4 tasks
- Phase 2 (Foundational): 9 tasks (BLOCKING)
- Phase 3 (US1 - Transitive Resolution Parity): 8 tasks
- Phase 4 (US2 - Categorization): 7 tasks
- Phase 5 (US3 - Error Messages): 7 tasks
- Phase 6 (US4 - Lock File Format): 7 tasks
- Phase 7 (Polish): 9 tasks

**Parallel Opportunities**: 45 tasks marked [P] can run in parallel within their phase or across phases

**Independent Test Criteria**:
- US1: All packages resolved by gonuget match NuGet.Client exactly
- US2: Direct vs transitive categorization matches NuGet.Client in project.assets.json
- US3: Error messages (NU1101/NU1102/NU1103) match NuGet.Client format exactly
- US4: dotnet build succeeds after gonuget restore with identical project.assets.json structure

**MVP Scope**: Phases 1-4 (Setup + Foundational + US1 + US2) = 28 tasks, delivering core transitive resolution and categorization validation

**Format Validation**: ✅ All tasks follow checklist format with checkbox, ID, optional [P] and [Story] markers, and file paths

---

## Notes

- [P] tasks = different files or independent test methods, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- All tests are integration/interop tests - they validate gonuget vs NuGet.Client parity
- Test failures indicate bugs in gonuget restore implementation (fix in restore/, not tests)
- Tests should be created to FAIL if gonuget behavior deviates from NuGet.Client
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Success criteria: 100% pass rate (SC-001), 90%+ coverage (SC-002), <2min execution (SC-005), zero regressions (SC-006)
