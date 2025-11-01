package restore

import (
	"context"
	"fmt"
	"strings"

	"github.com/willibrandon/gonuget/core"
	"github.com/willibrandon/gonuget/core/resolver"
	"github.com/willibrandon/gonuget/version"
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
	return e.FormatError(true) // Default to colorized for backward compatibility
}

// FormatError formats the error with optional ANSI color codes.
// When colorize is false, output is plain text (for piped output).
// When colorize is true, error codes are displayed in bright red (for TTY output).
func (e *NuGetError) FormatError(colorize bool) string {
	sourcesStr := ""
	if len(e.Sources) > 0 {
		// Reverse source order to match dotnet's display order
		reversedSources := make([]string, len(e.Sources))
		for i, s := range e.Sources {
			reversedSources[len(e.Sources)-1-i] = s
		}
		sourcesStr = " in source(s): " + strings.Join(reversedSources, ", ")
	}

	// Add ANSI red color for error code only if colorize is enabled (TTY output)
	if colorize {
		// ANSI color codes (use bright red like dotnet)
		// \033[1;31m = bold + red (bright red)
		// \033[0m = reset all attributes
		const (
			red   = "\033[1;31m"
			reset = "\033[0m"
		)
		// Add 4-space indentation and ANSI red color for error code (matches dotnet)
		return fmt.Sprintf("    %s : %serror %s%s: %s%s", e.ProjectPath, red, e.Code, reset, e.Message, sourcesStr)
	}

	// Plain text (no colors) for piped output
	return fmt.Sprintf("    %s : error %s: %s%s", e.ProjectPath, e.Code, e.Message, sourcesStr)
}

