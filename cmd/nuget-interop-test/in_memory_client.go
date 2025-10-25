package main

import (
	"context"
	"fmt"

	"github.com/willibrandon/gonuget/core/resolver"
	"github.com/willibrandon/gonuget/version"
)

// InMemoryPackageClient provides in-memory packages for testing.
// Matches NuGet.Client's InMemoryDependencyProvider test harness pattern.
type InMemoryPackageClient struct {
	packages        map[string]map[string]*resolver.PackageDependencyInfo
	targetFramework string
}

// NewInMemoryPackageClient creates a client from protocol-level in-memory packages.
func NewInMemoryPackageClient(packages []InMemoryPackage, targetFramework string) (*InMemoryPackageClient, error) {
	client := &InMemoryPackageClient{
		packages:        make(map[string]map[string]*resolver.PackageDependencyInfo),
		targetFramework: targetFramework,
	}

	// Convert protocol packages to resolver.PackageDependencyInfo
	for _, pkg := range packages {
		if err := client.addPackage(pkg); err != nil {
			return nil, fmt.Errorf("add package %s: %w", pkg.ID, err)
		}
	}

	return client, nil
}

// addPackage converts InMemoryPackage to PackageDependencyInfo and stores it.
func (c *InMemoryPackageClient) addPackage(pkg InMemoryPackage) error {
	// Validate version
	_, err := version.Parse(pkg.Version)
	if err != nil {
		return fmt.Errorf("parse version %s: %w", pkg.Version, err)
	}

	// Convert dependencies
	deps := make([]resolver.PackageDependency, 0, len(pkg.Dependencies))
	for _, dep := range pkg.Dependencies {
		deps = append(deps, resolver.PackageDependency{
			ID:           dep.ID,
			VersionRange: dep.VersionRange,
		})
	}

	// Create PackageDependencyInfo
	info := &resolver.PackageDependencyInfo{
		ID:           pkg.ID,
		Version:      pkg.Version,
		Dependencies: deps,
	}

	// Store by ID and version
	if c.packages[pkg.ID] == nil {
		c.packages[pkg.ID] = make(map[string]*resolver.PackageDependencyInfo)
	}
	c.packages[pkg.ID][pkg.Version] = info

	return nil
}

// GetPackageMetadata implements resolver.PackageMetadataClient.
// Returns in-memory packages instead of making HTTP calls.
func (c *InMemoryPackageClient) GetPackageMetadata(
	ctx context.Context,
	source string,
	packageID string,
) ([]*resolver.PackageDependencyInfo, error) {
	versions, found := c.packages[packageID]
	if !found {
		// Return empty slice (package not found)
		return []*resolver.PackageDependencyInfo{}, nil
	}

	// Return all versions of this package
	results := make([]*resolver.PackageDependencyInfo, 0, len(versions))
	for _, info := range versions {
		results = append(results, info)
	}

	return results, nil
}
