package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// ServiceIndexClient provides access to NuGet v3 service index.
type ServiceIndexClient struct {
	httpClient *nugethttp.Client

	mu    sync.RWMutex
	cache map[string]*cachedServiceIndex
}

type cachedServiceIndex struct {
	index     *ServiceIndex
	expiresAt time.Time
}

// NewServiceIndexClient creates a new service index client.
func NewServiceIndexClient(httpClient *nugethttp.Client) *ServiceIndexClient {
	return &ServiceIndexClient{
		httpClient: httpClient,
		cache:      make(map[string]*cachedServiceIndex),
	}
}

// GetServiceIndex retrieves the service index for a given source URL.
// Caches the result for ServiceIndexCacheTTL.
func (c *ServiceIndexClient) GetServiceIndex(ctx context.Context, sourceURL string) (*ServiceIndex, error) {
	// Check cache
	c.mu.RLock()
	cached, ok := c.cache[sourceURL]
	c.mu.RUnlock()

	if ok && time.Now().Before(cached.expiresAt) {
		return cached.index, nil
	}

	// Fetch from server
	index, err := c.fetchServiceIndex(ctx, sourceURL)
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.cache[sourceURL] = &cachedServiceIndex{
		index:     index,
		expiresAt: time.Now().Add(ServiceIndexCacheTTL),
	}
	c.mu.Unlock()

	return index, nil
}

func (c *ServiceIndexClient) fetchServiceIndex(ctx context.Context, sourceURL string) (*ServiceIndex, error) {
	// Ensure URL ends with /index.json
	indexURL := sourceURL
	if len(indexURL) > 0 && indexURL[len(indexURL)-1] != '/' {
		indexURL += "/"
	}
	indexURL += "index.json"

	resp, err := c.httpClient.DoWithRetry(ctx, mustNewRequest("GET", indexURL, nil))
	if err != nil {
		return nil, fmt.Errorf("fetch service index: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("service index returned %d: %s", resp.StatusCode, body)
	}

	var index ServiceIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("decode service index: %w", err)
	}

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
