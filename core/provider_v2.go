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
	"github.com/willibrandon/gonuget/protocol/v2"
)

// V2ResourceProvider implements ResourceProvider for NuGet v2 feeds
type V2ResourceProvider struct {
	sourceURL      string
	searchClient   *v2.SearchClient
	metadataClient *v2.MetadataClient
	downloadClient *v2.DownloadClient
	cache          *cache.MultiTierCache
}

// NewV2ResourceProvider creates a new v2 resource provider
// mtCache can be nil if caching is not desired
func NewV2ResourceProvider(sourceURL string, httpClient HTTPClient, mtCache *cache.MultiTierCache) *V2ResourceProvider {
	// Type assert to *nugethttp.Client for protocol clients
	// This is safe because HTTPClient interface is implemented by *nugethttp.Client
	// and authenticatedHTTPClient which wraps it
	var client *nugethttp.Client
	if c, ok := httpClient.(*nugethttp.Client); ok {
		client = c
	} else if ac, ok := httpClient.(*authenticatedHTTPClient); ok {
		client = ac.base
	}

	return &V2ResourceProvider{
		sourceURL:      sourceURL,
		searchClient:   v2.NewSearchClient(client),
		metadataClient: v2.NewMetadataClient(client),
		downloadClient: v2.NewDownloadClient(client),
		cache:          mtCache,
	}
}

// FindPackagesByID retrieves all versions of a package with full metadata in a single call.
// This is the efficient V2 method matching NuGet.Client's DependencyInfoResourceV2Feed.
func (p *V2ResourceProvider) FindPackagesByID(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID string) ([]*ProtocolMetadata, error) {
	// Use default cache context if none provided
	if cacheCtx == nil {
		cacheCtx = cache.NewSourceCacheContext()
	}

	// Check cache if enabled
	if p.cache != nil && !cacheCtx.NoCache {
		cacheKey := fmt.Sprintf("findpackagesbyid:%s", packageID)
		cached, hit, err := p.cache.Get(ctx, p.sourceURL, cacheKey, cacheCtx.MaxAge)
		if err == nil && hit {
			var packages []*ProtocolMetadata
			if err := json.Unmarshal(cached, &packages); err == nil {
				return packages, nil
			}
		}
	}

	// Fetch from network - single HTTP call gets all versions with dependencies
	v2Packages, err := p.metadataClient.FindPackagesByID(ctx, p.sourceURL, packageID)
	if err != nil {
		return nil, err
	}

	// Convert all packages to protocol format
	packages := make([]*ProtocolMetadata, 0, len(v2Packages))
	for _, v2Metadata := range v2Packages {
		metadata := &ProtocolMetadata{
			ID:                       v2Metadata.ID,
			Version:                  v2Metadata.Version,
			Title:                    v2Metadata.Title,
			Description:              v2Metadata.Description,
			IconURL:                  v2Metadata.IconURL,
			LicenseURL:               v2Metadata.LicenseURL,
			ProjectURL:               v2Metadata.ProjectURL,
			Tags:                     v2Metadata.Tags,
			DownloadCount:            v2Metadata.DownloadCount,
			IsPrerelease:             v2Metadata.IsPrerelease,
			Published:                v2Metadata.Published,
			RequireLicenseAcceptance: v2Metadata.RequireLicenseAcceptance,
			DownloadURL:              v2Metadata.DownloadURL,
		}

		// Parse authors
		if v2Metadata.Authors != "" {
			metadata.Authors = strings.Split(v2Metadata.Authors, ",")
			for i := range metadata.Authors {
				metadata.Authors[i] = strings.TrimSpace(metadata.Authors[i])
			}
		}

		// Parse v2 dependency string into ProtocolDependencyGroup
		if v2Metadata.Dependencies != "" {
			metadata.Dependencies = parseDependencies(v2Metadata.Dependencies)
		}

		packages = append(packages, metadata)
	}

	// Cache result if enabled
	if p.cache != nil && !cacheCtx.DirectDownload {
		cacheKey := fmt.Sprintf("findpackagesbyid:%s", packageID)
		if jsonData, err := json.Marshal(packages); err == nil {
			_ = p.cache.Set(ctx, p.sourceURL, cacheKey, bytes.NewReader(jsonData), cacheCtx.MaxAge, nil)
		}
	}

	return packages, nil
}

