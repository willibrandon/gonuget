package packaging

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

// Nuspec represents a parsed .nuspec manifest.
type Nuspec struct {
	XMLName  xml.Name       `xml:"package"`
	Xmlns    string         `xml:"xmlns,attr,omitempty"`
	Metadata NuspecMetadata `xml:"metadata"`
	Files    []NuspecFile   `xml:"files>file"`
}

// NuspecMetadata represents the metadata section.
type NuspecMetadata struct {
	// Required fields
	ID          string `xml:"id"`
	Version     string `xml:"version"`
	Description string `xml:"description"`
	Authors     string `xml:"authors"`

	// Optional fields
	Title                    string           `xml:"title"`
	Owners                   string           `xml:"owners"`
	ProjectURL               string           `xml:"projectUrl"`
	IconURL                  string           `xml:"iconUrl"`
	Icon                     string           `xml:"icon"`
	LicenseURL               string           `xml:"licenseUrl"`
	License                  *LicenseMetadata `xml:"license"`
	RequireLicenseAcceptance bool             `xml:"requireLicenseAcceptance"`
	DevelopmentDependency    bool             `xml:"developmentDependency"`
	Summary                  string           `xml:"summary"`
	ReleaseNotes             string           `xml:"releaseNotes"`
	Copyright                string           `xml:"copyright"`
	Language                 string           `xml:"language"`
	Tags                     string           `xml:"tags"`
	Serviceable              bool             `xml:"serviceable"`
	Readme                   string           `xml:"readme"`

	// Version constraints
	MinClientVersion string `xml:"minClientVersion,attr"`

	// Complex elements
	Dependencies        *DependenciesElement        `xml:"dependencies"`
	FrameworkReferences *FrameworkReferencesElement `xml:"frameworkReferences"`
	FrameworkAssemblies []FrameworkAssembly         `xml:"frameworkAssemblies>frameworkAssembly"`
	References          *ReferencesElement          `xml:"references"`
	ContentFiles        []ContentFilesEntry         `xml:"contentFiles>files"`
	PackageTypes        []PackageType               `xml:"packageTypes>packageType"`
	Repository          *RepositoryMetadata         `xml:"repository"`
}

// LicenseMetadata represents license information.
type LicenseMetadata struct {
	Type    string `xml:"type,attr"`    // "expression" or "file"
	Version string `xml:"version,attr"` // SPDX license version
	Text    string `xml:",chardata"`    // License expression or file path
}

// DependenciesElement represents the dependencies container.
type DependenciesElement struct {
	Groups []DependencyGroup `xml:"group"`
	// Legacy: dependencies without groups (applies to all frameworks)
	Dependencies []Dependency `xml:"dependency"`
}

// DependencyGroup represents dependencies for a specific framework.
type DependencyGroup struct {
	TargetFramework string       `xml:"targetFramework,attr"`
	Dependencies    []Dependency `xml:"dependency"`
}

// Dependency represents a package dependency.
type Dependency struct {
	ID      string `xml:"id,attr"`
	Version string `xml:"version,attr"` // Version range string
	Include string `xml:"include,attr"` // Asset include filter
	Exclude string `xml:"exclude,attr"` // Asset exclude filter
}

// FrameworkReferencesElement represents framework references container.
type FrameworkReferencesElement struct {
	Groups []FrameworkReferenceGroup `xml:"group"`
}

// FrameworkReferenceGroup represents framework references for a TFM.
type FrameworkReferenceGroup struct {
	TargetFramework string               `xml:"targetFramework,attr"`
	References      []FrameworkReference `xml:"frameworkReference"`
}

// FrameworkReference represents a reference to a framework assembly.
type FrameworkReference struct {
	Name string `xml:"name,attr"`
}

// FrameworkAssembly represents a legacy framework assembly reference.
type FrameworkAssembly struct {
	AssemblyName    string `xml:"assemblyName,attr"`
	TargetFramework string `xml:"targetFramework,attr"`
}

// ReferencesElement represents package assembly references.
type ReferencesElement struct {
	Groups []ReferenceGroup `xml:"group"`
}

// ReferenceGroup represents references for a specific framework.
type ReferenceGroup struct {
	TargetFramework string      `xml:"targetFramework,attr"`
	References      []Reference `xml:"reference"`
}

