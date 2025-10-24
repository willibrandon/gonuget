package core

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

// Client provides high-level NuGet package operations
type Client struct {
	repositoryManager *RepositoryManager
	targetFramework   *frameworks.NuGetFramework
}

// ClientConfig holds client configuration
type ClientConfig struct {
	RepositoryManager *RepositoryManager
	TargetFramework   *frameworks.NuGetFramework
}

// NewClient creates a new NuGet client
func NewClient(cfg ClientConfig) *Client {
	repoManager := cfg.RepositoryManager
	if repoManager == nil {
		repoManager = NewRepositoryManager()
	}

	return &Client{
		repositoryManager: repoManager,
		targetFramework:   cfg.TargetFramework,
	}
}

// GetRepositoryManager returns the repository manager
func (c *Client) GetRepositoryManager() *RepositoryManager {
	return c.repositoryManager
}

// SetTargetFramework sets the target framework for package operations
func (c *Client) SetTargetFramework(fw *frameworks.NuGetFramework) {
	c.targetFramework = fw
}

// GetTargetFramework returns the current target framework
func (c *Client) GetTargetFramework() *frameworks.NuGetFramework {
	return c.targetFramework
}

// SearchPackages searches for packages across all repositories
func (c *Client) SearchPackages(ctx context.Context, query string, opts SearchOptions) (map[string][]SearchResult, error) {
	return c.repositoryManager.SearchAll(ctx, nil, query, opts)
}

// GetPackageMetadata retrieves metadata from the first repository that has it
func (c *Client) GetPackageMetadata(ctx context.Context, packageID, versionStr string) (*ProtocolMetadata, error) {
	repos := c.repositoryManager.ListRepositories()
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories configured")
	}

	var lastErr error
	for _, repo := range repos {
		metadata, err := repo.GetMetadata(ctx, nil, packageID, versionStr)
		if err != nil {
			lastErr = err
			continue
		}
		return metadata, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("package not found: %w", lastErr)
	}

	return nil, fmt.Errorf("package %s %s not found in any repository", packageID, versionStr)
}

// ListVersions lists all versions from all repositories
func (c *Client) ListVersions(ctx context.Context, packageID string) ([]string, error) {
	repos := c.repositoryManager.ListRepositories()
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories configured")
	}

	// Collect versions from all repos
	versionsMap := make(map[string]bool)
	for _, repo := range repos {
		versions, err := repo.ListVersions(ctx, nil, packageID)
		if err != nil {
			continue // Skip repos that don't have the package
		}

		for _, v := range versions {
			versionsMap[v] = true
		}
	}

	if len(versionsMap) == 0 {
		return nil, fmt.Errorf("package %s not found in any repository", packageID)
	}

	// Convert to slice
	versions := make([]string, 0, len(versionsMap))
	for v := range versionsMap {
		versions = append(versions, v)
	}

	return versions, nil
}

// FindBestVersion finds the best matching version for a version range
func (c *Client) FindBestVersion(ctx context.Context, packageID string, versionRange *version.VersionRange) (*version.NuGetVersion, error) {
	// Get all versions
	versionStrings, err := c.ListVersions(ctx, packageID)
	if err != nil {
		return nil, err
	}

	// Parse versions
	versions := make([]*version.NuGetVersion, 0, len(versionStrings))
	for _, vStr := range versionStrings {
		v, err := version.Parse(vStr)
		if err != nil {
			continue // Skip invalid versions
		}
		versions = append(versions, v)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no valid versions found for %s", packageID)
	}

	// Find best match
	bestVersion := versionRange.FindBestMatch(versions)
	if bestVersion == nil {
		return nil, fmt.Errorf("no version satisfies range %s", versionRange.String())
	}

	return bestVersion, nil
}

// DownloadPackage downloads a package from the first repository that has it
func (c *Client) DownloadPackage(ctx context.Context, packageID, versionStr string) (io.ReadCloser, error) {
	repos := c.repositoryManager.ListRepositories()
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories configured")
	}

	var lastErr error
	for _, repo := range repos {
		body, err := repo.DownloadPackage(ctx, nil, packageID, versionStr)
		if err != nil {
			lastErr = err
			continue
		}
		return body, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("download failed: %w", lastErr)
	}

	return nil, fmt.Errorf("package %s %s not found in any repository", packageID, versionStr)
}

// GetCompatibleDependencies filters dependencies for the target framework
func (c *Client) GetCompatibleDependencies(metadata *PackageMetadata) ([]PackageDependency, error) {
	if c.targetFramework == nil {
		// No framework specified, return all dependencies
		if len(metadata.DependencyGroups) == 0 {
			return []PackageDependency{}, nil
		}
		// Return first group or merge all groups
		var allDeps []PackageDependency
		for _, group := range metadata.DependencyGroups {
			allDeps = append(allDeps, group.Dependencies...)
		}
		return allDeps, nil
	}

	// Use existing GetDependenciesForFramework method
	deps := metadata.GetDependenciesForFramework(c.targetFramework)
	if deps == nil {
		return []PackageDependency{}, nil
	}

	return deps, nil
}

// InstallPackageRequest represents a package installation request
type InstallPackageRequest struct {
	PackageID         string
	Version           string // Can be specific version or range
	TargetFramework   *frameworks.NuGetFramework
	IncludePrerelease bool
}

// ResolvePackageVersion resolves a version string (exact or range) to a specific version
func (c *Client) ResolvePackageVersion(ctx context.Context, packageID, versionStr string, includePrerelease bool) (*version.NuGetVersion, error) {
	// Try parsing as exact version first
	exactVer, err := version.Parse(versionStr)
	if err == nil {
		// Verify this version exists
		versions, err := c.ListVersions(ctx, packageID)
		if err != nil {
			return nil, err
		}

		if !slices.Contains(versions, versionStr) {
			return nil, fmt.Errorf("version %s not found", versionStr)
		}
		return exactVer, nil
	}

	// Try parsing as version range
	versionRange, err := version.ParseVersionRange(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version or range: %s", versionStr)
	}

	// Find best matching version
	return c.FindBestVersion(ctx, packageID, versionRange)
}
