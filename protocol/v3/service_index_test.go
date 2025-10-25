package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	nugethttp "github.com/willibrandon/gonuget/http"
)

var testServiceIndex = &ServiceIndex{
	Version: "3.0.0",
	Resources: []Resource{
		{
			ID:   "https://api.nuget.org/v3/registration5-gz-semver2/",
			Type: ResourceTypeRegistrationsBaseURL,
		},
		{
			ID:   "https://api.nuget.org/v3-flatcontainer/",
			Type: ResourceTypePackageBaseAddress,
		},
		{
			ID:   "https://azuresearch-usnc.nuget.org/query",
			Type: ResourceTypeSearchQueryService,
		},
		{
			ID:   "https://azuresearch-usnc.nuget.org/autocomplete",
			Type: ResourceTypeSearchAutocompleteService,
		},
	},
}

func TestServiceIndexClient_GetServiceIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.json" {
			t.Errorf("Path = %q, want /index.json", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(testServiceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	index, err := client.GetServiceIndex(ctx, server.URL+"/index.json")
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	if index.Version != "3.0.0" {
		t.Errorf("Version = %q, want 3.0.0", index.Version)
	}

	if len(index.Resources) != 4 {
		t.Errorf("Resources count = %d, want 4", len(index.Resources))
	}
}

func TestServiceIndexClient_Cache(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(testServiceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	// First call - should hit server
	_, err := client.GetServiceIndex(ctx, server.URL+"/index.json")
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	// Second call - should use cache
	_, err = client.GetServiceIndex(ctx, server.URL+"/index.json")
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (cache should be used)", callCount)
	}
}

func TestServiceIndexClient_CacheExpiration(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(testServiceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	// First call
	_, err := client.GetServiceIndex(ctx, server.URL+"/index.json")
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	// Manually expire cache
	client.mu.Lock()
	for k := range client.cache {
		client.cache[k].expiresAt = time.Now().Add(-1 * time.Second)
	}
	client.mu.Unlock()

	// Second call - cache expired, should hit server again
	_, err = client.GetServiceIndex(ctx, server.URL+"/index.json")
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (cache expired)", callCount)
	}
}

func TestServiceIndexClient_GetResourceURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(testServiceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	tests := []struct {
		resourceType string
		want         string
		wantErr      bool
	}{
		{
			resourceType: ResourceTypeSearchQueryService,
			want:         "https://azuresearch-usnc.nuget.org/query",
			wantErr:      false,
		},
		{
			resourceType: ResourceTypePackageBaseAddress,
			want:         "https://api.nuget.org/v3-flatcontainer/",
			wantErr:      false,
		},
		{
			resourceType: "NonExistentType",
			want:         "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			got, err := client.GetResourceURL(ctx, server.URL+"/index.json", tt.resourceType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetResourceURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetResourceURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceIndexClient_GetAllResourceURLs(t *testing.T) {
	multiResourceIndex := &ServiceIndex{
		Version: "3.0.0",
		Resources: []Resource{
			{
				ID:   "https://search1.nuget.org/query",
				Type: ResourceTypeSearchQueryService,
			},
			{
				ID:   "https://search2.nuget.org/query",
				Type: ResourceTypeSearchQueryService,
			},
			{
				ID:   "https://api.nuget.org/v3-flatcontainer/",
				Type: ResourceTypePackageBaseAddress,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(multiResourceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	urls, err := client.GetAllResourceURLs(ctx, server.URL+"/index.json", ResourceTypeSearchQueryService)
	if err != nil {
		t.Fatalf("GetAllResourceURLs() error = %v", err)
	}

	if len(urls) != 2 {
		t.Errorf("len(urls) = %d, want 2", len(urls))
	}

	expected := map[string]bool{
		"https://search1.nuget.org/query": true,
		"https://search2.nuget.org/query": true,
	}

	for _, url := range urls {
		if !expected[url] {
			t.Errorf("unexpected URL: %q", url)
		}
	}
}

func TestServiceIndexClient_ClearCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(testServiceIndex)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewServiceIndexClient(httpClient)
	ctx := context.Background()

	// Populate cache
	_, err := client.GetServiceIndex(ctx, server.URL+"/index.json")
	if err != nil {
		t.Fatalf("GetServiceIndex() error = %v", err)
	}

	if len(client.cache) == 0 {
		t.Error("cache should not be empty")
	}

	// Clear cache
	client.ClearCache()

	if len(client.cache) != 0 {
		t.Errorf("cache size = %d, want 0 after clear", len(client.cache))
	}
}
