package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/willibrandon/gonuget/auth"
)

func TestSourceRepository_Name(t *testing.T) {
	repo := NewSourceRepository(RepositoryConfig{
		Name:      "nuget.org",
		SourceURL: "https://api.nuget.org/v3/index.json",
	})

	if repo.Name() != "nuget.org" {
		t.Errorf("Name() = %q, want nuget.org", repo.Name())
	}
}

func TestSourceRepository_SourceURL(t *testing.T) {
	sourceURL := "https://api.nuget.org/v3/index.json"
	repo := NewSourceRepository(RepositoryConfig{
		Name:      "nuget.org",
		SourceURL: sourceURL,
	})

	if repo.SourceURL() != sourceURL {
		t.Errorf("SourceURL() = %q, want %q", repo.SourceURL(), sourceURL)
	}
}

func TestSourceRepository_GetProvider(t *testing.T) {
	server := setupV3TestServer()
	defer server.Close()

	repo := NewSourceRepository(RepositoryConfig{
		Name:      "test",
		SourceURL: server.URL + "/index.json",
	})

	ctx := context.Background()

	// First call - should create provider
	provider1, err := repo.GetProvider(ctx)
	if err != nil {
		t.Fatalf("GetProvider() error = %v", err)
	}

	if provider1 == nil {
		t.Fatal("GetProvider() returned nil provider")
	}

	// Second call - should return cached provider
	provider2, err := repo.GetProvider(ctx)
	if err != nil {
		t.Fatalf("GetProvider() error = %v", err)
	}

	if provider1 != provider2 {
		t.Error("GetProvider() should return cached provider")
	}
}

func TestSourceRepository_WithAuthentication(t *testing.T) {
	apiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key is present
		gotKey := r.Header.Get("X-NuGet-ApiKey")
		if gotKey != apiKey {
			t.Errorf("X-NuGet-ApiKey = %q, want %q", gotKey, apiKey)
		}

		if r.URL.Path == "/index.json" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"version":   "3.0.0",
				"resources": []map[string]string{},
			})
		}
	}))
	defer server.Close()

	authenticator := auth.NewAPIKeyAuthenticator(apiKey)

	repo := NewSourceRepository(RepositoryConfig{
		Name:          "test",
		SourceURL:     server.URL + "/index.json",
		Authenticator: authenticator,
	})

	ctx := context.Background()
	_, err := repo.GetProvider(ctx)
	if err != nil {
		t.Fatalf("GetProvider() error = %v", err)
	}
}

func TestRepositoryManager_AddRepository(t *testing.T) {
	manager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:      "nuget.org",
		SourceURL: "https://api.nuget.org/v3/index.json",
	})

	err := manager.AddRepository(repo)
	if err != nil {
		t.Fatalf("_ = AddRepository() error = %v", err)
	}

	// Try to add duplicate
	err = manager.AddRepository(repo)
	if err == nil {
		t.Error("_ = AddRepository() expected error for duplicate")
	}
}

func TestRepositoryManager_GetRepository(t *testing.T) {
	manager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:      "nuget.org",
		SourceURL: "https://api.nuget.org/v3/index.json",
	})

	_ = manager.AddRepository(repo)

	got, err := manager.GetRepository("nuget.org")
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}

	if got.Name() != "nuget.org" {
		t.Errorf("Name() = %q, want nuget.org", got.Name())
	}

	// Try to get non-existent repository
	_, err = manager.GetRepository("nonexistent")
	if err == nil {
		t.Error("GetRepository() expected error for non-existent repo")
	}
}

func TestRepositoryManager_RemoveRepository(t *testing.T) {
	manager := NewRepositoryManager()

	repo := NewSourceRepository(RepositoryConfig{
		Name:      "nuget.org",
		SourceURL: "https://api.nuget.org/v3/index.json",
	})

	_ = manager.AddRepository(repo)

	err := manager.RemoveRepository("nuget.org")
	if err != nil {
		t.Fatalf("RemoveRepository() error = %v", err)
	}

	// Verify removed
	_, err = manager.GetRepository("nuget.org")
	if err == nil {
		t.Error("GetRepository() should fail after removal")
	}

	// Try to remove non-existent
	err = manager.RemoveRepository("nonexistent")
	if err == nil {
		t.Error("RemoveRepository() expected error for non-existent repo")
	}
}

func TestRepositoryManager_ListRepositories(t *testing.T) {
	manager := NewRepositoryManager()

	repo1 := NewSourceRepository(RepositoryConfig{
		Name:      "nuget.org",
		SourceURL: "https://api.nuget.org/v3/index.json",
	})

	repo2 := NewSourceRepository(RepositoryConfig{
		Name:      "myget",
		SourceURL: "https://myget.org/v3/index.json",
	})

	_ = manager.AddRepository(repo1)
	_ = manager.AddRepository(repo2)

	repos := manager.ListRepositories()

	if len(repos) != 2 {
		t.Errorf("len(repos) = %d, want 2", len(repos))
	}
}
