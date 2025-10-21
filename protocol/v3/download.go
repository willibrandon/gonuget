package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// DownloadClient provides package download functionality.
type DownloadClient struct {
	httpClient         *nugethttp.Client
	serviceIndexClient *ServiceIndexClient
}

// NewDownloadClient creates a new download client.
func NewDownloadClient(httpClient *nugethttp.Client, serviceIndexClient *ServiceIndexClient) *DownloadClient {
	return &DownloadClient{
		httpClient:         httpClient,
		serviceIndexClient: serviceIndexClient,
	}
}

// DownloadPackage downloads a .nupkg file and returns the response body.
// Caller is responsible for closing the response body.
func (c *DownloadClient) DownloadPackage(ctx context.Context, sourceURL, packageID, version string) (io.ReadCloser, error) {
	// Get package base address from service index
	baseURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypePackageBaseAddress)
	if err != nil {
		return nil, fmt.Errorf("get package base URL: %w", err)
	}

	// Build download URL
	// Format: {baseURL}/{packageID}/{version}/{packageID}.{version}.nupkg
	packageIDLower := strings.ToLower(packageID)
	versionLower := strings.ToLower(version)
	downloadURL := fmt.Sprintf("%s/%s/%s/%s.%s.nupkg",
		strings.TrimSuffix(baseURL, "/"),
		packageIDLower,
		versionLower,
		packageIDLower,
		versionLower,
	)

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
		return nil, fmt.Errorf("package %s %s not found", packageID, version)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download returned %d: %s", resp.StatusCode, body)
	}

	return resp.Body, nil
}

// DownloadNuspec downloads the .nuspec manifest file for a package.
// Caller is responsible for closing the response body.
func (c *DownloadClient) DownloadNuspec(ctx context.Context, sourceURL, packageID, version string) (io.ReadCloser, error) {
	// Get package base address from service index
	baseURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypePackageBaseAddress)
	if err != nil {
		return nil, fmt.Errorf("get package base URL: %w", err)
	}

	// Build nuspec URL
	// Format: {baseURL}/{packageID}/{version}/{packageID}.nuspec
	packageIDLower := strings.ToLower(packageID)
	versionLower := strings.ToLower(version)
	nuspecURL := fmt.Sprintf("%s/%s/%s/%s.nuspec",
		strings.TrimSuffix(baseURL, "/"),
		packageIDLower,
		versionLower,
		packageIDLower,
	)

	// Execute request
	req, err := http.NewRequest("GET", nuspecURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("nuspec request: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("nuspec for %s %s not found", packageID, version)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("nuspec returned %d: %s", resp.StatusCode, body)
	}

	return resp.Body, nil
}

// GetPackageVersions lists all available versions for a package.
// Uses the package base address versions endpoint.
func (c *DownloadClient) GetPackageVersions(ctx context.Context, sourceURL, packageID string) ([]string, error) {
	// Get package base address from service index
	baseURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypePackageBaseAddress)
	if err != nil {
		return nil, fmt.Errorf("get package base URL: %w", err)
	}

	// Build versions URL
	// Format: {baseURL}/{packageID}/index.json
	packageIDLower := strings.ToLower(packageID)
	versionsURL := fmt.Sprintf("%s/%s/index.json",
		strings.TrimSuffix(baseURL, "/"),
		packageIDLower,
	)

	// Execute request
	req, err := http.NewRequest("GET", versionsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("versions request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %s not found", packageID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("versions returned %d: %s", resp.StatusCode, body)
	}

	// Parse versions response
	var versionsResp struct {
		Versions []string `json:"versions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&versionsResp); err != nil {
		return nil, fmt.Errorf("decode versions: %w", err)
	}

	return versionsResp.Versions, nil
}
