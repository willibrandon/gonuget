package project

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectRootElement_Marshal_SDKStyle(t *testing.T) {
	root := RootElement{
		Sdk: "Microsoft.NET.Sdk",
		PropertyGroup: []PropertyGroup{
			{TargetFramework: "net8.0"},
		},
	}

	data, err := xml.MarshalIndent(root, "", "  ")
	require.NoError(t, err)

	expected := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	assert.Equal(t, expected, string(data))
}

func TestProjectRootElement_Marshal_WithPackageReferences(t *testing.T) {
	root := RootElement{
		Sdk: "Microsoft.NET.Sdk",
		PropertyGroup: []PropertyGroup{
			{TargetFramework: "net8.0"},
		},
		ItemGroups: []ItemGroup{
			{
				PackageReferences: []PackageReference{
					{Include: "Newtonsoft.Json", Version: "13.0.3"},
				},
			},
		},
	}

	data, err := xml.MarshalIndent(root, "", "  ")
	require.NoError(t, err)

	expected := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3"></PackageReference>
  </ItemGroup>
</Project>`

	assert.Equal(t, expected, string(data))
}

func TestProjectRootElement_Unmarshal_SDKStyle(t *testing.T) {
	xmlData := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

	var root RootElement
	err := xml.Unmarshal([]byte(xmlData), &root)
	require.NoError(t, err)

	assert.Equal(t, "Microsoft.NET.Sdk", root.Sdk)
	require.Len(t, root.PropertyGroup, 1)
	assert.Equal(t, "net8.0", root.PropertyGroup[0].TargetFramework)
}

func TestProjectRootElement_Unmarshal_WithPackageReferences(t *testing.T) {
	xmlData := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageReference Include="System.Text.Json" Version="8.0.0" />
  </ItemGroup>
</Project>`

	var root RootElement
	err := xml.Unmarshal([]byte(xmlData), &root)
	require.NoError(t, err)

	require.Len(t, root.ItemGroups, 1)
	require.Len(t, root.ItemGroups[0].PackageReferences, 2)
	assert.Equal(t, "Newtonsoft.Json", root.ItemGroups[0].PackageReferences[0].Include)
	assert.Equal(t, "13.0.3", root.ItemGroups[0].PackageReferences[0].Version)
	assert.Equal(t, "System.Text.Json", root.ItemGroups[0].PackageReferences[1].Include)
	assert.Equal(t, "8.0.0", root.ItemGroups[0].PackageReferences[1].Version)
}

func TestProjectRootElement_Unmarshal_MultiTFM(t *testing.T) {
	xmlData := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>net6.0;net7.0;net8.0</TargetFrameworks>
  </PropertyGroup>
</Project>`

	var root RootElement
	err := xml.Unmarshal([]byte(xmlData), &root)
	require.NoError(t, err)

	require.Len(t, root.PropertyGroup, 1)
	assert.Equal(t, "net6.0;net7.0;net8.0", root.PropertyGroup[0].TargetFrameworks)
}

func TestProjectRootElement_Unmarshal_ConditionalItemGroup(t *testing.T) {
	xmlData := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup Condition="'$(TargetFramework)' == 'net8.0'">
    <PackageReference Include="System.Drawing.Common" Version="8.0.0" />
  </ItemGroup>
</Project>`

	var root RootElement
	err := xml.Unmarshal([]byte(xmlData), &root)
	require.NoError(t, err)

	require.Len(t, root.ItemGroups, 1)
	assert.Equal(t, "'$(TargetFramework)' == 'net8.0'", root.ItemGroups[0].Condition)
	require.Len(t, root.ItemGroups[0].PackageReferences, 1)
	assert.Equal(t, "System.Drawing.Common", root.ItemGroups[0].PackageReferences[0].Include)
}

func TestPackageReference_Marshal_WithAttributes(t *testing.T) {
	ref := PackageReference{
		Include:       "Newtonsoft.Json",
		Version:       "13.0.3",
		PrivateAssets: "all",
	}

	data, err := xml.Marshal(ref)
	require.NoError(t, err)

	assert.Contains(t, string(data), `Include="Newtonsoft.Json"`)
	assert.Contains(t, string(data), `Version="13.0.3"`)
	assert.Contains(t, string(data), `PrivateAssets="all"`)
}

func TestPackageReference_Unmarshal_AllAttributes(t *testing.T) {
	xmlData := `<PackageReference Include="Newtonsoft.Json" Version="13.0.3" PrivateAssets="all" IncludeAssets="compile;runtime" ExcludeAssets="build" GeneratePathProperty="true" />`

	var ref PackageReference
	err := xml.Unmarshal([]byte(xmlData), &ref)
	require.NoError(t, err)

	assert.Equal(t, "Newtonsoft.Json", ref.Include)
	assert.Equal(t, "13.0.3", ref.Version)
	assert.Equal(t, "all", ref.PrivateAssets)
	assert.Equal(t, "compile;runtime", ref.IncludeAssets)
	assert.Equal(t, "build", ref.ExcludeAssets)
	assert.Equal(t, "true", ref.GeneratePathProperty)
}

func TestPropertyGroup_Marshal_Multiple(t *testing.T) {
	root := RootElement{
		Sdk: "Microsoft.NET.Sdk",
		PropertyGroup: []PropertyGroup{
			{TargetFramework: "net8.0", OutputType: "Exe"},
			{Condition: "'$(Configuration)' == 'Release'", RootNamespace: "MyApp"},
		},
	}

	data, err := xml.MarshalIndent(root, "", "  ")
	require.NoError(t, err)

	assert.Contains(t, string(data), `<TargetFramework>net8.0</TargetFramework>`)
	assert.Contains(t, string(data), `<OutputType>Exe</OutputType>`)
	assert.Contains(t, string(data), `Condition="&#39;$(Configuration)&#39; == &#39;Release&#39;"`)
	assert.Contains(t, string(data), `<RootNamespace>MyApp</RootNamespace>`)
}

func TestItemGroup_Marshal_Mixed(t *testing.T) {
	ig := ItemGroup{
		PackageReferences: []PackageReference{
			{Include: "Newtonsoft.Json", Version: "13.0.3"},
		},
		ProjectReferences: []Reference{
			{Include: "../OtherProject/OtherProject.csproj"},
		},
	}

	data, err := xml.MarshalIndent(ig, "", "  ")
	require.NoError(t, err)

	assert.Contains(t, string(data), `<PackageReference Include="Newtonsoft.Json"`)
	assert.Contains(t, string(data), `<ProjectReference Include="../OtherProject/OtherProject.csproj"`)
}

func TestProjectRootElement_Unmarshal_Legacy(t *testing.T) {
	xmlData := `<Project ToolsVersion="15.0" xmlns="http://schemas.microsoft.com/developer/msbuild/2003">
  <PropertyGroup>
    <TargetFramework>net48</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <Reference Include="System" />
  </ItemGroup>
</Project>`

	var root RootElement
	err := xml.Unmarshal([]byte(xmlData), &root)
	require.NoError(t, err)

	// Legacy projects don't have Sdk attribute
	assert.Equal(t, "", root.Sdk)
	require.Len(t, root.ItemGroups, 1)
	require.Len(t, root.ItemGroups[0].References, 1)
	assert.Equal(t, "System", root.ItemGroups[0].References[0].Include)
}
