# Milestone 1: Foundation

**Goal:** Core abstractions and version handling
**Chunks:** 12
**Est. Total Time:** 24 hours
**Status:** 0/12 complete (0%)

---

## Overview

This milestone establishes the foundation for gonuget:
- Version parsing and comparison (NuGet SemVer 2.0 + legacy)
- Framework parsing and compatibility
- Core package identity types

These components are used throughout the library and must be bulletproof.

---

## Chunk M1.1: Initialize Go Module

**Time:** 15 min
**Dependencies:** None
**Status:** [ ] Not Started

### What You'll Build

Initialize the gonuget Go module with proper project structure and configuration files.

### Step-by-Step Instructions

**1. Create project directory and initialize module:**

```bash
cd /Users/brandon/src/gonuget
go mod init github.com/yourusername/gonuget
```

**2. Create package directories:**

```bash
mkdir -p version
mkdir -p frameworks
mkdir -p core
mkdir -p protocol/v3
mkdir -p protocol/v2
mkdir -p packaging
mkdir -p packaging/signing
mkdir -p cache
mkdir -p http
mkdir -p auth
mkdir -p circuitbreaker
mkdir -p ratelimit
mkdir -p observability
mkdir -p resolver
mkdir -p client
```

**3. Create .gitignore:**

```gitignore
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test

# Output
*.out
bin/
dist/

# Coverage
*.coverprofile
coverage.html

# IDEs
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Temporary
tmp/
temp/
*.tmp

# Vendor
vendor/
```

**4. Create initial README.md:**

```markdown
# gonuget

A complete, production-grade NuGet client library for Go.

## Status

🚧 **Under Development** - Pre-release

## Features (Planned)

- ✅ NuGet SemVer 2.0 version parsing and comparison
- ✅ Framework compatibility checking
- ✅ NuGet v3 protocol support
- ✅ NuGet v2 protocol support
- ✅ Package reading and validation
- ✅ Dependency resolution
- ✅ Package signing verification

## Installation

```bash
go get github.com/yourusername/gonuget
```

## Quick Start

```go
// Coming soon
```

## License

MIT License - See LICENSE file
```

**5. Create LICENSE file (MIT):**

```
MIT License

Copyright (c) 2025 [Your Name]

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### Verification Steps

Run these commands to verify:

```bash
# Should show go.mod created
ls -la go.mod

# Should list all package directories
ls -d */ | grep -E "(version|frameworks|core|protocol|packaging|cache|http|auth|client|resolver)"

# Should show .gitignore
cat .gitignore

# Should run without errors
go mod tidy
```

### Testing

No tests for this chunk - just project initialization.

### Commit

```bash
git init
git add .
git commit -m "chore: initialize go module with project structure

- Create gonuget Go module
- Set up package directory structure
- Add .gitignore, README, and LICENSE

Chunk: M1.1
Status: ✓ Complete"
```

### Next Chunk
→ **M1.2: Version Package - Basic Types**

---

## Chunk M1.2: Version Package - Basic Types

**Time:** 30 min
**Dependencies:** M1.1
**Status:** [ ] Not Started

### What You'll Build

Create the `NuGetVersion` type structure to represent NuGet package versions.

### Step-by-Step Instructions

**1. Create `version/version.go`:**

```go
// Package version provides NuGet version parsing and comparison.
//
// It supports both NuGet SemVer 2.0 format and legacy 4-part versions.
//
// Example:
//
//	v, err := version.Parse("1.2.3-beta.1")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(v.Major, v.Minor, v.Patch) // 1 2 3
package version

import "fmt"

// NuGetVersion represents a NuGet package version.
//
// It supports both SemVer 2.0 format (Major.Minor.Patch[-Prerelease][+Metadata])
// and legacy 4-part versions (Major.Minor.Build.Revision).
type NuGetVersion struct {
	// Major version number
	Major int

	// Minor version number
	Minor int

	// Patch version number (or Build for legacy versions)
	Patch int

	// Revision is only used for legacy 4-part versions (Major.Minor.Build.Revision)
	Revision int

	// IsLegacyVersion indicates this is a 4-part version, not SemVer 2.0
	IsLegacyVersion bool

	// ReleaseLabels contains prerelease labels (e.g., ["beta", "1"] for "1.0.0-beta.1")
	ReleaseLabels []string

	// Metadata is the build metadata (e.g., "20241019" for "1.0.0+20241019")
	// Metadata is ignored in version comparison per SemVer 2.0 spec
	Metadata string

	// originalString preserves the original version string
	originalString string
}

