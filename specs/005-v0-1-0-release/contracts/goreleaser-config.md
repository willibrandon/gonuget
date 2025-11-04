# goreleaser Configuration Contract

**File**: `.goreleaser.yml`
**Purpose**: Define multi-platform build and release automation configuration

## Configuration Structure

### Complete Configuration
```yaml
# .goreleaser.yml
project_name: gonuget

before:
  hooks:
    - go mod tidy
    - go mod verify

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
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w
      - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Version={{.Version}}
      - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Commit={{.ShortCommit}}
      - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Date={{.Date}}

archives:
  - id: default
    format: tar.gz
    name_template: "{{ .ProjectName }}-{{ .Version }}-{{ .Os }}-{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
    files:
      - LICENSE
      - README.md
      - CHANGELOG.md

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

release:
  github:
    owner: willibrandon
    name: gonuget
  draft: false
  prerelease: auto
  name_template: "{{.Version}}"
  header: |
    ## gonuget {{.Version}}

    Download the binary for your platform below.
  footer: |
    **Full Changelog**: https://github.com/willibrandon/gonuget/blob/main/CHANGELOG.md

changelog:
  disable: true  # Use manually generated CHANGELOG.md instead

snapshot:
  name_template: "{{ .Version }}-next"

dist: dist
```

## Build Matrix

### Target Platforms (5 total)
```
linux/amd64    → gonuget-v0.1.0-linux-amd64.tar.gz
linux/arm64    → gonuget-v0.1.0-linux-arm64.tar.gz
darwin/amd64   → gonuget-v0.1.0-darwin-amd64.tar.gz
darwin/arm64   → gonuget-v0.1.0-darwin-arm64.tar.gz
windows/amd64  → gonuget-v0.1.0-windows-amd64.zip
```

**Note**: windows/arm64 excluded (limited user base)

### Build Flags

#### Environment Variables
```yaml
env:
  - CGO_ENABLED=0  # Static binaries, no C dependencies
```

#### Linker Flags
```yaml
ldflags:
  - -s -w  # Strip debug symbols and DWARF tables (smaller binaries)
  - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Version={{.Version}}
  - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Commit={{.ShortCommit}}
  - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Date={{.Date}}
```

**Template Variables**:
- `{{.Version}}`: Git tag (e.g., "v0.1.0")
- `{{.ShortCommit}}`: Short SHA (7 chars)
- `{{.Date}}`: Build timestamp (ISO 8601)

## Archive Configuration

### Archive Naming
```
Template: {{ .ProjectName }}-{{ .Version }}-{{ .Os }}-{{ .Arch }}
Example:  gonuget-v0.1.0-linux-amd64.tar.gz
```

### Included Files
- Binary: `gonuget` (or `gonuget.exe` on Windows)
- `LICENSE`: Project license file
- `README.md`: Installation and usage instructions
- `CHANGELOG.md`: Generated changelog

### Format
- **Linux/macOS**: `tar.gz` (standard for Unix systems)
- **Windows**: `zip` (native Windows format)

## Checksum Generation

### Checksum File
```
File:      checksums.txt
Algorithm: SHA256
Format:    <64-char-hex> <filename>
```

### Example checksums.txt
```
a1b2c3d4e5f6... gonuget-v0.1.0-linux-amd64.tar.gz
b2c3d4e5f6g7... gonuget-v0.1.0-linux-arm64.tar.gz
c3d4e5f6g7h8... gonuget-v0.1.0-darwin-amd64.tar.gz
d4e5f6g7h8i9... gonuget-v0.1.0-darwin-arm64.tar.gz
e5f6g7h8i9j0... gonuget-v0.1.0-windows-amd64.zip
```

### Verification
```bash
# Verify checksum after download
sha256sum -c checksums.txt --ignore-missing
```

## GitHub Release Integration

### Release Metadata
```yaml
release:
  github:
    owner: willibrandon
    name: gonuget
  draft: false           # Publish immediately (not draft)
  prerelease: auto       # Auto-detect pre-release (v0.x.x, alpha, beta, rc)
  name_template: "{{.Version}}"  # Release name: "v0.1.0"
```