// Reference represents a reference to an assembly in the package.
type Reference struct {
	File string `xml:"file,attr"`
}

// ContentFilesEntry represents content files metadata.
type ContentFilesEntry struct {
	Include      string `xml:"include,attr"`
	Exclude      string `xml:"exclude,attr"`
	BuildAction  string `xml:"buildAction,attr"`
	CopyToOutput string `xml:"copyToOutput,attr"`
	Flatten      string `xml:"flatten,attr"`
}

// PackageType represents the type of package.
type PackageType struct {
	Name    string `xml:"name,attr"`
	Version string `xml:"version,attr"`
}

// RepositoryMetadata represents repository information.
type RepositoryMetadata struct {
	Type   string `xml:"type,attr"`
	URL    string `xml:"url,attr"`
	Branch string `xml:"branch,attr"`
	Commit string `xml:"commit,attr"`
}

// NuspecFile represents a file entry in the nuspec.
type NuspecFile struct {
	Source  string `xml:"src,attr"`
	Target  string `xml:"target,attr"`
	Exclude string `xml:"exclude,attr"`
}

// ParseNuspec parses a .nuspec XML document.
func ParseNuspec(r io.Reader) (*Nuspec, error) {
	decoder := xml.NewDecoder(r)

	var nuspec Nuspec
	if err := decoder.Decode(&nuspec); err != nil {
		return nil, fmt.Errorf("parse nuspec: %w", err)
	}

	return &nuspec, nil
}

// GetParsedIdentity returns the package identity from nuspec.
func (n *Nuspec) GetParsedIdentity() (*PackageIdentity, error) {
	ver, err := version.Parse(n.Metadata.Version)
	if err != nil {
		return nil, fmt.Errorf("parse version: %w", err)
	}

	return &PackageIdentity{
		ID:      n.Metadata.ID,
		Version: ver,
	}, nil
}

// GetAuthors returns the list of authors.
func (n *Nuspec) GetAuthors() []string {
	if n.Metadata.Authors == "" {
		return []string{}
	}

	// Authors are comma-separated
	authors := strings.Split(n.Metadata.Authors, ",")
	for i := range authors {
		authors[i] = strings.TrimSpace(authors[i])
	}

	return authors
}

// GetOwners returns the list of owners.
func (n *Nuspec) GetOwners() []string {
	if n.Metadata.Owners == "" {
		return []string{}
	}

	owners := strings.Split(n.Metadata.Owners, ",")
	for i := range owners {
		owners[i] = strings.TrimSpace(owners[i])
	}

	return owners
}

// GetTags returns the list of tags.
func (n *Nuspec) GetTags() []string {
	if n.Metadata.Tags == "" {
		return []string{}
	}

	// Tags are space-separated
	tags := strings.Fields(n.Metadata.Tags)
	return tags
}

// GetDependencyGroups returns all dependency groups with parsed frameworks.
func (n *Nuspec) GetDependencyGroups() ([]ParsedDependencyGroup, error) {
	if n.Metadata.Dependencies == nil {
		return []ParsedDependencyGroup{}, nil
	}

	var groups []ParsedDependencyGroup

	// Handle legacy dependencies (no groups)
	if len(n.Metadata.Dependencies.Dependencies) > 0 {
		// Dependencies without group apply to all frameworks
		anyFramework := frameworks.AnyFramework

		deps, err := parseDependencies(n.Metadata.Dependencies.Dependencies)
		if err != nil {
			return nil, err
		}

		groups = append(groups, ParsedDependencyGroup{
			TargetFramework: &anyFramework,
			Dependencies:    deps,
		})
	}

	// Handle grouped dependencies
	for _, group := range n.Metadata.Dependencies.Groups {
		var targetFramework *frameworks.NuGetFramework

		if group.TargetFramework != "" {
			fw, err := frameworks.ParseFramework(group.TargetFramework)
			if err != nil {
				return nil, fmt.Errorf("parse target framework %q: %w", group.TargetFramework, err)
			}
			targetFramework = fw
		} else {
			// Empty target framework means "any"
			anyFramework := frameworks.AnyFramework
			targetFramework = &anyFramework
		}

		deps, err := parseDependencies(group.Dependencies)
		if err != nil {
			return nil, err
		}

		groups = append(groups, ParsedDependencyGroup{
			TargetFramework: targetFramework,
			Dependencies:    deps,
		})
	}

	return groups, nil
}