// String returns the string representation of the version.
func (v *NuGetVersion) String() string {
	if v.originalString != "" {
		return v.originalString
	}
	return v.format()
}

// format creates a formatted version string.
func (v *NuGetVersion) format() string {
	var s string

	if v.IsLegacyVersion {
		s = fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Patch, v.Revision)
	} else {
		s = fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	}

	if len(v.ReleaseLabels) > 0 {
		s += "-"
		for i, label := range v.ReleaseLabels {
			if i > 0 {
				s += "."
			}
			s += label
		}
	}

	if v.Metadata != "" {
		s += "+" + v.Metadata
	}

	return s
}
```

**2. Create `version/version_test.go`:**

```go
package version

import "testing"

func TestNuGetVersion_String(t *testing.T) {
	tests := []struct {
		name     string
		version  *NuGetVersion
		expected string
	}{
		{
			name: "simple version",
			version: &NuGetVersion{
				Major: 1,
				Minor: 0,
				Patch: 0,
			},
			expected: "1.0.0",
		},
		{
			name: "version with prerelease",
			version: &NuGetVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				ReleaseLabels: []string{"beta", "1"},
			},
			expected: "1.2.3-beta.1",
		},
		{
			name: "version with metadata",
			version: &NuGetVersion{
				Major:    1,
				Minor:    0,
				Patch:    0,
				Metadata: "20241019",
			},
			expected: "1.0.0+20241019",
		},
		{
			name: "legacy 4-part version",
			version: &NuGetVersion{
				Major:           2,
				Minor:           5,
				Patch:           3,
				Revision:        1,
				IsLegacyVersion: true,
			},
			expected: "2.5.3.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}
```

### Verification Steps

```bash
# Build the package
go build ./version

# Run tests
go test ./version -v

# Check test output - should show PASS
```

Expected output:
```
=== RUN   TestNuGetVersion_String
=== RUN   TestNuGetVersion_String/simple_version
=== RUN   TestNuGetVersion_String/version_with_prerelease
=== RUN   TestNuGetVersion_String/version_with_metadata
=== RUN   TestNuGetVersion_String/legacy_4-part_version
--- PASS: TestNuGetVersion_String (0.00s)
PASS
```

### Testing

All tests included in verification steps above.

### Commit

```bash
git add version/
git commit -m "feat: add NuGetVersion type and basic structure

- Define NuGetVersion struct with SemVer 2.0 fields
- Add legacy 4-part version support
- Implement String() method with formatting
- Add initial tests for version formatting

Chunk: M1.2
Status: ✓ Complete"
```

### Next Chunk
→ **M1.3: Version Package - Parsing (SemVer 2.0)**

---

## Chunk M1.3: Version Package - Parsing (SemVer 2.0)

**Time:** 2 hours
**Dependencies:** M1.2
**Status:** [ ] Not Started

### What You'll Build

Implement parsing for NuGet SemVer 2.0 version strings.

### Step-by-Step Instructions

**1. Add to `version/version.go`:**

```go
import (
	"fmt"
	"strconv"
	"strings"
)

// Parse parses a version string into a NuGetVersion.
//
// Supported formats:
//   - SemVer 2.0: Major.Minor.Patch[-Prerelease][+Metadata]
//   - Legacy: Major.Minor.Build.Revision
//
// Returns an error if the version string is invalid.
//
// Example:
//
//	v, err := Parse("1.0.0-beta.1+build.123")
//	if err != nil {
//	    return err
//	}
func Parse(s string) (*NuGetVersion, error) {
	if s == "" {
		return nil, fmt.Errorf("version string cannot be empty")
	}

	v := &NuGetVersion{
		originalString: s,
	}

	// Split on '+' to extract metadata
	parts := strings.SplitN(s, "+", 2)
	versionPart := parts[0]
	if len(parts) == 2 {
		v.Metadata = parts[1]
	}

	// Split on '-' to extract prerelease labels
	parts = strings.SplitN(versionPart, "-", 2)
	numberPart := parts[0]
	if len(parts) == 2 {
		v.ReleaseLabels = parseReleaseLabels(parts[1])
	}

	// Parse the numeric version parts
	numbers := strings.Split(numberPart, ".")
	if len(numbers) < 2 || len(numbers) > 4 {
		return nil, fmt.Errorf("invalid version format: %q", s)
	}

	// Parse major
	major, err := strconv.Atoi(numbers[0])
	if err != nil || major < 0 {
		return nil, fmt.Errorf("invalid major version: %q", numbers[0])
	}
	v.Major = major

	// Parse minor
	minor, err := strconv.Atoi(numbers[1])
	if err != nil || minor < 0 {
		return nil, fmt.Errorf("invalid minor version: %q", numbers[1])
	}
	v.Minor = minor

	// Parse patch (or build)
	if len(numbers) >= 3 {
		patch, err := strconv.Atoi(numbers[2])
		if err != nil || patch < 0 {
			return nil, fmt.Errorf("invalid patch version: %q", numbers[2])
		}
		v.Patch = patch
	}

	// If 4 parts, this is a legacy version
	if len(numbers) == 4 {
		revision, err := strconv.Atoi(numbers[3])
		if err != nil || revision < 0 {
			return nil, fmt.Errorf("invalid revision: %q", numbers[3])
		}
		v.Revision = revision
		v.IsLegacyVersion = true
	}

	return v, nil
}

// MustParse parses a version string and panics on error.
// Use this only when you know the version string is valid.
func MustParse(s string) *NuGetVersion {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

// parseReleaseLabels splits a prerelease string into labels.
func parseReleaseLabels(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ".")
}
```

**2. Create `version/parse_test.go`:**

```go
package version

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *NuGetVersion
		wantErr bool
	}{
		{
			name:  "simple version",
			input: "1.0.0",
			want: &NuGetVersion{
				Major:          1,
				Minor:          0,
				Patch:          0,
				originalString: "1.0.0",
			},
		},
		{
			name:  "version with prerelease",
			input: "1.2.3-beta",
			want: &NuGetVersion{
				Major:          1,
				Minor:          2,
				Patch:          3,
				ReleaseLabels:  []string{"beta"},
				originalString: "1.2.3-beta",
			},
		},
		{
			name:  "version with multiple prerelease labels",
			input: "1.0.0-alpha.1",
			want: &NuGetVersion{
				Major:          1,
				Minor:          0,
				Patch:          0,
				ReleaseLabels:  []string{"alpha", "1"},
				originalString: "1.0.0-alpha.1",
			},
		},
		{
			name:  "version with metadata",
			input: "1.0.0+20241019",
			want: &NuGetVersion{
				Major:          1,
				Minor:          0,
				Patch:          0,
				Metadata:       "20241019",
				originalString: "1.0.0+20241019",
			},
		},
		{
			name:  "version with prerelease and metadata",
			input: "1.0.0-rc.1+build.123",
			want: &NuGetVersion{
				Major:          1,
				Minor:          0,
				Patch:          0,
				ReleaseLabels:  []string{"rc", "1"},
				Metadata:       "build.123",
				originalString: "1.0.0-rc.1+build.123",
			},
		},
		{
			name:  "major.minor only",
			input: "1.0",
			want: &NuGetVersion{
				Major:          1,
				Minor:          0,
				Patch:          0,
				originalString: "1.0",
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format - too many parts",
			input:   "1.2.3.4.5",
			wantErr: true,
		},
		{
			name:    "invalid format - single number",
			input:   "1",
			wantErr: true,
		},
		{
			name:    "invalid major",
			input:   "a.0.0",
			wantErr: true,
		},
		{
			name:    "negative version",
			input:   "1.-1.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Compare fields
			if got.Major != tt.want.Major {
				t.Errorf("Major = %v, want %v", got.Major, tt.want.Major)
			}
			if got.Minor != tt.want.Minor {
				t.Errorf("Minor = %v, want %v", got.Minor, tt.want.Minor)
			}
			if got.Patch != tt.want.Patch {
				t.Errorf("Patch = %v, want %v", got.Patch, tt.want.Patch)
			}
			if got.Metadata != tt.want.Metadata {
				t.Errorf("Metadata = %v, want %v", got.Metadata, tt.want.Metadata)
			}
			if len(got.ReleaseLabels) != len(tt.want.ReleaseLabels) {
				t.Errorf("ReleaseLabels length = %v, want %v", len(got.ReleaseLabels), len(tt.want.ReleaseLabels))
			}
			for i := range got.ReleaseLabels {
				if got.ReleaseLabels[i] != tt.want.ReleaseLabels[i] {
					t.Errorf("ReleaseLabels[%d] = %v, want %v", i, got.ReleaseLabels[i], tt.want.ReleaseLabels[i])
				}
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	// Should not panic
	v := MustParse("1.0.0")
	if v.Major != 1 {
		t.Errorf("MustParse() Major = %v, want 1", v.Major)
	}

	// Should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse() should panic on invalid version")
		}
	}()
	MustParse("invalid")
}
```

### Verification Steps

```bash
# Run tests
go test ./version -v -run TestParse

# Check coverage
go test ./version -cover
```

Expected: All tests pass, coverage >80%

### Testing

All tests included above. Run with:
```bash
go test ./version -v
```

### Commit

```bash
git add version/
git commit -m "feat: implement NuGet SemVer 2.0 parsing

- Add Parse() function for version strings
- Support Major.Minor.Patch format
- Support prerelease labels (e.g., -beta.1)
- Support build metadata (e.g., +build.123)
- Add MustParse() for known-valid versions
- Add comprehensive parsing tests

Chunk: M1.3
Status: ✓ Complete"
```

### Next Chunk
→ **M1.4: Version Package - Parsing (Legacy 4-part)**

---

## Chunk M1.4: Version Package - Parsing (Legacy 4-part)

**Time:** 1 hour
**Dependencies:** M1.3
**Status:** [ ] Not Started

### What You'll Build

Add support for parsing legacy 4-part version numbers (Major.Minor.Build.Revision).

### Step-by-Step Instructions

**The parsing logic was already added in M1.3 in the `Parse()` function.**

Now we need to add comprehensive tests for the legacy format.

**1. Add to `version/parse_test.go`:**

```go
func TestParse_Legacy(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *NuGetVersion
	}{
		{
			name:  "4-part version",
			input: "1.0.0.0",
			want: &NuGetVersion{
				Major:           1,
				Minor:           0,
				Patch:           0,
				Revision:        0,
				IsLegacyVersion: true,
				originalString:  "1.0.0.0",
			},
		},
		{
			name:  "4-part with non-zero revision",
			input: "2.5.3.1",
			want: &NuGetVersion{
				Major:           2,
				Minor:           5,
				Patch:           3,
				Revision:        1,
				IsLegacyVersion: true,
				originalString:  "2.5.3.1",
			},
		},
		{
			name:  "4-part with all non-zero",
			input: "10.20.30.40",
			want: &NuGetVersion{
				Major:           10,
				Minor:           20,
				Patch:           30,
				Revision:        40,
				IsLegacyVersion: true,
				originalString:  "10.20.30.40",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}

			if got.Major != tt.want.Major {
				t.Errorf("Major = %v, want %v", got.Major, tt.want.Major)
			}
			if got.Minor != tt.want.Minor {
				t.Errorf("Minor = %v, want %v", got.Minor, tt.want.Minor)
			}
			if got.Patch != tt.want.Patch {
				t.Errorf("Patch = %v, want %v", got.Patch, tt.want.Patch)
			}
			if got.Revision != tt.want.Revision {
				t.Errorf("Revision = %v, want %v", got.Revision, tt.want.Revision)
			}
			if got.IsLegacyVersion != tt.want.IsLegacyVersion {
				t.Errorf("IsLegacyVersion = %v, want %v", got.IsLegacyVersion, tt.want.IsLegacyVersion)
			}
		})
	}
}

