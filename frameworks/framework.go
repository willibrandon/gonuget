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
	"sort"
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

// String returns the string representation of the framework version.
// It trims trailing zero components to match NuGet.Client behavior:
//   - 4.7.2.0 → "4.7.2"
//   - 6.0.0.0 → "6.0"
//   - 4.8.0.0 → "4.8"
//   - 1.0.0.0 → "1.0"
func (v FrameworkVersion) String() string {
	if v.Revision > 0 {
		return fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Build, v.Revision)
	}
	if v.Build > 0 {
		return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Build)
	}
	if v.Minor > 0 {
		return fmt.Sprintf("%d.%d", v.Major, v.Minor)
	}
	return fmt.Sprintf("%d.0", v.Major)
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

// IsSpecificFramework returns true if this is a specific framework (not special or unsupported).
func (fw *NuGetFramework) IsSpecificFramework() bool {
	return fw.Framework != "" &&
		fw.Framework != "Any" &&
		fw.Framework != "Unsupported" &&
		fw.Framework != "Agnostic" &&
		strings.ToLower(fw.Framework) != "any" &&
		strings.ToLower(fw.Framework) != "unsupported" &&
		strings.ToLower(fw.Framework) != "agnostic"
}

// IsPCL returns true if this is a Portable Class Library framework.
func (fw *NuGetFramework) IsPCL() bool {
	return fw.Framework == ".NETPortable"
}

// IsNet5Era returns true if this is .NET 5+ (.NETCoreApp with version >= 5).
func (fw *NuGetFramework) IsNet5Era() bool {
	return fw.Framework == ".NETCoreApp" && fw.Version.Major >= 5
}

// IsEmpty returns true if the version is empty (0.0.0.0).
func (v FrameworkVersion) IsEmpty() bool {
	return v.Major == 0 && v.Minor == 0 && v.Build == 0 && v.Revision == 0
}

// Equals checks if two frameworks are equal.
func (fw *NuGetFramework) Equals(other *NuGetFramework) bool {
	if fw == nil || other == nil {
		return fw == other
	}
	return fw.Framework == other.Framework &&
		fw.Version.Compare(other.Version) == 0 &&
		fw.Platform == other.Platform &&
		fw.PlatformVersion.Compare(other.PlatformVersion) == 0 &&
		fw.Profile == other.Profile
}

// format creates a formatted TFM string.
func (fw *NuGetFramework) format() string {
	return fw.GetShortFolderName(DefaultFrameworkNameProvider())
}