// ParsedDependencyGroup represents a dependency group with parsed framework.
type ParsedDependencyGroup struct {
	TargetFramework *frameworks.NuGetFramework
	Dependencies    []ParsedDependency
}

// ToPackageDependencyGroup converts a ParsedDependencyGroup to PackageDependencyGroup.
func (g *ParsedDependencyGroup) ToPackageDependencyGroup() PackageDependencyGroup {
	deps := make([]PackageDependency, len(g.Dependencies))
	for i, dep := range g.Dependencies {
		deps[i] = dep.ToPackageDependency()
	}

	return PackageDependencyGroup{
		TargetFramework: g.TargetFramework,
		Dependencies:    deps,
	}
}

// ParsedDependency represents a dependency with parsed version range.
type ParsedDependency struct {
	ID           string
	VersionRange *version.VersionRange
	Include      []string // Asset include patterns
	Exclude      []string // Asset exclude patterns
}

// ToPackageDependency converts a ParsedDependency to PackageDependency.
func (d *ParsedDependency) ToPackageDependency() PackageDependency {
	return PackageDependency{
		ID:           d.ID,
		VersionRange: d.VersionRange,
		Include:      d.Include,
		Exclude:      d.Exclude,
	}
}

func parseDependencies(deps []Dependency) ([]ParsedDependency, error) {
	var parsed []ParsedDependency

	for _, dep := range deps {
		var versionRange *version.VersionRange

		if dep.Version != "" {
			vr, err := version.ParseVersionRange(dep.Version)
			if err != nil {
				return nil, fmt.Errorf("parse version range %q for %q: %w", dep.Version, dep.ID, err)
			}
			versionRange = vr
		}

		parsedDep := ParsedDependency{
			ID:           dep.ID,
			VersionRange: versionRange,
		}

		if dep.Include != "" {
			// Support both comma and semicolon as separators
			separator := ";"
			if strings.Contains(dep.Include, ",") {
				separator = ","
			}
			for part := range strings.SplitSeq(dep.Include, separator) {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					parsedDep.Include = append(parsedDep.Include, trimmed)
				}
			}
		}

		if dep.Exclude != "" {
			// Support both comma and semicolon as separators
			separator := ";"
			if strings.Contains(dep.Exclude, ",") {
				separator = ","
			}
			for part := range strings.SplitSeq(dep.Exclude, separator) {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					parsedDep.Exclude = append(parsedDep.Exclude, trimmed)
				}
			}
		}

		parsed = append(parsed, parsedDep)
	}

	return parsed, nil
}

// GetFrameworkReferenceGroups returns all framework reference groups.
func (n *Nuspec) GetFrameworkReferenceGroups() ([]ParsedFrameworkReferenceGroup, error) {
	if n.Metadata.FrameworkReferences == nil {
		return []ParsedFrameworkReferenceGroup{}, nil
	}

	var groups []ParsedFrameworkReferenceGroup

	for _, group := range n.Metadata.FrameworkReferences.Groups {
		fw, err := frameworks.ParseFramework(group.TargetFramework)
		if err != nil {
			return nil, fmt.Errorf("parse target framework %q: %w", group.TargetFramework, err)
		}

		var refs []string
		for _, ref := range group.References {
			refs = append(refs, ref.Name)
		}

		groups = append(groups, ParsedFrameworkReferenceGroup{
			TargetFramework: fw,
			References:      refs,
		})
	}

	return groups, nil
}

// ParsedFrameworkReferenceGroup represents framework references with parsed TFM.
type ParsedFrameworkReferenceGroup struct {
	TargetFramework *frameworks.NuGetFramework
	References      []string
}

// ToPackageFrameworkReferenceGroup converts to PackageFrameworkReferenceGroup.
func (g *ParsedFrameworkReferenceGroup) ToPackageFrameworkReferenceGroup() PackageFrameworkReferenceGroup {
	return PackageFrameworkReferenceGroup{
		TargetFramework: g.TargetFramework,
		References:      g.References,
	}
}
