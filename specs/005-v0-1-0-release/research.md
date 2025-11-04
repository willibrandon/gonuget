# v0.1.0 Release Preparation Research

This document captures research findings for gonuget v0.1.0 release preparation, focusing on modern (2025) best practices for Go project releases.

## 1. goreleaser Best Practices for Go Projects (2025)

### Decision

Use goreleaser v2 with the following configuration approach:
- YAML configuration file at `.goreleaser.yaml` checked into source control
- Multi-platform builds for: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- SHA256 checksums for all artifacts
- GitHub release integration with automatic changelog generation
- Build-time ldflags injection for version, commit SHA, and build date

### Rationale

**Configuration Management**: goreleaser documentation explicitly states "It is best practice to check .goreleaser.yaml into the source control." This ensures reproducible builds and makes the release process transparent to contributors.

**Platform Coverage**: The selected platforms cover:
- Linux: Primary server/container platform (amd64 + arm64 for cloud/edge)
- macOS: Developer workstations (Intel + Apple Silicon)
- Windows: Enterprise development environments

**Security & Verification**: SHA256 checksums are industry standard for binary verification and are the default algorithm in goreleaser. They enable users to verify download integrity.

**Automation**: GitHub release integration streamlines the release workflow by:
- Creating releases automatically on git tags
- Attaching binaries and checksums
- Generating changelog from commits
- Supporting markdown release notes

### Configuration Pattern

```yaml
# .goreleaser.yaml
builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.FullCommit}}
      - -X main.date={{.Date}}

checksum:
  algorithm: sha256
  name_template: 'checksums.txt'

archives:
  - format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
    format_overrides:
      - goos: windows
        format: zip
```

**Key Features**:
- `CGO_ENABLED=0`: Produces static binaries without C dependencies
- `ignore`: Excludes unsupported platform combinations (Windows ARM64)
- `-s -w`: Strips debug info and symbol table to reduce binary size
- Format override: Uses ZIP for Windows (convention), tar.gz for Unix

### Alternatives Considered

**Manual Builds with Makefile**: Rejected because:
- More maintenance burden (manual platform matrix)
- No automatic GitHub release integration
- No built-in checksum generation
- Inconsistent artifact naming

**GitHub Actions Build Matrix Only**: Rejected because:
- Duplicates goreleaser functionality
- Requires custom logic for checksums, archives, releases
- Less standardized output format

**TaskFile or Just**: Rejected because:
- goreleaser is purpose-built for Go releases
- Better GitHub ecosystem integration
- More mature changelog/release note generation

### References

