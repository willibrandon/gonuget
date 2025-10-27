package v2

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// DownloadClient provides v2 package download functionality.
type DownloadClient struct {
	httpClient *nugethttp.Client
}

// NewDownloadClient creates a new v2 download client.
func NewDownloadClient(httpClient *nugethttp.Client) *DownloadClient {
	return &DownloadClient{
		httpClient: httpClient,
	}
}

// DownloadPackage downloads a .nupkg file and returns the response body.
// Caller is responsible for closing the response body.
func (c *DownloadClient) DownloadPackage(ctx context.Context, feedURL, packageID, version string) (io.ReadCloser, error) {
	// Build download URL
	// Format: /package/{id}/{version}
	downloadURL, err := c.buildDownloadURL(feedURL, packageID, version)
	if err != nil {
		return nil, fmt.Errorf("build download URL: %w", err)
	}

	// Check redirect cache first (eliminates V2→CDN redirect on fresh processes!)
	// V2 downloads redirect from www.nuget.org to globalcdn.nuget.org
	if cachedURL, found := nugethttp.GetCachedRedirect(downloadURL); found {
		downloadURL = cachedURL
	}

	// Execute download request
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("download request: %w", err)
	}

	// Cache redirect if one occurred (for next fresh process)
	if resp.Request.URL.String() != downloadURL {
		_ = nugethttp.SetCachedRedirect(downloadURL, resp.Request.URL.String())
	}

	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("package %s %s not found", packageID, version)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download returned %d: %s", resp.StatusCode, body)
	}

	return resp.Body, nil
}

// DownloadLatestPackage downloads the latest version of a package.
// Caller is responsible for closing the response body.
func (c *DownloadClient) DownloadLatestPackage(ctx context.Context, feedURL, packageID string) (io.ReadCloser, error) {
	// Build latest download URL
	// Format: /package/{id}
	downloadURL, err := c.buildLatestDownloadURL(feedURL, packageID)
	if err != nil {
		return nil, fmt.Errorf("build download URL: %w", err)
	}

	// Execute download request
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("download request: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("package %s not found", packageID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download returned %d: %s", resp.StatusCode, body)
	}

	return resp.Body, nil
}

func (c *DownloadClient) buildDownloadURL(feedURL, packageID, version string) (string, error) {
	baseURL := feedURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// V2 download endpoint: /package/{id}/{version}
	downloadURL := fmt.Sprintf("%spackage/%s/%s", baseURL, packageID, version)

	return downloadURL, nil
}

func (c *DownloadClient) buildLatestDownloadURL(feedURL, packageID string) (string, error) {
	baseURL := feedURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// V2 latest download endpoint: /package/{id}
	downloadURL := fmt.Sprintf("%spackage/%s", baseURL, packageID)

	return downloadURL, nil
}
