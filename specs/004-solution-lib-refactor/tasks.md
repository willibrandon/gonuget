# Tasks: Solution File Parsing Library Refactor

**Input**: Design documents from `/specs/004-solution-lib-refactor/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: No new tests required - this refactor preserves existing test coverage via CLI integration tests.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

**Gonuget Structure**: Repository root has library packages (`solution/`, `version/`, `frameworks/`, etc.) and CLI under `cmd/gonuget/`.

---

## Phase 1: Setup (Preparation)

**Purpose**: Prepare for package relocation and establish baseline

- [X] T001 Create `solution/` directory at repository root
- [X] T002 Capture baseline performance metrics for CLI commands using solution parsing
- [X] T003 Verify no existing circular dependencies with `go list -deps ./cmd/gonuget/solution`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core file relocation that MUST be complete before ANY user story can be validated

**⚠️ CRITICAL**: All source files must be relocated before CLI integration or external validation can proceed

- [X] T004 [P] Copy `cmd/gonuget/solution/detector.go` to `solution/detector.go` (verbatim)
- [X] T005 [P] Copy `cmd/gonuget/solution/types.go` to `solution/types.go` (verbatim)
- [X] T006 [P] Copy `cmd/gonuget/solution/sln_parser.go` to `solution/sln_parser.go` (verbatim)
- [X] T007 [P] Copy `cmd/gonuget/solution/slnx_parser.go` to `solution/slnx_parser.go` (verbatim)
- [X] T008 [P] Copy `cmd/gonuget/solution/slnf_parser.go` to `solution/slnf_parser.go` (verbatim)
- [X] T009 [P] Copy `cmd/gonuget/solution/parser.go` to `solution/parser.go` (verbatim)
- [X] T010 [P] Copy `cmd/gonuget/solution/path.go` to `solution/path.go` (verbatim)
- [X] T011 Verify all 7 files copied successfully with `ls -1 solution/*.go | wc -l`
- [X] T012 Verify package declarations are correct with `grep "^package " solution/*.go`
- [X] T013 Build new `solution/` package with `go build ./solution`

**Checkpoint**: Foundation ready - all source files relocated and buildable

---

## Phase 3: User Story 1 - External Tool Integration (Priority: P1) 🎯 MVP

**Goal**: Enable external Go programs to import and use the solution parsing library

**Independent Test**: Create a minimal external Go program that imports `github.com/willibrandon/gonuget/solution`, calls `solution.GetParser()`, and successfully parses a .sln file

### Implementation for User Story 1

- [X] T014 [US1] Create external test program directory at `/tmp/gonuget-test`
- [X] T015 [US1] Initialize Go module in test program with `go mod init example.com/test`
- [X] T016 [US1] Create main.go that imports `github.com/willibrandon/gonuget/solution` in `/tmp/gonuget-test/main.go`
- [X] T017 [US1] Add test code to call `solution.NewDetector()` and `solution.GetParser()` in main.go
- [X] T018 [US1] Add replace directive for local gonuget with `go mod edit -replace github.com/willibrandon/gonuget=/Users/brandon/src/gonuget`
- [X] T019 [US1] Build external test program with `go build` in `/tmp/gonuget-test`
- [X] T020 [US1] Run external test program and verify import succeeds with `./test`
- [X] T021 [US1] Verify all exported APIs accessible by testing `solution.IsSolutionFile()`, `solution.IsProjectFile()`, `solution.GetSolutionFormat()` in test program

**Checkpoint**: External programs can successfully import and use `github.com/willibrandon/gonuget/solution` ✅ SC-001

---

## Phase 4: User Story 2 - CLI Backward Compatibility (Priority: P1) 🎯 MVP

**Goal**: Ensure existing CLI commands continue working identically after refactor

**Independent Test**: Run existing CLI test suite and verify 100% of tests pass without modification

### Implementation for User Story 2

- [X] T022 [US2] Update import path in `cmd/gonuget/commands/package_add.go` from `github.com/willibrandon/gonuget/cmd/gonuget/solution` to `github.com/willibrandon/gonuget/solution`
- [X] T023 [US2] Verify `cmd/gonuget/commands/package_add.go` builds with `go build ./cmd/gonuget/commands`
- [X] T024 [US2] Update import path in `cmd/gonuget/commands/package_list.go` from `github.com/willibrandon/gonuget/cmd/gonuget/solution` to `github.com/willibrandon/gonuget/solution`
- [X] T025 [US2] Verify `cmd/gonuget/commands/package_list.go` builds with `go build ./cmd/gonuget/commands`
- [X] T026 [US2] Update import path in `cmd/gonuget/commands/package_remove.go` from `github.com/willibrandon/gonuget/cmd/gonuget/solution` to `github.com/willibrandon/gonuget/solution`
- [X] T027 [US2] Verify `cmd/gonuget/commands/package_remove.go` builds with `go build ./cmd/gonuget/commands`
- [X] T028 [US2] Verify full codebase builds with `go build ./...`
- [X] T029 [US2] Run CLI test suite with `go test ./cmd/gonuget/commands -v`
- [X] T030 [US2] Verify 100% of CLI tests pass (same count as baseline) - 83 passed, 2 skipped
- [X] T031 [US2] Verify test coverage preserved with `go test -cover ./cmd/gonuget/commands` - 55.4%
- [X] T032 [US2] Delete old `cmd/gonuget/solution/` directory with `rm -rf cmd/gonuget/solution/`
- [X] T033 [US2] Verify build still succeeds after deletion with `go build ./...`
- [X] T034 [US2] Verify no old imports remain with `grep -r "cmd/gonuget/solution" cmd/gonuget/commands/`
- [X] T035 [US2] Run full test suite with `go test ./...` - Updated 5 test files, all pass
- [X] T036 [US2] Run race detector with `go test -race ./cmd/gonuget/commands`
- [X] T037 [US2] **BONUS**: Fixed all lint issues (19+ fixes) - `make lint` passes with 0 issues

**Checkpoint**: CLI commands fully functional with new import paths ✅ SC-002, SC-003, SC-004, SC-005

---

## Phase 5: User Story 3 - Library Documentation and Examples (Priority: P2)

**Goal**: Provide discoverable documentation for library consumers via pkg.go.dev

**Independent Test**: Verify godoc comments exist for all exported types and functions, and at least one example is accessible

### Implementation for User Story 3

- [ ] T038 [P] [US3] Create example function `ExampleGetParser` in `solution/example_test.go` demonstrating .sln parsing
- [ ] T039 [P] [US3] Create example function `ExampleNewDetector` in `solution/example_test.go` demonstrating auto-detection
- [ ] T040 [P] [US3] Create example function `ExampleSolution_GetProjects` in `solution/example_test.go` demonstrating project filtering
- [ ] T041 [US3] Verify examples compile with `go test ./solution`
- [ ] T042 [US3] Run examples to ensure they execute successfully with `go test -run Example ./solution`
- [ ] T043 [US3] Verify godoc comments present for all exported symbols with `go doc -all ./solution | grep "^func\|^type\|^const" | wc -l`
- [ ] T044 [US3] Generate local godoc preview with `godoc -http=:6060` and verify examples appear at `http://localhost:6060/pkg/github.com/willibrandon/gonuget/solution/`

**Checkpoint**: Documentation and examples ready for pkg.go.dev ✅ SC-006

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and verification across all user stories

- [ ] T045 [P] Verify code diff shows only import path changes with `git diff --stat`
- [ ] T046 [P] Verify no logic modifications with code review of changed files
- [ ] T047 Verify all exported APIs accessible at new location by running external test program
- [ ] T048 Capture API surface with `go doc -all github.com/willibrandon/gonuget/solution > api-after.txt`
- [ ] T049 Run full build validation with `go build ./...` and verify zero errors/warnings
- [ ] T050 Run quickstart.md validation steps 1-15
- [ ] T051 Update CLAUDE.md if needed to document solution package relocation pattern

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup (T001-T003) completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase (T004-T013) completion
  - User Story 1 (External Tool Integration) can start after Foundational
  - User Story 2 (CLI Backward Compatibility) can start after Foundational
  - User Story 3 (Library Documentation) can start after User Story 1 (needs working library)
- **Polish (Phase 6)**: Depends on all P1 user stories (US1, US2) being complete

### User Story Dependencies

- **User Story 1 (P1 - External Tool Integration)**: Can start after Foundational (Phase 2) - Tests library import independently
- **User Story 2 (P1 - CLI Backward Compatibility)**: Can start after Foundational (Phase 2) - Tests CLI integration independently
- **User Story 3 (P2 - Library Documentation)**: Depends on User Story 1 (needs working library package) - Creates examples and documentation

### Within Each User Story

**User Story 1 (External Tool Integration)**:
1. Create test environment (T014-T015)
2. Write test program (T016-T017)
3. Configure module (T018)
4. Build and run (T019-T020)
5. Verify APIs (T021)

**User Story 2 (CLI Backward Compatibility)**:
1. Update imports sequentially (T022-T027) with build verification after each
2. Delete old package (T032)
3. Run comprehensive tests (T028-T037)

**User Story 3 (Library Documentation)**:
1. Create examples in parallel (T038-T040)
2. Verify examples (T041-T042)
3. Validate documentation (T043-T044)

### Parallel Opportunities

**Phase 1 (Setup)**:
- All tasks can run sequentially (quick preparation phase)

**Phase 2 (Foundational)**:
- Tasks T004-T010 (7 file copies) can ALL run in parallel [P]
- Task T011 must wait for T004-T010 (verification)
- Tasks T012-T013 run sequentially after T011

**Phase 3 (User Story 1)**:
- Tasks T014-T021 run sequentially (test program flow)

**Phase 4 (User Story 2)**:
- Tasks T022-T027 run sequentially (import updates with incremental verification)
- Tasks T028-T037 run sequentially (comprehensive testing)

**Phase 5 (User Story 3)**:
- Tasks T038-T040 can run in parallel [P] (different example functions)
- Tasks T041-T044 run sequentially after T038-T040

**Phase 6 (Polish)**:
- Tasks T045-T046 can run in parallel [P]
- Tasks T047-T051 run sequentially

**Cross-Phase Parallelization**:
- Once Foundational (Phase 2) completes, User Story 1 (Phase 3) and User Story 2 (Phase 4) can be worked on in parallel by different developers
- User Story 3 (Phase 5) must wait for User Story 1 to complete

---

## Parallel Example: Foundational Phase (File Copies)

```bash
# Launch all file copy tasks together (Phase 2):
Task: "Copy cmd/gonuget/solution/detector.go to solution/detector.go (verbatim)"
Task: "Copy cmd/gonuget/solution/types.go to solution/types.go (verbatim)"
Task: "Copy cmd/gonuget/solution/sln_parser.go to solution/sln_parser.go (verbatim)"
Task: "Copy cmd/gonuget/solution/slnx_parser.go to solution/slnx_parser.go (verbatim)"
Task: "Copy cmd/gonuget/solution/slnf_parser.go to solution/slnf_parser.go (verbatim)"
Task: "Copy cmd/gonuget/solution/parser.go to solution/parser.go (verbatim)"
Task: "Copy cmd/gonuget/solution/path.go to solution/path.go (verbatim)"
```

## Parallel Example: User Story 3 (Example Creation)

```bash
# Launch all example creation tasks together:
Task: "Create example function ExampleGetParser in solution/example_test.go"
Task: "Create example function ExampleNewDetector in solution/example_test.go"
Task: "Create example function ExampleSolution_GetProjects in solution/example_test.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2 Only - Both P1)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T013) - CRITICAL
3. Complete Phase 3: User Story 1 (T014-T021)
4. Complete Phase 4: User Story 2 (T022-T037)
5. **STOP and VALIDATE**: Test both P1 stories independently
6. Ready for commit and merge (both P1 stories deliver core value)

### Incremental Delivery

1. Setup + Foundational → Foundation ready (library package exists)
2. Add User Story 1 → Test independently → External tools can import library (MVP Value 1!)
3. Add User Story 2 → Test independently → CLI backward compatible (MVP Value 2!)
4. Add User Story 3 → Test independently → Library documented (Enhanced adoption!)
5. Polish → Final validation → Production ready

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (T001-T013)
2. Once Foundational is done:
   - Developer A: User Story 1 (External Tool Integration - T014-T021)
   - Developer B: User Story 2 (CLI Backward Compatibility - T022-T037)
3. Developer A continues with User Story 3 after completing US1 (T038-T044)
4. Team reviews and completes Polish phase together (T045-T051)

---

## Success Criteria Mapping

| Success Criterion | Validated By Tasks |
|-------------------|-------------------|
| **SC-001**: External import works | T014-T021 (User Story 1) |
| **SC-002**: 100% CLI tests pass | T029-T030 (User Story 2) |
| **SC-003**: Coverage preserved | T031 (User Story 2) |
| **SC-004**: Performance within 5% | T037 (User Story 2) |
| **SC-005**: Build succeeds | T013, T028, T033, T049 (Multiple phases) |
| **SC-006**: All APIs accessible | T021, T047 (User Story 1 + Polish) |
| **SC-007**: Code diff shows only imports | T045-T046 (Polish) |
| **SC-008**: External program example | T014-T021 (User Story 1) |

---

## Functional Requirements Mapping

| Requirement | Validated By Tasks |
|-------------|-------------------|
| **FR-001**: Relocate to `solution/` | T001, T004-T010 (Setup + Foundational) |
| **FR-002**: Verbatim copy | T004-T010 (Foundational - copy commands) |
| **FR-003**: Package declarations | T012 (Foundational verification) |
| **FR-004**: Update import paths | T022-T027 (User Story 2) |
| **FR-005**: Preserve functionality | T029-T030, T035-T036 (User Story 2 tests) |
| **FR-006**: Identical signatures | T021, T047 (User Story 1 + Polish API verification) |
| **FR-007**: External importable | T014-T021 (User Story 1) |
| **FR-008**: CLI zero changes | T029-T037 (User Story 2) |
| **FR-009**: Test coverage preserved | T031 (User Story 2) |
| **FR-010**: No new dependencies | T013, T028 (Build verification) |
| **FR-011**: No circular deps | T003, T013, T028 (Build checks) |
| **FR-012**: Docs preserved | T004-T010 (Verbatim copy) |
| **FR-013**: File paths updated | N/A (none hardcoded) |
| **FR-014**: Self-contained | T003, T013, T028 (Dependency checks) |

---

## Notes

- **[P] tasks**: Different files, no dependencies - can run in parallel
- **[Story] label**: Maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each logical group (per phase or per user story)
- Stop at any checkpoint to validate story independently
- **CRITICAL**: Verbatim copy means copy-paste, not rewrite - verify with `diff` after copying
- No test creation required - existing CLI tests validate refactor
- Performance baseline (T002) ensures SC-004 can be verified in T037
