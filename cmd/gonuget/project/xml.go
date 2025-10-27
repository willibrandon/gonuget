package project

import "encoding/xml"

// RootElement represents the root <Project> element of a .csproj file.
type RootElement struct {
	XMLName       xml.Name        `xml:"Project"`
	Sdk           string          `xml:"Sdk,attr,omitempty"`
	PropertyGroup []PropertyGroup `xml:"PropertyGroup"`
	ItemGroups    []ItemGroup     `xml:"ItemGroup"`
	RawXML        []byte          `xml:"-"` // Store original XML for formatting preservation
}

// PropertyGroup represents a <PropertyGroup> element.
type PropertyGroup struct {
	Condition        string `xml:"Condition,attr,omitempty"`
	TargetFramework  string `xml:"TargetFramework,omitempty"`
	TargetFrameworks string `xml:"TargetFrameworks,omitempty"`
	OutputType       string `xml:"OutputType,omitempty"`
	RootNamespace    string `xml:"RootNamespace,omitempty"`
	AssemblyName     string `xml:"AssemblyName,omitempty"`
}

// ItemGroup represents an <ItemGroup> element containing package references or other items.
type ItemGroup struct {
	Condition         string              `xml:"Condition,attr,omitempty"`
	PackageReferences []PackageReference  `xml:"PackageReference,omitempty"`
	ProjectReferences []Reference         `xml:"ProjectReference,omitempty"`
	References        []AssemblyReference `xml:"Reference,omitempty"`
}

// PackageReference represents a <PackageReference> element.
type PackageReference struct {
	Include string `xml:"Include,attr"`
	Version string `xml:"Version,attr,omitempty"`
	// Additional attributes for advanced scenarios (M2.2)
	PrivateAssets        string `xml:"PrivateAssets,attr,omitempty"`
	IncludeAssets        string `xml:"IncludeAssets,attr,omitempty"`
	ExcludeAssets        string `xml:"ExcludeAssets,attr,omitempty"`
	GeneratePathProperty string `xml:"GeneratePathProperty,attr,omitempty"`
}

// Reference represents a <ProjectReference> element (references to other projects).
// Named Reference rather than ProjectReference to avoid package name stuttering.
type Reference struct {
	Include string `xml:"Include,attr"`
}

// AssemblyReference represents a <Reference> element (legacy .NET Framework assembly references).
// Named AssemblyReference to distinguish from Reference (project references) and avoid conflicts.
type AssemblyReference struct {
	Include string `xml:"Include,attr"`
}
