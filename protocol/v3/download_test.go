package v3

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

func setupDownloadServer() (*httptest.Server, *DownloadClient) {
	mux := http.NewServeMux()

	// Service index endpoint
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		index := &ServiceIndex{
			Version: "3.0.0",
			Resources: []Resource{
				{
					ID:   "http://" + r.Host + "/packages/",
					Type: ResourceTypePackageBaseAddress,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(index)
	})

	// Package download endpoint
	mux.HandleFunc("/packages/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/packages/")

		// Handle versions endpoint
		if strings.HasSuffix(path, "/index.json") {
			packageID := strings.TrimSuffix(path, "/index.json")
			if packageID == "newtonsoft.json" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"versions": []string{"13.0.1", "13.0.2", "13.0.3"},
				})
				return
			}
			http.NotFound(w, r)
			return
		}

		// Handle .nupkg download
		if strings.HasSuffix(path, ".nupkg") {
			if strings.Contains(path, "newtonsoft.json/13.0.3/") {
				w.Header().Set("Content-Type", "application/zip")
				_, _ = w.Write([]byte("PK\x03\x04")) // ZIP file signature
				_, _ = w.Write([]byte("fake nupkg content"))
				return
			}
			http.NotFound(w, r)
			return
		}

		// Handle .nuspec download
		if strings.HasSuffix(path, ".nuspec") {
			if strings.Contains(path, "newtonsoft.json/13.0.3/") {
				w.Header().Set("Content-Type", "application/xml")
				_, _ = w.Write([]byte(`<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>Newtonsoft.Json</id>
    <version>13.0.3</version>
  </metadata>
</package>`))
				return
			}
			http.NotFound(w, r)
			return
		}

		http.NotFound(w, r)
	})

	server := httptest.NewServer(mux)

	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := NewServiceIndexClient(httpClient)
	downloadClient := NewDownloadClient(httpClient, serviceIndexClient)

	return server, downloadClient
}

func TestDownloadClient_DownloadPackage(t *testing.T) {
	server, client := setupDownloadServer()
	defer server.Close()

	ctx := context.Background()

	body, err := client.DownloadPackage(ctx, server.URL+"/index.json", "Newtonsoft.Json", "13.0.3")
	if err != nil {
		t.Fatalf("DownloadPackage() error = %v", err)
	}
	defer func() { _ = body.Close() }()

	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Check ZIP signature
	if len(content) < 4 || content[0] != 'P' || content[1] != 'K' {
		t.Error("downloaded content does not have ZIP signature")
	}

	if !strings.Contains(string(content), "fake nupkg content") {
		t.Error("downloaded content missing expected data")
	}
}

func TestDownloadClient_DownloadPackage_NotFound(t *testing.T) {
	server, client := setupDownloadServer()
	defer server.Close()

	ctx := context.Background()

	_, err := client.DownloadPackage(ctx, server.URL+"/index.json", "NonExistent.Package", "1.0.0")
	if err == nil {
		t.Error("expected error for non-existent package")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestDownloadClient_DownloadNuspec(t *testing.T) {
	server, client := setupDownloadServer()
	defer server.Close()

	ctx := context.Background()

	body, err := client.DownloadNuspec(ctx, server.URL+"/index.json", "Newtonsoft.Json", "13.0.3")
	if err != nil {
		t.Fatalf("DownloadNuspec() error = %v", err)
	}
	defer func() { _ = body.Close() }()

	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "<?xml") {
		t.Error("nuspec content missing XML declaration")
	}

	if !strings.Contains(contentStr, "<id>Newtonsoft.Json</id>") {
		t.Error("nuspec content missing package ID")
	}

	if !strings.Contains(contentStr, "<version>13.0.3</version>") {
		t.Error("nuspec content missing version")
	}
}

func TestDownloadClient_DownloadNuspec_NotFound(t *testing.T) {
	server, client := setupDownloadServer()
	defer server.Close()

	ctx := context.Background()

	_, err := client.DownloadNuspec(ctx, server.URL+"/index.json", "NonExistent.Package", "1.0.0")
	if err == nil {
		t.Error("expected error for non-existent nuspec")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestDownloadClient_GetPackageVersions(t *testing.T) {
	server, client := setupDownloadServer()
	defer server.Close()

	ctx := context.Background()

	versions, err := client.GetPackageVersions(ctx, server.URL+"/index.json", "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("GetPackageVersions() error = %v", err)
	}

	expected := []string{"13.0.1", "13.0.2", "13.0.3"}
	if len(versions) != len(expected) {
		t.Fatalf("len(versions) = %d, want %d", len(versions), len(expected))
	}

	for i, want := range expected {
		if versions[i] != want {
			t.Errorf("versions[%d] = %q, want %q", i, versions[i], want)
		}
	}
}

func TestDownloadClient_GetPackageVersions_NotFound(t *testing.T) {
	server, client := setupDownloadServer()
	defer server.Close()

	ctx := context.Background()

	_, err := client.GetPackageVersions(ctx, server.URL+"/index.json", "NonExistent.Package")
	if err == nil {
		t.Error("expected error for non-existent package")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}
