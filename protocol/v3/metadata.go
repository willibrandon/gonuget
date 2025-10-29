package v3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/willibrandon/gonuget/cache"
	nugethttp "github.com/willibrandon/gonuget/http"
)

// MetadataClient provides package metadata functionality.
type MetadataClient struct {
	httpClient         *nugethttp.Client
	serviceIndexClient *ServiceIndexClient
	httpCache          *cache.DiskCache // HTTP disk cache (30min TTL like NuGet.Client)
}

// NewMetadataClient creates a new metadata client.
func NewMetadataClient(httpClient *nugethttp.Client, serviceIndexClient *ServiceIndexClient) *MetadataClient {
	return &MetadataClient{
		httpClient:         httpClient,
		serviceIndexClient: serviceIndexClient,
		httpCache:          nil, // No cache by default (set via SetHTTPCache)
	}
}

// SetHTTPCache configures the HTTP disk cache for registration API responses.
// Cache key format matches NuGet.Client: list_{packageid}_index, list_{packageid}_range_{lower}-{upper}
func (c *MetadataClient) SetHTTPCache(httpCache *cache.DiskCache) {
	c.httpCache = httpCache
}

// GetPackageMetadata retrieves metadata for a specific package ID.
// Returns all versions and their metadata.
func (c *MetadataClient) GetPackageMetadata(ctx context.Context, sourceURL, packageID string) (*RegistrationIndex, error) {
	// Get registration base URL from service index
	baseURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypeRegistrationsBaseURL)
	if err != nil {
		return nil, fmt.Errorf("get registration URL: %w", err)
	}

	// Build registration index URL
	// Format: {baseURL}/{packageID}/index.json
	packageIDLower := strings.ToLower(packageID)
	registrationURL := strings.TrimSuffix(baseURL, "/") + "/" + packageIDLower + "/index.json"

	// Cache key matches NuGet.Client: list_{packageid}
	cacheKey := fmt.Sprintf("list_%s", packageIDLower)

	// Try HTTP disk cache first (30min TTL like NuGet.Client)
	// But skip if NoCache is set in context
	const httpCacheTTL = 30 * time.Minute
	var index *RegistrationIndex

	// Check if NoCache is set via context
	cacheCtx := cache.FromContext(ctx)
	skipCache := cacheCtx != nil && cacheCtx.NoCache

	if c.httpCache != nil && !skipCache {
		cachedReader, hit, err := c.httpCache.Get(registrationURL, cacheKey, httpCacheTTL)
		if err == nil && hit && cachedReader != nil {
			// Cache hit - decode from cache
			defer func() { _ = cachedReader.Close() }()
			var cachedIndex RegistrationIndex
			if err := json.NewDecoder(cachedReader).Decode(&cachedIndex); err == nil {
				index = &cachedIndex
			}
		}
	}

	// Cache miss - fetch from network
	if index == nil {
		req, err := http.NewRequest("GET", registrationURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		resp, err := c.httpClient.DoWithRetry(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("fetch registration: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("package %q not found", packageID)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			return nil, fmt.Errorf("registration returned %d: %s", resp.StatusCode, body)
		}

		// Read response body into buffer for caching
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		// Decode index
		var fetchedIndex RegistrationIndex
		if err := json.Unmarshal(bodyBytes, &fetchedIndex); err != nil {
			return nil, fmt.Errorf("decode registration: %w", err)
		}
		index = &fetchedIndex

		// Write to HTTP cache if enabled
		if c.httpCache != nil {
			// Ignore cache write failures - they shouldn't fail the request
			_ = c.httpCache.Set(registrationURL, cacheKey, bytes.NewReader(bodyBytes), nil)
		}
	}

	// Fetch inline pages if items are not populated
	// OPTIMIZATION: Fetch all pages in parallel for massive speedup (5-6x for packages with many versions)
	var pagesToFetch []int
	for i := range index.Items {
		if len(index.Items[i].Items) == 0 && index.Items[i].ID != "" {
			pagesToFetch = append(pagesToFetch, i)
		}
	}

	if len(pagesToFetch) > 0 {
		// Parallel fetch with goroutines and channels
		type pageResult struct {
			index int
			page  *RegistrationPage
			err   error
		}

		results := make(chan pageResult, len(pagesToFetch))

		for _, idx := range pagesToFetch {
			go func(i int, url string) {
				page, err := c.fetchRegistrationPage(ctx, url)
				results <- pageResult{index: i, page: page, err: err}
			}(idx, index.Items[idx].ID)
		}

		// Collect results
		for range pagesToFetch {
			result := <-results
			if result.err != nil {
				return nil, fmt.Errorf("fetch page: %w", result.err)
			}
			index.Items[result.index] = *result.page
		}
	}

	return index, nil
}

// GetVersionMetadata retrieves metadata for a specific package version.
func (c *MetadataClient) GetVersionMetadata(ctx context.Context, sourceURL, packageID, version string) (*RegistrationCatalog, error) {
	index, err := c.GetPackageMetadata(ctx, sourceURL, packageID)
	if err != nil {
		return nil, err
	}

	// Search for version in all pages
	for _, page := range index.Items {
		for _, leaf := range page.Items {
			if leaf.CatalogEntry != nil && leaf.CatalogEntry.Version == version {
				return leaf.CatalogEntry, nil
			}
		}
	}

	return nil, fmt.Errorf("version %q not found for package %q", version, packageID)
}

// ListVersions returns all available versions for a package.
func (c *MetadataClient) ListVersions(ctx context.Context, sourceURL, packageID string) ([]string, error) {
	index, err := c.GetPackageMetadata(ctx, sourceURL, packageID)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, page := range index.Items {
		for _, leaf := range page.Items {
			if leaf.CatalogEntry != nil {
				versions = append(versions, leaf.CatalogEntry.Version)
			}
		}
	}

	return versions, nil
}

func (c *MetadataClient) fetchRegistrationPage(ctx context.Context, pageURL string) (*RegistrationPage, error) {
	// Extract package ID and version range from URL for cache key
	// URL format: {base}/{packageid}/page/{lower}/{upper}.json
	// Example: https://api.nuget.org/v3/registration5-gz-semver2/newtonsoft.json/page/0.1.1/13.0.3.json
	var cacheKey string
	if c.httpCache != nil {
		// Parse URL to extract components
		parts := strings.Split(pageURL, "/")
		if len(parts) >= 3 {
			// Find "page" in the path
			for i, part := range parts {
				if part == "page" && i+2 < len(parts) {
					// parts[i-1] is package ID
					// parts[i+1] is lower version
					// parts[i+2] is upper.json
					packageID := strings.ToLower(parts[i-1])
					lower := parts[i+1]
					upper := strings.TrimSuffix(parts[i+2], ".json")
					// Cache key matches NuGet.Client: list_{packageid}_range_{lower}-{upper}
					cacheKey = fmt.Sprintf("list_%s_range_%s-%s", packageID, lower, upper)
					break
				}
			}
		}
	}

	// Try HTTP disk cache first (30min TTL like NuGet.Client)
	const httpCacheTTL = 30 * time.Minute
	if c.httpCache != nil && cacheKey != "" {
		cachedReader, hit, err := c.httpCache.Get(pageURL, cacheKey, httpCacheTTL)
		if err == nil && hit && cachedReader != nil {
			// Cache hit - decode from cache
			defer func() { _ = cachedReader.Close() }()
			var cachedPage RegistrationPage
			if err := json.NewDecoder(cachedReader).Decode(&cachedPage); err == nil {
				return &cachedPage, nil
			}
		}
	}

	// Cache miss - fetch from network
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("page returned %d: %s", resp.StatusCode, body)
	}

	// Read response body into buffer for caching
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Decode page
	var page RegistrationPage
	if err := json.Unmarshal(bodyBytes, &page); err != nil {
		return nil, fmt.Errorf("decode page: %w", err)
	}

	// Write to HTTP cache if enabled
	if c.httpCache != nil && cacheKey != "" {
		_ = c.httpCache.Set(pageURL, cacheKey, bytes.NewReader(bodyBytes), nil)
	}

	return &page, nil
}
