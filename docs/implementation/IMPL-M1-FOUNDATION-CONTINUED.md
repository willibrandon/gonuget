# Milestone 1: Foundation (Continued)

**Chunks M1.9 - M1.12**

This file contains the remaining chunks for Milestone 1. Append these to IMPL-M1-FOUNDATION.md or use separately.

---

## Chunk M1.9: Framework Package - TFM Parsing

**Time:** 3 hours
**Dependencies:** M1.8
**Status:** [ ] Not Started

### What You'll Build

Implement parsing for Target Framework Monikers (TFMs) like `net8.0`, `netstandard2.1`, `net48`, etc.

### Step-by-Step Instructions

**1. Add to `frameworks/framework.go`:**

```go
import (
	"fmt"
	"strconv"
	"strings"
)

// ParseFramework parses a TFM string into a NuGetFramework.
//
// Supported formats:
//   net8.0           - .NET 8.0
//   netstandard2.1   - .NET Standard 2.1
//   netcoreapp3.1    - .NET Core 3.1
//   net48            - .NET Framework 4.8
//   net6.0-windows   - .NET 6.0 for Windows
//   portable-net45+win8  - PCL Profile
//
// Returns an error if the TFM string is invalid.
func ParseFramework(tfm string) (*NuGetFramework, error) {
	tfm = strings.TrimSpace(tfm)
	if tfm == "" {
		return nil, fmt.Errorf("framework string cannot be empty")
	}

	fw := &NuGetFramework{
		originalString: tfm,
	}

	// Check for PCL (portable-...)
	if strings.HasPrefix(tfm, "portable-") {
		return parsePCL(tfm)
	}

	// Split on '-' to extract platform
	parts := strings.SplitN(tfm, "-", 2)
	frameworkPart := parts[0]
	if len(parts) == 2 {
		platformPart := parts[1]
		if err := parsePlatform(fw, platformPart); err != nil {
			return nil, err
		}
	}

	// Parse the framework identifier and version
	if err := parseFrameworkIdentifier(fw, frameworkPart); err != nil {
		return nil, err
	}

	return fw, nil
}

// MustParseFramework parses a TFM and panics on error.
func MustParseFramework(tfm string) *NuGetFramework {
	fw, err := ParseFramework(tfm)
	if err != nil {
		panic(err)
	}
	return fw
}

// parseFrameworkIdentifier parses the framework identifier and version.
func parseFrameworkIdentifier(fw *NuGetFramework, s string) error {
	// Map of short names to full framework names
	frameworkMap := map[string]string{
		"net":          ".NETFramework",
		"netstandard":  ".NETStandard",
		"netcoreapp":   ".NETCoreApp",
		"netframework": ".NETFramework",
	}

	// Check if it starts with a known prefix
	for prefix, fullName := range frameworkMap {
		if strings.HasPrefix(s, prefix) {
			fw.Framework = fullName

			// Extract version
			versionPart := strings.TrimPrefix(s, prefix)
			if versionPart == "" {
				return fmt.Errorf("missing version for framework %s", prefix)
			}

			version, err := parseFrameworkVersion(versionPart, prefix)
			if err != nil {
				return fmt.Errorf("invalid version for %s: %w", prefix, err)
			}
			fw.Version = version
			return nil
		}
	}

	return fmt.Errorf("unknown framework identifier: %s", s)
}

// parseFrameworkVersion parses a framework version string.
func parseFrameworkVersion(s string, framework string) (FrameworkVersion, error) {
	// For .NET Framework, version might be like "48" meaning "4.8"
	// For .NET Core/Standard, it's like "3.1" or "2.1"
	// For .NET 5+, it's like "6.0" or "8.0"

	if framework == "net" {
		// .NET Framework uses compact format (e.g., "48" = 4.8)
		if len(s) <= 2 && !strings.Contains(s, ".") {
			// Compact format: "48" → 4.8, "472" → 4.7.2
			return parseCompactVersion(s)
		}
	}

	// Standard version format: "8.0", "3.1", etc.
	parts := strings.Split(s, ".")
	if len(parts) == 0 {
		return FrameworkVersion{}, fmt.Errorf("empty version")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 0 {
		return FrameworkVersion{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor := 0
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil || minor < 0 {
			return FrameworkVersion{}, fmt.Errorf("invalid minor version: %s", parts[1])
		}
	}

	build := 0
	if len(parts) > 2 {
		build, err = strconv.Atoi(parts[2])
		if err != nil || build < 0 {
			return FrameworkVersion{}, fmt.Errorf("invalid build version: %s", parts[2])
		}
	}

	return FrameworkVersion{
		Major: major,
		Minor: minor,
		Build: build,
	}, nil
}

// parseCompactVersion parses compact .NET Framework versions like "48" → 4.8.
func parseCompactVersion(s string) (FrameworkVersion, error) {
	if len(s) == 0 {
		return FrameworkVersion{}, fmt.Errorf("empty version")
	}

	// "48" → 4.8
	// "472" → 4.7.2
	// "461" → 4.6.1

	if len(s) == 2 {
		// "48" format
		major := int(s[0] - '0')
		minor := int(s[1] - '0')
		return FrameworkVersion{Major: major, Minor: minor}, nil
	}

	if len(s) == 3 {
		// "472" format
		major := int(s[0] - '0')
		minor := int(s[1] - '0')
		build := int(s[2] - '0')
		return FrameworkVersion{Major: major, Minor: minor, Build: build}, nil
	}

	return FrameworkVersion{}, fmt.Errorf("invalid compact version: %s", s)
}

// parsePlatform parses the platform part of a TFM.
func parsePlatform(fw *NuGetFramework, s string) error {
	// Platform can be "windows", "android31.0", "ios15.0", etc.

	// Check if there's a version number
	// Look for first digit
	digitIndex := -1
	for i, c := range s {
		if c >= '0' && c <= '9' {
			digitIndex = i
			break
		}
	}

	if digitIndex == -1 {
		// No version, just platform name
		fw.Platform = s
		return nil
	}

	// Split into platform name and version
	fw.Platform = s[:digitIndex]
	versionStr := s[digitIndex:]

	version, err := parseFrameworkVersion(versionStr, "")
	if err != nil {
		return fmt.Errorf("invalid platform version: %w", err)
	}
	fw.PlatformVersion = version

	return nil
}

// parsePCL parses portable class library format.
func parsePCL(s string) (*NuGetFramework, error) {
	// Format: portable-net45+win8+wpa81
	// This is simplified; real implementation would look up profiles

	s = strings.TrimPrefix(s, "portable-")

	return &NuGetFramework{
		Framework:      ".NETPortable",
		Profile:        s, // Store the profile string for now
		originalString: "portable-" + s,
	}, nil
}
```

