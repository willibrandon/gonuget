package v2

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// FeedClient provides v2 feed detection and access.
type FeedClient struct {
	httpClient *nugethttp.Client
}

// NewFeedClient creates a new v2 feed client.
func NewFeedClient(httpClient *nugethttp.Client) *FeedClient {
	return &FeedClient{
		httpClient: httpClient,
	}
}

// DetectV2Feed checks if a URL is a valid NuGet v2 feed.
// Returns true if the feed is detected, false otherwise.
func (c *FeedClient) DetectV2Feed(ctx context.Context, feedURL string) (bool, error) {
	// Try to fetch the service document
	// V2 feeds typically have a service document at the base URL
	serviceURL := feedURL
	if !strings.HasSuffix(serviceURL, "/") {
		serviceURL += "/"
	}

	req, err := http.NewRequest("GET", serviceURL, nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return false, fmt.Errorf("fetch service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	// Check Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "xml") && !strings.Contains(contentType, "atom") {
		return false, nil
	}

	// Try to parse as service document
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return false, fmt.Errorf("read response: %w", err)
	}

	var service Service
	if err := xml.Unmarshal(body, &service); err != nil {
		// Not a valid service document
		return false, nil
	}

	// Check if it has the expected collections
	for _, collection := range service.Workspace.Collections {
		if strings.Contains(strings.ToLower(collection.Href), "packages") {
			return true, nil
		}
	}

	return false, nil
}

// GetServiceDocument retrieves the OData service document.
func (c *FeedClient) GetServiceDocument(ctx context.Context, feedURL string) (*Service, error) {
	serviceURL := feedURL
	if !strings.HasSuffix(serviceURL, "/") {
		serviceURL += "/"
	}

	req, err := http.NewRequest("GET", serviceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("service returned %d: %s", resp.StatusCode, body)
	}

	var service Service
	if err := xml.NewDecoder(resp.Body).Decode(&service); err != nil {
		return nil, fmt.Errorf("decode service: %w", err)
	}

	return &service, nil
}
