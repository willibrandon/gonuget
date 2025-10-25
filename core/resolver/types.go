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

// ResolutionResult represents the result of dependency resolution
type ResolutionResult struct {
	Packages   []*PackageDependencyInfo
	Conflicts  []VersionConflict
	Downgrades []DowngradeWarning
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