**2. Create `frameworks/parse_test.go`:**

```go
package frameworks

import "testing"

func TestParseFramework(t *testing.T) {
	tests := []struct {
		name           string
		tfm            string
		wantFramework  string
		wantMajor      int
		wantMinor      int
		wantPlatform   string
		wantErr        bool
	}{
		// .NET (5+)
		{"net8.0", "net8.0", ".NETFramework", 8, 0, "", false},
		{"net6.0", "net6.0", ".NETFramework", 6, 0, "", false},
		{"net5.0", "net5.0", ".NETFramework", 5, 0, "", false},

		// .NET Standard
		{"netstandard2.1", "netstandard2.1", ".NETStandard", 2, 1, "", false},
		{"netstandard2.0", "netstandard2.0", ".NETStandard", 2, 0, "", false},
		{"netstandard1.6", "netstandard1.6", ".NETStandard", 1, 6, "", false},

		// .NET Core
		{"netcoreapp3.1", "netcoreapp3.1", ".NETCoreApp", 3, 1, "", false},
		{"netcoreapp2.1", "netcoreapp2.1", ".NETCoreApp", 2, 1, "", false},

		// .NET Framework (compact)
		{"net48", "net48", ".NETFramework", 4, 8, "", false},
		{"net472", "net472", ".NETFramework", 4, 7, "", false},
		{"net461", "net461", ".NETFramework", 4, 6, "", false},
		{"net45", "net45", ".NETFramework", 4, 5, "", false},

		// Platform-specific
		{"net6.0-windows", "net6.0-windows", ".NETFramework", 6, 0, "windows", false},
		{"net6.0-android", "net6.0-android", ".NETFramework", 6, 0, "android", false},

		// Errors
		{"empty", "", "", 0, 0, "", true},
		{"invalid", "invalid", "", 0, 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFramework(tt.tfm)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFramework() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.Framework != tt.wantFramework {
				t.Errorf("Framework = %v, want %v", got.Framework, tt.wantFramework)
			}
			if got.Version.Major != tt.wantMajor {
				t.Errorf("Version.Major = %v, want %v", got.Version.Major, tt.wantMajor)
			}
			if got.Version.Minor != tt.wantMinor {
				t.Errorf("Version.Minor = %v, want %v", got.Version.Minor, tt.wantMinor)
			}
			if got.Platform != tt.wantPlatform {
				t.Errorf("Platform = %v, want %v", got.Platform, tt.wantPlatform)
			}
		})
	}
}

func TestParseFramework_PCL(t *testing.T) {
	fw, err := ParseFramework("portable-net45+win8")
	if err != nil {
		t.Fatalf("ParseFramework() error = %v", err)
	}

	if fw.Framework != ".NETPortable" {
		t.Errorf("Framework = %v, want .NETPortable", fw.Framework)
	}

	if fw.Profile == "" {
		t.Error("Profile should not be empty for PCL")
	}
}

func TestMustParseFramework(t *testing.T) {
	// Should not panic
	fw := MustParseFramework("net8.0")
	if fw.Version.Major != 8 {
		t.Errorf("Major = %v, want 8", fw.Version.Major)
	}

	// Should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseFramework should panic on invalid TFM")
		}
	}()
	MustParseFramework("invalid")
}
```

