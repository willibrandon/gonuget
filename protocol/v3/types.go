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

// RegistrationIndex represents the top-level registration index.
type RegistrationIndex struct {
	Count int                `json:"count"`
	Items []RegistrationPage `json:"items"`
}

// RegistrationPage represents a page of registration entries.
type RegistrationPage struct {
	ID    string             `json:"@id"`
	Count int                `json:"count"`
	Items []RegistrationLeaf `json:"items,omitempty"`
	Lower string             `json:"lower"`
	Upper string             `json:"upper"`
}

// RegistrationLeaf represents a single package version registration.
type RegistrationLeaf struct {
	ID             string               `json:"@id"`
	CatalogEntry   *RegistrationCatalog `json:"catalogEntry"`
	PackageContent string               `json:"packageContent"`
}

// RegistrationCatalog contains detailed package metadata.
type RegistrationCatalog struct {
	ID                       string            `json:"@id"`
	PackageID                string            `json:"id"`
	Version                  string            `json:"version"`
	Authors                  string            `json:"authors,omitempty"`
	Description              string            `json:"description,omitempty"`
	IconURL                  string            `json:"iconUrl,omitempty"`
	LicenseURL               string            `json:"licenseUrl,omitempty"`
	LicenseExpression        string            `json:"licenseExpression,omitempty"`
	ProjectURL               string            `json:"projectUrl,omitempty"`
	Published                string            `json:"published,omitempty"`
	RequireLicenseAcceptance bool              `json:"requireLicenseAcceptance"`
	Summary                  string            `json:"summary,omitempty"`
	Tags                     string            `json:"tags,omitempty"`
	Title                    string            `json:"title,omitempty"`
	DependencyGroups         []DependencyGroup `json:"dependencyGroups,omitempty"`
	PackageTypes             []PackageType     `json:"packageTypes,omitempty"`
}

// DependencyGroup represents dependencies for a specific target framework.
type DependencyGroup struct {
	TargetFramework string       `json:"targetFramework,omitempty"`
	Dependencies    []Dependency `json:"dependencies,omitempty"`
}

// Dependency represents a single package dependency.
type Dependency struct {
	ID    string `json:"id"`
	Range string `json:"range,omitempty"`
}

// PackageType represents the type of package.
type PackageType struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// AutocompleteResponse represents the response from autocomplete API.
type AutocompleteResponse struct {
	TotalHits int         `json:"totalHits"`
	Data      []string    `json:"data"`
	Context   interface{} `json:"@context,omitempty"`
}
