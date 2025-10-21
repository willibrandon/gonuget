package v2

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// MetadataClient provides v2 metadata functionality.
type MetadataClient struct {
	httpClient *nugethttp.Client
}

// PackageMetadata represents detailed metadata for a package version.
type PackageMetadata struct {
	ID                       string
	Version                  string
	Description              string
	Authors                  string
	IconURL                  string
	LicenseURL               string
	ProjectURL               string
	Tags                     []string
	Dependencies             string
	DownloadCount            int64
	IsPrerelease             bool
	Published                string
	RequireLicenseAcceptance bool
	DownloadURL              string
	Title                    string
	Updated                  string
}

// NewMetadataClient creates a new v2 metadata client.
func NewMetadataClient(httpClient *nugethttp.Client) *MetadataClient {
	return &MetadataClient{
		httpClient: httpClient,
	}
}

// GetPackageMetadata retrieves metadata for a specific package version.
// Uses the /Packages(Id='...',Version='...') endpoint.
func (c *MetadataClient) GetPackageMetadata(ctx context.Context, feedURL, packageID, version string) (*PackageMetadata, error) {
	// Build metadata URL
	metadataURL, err := c.buildMetadataURL(feedURL, packageID, version)
	if err != nil {
		return nil, fmt.Errorf("build metadata URL: %w", err)
	}

	// Execute request
	req, err := http.NewRequest("GET", metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("metadata request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %q version %q not found", packageID, version)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("metadata returned %d: %s", resp.StatusCode, body)
	}

	// Parse Atom entry response
	var entry Entry
	if err := xml.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("decode entry: %w", err)
	}

	// Convert to PackageMetadata
	metadata := &PackageMetadata{
		ID:                       entry.Properties.ID,
		Version:                  entry.Properties.Version,
		Description:              entry.Properties.Description,
		Authors:                  entry.Properties.Authors,
		IconURL:                  entry.Properties.IconURL,
		LicenseURL:               entry.Properties.LicenseURL,
		ProjectURL:               entry.Properties.ProjectURL,
		Dependencies:             entry.Properties.Dependencies,
		DownloadCount:            entry.Properties.DownloadCount,
		IsPrerelease:             entry.Properties.IsPrerelease,
		Published:                entry.Properties.Published,
		RequireLicenseAcceptance: entry.Properties.RequireLicenseAcceptance,
		DownloadURL:              entry.Content.Src,
		Title:                    entry.Title,
		Updated:                  entry.Updated,
	}

	// Parse tags
	if entry.Properties.Tags != "" {
		metadata.Tags = strings.Split(entry.Properties.Tags, " ")
	}

	return metadata, nil
}

// ListVersions returns all available versions for a package ID.
// Uses the /FindPackagesById() endpoint.
func (c *MetadataClient) ListVersions(ctx context.Context, feedURL, packageID string) ([]string, error) {
	// Build list versions URL
	listURL, err := c.buildListVersionsURL(feedURL, packageID)
	if err != nil {
		return nil, fmt.Errorf("build list versions URL: %w", err)
	}

	// Execute request
	req, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list versions request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %q not found", packageID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("list versions returned %d: %s", resp.StatusCode, body)
	}

	// Parse Atom feed response
	var feed Feed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("decode feed: %w", err)
	}

	// Extract versions from entries
	versions := make([]string, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		if entry.Properties.Version != "" {
			versions = append(versions, entry.Properties.Version)
		}
	}

	return versions, nil
}

func (c *MetadataClient) buildMetadataURL(feedURL, packageID, version string) (string, error) {
	// Ensure feedURL ends with /
	baseURL := feedURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// V2 metadata endpoint: /Packages(Id='...',Version='...')
	// URL encode the ID and version to handle special characters
	encodedID := url.QueryEscape(packageID)
	encodedVersion := url.QueryEscape(version)

	metadataURL := fmt.Sprintf("%sPackages(Id='%s',Version='%s')",
		baseURL, encodedID, encodedVersion)

	return metadataURL, nil
}

func (c *MetadataClient) buildListVersionsURL(feedURL, packageID string) (string, error) {
	// Ensure feedURL ends with /
	baseURL := feedURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// V2 list versions endpoint: /FindPackagesById()?id='...'
	encodedID := url.QueryEscape(packageID)

	listURL := fmt.Sprintf("%sFindPackagesById()?id='%s'",
		baseURL, encodedID)

	return listURL, nil
}
