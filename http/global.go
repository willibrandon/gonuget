package http

import (
	"sync"
)

var (
	// globalClientOnce ensures global client is created only once
	globalClientOnce sync.Once
	// globalClient is the shared HTTP client instance used across all operations
	globalClient *Client
)

// GetGlobalClient returns the global singleton HTTP client.
// This client is shared across all NuGet operations for maximum connection reuse.
// Matches NuGet.Client's HttpHandlerResourceV3Provider behavior where a single
// HttpClient instance is reused across all package sources.
func GetGlobalClient() *Client {
	globalClientOnce.Do(func() {
		globalClient = NewClient(nil) // Use default configuration
	})
	return globalClient
}

// ResetGlobalClient resets the global client (for testing only).
// WARNING: This should only be used in tests.
func ResetGlobalClient() {
	globalClientOnce = sync.Once{}
	globalClient = nil
}
