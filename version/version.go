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

import (
	"fmt"
	"strconv"
	"strings"
)

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

// IsPrerelease returns true if pre-release labels exist for the version.
//
// A version is considered a prerelease if it has any non-empty release labels
// (e.g., "1.0.0-beta", "1.0.0-rc.1").
//
// Note: Metadata (after '+') does not affect prerelease status.
// For example, "1.0.0+build123" is NOT a prerelease.
//
// Reference: SemanticVersion.IsPrerelease in NuGet.Client
func (v *NuGetVersion) IsPrerelease() bool {
	if v.ReleaseLabels != nil {
		for _, label := range v.ReleaseLabels {
			if label != "" {
				return true
			}
		}
	}
	return false
}

// ToNormalizedString returns the normalized version string.
//
// Normalization rules:
//   - Remove leading zeros: 1.01.1 → 1.1.1
//   - Legacy 4-part versions preserve all parts: 1.0.0.0 → 1.0.0.0
//   - SemVer versions omit trailing zeros: 1.0.0 → 1.0.0
//   - Prerelease labels preserved: 1.0.0-beta → 1.0.0-beta
//   - Metadata preserved: 1.0.0+build → 1.0.0+build
func (v *NuGetVersion) ToNormalizedString() string {
	if v.IsLegacyVersion {
		// Legacy versions: preserve 4-part format
		return fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Patch, v.Revision)
	}

	// SemVer format
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)

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
	if len(numbers) < 1 || len(numbers) > 4 {
		return nil, fmt.Errorf("invalid version format: %q", s)
	}

	// Parse major
	major, err := strconv.Atoi(numbers[0])
	if err != nil || major < 0 {
		return nil, fmt.Errorf("invalid major version: %q", numbers[0])
	}
	v.Major = major

	// Parse minor (default to 0 if not present)
	if len(numbers) >= 2 {
		minor, err := strconv.Atoi(numbers[1])
		if err != nil || minor < 0 {
			return nil, fmt.Errorf("invalid minor version: %q", numbers[1])
		}
		v.Minor = minor
	}

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

// Compare compares two NuGet versions.
//
// Returns:
//
//	-1 if v < other
//	 0 if v == other
//	 1 if v > other
//
// Comparison follows NuGet SemVer 2.0 rules:
//  1. Compare Major, Minor, Patch numerically
//  2. For legacy versions, compare Revision
//  3. Release version > Prerelease version
//  4. Compare prerelease labels lexicographically
//  5. Metadata is ignored in comparison
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
	minLen := min(len(a), len(b))

	for i := range minLen {
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