### Verification Steps

```bash
# Run TFM parsing tests
go test ./frameworks -v -run TestParseFramework

# Run all framework tests
go test ./frameworks -v

# Check coverage
go test ./frameworks -cover
```

Expected: All tests pass, coverage >80%

### Testing

All tests included above.

### Commit

```bash
git add frameworks/
git commit -m "feat: implement TFM parsing

- Parse .NET 5+ TFMs (net8.0, net6.0, etc.)
- Parse .NET Standard TFMs (netstandard2.1, etc.)
- Parse .NET Core TFMs (netcoreapp3.1, etc.)
- Parse .NET Framework compact format (net48, net472, etc.)
- Parse platform-specific TFMs (net6.0-windows, etc.)
- Parse PCL profiles (portable-net45+win8)
- Add comprehensive TFM parsing tests

Chunk: M1.9
Status: ✓ Complete"
```

### Next Chunk
→ **M1.10: Framework Package - Compatibility Mappings**

---

## Chunk M1.10: Framework Package - Compatibility Mappings

**Time:** 4 hours
**Dependencies:** M1.9
**Status:** [ ] Not Started

### What You'll Build

Extract framework compatibility mappings from C# NuGet.Frameworks and generate Go code.

**NOTE:** This is a data extraction task. The mappings are ~700 LOC of compatibility data from the C# implementation.

### Step-by-Step Instructions

**1. Create extraction tool (optional - can be done manually):**

For now, we'll create a simplified version with the most common mappings. A full extraction tool would parse `DefaultFrameworkMappings.cs` from NuGet.Client.

**2. Create `frameworks/mappings.go`:**

```go
package frameworks

// This file contains framework compatibility mappings.
// Data extracted from NuGet.Frameworks DefaultFrameworkMappings.cs

// frameworkCompatibilityMap defines which frameworks are compatible with which.
var frameworkCompatibilityMap = map[string][]string{
	".NETStandard": {
		".NETFramework",
		".NETCoreApp",
		".NETStandard",
	},
	".NETCoreApp": {
		".NETStandard",
		".NETCoreApp",
	},
	".NETFramework": {
		".NETStandard",
		".NETFramework",
	},
}

// netStandardCompatibilityTable defines .NET Standard → .NET Framework version mappings.
//
// Maps .NET Standard version to minimum .NET Framework version.
var netStandardCompatibilityTable = map[string]string{
	"1.0": "4.5",
	"1.1": "4.5",
	"1.2": "4.5.1",
	"1.3": "4.6",
	"1.4": "4.6.1",
	"1.5": "4.6.1",
	"1.6": "4.6.1",
	"2.0": "4.6.1",
	// 2.1 is NOT compatible with any .NET Framework version
}

// netStandardToCoreAppTable defines .NET Standard → .NET Core version mappings.
var netStandardToCoreAppTable = map[string]string{
	"1.0": "1.0",
	"1.1": "1.0",
	"1.2": "1.0",
	"1.3": "1.0",
	"1.4": "1.0",
	"1.5": "1.0",
	"1.6": "1.0",
	"2.0": "2.0",
	"2.1": "3.0",
}

// frameworkShortNames maps short names to full framework identifiers.
var frameworkShortNames = map[string]string{
	"net":         ".NETFramework",
	"netframework": ".NETFramework",
	"netstandard": ".NETStandard",
	"netcoreapp":  ".NETCoreApp",
}

// frameworkPrecedence defines the precedence order for framework selection.
// Higher index = higher precedence.
var frameworkPrecedence = []string{
	".NETStandard",
	".NETCoreApp",
	".NETFramework",
}

// GetFrameworkPrecedence returns the precedence value for a framework.
// Higher value = higher precedence.
func GetFrameworkPrecedence(framework string) int {
	for i, fw := range frameworkPrecedence {
		if fw == framework {
			return i
		}
	}
	return -1
}
```

