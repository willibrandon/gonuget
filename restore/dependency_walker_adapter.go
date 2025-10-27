package restore

import (
	"context"
	"fmt"

	"github.com/willibrandon/gonuget/core"
	"github.com/willibrandon/gonuget/core/resolver"
)

// DependencyWalkerAdapter adapts core.Client to resolver.PackageMetadataClient.
// Matches NuGet.Client's RemoteWalkContext which provides package metadata
// to RemoteDependencyWalker (line 90-95 in RemoteDependencyWalker.cs).
type DependencyWalkerAdapter struct {
	client  *core.Client
	sources []string
}

// NewDependencyWalkerAdapter creates adapter.
func NewDependencyWalkerAdapter(client *core.Client, sources []string) *DependencyWalkerAdapter {
	return &DependencyWalkerAdapter{
		client:  client,
		sources: sources,
	}
}

// GetPackageMetadata fetches package metadata from sources.
// Matches NuGet.Client's ResolverUtility.FindLibraryCachedAsync behavior
// (RestoreCommand.cs line 196-200).
func (a *DependencyWalkerAdapter) GetPackageMetadata(
	ctx context.Context,
	source string,
	packageID string,
) ([]*resolver.PackageDependencyInfo, error) {
	// Get all versions of package
	versions, err := a.client.ListVersions(ctx, packageID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}

	// Fetch metadata for each version
	infos := make([]*resolver.PackageDependencyInfo, 0, len(versions))
	for _, ver := range versions {
		metadata, err := a.client.GetPackageMetadata(ctx, packageID, ver)
		if err != nil {
			continue // Skip unavailable versions
		}

		// Parse dependencies from ProtocolMetadata.Dependencies
		deps := make([]resolver.PackageDependency, 0)
		depGroups := make([]resolver.DependencyGroup, 0, len(metadata.Dependencies))

		for _, depGroup := range metadata.Dependencies {
			groupDeps := make([]resolver.PackageDependency, 0, len(depGroup.Dependencies))
			for _, dep := range depGroup.Dependencies {
				pd := resolver.PackageDependency{
					ID:           dep.ID,
					VersionRange: dep.Range,
				}
				groupDeps = append(groupDeps, pd)
				deps = append(deps, pd)
			}

			depGroups = append(depGroups, resolver.DependencyGroup{
				TargetFramework: depGroup.TargetFramework,
				Dependencies:    groupDeps,
			})
		}

		info := &resolver.PackageDependencyInfo{
			ID:               packageID,
			Version:          ver,
			Dependencies:     deps,
			DependencyGroups: depGroups,
		}
		infos = append(infos, info)
	}

	return infos, nil
}
