# Tasks: v0.1.0 Release Preparation

**Input**: Design documents from `/specs/005-v0-1-0-release/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: No new automated tests required - validation through manual checklist and existing test suite pass requirements.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

**Gonuget Structure**: Repository root has library packages and CLI under `cmd/gonuget/`.

---

## Phase 1: Setup (Preparation)

**Purpose**: Verify prerequisites and prepare for implementation

- [ ] T001 Verify Go version is 1.23 or later with `go version`
- [ ] T002 Install goreleaser with `go install github.com/goreleaser/goreleaser@latest`
- [ ] T003 Verify goreleaser installation with `goreleaser --version`
- [ ] T004 Verify all existing tests pass with `make test`
- [ ] T005 Verify lint passes with `make lint`
- [ ] T006 Verify clean git working directory with `git status`

**Checkpoint**: Prerequisites satisfied - ready for implementation

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: No foundational tasks - all user stories can proceed independently after setup

**Note**: User stories have minimal dependencies. US3 (Versioning) should complete before US4 (Checklist) but can start in parallel.

---

## Phase 3: User Story 1 - Automated CHANGELOG Generation (Priority: P1) 🎯 MVP

**Goal**: Generate CHANGELOG.md from git history using custom shell script

**Independent Test**: Run `./scripts/generate-changelog.sh --dry-run` and verify output contains all commits from git history categorized into Features, Fixes, Performance, Tests, Documentation, Refactoring, and Uncategorized sections.

### Implementation for User Story 1

- [ ] T007 [US1] Create scripts directory with `mkdir -p /Users/brandon/src/gonuget/scripts`
- [ ] T008 [US1] Create generate-changelog.sh script skeleton in `/Users/brandon/src/gonuget/scripts/generate-changelog.sh`
- [ ] T009 [US1] Implement argument parsing (--from-commit, --to-commit, --output, --version, --dry-run, --help) in generate-changelog.sh
- [ ] T010 [US1] Implement conventional commit regex parsing logic in generate-changelog.sh (see contracts/changelog-script.md)
- [ ] T011 [US1] Implement commit categorization logic (feat→Features, fix→Bug Fixes, etc.) in generate-changelog.sh
- [ ] T012 [US1] Implement uncategorized section for non-conventional commits in generate-changelog.sh
- [ ] T013 [US1] Implement markdown generation with Keep a Changelog format in generate-changelog.sh
- [ ] T014 [US1] Implement version link generation at footer in generate-changelog.sh
- [ ] T015 [US1] Make script executable with `chmod +x /Users/brandon/src/gonuget/scripts/generate-changelog.sh`
- [ ] T016 [US1] Test script with dry-run mode: `./scripts/generate-changelog.sh --dry-run`
- [ ] T017 [US1] Generate actual CHANGELOG.md with `./scripts/generate-changelog.sh --version "0.1.0"`
- [ ] T018 [US1] Manually review CHANGELOG.md for accuracy and completeness
- [ ] T019 [US1] Fix any categorization issues in Uncategorized section
- [ ] T020 [US1] Verify CHANGELOG.md matches Keep a Changelog format with `grep -q "## \[0.1.0\]" CHANGELOG.md`

**Checkpoint**: CHANGELOG.md generated, reviewed, and accurate ✅ SC-001

---

## Phase 4: User Story 2 - Modern CI/CD Pipeline (Priority: P1) 🎯 MVP

**Goal**: Update CI/CD pipeline to 2025 best practices with GitHub Actions and goreleaser

**Independent Test**: Trigger CI workflow and verify it runs lint, tests on all platforms (Linux, macOS, Windows), builds binaries, and produces artifacts (checksums).

### Implementation for User Story 2

#### CI Workflow Update

- [ ] T021 [P] [US2] Update `.github/workflows/ci.yml` to use latest actions (setup-go@v5, checkout@v4)
- [ ] T022 [US2] Add lint job with golangci-lint-action@v8 in ci.yml
- [ ] T023 [US2] Add test matrix for 3 platforms (ubuntu-latest, macos-latest, windows-latest) in ci.yml
- [ ] T024 [US2] Configure Go version to 'stable' in ci.yml setup-go step
- [ ] T025 [US2] Add race detector to test job with `go test -race ./...` in ci.yml
- [ ] T026 [US2] Add test coverage reporting in ci.yml (optional enhancement)
- [ ] T027 [US2] Commit ci.yml changes with `git add .github/workflows/ci.yml && git commit -m "ci: update CI workflow to modern Go best practices"`
- [ ] T028 [US2] Push to trigger CI and verify all jobs pass on all platforms

#### goreleaser Configuration

- [ ] T029 [P] [US2] Create `.goreleaser.yml` in repository root (see contracts/goreleaser-config.md)
- [ ] T030 [US2] Configure builds section with 5 target platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64) in .goreleaser.yml
- [ ] T031 [US2] Configure ldflags for version injection in .goreleaser.yml
- [ ] T032 [US2] Configure archives section with tar.gz for Unix, zip for Windows in .goreleaser.yml
- [ ] T033 [US2] Configure checksum generation (SHA256) in .goreleaser.yml
- [ ] T034 [US2] Configure GitHub release integration in .goreleaser.yml
- [ ] T035 [US2] Disable auto-generated changelog (use manual CHANGELOG.md) in .goreleaser.yml
- [ ] T036 [US2] Validate goreleaser config with `goreleaser check`
- [ ] T037 [US2] Test local snapshot build with `goreleaser build --snapshot --clean`
- [ ] T038 [US2] Verify all 5 binaries built successfully in dist/ directory
- [ ] T039 [US2] Test one binary to ensure it runs: `dist/gonuget_linux_amd64_v1/gonuget --help`

#### Release Workflow Creation

- [ ] T040 [P] [US2] Create `.github/workflows/release.yml` for tag-triggered releases
- [ ] T041 [US2] Configure release workflow to trigger on tag push (v*.*.*)  in release.yml
- [ ] T042 [US2] Add goreleaser-action@v6 step in release.yml
- [ ] T043 [US2] Configure GITHUB_TOKEN permissions (contents: write) in release.yml
- [ ] T044 [US2] Add full git history fetch (fetch-depth: 0) in release.yml for goreleaser

**Checkpoint**: CI/CD pipeline modernized, goreleaser configured, all platforms build successfully ✅ SC-002, SC-003

---

## Phase 5: User Story 3 - Semantic Versioning Infrastructure (Priority: P2)

**Goal**: Implement version command and build-time version embedding

**Independent Test**: Run `gonuget version` command and verify it outputs v0.1.0, check version.go file contains correct version constant, verify versioning policy is documented in CONTRIBUTING.md.

### Implementation for User Story 3

#### Version Package

- [ ] T045 [US3] Create version package directory with `mkdir -p /Users/brandon/src/gonuget/cmd/gonuget/version`
- [ ] T046 [US3] Create `/Users/brandon/src/gonuget/cmd/gonuget/version/version.go` with package-level variables (Version, Commit, Date)
- [ ] T047 [US3] Implement `Info()` function in version/version.go that returns formatted version string
- [ ] T048 [US3] Add default values ("dev", "none", "unknown") to version variables in version.go
- [ ] T049 [US3] Add godoc comments to all exported variables and functions in version/version.go

#### Version Command

- [ ] T050 [US3] Create `/Users/brandon/src/gonuget/cmd/gonuget/commands/version.go` for version command
- [ ] T051 [US3] Implement Cobra command for `gonuget version` in commands/version.go
- [ ] T052 [US3] Import version package and call `version.Info()` in version command
- [ ] T053 [US3] Register version command with root command in cmd/gonuget/commands/root.go or main.go
- [ ] T054 [US3] Test version command with development build: `go run ./cmd/gonuget version`
- [ ] T055 [US3] Verify output shows "dev (commit: none, built: unknown)"

#### Makefile Integration

- [ ] T056 [US3] Update `/Users/brandon/src/gonuget/Makefile` with VERSION, COMMIT, DATE variables
- [ ] T057 [US3] Add LDFLAGS variable with -X flags for version injection in Makefile
- [ ] T058 [US3] Update build target to use LDFLAGS in Makefile
- [ ] T059 [US3] Test make build with version injection: `make build && ./gonuget version`
- [ ] T060 [US3] Verify version output shows actual commit SHA and build date

#### Documentation

- [ ] T061 [P] [US3] Create or update `/Users/brandon/src/gonuget/CONTRIBUTING.md` with versioning policy section
- [ ] T062 [US3] Document semantic versioning rules (MAJOR.MINOR.PATCH) in CONTRIBUTING.md
- [ ] T063 [US3] Document pre-1.0 release semantics (v0.x.x allows breaking changes) in CONTRIBUTING.md
- [ ] T064 [US3] Document release process steps in CONTRIBUTING.md

**Checkpoint**: Version command implemented, Makefile configured, versioning policy documented ✅ SC-004

---

## Phase 6: User Story 4 - Release Execution Checklist (Priority: P3)

**Goal**: Create comprehensive release checklist for repeatable release process

**Independent Test**: Follow release checklist step-by-step and successfully publish v0.1.0 release with all artifacts and announcements (or dry-run validation).

### Implementation for User Story 4

- [ ] T065 [US4] Create `/Users/brandon/src/gonuget/RELEASE_CHECKLIST.md` file
- [ ] T066 [US4] Add pre-release validation section (tests, lint, build, interop tests) to RELEASE_CHECKLIST.md
- [ ] T067 [US4] Add release execution section (tag creation, push, monitor CI) to RELEASE_CHECKLIST.md
- [ ] T068 [US4] Add post-release verification section (download binaries, verify checksums) to RELEASE_CHECKLIST.md
- [ ] T069 [US4] Add post-release announcement section (GitHub Discussions, etc.) to RELEASE_CHECKLIST.md
- [ ] T070 [US4] Update `/Users/brandon/src/gonuget/README.md` with installation instructions for v0.1.0 binaries
- [ ] T071 [US4] Add download links for all 5 platforms in README.md
- [ ] T072 [US4] Add checksum verification instructions in README.md
- [ ] T073 [US4] Verify checklist is complete by walking through each item

**Checkpoint**: Release checklist created and validated ✅ SC-007

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and integration testing across all user stories

- [ ] T074 [P] Verify CHANGELOG.md exists and is properly formatted
- [ ] T075 [P] Verify goreleaser config passes validation: `goreleaser check`
- [ ] T076 [P] Verify all CI workflows have valid YAML syntax
- [ ] T077 Run full local build with version injection: `make build`
- [ ] T078 Verify version command output: `./gonuget version | grep -q "dev"`
- [ ] T079 Run all existing tests: `make test`
- [ ] T080 Run interop tests: `make test-interop`
- [ ] T081 Run lint: `make lint`
- [ ] T082 Verify all 550 interop tests pass
- [ ] T083 Test goreleaser snapshot build: `goreleaser build --snapshot --clean`
- [ ] T084 Verify all 5 platform binaries exist in dist/ directory
- [ ] T085 Test each platform binary runs: `dist/gonuget_*/gonuget version`
- [ ] T086 Commit all changes (CHANGELOG, workflows, version code, docs) with appropriate commit messages
- [ ] T087 Verify git working directory is clean: `git status`
- [ ] T088 Review all documentation updates (README, CONTRIBUTING, RELEASE_CHECKLIST)
- [ ] T089 Verify CI passes on feature branch before final release

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: No tasks - all user stories are independent
- **User Story 1 (CHANGELOG - Phase 3)**: Depends only on Setup (T001-T006)
- **User Story 2 (CI/CD - Phase 4)**: Depends only on Setup (T001-T006)
- **User Story 3 (Versioning - Phase 5)**: Depends only on Setup (T001-T006)
- **User Story 4 (Checklist - Phase 6)**: Should run after US3 (needs version command reference), but can start in parallel
- **Polish (Phase 7)**: Depends on all P1 user stories (US1, US2, US3) being complete

### User Story Dependencies

- **User Story 1 (P1 - CHANGELOG)**: Independent - can start after Setup
- **User Story 2 (P1 - CI/CD)**: Independent - can start after Setup
- **User Story 3 (P2 - Versioning)**: Independent - can start after Setup
- **User Story 4 (P3 - Checklist)**: Weak dependency on US3 (references version command) - can start in parallel

### Within Each User Story

**User Story 1 (CHANGELOG Generation)**:
1. Create script skeleton (T007-T008)
2. Implement parsing and categorization (T009-T014)
3. Test and generate (T015-T020)

**User Story 2 (CI/CD Pipeline)**:
1. CI workflow (T021-T028) - can run in parallel with goreleaser tasks
2. goreleaser config (T029-T039) - can run in parallel with CI tasks
3. Release workflow (T040-T044) - depends on goreleaser config

**User Story 3 (Versioning Infrastructure)**:
1. Version package (T045-T049) - can run in parallel with version command
2. Version command (T050-T055) - can run in parallel with version package
3. Makefile integration (T056-T060) - depends on version package
4. Documentation (T061-T064) - can run in parallel with code tasks

**User Story 4 (Release Checklist)**:
1. Create checklist (T065-T069) - can run in parallel with README updates
2. Update README (T070-T073) - can run in parallel with checklist

### Parallel Opportunities

**Phase 1 (Setup)**:
- T001-T006 should run sequentially (verification steps)

**Phase 3 (User Story 1 - CHANGELOG)**:
- T007-T014 run sequentially (script development)
- T015-T020 run sequentially (testing and validation)

**Phase 4 (User Story 2 - CI/CD)**:
- T021-T028 (CI workflow) can run IN PARALLEL with T029-T039 (goreleaser config)
- T040-T044 (release workflow) runs after goreleaser config validated

**Phase 5 (User Story 3 - Versioning)**:
- T045-T049 (version package) can run IN PARALLEL with T050-T055 (version command)
- T061-T064 (documentation) can run IN PARALLEL with T045-T060 (code)

**Phase 6 (User Story 4 - Checklist)**:
- T065-T069 (checklist) can run IN PARALLEL with T070-T073 (README)

**Phase 7 (Polish)**:
- T074-T076 can run IN PARALLEL (different file verifications)
- T077-T089 run sequentially (integration testing)

**Cross-Phase Parallelization**:
- Once Setup (Phase 1) completes, User Story 1, 2, and 3 can ALL run in parallel by different developers
- User Story 4 can run in parallel with others (weak dependency on US3)

---

## Parallel Example: User Story 2 (CI/CD)

```bash
# Launch CI workflow and goreleaser config tasks in parallel:
Task: "Update .github/workflows/ci.yml to use latest actions"
Task: "Create .goreleaser.yml in repository root"

