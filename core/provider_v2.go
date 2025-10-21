package core

import (
	"context"
	"io"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/protocol/v2"
)

// V2ResourceProvider implements ResourceProvider for NuGet v2 feeds
type V2ResourceProvider struct {
	sourceURL      string
	searchClient   *v2.SearchClient
	metadataClient *v2.MetadataClient
	downloadClient *v2.DownloadClient
}

// NewV2ResourceProvider creates a new v2 resource provider
func NewV2ResourceProvider(sourceURL string, httpClient HTTPClient) *V2ResourceProvider {
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
	}
}

// GetMetadata retrieves metadata for a specific package version
func (p *V2ResourceProvider) GetMetadata(ctx context.Context, packageID, version string) (*ProtocolMetadata, error) {
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

		// Parse id:range:targetFramework
		id, rest, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		id = trimWhitespace(id)

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
func (p *V2ResourceProvider) ListVersions(ctx context.Context, packageID string) ([]string, error) {
	return p.metadataClient.ListVersions(ctx, p.sourceURL, packageID)
}

// Search searches for packages matching the query
func (p *V2ResourceProvider) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
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

	return results, nil
}

// DownloadPackage downloads a .nupkg file
func (p *V2ResourceProvider) DownloadPackage(ctx context.Context, packageID, version string) (io.ReadCloser, error) {
	return p.downloadClient.DownloadPackage(ctx, p.sourceURL, packageID, version)
}

// SourceURL returns the source URL
func (p *V2ResourceProvider) SourceURL() string {
	return p.sourceURL
}

// ProtocolVersion returns "v2"
func (p *V2ResourceProvider) ProtocolVersion() string {
	return "v2"
}
