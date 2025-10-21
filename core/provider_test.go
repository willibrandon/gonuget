package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/protocol/v3"
)

func setupV3TestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.json" {
			w.Header().Set("Content-Type", "application/json")
			index := v3.ServiceIndex{
				Version: "3.0.0",
				Resources: []v3.Resource{
					{ID: "http://localhost/search", Type: "SearchQueryService"},
					{ID: "http://localhost/registration/", Type: "RegistrationsBaseUrl"},
					{ID: "http://localhost/packages/", Type: "PackageBaseAddress"},
				},
			}
			json.NewEncoder(w).Encode(index)
			return
		}
		http.NotFound(w, r)
	}))
}

func setupV2TestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "" {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0"?>
<service xmlns="http://www.w3.org/2007/app">
  <workspace>
    <title>Default</title>
    <collection href="Packages">
      <title>Packages</title>
    </collection>
  </workspace>
</service>`))
			return
		}
		http.NotFound(w, r)
	}))
}

func TestProviderFactory_CreateProvider_V3(t *testing.T) {
	server := setupV3TestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	factory := NewProviderFactory(httpClient)

	ctx := context.Background()
	provider, err := factory.CreateProvider(ctx, server.URL)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if provider.ProtocolVersion() != "v3" {
		t.Errorf("ProtocolVersion() = %q, want v3", provider.ProtocolVersion())
	}

	if provider.SourceURL() != server.URL {
		t.Errorf("SourceURL() = %q, want %q", provider.SourceURL(), server.URL)
	}
}

func TestProviderFactory_CreateProvider_V2(t *testing.T) {
	server := setupV2TestServer()
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	factory := NewProviderFactory(httpClient)

	ctx := context.Background()
	provider, err := factory.CreateProvider(ctx, server.URL)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if provider.ProtocolVersion() != "v2" {
		t.Errorf("ProtocolVersion() = %q, want v2", provider.ProtocolVersion())
	}

	if provider.SourceURL() != server.URL {
		t.Errorf("SourceURL() = %q, want %q", provider.SourceURL(), server.URL)
	}
}

func TestProviderFactory_CreateProvider_Unknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	httpClient := nugethttp.NewClient(nil)
	factory := NewProviderFactory(httpClient)

	ctx := context.Background()
	_, err := factory.CreateProvider(ctx, server.URL)
	if err == nil {
		t.Error("CreateProvider() expected error for unknown protocol")
	}
}

func TestProviderFactory_CreateV3Provider(t *testing.T) {
	httpClient := nugethttp.NewClient(nil)
	factory := NewProviderFactory(httpClient)

	provider := factory.CreateV3Provider("https://api.nuget.org/v3/index.json")

	if provider.ProtocolVersion() != "v3" {
		t.Errorf("ProtocolVersion() = %q, want v3", provider.ProtocolVersion())
	}
}

func TestProviderFactory_CreateV2Provider(t *testing.T) {
	httpClient := nugethttp.NewClient(nil)
	factory := NewProviderFactory(httpClient)

	provider := factory.CreateV2Provider("https://www.nuget.org/api/v2")

	if provider.ProtocolVersion() != "v2" {
		t.Errorf("ProtocolVersion() = %q, want v2", provider.ProtocolVersion())
	}
}

func TestV3ResourceProvider_SourceURL(t *testing.T) {
	httpClient := nugethttp.NewClient(nil)
	sourceURL := "https://api.nuget.org/v3/index.json"

	provider := NewV3ResourceProvider(sourceURL, httpClient)

	if provider.SourceURL() != sourceURL {
		t.Errorf("SourceURL() = %q, want %q", provider.SourceURL(), sourceURL)
	}
}

func TestV2ResourceProvider_SourceURL(t *testing.T) {
	httpClient := nugethttp.NewClient(nil)
	sourceURL := "https://www.nuget.org/api/v2"

	provider := NewV2ResourceProvider(sourceURL, httpClient)

	if provider.SourceURL() != sourceURL {
		t.Errorf("SourceURL() = %q, want %q", provider.SourceURL(), sourceURL)
	}
}

func TestParseDependencies(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // number of dependency groups
	}{
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
		{
			name:  "single dependency with framework",
			input: "Newtonsoft.Json:13.0.1:net6.0",
			want:  1,
		},
		{
			name:  "multiple dependencies same framework",
			input: "Newtonsoft.Json:13.0.1:net6.0|System.Text.Json:6.0.0:net6.0",
			want:  1,
		},
		{
			name:  "multiple dependencies different frameworks",
			input: "Newtonsoft.Json:13.0.1:net6.0|System.Text.Json:6.0.0:netstandard2.0",
			want:  2,
		},
		{
			name:  "dependency without framework",
			input: "Newtonsoft.Json:13.0.1",
			want:  1,
		},
		{
			name:  "mixed with and without frameworks",
			input: "PackageA:1.0|PackageB:2.0:net45|PackageC:3.0:net45",
			want:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDependencies(tt.input)
			if len(got) != tt.want {
				t.Errorf("parseDependencies() returned %d groups, want %d", len(got), tt.want)
			}
		})
	}
}

func TestParseDependencies_Details(t *testing.T) {
	input := "Newtonsoft.Json:13.0.1:net6.0|System.Text.Json:6.0.0:net6.0|Dapper:2.0.0:netstandard2.0"
	groups := parseDependencies(input)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// Check that we have the right frameworks
	frameworks := make(map[string]bool)
	for _, group := range groups {
		frameworks[group.TargetFramework] = true

		switch group.TargetFramework {
		case "net6.0":
			if len(group.Dependencies) != 2 {
				t.Errorf("net6.0 group has %d dependencies, want 2", len(group.Dependencies))
			}
		case "netstandard2.0":
			if len(group.Dependencies) != 1 {
				t.Errorf("netstandard2.0 group has %d dependencies, want 1", len(group.Dependencies))
			}
			if group.Dependencies[0].ID != "Dapper" {
				t.Errorf("dependency ID = %q, want Dapper", group.Dependencies[0].ID)
			}
			if group.Dependencies[0].Range != "2.0.0" {
				t.Errorf("dependency Range = %q, want 2.0.0", group.Dependencies[0].Range)
			}
		}
	}

	if !frameworks["net6.0"] || !frameworks["netstandard2.0"] {
		t.Error("expected frameworks net6.0 and netstandard2.0")
	}
}

func BenchmarkParseDependencies(b *testing.B) {
	input := "Newtonsoft.Json:13.0.1:net6.0|System.Text.Json:6.0.0:net6.0|Dapper:2.0.0:netstandard2.0|AutoMapper:12.0.0:net6.0|FluentValidation:11.0.0:netstandard2.0"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = parseDependencies(input)
	}
}

// parseDependenciesSplit is the old implementation using strings.Split for comparison
func parseDependenciesSplit(deps string) []ProtocolDependencyGroup {
	if deps == "" {
		return nil
	}

	groups := make(map[string][]ProtocolDependency)

	parts := strings.Split(deps, "|")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		fields := strings.Split(part, ":")
		if len(fields) < 2 {
			continue
		}

		id := strings.TrimSpace(fields[0])
		versionRange := strings.TrimSpace(fields[1])
		targetFramework := ""
		if len(fields) >= 3 {
			targetFramework = strings.TrimSpace(fields[2])
		}

		dep := ProtocolDependency{
			ID:    id,
			Range: versionRange,
		}

		groups[targetFramework] = append(groups[targetFramework], dep)
	}

	var result []ProtocolDependencyGroup
	for framework, deps := range groups {
		result = append(result, ProtocolDependencyGroup{
			TargetFramework: framework,
			Dependencies:    deps,
		})
	}

	return result
}

func BenchmarkParseDependencies_Split(b *testing.B) {
	input := "Newtonsoft.Json:13.0.1:net6.0|System.Text.Json:6.0.0:net6.0|Dapper:2.0.0:netstandard2.0|AutoMapper:12.0.0:net6.0|FluentValidation:11.0.0:netstandard2.0"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = parseDependenciesSplit(input)
	}
}

// trimString trims leading/trailing spaces only if needed (avoids allocation)
func trimString(s string) string {
	start := 0
	end := len(s)

	// Trim leading spaces
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Trim trailing spaces
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	if start == 0 && end == len(s) {
		return s // No trimming needed, return original (no allocation)
	}

	return s[start:end] // Return substring (no allocation, just new string header)
}

// parseDependenciesOptimized minimizes allocations
func parseDependenciesOptimized(deps string) []ProtocolDependencyGroup {
	if deps == "" {
		return nil
	}

	// Pre-allocate map with expected capacity (reduces map growth allocations)
	groups := make(map[string][]ProtocolDependency, 2)

	// Iterate over pipe-separated parts using Cut
	for len(deps) > 0 {
		var part string
		part, deps, _ = strings.Cut(deps, "|")
		part = trimString(part)
		if part == "" {
			continue
		}

		// Parse id:range:targetFramework
		id, rest, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		id = trimString(id)

		versionRange, targetFramework, _ := strings.Cut(rest, ":")
		versionRange = trimString(versionRange)
		targetFramework = trimString(targetFramework)

		dep := ProtocolDependency{
			ID:    id,
			Range: versionRange,
		}

		groups[targetFramework] = append(groups[targetFramework], dep)
	}

	// Pre-allocate result slice with exact size (avoids growth)
	result := make([]ProtocolDependencyGroup, 0, len(groups))
	for framework, deps := range groups {
		result = append(result, ProtocolDependencyGroup{
			TargetFramework: framework,
			Dependencies:    deps,
		})
	}

	return result
}

func BenchmarkParseDependencies_Optimized(b *testing.B) {
	input := "Newtonsoft.Json:13.0.1:net6.0|System.Text.Json:6.0.0:net6.0|Dapper:2.0.0:netstandard2.0|AutoMapper:12.0.0:net6.0|FluentValidation:11.0.0:netstandard2.0"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = parseDependenciesOptimized(input)
	}
}

// parseDependenciesTwoPass pre-counts to eliminate all growth allocations
func parseDependenciesTwoPass(deps string) []ProtocolDependencyGroup {
	if deps == "" {
		return nil
	}

	// First pass: count dependencies per framework
	frameworkCounts := make(map[string]int, 2)
	tempDeps := deps
	for len(tempDeps) > 0 {
		var part string
		part, tempDeps, _ = strings.Cut(tempDeps, "|")
		part = trimString(part)
		if part == "" {
			continue
		}

		// Parse to get framework
		_, rest, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		_, targetFramework, _ := strings.Cut(rest, ":")
		targetFramework = trimString(targetFramework)

		frameworkCounts[targetFramework]++
	}

	// Pre-allocate slices for each framework
	groups := make(map[string][]ProtocolDependency, len(frameworkCounts))
	for framework, count := range frameworkCounts {
		groups[framework] = make([]ProtocolDependency, 0, count)
	}

	// Second pass: populate dependencies
	for len(deps) > 0 {
		var part string
		part, deps, _ = strings.Cut(deps, "|")
		part = trimString(part)
		if part == "" {
			continue
		}

		id, rest, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		id = trimString(id)

		versionRange, targetFramework, _ := strings.Cut(rest, ":")
		versionRange = trimString(versionRange)
		targetFramework = trimString(targetFramework)

		dep := ProtocolDependency{
			ID:    id,
			Range: versionRange,
		}

		groups[targetFramework] = append(groups[targetFramework], dep)
	}

	// Pre-allocate result with exact size
	result := make([]ProtocolDependencyGroup, 0, len(groups))
	for framework, deps := range groups {
		result = append(result, ProtocolDependencyGroup{
			TargetFramework: framework,
			Dependencies:    deps,
		})
	}

	return result
}

func BenchmarkParseDependencies_TwoPass(b *testing.B) {
	input := "Newtonsoft.Json:13.0.1:net6.0|System.Text.Json:6.0.0:net6.0|Dapper:2.0.0:netstandard2.0|AutoMapper:12.0.0:net6.0|FluentValidation:11.0.0:netstandard2.0"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = parseDependenciesTwoPass(input)
	}
}
