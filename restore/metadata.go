package restore

import (
	"context"
	"fmt"
	"strings"

	"github.com/willibrandon/gonuget/core/resolver"
	"github.com/willibrandon/gonuget/frameworks"
)

// createLocalFirstMetadataClient creates a metadata client that checks local cache first before HTTP.
// Returns a resolver.PackageMetadataClient for use with Resolver/TransitiveResolver.
// Matches NuGet.Client's provider list prioritization: LocalLibraryProviders -> RemoteLibraryProviders
func (r *Restorer) createLocalFirstMetadataClient(
	localProvider *LocalDependencyProvider,
	targetFramework *frameworks.NuGetFramework,
) (resolver.PackageMetadataClient, error) {
	// Create local-first metadata client
	// Remote metadata client is created lazily only when needed (when local provider returns nil)
	return &localFirstMetadataClient{
		localProvider:   localProvider,
		restorer:        r,
		targetFramework: targetFramework,
	}, nil
}

// getRemoteMetadataClient creates a metadata client that implements resolver.PackageMetadataClient.
// This client makes HTTP calls to fetch package metadata using V3 registration API.
func (r *Restorer) getRemoteMetadataClient() (resolver.PackageMetadataClient, error) {
	// Use the client's new CreateMetadataClient method
	// This creates the efficient V3 metadata adapter that fetches all versions in a single HTTP call
	return r.client.CreateMetadataClient(r.opts.Sources)
}

// localFirstMetadataClient implements resolver.PackageMetadataClient.
// It checks the local dependency provider FIRST (no HTTP), then falls back to remote.
// Matches NuGet.Client's provider list prioritization: LocalLibraryProviders -> RemoteLibraryProviders
type localFirstMetadataClient struct {
	localProvider        *LocalDependencyProvider
	restorer             *Restorer
	remoteMetadataClient resolver.PackageMetadataClient // Lazy-initialized only when needed
	targetFramework      *frameworks.NuGetFramework
}

// GetPackageMetadata implements resolver.PackageMetadataClient.
// Tries local provider first (reads from cached .nuspec), falls back to HTTP if not cached.
func (c *localFirstMetadataClient) GetPackageMetadata(
	ctx context.Context,
	source string,
	packageID string,
	versionRange string,
) ([]*resolver.PackageDependencyInfo, error) {
	// Try local provider first (NO HTTP!)
	// LocalDependencyProvider now handles both exact versions and version ranges
	// Matches NuGet.Client: LocalLibraryProviders are tried before RemoteLibraryProviders
	depGroups, resolvedVersion, err := c.localProvider.GetDependencies(ctx, packageID, versionRange)
	if err != nil {
		// Error reading from cache - log and fall back to remote
		// Don't fail the restore just because we couldn't read a cached file
		// Silent fallback to HTTP (no logging)
	} else if depGroups != nil {
		// Found in local cache! Build PackageDependencyInfo from cached .nuspec
		// No logging - this is the fast path
		info := &resolver.PackageDependencyInfo{
			ID:               packageID,
			Version:          resolvedVersion, // Use resolved specific version (not the range!)
			DependencyGroups: depGroups,       // Return ALL groups (walker filters by framework)
		}

		return []*resolver.PackageDependencyInfo{info}, nil
	}

	// Not in local cache - lazy-initialize remote metadata client (only when needed)
	// This avoids creating HTTP clients and fetching service index until we actually need it
	if c.remoteMetadataClient == nil {
		remoteClient, err := c.restorer.getRemoteMetadataClient()
		if err != nil {
			return nil, fmt.Errorf("create remote metadata client: %w", err)
		}
		c.remoteMetadataClient = remoteClient
	}

	// Fall back to remote metadata client (HTTP)
	// This will fetch from nuget.org using V3 registration API
	// Matches NuGet.Client: RemoteLibraryProviders fallback
	return c.remoteMetadataClient.GetPackageMetadata(ctx, source, packageID, versionRange)
}

// isFrameworkReferencePack checks if a package ID is a framework reference pack.
// These are special packages downloaded by the SDK for targeting packs and should
// not be included in the regular package dependency lists.
func isFrameworkReferencePack(packageID string) bool {
	// Normalize to lowercase for comparison
	id := strings.ToLower(packageID)

	// Framework reference packs follow the pattern *.app.ref
	return strings.HasSuffix(id, ".app.ref") ||
		strings.HasSuffix(id, ".app.runtime.linux-x64") ||
		strings.HasSuffix(id, ".app.runtime.win-x64") ||
		strings.HasSuffix(id, ".app.runtime.osx-x64") ||
		strings.HasPrefix(id, "microsoft.netcore.app.runtime.") ||
		strings.HasPrefix(id, "microsoft.aspnetcore.app.runtime.") ||
		strings.HasPrefix(id, "microsoft.windowsdesktop.app.runtime.")
}
