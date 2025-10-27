package v3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/willibrandon/gonuget/cache"
	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/observability"
)

// ServiceIndexClient provides access to NuGet v3 service index.
type ServiceIndexClient struct {
	httpClient *nugethttp.Client
	diskCache  *cache.MultiTierCache // Optional disk cache (nil disables)

	mu    sync.RWMutex
	cache map[string]*cachedServiceIndex
}

type cachedServiceIndex struct {
	index     *ServiceIndex
	expiresAt time.Time
}

// NewServiceIndexClient creates a new service index client.
func NewServiceIndexClient(httpClient *nugethttp.Client) *ServiceIndexClient {
	return NewServiceIndexClientWithCache(httpClient, nil)
}

// NewServiceIndexClientWithCache creates a new service index client with optional disk cache.
func NewServiceIndexClientWithCache(httpClient *nugethttp.Client, mtCache *cache.MultiTierCache) *ServiceIndexClient {
	return &ServiceIndexClient{
		httpClient: httpClient,
		diskCache:  mtCache,
		cache:      make(map[string]*cachedServiceIndex),
	}
}

// GetServiceIndex retrieves the service index for a given source URL.
// Caches the result for ServiceIndexCacheTTL.
func (c *ServiceIndexClient) GetServiceIndex(ctx context.Context, sourceURL string) (*ServiceIndex, error) {
	ctx, span := observability.StartServiceIndexFetchSpan(ctx, sourceURL)
	defer span.End()

	// Check memory cache first (L1)
	c.mu.RLock()
	cached, ok := c.cache[sourceURL]
	c.mu.RUnlock()

	if ok && time.Now().Before(cached.expiresAt) {
		span.SetAttributes(
			attribute.Bool("cache.hit", true),
			attribute.String("cache.tier", "memory"))
		return cached.index, nil
	}

	// Check disk cache (L2) if available
	if c.diskCache != nil {
		data, ok, err := c.diskCache.Get(ctx, sourceURL, "service_index", ServiceIndexCacheTTL)
		if err == nil && ok {
			// Deserialize from disk cache
			var index ServiceIndex
			if err := json.Unmarshal(data, &index); err == nil {
				span.SetAttributes(
					attribute.Bool("cache.hit", true),
					attribute.String("cache.tier", "disk"))

				// Promote to memory cache
				c.mu.Lock()
				c.cache[sourceURL] = &cachedServiceIndex{
					index:     &index,
					expiresAt: time.Now().Add(ServiceIndexCacheTTL),
				}
				c.mu.Unlock()

				return &index, nil
			}
		}
	}

	span.SetAttributes(
		attribute.Bool("cache.hit", false),
		attribute.String("cache.tier", "none"))

	// Fetch from server
	index, err := c.fetchServiceIndex(ctx, sourceURL)
	if err != nil {
		observability.EndSpanWithError(span, err)
		return nil, err
	}

	// Update memory cache
	c.mu.Lock()
	c.cache[sourceURL] = &cachedServiceIndex{
		index:     index,
		expiresAt: time.Now().Add(ServiceIndexCacheTTL),
	}
	c.mu.Unlock()

	// Update disk cache if available
	if c.diskCache != nil {
		data, err := json.Marshal(index)
		if err == nil {
			// Ignore disk cache write errors (best-effort)
			_ = c.diskCache.Set(ctx, sourceURL, "service_index", bytes.NewReader(data), ServiceIndexCacheTTL, nil)
		}
	}

	return index, nil
}

func (c *ServiceIndexClient) fetchServiceIndex(ctx context.Context, sourceURL string) (*ServiceIndex, error) {
	// Add detailed event to track HTTP fetch timing
	observability.AddEvent(ctx, "fetch_service_index.start", attribute.String("url", sourceURL))

	// Use source URL as-is for NuGet.Client parity
	// Callers must provide full service index URL (e.g., https://api.nuget.org/v3/index.json)
	resp, err := c.httpClient.DoWithRetry(ctx, mustNewRequest("GET", sourceURL, nil))
	if err != nil {
		return nil, fmt.Errorf("fetch service index: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	observability.AddEvent(ctx, "fetch_service_index.http_complete",
		attribute.Int("status_code", resp.StatusCode),
		attribute.String("content_type", resp.Header.Get("Content-Type")))

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("service index returned %d: %s", resp.StatusCode, body)
	}

	var index ServiceIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("decode service index: %w", err)
	}

	observability.AddEvent(ctx, "fetch_service_index.decode_complete",
		attribute.Int("resource_count", len(index.Resources)))

	return &index, nil
}

// GetResourceURL finds the first resource of the given type.
// Matches resource types with or without version suffixes (e.g., "PackageBaseAddress" matches "PackageBaseAddress/3.0.0").
func (c *ServiceIndexClient) GetResourceURL(ctx context.Context, sourceURL, resourceType string) (string, error) {
	index, err := c.GetServiceIndex(ctx, sourceURL)
	if err != nil {
		return "", err
	}

	for _, resource := range index.Resources {
		if matchesResourceType(resource.Type, resourceType) {
			return resource.ID, nil
		}
	}

	return "", fmt.Errorf("resource type %q not found in service index", resourceType)
}

// matchesResourceType returns true if the resource type matches, ignoring version suffixes.
// For example, "PackageBaseAddress/3.0.0" matches "PackageBaseAddress".
func matchesResourceType(actual, requested string) bool {
	if actual == requested {
		return true
	}
	// Check if actual starts with requested followed by a slash (version suffix)
	if len(actual) > len(requested) && actual[:len(requested)] == requested && actual[len(requested)] == '/' {
		return true
	}
	return false
}

// GetAllResourceURLs finds all resources of the given type.
// Matches resource types with or without version suffixes (e.g., "PackageBaseAddress" matches "PackageBaseAddress/3.0.0").
func (c *ServiceIndexClient) GetAllResourceURLs(ctx context.Context, sourceURL, resourceType string) ([]string, error) {
	index, err := c.GetServiceIndex(ctx, sourceURL)
	if err != nil {
		return nil, err
	}

	var urls []string
	for _, resource := range index.Resources {
		if matchesResourceType(resource.Type, resourceType) {
			urls = append(urls, resource.ID)
		}
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("resource type %q not found in service index", resourceType)
	}

	return urls, nil
}

// ClearCache removes all cached service indexes.
func (c *ServiceIndexClient) ClearCache() {
	c.mu.Lock()
	c.cache = make(map[string]*cachedServiceIndex)
	c.mu.Unlock()
}

// mustNewRequest creates an HTTP request, panicking on error for cleaner code.
func mustNewRequest(method, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(fmt.Sprintf("invalid request: %v", err))
	}
	return req
}