func TestParse_Legacy_String(t *testing.T) {
	// Test that legacy versions format correctly
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0.0.0", "1.0.0.0"},
		{"2.5.3.1", "2.5.3.1"},
		{"10.20.30.40", "10.20.30.40"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			got := v.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}
```

### Verification Steps

```bash
# Run legacy-specific tests
go test ./version -v -run TestParse_Legacy

# Run all version tests
go test ./version -v

# Check coverage
go test ./version -cover
```

Expected: All tests pass

### Testing

All tests included above.

### Commit

```bash
git add version/
git commit -m "feat: add legacy 4-part version support

- Parse Major.Minor.Build.Revision format
- Set IsLegacyVersion flag for 4-part versions
- Preserve revision in string formatting
- Add comprehensive legacy version tests

Chunk: M1.4
Status: ✓ Complete"
```

### Next Chunk
→ **M1.5: Version Package - Comparison**

---

## Chunk M1.5: Version Package - Comparison

**Time:** 2 hours
**Dependencies:** M1.4
**Status:** [ ] Not Started

### What You'll Build

Implement version comparison following NuGet rules (SemVer 2.0 with legacy support).

### Step-by-Step Instructions

**1. Add to `version/version.go`:**

```go
// Compare compares two NuGet versions.
//
// Returns:
//   -1 if v < other
//    0 if v == other
//    1 if v > other
//
// Comparison follows NuGet SemVer 2.0 rules:
//   1. Compare Major, Minor, Patch numerically
//   2. For legacy versions, compare Revision
//   3. Release version > Prerelease version
//   4. Compare prerelease labels lexicographically
//   5. Metadata is ignored in comparison
func (v *NuGetVersion) Compare(other *NuGetVersion) int {
	if v == nil && other == nil {
		return 0
	}
	if v == nil {
		return -1
	}
	if other == nil {
		return 1
	}

	// Compare major
	if v.Major != other.Major {
		return intCompare(v.Major, other.Major)
	}

	// Compare minor
	if v.Minor != other.Minor {
		return intCompare(v.Minor, other.Minor)
	}

	// Compare patch
	if v.Patch != other.Patch {
		return intCompare(v.Patch, other.Patch)
	}

	// Compare revision (only if both are legacy versions)
	if v.IsLegacyVersion && other.IsLegacyVersion {
		if v.Revision != other.Revision {
			return intCompare(v.Revision, other.Revision)
		}
	}

	// Compare release labels
	return compareReleaseLabels(v.ReleaseLabels, other.ReleaseLabels)
}

// Equals returns true if v equals other.
func (v *NuGetVersion) Equals(other *NuGetVersion) bool {
	return v.Compare(other) == 0
}

// LessThan returns true if v < other.
func (v *NuGetVersion) LessThan(other *NuGetVersion) bool {
	return v.Compare(other) < 0
}

// LessThanOrEqual returns true if v <= other.
func (v *NuGetVersion) LessThanOrEqual(other *NuGetVersion) bool {
	return v.Compare(other) <= 0
}

// GreaterThan returns true if v > other.
func (v *NuGetVersion) GreaterThan(other *NuGetVersion) bool {
	return v.Compare(other) > 0
}

// GreaterThanOrEqual returns true if v >= other.
func (v *NuGetVersion) GreaterThanOrEqual(other *NuGetVersion) bool {
	return v.Compare(other) >= 0
}

// intCompare compares two integers.
func intCompare(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// compareReleaseLabels compares prerelease labels.
//
// Rules:
//   - No labels (release) > with labels (prerelease)
//   - Numeric labels < alphanumeric labels
//   - Longer label list > shorter (if all previous labels equal)
func compareReleaseLabels(a, b []string) int {
	// Release version (no labels) is greater than prerelease
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	if len(a) == 0 {
		return 1 // a is release, b is prerelease
	}
	if len(b) == 0 {
		return -1 // a is prerelease, b is release
	}

	// Compare label by label
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	for i := 0; i < minLen; i++ {
		result := compareLabel(a[i], b[i])
		if result != 0 {
			return result
		}
	}

	// If all labels equal, longer list is greater
	return intCompare(len(a), len(b))
}

// compareLabel compares two prerelease labels.
//
// Numeric labels are compared numerically and are less than alphanumeric labels.
func compareLabel(a, b string) int {
	aNum, aIsNum := parseAsInt(a)
	bNum, bIsNum := parseAsInt(b)

	if aIsNum && bIsNum {
		return intCompare(aNum, bNum)
	}

	if aIsNum {
		return -1 // numeric < alphanumeric
	}
	if bIsNum {
		return 1 // alphanumeric > numeric
	}

	// Both alphanumeric, compare lexicographically
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// parseAsInt tries to parse string as int.
func parseAsInt(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}
```

**2. Create `version/compare_test.go`:**

```go
package version

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int // -1, 0, 1
	}{
		// Basic comparisons
		{"equal", "1.0.0", "1.0.0", 0},
		{"major less", "1.0.0", "2.0.0", -1},
		{"major greater", "2.0.0", "1.0.0", 1},
		{"minor less", "1.0.0", "1.1.0", -1},
		{"minor greater", "1.1.0", "1.0.0", 1},
		{"patch less", "1.0.0", "1.0.1", -1},
		{"patch greater", "1.0.1", "1.0.0", 1},

		// Prerelease comparisons
		{"release > prerelease", "1.0.0", "1.0.0-beta", 1},
		{"prerelease < release", "1.0.0-beta", "1.0.0", -1},
		{"prerelease alpha < beta", "1.0.0-alpha", "1.0.0-beta", -1},
		{"prerelease beta > alpha", "1.0.0-beta", "1.0.0-alpha", 1},

		// Numeric vs alphanumeric labels
		{"numeric < alphanumeric", "1.0.0-1", "1.0.0-alpha", -1},
		{"alphanumeric > numeric", "1.0.0-alpha", "1.0.0-1", 1},

		// Multiple labels
		{"shorter label list", "1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"longer label list", "1.0.0-alpha.1", "1.0.0-alpha", 1},
		{"equal multiple labels", "1.0.0-alpha.1", "1.0.0-alpha.1", 0},

		// Metadata ignored
		{"metadata ignored 1", "1.0.0+a", "1.0.0+b", 0},
		{"metadata ignored 2", "1.0.0+build", "1.0.0", 0},

		// Legacy versions
		{"legacy equal", "1.0.0.0", "1.0.0.0", 0},
		{"legacy revision", "1.0.0.0", "1.0.0.1", -1},
		{"legacy vs semver", "1.0.0.1", "1.0.0", 0}, // revision ignored when comparing to semver
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1 := MustParse(tt.v1)
			v2 := MustParse(tt.v2)

			got := v1.Compare(v2)
			if got != tt.expected {
				t.Errorf("Compare(%s, %s) = %d, want %d", tt.v1, tt.v2, got, tt.expected)
			}
		})
	}
}