# Both can proceed independently, then converge for testing
```

## Parallel Example: User Story 3 (Versioning)

```bash
# Launch version package and command tasks in parallel:
Task: "Create version package directory"
Task: "Create commands/version.go for version command"

# Documentation can also run in parallel:
Task: "Update CONTRIBUTING.md with versioning policy section"
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2 Only - Both P1)

1. Complete Phase 1: Setup (T001-T006)
2. Complete Phase 3: User Story 1 - CHANGELOG (T007-T020)
3. Complete Phase 4: User Story 2 - CI/CD (T021-T044)
4. **STOP and VALIDATE**: Test both P1 stories independently
5. Run Polish tasks T074-T089 for MVP scope only
6. Ready for optional release or continue to US3/US4

### Incremental Delivery

1. Setup → Foundation ready
2. Add User Story 1 → Test independently → CHANGELOG generation works (MVP Value 1!)
3. Add User Story 2 → Test independently → CI/CD modernized (MVP Value 2!)
4. Add User Story 3 → Test independently → Version command available (Enhanced!)
5. Add User Story 4 → Test independently → Release process documented (Complete!)
6. Polish → Final validation → Production ready

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup together (T001-T006)
2. Once Setup is done:
   - Developer A: User Story 1 (CHANGELOG - T007-T020)
   - Developer B: User Story 2 (CI/CD - T021-T044)
   - Developer C: User Story 3 (Versioning - T045-T064)
