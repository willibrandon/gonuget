// Package project provides abstractions for Directory.Packages.props files used in Central Package Management.
package project

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

// DirectoryPackagesProps represents a Directory.Packages.props file.
type DirectoryPackagesProps struct {
	Path     string
	Root     *DirectoryPackagesRootElement
	modified bool
}

// DirectoryPackagesRootElement represents the root <Project> element.
type DirectoryPackagesRootElement struct {
	XMLName    xml.Name              `xml:"Project"`
	Properties []PropertyGroup       `xml:"PropertyGroup"`
	ItemGroups []PackageVersionGroup `xml:"ItemGroup"`
}

// PackageVersionGroup represents an <ItemGroup> containing PackageVersion elements.
type PackageVersionGroup struct {
	XMLName         xml.Name         `xml:"ItemGroup"`
	PackageVersions []PackageVersion `xml:"PackageVersion"`
}

// PackageVersion represents a <PackageVersion> element in Directory.Packages.props.
type PackageVersion struct {
	XMLName xml.Name `xml:"PackageVersion"`
	Include string   `xml:"Include,attr"`
	Version string   `xml:"Version,attr"`
}

// LoadDirectoryPackagesProps loads a Directory.Packages.props file from disk.
func LoadDirectoryPackagesProps(path string) (*DirectoryPackagesProps, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read Directory.Packages.props: %w", err)
	}

	var root DirectoryPackagesRootElement
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to parse Directory.Packages.props: %w", err)
	}

	return &DirectoryPackagesProps{
		Path:     path,
		Root:     &root,
		modified: false,
	}, nil
}

// AddOrUpdatePackageVersion adds a new PackageVersion or updates an existing one.
// Returns true if an existing PackageVersion was updated, false if a new one was added.
func (dp *DirectoryPackagesProps) AddOrUpdatePackageVersion(packageID, version string) (bool, error) {
	// Find existing PackageVersion (case-insensitive)
	for i := range dp.Root.ItemGroups {
		ig := &dp.Root.ItemGroups[i]
		for j := range ig.PackageVersions {
			pv := &ig.PackageVersions[j]
			if strings.EqualFold(pv.Include, packageID) {
				pv.Version = version
				dp.modified = true
				return true, nil
			}
		}
	}

	// Not found, add new PackageVersion
	itemGroup := dp.findOrCreateItemGroup()
	itemGroup.PackageVersions = append(itemGroup.PackageVersions, PackageVersion{
		Include: packageID,
		Version: version,
	})

	dp.modified = true
	return false, nil
}

// GetPackageVersion returns the version for a package ID, or empty string if not found.
func (dp *DirectoryPackagesProps) GetPackageVersion(packageID string) string {
	for _, ig := range dp.Root.ItemGroups {
		for _, pv := range ig.PackageVersions {
			if strings.EqualFold(pv.Include, packageID) {
				return pv.Version
			}
		}
	}
	return ""
}

// RemovePackageVersion removes a PackageVersion by package ID.
// Returns true if a PackageVersion was removed, false if not found.
func (dp *DirectoryPackagesProps) RemovePackageVersion(packageID string) bool {
	for i := range dp.Root.ItemGroups {
		ig := &dp.Root.ItemGroups[i]
		for j := range ig.PackageVersions {
			if strings.EqualFold(ig.PackageVersions[j].Include, packageID) {
				// Remove PackageVersion
				ig.PackageVersions = append(ig.PackageVersions[:j], ig.PackageVersions[j+1:]...)
				dp.modified = true
				return true
			}
		}
	}
	return false
}

// Save saves the Directory.Packages.props file to disk.
func (dp *DirectoryPackagesProps) Save() error {
	if !dp.modified {
		return nil
	}

	// Marshal XML with indentation
	output, err := xml.MarshalIndent(dp.Root, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Directory.Packages.props: %w", err)
	}

	// Open file for writing
	file, err := os.Create(dp.Path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Write UTF-8 BOM (required for .NET compatibility)
	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}

	// Write XML declaration
	if _, err := file.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"); err != nil {
		return err
	}

	// Write XML
	if _, err := file.Write(output); err != nil {
		return err
	}

	dp.modified = false
	return nil
}

// findOrCreateItemGroup finds the first ItemGroup or creates a new one.
func (dp *DirectoryPackagesProps) findOrCreateItemGroup() *PackageVersionGroup {
	// Find first existing ItemGroup
	if len(dp.Root.ItemGroups) > 0 {
		return &dp.Root.ItemGroups[0]
	}

	// Create new ItemGroup
	dp.Root.ItemGroups = append(dp.Root.ItemGroups, PackageVersionGroup{
		PackageVersions: []PackageVersion{},
	})
	return &dp.Root.ItemGroups[0]
}
