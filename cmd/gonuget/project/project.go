// Package project provides abstractions for loading, parsing, and saving .NET project files (.csproj, .fsproj, .vbproj).
package project

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/willibrandon/gonuget/frameworks"
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

	// Do NOT resolve symlinks - dotnet preserves the user-provided path
	// On macOS, /tmp is a symlink to /private/tmp, but dotnet shows /tmp in output
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
// Parameters:
//   - id: Package ID
//   - version: Package version (can be empty for CPM)
//   - frameworks: Target frameworks for conditional references (pass nil/empty for unconditional)
//
// Behavior matches dotnet CLI:
//   - If frameworks is empty/nil: Adds unconditional reference
//   - If frameworks is non-empty: Creates conditional ItemGroup(s) with '$(TargetFramework)' == 'tfm' condition
//
// Returns true if an existing reference was updated, false if a new one was added.
func (p *Project) AddOrUpdatePackageReference(id, version string, frameworks []string) (bool, error) {
	if id == "" {
		return false, fmt.Errorf("package ID cannot be empty")
	}

	// Validate framework TFMs if specified
	for _, fw := range frameworks {
		if err := validateFrameworkCompatibility(fw); err != nil {
			return false, fmt.Errorf("invalid framework %s: %w", fw, err)
		}
	}

	// If frameworks specified, create conditional ItemGroups (one per framework)
	if len(frameworks) > 0 {
		return p.addOrUpdateConditionalPackageReference(id, version, frameworks)
	}

	// Otherwise, add unconditional reference
	return p.addOrUpdateUnconditionalPackageReference(id, version)
}

// addOrUpdateUnconditionalPackageReference adds/updates a package reference without framework conditions.
func (p *Project) addOrUpdateUnconditionalPackageReference(id, version string) (bool, error) {
	// Find existing PackageReference in unconditional ItemGroup
	for i := range p.Root.ItemGroups {
		ig := &p.Root.ItemGroups[i]

		// Skip conditional ItemGroups
		if ig.Condition != "" {
			continue
		}

		for j := range ig.PackageReferences {
			pr := &ig.PackageReferences[j]
			if strings.EqualFold(pr.Include, id) {
				// Update existing reference
				if version != "" {
					pr.Version = version
				}
				p.modified = true
				return true, nil
			}
		}
	}

	// Not found, add new PackageReference to unconditional ItemGroup
	itemGroup := p.findOrCreateItemGroup("")
	itemGroup.PackageReferences = append(itemGroup.PackageReferences, PackageReference{
		Include: id,
		Version: version,
	})

	p.modified = true
	return false, nil
}

