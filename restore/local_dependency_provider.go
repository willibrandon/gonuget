package restore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/willibrandon/gonuget/core/resolver"
	"github.com/willibrandon/gonuget/packaging"
	"github.com/willibrandon/gonuget/version"
)

// LocalDependencyProvider reads package dependencies from locally cached packages.
// Matches NuGet.Client LocalV3FindPackageByIdResource behavior.
// Reference: NuGet.Protocol/LocalRepositories/LocalV3FindPackageByIdResource.cs (lines 247-302, 430-468)
type LocalDependencyProvider struct {
	packagesFolder string
	resolver       *packaging.VersionFolderPathResolver
}

// NewLocalDependencyProvider creates a new local dependency provider.
func NewLocalDependencyProvider(packagesFolder string) *LocalDependencyProvider {
	return &LocalDependencyProvider{
		packagesFolder: packagesFolder,
		resolver:       packaging.NewVersionFolderPathResolver(packagesFolder, true),
	}
}

// GetDependencies reads package dependencies from locally cached .nuspec file.
// Returns ALL dependency groups (not filtered by framework), along with the resolved version.
// Returns nil if package is not found locally (caller should try remote).
//
// If packageVersion is a version range (e.g., "[2.10.0, )"), it will:
// 1. Enumerate all cached versions for packageID (matches GetVersionsCore lines 488-532)
// 2. Find the best matching version that satisfies the range (matches FindLibraryCoreAsync lines 274-277)
// 3. Return dependencies for that resolved version (matches GetDependencyInfoAsync lines 247-302)
//
// Returns: (dependency groups, resolved version string, error)
func (p *LocalDependencyProvider) GetDependencies(
	ctx context.Context,
	packageID string,
	packageVersion string,
) ([]resolver.DependencyGroup, string, error) {
	var resolvedVersion *version.NuGetVersion

	// Check if packageVersion is a version range (contains [ or ( or ,)
	// Ranges like "[2.10.0, )" or "(1.0.0, 2.0.0]"
	if strings.ContainsAny(packageVersion, "[](,") {
		// Parse version range
		versionRange, err := version.ParseVersionRange(packageVersion)
		if err != nil {
			return nil, "", nil // Invalid range, not cached
		}

		// Get all cached versions for this package
		// Matches GetVersionsCore (LocalV3FindPackageByIdResource.cs lines 488-532)
		allVersions, err := p.getAllVersions(packageID)
		if err != nil || len(allVersions) == 0 {
			return nil, "", nil // Package not cached or error
		}

		// Find best match for the range
		// Matches FindLibraryCoreAsync logic (SourceRepositoryDependencyProvider.cs line 277)
		bestMatch := versionRange.FindBestMatch(allVersions)
		if bestMatch == nil {
			return nil, "", nil // No cached version satisfies the range
		}

		resolvedVersion = bestMatch
	} else {
		// Exact version - parse it
		ver, err := version.Parse(packageVersion)
		if err != nil {
			return nil, "", nil // Invalid version, not cached
		}
		resolvedVersion = ver
	}

	// Check if resolved version exists locally (completion marker check)
	// Matches DoesVersionExist check in GetDependencyInfoAsync (lines 262-274)
	if !p.packageExists(packageID, resolvedVersion) {
		return nil, "", nil // Not cached locally, caller should try remote
	}

	// Read .nuspec file from cache
	// Matches ProcessNuspecReader (lines 430-457)
	nuspecPath := p.resolver.GetManifestFilePath(packageID, resolvedVersion)

	// Open nuspec file
	file, err := os.Open(nuspecPath)
	if err != nil {
		// File doesn't exist or can't be opened - not cached
		return nil, "", nil
	}
	defer func() { _ = file.Close() }()

	// Parse nuspec
	nuspec, err := packaging.ParseNuspec(file)
	if err != nil {
		return nil, "", fmt.Errorf("parse nuspec %s: %w", nuspecPath, err)
	}

	// Extract ALL dependency groups (framework selection happens in walker)
	// Matches GetDependencyInfo (FindPackageByIdResource.cs lines 145-156)
	groups, err := p.extractAllDependencyGroups(nuspec)
	if err != nil {
		return nil, "", err
	}

	// Return groups and resolved version string
	return groups, resolvedVersion.String(), nil
}

