// Package project provides abstractions for loading, parsing, and saving .NET project files (.csproj, .fsproj, .vbproj).
package project

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Project represents a .NET project file.
type Project struct {
	Path             string
	Root             *RootElement
	modified         bool
	TargetFramework  string   // Single target framework (e.g., "net8.0")
	TargetFrameworks []string // Multiple target frameworks (e.g., ["net6.0", "net7.0", "net8.0"])
}

// LoadProject loads and parses a project file from the given path.
func LoadProject(path string) (*Project, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read project file: %w", err)
	}

	// Parse XML
	var root RootElement
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to parse project XML: %w", err)
	}

	// Store raw XML for formatting preservation
	root.RawXML = data

	proj := &Project{
		Path:     path,
		Root:     &root,
		modified: false,
	}

	// Extract target framework(s)
	for _, pg := range root.PropertyGroup {
		if pg.TargetFramework != "" {
			proj.TargetFramework = pg.TargetFramework
		}
		if pg.TargetFrameworks != "" {
			// Parse semicolon-separated list
			proj.TargetFrameworks = strings.Split(pg.TargetFrameworks, ";")
		}
	}

	return proj, nil
}

// Save saves the project file with UTF-8 BOM and formatting preservation.
func (p *Project) Save() error {
	if !p.modified {
		return nil
	}

	// Marshal with indentation
	output, err := xml.MarshalIndent(p.Root, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	// Open file for writing
	file, err := os.Create(p.Path)
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

	// Write project XML
	if _, err := file.Write(output); err != nil {
		return err
	}

	p.modified = false
	return nil
}

// AddOrUpdatePackageReference adds a new PackageReference or updates an existing one.
// Returns true if an existing reference was updated, false if a new one was added.
func (p *Project) AddOrUpdatePackageReference(id, version string, frameworks []string) (bool, error) {
	// M2.1: Only support unconditional references (no framework-specific)
	if len(frameworks) > 0 {
		return false, fmt.Errorf("framework-specific references not supported in M2.1 (use M2.2 Chunk 14)")
	}

	// Find or create ItemGroup
	var targetItemGroup *ItemGroup
	var existingRef *PackageReference

	// Search for existing reference
	for i := range p.Root.ItemGroups {
		ig := &p.Root.ItemGroups[i]
		// M2.1: Only modify unconditional ItemGroups
		if ig.Condition != "" {
			continue
		}

		for j := range ig.PackageReferences {
			ref := &ig.PackageReferences[j]
			if strings.EqualFold(ref.Include, id) {
				existingRef = ref
				targetItemGroup = ig
				break
			}
		}
		if existingRef != nil {
			break
		}

		// Remember first unconditional ItemGroup for adding new references
		if targetItemGroup == nil && ig.Condition == "" {
			targetItemGroup = ig
		}
	}

	// Update existing reference
	if existingRef != nil {
		existingRef.Version = version
		p.modified = true
		return true, nil
	}

	// Add new reference
	newRef := PackageReference{
		Include: id,
		Version: version,
	}

	if targetItemGroup != nil {
		// Add to existing ItemGroup
		targetItemGroup.PackageReferences = append(targetItemGroup.PackageReferences, newRef)
	} else {
		// Create new ItemGroup
		p.Root.ItemGroups = append(p.Root.ItemGroups, ItemGroup{
			PackageReferences: []PackageReference{newRef},
		})
	}

	p.modified = true
	return false, nil
}

// RemovePackageReference removes a PackageReference by package ID.
// Returns true if a reference was removed, false if not found.
func (p *Project) RemovePackageReference(id string) bool {
	for i := range p.Root.ItemGroups {
		ig := &p.Root.ItemGroups[i]
		for j := range ig.PackageReferences {
			if strings.EqualFold(ig.PackageReferences[j].Include, id) {
				// Remove reference
				ig.PackageReferences = append(ig.PackageReferences[:j], ig.PackageReferences[j+1:]...)
				p.modified = true
				return true
			}
		}
	}
	return false
}

// GetPackageReferences returns all PackageReference elements in the project.
func (p *Project) GetPackageReferences() []PackageReference {
	var refs []PackageReference
	for _, ig := range p.Root.ItemGroups {
		refs = append(refs, ig.PackageReferences...)
	}
	return refs
}

// IsCentralPackageManagementEnabled checks if Central Package Management (CPM) is enabled.
// M2.1: Simple detection - check for Directory.Packages.props existence.
// M2.2: Full CPM support with Directory.Packages.props manipulation (Chunks 11-13).
func (p *Project) IsCentralPackageManagementEnabled() bool {
	dir := filepath.Dir(p.Path)
	cpmPath := filepath.Join(dir, "Directory.Packages.props")
	_, err := os.Stat(cpmPath)
	return err == nil
}

// IsSDKStyle returns true if this is an SDK-style project.
func (p *Project) IsSDKStyle() bool {
	return p.Root.Sdk != ""
}

// FindProjectFile finds a single .csproj, .fsproj, or .vbproj file in the directory.
// Returns error if 0 or >1 project files are found.
func FindProjectFile(dir string) (string, error) {
	// Check for .csproj
	allMatches, err := filepath.Glob(filepath.Join(dir, "*.csproj"))
	if err != nil {
		return "", err
	}

	// Check for .fsproj
	fsprojMatches, err := filepath.Glob(filepath.Join(dir, "*.fsproj"))
	if err != nil {
		return "", err
	}

	// Check for .vbproj
	vbprojMatches, err := filepath.Glob(filepath.Join(dir, "*.vbproj"))
	if err != nil {
		return "", err
	}

	// Combine all matches
	allMatches = append(allMatches, fsprojMatches...)
	allMatches = append(allMatches, vbprojMatches...)

	if len(allMatches) == 0 {
		return "", fmt.Errorf("no project file found in directory: %s", dir)
	}

	if len(allMatches) > 1 {
		return "", fmt.Errorf("multiple project files found in directory: %s. Specify which project to use", dir)
	}

	return allMatches[0], nil
}