func TestEquals(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1.0.0", "1.0.0", true},
		{"1.0.0", "2.0.0", false},
		{"1.0.0+a", "1.0.0+b", true}, // metadata ignored
	}

	for _, tt := range tests {
		v1 := MustParse(tt.v1)
		v2 := MustParse(tt.v2)

		got := v1.Equals(v2)
		if got != tt.expected {
			t.Errorf("Equals(%s, %s) = %v, want %v", tt.v1, tt.v2, got, tt.expected)
		}
	}
}

func TestLessThan(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1.0.0", "2.0.0", true},
		{"2.0.0", "1.0.0", false},
		{"1.0.0", "1.0.0", false},
	}

	for _, tt := range tests {
		v1 := MustParse(tt.v1)
		v2 := MustParse(tt.v2)

		got := v1.LessThan(v2)
		if got != tt.expected {
			t.Errorf("LessThan(%s, %s) = %v, want %v", tt.v1, tt.v2, got, tt.expected)
		}
	}
}

func TestGreaterThan(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"2.0.0", "1.0.0", true},
		{"1.0.0", "2.0.0", false},
		{"1.0.0", "1.0.0", false},
	}

	for _, tt := range tests {
		v1 := MustParse(tt.v1)
		v2 := MustParse(tt.v2)

		got := v1.GreaterThan(v2)
		if got != tt.expected {
			t.Errorf("GreaterThan(%s, %s) = %v, want %v", tt.v1, tt.v2, got, tt.expected)
		}
	}
}

