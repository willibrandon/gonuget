# Quickstart: v0.1.0 Release Preparation

**Feature**: 005-v0-1-0-release
**Branch**: `005-v0-1-0-release`
**Purpose**: Prepare gonuget for first official release

## Prerequisites

Before starting implementation:

1. **All existing tests pass**
   ```bash
   make test                # All tests (Go + interop)
   make lint                # Linting passes
   ```

2. **Clean working directory**
   ```bash
   git status               # No uncommitted changes
   ```

3. **Latest stable Go installed**
   ```bash
   go version               # Should be 1.23 or later
   ```

4. **goreleaser installed**
   ```bash
   go install github.com/goreleaser/goreleaser@latest
   goreleaser --version
   ```

## Implementation Order

### Phase 1: Version Infrastructure (30 minutes)

**Goal**: Add version package and command to CLI

1. **Create version package**
   ```bash
   mkdir -p cmd/gonuget/version
   ```

2. **Implement version.go** (see `contracts/version-package.md`)
   ```go
   // cmd/gonuget/version/version.go
   package version

   var (
       Version = "dev"
       Commit  = "none"
       Date    = "unknown"
   )

   func Info() string {
       return fmt.Sprintf("gonuget version %s (commit: %s, built: %s)",
           Version, Commit, Date)
   }
   ```

3. **Add version command**
   ```go
   // cmd/gonuget/commands/version.go
   var versionCmd = &cobra.Command{
       Use:   "version",
       Short: "Show gonuget version information",
       Run: func(cmd *cobra.Command, args []string) {
           fmt.Println(version.Info())
       },
   }
   ```

4. **Update Makefile**
   ```makefile
   VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
   COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
   DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
   LDFLAGS := -ldflags "-X github.com/willibrandon/gonuget/cmd/gonuget/version.Version=$(VERSION) \
                         -X github.com/willibrandon/gonuget/cmd/gonuget/version.Commit=$(COMMIT) \
                         -X github.com/willibrandon/gonuget/cmd/gonuget/version.Date=$(DATE)"

   build:
       go build $(LDFLAGS) -o gonuget ./cmd/gonuget
   ```

5. **Test version command**
   ```bash
   make build
   ./gonuget version
   # Expected: "gonuget version dev (commit: <sha>, built: <date>)"
   ```

**Checkpoint**: Version command works with build-time injection

---

### Phase 2: CHANGELOG Generation (45 minutes)

**Goal**: Create script to generate CHANGELOG.md from git history

1. **Create script directory**
   ```bash
   mkdir -p scripts
   ```

2. **Implement generate-changelog.sh** (see `contracts/changelog-script.md`)
   ```bash
   #!/bin/bash
   # Parse conventional commits and generate Keep a Changelog format
   # See contracts/changelog-script.md for full implementation
   ```

3. **Make executable**
   ```bash
   chmod +x scripts/generate-changelog.sh
   ```

4. **Test generation**
   ```bash
   ./scripts/generate-changelog.sh --dry-run
   # Review output for correct categorization
   ```

5. **Generate actual CHANGELOG**
   ```bash
   ./scripts/generate-changelog.sh --version "0.1.0"
   # Review CHANGELOG.md
   cat CHANGELOG.md
   ```

6. **Manual review**
   - Check "Uncategorized" section
   - Fix any typos or unclear descriptions
   - Ensure all major features included

**Checkpoint**: CHANGELOG.md generated and reviewed

---

### Phase 3: goreleaser Configuration (30 minutes)

**Goal**: Configure goreleaser for multi-platform builds

1. **Create .goreleaser.yml** (see `contracts/goreleaser-config.md`)
   ```yaml
   project_name: gonuget
   builds:
     - id: gonuget
       main: ./cmd/gonuget
       binary: gonuget
       env:
         - CGO_ENABLED=0
       goos:
         - linux
         - darwin
         - windows
       goarch:
         - amd64
         - arm64
       # ... see full config in contracts/goreleaser-config.md
   ```

2. **Validate configuration**
   ```bash
   goreleaser check
   # Expected: "your config is valid"
   ```

