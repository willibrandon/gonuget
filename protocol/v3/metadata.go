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

// MetadataClient provides package metadata functionality.
type MetadataClient struct {
	httpClient         *nugethttp.Client
	serviceIndexClient *ServiceIndexClient
}

// NewMetadataClient creates a new metadata client.
func NewMetadataClient(httpClient *nugethttp.Client, serviceIndexClient *ServiceIndexClient) *MetadataClient {
	return &MetadataClient{
		httpClient:         httpClient,
		serviceIndexClient: serviceIndexClient,
	}
}

// GetPackageMetadata retrieves metadata for a specific package ID.
// Returns all versions and their metadata.
func (c *MetadataClient) GetPackageMetadata(ctx context.Context, sourceURL, packageID string) (*RegistrationIndex, error) {
	// Get registration base URL from service index
	baseURL, err := c.serviceIndexClient.GetResourceURL(ctx, sourceURL, ResourceTypeRegistrationsBaseUrl)
	if err != nil {
		return nil, fmt.Errorf("get registration URL: %w", err)
	}

	// Build registration index URL
	// Format: {baseURL}/{packageID}/index.json
	packageIDLower := strings.ToLower(packageID)
	registrationURL := strings.TrimSuffix(baseURL, "/") + "/" + packageIDLower + "/index.json"

	// Fetch registration index
	req, err := http.NewRequest("GET", registrationURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch registration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %q not found", packageID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("registration returned %d: %s", resp.StatusCode, body)
	}

	var index RegistrationIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("decode registration: %w", err)
	}

	// Fetch inline pages if items are not populated
	for i := range index.Items {
		if len(index.Items[i].Items) == 0 && index.Items[i].ID != "" {
			page, err := c.fetchRegistrationPage(ctx, index.Items[i].ID)
			if err != nil {
				return nil, fmt.Errorf("fetch page %s: %w", index.Items[i].ID, err)
			}
			index.Items[i] = *page
		}
	}

	return &index, nil
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
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("page returned %d: %s", resp.StatusCode, body)
	}

	var page RegistrationPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("decode page: %w", err)
	}

	return &page, nil
}