- [GoReleaser Official Documentation](https://goreleaser.com/)
- [GoReleaser Quick Start](https://goreleaser.com/quick-start/)
- [GoReleaser Builds Configuration](https://goreleaser.com/customization/builds/go/)
- [GoReleaser Checksum Docs](https://goreleaser.com/customization/checksum/)
- [2025 Multi-Arch Docker with GoReleaser](https://schoenwald.aero/posts/2025-01-25_effortless-multi-arch-docker-images-with-goreleaser-and-github-actions/)
- [2025 GitLab + GoReleaser Guide](https://containerinfra.nl/blog/2025/01/26/using-goreleaser-with-gitlab-multi-arch-builds-cosign-and-sbom-generation/)

## 2. Keep a Changelog Format Specification

### Decision

Follow Keep a Changelog v1.1.0 specification with these requirements:
- Date format: ISO 8601 (YYYY-MM-DD)
- Standard categories: Added, Changed, Deprecated, Removed, Fixed, Security
- Version header format: `## [Version] - YYYY-MM-DD`
- Comparison links at document footer

### Rationale

**Standardization**: Keep a Changelog is the de facto standard for changelog formatting in open source projects. Using it ensures:
- Familiar format for users
- Machine-parseable structure
- Clear semantic organization

**Date Format**: ISO 8601 (YYYY-MM-DD) is chosen because:
- Unambiguous across locales
- Sorts correctly lexicographically
- International standard
- No cultural date format conflicts (MM/DD vs DD/MM)

**Category Standardization**: The six standard categories cover all change types:
- **Added**: New features (maps to `feat:` commits)
- **Changed**: Modifications to existing functionality
- **Deprecated**: Features marked for removal
- **Removed**: Deleted features (breaking changes)
- **Fixed**: Bug fixes (maps to `fix:` commits)
- **Security**: Security-related fixes (critical visibility)

**Version Links**: Comparison links enable users to:
- View all commits between versions
- Understand detailed code changes
- Verify changelog completeness

### Format Structure

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New features in development

## [1.0.0] - 2025-01-15

### Added
- Initial release with core features

### Fixed
- Bug fixes from beta testing

[Unreleased]: https://github.com/user/repo/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/user/repo/compare/v0.9.0...v1.0.0
```

### Alternatives Considered

**Conventional Changelog Auto-Generation**: Rejected as sole approach because:
- Commits may be too granular for user-facing changelog
- Requires editorial curation for clarity
- Technical commit messages don't always translate to user value
- However, can be used as **input** for manual curation

**GitHub Releases Only**: Rejected because:
- Changelog should exist in repository
- Enables offline access
- Single source of truth
- GitHub Releases can be generated FROM changelog

**Custom Format**: Rejected because:
- Reinventing wheel
- No tooling support
- Unfamiliar to users
- Harder to parse programmatically

### References

- [Keep a Changelog Official Site](https://keepachangelog.com/en/1.1.0/)
- [Keep a Changelog Repository](https://github.com/olivierlacan/keep-a-changelog)
- [Semantic Versioning](https://semver.org/spec/v2.0.0.html)

## 3. Conventional Commits Parsing in Bash

### Decision

Use bash built-in regex with `=~` operator and `BASH_REMATCH` array for parsing conventional commits:

```bash
#!/bin/bash

commit_msg="$1"

# Regex pattern for conventional commits
# Captures: type, scope (optional), breaking (optional), description
if [[ "$commit_msg" =~ ^([a-z]+)(\(([a-zA-Z0-9_-]+)\))?(!)?: (.+)$ ]]; then
    type="${BASH_REMATCH[1]}"
    scope="${BASH_REMATCH[3]}"  # Note: index 3, not 2 (2 is full parens group)
    breaking="${BASH_REMATCH[4]}"
    description="${BASH_REMATCH[5]}"

    echo "Type: $type"
    echo "Scope: $scope"
    echo "Breaking: $breaking"
    echo "Description: $description"
else
    echo "Not a conventional commit"
fi
```

### Rationale

**Native Bash Feature**: The `=~` operator is built into bash 3.0+ and provides:
- Zero external dependencies (no grep, sed, awk)
- Single operation for match + capture
- Array-based access to captured groups
- High performance for scripting

**Regex Pattern Breakdown**:
- `^([a-z]+)`: Commit type at start (feat, fix, etc.)
- `(\(([a-zA-Z0-9_-]+)\))?`: Optional scope in parentheses
- `(!)?`: Optional breaking change indicator
- `: (.+)$`: Colon, space, and description to end

**BASH_REMATCH Array**:
- Index 0: Full match
- Index 1: First capture group (type)
- Index 2: Second capture group (full scope with parens)
- Index 3: Third capture group (scope without parens)
- Index 4: Fourth capture group (breaking `!`)
- Index 5: Fifth capture group (description)

### Common Commit Types

Standard types to recognize:
- `feat`: New feature (MINOR version bump)
- `fix`: Bug fix (PATCH version bump)
- `docs`: Documentation changes
- `test`: Test additions or modifications
- `perf`: Performance improvements
- `refactor`: Code refactoring without behavior change
- `chore`: Build process, tooling, dependencies
- `style`: Code style changes (formatting)
- `ci`: CI/CD changes
- `build`: Build system changes

**Breaking Changes**: Indicated by either:
- `!` after type/scope: `feat!:` or `feat(api)!:`
- Footer: `BREAKING CHANGE:` (triggers MAJOR version bump)

### Handling Non-Conventional Commits

For commits that don't follow conventional format:
- Log as unstructured changes
- Place in "Changed" category by default
- Can filter out or categorize manually
- Consider requiring conventional commits via git hooks

### Alternatives Considered

**External Tools (commitizen, semantic-release)**: Rejected for parsing because:
- Adds Node.js dependency to Go project
- Overkill for simple parsing
- Bash solution is more portable

**grep/sed/awk Pipeline**: Rejected because:
- Multiple process spawns (slower)
- More complex error handling
- Less readable code
- Doesn't provide all captures in one operation

**Python/Go Script**: Rejected because:
- Unnecessary for simple pattern matching
- Adds build step or runtime dependency
- Bash is available in all CI environments

### References

- [Conventional Commits Specification](https://www.conventionalcommits.org/en/v1.0.0/)
- [Stack Overflow: Parse Commit Message into Variables](https://stackoverflow.com/questions/62317758/how-to-parse-commit-message-into-variables-using-bash)
- [Conventional Commits Regex Examples](https://regex101.com/library/JCoEea)
- [Conventional Commits Parser (npm)](https://www.npmjs.com/package/conventional-commits-parser)

## 4. Go Version Embedding with -ldflags

### Decision

Create a dedicated `cmd/gonuget/version` package with exported variables set via `-ldflags -X`:

**Package Structure**:
```go
// cmd/gonuget/version/version.go
package version

var (
    // Version is the semantic version (e.g., "v0.1.0")
    Version = "dev"

    // Commit is the git commit SHA
    Commit = "none"

    // Date is the build date in RFC3339 format
    Date = "unknown"
)

// Info returns formatted version information
func Info() string {
    return fmt.Sprintf("gonuget %s (commit: %s, built: %s)",
        Version, Commit, Date)
}
```

**ldflags Syntax**:
```bash
go build -ldflags "\
  -X 'github.com/wbreza/gonuget/cmd/gonuget/version.Version=v0.1.0' \
  -X 'github.com/wbreza/gonuget/cmd/gonuget/version.Commit=$(git rev-parse HEAD)' \
  -X 'github.com/wbreza/gonuget/cmd/gonuget/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
```

**Makefile Integration**:
```makefile
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -ldflags "\
  -X 'github.com/wbreza/gonuget/cmd/gonuget/version.Version=$(VERSION)' \
  -X 'github.com/wbreza/gonuget/cmd/gonuget/version.Commit=$(COMMIT)' \
  -X 'github.com/wbreza/gonuget/cmd/gonuget/version.Date=$(DATE)'"

build:
	go build $(LDFLAGS) -o gonuget ./cmd/gonuget
```

**goreleaser Integration**:
```yaml
builds:
  - ldflags:
      - -s -w
      - -X github.com/wbreza/gonuget/cmd/gonuget/version.Version={{.Version}}
      - -X github.com/wbreza/gonuget/cmd/gonuget/version.Commit={{.FullCommit}}
      - -X github.com/wbreza/gonuget/cmd/gonuget/version.Date={{.Date}}
```

### Rationale

**Dedicated Version Package**: Placing version variables in `cmd/gonuget/version` provides:
- Clean separation of concerns
- Importable by other packages (tests, subcommands)
- Single source of truth for version info
- Conventional Go project structure

**Package Location (`cmd/gonuget/version` vs `internal/version`)**: Using `cmd/gonuget/version` because:
- CLI-specific versioning (library may have different version)
- Conventional for CLI tools
- `internal/` would be overkill for simple version info
- However, if library also needs versioning, use `internal/version` and import from CLI

**Variable Names**:
- `Version`: Semantic version tag (v0.1.0, v1.2.3)
- `Commit`: Full git commit SHA (40 chars)
- `Date`: RFC3339 formatted timestamp (ISO 8601)

**Default Values**: Providing defaults ("dev", "none", "unknown") ensures:
- Meaningful output when building without ldflags
- Development builds are clearly marked
- No runtime panics from uninitialized variables

**Full Import Path Required**: The `-X` flag requires the complete import path (e.g., `github.com/wbreza/gonuget/cmd/gonuget/version.Version`), not just the package name. This is a common gotcha.

**Variables, Not Constants**: Go's linker can only set variables. Using `const` will silently fail to inject values.

**String Variables Only**: The `-X` flag only works with string variables. For version numbers, parse strings at runtime if needed.

### Best Practices

1. **Always provide default values** for dev builds
2. **Use RFC3339 for dates** (ISO 8601 compatible)
3. **Export variables** (capitalize) if other packages need access
4. **Add helper function** (like `Info()`) for formatted output
5. **Document in README** how to build with version info
6. **Test version output** in CI to ensure ldflags work
7. **Don't embed secrets** - ldflags values are visible in binaries (use `strings` command)

### Alternatives Considered

**Variables in main Package**: Rejected because:
- `main` is not importable
- Can't test version output easily
- Pollutes main package namespace
- Less flexible for subcommands

**Runtime Git Command Execution**: Rejected because:
- Requires git at runtime
- Fails in containers without .git directory
- Slower (process spawn)
- Unreliable in production

**Embedded File with //go:embed**: Rejected because:
- Still requires generation step
- Less standard than ldflags
- Complicates build process
- ldflags is Go ecosystem convention

**Version File Generation**: Rejected because:
- Requires pre-build script
- Creates git churn (generated file)
- ldflags is more elegant

### References

- [Using ldflags to Set Version Information (DigitalOcean)](https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications)
- [Set Version Information in Go (Neosync)](https://www.neosync.dev/blog/setting-go-version-information)
- [GoReleaser: Using main.version ldflag](https://goreleaser.com/cookbooks/using-main.version/)
- [Stack Overflow: How to set package variable using -ldflags](https://stackoverflow.com/questions/47509272/how-to-set-package-variable-using-ldflags-x-in-golang-build)

## 5. GitHub Actions Workflow Patterns for Go Projects

### Decision

Use a comprehensive GitHub Actions workflow with:
- Matrix builds for multiple OS (Linux, macOS, Windows)
- Go version: Latest stable
- Automatic Go module caching via `setup-go@v6`
- golangci-lint integration in separate job
- Test execution with artifact upload
- goreleaser for releases on tags

### Rationale

**Matrix Builds**: Testing on multiple operating systems ensures:
- Cross-platform compatibility validation
- OS-specific bug detection early
- User confidence in binary quality
- No surprises in production environments

**Latest Stable Go**: Using `go-version: stable` (instead of specific version) because:
- Automatically gets security patches
- Ensures compatibility with latest Go features
- Reduces maintenance (no version updates needed)
- Can add older versions to matrix if backward compat needed

**Automatic Caching**: `actions/setup-go@v6` enables caching by default:
- Caches `GOCACHE` and `GOMODCACHE`
- Uses `go.sum` as cache key
- Significantly speeds up builds (2-5x faster)
- Zero configuration required

**Separate Lint Job**: Running golangci-lint in parallel job because:
- Faster overall workflow (parallel execution)
- Clearer failure categorization (build vs lint)
- Independent caching strategies
- Can be required separately in branch protection

**Test Artifacts**: Uploading test results enables:
- Historical test analysis
- Debugging failed runs
- Test report generation
- Integration with third-party tools

### Workflow Structure

**CI Workflow** (`.github/workflows/ci.yml`):
```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

permissions:
  contents: read

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v5

      - uses: actions/setup-go@v6
        with:
          go-version: stable
          # cache: true is default, uses go.sum as key

      - name: Build
        run: go build -v ./...

      - name: Test
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: actions/upload-artifact@v4
        with:
          name: coverage-${{ matrix.os }}
          path: coverage.out

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5

      - uses: actions/setup-go@v6
        with:
          go-version: stable

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v8
        with:
          version: v2
          # Caching is automatic (GOCACHE, GOMODCACHE, golangci-lint cache)
```

**Release Workflow** (`.github/workflows/release.yml`):
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0  # Required for goreleaser changelog

      - uses: actions/setup-go@v6
        with:
          go-version: stable

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Key Configuration Details

**Permissions**: Minimal required permissions following principle of least privilege:
- CI: `contents: read` (default, explicit for clarity)
- Release: `contents: write` (create releases, upload assets)
- Add `packages: write` if pushing Docker images to GHCR

**Trigger Patterns**:
- CI: On pushes to main and all PRs
- Release: Only on version tags (`v*`)

**Race Detector**: `-race` flag in tests catches data race conditions:
- Critical for concurrent code
- Small performance overhead acceptable in CI
- Not available on all platforms (skipped automatically where unsupported)

**Coverage**: `-coverprofile=coverage.out` enables:
- Code coverage tracking
- Integration with Codecov/Coveralls
- Coverage trend analysis

**fetch-depth: 0**: Required for goreleaser because:
- Needs full git history
- Generates changelog from commits
- Calculates version from tags
- Default shallow clone is insufficient

### Caching Strategy

**setup-go Automatic Caching** (enabled by default in v4+):
- **Cache Key**: Hash of `go.sum` file
- **Cached Paths**:
  - `GOCACHE`: Compiled packages
  - `GOMODCACHE`: Downloaded modules (~/go/pkg/mod)
- **Invalidation**: Automatic when `go.sum` changes
- **Interval**: 7-day rolling window (configurable)

**golangci-lint Automatic Caching**:
- **Cache Key**: `runner_os`, `working_dir`, `interval_number`, `go.mod` hash
- **Cached Paths**: `~/.cache/golangci-lint`
- **Benefits**: 5-10x speedup on cache hit
- **Configuration**: No setup required, works out of box

**For Monorepos**: Use `cache-dependency-path` to specify multiple `go.sum` files:
```yaml
- uses: actions/setup-go@v6
  with:
    go-version: stable
    cache-dependency-path: |
      go.sum
      subproject/go.sum
```

### Advanced Patterns

**Testing Multiple Go Versions**:
```yaml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]
    go-version: ['1.21', '1.22', '1.23']
```

**fail-fast: false**: Keep testing other matrix combinations on failure:
```yaml
strategy:
  fail-fast: false
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]
```

**Conditional Steps**:
```yaml
- name: Run integration tests
  if: runner.os == 'Linux'
  run: make test-integration
```

### Alternatives Considered

**actions/cache Manually**: Rejected because:
- setup-go v4+ includes automatic caching
- Manual cache is redundant
- More configuration, same result
- Automatic cache is optimized by Go team

**Docker-based CI**: Rejected because:
- Slower (image pull overhead)
- Unnecessary for Go (compiles to static binary)
- Complicates Windows testing
- GitHub-hosted runners are sufficient

**Self-hosted Runners**: Rejected for initial release because:
- Adds infrastructure complexity
- GitHub-hosted runners are free for public repos
- Can migrate later if build times become issue

**CircleCI/Travis/Jenkins**: Rejected because:
- GitHub Actions tighter GitHub integration
- Better UI/UX for GitHub-centric workflow
- No external service dependency
- Free for open source

### References

- [GitHub Actions: Building and Testing Go](https://docs.github.com/en/actions/use-cases-and-examples/building-and-testing/building-and-testing-go)
- [actions/setup-go Repository](https://github.com/actions/setup-go)
- [golangci-lint-action Repository](https://github.com/golangci/golangci-lint-action)
- [GoReleaser GitHub Actions Integration](https://goreleaser.com/ci/actions/)
- [goreleaser-action Repository](https://github.com/goreleaser/goreleaser-action)
- [GitHub Actions Matrix Strategy Guide](https://codefresh.io/learn/github-actions/github-actions-matrix/)
- [2025 Blog: Go CI/CD Best Practices](https://medium.com/@tedious/go-linting-best-practices-for-ci-cd-with-github-actions-aa6d96e0c509)

---

## Summary and Recommendations

For gonuget v0.1.0 release, implement the following:

1. **Create `.goreleaser.yaml`** with multi-platform builds (linux, darwin, windows) for amd64 and arm64, SHA256 checksums, and ldflags for version injection.

2. **Initialize `CHANGELOG.md`** following Keep a Changelog v1.1.0 format with six standard categories and ISO 8601 dates.

3. **Create `cmd/gonuget/version/version.go`** with exported Version, Commit, and Date variables for ldflags injection.

4. **Implement bash script** (optional) for parsing conventional commits during changelog generation, using native bash regex.

5. **Add GitHub Actions workflows**:
   - `.github/workflows/ci.yml`: Matrix builds (Linux/macOS/Windows), tests, linting
   - `.github/workflows/release.yml`: goreleaser on version tags

6. **Update Makefile** with VERSION, COMMIT, DATE variables and ldflags integration.

7. **Document release process** in CONTRIBUTING.md:
   - How to cut a release (git tag)
   - Changelog update procedure
   - Version numbering (semantic versioning)

This approach provides a production-ready release system following 2025 Go ecosystem best practices, with minimal manual overhead and maximum automation.
