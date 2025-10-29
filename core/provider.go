package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"

	"github.com/willibrandon/gonuget/cache"
	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/observability"
)

// ResourceProvider provides access to NuGet resources (search, metadata, download)
// Abstracts differences between v2 and v3 protocols
type ResourceProvider interface {
	// GetMetadata retrieves metadata for a specific package version
	// cacheCtx controls caching behavior (can be nil for default behavior)
	GetMetadata(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID, version string) (*ProtocolMetadata, error)

	// ListVersions lists all available versions for a package
	// cacheCtx controls caching behavior (can be nil for default behavior)
	ListVersions(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID string) ([]string, error)

	// Search searches for packages matching the query
	// cacheCtx controls caching behavior (can be nil for default behavior)
	Search(ctx context.Context, cacheCtx *cache.SourceCacheContext, query string, opts SearchOptions) ([]SearchResult, error)

	// DownloadPackage downloads a .nupkg file
	// cacheCtx controls caching behavior (can be nil for default behavior)
	DownloadPackage(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID, version string) (io.ReadCloser, error)

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
	cache      *cache.MultiTierCache
}

// NewProviderFactory creates a new provider factory
// cache can be nil if caching is not desired
func NewProviderFactory(httpClient HTTPClient, mtCache *cache.MultiTierCache) *ProviderFactory {
	return &ProviderFactory{
		httpClient: httpClient,
		cache:      mtCache,
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
	ctx, span := observability.StartProtocolDetectionSpan(ctx, sourceURL)
	defer span.End()

	// Fast-path for nuget.org URLs -> skip protocol detection (saves ~170ms per invocation)
	// nuget.org V3 is the fastest protocol, always use it when available

	// Fast-path for nuget.org V3 URL (already V3, no detection needed)
	if strings.Contains(sourceURL, "api.nuget.org/v3/index.json") {
		span.SetAttributes(attribute.String("protocol.fastpath", "nuget.org-v3-direct"))
		return NewV3ResourceProvider(sourceURL, f.httpClient, f.cache), nil
	}

	// Fast-path for nuget.org V2 URL -> use V3 protocol (30-40% faster)
	// nuget.org supports both V2 and V3, but V3 is significantly faster (JSON vs XML)
	if strings.Contains(sourceURL, "nuget.org/api/v2") {
		span.SetAttributes(attribute.String("protocol.fastpath", "nuget.org-v2-to-v3"))
		// Keep original V2 URL as sourceURL for repository matching
		// Use V3 service index URL for actual API calls
		return NewV3ResourceProviderWithServiceIndex(
			sourceURL,                             // Repository identifier
			"https://api.nuget.org/v3/index.json", // Service index URL
			f.httpClient,
			f.cache,
		), nil
	}

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
	// Use source URL as-is for NuGet.Client parity (M6.1)
	// Callers must provide full service index URL (e.g., https://api.nuget.org/v3/index.json)
	resp, err := f.httpClient.Get(ctx, sourceURL)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			contentType := resp.Header.Get("Content-Type")
			// V3 service index should return JSON
			if strings.Contains(contentType, "json") {
				// V3 feed detected
				return NewV3ResourceProvider(sourceURL, f.httpClient, f.cache), nil
			}
		}
	}

	// Try v2 - make direct HTTP call with authentication
	// For V2 detection, strip /index.json if present (V2 doesn't use service index)
	v2URL := strings.TrimSuffix(sourceURL, "/index.json")

	resp, err = f.httpClient.Get(ctx, v2URL)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == http.StatusOK {
			contentType := resp.Header.Get("Content-Type")
			// V2 feeds typically return XML
			if strings.Contains(contentType, "xml") || strings.Contains(contentType, "atom") {
				// V2 feed detected
				return NewV2ResourceProvider(v2URL, f.httpClient, f.cache), nil
			}
		}
	}

	return nil, fmt.Errorf("unable to detect protocol version for %s", sourceURL)
}

// CreateV3Provider creates a v3 resource provider (no detection)
func (f *ProviderFactory) CreateV3Provider(sourceURL string) ResourceProvider {
	return NewV3ResourceProvider(sourceURL, f.httpClient, f.cache)
}

// CreateV2Provider creates a v2 resource provider (no detection)
func (f *ProviderFactory) CreateV2Provider(sourceURL string) ResourceProvider {
	return NewV2ResourceProvider(sourceURL, f.httpClient, f.cache)
}