3. **Test snapshot build**
   ```bash
   goreleaser build --snapshot --clean
   ls -lh dist/
   # Expected: 5 binaries (linux/darwin/windows, amd64/arm64)
   ```

4. **Test version injection**
   ```bash
   dist/gonuget_linux_amd64_v1/gonuget version
   # Expected: version info with snapshot tag
   ```

**Checkpoint**: goreleaser builds all platforms successfully

---

### Phase 4: GitHub Actions Workflows (45 minutes)

**Goal**: Update CI and add release workflow

1. **Update CI workflow**
   ```yaml
   # .github/workflows/ci.yml
   name: CI

   on:
     push:
       branches: [main]
     pull_request:

   jobs:
     lint:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
         - uses: actions/setup-go@v5
           with:
             go-version: stable
         - uses: golangci/golangci-lint-action@v8

     test:
       runs-on: ${{ matrix.os }}
       strategy:
         matrix:
           os: [ubuntu-latest, macos-latest, windows-latest]
       steps:
         - uses: actions/checkout@v4
         - uses: actions/setup-go@v5
           with:
             go-version: stable
         - run: make test
         - run: go test -race ./...
   ```

2. **Create release workflow**
   ```yaml
   # .github/workflows/release.yml
   name: Release

   on:
     push:
       tags:
         - 'v*.*.*'

   permissions:
     contents: write

   jobs:
     release:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
           with:
             fetch-depth: 0
         - uses: actions/setup-go@v5
           with:
             go-version: stable
         - uses: goreleaser/goreleaser-action@v6
           with:
             distribution: goreleaser
             version: latest
             args: release --clean
           env:
             GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
   ```

3. **Test CI workflow** (commit and push to trigger)
   ```bash
   git add .github/workflows/ci.yml
   git commit -m "ci: update CI workflow to modern Go best practices"
   git push
   # Wait for CI to run, verify all platforms pass
   ```

**Checkpoint**: CI workflow passes on all platforms

---

### Phase 5: Documentation Updates (20 minutes)

**Goal**: Document versioning policy

1. **Update CONTRIBUTING.md**
   ```markdown
   ## Versioning Policy

   gonuget follows [Semantic Versioning 2.0.0](https://semver.org/):

   - **MAJOR**: Incompatible API changes
   - **MINOR**: New functionality (backward compatible)
   - **PATCH**: Bug fixes (backward compatible)

   ### Pre-1.0 Releases

   - v0.x.x releases signal API is not yet stable
   - Minor version bumps MAY include breaking changes
   - v1.0.0 marks API stability commitment

   ### Release Process

   1. All tests pass (`make test`)
   2. Generate CHANGELOG (`./scripts/generate-changelog.sh`)
   3. Update version references
   4. Create git tag (`git tag -a v0.1.0 -m "Release v0.1.0"`)
   5. Push tag (`git push origin v0.1.0`)
   6. CI automatically builds and publishes release
   ```

2. **Update README.md** (installation instructions)
   ```markdown
   ## Installation

   ### Pre-built Binaries

   Download the latest release for your platform:

   **Linux (amd64)**:
   ```bash
   curl -sL https://github.com/willibrandon/gonuget/releases/download/v0.1.0/gonuget-v0.1.0-linux-amd64.tar.gz | tar xz
   sudo mv gonuget /usr/local/bin/
   ```

   **macOS (arm64, Apple Silicon)**:
   ```bash
   curl -sL https://github.com/willibrandon/gonuget/releases/download/v0.1.0/gonuget-v0.1.0-darwin-arm64.tar.gz | tar xz
   sudo mv gonuget /usr/local/bin/
   ```

   See [Releases](https://github.com/willibrandon/gonuget/releases) for all platforms.
   ```

**Checkpoint**: Documentation updated

---

### Phase 6: Release Checklist (15 minutes)

**Goal**: Create release execution checklist