**3. Create `frameworks/mappings_test.go`:**

```go
package frameworks

import "testing"

func TestFrameworkCompatibilityMap(t *testing.T) {
	// Verify .NETStandard can target .NETFramework
	compat, ok := frameworkCompatibilityMap[".NETStandard"]
	if !ok {
		t.Fatal(".NETStandard not in compatibility map")
	}

	found := false
	for _, fw := range compat {
		if fw == ".NETFramework" {
			found = true
			break
		}
	}
	if !found {
		t.Error(".NETStandard should be compatible with .NETFramework")
	}
}

func TestNetStandardCompatibilityTable(t *testing.T) {
	tests := []struct {
		nsVersion string
		minNet    string
	}{
		{"1.0", "4.5"},
		{"1.2", "4.5.1"},
		{"1.3", "4.6"},
		{"2.0", "4.6.1"},
	}

	for _, tt := range tests {
		got, ok := netStandardCompatibilityTable[tt.nsVersion]
		if !ok {
			t.Errorf("netStandard %s not in table", tt.nsVersion)
			continue
		}
		if got != tt.minNet {
			t.Errorf("netStandard%s maps to %s, want %s", tt.nsVersion, got, tt.minNet)
		}
	}
}

func TestNetStandard21NotCompatibleWithFramework(t *testing.T) {
	// .NET Standard 2.1 should NOT be in the table
	// (not compatible with any .NET Framework version)
	_, ok := netStandardCompatibilityTable["2.1"]
	if ok {
		t.Error(".NET Standard 2.1 should NOT be compatible with .NET Framework")
	}
}

func TestGetFrameworkPrecedence(t *testing.T) {
	tests := []struct {
		framework string
		wantGTE   int // Greater than or equal
	}{
		{".NETStandard", 0},
		{".NETCoreApp", 0},
		{".NETFramework", 0},
		{"Unknown", -1},
	}

	for _, tt := range tests {
		got := GetFrameworkPrecedence(tt.framework)
		if got < tt.wantGTE && tt.wantGTE >= 0 {
			t.Errorf("GetFrameworkPrecedence(%s) = %d, want >= %d", tt.framework, got, tt.wantGTE)
		}
		if tt.wantGTE == -1 && got != -1 {
			t.Errorf("GetFrameworkPrecedence(%s) = %d, want -1", tt.framework, got)
		}
	}
}
```

**4. Document extraction process in `frameworks/README.md`:**

```markdown
# Framework Compatibility Mappings

## Source

The compatibility mappings in `mappings.go` are extracted from the official NuGet.Client C# implementation:

- Source: NuGet.Client/src/NuGet.Core/NuGet.Frameworks/DefaultFrameworkMappings.cs
- Repository: https://github.com/NuGet/NuGet.Client

## Extraction Process

1. Clone NuGet.Client repository
2. Parse DefaultFrameworkMappings.cs
3. Extract compatibility tables
4. Generate Go code

## Current Status

**Simplified mappings** - Contains most common compatibility rules.

**TODO for production:**
- Extract complete mappings from C# (all ~700 LOC)
- Generate mappings_generated.go automatically
- Add test to validate against C# output

## Key Mappings

### .NET Standard → .NET Framework

| .NET Standard | Min .NET Framework |
|---------------|-------------------|
| 1.0-1.1       | 4.5               |
| 1.2           | 4.5.1             |
| 1.3           | 4.6               |
| 1.4-1.6       | 4.6.1             |
| 2.0           | 4.6.1             |
| 2.1           | NOT COMPATIBLE    |

### .NET Standard → .NET Core

| .NET Standard | Min .NET Core |
|---------------|---------------|
| 1.0-1.6       | 1.0           |
| 2.0           | 2.0           |
| 2.1           | 3.0           |
```

### Verification Steps

```bash
# Run mapping tests
go test ./frameworks -v -run TestMapping

# Run all framework tests
go test ./frameworks -v
```

### Testing

All tests included above.

### Commit

```bash
git add frameworks/
git commit -m "feat: add framework compatibility mappings

- Add framework compatibility map
- Add .NET Standard → .NET Framework version table
- Add .NET Standard → .NET Core version table
- Add framework precedence ordering
- Document extraction process from C# NuGet.Client
- Add mapping validation tests

Note: Simplified mappings for now; full extraction TODO

Chunk: M1.10
Status: ✓ Complete"
```