// packageExists checks if package is fully installed in local cache.
// Matches DoesVersionExist logic from LocalV3FindPackageByIdResource.
// Reference: Uses completion marker check (.nupkg.metadata or .nupkg.sha512)
func (p *LocalDependencyProvider) packageExists(packageID string, ver *version.NuGetVersion) bool {
	// Check for completion markers (primary: .nupkg.metadata, fallback: .nupkg.sha512)
	metadataPath := p.resolver.GetNupkgMetadataPath(packageID, ver)
	if _, err := os.Stat(metadataPath); err == nil {
		return true // Package fully installed
	}

	// Fallback: check for .nupkg.sha512 (older completion marker)
	hashPath := p.resolver.GetHashPath(packageID, ver)
	if _, err := os.Stat(hashPath); err == nil {
		return true // Package fully installed
	}

	return false // Not cached
}

// extractAllDependencyGroups parses ALL dependency groups from nuspec.
// Returns all groups without framework filtering (walker does that).
// Matches GetDependencyInfo logic from FindPackageByIdResource.cs (lines 145-156)
func (p *LocalDependencyProvider) extractAllDependencyGroups(
	nuspec *packaging.Nuspec,
) ([]resolver.DependencyGroup, error) {
	// Get dependency groups from nuspec
	parsedGroups, err := nuspec.GetDependencyGroups()
	if err != nil {
		return nil, err
	}

	// Convert all groups to resolver.DependencyGroup format
	groups := make([]resolver.DependencyGroup, 0, len(parsedGroups))
	for _, parsedGroup := range parsedGroups {
		group := resolver.DependencyGroup{
			TargetFramework: "", // Will be set below if framework exists
			Dependencies:    make([]resolver.PackageDependency, 0, len(parsedGroup.Dependencies)),
		}

		// Set target framework string (empty for "any" framework)
		if parsedGroup.TargetFramework != nil {
			group.TargetFramework = parsedGroup.TargetFramework.String()
		}

		// Convert dependencies
		for _, dep := range parsedGroup.Dependencies {
			versionRangeStr := ""
			if dep.VersionRange != nil {
				versionRangeStr = dep.VersionRange.String()
			}

			group.Dependencies = append(group.Dependencies, resolver.PackageDependency{
				ID:              dep.ID,
				VersionRange:    versionRangeStr,
				TargetFramework: group.TargetFramework,
			})
		}

		groups = append(groups, group)
	}

	return groups, nil
}

// getAllVersions enumerates all cached versions for a package ID.
// Returns nil if package directory doesn't exist or no valid versions found.
// Matches GetVersionsCore logic from LocalV3FindPackageByIdResource.cs (lines 488-532).
func (p *LocalDependencyProvider) getAllVersions(packageID string) ([]*version.NuGetVersion, error) {
	// Get package directory path (lowercase, matching NuGet behavior)
	// Matches _resolver.GetVersionListPath(id) logic
	packageDir := filepath.Join(p.packagesFolder, strings.ToLower(packageID))

	// Check if package directory exists
	// Matches if (idDir.Exists) check (line 494)
	dirInfo, err := os.Stat(packageDir)
	if err != nil || !dirInfo.IsDir() {
		return nil, nil // Package not cached
	}

	// Enumerate version directories
	// Matches foreach (var versionDir in idDir.EnumerateDirectories()) (line 497)
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return nil, nil
	}

	var versions []*version.NuGetVersion

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to parse directory name as version
		// Matches NuGetVersion.TryParse(versionPart, out version) (line 502)
		ver, err := version.Parse(entry.Name())
		if err != nil {
			// Skip invalid version directories
			// Matches continue on parse failure (line 509)
			continue
		}

		// Check if this version has completion markers (fully installed)
		// Matches if (DoesVersionExist(id, version)) (line 511)
		if p.packageExists(packageID, ver) {
			versions = append(versions, ver)
		}
	}

	return versions, nil
}
