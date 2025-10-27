package resolver

import "fmt"

// DependencyResult indicates the result of evaluating a dependency against the graph
type DependencyResult int

const (
	// DependencyResultAcceptable - Dependency can be added to graph
	DependencyResultAcceptable DependencyResult = iota
	// DependencyResultEclipsed - Dependency is shadowed by another version
	DependencyResultEclipsed
	// DependencyResultPotentiallyDowngraded - Dependency might cause a downgrade
	DependencyResultPotentiallyDowngraded
	// DependencyResultCycle - Dependency creates a cycle
	DependencyResultCycle
)

// LibraryIncludeFlags specifies what should be included from a dependency.
// Maps to NuGet's LibraryIncludeFlags for PrivateAssets/ExcludeAssets support.
type LibraryIncludeFlags int

const (
	// LibraryIncludeFlagsNone - Include nothing
	LibraryIncludeFlagsNone LibraryIncludeFlags = 0
	// LibraryIncludeFlagsRuntime - Include runtime assets
	LibraryIncludeFlagsRuntime LibraryIncludeFlags = 1 << 0
	// LibraryIncludeFlagsCompile - Include compile-time assets
	LibraryIncludeFlagsCompile LibraryIncludeFlags = 1 << 1
	// LibraryIncludeFlagsBuild - Include build assets
	LibraryIncludeFlagsBuild LibraryIncludeFlags = 1 << 2
	// LibraryIncludeFlagsContentFiles - Include content files
	LibraryIncludeFlagsContentFiles LibraryIncludeFlags = 1 << 3
	// LibraryIncludeFlagsNative - Include native assets
	LibraryIncludeFlagsNative LibraryIncludeFlags = 1 << 4
	// LibraryIncludeFlagsAnalyzers - Include analyzers
	LibraryIncludeFlagsAnalyzers LibraryIncludeFlags = 1 << 5
	// LibraryIncludeFlagsBuildTransitive - Include transitive build assets
	LibraryIncludeFlagsBuildTransitive LibraryIncludeFlags = 1 << 6
	// LibraryIncludeFlagsAll - Include everything
	LibraryIncludeFlagsAll LibraryIncludeFlags = 0x7F
)

// PackageDependency represents a dependency on another package.
// Maps to NuGet's LibraryDependency.
type PackageDependency struct {
	ID              string
	VersionRange    string
	TargetFramework string // Empty = all frameworks

	// Include/Exclude flags for assets
	IncludeType LibraryIncludeFlags
	ExcludeType LibraryIncludeFlags

	// SuppressParent - when LibraryIncludeFlagsAll, parent is completely suppressed (PrivateAssets="All")
	SuppressParent LibraryIncludeFlags
}

// PackageDependencyInfo represents complete package metadata with dependencies.
// Maps to NuGet's RemoteResolveResult.
type PackageDependencyInfo struct {
	ID           string
	Version      string
	Dependencies []PackageDependency

	// For framework-specific dependencies
	DependencyGroups []DependencyGroup

	// IsUnresolved indicates this package could not be found
	// Maps to LibraryType.Unresolved in NuGet.Client
	IsUnresolved bool
}

// Key returns a unique key for this package
func (p *PackageDependencyInfo) Key() string {
	return fmt.Sprintf("%s|%s", p.ID, p.Version)
}

func (p *PackageDependencyInfo) String() string {
	return fmt.Sprintf("%s %s", p.ID, p.Version)
}

// DependencyGroup represents dependencies for a specific target framework
type DependencyGroup struct {
	TargetFramework string
	Dependencies    []PackageDependency
}

// ResolutionResult represents the result of dependency resolution.
// Maps to NuGet's RestoreTargetGraph.
type ResolutionResult struct {
	Packages   []*PackageDependencyInfo
	Conflicts  []VersionConflict
	Downgrades []DowngradeWarning
	Cycles     []CycleReport
	Unresolved []UnresolvedPackage // Packages that could not be resolved
}

// Success returns true if resolution completed without unresolved packages.
// Matches NuGet.Client's success check: graphs.All(g => g.Unresolved.Count == 0)
func (r *ResolutionResult) Success() bool {
	return len(r.Unresolved) == 0
}

// VersionConflict represents a version conflict between dependencies
type VersionConflict struct {
	PackageID string
	Versions  []string
	Paths     [][]string // Path from root to each conflicting version
}

// DowngradeWarning represents a potential package downgrade
type DowngradeWarning struct {
	PackageID      string
	CurrentVersion string
	TargetVersion  string
	Path           []string // Path from root to downgrade
}

// CycleReport provides detailed information about a detected cycle
type CycleReport struct {
	// PackageID involved in the cycle
	PackageID string

	// PathToSelf from root to the cycle point
	PathToSelf []string

	// Depth at which cycle was detected
	Depth int

	// Description is a human-readable description
	Description string
}

// UnresolvedPackage represents a package that could not be resolved.
// Maps to NuGet's LibraryRange with LibraryType.Unresolved.
type UnresolvedPackage struct {
	// ID is the package identifier
	ID string

	// VersionRange is the requested version range
	VersionRange string

	// TargetFramework where this was unresolved (empty for all)
	TargetFramework string

	// ErrorCode is the NuGet error code (NU1101, NU1102, NU1103)
	ErrorCode string

	// Message is the detailed error message
	Message string

	// Sources lists sources that were checked
	Sources []string

	// AvailableVersions lists versions found (for NU1102)
	AvailableVersions []string

	// NearestVersion is the closest version found (for NU1102)
	NearestVersion string
}

// NuGetErrorCode represents standard NuGet error codes
type NuGetErrorCode string

const (
	// NU1101 - No versions of package exist on any configured source
	NU1101 NuGetErrorCode = "NU1101"

	// NU1102 - Package exists but no version matches the requested range
	NU1102 NuGetErrorCode = "NU1102"

	// NU1103 - Only prerelease versions available when stable requested
	NU1103 NuGetErrorCode = "NU1103"
)
