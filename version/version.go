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