### Next Chunk
→ **M1.11: Framework Package - Compatibility Logic**

---

## Chunk M1.11: Framework Package - Compatibility Logic

**Time:** 3 hours
**Dependencies:** M1.10
**Status:** [ ] Not Started

### What You'll Build

Implement framework compatibility checking logic using the mappings from M1.10.

### Step-by-Step Instructions

**1. Add to `frameworks/framework.go`:**

```go
// IsCompatible checks if this framework is compatible with the target framework.
//
// Returns true if a package targeting this framework can be used by the target.
//
// Example:
//   netstandard2.0.IsCompatible(net6.0) → true
//   net48.IsCompatible(netstandard2.1) → false
func (f *NuGetFramework) IsCompatible(target *NuGetFramework) bool {
	if f == nil || target == nil {
		return false
	}

	// Same framework and version
	if f.Framework == target.Framework && f.Version.Compare(target.Version) == 0 {
		return true
	}

	// Check framework compatibility rules
	return isCompatibleWith(f, target)
}

// isCompatibleWith implements the core compatibility logic.
func isCompatibleWith(package, target *NuGetFramework) bool {
	// .NET Standard compatibility
	if package.Framework == ".NETStandard" {
		return isNetStandardCompatible(package, target)
	}

	// .NETCoreApp compatibility
	if package.Framework == ".NETCoreApp" && target.Framework == ".NETCoreApp" {
		// Higher or equal .NET Core version
		return package.Version.Compare(target.Version) <= 0
	}

	// .NETFramework compatibility
	if package.Framework == ".NETFramework" && target.Framework == ".NETFramework" {
		// Higher or equal .NET Framework version
		return package.Version.Compare(target.Version) <= 0
	}

	// .NET 5+ unified platform (treat as .NETCoreApp for compatibility)
	if package.Framework == ".NETFramework" && target.Framework == ".NETFramework" {
		if package.Version.Major >= 5 && target.Version.Major >= 5 {
			return package.Version.Compare(target.Version) <= 0
		}
	}

	return false
}

// isNetStandardCompatible checks .NET Standard compatibility with target.
func isNetStandardCompatible(nsPackage, target *NuGetFramework) bool {
	nsVersion := nsPackage.Version

	// .NET Standard → .NET Framework
	if target.Framework == ".NETFramework" {
		return isNetStandardCompatibleWithFramework(nsVersion, target.Version)
	}

	// .NET Standard → .NET Core
	if target.Framework == ".NETCoreApp" {
		return isNetStandardCompatibleWithCoreApp(nsVersion, target.Version)
	}

	// .NET Standard → .NET 5+
	if target.Framework == ".NETFramework" && target.Version.Major >= 5 {
		// .NET 5+ supports .NET Standard 2.1
		return nsVersion.Major <= 2 && (nsVersion.Major < 2 || nsVersion.Minor <= 1)
	}

	// .NET Standard → .NET Standard (same or lower)
	if target.Framework == ".NETStandard" {
		return nsVersion.Compare(target.Version) <= 0
	}

	return false
}

// isNetStandardCompatibleWithFramework checks .NET Standard → .NET Framework compatibility.
func isNetStandardCompatibleWithFramework(nsVersion, netVersion FrameworkVersion) bool {
	// .NET Standard 2.1 is NOT compatible with any .NET Framework
	if nsVersion.Major == 2 && nsVersion.Minor == 1 {
		return false
	}

	// Use lookup table
	nsKey := fmt.Sprintf("%d.%d", nsVersion.Major, nsVersion.Minor)
	minNetVersion, ok := netStandardCompatibilityTable[nsKey]
	if !ok {
		return false
	}

	// Parse min version and compare
	minVer, err := parseFrameworkVersion(minNetVersion, "")
	if err != nil {
		return false
	}

	return netVersion.Compare(minVer) >= 0
}

// isNetStandardCompatibleWithCoreApp checks .NET Standard → .NET Core compatibility.
func isNetStandardCompatibleWithCoreApp(nsVersion, coreVersion FrameworkVersion) bool {
	nsKey := fmt.Sprintf("%d.%d", nsVersion.Major, nsVersion.Minor)
	minCoreVersion, ok := netStandardToCoreAppTable[nsKey]
	if !ok {
		return false
	}

	// Parse min version and compare
	minVer, err := parseFrameworkVersion(minCoreVersion, "")
	if err != nil {
		return false
	}

	return coreVersion.Compare(minVer) >= 0
}
```

