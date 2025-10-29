package restore

import (
	"fmt"
	"strings"
)

// NuGetError represents a NuGet-specific error with an error code.
type NuGetError struct {
	Code         string        // Error code (e.g., "NU1101", "NU1102")
	Message      string        // Error message
	ProjectPath  string        // Absolute path to project file
	Sources      []string      // Source URLs involved in the error (for NU1101)
	VersionInfos []VersionInfo // Version information per source (for NU1102)
	PackageID    string        // Package ID (for NU1102 formatting)
	Constraint   string        // Version constraint (for NU1102 formatting)
}

// Error implements the error interface.
// Formats NU1101 errors with indentation and ANSI colors to match dotnet output.
func (e *NuGetError) Error() string {
	// ANSI color codes (use bright red like dotnet)
	// \033[1;31m = bold + red (bright red)
	// \033[0m = reset all attributes
	const (
		red   = "\033[1;31m"
		reset = "\033[0m"
	)

	sourcesStr := ""
	if len(e.Sources) > 0 {
		// Reverse source order to match dotnet's display order
		reversedSources := make([]string, len(e.Sources))
		for i, s := range e.Sources {
			reversedSources[len(e.Sources)-1-i] = s
		}
		sourcesStr = " in source(s): " + strings.Join(reversedSources, ", ")
	}

	// Add 4-space indentation and ANSI red color for error code (matches dotnet)
	return fmt.Sprintf("    %s : %serror %s%s: %s%s", e.ProjectPath, red, e.Code, reset, e.Message, sourcesStr)
}

// Common NuGet error codes (matching NuGet.Client)
const (
	// NU1101: Unable to find package
	ErrorCodePackageNotFound = "NU1101"

	// NU1102: Unable to find package with version constraint
	ErrorCodePackageVersionNotFound = "NU1102"

	// NU1103: Unable to download package
	ErrorCodePackageDownloadFailed = "NU1103"
)

// VersionInfo holds version information for NU1102 errors.
type VersionInfo struct {
	Source         string
	VersionCount   int
	NearestVersion string
}

// NewPackageNotFoundError creates a NU1101 error for a package that doesn't exist.
func NewPackageNotFoundError(projectPath, packageID, version string, sources []string) *NuGetError {
	// Format sources (convert URLs to friendly names when possible)
	sourceNames := make([]string, len(sources))
	for i, source := range sources {
		switch {
		case strings.Contains(source, "nuget.org"):
			sourceNames[i] = "nuget.org"
		case strings.Contains(source, "/v2") || strings.Contains(source, "/v2/"):
			sourceNames[i] = "NuGet V2"
		default:
			sourceNames[i] = source
		}
	}

	message := fmt.Sprintf("Unable to find package %s. No packages exist with this id", packageID)

	return &NuGetError{
		Code:        ErrorCodePackageNotFound,
		Message:     message,
		ProjectPath: projectPath,
		Sources:     sourceNames,
		PackageID:   packageID, // Set package ID for cache file logging
	}
}

// NewPackageVersionNotFoundError creates a NU1102 error for when a package exists but no compatible version is found.
func NewPackageVersionNotFoundError(projectPath, packageID, versionConstraint string, versionInfos []VersionInfo) *NuGetError {
	// Format version constraint for display
	// Plain versions (no brackets/parentheses) should be displayed as ">= version"
	displayConstraint := formatVersionConstraintForDisplay(versionConstraint)

	// First line: package name with version constraint
	message := fmt.Sprintf("Unable to find package %s with version (%s)", packageID, displayConstraint)

	return &NuGetError{
		Code:         ErrorCodePackageVersionNotFound,
		Message:      message,
		ProjectPath:  projectPath,
		PackageID:    packageID,
		Constraint:   displayConstraint,
		VersionInfos: versionInfos,
	}
}

// NewPackageDownloadFailedError creates a NU1103 error for when only prerelease versions exist but stable requested.
func NewPackageDownloadFailedError(projectPath, packageID, versionConstraint string, versionInfos []VersionInfo) *NuGetError {
	// Format version constraint for display
	displayConstraint := formatVersionConstraintForDisplay(versionConstraint)

	// Message indicating only prerelease versions available
	message := fmt.Sprintf("Unable to find a stable package %s with version (%s)", packageID, displayConstraint)

	return &NuGetError{
		Code:         ErrorCodePackageDownloadFailed,
		Message:      message,
		ProjectPath:  projectPath,
		PackageID:    packageID,
		Constraint:   displayConstraint,
		VersionInfos: versionInfos,
	}
}

