package v2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

const testPackageEntry = `<?xml version="1.0" encoding="utf-8"?>
<entry xml:base="https://www.nuget.org/api/v2" xmlns="http://www.w3.org/2005/Atom" xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
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
    <d:DownloadCount m:type="Edm.Int64">1500000000</d:DownloadCount>
    <d:IsPrerelease m:type="Edm.Boolean">false</d:IsPrerelease>
    <d:Published>2023-03-08T18:36:53.147Z</d:Published>
    <d:RequireLicenseAcceptance m:type="Edm.Boolean">false</d:RequireLicenseAcceptance>
  </m:properties>
</entry>`

const testVersionsFeed = `<?xml version="1.0" encoding="utf-8"?>
<feed xml:base="https://www.nuget.org/api/v2" xmlns="http://www.w3.org/2005/Atom" xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <title type="text">Packages</title>
  <id>http://schemas.datacontract.org/2004/07/</id>
  <updated>2023-01-01T00:00:00Z</updated>
  <entry>
    <id>https://www.nuget.org/api/v2/Packages(Id='Newtonsoft.Json',Version='13.0.1')</id>
    <title type="text">Newtonsoft.Json</title>
    <updated>2021-09-01T00:00:00Z</updated>
    <content type="application/zip" src="https://www.nuget.org/api/v2/package/Newtonsoft.Json/13.0.1" />
    <m:properties>
      <d:Id>Newtonsoft.Json</d:Id>
      <d:Version>13.0.1</d:Version>
      <d:Description>Json.NET is a popular high-performance JSON framework for .NET</d:Description>
      <d:Authors>James Newton-King</d:Authors>
      <d:IconUrl>https://www.newtonsoft.com/content/images/nugeticon.png</d:IconUrl>
      <d:LicenseUrl>https://licenses.nuget.org/MIT</d:LicenseUrl>
      <d:ProjectUrl>https://www.newtonsoft.com/json</d:ProjectUrl>
      <d:Tags>json</d:Tags>
      <d:Dependencies></d:Dependencies>
      <d:DownloadCount m:type="Edm.Int64">1400000000</d:DownloadCount>
      <d:IsPrerelease m:type="Edm.Boolean">false</d:IsPrerelease>
      <d:Published>2021-09-01T00:00:00.000Z</d:Published>
      <d:RequireLicenseAcceptance m:type="Edm.Boolean">false</d:RequireLicenseAcceptance>
    </m:properties>
  </entry>
  <entry>
    <id>https://www.nuget.org/api/v2/Packages(Id='Newtonsoft.Json',Version='13.0.2')</id>
    <title type="text">Newtonsoft.Json</title>
    <updated>2022-02-20T00:00:00Z</updated>
    <content type="application/zip" src="https://www.nuget.org/api/v2/package/Newtonsoft.Json/13.0.2" />
    <m:properties>
      <d:Id>Newtonsoft.Json</d:Id>
      <d:Version>13.0.2</d:Version>
      <d:Description>Json.NET is a popular high-performance JSON framework for .NET</d:Description>
      <d:Authors>James Newton-King</d:Authors>
      <d:IconUrl>https://www.newtonsoft.com/content/images/nugeticon.png</d:IconUrl>
      <d:LicenseUrl>https://licenses.nuget.org/MIT</d:LicenseUrl>
      <d:ProjectUrl>https://www.newtonsoft.com/json</d:ProjectUrl>
      <d:Tags>json</d:Tags>
      <d:Dependencies></d:Dependencies>
      <d:DownloadCount m:type="Edm.Int64">1450000000</d:DownloadCount>
      <d:IsPrerelease m:type="Edm.Boolean">false</d:IsPrerelease>
      <d:Published>2022-02-20T00:00:00.000Z</d:Published>
      <d:RequireLicenseAcceptance m:type="Edm.Boolean">false</d:RequireLicenseAcceptance>
    </m:properties>
  </entry>
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
      <d:DownloadCount m:type="Edm.Int64">1500000000</d:DownloadCount>
      <d:IsPrerelease m:type="Edm.Boolean">false</d:IsPrerelease>
      <d:Published>2023-03-08T18:36:53.147Z</d:Published>
      <d:RequireLicenseAcceptance m:type="Edm.Boolean">false</d:RequireLicenseAcceptance>
    </m:properties>
  </entry>
</feed>`

func setupV2MetadataServer(t *testing.T) (*httptest.Server, *MetadataClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle /Packages(Id='...',Version='...') endpoint
		if strings.Contains(r.URL.Path, "Packages(") {
			if strings.Contains(r.URL.Path, "Newtonsoft.Json") && strings.Contains(r.URL.Path, "13.0.3") {
				w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
				_, _ = w.Write([]byte(testPackageEntry))
				return
			}
			// Version not found
			http.NotFound(w, r)
			return
		}

		// Handle /FindPackagesById() endpoint
		if strings.Contains(r.URL.Path, "FindPackagesById") {
			query := r.URL.Query()
			if query.Get("id") == "'Newtonsoft.Json'" || query.Get("id") == "Newtonsoft.Json" {
				w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
				_, _ = w.Write([]byte(testVersionsFeed))
				return
			}
			// Package not found
			http.NotFound(w, r)
			return
		}

		http.NotFound(w, r)
	}))

	httpClient := nugethttp.NewClient(nil)
	metadataClient := NewMetadataClient(httpClient)

	return server, metadataClient
}