// Benchmark version comparison
func BenchmarkCompare(b *testing.B) {
	v1 := MustParse("1.2.3-beta.1")
	v2 := MustParse("1.2.3-beta.2")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v1.Compare(v2)
	}
}
```

### Verification Steps

```bash
# Run comparison tests
go test ./version -v -run TestCompare

# Run all tests
go test ./version -v

# Run benchmarks
go test ./version -bench=BenchmarkCompare

# Check for zero allocations in comparison
go test ./version -bench=BenchmarkCompare -benchmem
```

Expected: BenchmarkCompare should show 0 allocs/op

### Testing

All tests included above.

### Commit

```bash
git add version/
git commit -m "feat: implement version comparison logic

- Add Compare() method following NuGet SemVer 2.0 rules
- Implement convenience methods (Equals, LessThan, GreaterThan, etc.)
- Support legacy version comparison
- Prerelease label comparison (numeric < alphanumeric)
- Metadata ignored in comparisons
- Zero-allocation implementation
- Add comprehensive comparison tests
- Add benchmark for comparison performance

Chunk: M1.5
Status: ✓ Complete"
```

### Next Chunk
→ **M1.6: Version Package - Ranges**

---

## Chunk M1.6: Version Package - Ranges

**Time:** 3 hours
**Dependencies:** M1.5
**Status:** [ ] Not Started

### What You'll Build

Implement version range parsing and evaluation (e.g., `[1.0, 2.0)`, `(, 3.0]`).

### Step-by-Step Instructions

**1. Create `version/range.go`:**

```go
package version

