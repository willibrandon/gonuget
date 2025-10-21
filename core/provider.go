package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
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

// HTTPClient defines the interface for making HTTP requests
type HTTPClient interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
	Get(ctx context.Context, url string) (*http.Response, error)
	DoWithRetry(ctx context.Context, req *http.Request) (*http.Response, error)
	SetUserAgent(ua string)
}

// ProviderFactory creates resource providers based on protocol detection
type ProviderFactory struct {
	httpClient HTTPClient
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(httpClient HTTPClient) *ProviderFactory {
	return &ProviderFactory{
		httpClient: httpClient,
	}
}

// getConcreteClient extracts the underlying *nugethttp.Client from HTTPClient
func getConcreteClient(client HTTPClient) *nugethttp.Client {
	if c, ok := client.(*nugethttp.Client); ok {
		return c
	}
	if ac, ok := client.(*authenticatedHTTPClient); ok {
		return ac.base
	}
	// This should never happen if the interface is used correctly
	return nil
}

// CreateProvider creates a resource provider for the given source URL
// Automatically detects v2 vs v3 protocol
func (f *ProviderFactory) CreateProvider(ctx context.Context, sourceURL string) (ResourceProvider, error) {
	// Extract concrete client for protocol detection
	// Note: If httpClient is already authenticated, the concrete client will still be wrapped
	// by the authenticatedHTTPClient, so protocol detection requests will be authenticated
	concreteClient := getConcreteClient(f.httpClient)
	if concreteClient == nil {
		return nil, fmt.Errorf("invalid HTTP client")
	}

	// For protocol detection, we need to use the httpClient (potentially authenticated)
	// but the protocol clients need *nugethttp.Client. We'll make the detection calls
	// directly using the HTTPClient interface.

	// Try v3 first (modern protocol) - make direct HTTP call with authentication
	indexURL := sourceURL
	if indexURL[len(indexURL)-1] != '/' {
		indexURL += "/"
	}
	indexURL += "index.json"

	resp, err := f.httpClient.Get(ctx, indexURL)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			// V3 feed detected
			return NewV3ResourceProvider(sourceURL, f.httpClient), nil
		}
	}

	// Try v2 - make direct HTTP call with authentication
	resp, err = f.httpClient.Get(ctx, sourceURL)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == http.StatusOK {
			contentType := resp.Header.Get("Content-Type")
			// V2 feeds typically return XML
			if strings.Contains(contentType, "xml") || strings.Contains(contentType, "atom") {
				return NewV2ResourceProvider(sourceURL, f.httpClient), nil
			}
		}
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
