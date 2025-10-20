package version

import (
	"fmt"
	"strings"
)

// FloatBehavior defines how floating versions behave.
type FloatBehavior int

const (
	// FloatNone means no floating
	FloatNone FloatBehavior = iota

	// FloatPrerelease floats to latest prerelease: 1.0.0-*
	FloatPrerelease

	// FloatRevision floats to latest revision: 1.0.0.*
	FloatRevision

	// FloatPatch floats to latest patch: 1.0.*
	FloatPatch

	// FloatMinor floats to latest minor: 1.*
	FloatMinor

	// FloatMajor floats to latest major: *
	FloatMajor
)

// String returns the string representation of FloatBehavior.
func (f FloatBehavior) String() string {
	switch f {
	case FloatNone:
		return "none"
	case FloatPrerelease:
		return "prerelease"
	case FloatRevision:
		return "revision"
	case FloatPatch:
		return "patch"
	case FloatMinor:
		return "minor"
	case FloatMajor:
		return "major"
	default:
		return "unknown"
	}
}

// FloatRange represents a floating version range.
type FloatRange struct {
	MinVersion    *NuGetVersion
	FloatBehavior FloatBehavior
}

// ParseFloatRange parses floating version ranges like 1.0.*, 1.0.0-*, or *.
func ParseFloatRange(s string) (*FloatRange, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("float range cannot be empty")
	}

	// Wildcard only: *
	if s == "*" {
		return &FloatRange{
			MinVersion:    nil,
			FloatBehavior: FloatMajor,
		}, nil
	}

	// Prerelease float: 1.0.0-*
	if strings.HasSuffix(s, "-*") {
		versionPart := s[:len(s)-2]
		v, err := Parse(versionPart)
		if err != nil {
			return nil, fmt.Errorf("invalid float range: %w", err)
		}
		return &FloatRange{
			MinVersion:    v,
			FloatBehavior: FloatPrerelease,
		}, nil
	}

	// Patch/Minor/Major float: 1.0.*, 1.*, or *.*.*
	if !strings.Contains(s, "*") {
		return nil, fmt.Errorf("float range must contain wildcard: %s", s)
	}

	parts := strings.Split(s, ".")
	floatIndex := -1
	for i, part := range parts {
		if part == "*" {
			floatIndex = i
			break
		}
	}

	if floatIndex == -1 {
		return nil, fmt.Errorf("invalid float range: %s", s)
	}

	// Determine float behavior based on position
	var behavior FloatBehavior
	switch floatIndex {
	case 0:
		behavior = FloatMajor
	case 1:
		behavior = FloatMinor
	case 2:
		behavior = FloatPatch
	case 3:
		behavior = FloatRevision
	default:
		return nil, fmt.Errorf("invalid wildcard position in: %s", s)
	}

	// Build minimum version from non-wildcard parts
	var minVersion *NuGetVersion
	if floatIndex > 0 {
		versionParts := parts[:floatIndex]
		// Pad with zeros to make a valid version
		for len(versionParts) < 2 {
			versionParts = append(versionParts, "0")
		}
		versionStr := strings.Join(versionParts, ".")
		v, err := Parse(versionStr)
		if err != nil {
			return nil, fmt.Errorf("invalid float range: %w", err)
		}
		minVersion = v
	}

	return &FloatRange{
		MinVersion:    minVersion,
		FloatBehavior: behavior,
	}, nil
}

// Satisfies returns true if the version satisfies this floating range.
func (f *FloatRange) Satisfies(version *NuGetVersion) bool {
	if version == nil {
		return false
	}

	// No minimum version means any version satisfies (for * wildcard)
	if f.MinVersion == nil {
		return true
	}

	switch f.FloatBehavior {
	case FloatPrerelease:
		// Must match major.minor.patch exactly, can have any prerelease
		return version.Major == f.MinVersion.Major &&
			version.Minor == f.MinVersion.Minor &&
			version.Patch == f.MinVersion.Patch

	case FloatRevision:
		// Must match major.minor.patch, can have any revision
		return version.Major == f.MinVersion.Major &&
			version.Minor == f.MinVersion.Minor &&
			version.Patch == f.MinVersion.Patch

	case FloatPatch:
		// Must match major.minor, can have any patch
		return version.Major == f.MinVersion.Major &&
			version.Minor == f.MinVersion.Minor

	case FloatMinor:
		// Must match major, can have any minor
		return version.Major == f.MinVersion.Major

	case FloatMajor:
		// Any version satisfies
		return true

	default:
		return false
	}
}

// FindBestMatch finds the highest version that satisfies this floating range.
func (f *FloatRange) FindBestMatch(versions []*NuGetVersion) *NuGetVersion {
	var best *NuGetVersion

	for _, v := range versions {
		if f.Satisfies(v) {
			if best == nil || v.GreaterThan(best) {
				best = v
			}
		}
	}

	return best
}

// String returns the string representation of the floating range.
func (f *FloatRange) String() string {
	if f.MinVersion == nil {
		return "*"
	}

	switch f.FloatBehavior {
	case FloatPrerelease:
		return f.MinVersion.String() + "-*"
	case FloatRevision:
		return fmt.Sprintf("%d.%d.%d.*", f.MinVersion.Major, f.MinVersion.Minor, f.MinVersion.Patch)
	case FloatPatch:
		return fmt.Sprintf("%d.%d.*", f.MinVersion.Major, f.MinVersion.Minor)
	case FloatMinor:
		return fmt.Sprintf("%d.*", f.MinVersion.Major)
	case FloatMajor:
		return "*"
	default:
		return ""
	}
}