// GetMetadata retrieves metadata for a specific package version.
// Uses V2 OData feed to fetch detailed package information.
func (p *V2ResourceProvider) GetMetadata(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID, version string) (*ProtocolMetadata, error) {
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
	v2Metadata, err := p.metadataClient.GetPackageMetadata(ctx, p.sourceURL, packageID, version)
	if err != nil {
		return nil, err
	}

	// Convert v2 metadata to protocol format
	metadata := &ProtocolMetadata{
		ID:                       v2Metadata.ID,
		Version:                  v2Metadata.Version,
		Title:                    v2Metadata.Title,
		Description:              v2Metadata.Description,
		IconURL:                  v2Metadata.IconURL,
		LicenseURL:               v2Metadata.LicenseURL,
		ProjectURL:               v2Metadata.ProjectURL,
		Tags:                     v2Metadata.Tags,
		DownloadCount:            v2Metadata.DownloadCount,
		IsPrerelease:             v2Metadata.IsPrerelease,
		Published:                v2Metadata.Published,
		RequireLicenseAcceptance: v2Metadata.RequireLicenseAcceptance,
		DownloadURL:              v2Metadata.DownloadURL,
	}

	// Parse authors
	if v2Metadata.Authors != "" {
		metadata.Authors = strings.Split(v2Metadata.Authors, ",")
		for i := range metadata.Authors {
			metadata.Authors[i] = strings.TrimSpace(metadata.Authors[i])
		}
	}

	// Parse v2 dependency string into ProtocolDependencyGroup
	// V2 format: "PackageA:1.0:net45|PackageB:2.0:netstandard2.0"
	if v2Metadata.Dependencies != "" {
		metadata.Dependencies = parseDependencies(v2Metadata.Dependencies)
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

// trimWhitespace trims leading/trailing whitespace without allocating if unchanged.
func trimWhitespace(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Trim trailing whitespace
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	if start == 0 && end == len(s) {
		return s // No trimming needed, return original (no allocation)
	}

	return s[start:end] // Return substring (no allocation, just new string header)
}

// parseDependencies parses v2 dependency string format into ProtocolDependencyGroup.
// Format: "id:range:targetFramework|id:range:targetFramework|..."
func parseDependencies(deps string) []ProtocolDependencyGroup {
	if deps == "" {
		return nil
	}

	// Group dependencies by target framework (pre-allocate with expected capacity)
	groups := make(map[string][]ProtocolDependency, 2)

	// Iterate over pipe-separated parts using Cut (more efficient than Split)
	for len(deps) > 0 {
		var part string
		part, deps, _ = strings.Cut(deps, "|")
		part = trimWhitespace(part)
		if part == "" {
			continue
		}

		// Handle empty dependency groups: "::net45" means no dependencies for net45
		// Format: "::framework" for empty groups, "id:range:framework" for dependencies
		if strings.HasPrefix(part, "::") {
			// Empty dependency group marker
			framework := trimWhitespace(part[2:]) // Skip "::"
			if framework != "" {
				// Ensure empty group exists in map
				if _, exists := groups[framework]; !exists {
					groups[framework] = []ProtocolDependency{}
				}
			}
			continue
		}

		// Parse id:range:targetFramework
		id, rest, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		id = trimWhitespace(id)

		// Skip empty IDs (can occur in malformed V2 metadata)
		if id == "" {
			continue
		}

		versionRange, targetFramework, _ := strings.Cut(rest, ":")
		versionRange = trimWhitespace(versionRange)
		targetFramework = trimWhitespace(targetFramework)

		dep := ProtocolDependency{
			ID:    id,
			Range: versionRange,
		}

		groups[targetFramework] = append(groups[targetFramework], dep)
	}

	// Pre-allocate result slice with exact size (avoids growth)
	result := make([]ProtocolDependencyGroup, 0, len(groups))
	for framework, deps := range groups {
		result = append(result, ProtocolDependencyGroup{
			TargetFramework: framework,
			Dependencies:    deps,
		})
	}

	return result
}

// ListVersions lists all available versions for a package
func (p *V2ResourceProvider) ListVersions(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID string) ([]string, error) {
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
	versions, err := p.metadataClient.ListVersions(ctx, p.sourceURL, packageID)
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
func (p *V2ResourceProvider) Search(ctx context.Context, cacheCtx *cache.SourceCacheContext, query string, opts SearchOptions) ([]SearchResult, error) {
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
	searchOpts := v2.SearchOptions{
		Query:             query,
		Skip:              opts.Skip,
		Top:               opts.Take,
		IncludePrerelease: opts.IncludePrerelease,
	}

	v2Results, err := p.searchClient.Search(ctx, p.sourceURL, searchOpts)
	if err != nil {
		return nil, err
	}

	// Convert v2 results to common format
	results := make([]SearchResult, 0, len(v2Results))
	for _, r := range v2Results {
		result := SearchResult{
			ID:             r.ID,
			Version:        r.Version,
			Description:    r.Description,
			IconURL:        r.IconURL,
			Tags:           r.Tags,
			TotalDownloads: r.DownloadCount,
		}

		// Parse authors
		if r.Authors != "" {
			result.Authors = strings.Split(r.Authors, ",")
			for i := range result.Authors {
				result.Authors[i] = strings.TrimSpace(result.Authors[i])
			}
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
func (p *V2ResourceProvider) DownloadPackage(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID, version string) (io.ReadCloser, error) {
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
	reader, err := p.downloadClient.DownloadPackage(ctx, p.sourceURL, packageID, version)
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
func (p *V2ResourceProvider) SourceURL() string {
	return p.sourceURL
}

// ProtocolVersion returns "v2"
func (p *V2ResourceProvider) ProtocolVersion() string {
	return "v2"
}
