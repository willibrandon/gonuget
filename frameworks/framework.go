// Package frameworks provides Target Framework Moniker (TFM) parsing and compatibility checking.
//
// It supports parsing TFMs for .NET, .NET Standard, .NET Core, .NET Framework, and PCL.
//
// Example:
//
//	fw, err := frameworks.ParseFramework("net8.0")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(fw.Framework, fw.Version.Major) // .NETCoreApp 8
package frameworks

import (
	"fmt"
	"strconv"
	"strings"
)

// NuGetFramework represents a Target Framework Moniker (TFM).
type NuGetFramework struct {
	// Framework is the framework identifier (e.g., ".NETFramework", ".NETStandard")
	Framework string

	// Version is the framework version
	Version FrameworkVersion

	// Platform is the platform identifier (e.g., "windows", "android")
	Platform string

	// PlatformVersion is the platform version
	PlatformVersion FrameworkVersion

	// Profile is used for PCL (Portable Class Library) profiles
	Profile string

	// originalString preserves the original TFM string
	originalString string
}

// FrameworkVersion represents a framework version number.
type FrameworkVersion struct {
	Major    int
	Minor    int
	Build    int
	Revision int
}

// AnyFramework represents the special "any" framework that matches all target frameworks.
// Used for dependencies without a target framework group.
var AnyFramework = NuGetFramework{
	Framework: "Any",
	Version:   FrameworkVersion{Major: 0, Minor: 0, Build: 0, Revision: 0},
}

// Compare compares two framework versions.
// Returns -1 if v < other, 0 if v == other, 1 if v > other.
func (v FrameworkVersion) Compare(other FrameworkVersion) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Build != other.Build {
		if v.Build < other.Build {
			return -1
		}
		return 1
	}
	if v.Revision != other.Revision {
		if v.Revision < other.Revision {
			return -1
		}
		return 1
	}
	return 0
}

// String returns the string representation of the framework.
func (fw *NuGetFramework) String() string {
	if fw.originalString != "" {
		return fw.originalString
	}
	return fw.format()
}

// IsAny returns true if this framework represents the special "any" framework.
func (fw *NuGetFramework) IsAny() bool {
	return fw.Framework == "Any"
}

// format creates a formatted TFM string.
func (fw *NuGetFramework) format() string {
	// Implementation for formatting back to TFM string
	// This is a simplified version
	return fw.Framework
}

