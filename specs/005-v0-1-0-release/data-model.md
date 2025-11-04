# Data Model: v0.1.0 Release Preparation

**Feature**: 005-v0-1-0-release
**Created**: 2025-11-04

## Overview

This feature involves release infrastructure with minimal runtime data structures. Most entities are file-based artifacts or build-time constants.

## Entities

### 1. Version Information

**Purpose**: Stores version metadata embedded in compiled binary at build time

**Fields**:
- `Version` (string): Semantic version number (e.g., "v0.1.0")
- `Commit` (string): Git commit SHA (short or full)
- `Date` (string): Build timestamp (ISO 8601 format)
- `GoVersion` (string, optional): Go version used for build

**Relationships**: None (standalone package-level variables)

**Lifecycle**: Set once at build time via -ldflags, read-only at runtime

**Validation Rules**:
- Version MUST match semver format (vMAJOR.MINOR.PATCH)
- Commit MUST be valid git SHA (7-40 characters, hexadecimal)
- Date MUST be ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)
- Default values for dev builds: Version="dev", Commit="none", Date="unknown"

**Location**: `cmd/gonuget/version/version.go`

**Example**:
```go
package version

var (
    Version   = "dev"      // Injected via -ldflags -X
    Commit    = "none"     // Injected via -ldflags -X
    Date      = "unknown"  // Injected via -ldflags -X
    GoVersion = ""         // Optional, injected via -ldflags -X
)
```

---

### 2. Changelog Entry

**Purpose**: Represents a single commit entry in generated CHANGELOG.md

**Fields**:
- `Type` (string): Commit type (feat, fix, docs, test, perf, refactor, chore, uncategorized)
- `Scope` (string, optional): Commit scope from conventional format
- `Description` (string): Commit message description
- `SHA` (string): Commit SHA (short, 7 characters)
- `Author` (string): Commit author name
- `Date` (string): Commit date (ISO 8601)
- `Breaking` (boolean): Whether commit has breaking change marker (!)

**Relationships**: Grouped by Type into CHANGELOG sections

**Lifecycle**: Parsed from git log, formatted into markdown, written to CHANGELOG.md

**Validation Rules**:
- Type MUST be one of: feat, fix, docs, test, perf, refactor, chore, uncategorized
- Description MUST be non-empty
- SHA MUST be 7 hexadecimal characters
- Date MUST be valid ISO 8601 timestamp

**State Transitions**: Raw commit → Parsed entry → Categorized → Markdown formatted

**Example** (internal shell script representation):
```bash
# Parsed from: feat(cli): add version command
TYPE="feat"
SCOPE="cli"
DESCRIPTION="add version command"
SHA="a1b2c3d"
AUTHOR="Brandon Williams"
DATE="2025-11-04"
BREAKING="false"
```

---

### 3. Release Artifact

**Purpose**: Represents a single compiled binary artifact for a target platform

**Fields**:
- `Platform` (string): Target OS (linux, darwin, windows)
- `Architecture` (string): Target architecture (amd64, arm64)
- `BinaryPath` (string): Path to compiled binary file
- `Checksum` (string): SHA256 checksum (hex-encoded, 64 characters)
- `Size` (int64): File size in bytes

**Relationships**: Multiple artifacts per release (5 total for v0.1.0)

**Lifecycle**: Built by goreleaser → Checksummed → Uploaded to GitHub release

**Validation Rules**:
- Platform MUST be one of: linux, darwin, windows
- Architecture MUST be one of: amd64, arm64
- BinaryPath MUST exist and be executable
- Checksum MUST be 64-character hexadecimal string
- All 5 target platforms MUST succeed (any failure fails entire release)

**State Transitions**: Source code → Compiled binary → Checksummed → Published

**Example** (goreleaser internal representation):
```yaml
# Platform: linux, Architecture: amd64
artifact:
  name: gonuget-linux-amd64
  path: dist/gonuget-linux-amd64
  goos: linux
  goarch: amd64
  checksum: a1b2c3d4e5f6...  # 64 chars
  size: 12345678
```

---

### 4. CI Workflow Run

**Purpose**: Represents execution of CI workflow on GitHub Actions

**Fields**:
- `WorkflowName` (string): Workflow identifier (ci, release)
- `TriggerEvent` (string): Event type (push, pull_request, tag)
- `Platform` (string): Runner OS (ubuntu, macos, windows)
- `GoVersion` (string): Go version used (e.g., "1.23")
- `Status` (string): Workflow status (success, failure, cancelled)
- `Duration` (int): Execution time in seconds
- `Jobs` (array): Individual job results (lint, test, build)

**Relationships**: One workflow run contains multiple jobs (lint, test, build)

**Lifecycle**: Triggered → Running → Completed (success/failure)

