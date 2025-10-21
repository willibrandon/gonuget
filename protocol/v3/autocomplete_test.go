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

func setupAutocompleteServer() (*httptest.Server, *AutocompleteClient) {
	mux := http.NewServeMux()

	// Service index endpoint
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		index := &ServiceIndex{
			Version: "3.0.0",
			Resources: []Resource{
				{
					ID:   "http://" + r.Host + "/autocomplete",
					Type: ResourceTypeSearchAutocompleteService,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	})

	// Autocomplete endpoint
	mux.HandleFunc("/autocomplete", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		packageID := query.Get("id")

		// Package ID autocomplete
		if packageID == "" {
			q := query.Get("q")
			w.Header().Set("Content-Type", "application/json")

			if q == "newtonsoft" {
				json.NewEncoder(w).Encode(&AutocompleteResponse{
					TotalHits: 2,
					Data: []string{
						"Newtonsoft.Json",
						"Newtonsoft.Json.Schema",
					},
				})
				return
			}

			// Empty query
			json.NewEncoder(w).Encode(&AutocompleteResponse{
				TotalHits: 0,
				Data:      []string{},
			})
			return
		}

		// Package version autocomplete
		if strings.ToLower(packageID) == "newtonsoft.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&AutocompleteResponse{
				TotalHits: 3,
				Data: []string{
					"13.0.1",
					"13.0.2",
					"13.0.3",
				},
			})
			return
		}

		// Package not found
		http.NotFound(w, r)
	})

	server := httptest.NewServer(mux)

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	autocompleteClient := NewAutocompleteClient(httpClient, serviceIndexClient)

	return server, autocompleteClient
}

func TestAutocompleteClient_AutocompletePackageIDs(t *testing.T) {
	server, client := setupAutocompleteServer()
	defer server.Close()

	ctx := context.Background()

	result, err := client.AutocompletePackageIDs(ctx, server.URL, "newtonsoft", 0, 20, false)
	if err != nil {
		t.Fatalf("AutocompletePackageIDs() error = %v", err)
	}

	if result.TotalHits != 2 {
		t.Errorf("TotalHits = %d, want 2", result.TotalHits)
	}

	if len(result.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(result.Data))
	}

	expected := []string{"Newtonsoft.Json", "Newtonsoft.Json.Schema"}
	for i, want := range expected {
		if result.Data[i] != want {
			t.Errorf("Data[%d] = %q, want %q", i, result.Data[i], want)
		}
	}
}

func TestAutocompleteClient_AutocompletePackageIDs_EmptyQuery(t *testing.T) {
	server, client := setupAutocompleteServer()
	defer server.Close()

	ctx := context.Background()

	result, err := client.AutocompletePackageIDs(ctx, server.URL, "", 0, 20, false)
	if err != nil {
		t.Fatalf("AutocompletePackageIDs() error = %v", err)
	}

	if result.TotalHits != 0 {
		t.Errorf("TotalHits = %d, want 0", result.TotalHits)
	}

	if len(result.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0", len(result.Data))
	}
}

func TestAutocompleteClient_AutocompletePackageIDs_WithPagination(t *testing.T) {
	server, client := setupAutocompleteServer()
	defer server.Close()

	ctx := context.Background()

	result, err := client.AutocompletePackageIDs(ctx, server.URL, "newtonsoft", 10, 5, true)
	if err != nil {
		t.Fatalf("AutocompletePackageIDs() error = %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestAutocompleteClient_AutocompletePackageVersions(t *testing.T) {
	server, client := setupAutocompleteServer()
	defer server.Close()

	ctx := context.Background()

	result, err := client.AutocompletePackageVersions(ctx, server.URL, "Newtonsoft.Json", false)
	if err != nil {
		t.Fatalf("AutocompletePackageVersions() error = %v", err)
	}

	if result.TotalHits != 3 {
		t.Errorf("TotalHits = %d, want 3", result.TotalHits)
	}

	if len(result.Data) != 3 {
		t.Fatalf("len(Data) = %d, want 3", len(result.Data))
	}

	expected := []string{"13.0.1", "13.0.2", "13.0.3"}
	for i, want := range expected {
		if result.Data[i] != want {
			t.Errorf("Data[%d] = %q, want %q", i, result.Data[i], want)
		}
	}
}

func TestAutocompleteClient_AutocompletePackageVersions_NotFound(t *testing.T) {
	server, client := setupAutocompleteServer()
	defer server.Close()

	ctx := context.Background()

	_, err := client.AutocompletePackageVersions(ctx, server.URL, "NonExistent.Package", false)
	if err == nil {
		t.Error("expected error for non-existent package")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}
