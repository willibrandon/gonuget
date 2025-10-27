package restore

import (
	"context"
	"fmt"

	"github.com/willibrandon/gonuget/core"
	"github.com/willibrandon/gonuget/version"
)

// ResolveLatestVersionOptions holds options for version resolution.
type ResolveLatestVersionOptions struct {
	Source     string
	Prerelease bool
}

// ResolveLatestVersion finds the latest version of a package.
// Returns the latest stable version by default, or latest prerelease if Prerelease is true.
// Ported from NuGet.Protocol version resolution logic.
func ResolveLatestVersion(ctx context.Context, packageID string, opts *ResolveLatestVersionOptions) (string, error) {
	// Use default source if not specified
	source := opts.Source
	if source == "" {
		source = "https://api.nuget.org/v3/index.json"
	}

	// Create repository manager and add the source
	repoManager := core.NewRepositoryManager()
	repo := core.NewSourceRepository(core.RepositoryConfig{
		Name:      "default",
		SourceURL: source,
	})

	if err := repoManager.AddRepository(repo); err != nil {
		return "", fmt.Errorf("failed to add repository %s: %w", source, err)
	}

	// List all versions
	versions, err := repo.ListVersions(ctx, nil, packageID)
	if err != nil {
		return "", fmt.Errorf("failed to list versions: %w", err)
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("package '%s' not found in source %s", packageID, source)
	}

	// Filter and find latest version
	// NuGet.Client behavior:
	// - When includePrerelease=false: Only consider stable versions, return max
	// - When includePrerelease=true: Consider ALL versions (stable + prerelease), return max
	var latest *version.NuGetVersion

	for _, v := range versions {
		parsed, err := version.Parse(v)
		if err != nil {
			// Skip invalid versions
			continue
		}

		// Skip prerelease versions if not requested
		if !opts.Prerelease && parsed.IsPrerelease() {
			continue
		}

		// Find the maximum version
		if latest == nil || parsed.Compare(latest) > 0 {
			latest = parsed
		}
	}

	if latest != nil {
		return latest.String(), nil
	}

	if !opts.Prerelease {
		return "", fmt.Errorf("no stable version found for package '%s'. Use --prerelease to include prerelease versions", packageID)
	}

	return "", fmt.Errorf("no versions found for package '%s'", packageID)
}