3. Developer D can work on User Story 4 after US3 basics complete (T065-T073)
4. Team reviews and completes Polish phase together (T074-T089)

---

## Success Criteria Mapping

| Success Criterion | Validated By Tasks |
|-------------------|----------------------|
| **SC-001**: CHANGELOG.md generated and categorized correctly | T007-T020 (User Story 1) |
| **SC-002**: CI workflow completes in <15 minutes on all platforms | T021-T028 (User Story 2 - CI) |
| **SC-003**: Release workflow produces binaries for all 5 platforms with checksums | T029-T044 (User Story 2 - goreleaser + release) |
| **SC-004**: `gonuget version` outputs correct version info | T045-T064 (User Story 3) |
| **SC-005**: All pre-release validation checks pass | T079-T082 (Polish) |
| **SC-006**: GitHub release contains all required artifacts | T029-T044 (User Story 2) + T083-T085 (Polish validation) |
| **SC-007**: Release process executable via checklist | T065-T073 (User Story 4) |

---

## Functional Requirements Mapping

| Requirement | Validated By Tasks |
|-------------|----------------------|
| **FR-001**: Generate CHANGELOG.md from git history | T007-T020 (User Story 1) |
| **FR-002**: CHANGELOG categorizes commits (Features, Fixes, etc., Uncategorized) | T010-T012 (User Story 1 - categorization) |
| **FR-003**: Use shell script with git log parsing | T008-T014 (User Story 1 - script implementation) |
| **FR-004**: CI runs on Linux, macOS, Windows with Go 1.23+ | T021-T028 (User Story 2 - CI workflow) |
| **FR-005**: CI executes linting, unit tests, integration tests | T022, T023 (User Story 2 - lint + test jobs) |
| **FR-006**: Use goreleaser for cross-platform builds | T029-T039 (User Story 2 - goreleaser config) |
| **FR-007**: Generate SHA256 checksums | T033 (User Story 2 - checksum config) |
| **FR-008**: Automatically create GitHub release | T034, T040-T044 (User Story 2 - release automation) |
| **FR-009**: Embed version info using -ldflags | T031, T056-T060 (User Story 2 + 3 - ldflags) |
| **FR-010**: CLI provides version command | T050-T055 (User Story 3 - version command) |
| **FR-011**: Document semantic versioning policy | T061-T064 (User Story 3 - CONTRIBUTING.md) |
| **FR-012**: Release checklist includes pre-release validation | T066 (User Story 4 - checklist validation section) |
| **FR-013**: Release checklist includes documentation updates | T070-T072 (User Story 4 - README updates) |
| **FR-014**: Release checklist includes tag creation and publishing | T067 (User Story 4 - release execution section) |
| **FR-015**: All workflows are idempotent | Design validated in contracts/ (no specific task) |
| **FR-016**: CHANGELOG follows Keep a Changelog format | T013 (User Story 1 - markdown generation) |
| **FR-017**: Release artifacts for all 5 platforms (any failure fails release) | T030, T038 (User Story 2 - platform config + validation) |

---

## Notes

- **[P] tasks**: Different files, no dependencies - can run in parallel
- **[Story] label**: Maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Stop at any checkpoint to validate story independently
- **No automated tests required**: Validation through manual checklist and existing test suite
- Performance validation (SC-002 <15 min CI) happens during T028 (CI push and monitor)
- All file paths are absolute for clarity