// formatVersionConstraintForDisplay formats a version constraint for error message display.
// Converts NuGet range syntax to dotnet's display format:
// - [1.0.0,) → >= 1.0.0
// - [1.0.0] → = 1.0.0
// - [1.0.0, 2.0.0] → >= 1.0.0 && <= 2.0.0
// - (1.0.0, 2.0.0) → > 1.0.0 && < 2.0.0
func formatVersionConstraintForDisplay(constraint string) string {
	constraint = strings.TrimSpace(constraint)

	// Check if it's bracket range syntax
	if !strings.HasPrefix(constraint, "[") && !strings.HasPrefix(constraint, "(") {
		// Plain version: add ">= " prefix
		return ">= " + constraint
	}

	// Parse the range
	minInclusive := strings.HasPrefix(constraint, "[")
	maxInclusive := strings.HasSuffix(constraint, "]")

	// Remove brackets
	inner := constraint[1 : len(constraint)-1]
	parts := strings.Split(inner, ",")

	if len(parts) != 2 {
		// Single version [1.0.0] → = 1.0.0
		if minInclusive && maxInclusive {
			return "= " + strings.TrimSpace(inner)
		}
		// Fallback: return as-is
		return constraint
	}

	minPart := strings.TrimSpace(parts[0])
	maxPart := strings.TrimSpace(parts[1])

	// Open-ended range [1.0.0,) → >= 1.0.0
	if maxPart == "" {
		if minInclusive {
			return ">= " + minPart
		}
		return "> " + minPart
	}

	// Open-ended lower bound (,2.0.0] → <= 2.0.0
	if minPart == "" {
		if maxInclusive {
			return "<= " + maxPart
		}
		return "< " + maxPart
	}

	// Closed range [1.0.0, 2.0.0] → >= 1.0.0 && <= 2.0.0
	var minOp, maxOp string
	if minInclusive {
		minOp = ">="
	} else {
		minOp = ">"
	}
	if maxInclusive {
		maxOp = "<="
	} else {
		maxOp = "<"
	}

	return fmt.Sprintf("%s %s && %s %s", minOp, minPart, maxOp, maxPart)
}

// FormatVersionNotFoundError formats multi-line version-not-found errors (NU1102 and NU1103)
// with per-source version information. Matches dotnet's exact formatting: indentation, line breaks, and ANSI color codes.
func FormatVersionNotFoundError(projectPath, packageID, versionConstraint string, versionInfos []VersionInfo, errorCode string) string {
	var sb strings.Builder

	// ANSI color codes (use bright red like dotnet)
	// \033[1;31m = bold + red (bright red)
	// \033[0m = reset all attributes
	const (
		red   = "\033[1;31m"
		reset = "\033[0m"
	)

	// Determine error message based on error code
	var message string
	switch errorCode {
	case ErrorCodePackageDownloadFailed: // NU1103
		message = fmt.Sprintf("Unable to find a stable package %s with version (%s)", packageID, versionConstraint)
	default: // NU1102
		message = fmt.Sprintf("Unable to find package %s with version (%s)", packageID, versionConstraint)
	}

	// First line: 4 spaces + project path + " : " + colored "error CODE" + ":"
	// dotnet format: "    /path : error NU1103: " (with newline after)
	sb.WriteString(fmt.Sprintf("    %s : %serror %s%s: \n", projectPath, red, errorCode, reset))

	// Second line: 6 spaces + main message
	sb.WriteString(fmt.Sprintf("      %s\n", message))

	// Per-source lines: 8 spaces + "- Found..."
	for _, info := range versionInfos {
		sb.WriteString(fmt.Sprintf("        - Found %d version(s) in %s [ Nearest version: %s ]\n",
			info.VersionCount, info.Source, info.NearestVersion))
	}

	// Remove trailing newline
	result := sb.String()
	return strings.TrimSuffix(result, "\n")
}