// ParseFramework parses a TFM string into a NuGetFramework.
//
// Supported formats:
//
//	net10.0          - .NET 10.0 (.NETCoreApp)
//	net9.0           - .NET 9.0 (.NETCoreApp)
//	net8.0           - .NET 8.0 (.NETCoreApp)
//	net5.0           - .NET 5.0 (.NETCoreApp)
//	netstandard2.1   - .NET Standard 2.1
//	netcoreapp3.1    - .NET Core 3.1
//	net481           - .NET Framework 4.8.1 (compact: 3-digit)
//	net48            - .NET Framework 4.8 (compact: 2-digit)
//	net4721          - .NET Framework 4.7.2.1 (compact: 4-digit)
//	net6.0-windows   - .NET 6.0 for Windows (platform-specific)
//	portable-net45+win8  - PCL Profile
//
// .NET 5+ (net5.0, net6.0, etc.) maps to .NETCoreApp.
// .NET Framework 4.x and below (net48, net472, etc.) maps to .NETFramework.
// Compact versions support 2-4 digits without dots (net48, net472, net4721).
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
	// Check prefixes in order from longest to shortest to avoid greedy matching
	// (e.g., "netstandard" before "net")
	prefixes := []struct {
		prefix   string
		fullName string
	}{
		{"netframework", ".NETFramework"},
		{"netstandard", ".NETStandard"},
		{"netcoreapp", ".NETCoreApp"},
		{"net", ""}, // Special handling for "net" prefix
	}

	for _, p := range prefixes {
		if strings.HasPrefix(s, p.prefix) {
			// Extract version
			versionPart := strings.TrimPrefix(s, p.prefix)
			if versionPart == "" {
				return fmt.Errorf("missing version for framework %s", p.prefix)
			}

			// Special handling for "net" prefix - could be .NET Framework or .NET 5+
			if p.prefix == "net" {
				version, err := parseFrameworkVersion(versionPart, p.prefix)
				if err != nil {
					return fmt.Errorf("invalid version for %s: %w", p.prefix, err)
				}
				fw.Version = version

				// .NET 5+ uses .NETCoreApp, .NET Framework 4.x and below uses .NETFramework
				if version.Major >= 5 {
					fw.Framework = ".NETCoreApp"
				} else {
					fw.Framework = ".NETFramework"
				}
				return nil
			}

			fw.Framework = p.fullName
			version, err := parseFrameworkVersion(versionPart, p.prefix)
			if err != nil {
				return fmt.Errorf("invalid version for %s: %w", p.prefix, err)
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
		if len(s) <= 4 && !strings.Contains(s, ".") {
			// Compact format: "48" → 4.8, "472" → 4.7.2, "4721" → 4.7.2.1
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

	// NuGet.Client supports up to 4 digits:
	// "48"   → 4.8.0.0
	// "472"  → 4.7.2.0
	// "461"  → 4.6.1.0
	// "4721" → 4.7.2.1

	// Validate all characters are digits
	for _, c := range s {
		if c < '0' || c > '9' {
			return FrameworkVersion{}, fmt.Errorf("invalid compact version: %s", s)
		}
	}

	if len(s) == 1 {
		// Single digit not valid for compact format
		return FrameworkVersion{}, fmt.Errorf("invalid compact version: %s", s)
	}

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

	if len(s) == 4 {
		// "4721" format - supported by NuGet.Client
		major := int(s[0] - '0')
		minor := int(s[1] - '0')
		build := int(s[2] - '0')
		revision := int(s[3] - '0')
		return FrameworkVersion{Major: major, Minor: minor, Build: build, Revision: revision}, nil
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

// IsCompatible checks if this framework is compatible with the target framework.
//
// Returns true if a package targeting this framework can be used by the target.
//
// Example:
//
//	netstandard2.0.IsCompatible(net6.0) → true
//	net48.IsCompatible(netstandard2.1) → false
func (fw *NuGetFramework) IsCompatible(target *NuGetFramework) bool {
	if fw == nil || target == nil {
		return false
	}

	// Same framework and version
	if fw.Framework == target.Framework && fw.Version.Compare(target.Version) == 0 {
		return true
	}

	// Check framework compatibility rules
	return isCompatibleWith(fw, target)
}

// isCompatibleWith implements the core compatibility logic.
func isCompatibleWith(pkg, target *NuGetFramework) bool {
	// .NET Standard compatibility
	if pkg.Framework == ".NETStandard" {
		return isNetStandardCompatible(pkg, target)
	}

	// .NETCoreApp compatibility
	if pkg.Framework == ".NETCoreApp" && target.Framework == ".NETCoreApp" {
		// Higher or equal .NET Core version
		return pkg.Version.Compare(target.Version) <= 0
	}

	// .NETFramework compatibility
	if pkg.Framework == ".NETFramework" && target.Framework == ".NETFramework" {
		// Higher or equal .NET Framework version
		return pkg.Version.Compare(target.Version) <= 0
	}

	// .NET 5+ unified platform (treat as .NETCoreApp for compatibility)
	if pkg.Framework == ".NETCoreApp" && target.Framework == ".NETCoreApp" {
		if pkg.Version.Major >= 5 && target.Version.Major >= 5 {
			return pkg.Version.Compare(target.Version) <= 0
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
	if target.Framework == ".NETCoreApp" && target.Version.Major >= 5 {
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

	// Use lookup table with struct key (zero allocations)
	minVer, ok := NetStandardCompatibilityTable[versionKey{nsVersion.Major, nsVersion.Minor}]
	if !ok {
		return false
	}

	return netVersion.Compare(minVer) >= 0
}

// isNetStandardCompatibleWithCoreApp checks .NET Standard → .NET Core compatibility.
func isNetStandardCompatibleWithCoreApp(nsVersion, coreVersion FrameworkVersion) bool {
	// Use lookup table with struct key (zero allocations)
	minVer, ok := NetStandardToCoreAppTable[versionKey{nsVersion.Major, nsVersion.Minor}]
	if !ok {
		return false
	}

	return coreVersion.Compare(minVer) >= 0
}