import (
	"fmt"
	"strings"
)

// VersionRange represents a range of acceptable versions.
//
// Syntax:
//   [1.0, 2.0]   - 1.0 ≤ x ≤ 2.0 (inclusive)
//   (1.0, 2.0)   - 1.0 < x < 2.0 (exclusive)
//   [1.0, 2.0)   - 1.0 ≤ x < 2.0 (mixed)
//   [1.0, )      - x ≥ 1.0 (open upper)
//   (, 2.0]      - x ≤ 2.0 (open lower)
//   1.0          - x ≥ 1.0 (implicit minimum)
type VersionRange struct {
	MinVersion   *NuGetVersion
	MaxVersion   *NuGetVersion
	MinInclusive bool
	MaxInclusive bool
}

// ParseVersionRange parses a version range string.
func ParseVersionRange(s string) (*VersionRange, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("version range cannot be empty")
	}

	// Check if it starts with bracket (range syntax)
	if strings.HasPrefix(s, "[") || strings.HasPrefix(s, "(") {
		return parseRangeSyntax(s)
	}

	// Otherwise, it's a simple version (>= that version)
	v, err := Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid version range: %w", err)
	}

	return &VersionRange{
		MinVersion:   v,
		MinInclusive: true,
		MaxVersion:   nil,
		MaxInclusive: false,
	}, nil
}

