# Version Package Contract

**Package**: `github.com/willibrandon/gonuget/cmd/gonuget/version`
**Purpose**: Provide build-time version information for gonuget CLI

## Public API

### Package Variables

```go
package version

var (
    // Version is the semantic version of gonuget (e.g., "v0.1.0")
    // Injected at build time via: -ldflags "-X github.com/willibrandon/gonuget/cmd/gonuget/version.Version=v0.1.0"
    // Default: "dev" (for development builds)
    Version = "dev"

    // Commit is the git commit SHA (short or full) this build was created from
    // Injected at build time via: -ldflags "-X github.com/willibrandon/gonuget/cmd/gonuget/version.Commit=$(git rev-parse --short HEAD)"
    // Default: "none" (for builds without git)
    Commit = "none"

    // Date is the build timestamp in ISO 8601 format
    // Injected at build time via: -ldflags "-X github.com/willibrandon/gonuget/cmd/gonuget/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    // Default: "unknown" (for builds without timestamp)
    Date = "unknown"
)
```

### Functions

```go
// Info returns a formatted string with all version information
// Format: "gonuget version v0.1.0 (commit: a1b2c3d, built: 2025-11-04T12:00:00Z)"
// Used by: gonuget version command
func Info() string
```

## Usage Examples

### Development Build (no ldflags)
```bash
$ go build -o gonuget ./cmd/gonuget
$ ./gonuget version
gonuget version dev (commit: none, built: unknown)
```

### Production Build (with ldflags via Makefile)
```bash
$ make build
$ ./gonuget version
gonuget version v0.1.0 (commit: a1b2c3d, built: 2025-11-04T12:00:00Z)
```

### Production Build (with ldflags via goreleaser)
```yaml
# .goreleaser.yml
builds:
  - ldflags:
      - -s -w
      - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Version={{.Version}}
      - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Commit={{.Commit}}
      - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Date={{.Date}}
```

## Integration Points

### 1. Makefile Integration
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

### 2. goreleaser Integration
```yaml
# .goreleaser.yml
builds:
  - id: gonuget
    main: ./cmd/gonuget
    binary: gonuget
    ldflags:
      - -s -w
      - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Version={{.Version}}
      - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Commit={{.ShortCommit}}
      - -X github.com/willibrandon/gonuget/cmd/gonuget/version.Date={{.Date}}
```

### 3. CLI Command Integration
```go
// cmd/gonuget/commands/version.go
package commands

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/willibrandon/gonuget/cmd/gonuget/version"
)

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Show gonuget version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println(version.Info())
    },
}
```

## Validation Rules

1. **Version Format**:
   - MUST match semver pattern: `^v\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?$`
   - For tagged releases: exact tag (e.g., "v0.1.0")
   - For dev builds: "dev"

2. **Commit Format**:
   - MUST be valid git SHA (7-40 hexadecimal characters)
   - OR "none" for builds without git

3. **Date Format**:
   - MUST be ISO 8601: `YYYY-MM-DDTHH:MM:SSZ`
   - OR "unknown" for builds without timestamp

## Testing

### Unit Tests
```go
// cmd/gonuget/version/version_test.go
func TestInfo(t *testing.T) {
    // Test default values
    got := Info()
    want := "gonuget version dev (commit: none, built: unknown)"
    if got != want {
        t.Errorf("Info() = %q, want %q", got, want)
    }
}

func TestInfoWithInjectedValues(t *testing.T) {
    // Simulate build-time injection
    oldVersion, oldCommit, oldDate := Version, Commit, Date
    defer func() {
        Version, Commit, Date = oldVersion, oldCommit, oldDate
    }()

    Version = "v0.1.0"
    Commit = "a1b2c3d"
    Date = "2025-11-04T12:00:00Z"

    got := Info()
    want := "gonuget version v0.1.0 (commit: a1b2c3d, built: 2025-11-04T12:00:00Z)"
    if got != want {
        t.Errorf("Info() = %q, want %q", got, want)
    }
}
```

### Integration Tests
```bash
# Test version command with injected values
go build -ldflags "-X github.com/willibrandon/gonuget/cmd/gonuget/version.Version=v0.1.0-test" -o gonuget-test ./cmd/gonuget
output=$(./gonuget-test version)
echo "$output" | grep -q "v0.1.0-test" || exit 1
rm gonuget-test
```

## Backward Compatibility

- Package is new in v0.1.0, no backward compatibility concerns
- If version package is missing (old builds), commands should gracefully handle missing version info
- Future versions can add additional variables (e.g., `GoVersion`, `Platform`) without breaking changes

## Security Considerations

- Version information is read-only at runtime (no mutation)
- No sensitive information embedded (public git commit SHAs only)
- -s -w ldflags strip symbol tables for smaller binaries
