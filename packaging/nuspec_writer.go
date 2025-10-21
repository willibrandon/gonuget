package packaging

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// Nuspec schema namespaces
// Reference: ManifestSchemaUtility.cs in NuGet.Client
const (
	// NuspecNamespaceV1 - Baseline schema (2010/07)
	NuspecNamespaceV1 = "http://schemas.microsoft.com/packaging/2010/07/nuspec.xsd"

	// NuspecNamespaceV2 - Added copyrights, references and release notes (2011/08)
	NuspecNamespaceV2 = "http://schemas.microsoft.com/packaging/2011/08/nuspec.xsd"

	// NuspecNamespaceV3 - Used if the version is a semantic version (2011/10)
	NuspecNamespaceV3 = "http://schemas.microsoft.com/packaging/2011/10/nuspec.xsd"

	// NuspecNamespaceV4 - Added 'targetFramework' attribute for 'dependency' elements (2012/06)
	// Allow framework folders under 'content' and 'tools' folders
	NuspecNamespaceV4 = "http://schemas.microsoft.com/packaging/2012/06/nuspec.xsd"

	// NuspecNamespaceV5 - Added 'targetFramework' attribute for 'references' elements (2013/01)
	// Added 'minClientVersion' attribute
	NuspecNamespaceV5 = "http://schemas.microsoft.com/packaging/2013/01/nuspec.xsd"

	// NuspecNamespaceV6 - Allows XDT transformation (2013/05) - most recent
	NuspecNamespaceV6 = "http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd"
)

// GenerateNuspecXML generates nuspec XML from package metadata.
func GenerateNuspecXML(metadata PackageMetadata) ([]byte, error) {
	// Determine schema version based on features used
	namespace := determineNuspecNamespace(metadata)

	// Build nuspec structure
	nuspec := buildNuspecStructure(metadata, namespace)

	// Encode to XML
	var buf strings.Builder
	buf.WriteString(xml.Header)

	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")

	if err := encoder.Encode(nuspec); err != nil {
		return nil, fmt.Errorf("encode nuspec: %w", err)
	}

	return []byte(buf.String()), nil
}

// determineNuspecNamespace inspects metadata to determine minimum required schema version
// Reference: ManifestVersionUtility.GetManifestVersion in NuGet.Client
func determineNuspecNamespace(metadata PackageMetadata) string {
	// Check for features requiring newer schema versions (check newest first)

	// V5 (2013/01): References with target frameworks
	if hasReferencesWithTargetFramework(metadata) {
		return NuspecNamespaceV5
	}

	// V4 (2012/06): Dependencies with target frameworks
	if hasDependenciesWithTargetFramework(metadata) {
		return NuspecNamespaceV4
	}

	// V3 (2011/10): Prerelease/semantic versions
	if metadata.Version != nil && metadata.Version.IsPrerelease() {
		return NuspecNamespaceV3
	}

	// V6 (2013/05): Default to most recent for best compatibility
	return NuspecNamespaceV6
}

func hasReferencesWithTargetFramework(metadata PackageMetadata) bool {
	// Check if any reference groups have specific target frameworks
	for _, group := range metadata.FrameworkReferenceGroups {
		if group.TargetFramework != nil && !group.TargetFramework.IsAny() {
			return true
		}
	}
	return false
}

func hasDependenciesWithTargetFramework(metadata PackageMetadata) bool {
	// Check if any dependency groups have specific target frameworks
	for _, group := range metadata.DependencyGroups {
		if group.TargetFramework != nil && !group.TargetFramework.IsAny() {
			return true
		}
	}
	return false
}