// parseRangeSyntax parses bracket range syntax like [1.0, 2.0).
func parseRangeSyntax(s string) (*VersionRange, error) {
	// Determine inclusive/exclusive from brackets
	if !strings.HasPrefix(s, "[") && !strings.HasPrefix(s, "(") {
		return nil, fmt.Errorf("range must start with [ or (")
	}
	if !strings.HasSuffix(s, "]") && !strings.HasSuffix(s, ")") {
		return nil, fmt.Errorf("range must end with ] or )")
	}

	minInclusive := strings.HasPrefix(s, "[")
	maxInclusive := strings.HasSuffix(s, "]")

	// Remove brackets
	s = s[1 : len(s)-1]

	// Split on comma
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("range must have exactly two parts separated by comma")
	}

	minPart := strings.TrimSpace(parts[0])
	maxPart := strings.TrimSpace(parts[1])

	var minVersion, maxVersion *NuGetVersion
	var err error

	// Parse min version (empty means no lower bound)
	if minPart != "" {
		minVersion, err = Parse(minPart)
		if err != nil {
			return nil, fmt.Errorf("invalid min version: %w", err)
		}
	}

	// Parse max version (empty means no upper bound)
	if maxPart != "" {
		maxVersion, err = Parse(maxPart)
		if err != nil {
			return nil, fmt.Errorf("invalid max version: %w", err)
		}
	}

	return &VersionRange{
		MinVersion:   minVersion,
		MaxVersion:   maxVersion,
		MinInclusive: minInclusive,
		MaxInclusive: maxInclusive,
	}, nil
}

// Satisfies returns true if the version satisfies this range.
func (r *VersionRange) Satisfies(version *NuGetVersion) bool {
	if version == nil {
		return false
	}

	// Check lower bound
	if r.MinVersion != nil {
		cmp := version.Compare(r.MinVersion)
		if r.MinInclusive {
			if cmp < 0 {
				return false
			}
		} else {
			if cmp <= 0 {
				return false
			}
		}
	}

	// Check upper bound
	if r.MaxVersion != nil {
		cmp := version.Compare(r.MaxVersion)
		if r.MaxInclusive {
			if cmp > 0 {
				return false
			}
		} else {
			if cmp >= 0 {
				return false
			}
		}
	}

	return true
}

// FindBestMatch finds the highest version that satisfies this range.
//
// Returns nil if no version satisfies the range.
func (r *VersionRange) FindBestMatch(versions []*NuGetVersion) *NuGetVersion {
	var best *NuGetVersion

	for _, v := range versions {
		if r.Satisfies(v) {
			if best == nil || v.GreaterThan(best) {
				best = v
			}
		}
	}

	return best
}

// String returns the string representation of the range.
func (r *VersionRange) String() string {
	if r.MinVersion != nil && r.MaxVersion == nil && r.MinInclusive {
		// Simple ">= version" case
		return r.MinVersion.String()
	}

	minBracket := "("
	if r.MinInclusive {
		minBracket = "["
	}
	maxBracket := ")"
	if r.MaxInclusive {
		maxBracket = "]"
	}

	minStr := ""
	if r.MinVersion != nil {
		minStr = r.MinVersion.String()
	}

	maxStr := ""
	if r.MaxVersion != nil {
		maxStr = r.MaxVersion.String()
	}

	return fmt.Sprintf("%s%s, %s%s", minBracket, minStr, maxStr, maxBracket)
}
```

**2. Create `version/range_test.go`:**

```go
package version