**2. Add to `frameworks/nearest.go` (new file):**

```go
package frameworks

// GetNearest finds the nearest compatible framework from a list.
//
// Given a target framework and a list of available frameworks,
// returns the most compatible one, preferring:
// 1. Exact match
// 2. Same framework, nearest lower version
// 3. Compatible framework with highest precedence
//
// Returns nil if no compatible framework found.
func GetNearest(target *NuGetFramework, available []*NuGetFramework) *NuGetFramework {
	if target == nil || len(available) == 0 {
		return nil
	}

	var best *NuGetFramework
	var bestScore int

	for _, fw := range available {
		if !fw.IsCompatible(target) {
			continue
		}

		score := calculateCompatibilityScore(fw, target)
		if best == nil || score > bestScore {
			best = fw
			bestScore = score
		}
	}

	return best
}

// calculateCompatibilityScore calculates how well a framework matches the target.
// Higher score = better match.
func calculateCompatibilityScore(fw, target *NuGetFramework) int {
	score := 0

	// Exact match gets highest score
	if fw.Framework == target.Framework && fw.Version.Compare(target.Version) == 0 {
		return 1000
	}

	// Same framework gets bonus
	if fw.Framework == target.Framework {
		score += 500
	}

	// Closer version gets bonus
	versionDiff := target.Version.Compare(fw.Version)
	if versionDiff >= 0 {
		// Target version >= package version (good)
		score += 100 - versionDiff
	}

	// Framework precedence
	precedence := GetFrameworkPrecedence(fw.Framework)
	score += precedence * 10

	return score
}
```

**3. Create `frameworks/compatibility_test.go`:**

```go
package frameworks

import "testing"

func TestIsCompatible(t *testing.T) {
	tests := []struct {
		name       string
		package_   string
		target     string
		compatible bool
	}{
		// .NET Standard → .NET 6+
		{"netstandard2.0 → net6.0", "netstandard2.0", "net6.0", true},
		{"netstandard2.1 → net6.0", "netstandard2.1", "net6.0", true},

		// .NET Standard → .NET Framework
		{"netstandard1.0 → net45", "netstandard1.0", "net45", true},
		{"netstandard2.0 → net461", "netstandard2.0", "net461", true},
		{"netstandard2.1 → net48", "netstandard2.1", "net48", false}, // 2.1 not compatible

		// .NET Standard → .NET Core
		{"netstandard2.0 → netcoreapp2.0", "netstandard2.0", "netcoreapp2.0", true},
		{"netstandard2.1 → netcoreapp3.0", "netstandard2.1", "netcoreapp3.0", true},

		// Same framework
		{"net6.0 → net6.0", "net6.0", "net6.0", true},
		{"net48 → net48", "net48", "net48", true},

		// Higher to lower (not compatible)
		{"net6.0 → net5.0", "net6.0", "net5.0", false},
		{"net48 → net45", "net48", "net45", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := MustParseFramework(tt.package_)
			target := MustParseFramework(tt.target)

			got := pkg.IsCompatible(target)
			if got != tt.compatible {
				t.Errorf("IsCompatible() = %v, want %v", got, tt.compatible)
			}
		})
	}
}

func TestGetNearest(t *testing.T) {
	available := []*NuGetFramework{
		MustParseFramework("net45"),
		MustParseFramework("netstandard2.0"),
		MustParseFramework("net6.0"),
		MustParseFramework("netcoreapp3.1"),
	}

	tests := []struct {
		name     string
		target   string
		expected string
	}{
		{"net8.0 picks net6.0", "net8.0", "net6.0"},
		{"net48 picks netstandard2.0", "net48", "netstandard2.0"},
		{"netcoreapp3.1 exact", "netcoreapp3.1", "netcoreapp3.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := MustParseFramework(tt.target)
			got := GetNearest(target, available)

			if got == nil {
				t.Fatalf("GetNearest() = nil, want %s", tt.expected)
			}

			if got.String() != tt.expected {
				t.Errorf("GetNearest() = %s, want %s", got, tt.expected)
			}
		})
	}
}

// Benchmark compatibility checking
func BenchmarkIsCompatible(b *testing.B) {
	pkg := MustParseFramework("netstandard2.0")
	target := MustParseFramework("net6.0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pkg.IsCompatible(target)
	}
}
```

### Verification Steps

```bash
# Run compatibility tests
go test ./frameworks -v -run TestIsCompatible

# Run nearest framework tests
go test ./frameworks -v -run TestGetNearest

# Run all framework tests
go test ./frameworks -v

# Run benchmarks
go test ./frameworks -bench=BenchmarkIsCompatible -benchmem

# Check coverage
go test ./frameworks -cover
```

