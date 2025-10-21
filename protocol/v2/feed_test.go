package v2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
)

const testServiceDocument = `<?xml version="1.0" encoding="utf-8"?>
<service xml:base="https://www.nuget.org/api/v2" xmlns="http://www.w3.org/2007/app" xmlns:atom="http://www.w3.org/2005/Atom">
  <workspace>
    <atom:title type="text">Default</atom:title>
    <collection href="Packages">
      <atom:title type="text">Packages</atom:title>
    </collection>
  </workspace>
</service>`

func setupV2Server(t *testing.T) (*httptest.Server, *FeedClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "" {
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			_, _ = w.Write([]byte(testServiceDocument))
			return
		}
		http.NotFound(w, r)
	}))

	httpClient := nugethttp.NewClient(nil)
	feedClient := NewFeedClient(httpClient)

	return server, feedClient
}

func TestFeedClient_DetectV2Feed(t *testing.T) {
	server, client := setupV2Server(t)
	defer server.Close()

	ctx := context.Background()

	detected, err := client.DetectV2Feed(ctx, server.URL)
	if err != nil {
		t.Fatalf("DetectV2Feed() error = %v", err)
	}

	if !detected {
		t.Error("DetectV2Feed() = false, want true")
	}
}

func TestFeedClient_DetectV2Feed_NotV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"3.0.0"}`))
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	client := NewFeedClient(httpClient)

	ctx := context.Background()

	detected, err := client.DetectV2Feed(ctx, server.URL)
	if err != nil {
		t.Fatalf("DetectV2Feed() error = %v", err)
	}

	if detected {
		t.Error("DetectV2Feed() = true, want false for non-v2 feed")
	}
}

func TestFeedClient_GetServiceDocument(t *testing.T) {
	server, client := setupV2Server(t)
	defer server.Close()

	ctx := context.Background()

	service, err := client.GetServiceDocument(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetServiceDocument() error = %v", err)
	}

	if service.Workspace.Title != "Default" {
		t.Errorf("Workspace.Title = %q, want Default", service.Workspace.Title)
	}

	if len(service.Workspace.Collections) != 1 {
		t.Fatalf("len(Collections) = %d, want 1", len(service.Workspace.Collections))
	}

	collection := service.Workspace.Collections[0]
	if collection.Href != "Packages" {
		t.Errorf("Collection.Href = %q, want Packages", collection.Href)
	}

	if collection.Title != "Packages" {
		t.Errorf("Collection.Title = %q, want Packages", collection.Title)
	}
}
