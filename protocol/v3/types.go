// Package v3 implements the NuGet v3 protocol.
//
// It provides service index discovery, package search, metadata access,
// and package download functionality for NuGet v3 feeds.
package v3

import (
	"time"
)

// ServiceIndex represents the NuGet v3 service index.
// See: https://docs.microsoft.com/en-us/nuget/api/service-index
type ServiceIndex struct {
	Version   string      `json:"version"`
	Resources []Resource  `json:"resources"`
	Context   interface{} `json:"@context,omitempty"`
}

// Resource represents a service resource in the service index.
type Resource struct {
	ID      string `json:"@id"`
	Type    string `json:"@type"`
	Comment string `json:"comment,omitempty"`
}

// Well-known resource types
const (
	// Search
	ResourceTypeSearchQueryService        = "SearchQueryService"
	ResourceTypeSearchAutocompleteService = "SearchAutocompleteService"

	// Registration (metadata)
	ResourceTypeRegistrationsBaseUrl = "RegistrationsBaseUrl"

	// Package download
	ResourceTypePackageBaseAddress = "PackageBaseAddress"

	// Package publish
	ResourceTypePackagePublish = "PackagePublish"

	// Catalog
	ResourceTypeCatalog = "Catalog/3.0.0"
)

// ServiceIndexCacheTTL is the default service index cache TTL (40 minutes as per NuGet spec).
const ServiceIndexCacheTTL = 40 * time.Minute

// SearchResponse represents the response from the search API.
type SearchResponse struct {
	TotalHits int            `json:"totalHits"`
	Data      []SearchResult `json:"data"`
	Context   interface{}    `json:"@context,omitempty"`
}

// SearchResult represents a single search result.
type SearchResult struct {
	ID             string          `json:"@id"`
	Type           string          `json:"@type"`
	Registration   string          `json:"registration,omitempty"`
	PackageID      string          `json:"id"`
	Version        string          `json:"version"`
	Description    string          `json:"description"`
	Summary        string          `json:"summary,omitempty"`
	Title          string          `json:"title,omitempty"`
	IconURL        string          `json:"iconUrl,omitempty"`
	LicenseURL     string          `json:"licenseUrl,omitempty"`
	ProjectURL     string          `json:"projectUrl,omitempty"`
	Tags           []string        `json:"tags,omitempty"`
	Authors        []string        `json:"authors,omitempty"`
	TotalDownloads int64           `json:"totalDownloads"`
	Verified       bool            `json:"verified"`
	Versions       []SearchVersion `json:"versions,omitempty"`
}

// SearchVersion represents a version in search results.
type SearchVersion struct {
	Version   string `json:"version"`
	Downloads int64  `json:"downloads"`
	ID        string `json:"@id"`
}
