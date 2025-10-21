package packaging

import (
	"net/url"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

// PackageMetadata represents package metadata for building.
type PackageMetadata struct {
	// Required
	ID          string
	Version     *version.NuGetVersion
	Description string
	Authors     []string

	// Optional
	Title                    string
	Owners                   []string
	ProjectURL               *url.URL
	IconURL                  *url.URL
	Icon                     string
	LicenseURL               *url.URL
	LicenseMetadata          *LicenseMetadata
	RequireLicenseAcceptance bool
	DevelopmentDependency    bool
	Summary                  string
	ReleaseNotes             string
	Copyright                string
	Language                 string
	Tags                     []string
	Serviceable              bool
	Readme                   string

	// Version constraints
	MinClientVersion *version.NuGetVersion

	// Complex elements
	DependencyGroups         []PackageDependencyGroup
	FrameworkReferenceGroups []PackageFrameworkReferenceGroup
	FrameworkAssemblies      []PackageFrameworkAssembly
	PackageTypes             []PackageTypeInfo
	Repository               *PackageRepositoryMetadata
}

// PackageDependencyGroup represents dependencies for a target framework.
type PackageDependencyGroup struct {
	TargetFramework *frameworks.NuGetFramework
	Dependencies    []PackageDependency
}

// PackageDependency represents a single package dependency.
type PackageDependency struct {
	ID           string
	VersionRange *version.VersionRange
	Include      []string // Asset include filters
	Exclude      []string // Asset exclude filters
}

// PackageFrameworkReferenceGroup represents framework references for a TFM.
type PackageFrameworkReferenceGroup struct {
	TargetFramework *frameworks.NuGetFramework
	References      []string
}

// PackageFrameworkAssembly represents a framework assembly reference.
type PackageFrameworkAssembly struct {
	AssemblyName     string
	TargetFrameworks []*frameworks.NuGetFramework
}

// PackageTypeInfo represents a package type.
type PackageTypeInfo struct {
	Name    string
	Version *version.NuGetVersion
}

// PackageRepositoryMetadata represents repository metadata.
type PackageRepositoryMetadata struct {
	Type   string
	URL    string
	Branch string
	Commit string
}
