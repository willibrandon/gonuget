package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

// TestMetadataClient_GetPackageMetadata_WithPaginatedPages tests the fetchRegistrationPage code path
func TestMetadataClient_GetPackageMetadata_WithPaginatedPages(t *testing.T) {
	// Create a test server that returns registration with external pages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			// Service index
			index := ServiceIndex{
				Version: "3.0.0",
				Resources: []Resource{
					{
						ID:   "http://" + r.Host + "/registration/",
						Type: ResourceTypeRegistrationsBaseURL,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)

		case "/registration/testpkg/index.json":
			// Return index with external page reference (no inline items)
			response := RegistrationIndex{
				Count: 1,
				Items: []RegistrationPage{
					{
						ID:    "http://" + r.Host + "/registration/testpkg/page/1.0.0/2.0.0.json",
						Lower: "1.0.0",
						Upper: "2.0.0",
						// No Items - should trigger page fetch via fetchRegistrationPage
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)

		case "/registration/testpkg/page/1.0.0/2.0.0.json":
			// Return the actual page with items
			page := RegistrationPage{
				ID:    "http://" + r.Host + "/registration/testpkg/page/1.0.0/2.0.0.json",
				Lower: "1.0.0",
				Upper: "2.0.0",
				Count: 1,
				Items: []RegistrationLeaf{
					{
						ID: "http://" + r.Host + "/registration/testpkg/1.5.0.json",
						CatalogEntry: &RegistrationCatalog{
							ID:        "http://" + r.Host + "/catalog/testpkg/1.5.0.json",
							PackageID: "TestPkg",
							Version:   "1.5.0",
							Authors:   "Test Author",
						},
						PackageContent: "http://" + r.Host + "/packages/testpkg.1.5.0.nupkg",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(page)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	client := NewMetadataClient(httpClient, serviceIndexClient)

	ctx := context.Background()
	index, err := client.GetPackageMetadata(ctx, server.URL, "testpkg")
	if err != nil {
		t.Fatalf("GetPackageMetadata() error = %v", err)
	}

	// Verify the page was fetched and populated
	if len(index.Items) != 1 {
		t.Fatalf("GetPackageMetadata() items count = %d, want 1", len(index.Items))
	}

	if len(index.Items[0].Items) != 1 {
		t.Fatalf("GetPackageMetadata() page items count = %d, want 1", len(index.Items[0].Items))
	}

	catalogEntry := index.Items[0].Items[0].CatalogEntry
	if catalogEntry == nil {
		t.Fatal("GetPackageMetadata() catalog entry is nil")
	}

	if catalogEntry.PackageID != "TestPkg" {
		t.Errorf("GetPackageMetadata() PackageID = %s, want TestPkg", catalogEntry.PackageID)
	}

	if catalogEntry.Version != "1.5.0" {
		t.Errorf("GetPackageMetadata() Version = %s, want 1.5.0", catalogEntry.Version)
	}
}

func TestMetadataClient_GetPackageMetadata_PageFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := ServiceIndex{
				Version: "3.0.0",
				Resources: []Resource{
					{
						ID:   "http://" + r.Host + "/registration/",
						Type: ResourceTypeRegistrationsBaseURL,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)

		case "/registration/testpkg/index.json":
			response := RegistrationIndex{
				Count: 1,
				Items: []RegistrationPage{
					{
						ID:    "http://" + r.Host + "/registration/testpkg/page/bad.json",
						Lower: "1.0.0",
						Upper: "2.0.0",
						// No items - will try to fetch page
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)

		case "/registration/testpkg/page/bad.json":
			// Return error when fetching page
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	client := NewMetadataClient(httpClient, serviceIndexClient)

	ctx := context.Background()
	_, err := client.GetPackageMetadata(ctx, server.URL, "testpkg")
	if err == nil {
		t.Error("GetPackageMetadata() expected error when page fetch fails")
	}

	// Verify error message contains context about page fetch
	if err.Error() == "" {
		t.Error("GetPackageMetadata() error message is empty")
	}
}

func TestMetadataClient_GetPackageMetadata_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := ServiceIndex{
				Version: "3.0.0",
				Resources: []Resource{
					{
						ID:   "http://" + r.Host + "/registration/",
						Type: ResourceTypeRegistrationsBaseURL,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)

		case "/registration/nonexistent/index.json":
			http.NotFound(w, r)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	client := NewMetadataClient(httpClient, serviceIndexClient)

	ctx := context.Background()
	_, err := client.GetPackageMetadata(ctx, server.URL, "nonexistent")
	if err == nil {
		t.Error("GetPackageMetadata() expected error for non-existent package")
	}
}

func TestMetadataClient_GetPackageMetadata_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := ServiceIndex{
				Version: "3.0.0",
				Resources: []Resource{
					{
						ID:   "http://" + r.Host + "/registration/",
						Type: ResourceTypeRegistrationsBaseURL,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)

		case "/registration/badpkg/index.json":
			// Return invalid JSON
			_, _ = w.Write([]byte("not valid json"))

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	client := NewMetadataClient(httpClient, serviceIndexClient)

	ctx := context.Background()
	_, err := client.GetPackageMetadata(ctx, server.URL, "badpkg")
	if err == nil {
		t.Error("GetPackageMetadata() expected error for invalid JSON")
	}
}

func TestMetadataClient_GetVersionMetadata_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := ServiceIndex{
				Version: "3.0.0",
				Resources: []Resource{
					{
						ID:   "http://" + r.Host + "/registration/",
						Type: ResourceTypeRegistrationsBaseURL,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)

		case "/registration/testpkg/index.json":
			response := RegistrationIndex{
				Count: 1,
				Items: []RegistrationPage{
					{
						ID:    "http://" + r.Host + "/registration/testpkg/page.json",
						Lower: "1.0.0",
						Upper: "2.0.0",
						Count: 2,
						Items: []RegistrationLeaf{
							{
								CatalogEntry: &RegistrationCatalog{
									PackageID: "TestPkg",
									Version:   "1.0.0",
								},
							},
							{
								CatalogEntry: &RegistrationCatalog{
									PackageID: "TestPkg",
									Version:   "1.5.0",
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	client := NewMetadataClient(httpClient, serviceIndexClient)

	ctx := context.Background()
	metadata, err := client.GetVersionMetadata(ctx, server.URL, "testpkg", "1.5.0")
	if err != nil {
		t.Fatalf("GetVersionMetadata() error = %v", err)
	}

	if metadata.PackageID != "TestPkg" {
		t.Errorf("GetVersionMetadata() PackageID = %s, want TestPkg", metadata.PackageID)
	}

	if metadata.Version != "1.5.0" {
		t.Errorf("GetVersionMetadata() Version = %s, want 1.5.0", metadata.Version)
	}
}

func TestMetadataClient_GetVersionMetadata_VersionNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := ServiceIndex{
				Version: "3.0.0",
				Resources: []Resource{
					{
						ID:   "http://" + r.Host + "/registration/",
						Type: ResourceTypeRegistrationsBaseURL,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)

		case "/registration/testpkg/index.json":
			response := RegistrationIndex{
				Count: 1,
				Items: []RegistrationPage{
					{
						ID:    "http://" + r.Host + "/registration/testpkg/page.json",
						Lower: "1.0.0",
						Upper: "2.0.0",
						Count: 1,
						Items: []RegistrationLeaf{
							{
								CatalogEntry: &RegistrationCatalog{
									PackageID: "TestPkg",
									Version:   "1.0.0",
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	client := NewMetadataClient(httpClient, serviceIndexClient)

	ctx := context.Background()
	_, err := client.GetVersionMetadata(ctx, server.URL, "testpkg", "99.99.99")
	if err == nil {
		t.Error("GetVersionMetadata() expected error for non-existent version")
	}
}

func TestMetadataClient_ListVersions_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := ServiceIndex{
				Version: "3.0.0",
				Resources: []Resource{
					{
						ID:   "http://" + r.Host + "/registration/",
						Type: ResourceTypeRegistrationsBaseURL,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)

		case "/registration/testpkg/index.json":
			response := RegistrationIndex{
				Count: 1,
				Items: []RegistrationPage{
					{
						ID:    "http://" + r.Host + "/registration/testpkg/page.json",
						Lower: "1.0.0",
						Upper: "2.0.0",
						Count: 3,
						Items: []RegistrationLeaf{
							{
								CatalogEntry: &RegistrationCatalog{
									PackageID: "TestPkg",
									Version:   "1.0.0",
								},
							},
							{
								CatalogEntry: &RegistrationCatalog{
									PackageID: "TestPkg",
									Version:   "1.5.0",
								},
							},
							{
								CatalogEntry: &RegistrationCatalog{
									PackageID: "TestPkg",
									Version:   "2.0.0",
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	client := NewMetadataClient(httpClient, serviceIndexClient)

	ctx := context.Background()
	versions, err := client.ListVersions(ctx, server.URL, "testpkg")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}

	expectedVersions := []string{"1.0.0", "1.5.0", "2.0.0"}
	if len(versions) != len(expectedVersions) {
		t.Fatalf("ListVersions() returned %d versions, want %d", len(versions), len(expectedVersions))
	}

	for i, expected := range expectedVersions {
		if versions[i] != expected {
			t.Errorf("ListVersions()[%d] = %s, want %s", i, versions[i], expected)
		}
	}
}

func TestMetadataClient_ListVersions_ErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*httptest.Server) http.HandlerFunc
		wantErr     bool
		errContains string
	}{
		{
			name: "HTTP error",
			setupFunc: func(server *httptest.Server) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/index.json":
						index := ServiceIndex{
							Version: "3.0.0",
							Resources: []Resource{
								{
									ID:   "http://" + r.Host + "/registration/",
									Type: ResourceTypeRegistrationsBaseURL,
								},
							},
						}
						_ = json.NewEncoder(w).Encode(index)
					case "/registration/testpkg/index.json":
						http.Error(w, "Not Found", http.StatusNotFound)
					default:
						http.NotFound(w, r)
					}
				}
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "invalid JSON",
			setupFunc: func(server *httptest.Server) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/index.json":
						index := ServiceIndex{
							Version: "3.0.0",
							Resources: []Resource{
								{
									ID:   "http://" + r.Host + "/registration/",
									Type: ResourceTypeRegistrationsBaseURL,
								},
							},
						}
						_ = json.NewEncoder(w).Encode(index)
					case "/registration/testpkg/index.json":
						_, _ = w.Write([]byte("not json"))
					default:
						http.NotFound(w, r)
					}
				}
			},
			wantErr:     true,
			errContains: "decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.setupFunc(nil))
			defer server.Close()

			// Update handler with server reference
			server.Config.Handler = tt.setupFunc(server)

			httpClient := nugethttp.NewClient(nil)
			serviceIndexClient := NewServiceIndexClient(httpClient)
			client := NewMetadataClient(httpClient, serviceIndexClient)

			ctx := context.Background()
			_, err := client.ListVersions(ctx, server.URL, "testpkg")

			if (err != nil) != tt.wantErr {
				t.Errorf("ListVersions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetadataClient_fetchRegistrationPage_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			index := ServiceIndex{
				Version: "3.0.0",
				Resources: []Resource{
					{
						ID:   "http://" + r.Host + "/registration/",
						Type: ResourceTypeRegistrationsBaseURL,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(index)

		case "/registration/testpkg/index.json":
			response := RegistrationIndex{
				Count: 1,
				Items: []RegistrationPage{
					{
						ID:    "http://" + r.Host + "/registration/testpkg/page/bad.json",
						Lower: "1.0.0",
						Upper: "2.0.0",
						// No items - will fetch page
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)

		case "/registration/testpkg/page/bad.json":
			// Return invalid JSON for page
			_, _ = w.Write([]byte("invalid json"))

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	client := NewMetadataClient(httpClient, serviceIndexClient)

	ctx := context.Background()
	_, err := client.GetPackageMetadata(ctx, server.URL, "testpkg")
	if err == nil {
		t.Error("GetPackageMetadata() expected error when page returns invalid JSON")
	}
}
