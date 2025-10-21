package core

import (
	"context"
	"io"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/protocol/v3"
)

// V3ResourceProvider implements ResourceProvider for NuGet v3 feeds
type V3ResourceProvider struct {
	sourceURL          string
	serviceIndexClient *v3.ServiceIndexClient
	searchClient       *v3.SearchClient
	metadataClient     *v3.MetadataClient
	downloadClient     *v3.DownloadClient
}

// NewV3ResourceProvider creates a new v3 resource provider
func NewV3ResourceProvider(sourceURL string, httpClient *nugethttp.Client) *V3ResourceProvider {
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)

	return &V3ResourceProvider{
		sourceURL:          sourceURL,
		serviceIndexClient: serviceIndexClient,
		searchClient:       v3.NewSearchClient(httpClient, serviceIndexClient),
		metadataClient:     v3.NewMetadataClient(httpClient, serviceIndexClient),
		downloadClient:     v3.NewDownloadClient(httpClient, serviceIndexClient),
	}
}

// GetMetadata retrieves metadata for a specific package version
func (p *V3ResourceProvider) GetMetadata(ctx context.Context, packageID, version string) (*ProtocolMetadata, error) {
	catalog, err := p.metadataClient.GetVersionMetadata(ctx, p.sourceURL, packageID, version)
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

	// Parse tags
	if catalog.Tags != "" {
		metadata.Tags = strings.Split(catalog.Tags, " ")
	}

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

	return metadata, nil
}

// ListVersions lists all available versions for a package
func (p *V3ResourceProvider) ListVersions(ctx context.Context, packageID string) ([]string, error) {
	return p.metadataClient.ListVersions(ctx, p.sourceURL, packageID)
}

// Search searches for packages matching the query
func (p *V3ResourceProvider) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	searchOpts := v3.SearchOptions{
		Query:      query,
		Skip:       opts.Skip,
		Take:       opts.Take,
		Prerelease: opts.IncludePrerelease,
	}

	resp, err := p.searchClient.Search(ctx, p.sourceURL, searchOpts)
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

	return results, nil
}

// DownloadPackage downloads a .nupkg file
func (p *V3ResourceProvider) DownloadPackage(ctx context.Context, packageID, version string) (io.ReadCloser, error) {
	return p.downloadClient.DownloadPackage(ctx, p.sourceURL, packageID, version)
}

// SourceURL returns the source URL
func (p *V3ResourceProvider) SourceURL() string {
	return p.sourceURL
}

// ProtocolVersion returns "v3"
func (p *V3ResourceProvider) ProtocolVersion() string {
	return "v3"
}