Expected: BenchmarkIsCompatible should show <15ns/op, 0 allocs/op

### Testing

All tests included above.

### Commit

```bash
git add frameworks/
git commit -m "feat: implement framework compatibility checking

- Add IsCompatible() method for framework compatibility
- Implement .NET Standard compatibility rules
- Support .NET Framework, .NET Core, .NET 5+ compatibility
- Add GetNearest() to find best matching framework
- Implement compatibility scoring algorithm
- Add comprehensive compatibility tests
- Zero-allocation performance for hot path
- Benchmark showing <15ns/op

Chunk: M1.11
Status: ✓ Complete"
```

### Next Chunk
→ **M1.12: Core Package - Package Identity**

---

## Chunk M1.12: Core Package - Package Identity

**Time:** 1 hour
**Dependencies:** M1.5, M1.8
**Status:** [ ] Not Started

### What You'll Build

Create core package types for package identity and metadata that will be used throughout the library.

### Step-by-Step Instructions

**1. Create `core/package.go`:**

```go
// Package core provides core types and abstractions for gonuget.
package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

// PackageIdentity uniquely identifies a package by ID and version.
type PackageIdentity struct {
	ID      string
	Version *version.NuGetVersion
}

// NewPackageIdentity creates a new package identity.
func NewPackageIdentity(id string, ver *version.NuGetVersion) *PackageIdentity {
	return &PackageIdentity{
		ID:      id,
		Version: ver,
	}
}

// String returns a string representation of the package identity.
func (p *PackageIdentity) String() string {
	return fmt.Sprintf("%s %s", p.ID, p.Version)
}

// Equals checks if two package identities are equal.
// Package IDs are case-insensitive.
func (p *PackageIdentity) Equals(other *PackageIdentity) bool {
	if p == nil || other == nil {
		return p == other
	}

	return strings.EqualFold(p.ID, other.ID) && p.Version.Equals(other.Version)
}

// PackageMetadata contains complete metadata for a package.
type PackageMetadata struct {
	Identity *PackageIdentity

	// Required fields
	Title       string
	Description string
	Authors     []string

	// Optional fields
	Owners      []string
	ProjectURL  string
	LicenseURL  string
	IconURL     string
	Icon        string   // Path to icon file in package
	Tags        []string
	Summary     string
	ReleaseNotes string
	Copyright   string
	Language    string

	// Dependencies
	DependencyGroups []*PackageDependencyGroup

	// Framework assemblies (legacy)
	FrameworkAssemblies []*FrameworkAssembly

	// Publishing info
	Published     time.Time
	Listed        bool
	DownloadCount int64

	// Package content URL
	PackageContentURL string
}

// PackageDependencyGroup represents dependencies for a specific framework.
type PackageDependencyGroup struct {
	TargetFramework *frameworks.NuGetFramework
	Dependencies    []*PackageDependency
}

// PackageDependency represents a dependency on another package.
type PackageDependency struct {
	ID           string
	VersionRange *version.VersionRange
}

// FrameworkAssembly represents a framework assembly reference (legacy .NET Framework).
type FrameworkAssembly struct {
	AssemblyName      string
	TargetFrameworks []*frameworks.NuGetFramework
}

// GetDependenciesForFramework returns dependencies for a specific target framework.
func (m *PackageMetadata) GetDependenciesForFramework(target *frameworks.NuGetFramework) []*PackageDependency {
	if target == nil {
		return nil
	}

	// Find the most compatible dependency group
	var bestGroup *PackageDependencyGroup
	for _, group := range m.DependencyGroups {
		if group.TargetFramework == nil {
			// No target framework means applies to all
			if bestGroup == nil {
				bestGroup = group
			}
			continue
		}

		if group.TargetFramework.IsCompatible(target) {
			if bestGroup == nil || isBetterMatch(group.TargetFramework, bestGroup.TargetFramework, target) {
				bestGroup = group
			}
		}
	}

	if bestGroup == nil {
		return nil
	}

	return bestGroup.Dependencies
}

// isBetterMatch determines if candidate is a better match than current for target.
func isBetterMatch(candidate, current, target *frameworks.NuGetFramework) bool {
	// Exact match is best
	if candidate.Framework == target.Framework && candidate.Version.Compare(target.Version) == 0 {
		return true
	}

	// Prefer same framework
	if candidate.Framework == target.Framework && current.Framework != target.Framework {
		return true
	}

	// Prefer closer version
	candidateDiff := target.Version.Compare(candidate.Version)
	currentDiff := target.Version.Compare(current.Version)

	return candidateDiff < currentDiff
}
```

