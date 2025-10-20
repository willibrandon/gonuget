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