// Common NuGet error codes (matching NuGet.Client)
const (
	// NU1101: Unable to find package
	ErrorCodePackageNotFound = "NU1101"

	// NU1102: Unable to find package with version constraint
	ErrorCodePackageVersionNotFound = "NU1102"

	// NU1103: Unable to download package
	ErrorCodePackageDownloadFailed = "NU1103"

	// NU1605: Detected package downgrade
	ErrorCodePackageDowngrade = "NU1605"
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
// with per-source version information. Matches dotnet's exact formatting where each line has full project path prefix.
// When colorize is true (TTY output), error codes are displayed in bright red.
// When colorize is false (piped output), output is plain text.
func FormatVersionNotFoundError(projectPath, packageID, versionConstraint string, versionInfos []VersionInfo, errorCode string, colorize bool) string {
	var sb strings.Builder

	// Determine error message based on error code
	var message string
	switch errorCode {
	case ErrorCodePackageDownloadFailed: // NU1103
		message = fmt.Sprintf("Unable to find a stable package %s with version (%s)", packageID, versionConstraint)
	default: // NU1102
		message = fmt.Sprintf("Unable to find package %s with version (%s)", packageID, versionConstraint)
	}

	// Dotnet uses prefix format for NU1102/NU1103: each line has full project path prefix
	// First line: project path + error code + message (all on one line)
	if colorize {
		// TTY output with ANSI colors
		const (
			red   = "\033[1;31m"
			reset = "\033[0m"
		)
		sb.WriteString(fmt.Sprintf("%s : %serror %s%s: %s\n", projectPath, red, errorCode, reset, message))

		// Per-source lines: project path + colored error code + "  - Found..." (2 spaces before dash)
		for _, info := range versionInfos {
			sb.WriteString(fmt.Sprintf("%s : %serror %s%s:   - Found %d version(s) in %s [ Nearest version: %s ]\n",
				projectPath, red, errorCode, reset, info.VersionCount, info.Source, info.NearestVersion))
		}
	} else {
		// Piped output without colors
		sb.WriteString(fmt.Sprintf("%s : error %s: %s\n", projectPath, errorCode, message))

		// Per-source lines: project path + error code + "  - Found..." (2 spaces before dash)
		for _, info := range versionInfos {
			sb.WriteString(fmt.Sprintf("%s : error %s:   - Found %d version(s) in %s [ Nearest version: %s ]\n",
				projectPath, errorCode, info.VersionCount, info.Source, info.NearestVersion))
		}
	}

	// Remove trailing newline
	result := sb.String()
	return strings.TrimSuffix(result, "\n")
}

// buildUnresolvedError converts unresolved packages into NuGet errors (NU1101, NU1102, NU1103).
// Matches NuGet.Client's error detection in RestoreCommand.
func (r *Restorer) buildUnresolvedError(ctx context.Context, unresolvedPkgs []resolver.UnresolvedPackage, projectPath string) []*NuGetError {
	if len(unresolvedPkgs) == 0 {
		return nil
	}

	errors := make([]*NuGetError, 0, len(unresolvedPkgs))
	for _, pkg := range unresolvedPkgs {
		// Try to detect if this is NU1101, NU1102, or NU1103
		queryResult := r.tryGetVersionInfo(ctx, pkg.ID, pkg.VersionRange)

		if queryResult.packageFound && len(queryResult.versionInfos) > 0 {
			// Package exists but no compatible version found
			// Check if this is NU1103 (only prerelease versions satisfy the range when stable requested)
			// Matches NuGet.Client logic: !IsPrereleaseAllowed(range) && HasPrereleaseVersionsOnly(range, allVersions)
			if !isPrereleaseAllowed(pkg.VersionRange) && hasPrereleaseVersionsOnly(pkg.VersionRange, queryResult.allVersions) {
				// NU1103: Only prerelease versions satisfy the range, but stable requested
				err := NewPackageDownloadFailedError(
					projectPath,
					pkg.ID,
					pkg.VersionRange,
					queryResult.versionInfos,
				)
				errors = append(errors, err)
			} else {
				// NU1102: Package exists but no compatible version
				err := NewPackageVersionNotFoundError(
					projectPath,
					pkg.ID,
					pkg.VersionRange,
					queryResult.versionInfos,
				)
				errors = append(errors, err)
			}
		} else {
			// NU1101: Package doesn't exist at all
			err := NewPackageNotFoundError(
				projectPath,
				pkg.ID,
				pkg.VersionRange,
				r.opts.Sources,
			)
			errors = append(errors, err)
		}
	}

	return errors
}

// buildDowngradeErrors converts downgrade warnings into NU1605 errors.
// Matches NuGet.Client's RestoreCommand.GetDowngradeErrors behavior.
func (r *Restorer) buildDowngradeErrors(downgrades []resolver.DowngradeWarning, projectPath string) []*NuGetError {
	if len(downgrades) == 0 {
		return nil
	}

	errors := make([]*NuGetError, 0, len(downgrades))
	for _, downgrade := range downgrades {
		// Format message to match NuGet.Client:
		// "Detected package downgrade: PackageID from CurrentVersion to TargetVersion.
		//  Reference the package directly from the project to select a different version."
		message := fmt.Sprintf("Detected package downgrade: %s from %s to %s. Reference the package directly from the project to select a different version",
			downgrade.PackageID,
			downgrade.CurrentVersion,
			downgrade.TargetVersion)

		// Add path information if available (shows dependency chain)
		if len(downgrade.Path) > 0 {
			// Format: " project -> PackageA 1.0.0 -> PackageB (>= 2.0.0)"
			pathStr := strings.Join(downgrade.Path, " -> ")
			message = fmt.Sprintf("%s. \n %s", message, pathStr)
		}

		err := &NuGetError{
			Code:        ErrorCodePackageDowngrade,
			Message:     message,
			ProjectPath: projectPath,
			PackageID:   downgrade.PackageID,
		}
		errors = append(errors, err)
	}

	return errors
}

// checkVersionAvailability checks if any version satisfying the constraint exists across all sources.
// This is an optimization to fail fast for NU1102/NU1103 cases without running expensive dependency walk.
// Returns version information per source, all versions, all queried source names, and a boolean indicating if constraint can be satisfied.
func (r *Restorer) checkVersionAvailability(ctx context.Context, packageID, versionConstraint string) ([]VersionInfo, []string, []string, bool) {
	// Parse version range constraint
	versionRange, err := version.ParseVersionRange(versionConstraint)
	if err != nil {
		// If we can't parse the constraint, let the walk handle it
		return nil, nil, nil, true
	}

	// Get all repositories from the client
	repos := r.client.GetRepositoryManager().ListRepositories()

	// Parallel source queries for 2x faster error reporting (critical for NU1101/NU1102/NU1103)
	type sourceResult struct {
		index          int
		sourceName     string
		versions       []string
		nearestVersion string
		canSatisfy     bool
		hasVersions    bool
	}

	results := make(chan sourceResult, len(repos))

	// Query all sources in parallel - network I/O is the bottleneck
	for idx, repo := range repos {
		go func(idx int, repo *core.SourceRepository) {
			// Format source name (check V2 first since it also contains "nuget.org")
			sourceName := repo.SourceURL()
			if strings.Contains(sourceName, "/api/v2") {
				sourceName = "NuGet V2"
			} else if strings.Contains(sourceName, "nuget.org") {
				sourceName = "nuget.org"
			}

			// Try to list all versions of this package from this repository
			versions, err := repo.ListVersions(ctx, nil, packageID)

			if err != nil || len(versions) == 0 {
				// Package doesn't exist in this source
				results <- sourceResult{index: idx, sourceName: sourceName, hasVersions: false}
				return
			}

			// Package exists! Optimize by checking max version first for early rejection
			var nearestVersion string
			var maxVersion *version.NuGetVersion

			// Find max version (versions are typically sorted, so check last first)
			// For NU1102 error display: use HIGHEST version (nearest to requested version)
			// For NU1103 error display: use LOWEST prerelease (will be updated later)
			if len(versions) > 0 {
				// Try last version first (usually the highest) for optimization
				if maxV, err := version.Parse(versions[len(versions)-1]); err == nil {
					maxVersion = maxV
					nearestVersion = versions[len(versions)-1]
				}

				// Verify it's actually the max by checking a few more
				for i := len(versions) - 2; i >= 0 && i >= len(versions)-5; i-- {
					if v, err := version.Parse(versions[i]); err == nil {
						if maxVersion == nil || v.Compare(maxVersion) > 0 {
							maxVersion = v
							nearestVersion = versions[i]
						}
					}
				}
			}

			// OPTIMIZATION: If constraint minimum > max version, no version can satisfy
			// This provides fast rejection for NU1102 cases like "99.99.99" > "13.0.4"
			canSatisfy := false
			if maxVersion != nil && versionRange.MinVersion != nil {
				cmp := versionRange.MinVersion.Compare(maxVersion)
				switch {
				case !versionRange.MinInclusive && cmp == 0:
					// Constraint requires > maxVersion, which is impossible
					// Don't set canSatisfy, use nearestVersion = maxVersion
				case cmp > 0:
					// Constraint requires higher than any available version - fast fail
					// Don't set canSatisfy, use nearestVersion = maxVersion
				default:
					// Constraint might be satisfiable - check if max version satisfies
					if versionRange.Satisfies(maxVersion) {
						canSatisfy = true
					} else {
						// Max doesn't satisfy, need to check other versions
						for _, v := range versions {
							nv, err := version.Parse(v)
							if err != nil {
								continue
							}
							if versionRange.Satisfies(nv) {
								canSatisfy = true
								nearestVersion = v
								break
							}
						}
					}
				}
			}

			results <- sourceResult{
				index:          idx,
				sourceName:     sourceName,
				versions:       versions,
				nearestVersion: nearestVersion,
				canSatisfy:     canSatisfy,
				hasVersions:    true,
			}
		}(idx, repo)
	}

	// Collect results from parallel queries and preserve original order
	resultsByIndex := make([]sourceResult, len(repos))
	for range len(repos) {
		result := <-results
		resultsByIndex[result.index] = result
	}

	// Process results in original source order (critical for source name display order)
	versionInfos := make([]VersionInfo, 0, len(repos))
	allVersions := make([]string, 0)
	allSourceNames := make([]string, 0, len(repos))
	canSatisfy := false

	for _, result := range resultsByIndex {
		// Track all sources queried (for NU1101 error reporting)
		allSourceNames = append(allSourceNames, result.sourceName)

		if !result.hasVersions {
			continue
		}

		// Collect all versions for NU1103 detection
		allVersions = append(allVersions, result.versions...)

		if result.canSatisfy {
			canSatisfy = true
		}

		versionInfos = append(versionInfos, VersionInfo{
			Source:         result.sourceName,
			VersionCount:   len(result.versions),
			NearestVersion: result.nearestVersion,
		})
	}

	return versionInfos, allVersions, allSourceNames, canSatisfy
}

// updateNearestVersionForNU1103 updates versionInfos to show the LOWEST prerelease version
// for NU1103 errors (dotnet shows lowest, not highest, for prerelease-only scenarios)
func (r *Restorer) updateNearestVersionForNU1103(versionInfos []VersionInfo, allVersions []string, versionRange *version.Range) []VersionInfo {
	// Parse all versions once
	parsedVersions := make([]*version.NuGetVersion, 0, len(allVersions))
	versionStrings := make([]string, 0, len(allVersions))

	for _, vStr := range allVersions {
		if v, err := version.Parse(vStr); err == nil {
			parsedVersions = append(parsedVersions, v)
			versionStrings = append(versionStrings, vStr)
		}
	}

	// Find lowest prerelease that satisfies numeric bounds
	var lowestPrerelease *version.NuGetVersion
	var lowestPrereleaseStr string

	for i, v := range parsedVersions {
		// Check if it's prerelease and satisfies numeric bounds
		if v.IsPrerelease() && versionRange.SatisfiesNumericBounds(v) {
			if lowestPrerelease == nil || v.LessThan(lowestPrerelease) {
				lowestPrerelease = v
				lowestPrereleaseStr = versionStrings[i]
			}
		}
	}

	// Update all versionInfos to use the lowest prerelease
	if lowestPrereleaseStr != "" {
		updatedInfos := make([]VersionInfo, len(versionInfos))
		for i, info := range versionInfos {
			updatedInfos[i] = VersionInfo{
				Source:         info.Source,
				VersionCount:   info.VersionCount,
				NearestVersion: lowestPrereleaseStr,
			}
		}
		return updatedInfos
	}

	// Fallback: return original if no prerelease found
	return versionInfos
}

// getBestMatch finds the best matching version from available versions based on a version range.
// Matches NuGet.Client's GetBestMatch algorithm in UnresolvedMessages.cs.
//
// Algorithm:
// 1. If no versions available, return empty string
// 2. Find pivot point from range (MinVersion or MaxVersion)
// 3. For ranges with bounds, find first version above pivot that is closest
// 4. If no match, return highest version
//
// Examples:
//   - Range [1.0.0, ), Available [0.7.0, 0.9.0] → 0.7.0 (closest below lower bound)
//   - Range (0.5.0, 1.0.0), Available [0.1.0, 1.0.0] → 1.0.0 (closest to upper bound)
//   - Range (, 1.0.0), Available [2.0.0, 3.0.0] → 2.0.0 (closest above upper bound)
//   - Range [1.*,), Available [0.0.1, 0.9.0] → 0.9.0 (highest below lower bound)
func getBestMatch(versions []string, vr *version.Range) string {
	if len(versions) == 0 {
		return ""
	}

	// Parse all versions
	parsedVersions := make([]*version.NuGetVersion, 0, len(versions))
	for _, v := range versions {
		parsed, err := version.Parse(v)
		if err == nil {
			parsedVersions = append(parsedVersions, parsed)
		}
	}

	if len(parsedVersions) == 0 {
		return ""
	}

	// If no range provided, return highest version
	if vr == nil {
		return parsedVersions[len(parsedVersions)-1].String()
	}

	// Find pivot point (prefer MinVersion, fallback to MaxVersion)
	var ideal *version.NuGetVersion
	switch {
	case vr.MinVersion != nil:
		ideal = vr.MinVersion
	case vr.MaxVersion != nil:
		ideal = vr.MaxVersion
	default:
		// No bounds, return highest version
		return parsedVersions[len(parsedVersions)-1].String()
	}

	var bestMatch *version.NuGetVersion

	// If range has bounds, find first version above pivot that is closest
	if vr.MinVersion != nil || vr.MaxVersion != nil {
		for _, v := range parsedVersions {
			if v.Compare(ideal) == 0 {
				return v.String()
			}

			if v.Compare(ideal) > 0 {
				if bestMatch == nil || v.Compare(bestMatch) < 0 {
					bestMatch = v
				}
			}
		}
	}

	if bestMatch == nil {
		// Take the highest possible version
		bestMatch = parsedVersions[len(parsedVersions)-1]
	}

	return bestMatch.String()
}

// tryGetVersionInfo attempts to query available versions for a package to distinguish NU1101 vs NU1102 vs NU1103.
// Returns version information per source, all version strings, and a boolean indicating if package was found.
func (r *Restorer) tryGetVersionInfo(ctx context.Context, packageID, versionConstraint string) versionQueryResult {
	// Parse version range for best match calculation
	vr, err := version.ParseVersionRange(versionConstraint)
	if err != nil {
		// If parsing fails, use nil range (will fall back to highest version)
		vr = nil
	}

	// Get all repositories from the client
	repos := r.client.GetRepositoryManager().ListRepositories()
	versionInfos := make([]VersionInfo, 0, len(repos))
	allVersions := make([]string, 0)

	for _, repo := range repos {
		// Try to list all versions of this package from this repository
		versions, err := repo.ListVersions(ctx, nil, packageID)

		if err != nil || len(versions) == 0 {
			// Package doesn't exist in this source
			continue
		}

		// Package exists! Collect all versions for NU1103 detection
		allVersions = append(allVersions, versions...)

		// Calculate nearest version based on version range (matches NuGet.Client's GetBestMatch)
		nearestVersion := getBestMatch(versions, vr)

		// Format source name (check V2 first since it also contains "nuget.org")
		sourceName := repo.SourceURL()
		if strings.Contains(sourceName, "/api/v2") {
			sourceName = "NuGet V2"
		} else if strings.Contains(sourceName, "nuget.org") {
			sourceName = "nuget.org"
		}

		versionInfos = append(versionInfos, VersionInfo{
			Source:         sourceName,
			VersionCount:   len(versions),
			NearestVersion: nearestVersion,
		})
	}

	return versionQueryResult{
		versionInfos: versionInfos,
		allVersions:  allVersions,
		packageFound: len(versionInfos) > 0,
	}
}

// hasPrereleaseVersionsOnly checks if prerelease versions satisfy the range but no stable versions do.
// Matches NuGet.Client's HasPrereleaseVersionsOnly logic.
// Returns true if:
//  1. There exists at least one prerelease version that satisfies the range (numeric bounds only)
//  2. There exists NO stable version that satisfies the range (numeric bounds only)
//
// Note: Uses SatisfiesNumericBounds instead of Satisfies to check if versions WOULD satisfy
// the range if the prerelease restriction were lifted. This is necessary for NU1103 detection.
func hasPrereleaseVersionsOnly(versionRangeStr string, versions []string) bool {
	vr, err := version.ParseVersionRange(versionRangeStr)
	if err != nil {
		return false
	}

	// Check if this is an exact version range (e.g., [1.0.0-alpha])
	// Exact version ranges require exact match including prerelease labels
	isExactVersion := vr.MinVersion != nil && vr.MaxVersion != nil &&
		vr.MinInclusive && vr.MaxInclusive &&
		vr.MinVersion.Equals(vr.MaxVersion)

	hasPrereleaseInRange := false
	hasStableInRange := false

	for _, versionStr := range versions {
		v, err := version.Parse(versionStr)
		if err != nil {
			continue
		}

		// For exact version ranges, require exact match (including prerelease)
		// For other ranges, check numeric bounds only (ignore prerelease restriction)
		satisfies := false
		if isExactVersion {
			satisfies = vr.Satisfies(v)
		} else {
			satisfies = vr.SatisfiesNumericBounds(v)
		}

		if satisfies {
			if v.IsPrerelease() {
				hasPrereleaseInRange = true
			} else {
				hasStableInRange = true
			}
		}
	}

	// True if prerelease versions satisfy the range but no stable versions do
	return hasPrereleaseInRange && !hasStableInRange
}

// isPrereleaseAllowed checks if the version range allows prerelease versions.
// Matches NuGet.Client's IsPrereleaseAllowed logic.
// Returns true if the min or max version of the range has a prerelease label.
func isPrereleaseAllowed(versionRangeStr string) bool {
	vr, err := version.ParseVersionRange(versionRangeStr)
	if err != nil {
		return false
	}

	if vr.MinVersion != nil && vr.MinVersion.IsPrerelease() {
		return true
	}
	if vr.MaxVersion != nil && vr.MaxVersion.IsPrerelease() {
		return true
	}

	return false
}
