package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

var testSearchResponse = &SearchResponse{
	TotalHits: 2,
	Data: []SearchResult{
		{
			PackageID:      "Newtonsoft.Json",
			Version:        "13.0.3",
			Description:    "Json.NET is a popular high-performance JSON framework for .NET",
			Authors:        []string{"James Newton-King"},
			TotalDownloads: 1000000000,
			Verified:       true,
			Tags:           []string{"json", "serialization"},
			Versions: []SearchVersion{
				{Version: "13.0.3", Downloads: 50000000},
				{Version: "13.0.2", Downloads: 45000000},
			},
		},
		{
			PackageID:      "Newtonsoft.Json.Bson",
			Version:        "1.0.2",
			Description:    "Json.NET BSON adds support for reading and writing BSON",
			Authors:        []string{"James Newton-King"},
			TotalDownloads: 10000000,
			Verified:       false,
		},
	},
}

func setupSearchServer(t *testing.T) (*httptest.Server, *SearchClient) {
	mux := http.NewServeMux()

	// Service index endpoint
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		index := &ServiceIndex{
			Version: "3.0.0",
			Resources: []Resource{
				{
					ID:   "http://" + r.Host + "/search",
					Type: ResourceTypeSearchQueryService,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(index)
	})

	// Search endpoint
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		// Validate query parameters
		if q := query.Get("q"); q == "" {
			t.Error("expected 'q' parameter")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(testSearchResponse)
	})

	server := httptest.NewServer(mux)

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	searchClient := NewSearchClient(httpClient, serviceIndexClient)

	return server, searchClient
}

func TestSearchClient_Search(t *testing.T) {
	server, client := setupSearchServer(t)
	defer server.Close()

	ctx := context.Background()

	resp, err := client.Search(ctx, server.URL, SearchOptions{
		Query:      "newtonsoft",
		Take:       20,
		Prerelease: true,
	})

	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if resp.TotalHits != 2 {
		t.Errorf("TotalHits = %d, want 2", resp.TotalHits)
	}

	if len(resp.Data) != 2 {
		t.Errorf("len(Data) = %d, want 2", len(resp.Data))
	}

	first := resp.Data[0]
	if first.PackageID != "Newtonsoft.Json" {
		t.Errorf("PackageID = %q, want Newtonsoft.Json", first.PackageID)
	}

	if first.Version != "13.0.3" {
		t.Errorf("Version = %q, want 13.0.3", first.Version)
	}

	if !first.Verified {
		t.Error("Verified = false, want true")
	}

	if len(first.Versions) != 2 {
		t.Errorf("len(Versions) = %d, want 2", len(first.Versions))
	}
}

func TestSearchClient_SearchWithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/index.json") {
			index := &ServiceIndex{
				Version: "3.0.0",
				Resources: []Resource{
					{
						ID:   "http://" + r.Host + "/search",
						Type: ResourceTypeSearchQueryService,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(index)
			return
		}

		query := r.URL.Query()

		// Validate pagination parameters
		skip := query.Get("skip")
		if skip != "10" {
			t.Errorf("skip = %q, want 10", skip)
		}

		take := query.Get("take")
		if take != "5" {
			t.Errorf("take = %q, want 5", take)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&SearchResponse{TotalHits: 100, Data: []SearchResult{}})
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	client := NewSearchClient(httpClient, serviceIndexClient)

	ctx := context.Background()

	_, err := client.Search(ctx, server.URL, SearchOptions{
		Query: "test",
		Skip:  10,
		Take:  5,
	})

	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
}

func TestSearchClient_SearchSimple(t *testing.T) {
	server, client := setupSearchServer(t)
	defer server.Close()

	ctx := context.Background()

	resp, err := client.SearchSimple(ctx, server.URL, "newtonsoft")
	if err != nil {
		t.Fatalf("SearchSimple() error = %v", err)
	}

	if resp.TotalHits != 2 {
		t.Errorf("TotalHits = %d, want 2", resp.TotalHits)
	}
}