**Validation Rules**:
- Duration MUST be <900 seconds (15 minutes per success criteria)
- Status MUST be "success" for release workflow to proceed
- All platform jobs (ubuntu, macos, windows) MUST succeed

**State Transitions**: Idle → Queued → Running → Completed

**Example** (GitHub Actions internal):
```yaml
run:
  workflow: ci
  trigger: push
  platform: ubuntu-latest
  go_version: "1.23"
  status: success
  duration: 420  # 7 minutes
  jobs:
    - name: lint
      status: success
      duration: 60
    - name: test
      status: success
      duration: 300
    - name: build
      status: success
      duration: 60
```

---

### 5. Release Version

**Purpose**: Represents a complete v0.1.0 release with all metadata

**Fields**:
- `Version` (string): Semantic version tag (v0.1.0)
- `GitTag` (string): Git tag name (v0.1.0)
- `ReleaseDate` (string): Release publication date (ISO 8601)
- `CommitSHA` (string): Commit SHA this release points to
- `Changelog` (string): Generated changelog content (markdown)
- `Artifacts` (array): List of release artifacts (5 binaries)
- `Checksums` (string): Path to checksums file

**Relationships**:
- Has many Artifacts (5 platform builds)
- References one Commit (git SHA)
- Contains one Changelog

**Lifecycle**: Planned → Built → Tested → Tagged → Published

**Validation Rules**:
- Version MUST match regex: `^v0\.\d+\.\d+$` (pre-1.0 release)
- GitTag MUST match Version exactly
- All 5 Artifacts MUST be present
- Checksums file MUST contain all 5 artifacts
- All interop tests MUST pass before tagging

**State Transitions**: Development → Pre-release validation → Tagged → Released

**Example** (GitHub release API response):
```json
{
  "tag_name": "v0.1.0",
  "name": "v0.1.0",
  "published_at": "2025-11-04T12:00:00Z",
  "target_commitish": "a1b2c3d4e5f6",
  "body": "## Features\n- Initial release\n...",
  "assets": [
    {
      "name": "gonuget-linux-amd64",
      "size": 12345678,
      "browser_download_url": "https://..."
    }
  ]
}
```

---

## Entity Relationships

```
Release Version (v0.1.0)
├── Git Tag (v0.1.0)
├── Commit SHA (a1b2c3d)
├── Changelog
│   └── Changelog Entries (grouped by type)
│       ├── feat entries
│       ├── fix entries
│       ├── docs entries
│       ├── test entries
│       ├── perf entries
│       ├── refactor entries
│       ├── chore entries
│       └── uncategorized entries
└── Release Artifacts (5 total)
    ├── linux-amd64 (binary + checksum)
    ├── linux-arm64 (binary + checksum)
    ├── darwin-amd64 (binary + checksum)
    ├── darwin-arm64 (binary + checksum)
    └── windows-amd64 (binary + checksum)

CI Workflow Run
├── Lint Job (ubuntu-latest)
├── Test Jobs (matrix: ubuntu, macos, windows)
└── Build Job (ubuntu-latest)
```

---

## File-Based Artifacts

These are file system artifacts, not runtime data structures:

### CHANGELOG.md
**Format**: Markdown following Keep a Changelog specification
**Sections**: Unreleased, v0.1.0 (2025-11-04)
**Categories**: Added, Changed, Fixed, Deprecated, Removed, Security, Uncategorized
**Generated by**: `scripts/generate-changelog.sh`

### .goreleaser.yml
**Format**: YAML configuration file
**Sections**: builds, archives, checksum, release, changelog
**Purpose**: Defines goreleaser build configuration

### .github/workflows/ci.yml
**Format**: GitHub Actions workflow YAML
**Triggers**: push, pull_request
**Jobs**: lint, test (matrix: 3 OS)

### .github/workflows/release.yml
**Format**: GitHub Actions workflow YAML
**Triggers**: tag push (v*.*.*)
**Jobs**: release (goreleaser build + publish)

---

## Constants

```go
// cmd/gonuget/version/version.go
const (
    // SupportedPlatforms lists all release target platforms
    SupportedPlatforms = []string{
        "linux/amd64",
        "linux/arm64",
        "darwin/amd64",
        "darwin/arm64",
        "windows/amd64",
    }
)

// scripts/generate-changelog.sh
COMMIT_TYPES=(
    "feat:Features"
    "fix:Bug Fixes"
    "perf:Performance"
    "test:Tests"
    "docs:Documentation"
    "refactor:Refactoring"
    "chore:Chores"
)
```

---

## Notes

1. **No Database**: All entities are either file-based artifacts or build-time constants
2. **Stateless**: Version information is read-only after build, no runtime mutation
3. **Validation**: Most validation happens at build/release time (CI workflows, goreleaser)
4. **Idempotency**: All workflows can be safely re-run without side effects