// addOrUpdateConditionalPackageReference adds/updates conditional package references (one per framework).
func (p *Project) addOrUpdateConditionalPackageReference(id, version string, frameworks []string) (bool, error) {
	updated := false

	for _, fw := range frameworks {
		condition := buildFrameworkCondition([]string{fw})
		normalizedCondition := normalizeCondition(condition)

		// Find existing PackageReference in matching conditional ItemGroup
		found := false
		for i := range p.Root.ItemGroups {
			ig := &p.Root.ItemGroups[i]

			// Match condition
			if normalizeCondition(ig.Condition) != normalizedCondition {
				continue
			}

			for j := range ig.PackageReferences {
				pr := &ig.PackageReferences[j]
				if strings.EqualFold(pr.Include, id) {
					// Update existing reference
					if version != "" {
						pr.Version = version
					}
					p.modified = true
					updated = true
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			// Not found, add new PackageReference to conditional ItemGroup
			itemGroup := p.findOrCreateItemGroup(condition)
			itemGroup.PackageReferences = append(itemGroup.PackageReferences, PackageReference{
				Include: id,
				Version: version,
			})
			p.modified = true
		}
	}

	return updated, nil
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
// Checks the ManagePackageVersionsCentrally property in the project file.
func (p *Project) IsCentralPackageManagementEnabled() bool {
	for _, pg := range p.Root.PropertyGroup {
		if strings.EqualFold(pg.ManagePackageVersionsCentrally, "true") {
			return true
		}
	}
	return false
}

// GetDirectoryPackagesPropsPath returns the path to Directory.Packages.props.
// It checks the DirectoryPackagesPropsPath property first, then walks up the directory tree.
func (p *Project) GetDirectoryPackagesPropsPath() string {
	dir := filepath.Dir(p.Path)

	// Check DirectoryPackagesPropsPath property
	for _, pg := range p.Root.PropertyGroup {
		if pg.DirectoryPackagesPropsPath != "" {
			// Resolve relative path
			if !filepath.IsAbs(pg.DirectoryPackagesPropsPath) {
				return filepath.Join(dir, pg.DirectoryPackagesPropsPath)
			}
			return pg.DirectoryPackagesPropsPath
		}
	}

	// Walk up directory tree looking for Directory.Packages.props
	current := dir
	for {
		propsPath := filepath.Join(current, "Directory.Packages.props")
		if _, err := os.Stat(propsPath); err == nil {
			return propsPath
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root, return default location next to project
			return filepath.Join(dir, "Directory.Packages.props")
		}
		current = parent
	}
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

// GetTargetFrameworks returns the list of target frameworks for the project.
// Returns single framework from TargetFramework or multiple from TargetFrameworks.
func (p *Project) GetTargetFrameworks() []string {
	// Check cached fields first
	if len(p.TargetFrameworks) > 0 {
		return p.TargetFrameworks
	}
	if p.TargetFramework != "" {
		return []string{p.TargetFramework}
	}

	// Parse from PropertyGroup (fallback if fields not populated during load)
	for _, pg := range p.Root.PropertyGroup {
		// TargetFrameworks (plural) - multiple frameworks
		if pg.TargetFrameworks != "" {
			return strings.Split(pg.TargetFrameworks, ";")
		}

		// TargetFramework (singular) - single framework
		if pg.TargetFramework != "" {
			return []string{pg.TargetFramework}
		}
	}
	return []string{}
}

// IsMultiTargeting returns true if the project targets multiple frameworks.
func (p *Project) IsMultiTargeting() bool {
	return len(p.GetTargetFrameworks()) > 1
}

// buildFrameworkCondition builds an MSBuild condition string for framework filtering.
// Returns empty string if frameworks is empty (unconditional).
// Returns "'$(TargetFramework)' == 'net8.0'" for single framework.
// Returns "'$(TargetFramework)' == 'net8.0' OR '$(TargetFramework)' == 'net48'" for multiple frameworks.
func buildFrameworkCondition(frameworks []string) string {
	if len(frameworks) == 0 {
		return ""
	}

	if len(frameworks) == 1 {
		return fmt.Sprintf("'$(TargetFramework)' == '%s'", frameworks[0])
	}

	// Multiple frameworks: OR conditions
	conditions := make([]string, len(frameworks))
	for i, fw := range frameworks {
		conditions[i] = fmt.Sprintf("'$(TargetFramework)' == '%s'", fw)
	}
	return strings.Join(conditions, " OR ")
}

// normalizeCondition normalizes an MSBuild condition for comparison.
// Handles whitespace variations and case insensitivity.
func normalizeCondition(condition string) string {
	// Trim and convert to lowercase
	normalized := strings.TrimSpace(strings.ToLower(condition))

	// Normalize whitespace: collapse multiple spaces to single space
	normalized = strings.Join(strings.Fields(normalized), " ")

	// Normalize quotes (both single and double quotes)
	normalized = strings.ReplaceAll(normalized, "\"", "'")

	return normalized
}

// validateFrameworkCompatibility validates a target framework moniker.
// Full package-to-framework compatibility is validated during restore.
func validateFrameworkCompatibility(framework string) error {
	// Parse and validate the target framework moniker
	_, err := frameworks.ParseFramework(framework)
	if err != nil {
		return fmt.Errorf("invalid target framework '%s': %w", framework, err)
	}

	// Design Note: Package-to-framework compatibility validation is intentionally deferred
	// to the restore phase. This matches dotnet behavior (see sdk/src/Cli/dotnet/Commands/Package/Add/PackageAddCommand.cs).
	//
	// Rationale:
	// 1. The restore.Restorer already downloads packages and parses nuspecs
	// 2. Restore uses frameworks.FrameworkReducer for compatibility checks
	// 3. Restore provides detailed error messages when incompatible
	// 4. Pre-validation would duplicate work and slow down add command
	// 5. Users get immediate feedback since restore runs by default (unless --no-restore)
	//
	// If --no-restore is used, users won't see compatibility errors until they run restore.
	// This is acceptable and matches dotnet behavior.
	return nil
}

// findOrCreateItemGroup finds an ItemGroup with the given condition or creates a new one.
func (p *Project) findOrCreateItemGroup(condition string) *ItemGroup {
	// Find existing ItemGroup with matching condition
	normalizedCondition := normalizeCondition(condition)
	for i := range p.Root.ItemGroups {
		ig := &p.Root.ItemGroups[i]
		if normalizeCondition(ig.Condition) == normalizedCondition {
			return ig
		}
	}

	// Create new ItemGroup with condition
	itemGroup := ItemGroup{
		Condition:         condition,
		PackageReferences: []PackageReference{},
	}
	p.Root.ItemGroups = append(p.Root.ItemGroups, itemGroup)
	return &p.Root.ItemGroups[len(p.Root.ItemGroups)-1]
}