### Release Notes
```yaml
header: |
  ## gonuget {{.Version}}

  Download the binary for your platform below.

footer: |
  **Full Changelog**: https://github.com/willibrandon/gonuget/blob/main/CHANGELOG.md

changelog:
  disable: true  # Use manually generated CHANGELOG.md instead
```

**Rationale**: Manual CHANGELOG.md provides better control over formatting and categorization than goreleaser's auto-generated changelog.

## Workflow Integration

### Local Testing
```bash
# Install goreleaser
go install github.com/goreleaser/goreleaser@latest

# Test configuration (snapshot build)
goreleaser release --snapshot --clean

# Verify builds
ls -lh dist/

# Test binary
dist/gonuget_linux_amd64_v1/gonuget version
```

### GitHub Actions Integration
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
          fetch-depth: 0  # Full history for changelog

      - uses: actions/setup-go@v5
        with:
          go-version: stable

      - name: Run goreleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Validation

### Pre-conditions
1. Git tag must match semver format: `v*.*.*`
2. Working directory must be clean (no uncommitted changes)
3. All tests must pass before tagging
4. CHANGELOG.md must exist and be up-to-date

### Build Validation
```bash
# goreleaser checks
goreleaser check  # Validate .goreleaser.yml syntax

# Build all platforms
goreleaser build --snapshot --clean

# Verify all 5 binaries exist
test -f dist/gonuget_linux_amd64_v1/gonuget
test -f dist/gonuget_linux_arm64/gonuget
test -f dist/gonuget_darwin_amd64_v1/gonuget
test -f dist/gonuget_darwin_arm64/gonuget
test -f dist/gonuget_windows_amd64_v1/gonuget.exe

# Verify version injection
dist/gonuget_linux_amd64_v1/gonuget version | grep -q "v0.1.0"
```

### Post-release Validation
```bash
# Download and verify checksums
curl -sL https://github.com/willibrandon/gonuget/releases/download/v0.1.0/checksums.txt -o checksums.txt
curl -sL https://github.com/willibrandon/gonuget/releases/download/v0.1.0/gonuget-v0.1.0-linux-amd64.tar.gz -o gonuget-v0.1.0-linux-amd64.tar.gz
sha256sum -c checksums.txt --ignore-missing

# Extract and test
tar -xzf gonuget-v0.1.0-linux-amd64.tar.gz
./gonuget version
```

## Error Handling

### Build Failures
```yaml
# goreleaser fails if ANY platform build fails
# Per spec FR-017: "any build failure fails the entire release"
# This is default goreleaser behavior - no configuration needed
```

**Behavior**:
1. goreleaser attempts all platform builds
2. If ANY build fails, entire release fails
3. No partial artifacts published
4. GitHub release is NOT created

### Tag Mismatch
```bash
# Error if current commit is not tagged
goreleaser release
# Error: "git tag -l --points-at HEAD returned no tags"

# Error if tag doesn't match semver
git tag invalid-tag
goreleaser release
# Error: "tag invalid-tag doesn't match semver"
```

### Missing Files
```bash
# Error if required files missing
# Example: LICENSE file missing
goreleaser release
# Error: "LICENSE: file not found"
```

## Maintenance

### Updating goreleaser
```bash
# Check current version
goreleaser --version

# Update to latest
go install github.com/goreleaser/goreleaser@latest

# Verify config compatibility
goreleaser check
```

### Adding New Platforms
```yaml
# Example: Add linux/arm for Raspberry Pi
builds:
  - goos:
      - linux
    goarch:
      - amd64
      - arm64
      - arm  # New
    goarm:
      - 7    # ARMv7 (Raspberry Pi 2+)
```

### Binary Size Optimization
```yaml
# Current flags
ldflags:
  - -s -w  # Strip symbols (~30% size reduction)

# Optional: Additional compression
archives:
  - format: tar.gz
    compression: gzip
    compression_level: 9  # Maximum compression
```

## Dependencies

- goreleaser v2.0+ (latest stable)
- Go 1.23+ (build toolchain)
- git (for tag detection and commit info)
- GitHub token (for release publishing)

## Security

- `-s -w` flags strip debug symbols (smaller binaries, no source paths leaked)
- `CGO_ENABLED=0` produces static binaries (no dynamic library dependencies)
- Checksums prevent tampering (users can verify integrity)
- GitHub token requires `contents: write` permission only (minimal scope)
