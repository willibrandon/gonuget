package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/willibrandon/gonuget/cache"
	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/protocol/v3"
)

// V3ResourceProvider implements ResourceProvider for NuGet v3 feeds
type V3ResourceProvider struct {
	sourceURL          string // Repository source URL (for matching)
	serviceIndexURL    string // V3 service index URL (for API calls)
	serviceIndexClient *v3.ServiceIndexClient
	searchClient       *v3.SearchClient
	metadataClient     *v3.MetadataClient
	downloadClient     *v3.DownloadClient
	cache              *cache.MultiTierCache
}

// NewV3ResourceProvider creates a new v3 resource provider
// mtCache can be nil if caching is not desired
// For normal V3 feeds: sourceURL is the service index URL
// For fast-path: use NewV3ResourceProviderWithServiceIndex to specify different URLs
func NewV3ResourceProvider(sourceURL string, httpClient HTTPClient, mtCache *cache.MultiTierCache) *V3ResourceProvider {
	return NewV3ResourceProviderWithServiceIndex(sourceURL, sourceURL, httpClient, mtCache)
}

// NewV3ResourceProviderWithServiceIndex creates a V3 provider with separate source and service index URLs
// sourceURL: Repository identifier (used for matching repositories)
// serviceIndexURL: Actual V3 service index endpoint
// Used for nuget.org fast-path: sourceURL="https://www.nuget.org/api/v2", serviceIndexURL="https://api.nuget.org/v3/index.json"
func NewV3ResourceProviderWithServiceIndex(sourceURL, serviceIndexURL string, httpClient HTTPClient, mtCache *cache.MultiTierCache) *V3ResourceProvider {
	// Type assert to *nugethttp.Client for protocol clients
	// This is safe because HTTPClient interface is implemented by *nugethttp.Client
	// and authenticatedHTTPClient which wraps it
	var client *nugethttp.Client
	if c, ok := httpClient.(*nugethttp.Client); ok {
		client = c
	} else if ac, ok := httpClient.(*authenticatedHTTPClient); ok {
		client = ac.base
	}

	// Pass cache to service index client for disk caching (critical for first-run performance)
	serviceIndexClient := v3.NewServiceIndexClientWithCache(client, mtCache)

	return &V3ResourceProvider{
		sourceURL:          sourceURL,
		serviceIndexURL:    serviceIndexURL,
		serviceIndexClient: serviceIndexClient,
		searchClient:       v3.NewSearchClient(client, serviceIndexClient),
		metadataClient:     v3.NewMetadataClient(client, serviceIndexClient),
		downloadClient:     v3.NewDownloadClient(client, serviceIndexClient),
		cache:              mtCache,
	}
}

// GetMetadata retrieves metadata for a specific package version
func (p *V3ResourceProvider) GetMetadata(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID, version string) (*ProtocolMetadata, error) {
	// Use default cache context if none provided
	if cacheCtx == nil {
		cacheCtx = cache.NewSourceCacheContext()
	}

	// Check cache if enabled
	if p.cache != nil && !cacheCtx.NoCache {
		cacheKey := fmt.Sprintf("metadata:%s:%s", packageID, version)
		cached, hit, err := p.cache.Get(ctx, p.sourceURL, cacheKey, cacheCtx.MaxAge)
		if err == nil && hit {
			var metadata ProtocolMetadata
			if err := json.Unmarshal(cached, &metadata); err == nil {
				return &metadata, nil
			}
		}
	}

	// Fetch from network
	catalog, err := p.metadataClient.GetVersionMetadata(ctx, p.serviceIndexURL, packageID, version)
	if err != nil {
		return nil, err
	}

	// Convert v3 catalog to protocol metadata
	metadata := &ProtocolMetadata{
		ID:                       catalog.PackageID,
		Version:                  catalog.Version,
		Title:                    catalog.Title,
		Description:              catalog.Description,
		Summary:                  catalog.Summary,
		IconURL:                  catalog.IconURL,
		LicenseURL:               catalog.LicenseURL,
		LicenseExpression:        catalog.LicenseExpression,
		ProjectURL:               catalog.ProjectURL,
		RequireLicenseAcceptance: catalog.RequireLicenseAcceptance,
	}

	// Parse authors
	if catalog.Authors != "" {
		metadata.Authors = strings.Split(catalog.Authors, ",")
		for i := range metadata.Authors {
			metadata.Authors[i] = strings.TrimSpace(metadata.Authors[i])
		}
	}

	// Copy tags
	metadata.Tags = catalog.Tags

	// Convert dependency groups
	for _, dg := range catalog.DependencyGroups {
		group := ProtocolDependencyGroup{
			TargetFramework: dg.TargetFramework,
			Dependencies:    make([]ProtocolDependency, 0, len(dg.Dependencies)),
		}

		for _, dep := range dg.Dependencies {
			group.Dependencies = append(group.Dependencies, ProtocolDependency{
				ID:    dep.ID,
				Range: dep.Range,
			})
		}

		metadata.Dependencies = append(metadata.Dependencies, group)
	}

	// Cache result if enabled
	if p.cache != nil && !cacheCtx.DirectDownload {
		cacheKey := fmt.Sprintf("metadata:%s:%s", packageID, version)
		if jsonData, err := json.Marshal(metadata); err == nil {
			_ = p.cache.Set(ctx, p.sourceURL, cacheKey, bytes.NewReader(jsonData), cacheCtx.MaxAge, nil)
		}
	}

	return metadata, nil
}

