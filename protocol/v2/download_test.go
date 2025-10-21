package v2

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

func setupV2DownloadServer(t *testing.T) (*httptest.Server, *DownloadClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Handle versioned download
		if strings.HasPrefix(path, "/package/") {
			parts := strings.Split(strings.TrimPrefix(path, "/package/"), "/")

			if len(parts) == 2 {
				// /package/{id}/{version}
				packageID := parts[0]
				version := parts[1]

				if packageID == "Newtonsoft.Json" && version == "13.0.3" {
					w.Header().Set("Content-Type", "application/zip")
					_, _ = w.Write([]byte("PK\x03\x04")) // ZIP signature
					_, _ = w.Write([]byte("fake nupkg content"))
					return
				}
			} else if len(parts) == 1 {
				// /package/{id} (latest)
				packageID := parts[0]

				if packageID == "Newtonsoft.Json" {
					w.Header().Set("Content-Type", "application/zip")
					_, _ = w.Write([]byte("PK\x03\x04")) // ZIP signature
					_, _ = w.Write([]byte("fake latest nupkg content"))
					return
				}
			}

			http.NotFound(w, r)
			return
		}

		http.NotFound(w, r)
	}))

	httpClient := nugethttp.NewClient(nil)
	downloadClient := NewDownloadClient(httpClient)

	return server, downloadClient
}

func TestDownloadClient_DownloadPackage(t *testing.T) {
	server, client := setupV2DownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	body, err := client.DownloadPackage(ctx, server.URL, "Newtonsoft.Json", "13.0.3")
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
	server, client := setupV2DownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.DownloadPackage(ctx, server.URL, "NonExistent.Package", "1.0.0")
	if err == nil {
		t.Error("expected error for non-existent package")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestDownloadClient_DownloadLatestPackage(t *testing.T) {
	server, client := setupV2DownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	body, err := client.DownloadLatestPackage(ctx, server.URL, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("DownloadLatestPackage() error = %v", err)
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

	if !strings.Contains(string(content), "fake latest nupkg content") {
		t.Error("downloaded content missing expected data")
	}
}

func TestDownloadClient_DownloadLatestPackage_NotFound(t *testing.T) {
	server, client := setupV2DownloadServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.DownloadLatestPackage(ctx, server.URL, "NonExistent.Package")
	if err == nil {
		t.Error("expected error for non-existent package")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestDownloadClient_BuildDownloadURL(t *testing.T) {
	client := NewDownloadClient(nil)

	tests := []struct {
		name      string
		feedURL   string
		packageID string
		version   string
		want      string
	}{
		{
			name:      "basic",
			feedURL:   "https://api.nuget.org/v2/",
			packageID: "Newtonsoft.Json",
			version:   "13.0.3",
			want:      "https://api.nuget.org/v2/package/Newtonsoft.Json/13.0.3",
		},
		{
			name:      "no trailing slash",
			feedURL:   "https://api.nuget.org/v2",
			packageID: "Newtonsoft.Json",
			version:   "13.0.3",
			want:      "https://api.nuget.org/v2/package/Newtonsoft.Json/13.0.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.buildDownloadURL(tt.feedURL, tt.packageID, tt.version)
			if err != nil {
				t.Fatalf("buildDownloadURL() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("buildDownloadURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDownloadClient_BuildLatestDownloadURL(t *testing.T) {
	client := NewDownloadClient(nil)

	got, err := client.buildLatestDownloadURL("https://api.nuget.org/v2/", "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("buildLatestDownloadURL() error = %v", err)
	}

	want := "https://api.nuget.org/v2/package/Newtonsoft.Json"
	if got != want {
		t.Errorf("buildLatestDownloadURL() = %q, want %q", got, want)
	}
}
