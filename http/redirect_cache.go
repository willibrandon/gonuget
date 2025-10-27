package http

import (
	"context"
	"net/http"
	"sync"
)

var (
	// globalRedirectCacheMu protects the redirect cache
	globalRedirectCacheMu sync.RWMutex
	// globalRedirectCache caches final redirect destinations
	// Key: original URL, Value: final resolved URL
	globalRedirectCache = make(map[string]string)
)

// ResolveRedirect follows redirects and caches the final destination.
// This eliminates redirect overhead on subsequent requests.
// Critical for NuGet V2 which redirects downloads from www.nuget.org to globalcdn.nuget.org
func (c *Client) ResolveRedirect(ctx context.Context, url string) (string, error) {
	// Check cache first
	globalRedirectCacheMu.RLock()
	if cached, exists := globalRedirectCache[url]; exists {
		globalRedirectCacheMu.RUnlock()
		return cached, nil
	}
	globalRedirectCacheMu.RUnlock()

	// Follow redirects to get final URL
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return url, err
	}

	// Custom redirect policy that captures final URL
	var finalURL string
	checkRedirect := func(req *http.Request, via []*http.Request) error {
		finalURL = req.URL.String()
		// Allow up to 10 redirects
		if len(via) >= 10 {
			return http.ErrUseLastResponse
		}
		return nil
	}

	// Create temporary client with custom redirect policy
	tempClient := &http.Client{
		Transport:     c.httpClient.Transport,
		CheckRedirect: checkRedirect,
		Timeout:       c.timeout,
	}

	resp, err := tempClient.Do(req.WithContext(ctx))
	if err != nil && finalURL == "" {
		return url, err
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	// If no redirects occurred, final URL is the original
	if finalURL == "" {
		finalURL = url
	}

	// Cache the result
	globalRedirectCacheMu.Lock()
	globalRedirectCache[url] = finalURL
	globalRedirectCacheMu.Unlock()

	return finalURL, nil
}

// ResetRedirectCache clears the redirect cache (for testing only).
func ResetRedirectCache() {
	globalRedirectCacheMu.Lock()
	defer globalRedirectCacheMu.Unlock()
	globalRedirectCache = make(map[string]string)
}
