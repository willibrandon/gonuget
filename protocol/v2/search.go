package v2

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// SearchClient provides v2 search functionality.
type SearchClient struct {
	httpClient *nugethttp.Client
}

// SearchOptions holds v2 search parameters.
type SearchOptions struct {
	Query             string
	Skip              int
	Top               int
	Filter            string
	OrderBy           string
	IncludePrerelease bool
}

// SearchResult represents a v2 search result.
type SearchResult struct {
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
}

// NewSearchClient creates a new v2 search client.
func NewSearchClient(httpClient *nugethttp.Client) *SearchClient {
	return &SearchClient{
		httpClient: httpClient,
	}
}

// Search searches for packages using OData query syntax.
func (c *SearchClient) Search(ctx context.Context, feedURL string, opts SearchOptions) ([]SearchResult, error) {
	// Build search URL with OData parameters
	searchURL, err := c.buildSearchURL(feedURL, opts)
	if err != nil {
		return nil, fmt.Errorf("build search URL: %w", err)
	}

	// Execute search request
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.DoWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("search returned %d: %s", resp.StatusCode, body)
	}

	// Parse Atom feed response
	var feed Feed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("decode feed: %w", err)
	}

	// Convert entries to search results
	results := make([]SearchResult, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		result := SearchResult{
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
		}

		// Parse tags
		if entry.Properties.Tags != "" {
			result.Tags = strings.Split(entry.Properties.Tags, " ")
		}

		results = append(results, result)
	}

	return results, nil
}

func (c *SearchClient) buildSearchURL(feedURL string, opts SearchOptions) (string, error) {
	// Ensure feedURL ends with /
	baseURL := feedURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// V2 search endpoint: /Packages() or /Search()
	// Using /Packages() with $filter for compatibility
	searchURL := baseURL + "Packages()"

	// Build OData query parameters
	params := url.Values{}

	// $filter - combine query and prerelease filter
	var filters []string
	if opts.Query != "" {
		// Search in Id and Description
		filter := fmt.Sprintf("(substringof('%s',tolower(Id)) or substringof('%s',tolower(Description)))",
			strings.ToLower(opts.Query),
			strings.ToLower(opts.Query))
		filters = append(filters, filter)
	}

	if !opts.IncludePrerelease {
		filters = append(filters, "IsPrerelease eq false")
	}

	if opts.Filter != "" {
		filters = append(filters, opts.Filter)
	}

	if len(filters) > 0 {
		params.Set("$filter", strings.Join(filters, " and "))
	}

	// $orderby
	if opts.OrderBy != "" {
		params.Set("$orderby", opts.OrderBy)
	} else {
		params.Set("$orderby", "DownloadCount desc")
	}

	// $skip
	if opts.Skip > 0 {
		params.Set("$skip", strconv.Itoa(opts.Skip))
	}

	// $top
	if opts.Top > 0 {
		params.Set("$top", strconv.Itoa(opts.Top))
	} else {
		params.Set("$top", "20") // Default
	}

	// Build final URL
	if len(params) > 0 {
		searchURL += "?" + params.Encode()
	}

	return searchURL, nil
}

// FindPackagesById searches for all versions of a specific package ID.
func (c *SearchClient) FindPackagesById(ctx context.Context, feedURL, packageID string) ([]SearchResult, error) {
	return c.Search(ctx, feedURL, SearchOptions{
		Filter:            fmt.Sprintf("Id eq '%s'", packageID),
		OrderBy:           "Version desc",
		Top:               100,
		IncludePrerelease: true,
	})
}