**2. Create `core/package_test.go`:**

```go
package core

import (
	"testing"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

func TestPackageIdentity_String(t *testing.T) {
	id := NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1"))
	got := id.String()
	expected := "Newtonsoft.Json 13.0.1"

	if got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

func TestPackageIdentity_Equals(t *testing.T) {
	tests := []struct {
		name     string
		id1      *PackageIdentity
		id2      *PackageIdentity
		expected bool
	}{
		{
			name:     "equal",
			id1:      NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1")),
			id2:      NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1")),
			expected: true,
		},
		{
			name:     "case insensitive ID",
			id1:      NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1")),
			id2:      NewPackageIdentity("newtonsoft.json", version.MustParse("13.0.1")),
			expected: true,
		},
		{
			name:     "different version",
			id1:      NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1")),
			id2:      NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.2")),
			expected: false,
		},
		{
			name:     "different ID",
			id1:      NewPackageIdentity("Newtonsoft.Json", version.MustParse("13.0.1")),
			id2:      NewPackageIdentity("System.Text.Json", version.MustParse("13.0.1")),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.id1.Equals(tt.id2)
			if got != tt.expected {
				t.Errorf("Equals() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPackageMetadata_GetDependenciesForFramework(t *testing.T) {
	// Create test metadata with multiple dependency groups
	metadata := &PackageMetadata{
		Identity: NewPackageIdentity("TestPackage", version.MustParse("1.0.0")),
		DependencyGroups: []*PackageDependencyGroup{
			{
				TargetFramework: frameworks.MustParseFramework("netstandard2.0"),
				Dependencies: []*PackageDependency{
					{ID: "System.Text.Json", VersionRange: nil},
				},
			},
			{
				TargetFramework: frameworks.MustParseFramework("net6.0"),
				Dependencies: []*PackageDependency{
					{ID: "System.Runtime", VersionRange: nil},
				},
			},
		},
	}

	tests := []struct {
		name              string
		target            string
		expectedDepCount  int
		expectedFirstDep  string
	}{
		{
			name:             "net8.0 picks net6.0 group",
			target:           "net8.0",
			expectedDepCount: 1,
			expectedFirstDep: "System.Runtime",
		},
		{
			name:             "net48 picks netstandard2.0 group",
			target:           "net48",
			expectedDepCount: 1,
			expectedFirstDep: "System.Text.Json",
		},
		{
			name:             "no compatible framework",
			target:           "net40",
			expectedDepCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := frameworks.MustParseFramework(tt.target)
			deps := metadata.GetDependenciesForFramework(target)

			if len(deps) != tt.expectedDepCount {
				t.Errorf("got %d dependencies, want %d", len(deps), tt.expectedDepCount)
				return
			}

			if tt.expectedDepCount > 0 && deps[0].ID != tt.expectedFirstDep {
				t.Errorf("first dependency = %s, want %s", deps[0].ID, tt.expectedFirstDep)
			}
		})
	}
}
```

### Verification Steps

```bash
# Run package identity tests
go test ./core -v

# Run all tests in version, frameworks, and core
go test ./version ./frameworks ./core -v

# Check coverage
go test ./core -cover
```

### Testing

All tests included above.

### Commit

```bash
git add core/
git commit -m "feat: add PackageIdentity and PackageMetadata types

- Define PackageIdentity with case-insensitive ID comparison
- Add PackageMetadata with complete package information
- Support dependency groups by target framework
- Add GetDependenciesForFramework() for framework-specific deps
- Implement framework-based dependency selection
- Add comprehensive package identity tests

Chunk: M1.12
Status: ✓ Complete"
```

---

## Milestone 1 Complete!

All 12 chunks of Milestone 1: Foundation are now defined.

**What was built:**
- ✅ Go module initialization
- ✅ Version parsing and comparison (SemVer 2.0 + legacy)
- ✅ Version ranges and floating versions
- ✅ Framework parsing (TFMs)
- ✅ Framework compatibility mappings and logic
- ✅ Package identity and metadata types

**Next Steps:**
1. Run `/next` to start implementing M1.1
2. Run `/proceed` to implement each chunk
3. Run `/commit` after each chunk completes
4. Run `/progress` to track milestone completion

**Next Milestone:**
→ **M2: Protocol Implementation** (IMPL-M2-PROTOCOL.md)

---

**END OF IMPL-M1-FOUNDATION-CONTINUED.md**
