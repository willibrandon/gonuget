package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// AutocompleteClient provides package ID and version autocomplete functionality.
type AutocompleteClient struct {
	httpClient         *nugethttp.Client
	serviceIndexClient *ServiceIndexClient
}

// NewAutocompleteClient creates a new autocomplete client.
func NewAutocompleteClient(httpClient *nugethttp.Client, serviceIndexClient *ServiceIndexClient) *AutocompleteClient {
	return &AutocompleteClient{
		httpClient:         httpClient,
		serviceIndexClient: serviceIndexClient,
	}
}

// AutocompletePackageIDs returns package ID suggestions for a given query.
func (c *AutocompleteClient) AutocompletePackageIDs(ctx context.Context, sourceURL, query string, skip, take int, prerelease bool) (*AutocompleteResponse, error) {
	// Get autocomplete service URL from service index
	baseURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypeSearchAutocompleteService)
	if err != nil {
		return nil, fmt.Errorf("get autocomplete URL: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	if skip > 0 {
		params.Set("skip", strconv.Itoa(skip))
	}
	if take > 0 {
		params.Set("take", strconv.Itoa(take))
	} else {
		params.Set("take", "20")
	}
	params.Set("prerelease", strconv.FormatBool(prerelease))

	// Build full URL
	fullURL := baseURL
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
		return nil, fmt.Errorf("autocomplete request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("autocomplete returned %d: %s", resp.StatusCode, body)
	}

	// Parse response
	var result AutocompleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode autocomplete response: %w", err)
	}

	return &result, nil
}

// AutocompletePackageVersions returns version suggestions for a given package ID.
func (c *AutocompleteClient) AutocompletePackageVersions(ctx context.Context, sourceURL, packageID string, prerelease bool) (*AutocompleteResponse, error) {
	// Get autocomplete service URL from service index
	baseURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypeSearchAutocompleteService)
	if err != nil {
		return nil, fmt.Errorf("get autocomplete URL: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	params.Set("id", packageID)
	params.Set("prerelease", strconv.FormatBool(prerelease))

	// Build full URL
	packageIDLower := strings.ToLower(packageID)
	fullURL := strings.TrimSuffix(baseURL, "/") + "?id=" + url.QueryEscape(packageIDLower)
	if prerelease {
		fullURL += "&prerelease=true"
	} else {
		fullURL += "&prerelease=false"
	}

	// Execute request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("version autocomplete request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %q not found", packageID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("version autocomplete returned %d: %s", resp.StatusCode, body)
	}

	// Parse response
	var result AutocompleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode version autocomplete response: %w", err)
	}

	return &result, nil
}