// GetShortFolderName returns the short folder name representation of the framework.
// This matches NuGet.Client's GetShortFolderName implementation.
func (fw *NuGetFramework) GetShortFolderName(provider FrameworkNameProvider) string {
	var sb strings.Builder

	// Handle special frameworks
	if !fw.IsSpecificFramework() {
		return strings.ToLower(fw.Framework)
	}

	// Handle PCL - matches NuGet.Client GetShortFolderName behavior
	// NuGet.Client DOES expand profile numbers to framework lists
	// Example: "portable-Profile7" -> "portable-net45+win8"
	if fw.IsPCL() {
		sb.WriteString("portable-")
		if fw.Profile != "" {
			// Try to expand profile to framework list
			if strings.HasPrefix(fw.Profile, "Profile") {
				// This is a profile number (e.g., "Profile7")
				// Try to get the framework list for this profile
				if frameworks, ok := provider.TryGetPortableFrameworks(fw.Profile, false); ok && len(frameworks) > 0 {
					// Format each framework and sort alphabetically (case-insensitive)
					shortNames := make([]string, len(frameworks))
					for i, f := range frameworks {
						shortNames[i] = f.GetShortFolderName(provider)
					}
					// Sort case-insensitively to match NuGet.Client's OrdinalIgnoreCase
					sort.Slice(shortNames, func(i, j int) bool {
						return strings.ToLower(shortNames[i]) < strings.ToLower(shortNames[j])
					})
					sb.WriteString(strings.Join(shortNames, "+"))
					return sb.String()
				}
			}
			// If profile contains a framework list (e.g., "net45+win8"), sort it
			if strings.Contains(fw.Profile, "+") {
				parts := strings.Split(fw.Profile, "+")
				sort.Slice(parts, func(i, j int) bool {
					return strings.ToLower(parts[i]) < strings.ToLower(parts[j])
				})
				sb.WriteString(strings.Join(parts, "+"))
			} else {
				// Custom profile or single framework, use as-is (lowercased)
				sb.WriteString(strings.ToLower(fw.Profile))
			}
		}
		return sb.String()
	}

	// Get short identifier
	shortIdentifier := ""
	if short, ok := provider.TryGetShortIdentifier(fw.Framework); ok {
		shortIdentifier = short
	} else {
		shortIdentifier = strings.ToLower(fw.Framework)
	}

	// For .NET 5+, use "net" instead of "netcoreapp"
	if fw.IsNet5Era() {
		shortIdentifier = "net"
	}

	sb.WriteString(shortIdentifier)

	// Add version if not empty
	if !fw.Version.IsEmpty() {
		versionString := provider.GetVersionString(fw.Framework, fw.Version)
		sb.WriteString(versionString)
	}

	// Add profile if present (like "-client" for .NET Framework)
	if fw.Profile != "" {
		if shortProfile, ok := provider.TryGetShortProfile(fw.Framework, fw.Profile); ok {
			if shortProfile != "" {
				sb.WriteString("-")
				sb.WriteString(shortProfile)
			}
		} else {
			sb.WriteString("-")
			sb.WriteString(strings.ToLower(fw.Profile))
		}
	}

	// Add platform if present (like "-windows10.0")
	if fw.Platform != "" {
		sb.WriteString("-")
		sb.WriteString(strings.ToLower(fw.Platform))

		if !fw.PlatformVersion.IsEmpty() {
			// Format platform version - NuGet.Client uses standard .NET version string formatting
			// which omits trailing zeros (e.g., "10.0.19041.0" becomes "10.0.19041")
			sb.WriteString(fw.PlatformVersion.String())
		}
	}

	return sb.String()
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

// NormalizeFrameworkName converts various framework name formats to the standard TFM short folder name.
// It handles:
//   - V3 registration API format: ".NETStandard2.0" -> "netstandard2.0"
//   - Full framework names: ".NETCoreApp,Version=v2.2" -> "netcoreapp2.2"
//   - Already normalized names: "netstandard2.0" -> "netstandard2.0" (passthrough)
func NormalizeFrameworkName(frameworkName string) string {
	if frameworkName == "" {
		return ""
	}

	// Already in TFM format (no dot prefix or comma) - return as-is
	if !strings.HasPrefix(frameworkName, ".") && !strings.Contains(frameworkName, ",") {
		return strings.ToLower(frameworkName)
	}

	// Extract framework identifier and version
	// Handle formats like ".NETStandard2.0" or ".NETCoreApp,Version=v2.2"
	parts := strings.Split(frameworkName, ",")
	nameAndVersion := parts[0] // ".NETStandard2.0" or ".NETCoreApp"

	// Map full framework names to short identifiers
	var shortName string
	var versionStr string

	// Extract version from the name if present (e.g., ".NETStandard2.0")
	hasVersion := strings.Contains(nameAndVersion, "2.0") || strings.Contains(nameAndVersion, "2.1") ||
		strings.Contains(nameAndVersion, "2.2") || strings.Contains(nameAndVersion, "1.") ||
		strings.Contains(nameAndVersion, "3.") || strings.Contains(nameAndVersion, "4.") ||
		strings.Contains(nameAndVersion, "5.") || strings.Contains(nameAndVersion, "6.") ||
		strings.Contains(nameAndVersion, "7.") || strings.Contains(nameAndVersion, "8.")

	switch {
	case hasVersion:
		// Version is embedded in the name
		switch {
		case strings.HasPrefix(nameAndVersion, ".NETStandard"):
			shortName = "netstandard"
			versionStr = strings.TrimPrefix(nameAndVersion, ".NETStandard")
		case strings.HasPrefix(nameAndVersion, ".NETCoreApp"):
			shortName = "netcoreapp"
			versionStr = strings.TrimPrefix(nameAndVersion, ".NETCoreApp")
		case strings.HasPrefix(nameAndVersion, ".NETFramework"):
			shortName = "net"
			versionStr = strings.TrimPrefix(nameAndVersion, ".NETFramework")
			// .NET Framework uses compact form without dots
			versionStr = strings.ReplaceAll(versionStr, ".", "")
		case strings.HasPrefix(nameAndVersion, ".NETPortable"):
			// Portable class library
			return strings.ToLower(frameworkName) // Return as-is for PCL
		}
	case len(parts) > 1 && strings.Contains(parts[1], "Version="):
		// Version is in separate component: ".NETCoreApp,Version=v2.2"
		switch {
		case strings.HasPrefix(nameAndVersion, ".NETStandard"):
			shortName = "netstandard"
		case strings.HasPrefix(nameAndVersion, ".NETCoreApp"):
			shortName = "netcoreapp"
		case strings.HasPrefix(nameAndVersion, ".NETFramework"):
			shortName = "net"
		}

		// Extract version from "Version=v2.2"
		for _, part := range parts[1:] {
			if v, found := strings.CutPrefix(part, "Version="); found {
				versionStr, _ = strings.CutPrefix(v, "v")
				if shortName == "net" {
					// .NET Framework uses compact form
					versionStr = strings.ReplaceAll(versionStr, ".", "")
				}
			}
		}
	default:
		// Unknown format - return lowercase
		return strings.ToLower(frameworkName)
	}

	if shortName == "" {
		return strings.ToLower(frameworkName)
	}

	return shortName + versionStr
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
	// (e.g., "netstandard" before "net", "netcore" before "net")
	prefixes := []struct {
		prefix   string
		fullName string
	}{
		{"netframework", ".NETFramework"},
		{"netstandard", ".NETStandard"},
		{"netcoreapp", ".NETCoreApp"},
		{"netcore", "NetCore"}, // Legacy .NET Core (netcore45, netcore50, etc.)

		// Legacy PCL frameworks (used in portable profiles)
		{"windowsphone", "WindowsPhone"}, // wp, wpa
		{"windows", "Windows"},           // win, win8, win81, etc.
		{"silverlight", "Silverlight"},   // sl
		{"monoandroid", "MonoAndroid"},
		{"monotouch", "MonoTouch"},
		{"monomac", "MonoMac"},
		{"xamarin", "Xamarin"},
		{"tizen", "Tizen"},
		{"dnxcore", "DNXCore"},
		{"dnx", "DNX"},
		{"uap", "UAP"},

		// Short forms for legacy PCL frameworks
		{"wpa", "WindowsPhoneApp"}, // wpa81
		{"wp", "WindowsPhone"},     // wp8, wp7
		{"win", "Windows"},         // win8, win81, win10
		{"sl", "Silverlight"},      // sl4, sl5

		{"net", ""}, // Special handling for "net" prefix
	}

	for _, p := range prefixes {
		if versionPart, ok := strings.CutPrefix(s, p.prefix); ok {
			// Extract version
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

	// Check for special frameworks (any, unsupported, agnostic)
	// These are case-insensitive and have no version
	lower := strings.ToLower(s)
	if lower == "any" || lower == "unsupported" || lower == "agnostic" {
		fw.Framework = s                // Preserve original casing for now
		fw.Version = FrameworkVersion{} // Empty version
		return nil
	}

	return fmt.Errorf("unknown framework identifier: %s", s)
}

// parseFrameworkVersion parses a framework version string.
func parseFrameworkVersion(s string, framework string) (FrameworkVersion, error) {
	// For .NET Framework, version might be like "48" meaning "4.8"
	// For .NET Core/Standard, it's like "3.1" or "2.1"
	// For .NET 5+, it's like "6.0" or "8.0"
	// For legacy NetCore (netcore45, netcore50), uses compact format like "50" = 5.0
	// For legacy PCL frameworks (win8, wp8, wpa81, sl5), uses compact format

	// Frameworks that use compact version format (no dots)
	// .NET Framework requires minimum 2 digits (net40, net45, etc.)
	compactFrameworks := map[string]bool{
		"net":     true,
		"netcore": true,
	}

	// Legacy PCL frameworks that allow single-digit versions (win8, wp8, sl5)
	pclFrameworks := map[string]bool{
		"win": true, // Windows
		"wp":  true, // WindowsPhone
		"wpa": true, // WindowsPhoneApp
		"sl":  true, // Silverlight
	}

	if compactFrameworks[framework] {
		// Compact format, minimum 2 digits for .NET Framework (e.g., "48" = 4.8)
		if len(s) >= 2 && len(s) <= 4 && !strings.Contains(s, ".") {
			return parseCompactVersion(s)
		}
	} else if pclFrameworks[framework] {
		// PCL frameworks allow single-digit versions (e.g., "8" = 8.0)
		if len(s) >= 1 && len(s) <= 4 && !strings.Contains(s, ".") {
			return parseCompactVersion(s)
		}
	}

	// Standard version format: "8.0", "3.1", "4", "2", etc.
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
	// "8"    → 8.0.0.0 (for legacy PCL frameworks like win8, wp8)
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
		// Single digit: "8" → 8.0 (for legacy PCL like win8, wp8, sl5)
		major := int(s[0] - '0')
		return FrameworkVersion{Major: major, Minor: 0}, nil
	}

	if len(s) == 2 {
		// "48" format or "81" format
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
	// Format: portable-net45+win8+wpa81 or portable-Profile259

	originalString := s
	s = strings.TrimPrefix(s, "portable-")

	provider := DefaultFrameworkNameProvider()

	// Try to resolve profile to profile number
	if strings.HasPrefix(s, "Profile") {
		// Already in Profile format (e.g., "Profile259")
		return &NuGetFramework{
			Framework:      ".NETPortable",
			Profile:        s,
			originalString: originalString,
		}, nil
	}

	// Parse as framework list (e.g., "net45+win8")
	// NuGet.Client does NOT convert framework lists to profile numbers
	// It keeps them as-is and formats with alphabetical sorting
	if frameworks, ok := provider.TryGetPortableFrameworks(s, false); ok && len(frameworks) > 0 {
		// Format and sort the framework list
		shortNames := make([]string, len(frameworks))
		for i, fw := range frameworks {
			shortNames[i] = fw.GetShortFolderName(provider)
		}
		// Sort alphabetically (NuGet.Client behavior)
		sortedProfile := sortStrings(shortNames)

		return &NuGetFramework{
			Framework:      ".NETPortable",
			Profile:        strings.Join(sortedProfile, "+"),
			originalString: originalString,
		}, nil
	}

	// Fallback: store as custom profile
	return &NuGetFramework{
		Framework:      ".NETPortable",
		Profile:        s,
		originalString: originalString,
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

// sortStrings sorts a slice of strings alphabetically and returns the sorted slice.
func sortStrings(strs []string) []string {
	sorted := make([]string, len(strs))
	copy(sorted, strs)
	sort.Strings(sorted)
	return sorted
}
