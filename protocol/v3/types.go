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
