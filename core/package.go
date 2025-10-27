// Package core provides core NuGet types for package identity and metadata.
//
// It defines PackageIdentity for uniquely identifying packages and
// PackageMetadata for complete package information including dependencies.
package core

import (
	"strings"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

// PackageIdentity represents a unique package identifier.
type PackageIdentity struct {
	// ID is the package identifier (case-insensitive)
	ID string

	// Version is the package version
	Version *version.NuGetVersion
}

// NewPackageIdentity creates a new package identity.
func NewPackageIdentity(id string, ver *version.NuGetVersion) PackageIdentity {
	return PackageIdentity{
		ID:      id,
		Version: ver,
	}
}

// Equals checks if two package identities are equal.
// Package IDs are compared case-insensitively.
func (p PackageIdentity) Equals(other PackageIdentity) bool {
	return strings.EqualFold(p.ID, other.ID) && p.Version.Compare(other.Version) == 0
}

// String returns a string representation of the package identity.
func (p PackageIdentity) String() string {
	return p.ID + " " + p.Version.String()
}

// PackageDependency represents a dependency on another package.
type PackageDependency struct {
	// ID is the dependency package ID
	ID string

	// VersionRange is the accepted version range
	VersionRange *version.Range

	// Include specifies which assets to include (e.g., "All", "Runtime")
	Include []string

	// Exclude specifies which assets to exclude
	Exclude []string
}

// PackageDependencyGroup represents dependencies for a specific target framework.
type PackageDependencyGroup struct {
	// TargetFramework is the framework these dependencies apply to
	TargetFramework *frameworks.NuGetFramework

	// Dependencies is the list of package dependencies
	Dependencies []PackageDependency
}

// PackageMetadata represents complete package metadata.
type PackageMetadata struct {
	// Identity is the package identity
	Identity PackageIdentity

	// Title is the human-friendly package title
	Title string

	// Authors are the package authors
	Authors []string

	// Owners are the package owners
	Owners []string

	// Description is the package description
	Description string

	// Summary is a short package summary
	Summary string

	// ProjectURL is the project website URL
	ProjectURL string

	// LicenseURL is the license URL
	LicenseURL string

	// IconURL is the icon URL
	IconURL string

	// Tags are package tags/keywords
	Tags []string

	// DependencyGroups contains dependencies organized by target framework
	DependencyGroups []PackageDependencyGroup

	// RequireLicenseAcceptance indicates if license acceptance is required
	RequireLicenseAcceptance bool

	// Listed indicates if the package is listed in search results
	Listed bool
}

// GetDependenciesForFramework returns dependencies for a specific target framework.
// Returns the most compatible dependency group, or nil if no compatible group found.
func (m *PackageMetadata) GetDependenciesForFramework(target *frameworks.NuGetFramework) []PackageDependency {
	if target == nil || len(m.DependencyGroups) == 0 {
		return nil
	}

	// Look for exact match first
	for _, group := range m.DependencyGroups {
		if group.TargetFramework != nil &&
			group.TargetFramework.Framework == target.Framework &&
			group.TargetFramework.Version.Compare(target.Version) == 0 {
			return group.Dependencies
		}
	}

	// Find nearest compatible framework
	var available []*frameworks.NuGetFramework
	for _, group := range m.DependencyGroups {
		if group.TargetFramework != nil {
			available = append(available, group.TargetFramework)
		}
	}

	nearest := frameworks.GetNearest(target, available)
	if nearest == nil {
		return nil
	}

	// Return dependencies for nearest framework
	for _, group := range m.DependencyGroups {
		if group.TargetFramework != nil &&
			group.TargetFramework.Framework == nearest.Framework &&
			group.TargetFramework.Version.Compare(nearest.Version) == 0 {
			return group.Dependencies
		}
	}

	return nil
}