1. **Create RELEASE_CHECKLIST.md**
   ```markdown
   # Release Checklist: v0.1.0

   ## Pre-Release Validation

   - [ ] All tests pass locally (`make test`)
   - [ ] All tests pass on CI (all platforms)
   - [ ] Lint passes (`make lint`)
   - [ ] All 550 interop tests pass (`make test-interop`)
   - [ ] Manual smoke testing complete
   - [ ] CHANGELOG.md reviewed and accurate

   ## Release Execution

   - [ ] Update version references (if any hardcoded)
   - [ ] Commit CHANGELOG.md
   - [ ] Create annotated git tag: `git tag -a v0.1.0 -m "Release v0.1.0"`
   - [ ] Push tag: `git push origin v0.1.0`
   - [ ] Monitor CI release workflow
   - [ ] Verify GitHub release created with all artifacts
   - [ ] Download and test binaries for each platform
   - [ ] Verify checksums match

   ## Post-Release

   - [ ] Announce release (GitHub Discussions, Twitter, etc.)
   - [ ] Update project board/milestones
   - [ ] Close release milestone
   - [ ] Create v0.2.0 milestone (if applicable)
   ```

**Checkpoint**: Release checklist created

---

## Validation Steps

### Local Validation
```bash
# 1. Version command works
make build
./gonuget version | grep -q "v0.1.0" || echo "Version not set"

# 2. CHANGELOG exists and is valid
test -f CHANGELOG.md
grep -q "## \[0.1.0\]" CHANGELOG.md

# 3. goreleaser config valid
goreleaser check

# 4. All platforms build
goreleaser build --snapshot --clean
ls dist/ | grep -c gonuget  # Should be 5

# 5. CI workflow valid
yamllint .github/workflows/*.yml
```

### Pre-Release Validation
```bash
# 1. All tests pass
make test

# 2. Interop tests pass
make test-interop

# 3. Lint passes
make lint

# 4. No uncommitted changes
git status | grep "nothing to commit"

# 5. CHANGELOG reviewed
cat CHANGELOG.md
```

## Release Execution

### Step 1: Final Commit
```bash
git add CHANGELOG.md CONTRIBUTING.md README.md
git commit -m "docs: prepare for v0.1.0 release"
git push
```

### Step 2: Wait for CI
```bash
# Wait for CI to pass on main branch
# Check: https://github.com/willibrandon/gonuget/actions
```

### Step 3: Create Tag
```bash
git tag -a v0.1.0 -m "Release v0.1.0

First official release of gonuget with:
- Complete NuGet.Client parity for core operations
- Solution file support
- Transitive dependency resolution
- Modern CLI command structure"

git push origin v0.1.0
```

### Step 4: Monitor Release
```bash
# Watch release workflow
# Check: https://github.com/willibrandon/gonuget/actions/workflows/release.yml

# Verify release created
# Check: https://github.com/willibrandon/gonuget/releases/tag/v0.1.0
```

### Step 5: Verify Artifacts
```bash
# Download and test each platform
curl -sL https://github.com/willibrandon/gonuget/releases/download/v0.1.0/checksums.txt
curl -sL https://github.com/willibrandon/gonuget/releases/download/v0.1.0/gonuget-v0.1.0-linux-amd64.tar.gz | tar xz
./gonuget version

# Verify checksum
sha256sum gonuget-v0.1.0-linux-amd64.tar.gz | grep -f checksums.txt
```

## Troubleshooting

### goreleaser Fails
```bash
# Check detailed error
goreleaser release --clean --skip=publish

# Common issues:
# 1. Missing GITHUB_TOKEN: Set in CI secrets
# 2. Invalid tag format: Must be v*.*.*
# 3. Dirty git tree: Commit all changes
```

### CI Timeout
```bash
# Check workflow timeout (15 minutes max)
# Optimize test parallelization if needed
# Check matrix builds run in parallel
```

### Version Not Injected
```bash
# Verify ldflags in Makefile
make build
./gonuget version | grep "dev"  # Should NOT be "dev" for release

# Check goreleaser ldflags match package path exactly
```

## Estimated Timeline

- **Phase 1** (Version): 30 minutes
- **Phase 2** (CHANGELOG): 45 minutes
- **Phase 3** (goreleaser): 30 minutes
- **Phase 4** (CI/CD): 45 minutes
- **Phase 5** (Docs): 20 minutes
- **Phase 6** (Checklist): 15 minutes

**Total Implementation**: ~3 hours

**Plus validation and release**: +1 hour

**Total**: ~4 hours for complete v0.1.0 release preparation
