package v2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

const testFeedResponse = `<?xml version="1.0" encoding="utf-8"?>
<feed xml:base="https://www.nuget.org/api/v2" xmlns="http://www.w3.org/2005/Atom" xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <title type="text">Packages</title>
  <id>http://schemas.datacontract.org/2004/07/</id>
  <updated>2023-01-01T00:00:00Z</updated>
  <entry>
    <id>https://www.nuget.org/api/v2/Packages(Id='Newtonsoft.Json',Version='13.0.3')</id>
    <title type="text">Newtonsoft.Json</title>
    <updated>2023-03-08T18:36:53Z</updated>
    <content type="application/zip" src="https://www.nuget.org/api/v2/package/Newtonsoft.Json/13.0.3" />
    <m:properties>
      <d:Id>Newtonsoft.Json</d:Id>
      <d:Version>13.0.3</d:Version>
      <d:Description>Json.NET is a popular high-performance JSON framework for .NET</d:Description>
      <d:Authors>James Newton-King</d:Authors>
      <d:IconUrl>https://www.newtonsoft.com/content/images/nugeticon.png</d:IconUrl>
      <d:LicenseUrl>https://licenses.nuget.org/MIT</d:LicenseUrl>
      <d:ProjectUrl>https://www.newtonsoft.com/json</d:ProjectUrl>
      <d:Tags>json serialization</d:Tags>
      <d:Dependencies></d:Dependencies>
      <d:DownloadCount m:type="Edm.Int64">1000000000</d:DownloadCount>
      <d:IsPrerelease m:type="Edm.Boolean">false</d:IsPrerelease>
      <d:Published>2023-03-08T18:36:53.147Z</d:Published>
      <d:RequireLicenseAcceptance m:type="Edm.Boolean">false</d:RequireLicenseAcceptance>
    </m:properties>
  </entry>
</feed>`

func setupV2SearchServer(t *testing.T) (*httptest.Server, *SearchClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "Packages") {
			// Validate query parameters
			query := r.URL.Query()
			if query.Has("$filter") {
				filter := query.Get("$filter")
				// Basic validation
				if !strings.Contains(filter, "substringof") && !strings.Contains(filter, "Id eq") {
					t.Logf("Unexpected filter: %s", filter)
				}
			}

			w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
			_, _ = w.Write([]byte(testFeedResponse))
			return
		}
		http.NotFound(w, r)
	}))

	httpClient := nugethttp.NewClient(nil)
	searchClient := NewSearchClient(httpClient)

	return server, searchClient
}

func TestSearchClient_Search(t *testing.T) {
	server, client := setupV2SearchServer(t)
	defer server.Close()

	ctx := context.Background()

	results, err := client.Search(ctx, server.URL, SearchOptions{
		Query:             "newtonsoft",
		Top:               20,
		IncludePrerelease: true,
	})

	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	result := results[0]

	if result.ID != "Newtonsoft.Json" {
		t.Errorf("ID = %q, want Newtonsoft.Json", result.ID)
	}

	if result.Version != "13.0.3" {
		t.Errorf("Version = %q, want 13.0.3", result.Version)
	}

	if result.Authors != "James Newton-King" {
		t.Errorf("Authors = %q, want James Newton-King", result.Authors)
	}

	if result.DownloadCount != 1000000000 {
		t.Errorf("DownloadCount = %d, want 1000000000", result.DownloadCount)
	}

	if result.IsPrerelease {
		t.Error("IsPrerelease = true, want false")
	}

	if len(result.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(result.Tags))
	}

	expectedTags := []string{"json", "serialization"}
	for i, want := range expectedTags {
		if result.Tags[i] != want {
			t.Errorf("Tags[%d] = %q, want %q", i, result.Tags[i], want)
		}
	}
}

func TestSearchClient_FindPackagesById(t *testing.T) {
	server, client := setupV2SearchServer(t)
	defer server.Close()

	ctx := context.Background()

	results, err := client.FindPackagesById(ctx, server.URL, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("FindPackagesById() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	if results[0].ID != "Newtonsoft.Json" {
		t.Errorf("ID = %q, want Newtonsoft.Json", results[0].ID)
	}
}

func TestSearchClient_BuildSearchURL(t *testing.T) {
	client := NewSearchClient(nil)

	tests := []struct {
		name    string
		feedURL string
		opts    SearchOptions
		wantURL string
	}{
		{
			name:    "basic search",
			feedURL: "https://api.nuget.org/v2/",
			opts: SearchOptions{
				Query: "newtonsoft",
				Top:   20,
			},
			wantURL: "https://api.nuget.org/v2/Packages()",
		},
		{
			name:    "with skip",
			feedURL: "https://api.nuget.org/v2/",
			opts: SearchOptions{
				Query: "json",
				Skip:  10,
				Top:   5,
			},
			wantURL: "https://api.nuget.org/v2/Packages()",
		},
		{
			name:    "no query",
			feedURL: "https://api.nuget.org/v2",
			opts: SearchOptions{
				Top: 10,
			},
			wantURL: "https://api.nuget.org/v2/Packages()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.buildSearchURL(tt.feedURL, tt.opts)
			if err != nil {
				t.Fatalf("buildSearchURL() error = %v", err)
			}

			if !strings.HasPrefix(got, tt.wantURL) {
				t.Errorf("URL = %q, want prefix %q", got, tt.wantURL)
			}

			// Verify URL has query parameters
			if !strings.Contains(got, "?") {
				t.Error("URL missing query parameters")
			}
		})
	}
}
