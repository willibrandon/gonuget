package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/willibrandon/gonuget/auth"
	"github.com/willibrandon/gonuget/cache"
	nugethttp "github.com/willibrandon/gonuget/http"
)

// SourceRepository represents a NuGet package source with authentication
type SourceRepository struct {
	name            string
	sourceURL       string
	authenticator   auth.Authenticator
	httpClient      *nugethttp.Client
	providerFactory *ProviderFactory

	mu       sync.RWMutex
	provider ResourceProvider
}

// RepositoryConfig holds source repository configuration
type RepositoryConfig struct {
	Name          string
	SourceURL     string
	Authenticator auth.Authenticator
	HTTPClient    *nugethttp.Client
	Cache         *cache.MultiTierCache // Optional cache (nil disables caching)
}

// NewSourceRepository creates a new source repository
func NewSourceRepository(cfg RepositoryConfig) *SourceRepository {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = nugethttp.NewClient(nil)
	}

	return &SourceRepository{
		name:            cfg.Name,
		sourceURL:       cfg.SourceURL,
		authenticator:   cfg.Authenticator,
		httpClient:      httpClient,
		providerFactory: NewProviderFactory(httpClient, cfg.Cache),
	}
}

// GetProvider returns the resource provider for this repository
// Lazily initializes and caches the provider
func (r *SourceRepository) GetProvider(ctx context.Context) (ResourceProvider, error) {
	// Check if provider is already cached
	r.mu.RLock()
	if r.provider != nil {
		provider := r.provider
		r.mu.RUnlock()
		return provider, nil
	}
	r.mu.RUnlock()

	// Create provider (with write lock)
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check in case another goroutine created it
	if r.provider != nil {
		return r.provider, nil
	}

	// Create authenticated HTTP client wrapper if needed
	var httpClient HTTPClient = r.httpClient
	if r.authenticator != nil {
		httpClient = r.createAuthenticatedClient()
	}

	// Create new provider factory with authenticated client and cache from existing factory
	factory := NewProviderFactory(httpClient, r.providerFactory.cache)
	provider, err := factory.CreateProvider(ctx, r.sourceURL)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}

	r.provider = provider
	return r.provider, nil
}

// GetMetadata retrieves metadata for a specific package version
// cacheCtx controls caching behavior (can be nil for default behavior)
func (r *SourceRepository) GetMetadata(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID, version string) (*ProtocolMetadata, error) {
	provider, err := r.GetProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.GetMetadata(ctx, cacheCtx, packageID, version)
}

// ListVersions lists all available versions for a package
// cacheCtx controls caching behavior (can be nil for default behavior)
func (r *SourceRepository) ListVersions(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID string) ([]string, error) {
	provider, err := r.GetProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.ListVersions(ctx, cacheCtx, packageID)
}

// Search searches for packages matching the query
// cacheCtx controls caching behavior (can be nil for default behavior)
func (r *SourceRepository) Search(ctx context.Context, cacheCtx *cache.SourceCacheContext, query string, opts SearchOptions) ([]SearchResult, error) {
	provider, err := r.GetProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.Search(ctx, cacheCtx, query, opts)
}

// DownloadPackage downloads a .nupkg file
// cacheCtx controls caching behavior (can be nil for default behavior)
func (r *SourceRepository) DownloadPackage(ctx context.Context, cacheCtx *cache.SourceCacheContext, packageID, version string) (io.ReadCloser, error) {
	provider, err := r.GetProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.DownloadPackage(ctx, cacheCtx, packageID, version)
}

// Name returns the repository name
func (r *SourceRepository) Name() string {
	return r.name
}

// SourceURL returns the source URL
func (r *SourceRepository) SourceURL() string {
	return r.sourceURL
}

// createAuthenticatedClient creates an HTTP client wrapper with authentication
func (r *SourceRepository) createAuthenticatedClient() HTTPClient {
	return &authenticatedHTTPClient{
		base:          r.httpClient,
		authenticator: r.authenticator,
	}
}

// authenticatedHTTPClient wraps nugethttp.Client and applies authentication to all requests
type authenticatedHTTPClient struct {
	base          *nugethttp.Client
	authenticator auth.Authenticator
}

// Do executes an HTTP request with authentication applied
func (c *authenticatedHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Apply authentication to the request
	if err := c.authenticator.Authenticate(req); err != nil {
		return nil, fmt.Errorf("authenticate request: %w", err)
	}

	// Execute the request with the base client
	return c.base.Do(ctx, req)
}

// Get performs an authenticated GET request
func (c *authenticatedHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	return c.Do(ctx, req)
}

// DoWithRetry executes an HTTP request with retry logic and authentication
func (c *authenticatedHTTPClient) DoWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Apply authentication to the request
	if err := c.authenticator.Authenticate(req); err != nil {
		return nil, fmt.Errorf("authenticate request: %w", err)
	}

	// Execute with retry using the base client
	return c.base.DoWithRetry(ctx, req)
}

// SetUserAgent delegates to the base client
func (c *authenticatedHTTPClient) SetUserAgent(ua string) {
	c.base.SetUserAgent(ua)
}

// RepositoryManager manages multiple package sources
type RepositoryManager struct {
	repositories map[string]*SourceRepository
	mu           sync.RWMutex
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager() *RepositoryManager {
	return &RepositoryManager{
		repositories: make(map[string]*SourceRepository),
	}
}

// AddRepository adds a source repository
func (m *RepositoryManager) AddRepository(repo *SourceRepository) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.repositories[repo.name]; exists {
		return fmt.Errorf("repository %q already exists", repo.name)
	}

	m.repositories[repo.name] = repo
	return nil
}

// RemoveRepository removes a source repository by name
func (m *RepositoryManager) RemoveRepository(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.repositories[name]; !exists {
		return fmt.Errorf("repository %q not found", name)
	}

	delete(m.repositories, name)
	return nil
}

// GetRepository returns a repository by name
func (m *RepositoryManager) GetRepository(name string) (*SourceRepository, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	repo, exists := m.repositories[name]
	if !exists {
		return nil, fmt.Errorf("repository %q not found", name)
	}

	return repo, nil
}

// ListRepositories returns all registered repositories
func (m *RepositoryManager) ListRepositories() []*SourceRepository {
	m.mu.RLock()
	defer m.mu.RUnlock()

	repos := make([]*SourceRepository, 0, len(m.repositories))
	for _, repo := range m.repositories {
		repos = append(repos, repo)
	}

	return repos
}

// SearchAll searches for packages across all repositories
// cacheCtx controls caching behavior (can be nil for default behavior)
func (m *RepositoryManager) SearchAll(ctx context.Context, cacheCtx *cache.SourceCacheContext, query string, opts SearchOptions) (map[string][]SearchResult, error) {
	m.mu.RLock()
	repos := make([]*SourceRepository, 0, len(m.repositories))
	for _, repo := range m.repositories {
		repos = append(repos, repo)
	}
	m.mu.RUnlock()

	results := make(map[string][]SearchResult)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errs := make(chan error, len(repos))

	for _, repo := range repos {
		wg.Add(1)
		go func(r *SourceRepository) {
			defer wg.Done()

			res, err := r.Search(ctx, cacheCtx, query, opts)
			if err != nil {
				errs <- fmt.Errorf("%s: %w", r.name, err)
				return
			}

			mu.Lock()
			results[r.name] = res
			mu.Unlock()
		}(repo)
	}

	wg.Wait()
	close(errs)

	// Collect errors
	var firstError error
	for err := range errs {
		if firstError == nil {
			firstError = err
		}
	}

	return results, firstError
}
