package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// SearchClient provides package search functionality.
type SearchClient struct {
	httpClient         *nugethttp.Client
	serviceIndexClient *ServiceIndexClient
}

// SearchOptions holds search parameters.
type SearchOptions struct {
	Query       string
	Skip        int
	Take        int
	Prerelease  bool
	SemVerLevel string // "2.0.0" for SemVer 2.0 support
}

// NewSearchClient creates a new search client.
func NewSearchClient(httpClient *nugethttp.Client, serviceIndexClient *ServiceIndexClient) *SearchClient {
	return &SearchClient{
		httpClient:         httpClient,
		serviceIndexClient: serviceIndexClient,
	}
}

// Search searches for packages matching the query.
func (c *SearchClient) Search(ctx context.Context, sourceURL string, opts SearchOptions) (*SearchResponse, error) {
	// Get search endpoint from service index
	searchURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypeSearchQueryService)
	if err != nil {
		return nil, fmt.Errorf("get search URL: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	if opts.Query != "" {
		params.Set("q", opts.Query)
	}
	if opts.Skip > 0 {
		params.Set("skip", strconv.Itoa(opts.Skip))
	}
	if opts.Take > 0 {
		params.Set("take", strconv.Itoa(opts.Take))
	} else {
		params.Set("take", "20") // Default
	}
	params.Set("prerelease", strconv.FormatBool(opts.Prerelease))
	if opts.SemVerLevel != "" {
		params.Set("semVerLevel", opts.SemVerLevel)
	}

	// Build full URL
	fullURL := searchURL
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	// Execute request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("search returned %d: %s", resp.StatusCode, body)
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	return &searchResp, nil
}

// SearchSimple performs a simple search with default options.
func (c *SearchClient) SearchSimple(ctx context.Context, sourceURL, query string) (*SearchResponse, error) {
	return c.Search(ctx, sourceURL, SearchOptions{
		Query:      query,
		Take:       20,
		Prerelease: true,
	})
}
