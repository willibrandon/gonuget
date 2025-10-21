package core

import (
	"context"
	"fmt"
	"io"

	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/protocol/v2"
	"github.com/willibrandon/gonuget/protocol/v3"
)

// ResourceProvider provides access to NuGet resources (search, metadata, download)
// Abstracts differences between v2 and v3 protocols
type ResourceProvider interface {
	// GetMetadata retrieves metadata for a specific package version
	GetMetadata(ctx context.Context, packageID, version string) (*ProtocolMetadata, error)

	// ListVersions lists all available versions for a package
	ListVersions(ctx context.Context, packageID string) ([]string, error)

	// Search searches for packages matching the query
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)

	// DownloadPackage downloads a .nupkg file
	DownloadPackage(ctx context.Context, packageID, version string) (io.ReadCloser, error)

	// SourceURL returns the source URL for this provider
	SourceURL() string

	// ProtocolVersion returns the protocol version (v2 or v3)
	ProtocolVersion() string
}

// ProtocolMetadata represents package metadata from protocol (string-based, simple types)
type ProtocolMetadata struct {
	ID                       string
	Version                  string
	Title                    string
	Description              string
	Summary                  string
	Authors                  []string
	Owners                   []string
	IconURL                  string
	LicenseURL               string
	LicenseExpression        string
	ProjectURL               string
	Tags                     []string
	Dependencies             []ProtocolDependencyGroup
	DownloadCount            int64
	IsPrerelease             bool
	Published                string
	RequireLicenseAcceptance bool
	DownloadURL              string
}

// ProtocolDependencyGroup represents dependencies for a target framework (string-based)
type ProtocolDependencyGroup struct {
	TargetFramework string
	Dependencies    []ProtocolDependency
}

// ProtocolDependency represents a single dependency (string-based)
type ProtocolDependency struct {
	ID    string
	Range string
}

// SearchResult represents a search result from any protocol version
type SearchResult struct {
	ID             string
	Version        string
	Description    string
	Authors        []string
	IconURL        string
	Tags           []string
	TotalDownloads int64
	Verified       bool
}

// SearchOptions holds common search parameters
type SearchOptions struct {
	Skip              int
	Take              int
	IncludePrerelease bool
}

// ProviderFactory creates resource providers based on protocol detection
type ProviderFactory struct {
	httpClient *nugethttp.Client
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(httpClient *nugethttp.Client) *ProviderFactory {
	return &ProviderFactory{
		httpClient: httpClient,
	}
}

// CreateProvider creates a resource provider for the given source URL
// Automatically detects v2 vs v3 protocol
func (f *ProviderFactory) CreateProvider(ctx context.Context, sourceURL string) (ResourceProvider, error) {
	// Try v3 first (modern protocol)
	serviceIndexClient := v3.NewServiceIndexClient(f.httpClient)
	_, err := serviceIndexClient.GetServiceIndex(ctx, sourceURL)
	if err == nil {
		// V3 feed detected
		return NewV3ResourceProvider(sourceURL, f.httpClient), nil
	}

	// Try v2
	feedClient := v2.NewFeedClient(f.httpClient)
	isV2, err := feedClient.DetectV2Feed(ctx, sourceURL)
	if err == nil && isV2 {
		// V2 feed detected
		return NewV2ResourceProvider(sourceURL, f.httpClient), nil
	}

	return nil, fmt.Errorf("unable to detect protocol version for %s", sourceURL)
}

// CreateV3Provider creates a v3 resource provider (no detection)
func (f *ProviderFactory) CreateV3Provider(sourceURL string) ResourceProvider {
	return NewV3ResourceProvider(sourceURL, f.httpClient)
}

// CreateV2Provider creates a v2 resource provider (no detection)
func (f *ProviderFactory) CreateV2Provider(sourceURL string) ResourceProvider {
	return NewV2ResourceProvider(sourceURL, f.httpClient)
}
