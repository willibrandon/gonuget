package resolver

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestParallelResolver_MultiplePackages(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {ID: "A", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"C|1.0.0": {ID: "C", Version: "1.0.0", Dependencies: []PackageDependency{}},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")

	packages := []PackageDependency{
		{ID: "A", VersionRange: "[1.0.0]"},
		{ID: "B", VersionRange: "[1.0.0]"},
		{ID: "C", VersionRange: "[1.0.0]"},
	}

	results, err := resolver.ResolveMultiple(context.Background(), packages)

	if err != nil {
		t.Fatalf("ResolveMultiple() failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Verify each result
	for i, result := range results {
		if result == nil {
			t.Errorf("Result %d is nil", i)
			continue
		}
		if len(result.Packages) == 0 {
			t.Errorf("Result %d has no packages", i)
		}
	}
}

func TestParallelResolver_BatchProcessing(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {ID: "A", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"C|1.0.0": {ID: "C", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"D|1.0.0": {ID: "D", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"E|1.0.0": {ID: "E", Version: "1.0.0", Dependencies: []PackageDependency{}},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")

	packages := []PackageDependency{
		{ID: "A", VersionRange: "[1.0.0]"},
		{ID: "B", VersionRange: "[1.0.0]"},
		{ID: "C", VersionRange: "[1.0.0]"},
		{ID: "D", VersionRange: "[1.0.0]"},
		{ID: "E", VersionRange: "[1.0.0]"},
	}

	// Process in batches of 2
	results, err := resolver.ResolveBatch(context.Background(), packages, 2)

	if err != nil {
		t.Fatalf("ResolveBatch() failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	// Verify each result
	for i, result := range results {
		if result == nil {
			t.Errorf("Result %d is nil", i)
			continue
		}
		if len(result.Packages) == 0 {
			t.Errorf("Result %d has no packages", i)
		}
	}
}

func TestParallelResolver_Cancellation(t *testing.T) {
	client := &slowMockClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {ID: "A", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
		},
		delay: 2 * time.Second,
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")

	packages := []PackageDependency{
		{ID: "A", VersionRange: "[1.0.0]"},
		{ID: "B", VersionRange: "[1.0.0]"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := resolver.ResolveMultiple(ctx, packages)

	if err == nil {
		t.Error("Expected context cancellation error")
	}

	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
	}
}

func TestParallelResolver_WorkerPoolLimit(t *testing.T) {
	client := &countingMockClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {ID: "A", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"C|1.0.0": {ID: "C", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"D|1.0.0": {ID: "D", Version: "1.0.0", Dependencies: []PackageDependency{}},
			"E|1.0.0": {ID: "E", Version: "1.0.0", Dependencies: []PackageDependency{}},
		},
		delay: 100 * time.Millisecond,
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")
	// Set max workers to 2
	resolver.parallelResolver = NewParallelResolver(resolver, 2)

	packages := []PackageDependency{
		{ID: "A", VersionRange: "[1.0.0]"},
		{ID: "B", VersionRange: "[1.0.0]"},
		{ID: "C", VersionRange: "[1.0.0]"},
		{ID: "D", VersionRange: "[1.0.0]"},
		{ID: "E", VersionRange: "[1.0.0]"},
	}

	results, err := resolver.ResolveMultiple(context.Background(), packages)

	if err != nil {
		t.Fatalf("ResolveMultiple() failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	// Verify max concurrent was respected (should be <= 2)
	if client.maxConcurrent > 2 {
		t.Errorf("Expected max concurrent <= 2, got %d", client.maxConcurrent)
	}
}

func TestParallelResolver_EmptyPackages(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")

	packages := []PackageDependency{}

	results, err := resolver.ResolveMultiple(context.Background(), packages)

	if err != nil {
		t.Fatalf("ResolveMultiple() with empty packages failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestParallelResolver_ResolveProjectParallel(t *testing.T) {
	client := &mockPackageMetadataClient{
		packages: map[string]*PackageDependencyInfo{
			"A|1.0.0": {
				ID:      "A",
				Version: "1.0.0",
				Dependencies: []PackageDependency{
					{ID: "B", VersionRange: "[1.0.0]"},
				},
			},
			"B|1.0.0": {ID: "B", Version: "1.0.0", Dependencies: []PackageDependency{}},
		},
	}

	resolver := NewResolver(client, []string{"source1"}, "net8.0")

	roots := []PackageDependency{
		{ID: "A", VersionRange: "[1.0.0]"},
	}

	result, err := resolver.parallelResolver.ResolveProjectParallel(context.Background(), roots)

	if err != nil {
		t.Fatalf("ResolveProjectParallel() failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Packages) == 0 {
		t.Error("Expected packages, got none")
	}
}

// slowMockClient simulates slow network
type slowMockClient struct {
	packages map[string]*PackageDependencyInfo
	delay    time.Duration
}

func (c *slowMockClient) GetPackageMetadata(
	ctx context.Context,
	source string,
	packageID string,
) ([]*PackageDependencyInfo, error) {
	select {
	case <-time.After(c.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	result := make([]*PackageDependencyInfo, 0)
	for _, pkg := range c.packages {
		if pkg.ID == packageID {
			result = append(result, pkg)
		}
	}
	return result, nil
}

// countingMockClient tracks concurrent requests
type countingMockClient struct {
	packages      map[string]*PackageDependencyInfo
	delay         time.Duration
	concurrent    int
	maxConcurrent int
	mu            sync.Mutex
}

func (c *countingMockClient) GetPackageMetadata(
	ctx context.Context,
	source string,
	packageID string,
) ([]*PackageDependencyInfo, error) {
	// Track concurrent requests
	c.mu.Lock()
	c.concurrent++
	if c.concurrent > c.maxConcurrent {
		c.maxConcurrent = c.concurrent
	}
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.concurrent--
		c.mu.Unlock()
	}()

	select {
	case <-time.After(c.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	result := make([]*PackageDependencyInfo, 0)
	for _, pkg := range c.packages {
		if pkg.ID == packageID {
			result = append(result, pkg)
		}
	}
	return result, nil
}