// ListVersions lists all available versions for a package
func (p *V3ResourceProvider) ListVersions(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID string) ([]string, error) {
	// Use default cache context if none provided
	if cacheCtx == nil {
		cacheCtx = cache.NewSourceCacheContext()
	}

	// Check cache if enabled
	if p.cache != nil && !cacheCtx.NoCache {
		cacheKey := fmt.Sprintf("versions:%s", packageID)
		cached, hit, err := p.cache.Get(ctx, p.sourceURL, cacheKey, cacheCtx.MaxAge)
		if err == nil && hit {
			var versions []string
			if err := json.Unmarshal(cached, &versions); err == nil {
				return versions, nil
			}
		}
	}

	// Fetch from network
	versions, err := p.metadataClient.ListVersions(ctx, p.serviceIndexURL, packageID)
	if err != nil {
		return nil, err
	}

	// Cache result if enabled
	if p.cache != nil && !cacheCtx.DirectDownload {
		cacheKey := fmt.Sprintf("versions:%s", packageID)
		if jsonData, err := json.Marshal(versions); err == nil {
			_ = p.cache.Set(ctx, p.sourceURL, cacheKey, bytes.NewReader(jsonData), cacheCtx.MaxAge, nil)
		}
	}

	return versions, nil
}

// Search searches for packages matching the query
func (p *V3ResourceProvider) Search(ctx context.Context, cacheCtx *cache.SourceCacheContext, query string, opts SearchOptions) ([]SearchResult, error) {
	// Use default cache context if none provided
	if cacheCtx == nil {
		cacheCtx = cache.NewSourceCacheContext()
	}

	// Check cache if enabled
	if p.cache != nil && !cacheCtx.NoCache {
		cacheKey := fmt.Sprintf("search:%s:%d:%d:%t", query, opts.Skip, opts.Take, opts.IncludePrerelease)
		cached, hit, err := p.cache.Get(ctx, p.sourceURL, cacheKey, cacheCtx.MaxAge)
		if err == nil && hit {
			var results []SearchResult
			if err := json.Unmarshal(cached, &results); err == nil {
				return results, nil
			}
		}
	}

	// Fetch from network
	searchOpts := v3.SearchOptions{
		Query:      query,
		Skip:       opts.Skip,
		Take:       opts.Take,
		Prerelease: opts.IncludePrerelease,
	}

	resp, err := p.searchClient.Search(ctx, p.serviceIndexURL, searchOpts)
	if err != nil {
		return nil, err
	}

	// Convert v3 results to common format
	results := make([]SearchResult, 0, len(resp.Data))
	for _, r := range resp.Data {
		result := SearchResult{
			ID:             r.PackageID,
			Version:        r.Version,
			Description:    r.Description,
			Authors:        r.Authors,
			IconURL:        r.IconURL,
			Tags:           r.Tags,
			TotalDownloads: r.TotalDownloads,
			Verified:       r.Verified,
		}
		results = append(results, result)
	}

	// Cache result if enabled
	if p.cache != nil && !cacheCtx.DirectDownload {
		cacheKey := fmt.Sprintf("search:%s:%d:%d:%t", query, opts.Skip, opts.Take, opts.IncludePrerelease)
		if jsonData, err := json.Marshal(results); err == nil {
			_ = p.cache.Set(ctx, p.sourceURL, cacheKey, bytes.NewReader(jsonData), cacheCtx.MaxAge, nil)
		}
	}

	return results, nil
}

// DownloadPackage downloads a .nupkg file
func (p *V3ResourceProvider) DownloadPackage(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID, version string) (io.ReadCloser, error) {
	// Use default cache context if none provided
	if cacheCtx == nil {
		cacheCtx = cache.NewSourceCacheContext()
	}

	// Check cache if enabled
	if p.cache != nil && !cacheCtx.NoCache {
		cacheKey := fmt.Sprintf("package:%s.%s.nupkg", packageID, version)
		cached, hit, err := p.cache.Get(ctx, p.sourceURL, cacheKey, cacheCtx.MaxAge)
		if err == nil && hit {
			return io.NopCloser(bytes.NewReader(cached)), nil
		}
	}

	// Download from network
	reader, err := p.downloadClient.DownloadPackage(ctx, p.serviceIndexURL, packageID, version)
	if err != nil {
		return nil, err
	}

	// Read package data for caching
	packageData, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		return nil, err
	}

	// Cache result if enabled (with ZIP validation)
	if p.cache != nil && !cacheCtx.DirectDownload {
		cacheKey := fmt.Sprintf("package:%s.%s.nupkg", packageID, version)
		validator := func(rs io.ReadSeeker) error {
			// Basic ZIP validation - check for PK signature
			var sig [2]byte
			if _, err := rs.Read(sig[:]); err != nil {
				return fmt.Errorf("failed to read signature: %w", err)
			}
			if sig[0] != 0x50 || sig[1] != 0x4B { // PK signature
				return fmt.Errorf("invalid ZIP signature")
			}
			_, _ = rs.Seek(0, io.SeekStart) // Reset for caching
			return nil
		}
		_ = p.cache.Set(ctx, p.sourceURL, cacheKey, bytes.NewReader(packageData), cacheCtx.MaxAge, validator)
	}

	return io.NopCloser(bytes.NewReader(packageData)), nil
}

// SourceURL returns the source URL
func (p *V3ResourceProvider) SourceURL() string {
	return p.sourceURL
}

// ProtocolVersion returns "v3"
func (p *V3ResourceProvider) ProtocolVersion() string {
	return "v3"
}

// ServiceIndexURL returns the V3 service index URL
func (p *V3ResourceProvider) ServiceIndexURL() string {
	return p.serviceIndexURL
}