func buildNuspecStructure(metadata PackageMetadata, namespace string) *Nuspec {
	nuspec := &Nuspec{
		XMLName: xml.Name{
			Local: "package",
		},
		Xmlns: namespace,
		Metadata: NuspecMetadata{
			ID:          metadata.ID,
			Version:     metadata.Version.String(),
			Description: metadata.Description,
		},
	}

	// Authors (required)
	if len(metadata.Authors) > 0 {
		nuspec.Metadata.Authors = strings.Join(metadata.Authors, ", ")
	}

	// Optional fields
	if metadata.Title != "" {
		nuspec.Metadata.Title = metadata.Title
	}

	if len(metadata.Owners) > 0 {
		nuspec.Metadata.Owners = strings.Join(metadata.Owners, ", ")
	}

	if metadata.ProjectURL != nil {
		nuspec.Metadata.ProjectURL = metadata.ProjectURL.String()
	}

	if metadata.IconURL != nil {
		nuspec.Metadata.IconURL = metadata.IconURL.String()
	}

	if metadata.Icon != "" {
		nuspec.Metadata.Icon = metadata.Icon
	}

	if metadata.LicenseURL != nil {
		nuspec.Metadata.LicenseURL = metadata.LicenseURL.String()
	}

	if metadata.LicenseMetadata != nil {
		nuspec.Metadata.License = &LicenseMetadata{
			Type:    metadata.LicenseMetadata.Type,
			Version: metadata.LicenseMetadata.Version,
			Text:    metadata.LicenseMetadata.Text,
		}
	}

	nuspec.Metadata.RequireLicenseAcceptance = metadata.RequireLicenseAcceptance
	nuspec.Metadata.DevelopmentDependency = metadata.DevelopmentDependency

	if metadata.Summary != "" {
		nuspec.Metadata.Summary = metadata.Summary
	}

	if metadata.ReleaseNotes != "" {
		nuspec.Metadata.ReleaseNotes = metadata.ReleaseNotes
	}

	if metadata.Copyright != "" {
		nuspec.Metadata.Copyright = metadata.Copyright
	}

	if metadata.Language != "" {
		nuspec.Metadata.Language = metadata.Language
	}

	if len(metadata.Tags) > 0 {
		nuspec.Metadata.Tags = strings.Join(metadata.Tags, " ")
	}

	nuspec.Metadata.Serviceable = metadata.Serviceable

	if metadata.Readme != "" {
		nuspec.Metadata.Readme = metadata.Readme
	}

	if metadata.MinClientVersion != nil {
		nuspec.Metadata.MinClientVersion = metadata.MinClientVersion.String()
	}

	// Dependencies
	if len(metadata.DependencyGroups) > 0 {
		nuspec.Metadata.Dependencies = &DependenciesElement{}

		for _, group := range metadata.DependencyGroups {
			depGroup := DependencyGroup{}

			if group.TargetFramework != nil && !group.TargetFramework.IsAny() {
				depGroup.TargetFramework = group.TargetFramework.String()
			}

			for _, dep := range group.Dependencies {
				dependency := Dependency{
					ID: dep.ID,
				}

				if dep.VersionRange != nil {
					dependency.Version = dep.VersionRange.String()
				}

				if len(dep.Include) > 0 {
					dependency.Include = strings.Join(dep.Include, ";")
				}

				if len(dep.Exclude) > 0 {
					dependency.Exclude = strings.Join(dep.Exclude, ";")
				}

				depGroup.Dependencies = append(depGroup.Dependencies, dependency)
			}

			nuspec.Metadata.Dependencies.Groups = append(nuspec.Metadata.Dependencies.Groups, depGroup)
		}
	}

	// Framework references
	if len(metadata.FrameworkReferenceGroups) > 0 {
		nuspec.Metadata.FrameworkReferences = &FrameworkReferencesElement{}

		for _, group := range metadata.FrameworkReferenceGroups {
			fwRefGroup := FrameworkReferenceGroup{
				TargetFramework: group.TargetFramework.String(),
			}

			for _, ref := range group.References {
				fwRefGroup.References = append(fwRefGroup.References, FrameworkReference{
					Name: ref,
				})
			}

			nuspec.Metadata.FrameworkReferences.Groups = append(nuspec.Metadata.FrameworkReferences.Groups, fwRefGroup)
		}
	}

	// Package types
	if len(metadata.PackageTypes) > 0 {
		for _, pt := range metadata.PackageTypes {
			pkgType := PackageType{
				Name: pt.Name,
			}

			if pt.Version != nil {
				pkgType.Version = pt.Version.String()
			}

			nuspec.Metadata.PackageTypes = append(nuspec.Metadata.PackageTypes, pkgType)
		}
	}

	// Repository metadata
	if metadata.Repository != nil {
		nuspec.Metadata.Repository = &RepositoryMetadata{
			Type:   metadata.Repository.Type,
			URL:    metadata.Repository.URL,
			Branch: metadata.Repository.Branch,
			Commit: metadata.Repository.Commit,
		}
	}

	return nuspec
}