import "testing"

func TestParseVersionRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"inclusive both", "[1.0, 2.0]", false},
		{"exclusive both", "(1.0, 2.0)", false},
		{"mixed", "[1.0, 2.0)", false},
		{"open upper", "[1.0, )", false},
		{"open lower", "(, 2.0]", false},
		{"simple version", "1.0.0", false},
		{"empty", "", true},
		{"missing bracket", "[1.0, 2.0", true},
		{"wrong brackets", "]1.0, 2.0[", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseVersionRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersionRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVersionRange_Satisfies(t *testing.T) {
	tests := []struct {
		name     string
		rangeStr string
		version  string
		expected bool
	}{
		// Inclusive ranges
		{"inclusive min", "[1.0, 2.0]", "1.0.0", true},
		{"inclusive max", "[1.0, 2.0]", "2.0.0", true},
		{"inclusive middle", "[1.0, 2.0]", "1.5.0", true},
		{"inclusive below", "[1.0, 2.0]", "0.9.0", false},
		{"inclusive above", "[1.0, 2.0]", "2.1.0", false},

		// Exclusive ranges
		{"exclusive min", "(1.0, 2.0)", "1.0.0", false},
		{"exclusive max", "(1.0, 2.0)", "2.0.0", false},
		{"exclusive middle", "(1.0, 2.0)", "1.5.0", true},

		// Mixed
		{"mixed min inclusive", "[1.0, 2.0)", "1.0.0", true},
		{"mixed max exclusive", "[1.0, 2.0)", "2.0.0", false},

		// Open-ended
		{"open upper", "[1.0, )", "100.0.0", true},
		{"open lower", "(, 2.0]", "0.1.0", true},

		// Simple version (>= semantics)
		{"simple satisfies", "1.0.0", "1.5.0", true},
		{"simple not satisfies", "1.0.0", "0.9.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ParseVersionRange(tt.rangeStr)
			if err != nil {
				t.Fatalf("ParseVersionRange() error = %v", err)
			}

			v := MustParse(tt.version)
			got := r.Satisfies(v)

			if got != tt.expected {
				t.Errorf("Satisfies(%s) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}

func TestVersionRange_FindBestMatch(t *testing.T) {
	versions := []*NuGetVersion{
		MustParse("1.0.0"),
		MustParse("1.5.0"),
		MustParse("2.0.0"),
		MustParse("2.5.0"),
		MustParse("3.0.0"),
	}

	tests := []struct {
		name     string
		rangeStr string
		expected string
	}{
		{"range 1.0-2.0", "[1.0, 2.0]", "2.0.0"},
		{"range 1.0-2.0 exclusive", "[1.0, 2.0)", "1.5.0"},
		{"open upper from 2.0", "[2.0, )", "3.0.0"},
		{"open lower to 2.0", "(, 2.0]", "2.0.0"},
		{"no match", "[10.0, 20.0]", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ParseVersionRange(tt.rangeStr)
			if err != nil {
				t.Fatalf("ParseVersionRange() error = %v", err)
			}

			got := r.FindBestMatch(versions)

			if tt.expected == "" {
				if got != nil {
					t.Errorf("FindBestMatch() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("FindBestMatch() = nil, want %s", tt.expected)
				} else if got.String() != tt.expected {
					t.Errorf("FindBestMatch() = %v, want %s", got, tt.expected)
				}
			}
		})
	}
}
```

### Verification Steps

```bash
# Run range tests
go test ./version -v -run TestVersionRange

# Run all tests
go test ./version -v

# Check coverage
go test ./version -cover
```

### Testing

All tests included above.

### Commit

```bash
git add version/
git commit -m "feat: add version range parsing and evaluation

- Implement VersionRange type
- Parse bracket syntax ([1.0, 2.0], (1.0, 2.0), etc.)
- Support open-ended ranges ([1.0, ), (, 2.0])
- Support simple version (>= semantics)
- Implement Satisfies() to check if version is in range
- Implement FindBestMatch() to find highest matching version
- Add comprehensive range tests

Chunk: M1.6
Status: ✓ Complete"
```

### Next Chunk
→ **M1.7: Version Package - Floating Versions**

---