func TestMetadataClient_GetPackageMetadata(t *testing.T) {
	server, client := setupV2MetadataServer(t)
	defer server.Close()

	ctx := context.Background()

	metadata, err := client.GetPackageMetadata(ctx, server.URL, "Newtonsoft.Json", "13.0.3")
	if err != nil {
		t.Fatalf("GetPackageMetadata() error = %v", err)
	}

	if metadata.ID != "Newtonsoft.Json" {
		t.Errorf("ID = %q, want Newtonsoft.Json", metadata.ID)
	}

	if metadata.Version != "13.0.3" {
		t.Errorf("Version = %q, want 13.0.3", metadata.Version)
	}

	if metadata.Authors != "James Newton-King" {
		t.Errorf("Authors = %q, want James Newton-King", metadata.Authors)
	}

	if metadata.Description != "Json.NET is a popular high-performance JSON framework for .NET" {
		t.Errorf("Description = %q", metadata.Description)
	}

	if metadata.IconURL != "https://www.newtonsoft.com/content/images/nugeticon.png" {
		t.Errorf("IconURL = %q", metadata.IconURL)
	}

	if metadata.LicenseURL != "https://licenses.nuget.org/MIT" {
		t.Errorf("LicenseURL = %q, want https://licenses.nuget.org/MIT", metadata.LicenseURL)
	}

	if metadata.ProjectURL != "https://www.newtonsoft.com/json" {
		t.Errorf("ProjectURL = %q, want https://www.newtonsoft.com/json", metadata.ProjectURL)
	}

	if metadata.DownloadCount != 1500000000 {
		t.Errorf("DownloadCount = %d, want 1500000000", metadata.DownloadCount)
	}

	if metadata.IsPrerelease {
		t.Error("IsPrerelease = true, want false")
	}

	if metadata.RequireLicenseAcceptance {
		t.Error("RequireLicenseAcceptance = true, want false")
	}

	if metadata.DownloadURL != "https://www.nuget.org/api/v2/package/Newtonsoft.Json/13.0.3" {
		t.Errorf("DownloadURL = %q", metadata.DownloadURL)
	}

	if metadata.Title != "Newtonsoft.Json" {
		t.Errorf("Title = %q, want Newtonsoft.Json", metadata.Title)
	}

	if len(metadata.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(metadata.Tags))
	}

	expectedTags := []string{"json", "serialization"}
	for i, want := range expectedTags {
		if metadata.Tags[i] != want {
			t.Errorf("Tags[%d] = %q, want %q", i, metadata.Tags[i], want)
		}
	}
}

func TestMetadataClient_GetPackageMetadata_NotFound(t *testing.T) {
	server, client := setupV2MetadataServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.GetPackageMetadata(ctx, server.URL, "Newtonsoft.Json", "99.0.0")
	if err == nil {
		t.Error("expected error for non-existent version")
	}

	expectedMsg := "not found"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("error = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestMetadataClient_ListVersions(t *testing.T) {
	server, client := setupV2MetadataServer(t)
	defer server.Close()

	ctx := context.Background()

	versions, err := client.ListVersions(ctx, server.URL, "Newtonsoft.Json")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}

	if len(versions) != 3 {
		t.Fatalf("len(versions) = %d, want 3", len(versions))
	}

	expected := []string{"13.0.1", "13.0.2", "13.0.3"}
	for i, want := range expected {
		if versions[i] != want {
			t.Errorf("versions[%d] = %q, want %q", i, versions[i], want)
		}
	}
}

func TestMetadataClient_BuildMetadataURL(t *testing.T) {
	client := NewMetadataClient(nil)

	tests := []struct {
		name       string
		feedURL    string
		packageID  string
		version    string
		wantPrefix string
	}{
		{
			name:       "basic URL",
			feedURL:    "https://api.nuget.org/v2/",
			packageID:  "Newtonsoft.Json",
			version:    "13.0.3",
			wantPrefix: "https://api.nuget.org/v2/Packages(Id='Newtonsoft.Json',Version='13.0.3')",
		},
		{
			name:       "URL without trailing slash",
			feedURL:    "https://api.nuget.org/v2",
			packageID:  "Newtonsoft.Json",
			version:    "13.0.3",
			wantPrefix: "https://api.nuget.org/v2/Packages(Id='Newtonsoft.Json',Version='13.0.3')",
		},
		{
			name:       "package ID with special characters",
			feedURL:    "https://api.nuget.org/v2/",
			packageID:  "My.Package+Name",
			version:    "1.0.0",
			wantPrefix: "https://api.nuget.org/v2/Packages(Id='My.Package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.buildMetadataURL(tt.feedURL, tt.packageID, tt.version)
			if err != nil {
				t.Fatalf("buildMetadataURL() error = %v", err)
			}

			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("URL = %q, want prefix %q", got, tt.wantPrefix)
			}

			// Verify URL structure
			if !strings.Contains(got, "Packages(") {
				t.Error("URL missing Packages() function")
			}
		})
	}
}
