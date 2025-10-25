package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/willibrandon/gonuget/core/resolver"
	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/protocol/v3"
)

// WalkGraphHandler walks the dependency graph for a package.
type WalkGraphHandler struct{}

func (h *WalkGraphHandler) ErrorCode() string { return "WALK_001" }

func (h *WalkGraphHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req WalkGraphRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.PackageID == "" {
		return nil, fmt.Errorf("packageId is required")
	}
	if req.VersionRange == "" {
		return nil, fmt.Errorf("versionRange is required")
	}
	if req.TargetFramework == "" {
		return nil, fmt.Errorf("targetFramework is required")
	}
	if len(req.Sources) == 0 {
		return nil, fmt.Errorf("sources is required")
	}

	// Create real NuGet V3 metadata client
	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	v3Client := v3.NewMetadataClient(httpClient, serviceIndexClient)

	// Wrap V3 client to implement resolver.PackageMetadataClient
	client := &v3MetadataClientAdapter{v3Client: v3Client}

	// Create walker
	walker := resolver.NewDependencyWalker(client, req.Sources, req.TargetFramework)

	// Walk the graph (recursive=true for full transitive resolution)
	rootNode, err := walker.Walk(
		context.Background(),
		req.PackageID,
		req.VersionRange,
		req.TargetFramework,
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("walk graph: %w", err)
	}

	// Detect conflicts and downgrades
	conflictDetector := &resolver.ConflictDetector{}
	_, downgrades := conflictDetector.DetectFromGraph(rootNode)

	// Collect all nodes in flat array format
	nodes := make([]GraphNodeData, 0)
	collectNodesFlat(rootNode, &nodes)

	// Collect cycles (package IDs that form circular dependencies)
	cycles := make([]string, 0)
	collectCycles(rootNode, &cycles)

	// Convert downgrades to response format
	downgradeInfos := make([]DowngradeInfo, len(downgrades))
	for i, dw := range downgrades {
		downgradeInfos[i] = DowngradeInfo{
			PackageID:   dw.PackageID,
			FromVersion: dw.CurrentVersion,
			ToVersion:   dw.TargetVersion,
		}
	}

	// Build response with flat arrays
	resp := WalkGraphResponse{
		Nodes:      nodes,
		Cycles:     cycles,
		Downgrades: downgradeInfos,
	}

	return resp, nil
}

// ResolveConflictsHandler resolves version conflicts in a dependency set.
type ResolveConflictsHandler struct{}

func (h *ResolveConflictsHandler) ErrorCode() string { return "RESOLVE_001" }

func (h *ResolveConflictsHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ResolveConflictsRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate input
	if len(req.PackageIDs) == 0 {
		return nil, fmt.Errorf("packageIds is required")
	}
	if len(req.VersionRanges) != len(req.PackageIDs) {
		return nil, fmt.Errorf("versionRanges must match packageIds length")
	}
	if req.TargetFramework == "" {
		return nil, fmt.Errorf("targetFramework is required")
	}

	// Use default source (NuGet.org V3) if not provided
	sources := []string{"https://api.nuget.org/v3/index.json"}

	// Create real NuGet V3 metadata client
	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	v3Client := v3.NewMetadataClient(httpClient, serviceIndexClient)

	// Wrap V3 client to implement resolver.PackageMetadataClient
	client := &v3MetadataClientAdapter{v3Client: v3Client}

	// For now, resolve each package individually and combine results
	// In practice, we'd want to walk all packages together to detect cross-package conflicts
	allPackages := make(map[string]ResolvedPackage) // Use map to deduplicate

	for i, packageID := range req.PackageIDs {
		versionRange := req.VersionRanges[i]

		// Create resolver
		r := resolver.NewResolver(client, sources, req.TargetFramework)

		// Resolve the package
		result, err := r.Resolve(context.Background(), packageID, versionRange)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", packageID, err)
		}

		// Add all resolved packages to the result
		for _, pkg := range result.Packages {
			// Calculate depth (0 for root, traverse OuterEdge chain for others)
			depth := 0
			// In the current Resolver implementation, we don't track depth in Packages
			// For now, root packages have depth 0, all transitive deps have depth > 0
			// We can infer: if packageID matches input, depth is 0
			if pkg.ID == packageID {
				depth = 0
			} else {
				depth = 1 // Simplified: all transitive deps get depth 1+
			}

			// Use package (don't try to pick "latest" - just use what Resolver selected)
			// The Resolver already ran conflict resolution and picked the correct version
			if _, found := allPackages[pkg.ID]; !found {
				allPackages[pkg.ID] = ResolvedPackage{
					PackageID: pkg.ID,
					Version:   pkg.Version,
					Depth:     depth,
				}
			}
		}
	}

	// Convert map to slice
	packages := make([]ResolvedPackage, 0, len(allPackages))
	for _, pkg := range allPackages {
		packages = append(packages, pkg)
	}

	return ResolveConflictsResponse{Packages: packages}, nil
}

// collectNodesFlat recursively collects all nodes into a flat array.
func collectNodesFlat(node *resolver.GraphNode, nodes *[]GraphNodeData) {
	if node == nil {
		return
	}

	// Collect dependency IDs
	deps := make([]string, 0, len(node.InnerNodes))
	for _, child := range node.InnerNodes {
		if child.Item != nil {
			deps = append(deps, child.Item.ID)
		}
	}

	// Add current node
	nodeData := GraphNodeData{
		PackageID:    "",
		Version:      "",
		Disposition:  node.Disposition.String(),
		Depth:        node.Depth,
		Dependencies: deps,
	}

	if node.Item != nil {
		nodeData.PackageID = node.Item.ID
		nodeData.Version = node.Item.Version
	}

	*nodes = append(*nodes, nodeData)

	// Recursively collect children
	for _, child := range node.InnerNodes {
		collectNodesFlat(child, nodes)
	}
}

// collectCycles collects package IDs that have Cycle disposition.
func collectCycles(node *resolver.GraphNode, cycles *[]string) {
	if node == nil {
		return
	}

	// Check if this node is a cycle
	if node.Disposition == resolver.DispositionCycle && node.Item != nil {
		*cycles = append(*cycles, node.Item.ID)
	}

	// Recursively check children
	for _, child := range node.InnerNodes {
		collectCycles(child, cycles)
	}
}

// v3MetadataClientAdapter adapts v3.MetadataClient to resolver.PackageMetadataClient.
type v3MetadataClientAdapter struct {
	v3Client *v3.MetadataClient
}

// GetPackageMetadata implements resolver.PackageMetadataClient by fetching from NuGet V3 API.
func (a *v3MetadataClientAdapter) GetPackageMetadata(ctx context.Context, source string, packageID string) ([]*resolver.PackageDependencyInfo, error) {
	// Fetch registration index from V3 API
	index, err := a.v3Client.GetPackageMetadata(ctx, source, packageID)
	if err != nil {
		return nil, err
	}

	// Convert all versions to PackageDependencyInfo
	var packages []*resolver.PackageDependencyInfo
	for _, page := range index.Items {
		for _, leaf := range page.Items {
			if leaf.CatalogEntry == nil {
				continue
			}

			pkg := &resolver.PackageDependencyInfo{
				ID:               leaf.CatalogEntry.PackageID,
				Version:          leaf.CatalogEntry.Version,
				DependencyGroups: make([]resolver.DependencyGroup, 0, len(leaf.CatalogEntry.DependencyGroups)),
			}

			// Convert dependency groups
			for _, v3Group := range leaf.CatalogEntry.DependencyGroups {
				group := resolver.DependencyGroup{
					TargetFramework: normalizeFramework(v3Group.TargetFramework),
					Dependencies:    make([]resolver.PackageDependency, 0, len(v3Group.Dependencies)),
				}

				// Convert dependencies
				for _, v3Dep := range v3Group.Dependencies {
					dep := resolver.PackageDependency{
						ID:              v3Dep.ID,
						VersionRange:    v3Dep.Range,
						TargetFramework: group.TargetFramework,
					}
					group.Dependencies = append(group.Dependencies, dep)
				}

				pkg.DependencyGroups = append(pkg.DependencyGroups, group)
			}

			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

// normalizeFramework normalizes framework strings to match NuGet.Client format.
func normalizeFramework(fw string) string {
	if fw == "" {
		return ""
	}
	// V3 API returns frameworks like ".NETCoreApp3.1" but we need "netcoreapp3.1"
	fw = strings.ToLower(fw)
	fw = strings.TrimPrefix(fw, ".")
	fw = strings.ReplaceAll(fw, " ", "")
	return fw
}

// AnalyzeCyclesHandler analyzes dependency graph for cycles (M5.5).
type AnalyzeCyclesHandler struct{}

func (h *AnalyzeCyclesHandler) ErrorCode() string { return "CYCLES_001" }

func (h *AnalyzeCyclesHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req AnalyzeCyclesRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.PackageID == "" {
		return nil, fmt.Errorf("packageId is required")
	}
	if req.VersionRange == "" {
		return nil, fmt.Errorf("versionRange is required")
	}
	if req.TargetFramework == "" {
		return nil, fmt.Errorf("targetFramework is required")
	}
	if len(req.Sources) == 0 {
		return nil, fmt.Errorf("sources is required")
	}

	// Choose client based on whether in-memory packages are provided
	var client resolver.PackageMetadataClient
	if len(req.InMemoryPackages) > 0 {
		// Use in-memory client for testing (e.g., cycle detection)
		inMemoryClient, err := NewInMemoryPackageClient(req.InMemoryPackages, req.TargetFramework)
		if err != nil {
			return nil, fmt.Errorf("create in-memory client: %w", err)
		}
		client = inMemoryClient
	} else {
		// Create real NuGet V3 metadata client
		httpClient := nugethttp.NewClient(nil)
		serviceIndexClient := v3.NewServiceIndexClient(httpClient)
		v3Client := v3.NewMetadataClient(httpClient, serviceIndexClient)
		client = &v3MetadataClientAdapter{v3Client: v3Client}
	}

	// Create walker
	walker := resolver.NewDependencyWalker(client, req.Sources, req.TargetFramework)

	// Walk the graph (recursive=true for full transitive resolution)
	rootNode, err := walker.Walk(
		context.Background(),
		req.PackageID,
		req.VersionRange,
		req.TargetFramework,
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("walk graph: %w", err)
	}

	// Analyze cycles
	cycleAnalyzer := resolver.NewCycleAnalyzer()
	cycles := cycleAnalyzer.AnalyzeCycles(rootNode)

	// Convert to response format
	cycleInfos := make([]CycleInfo, len(cycles))
	for i, cycle := range cycles {
		// Build package IDs array from path + the cycle node itself
		packageIDs := make([]string, len(cycle.PathToSelf)+1)
		copy(packageIDs, cycle.PathToSelf)
		packageIDs[len(packageIDs)-1] = cycle.PackageID

		cycleInfos[i] = CycleInfo{
			PackageIDs: packageIDs,
			Length:     len(packageIDs),
		}
	}

	return AnalyzeCyclesResponse{Cycles: cycleInfos}, nil
}

// ResolveTransitiveHandler resolves transitive dependencies (M5.6).
type ResolveTransitiveHandler struct{}

func (h *ResolveTransitiveHandler) ErrorCode() string { return "TRANSITIVE_001" }

func (h *ResolveTransitiveHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ResolveTransitiveRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate input
	if len(req.RootPackages) == 0 {
		return nil, fmt.Errorf("rootPackages is required")
	}
	if req.TargetFramework == "" {
		return nil, fmt.Errorf("targetFramework is required")
	}
	if len(req.Sources) == 0 {
		return nil, fmt.Errorf("sources is required")
	}

	// Create real NuGet V3 metadata client
	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	v3Client := v3.NewMetadataClient(httpClient, serviceIndexClient)
	client := &v3MetadataClientAdapter{v3Client: v3Client}

	// Create resolver
	r := resolver.NewResolver(client, req.Sources, req.TargetFramework)

	// Convert request to PackageDependency slice
	deps := make([]resolver.PackageDependency, len(req.RootPackages))
	for i, pkg := range req.RootPackages {
		deps[i] = resolver.PackageDependency{
			ID:           pkg.ID,
			VersionRange: pkg.VersionRange,
		}
	}

	// Resolve project
	result, err := r.ResolveProject(context.Background(), deps)
	if err != nil {
		return nil, fmt.Errorf("resolve transitive: %w", err)
	}

	// Convert packages
	packages := make([]ResolvedPackage, len(result.Packages))
	for i, pkg := range result.Packages {
		packages[i] = ResolvedPackage{
			PackageID: pkg.ID,
			Version:   pkg.Version,
			Depth:     0, // TODO: Track depth in resolver
		}
	}

	// Convert conflicts
	conflicts := make([]ConflictInfo, len(result.Conflicts))
	for i, conflict := range result.Conflicts {
		// Determine winner version (the one that was actually resolved)
		// In gonuget, the resolver picks the highest version that satisfies all constraints
		winnerVersion := ""
		if len(conflict.Versions) > 0 {
			// The first version in the list is typically the winner
			winnerVersion = conflict.Versions[0]
		}

		conflicts[i] = ConflictInfo{
			PackageID:     conflict.PackageID,
			Versions:      conflict.Versions,
			WinnerVersion: winnerVersion,
		}
	}

	// Convert downgrades
	downgrades := make([]DowngradeInfo, len(result.Downgrades))
	for i, dw := range result.Downgrades {
		downgrades[i] = DowngradeInfo{
			PackageID:   dw.PackageID,
			FromVersion: dw.CurrentVersion,
			ToVersion:   dw.TargetVersion,
		}
	}

	return ResolveTransitiveResponse{
		Packages:   packages,
		Conflicts:  conflicts,
		Downgrades: downgrades,
	}, nil
}

// BenchmarkCacheHandler benchmarks cache deduplication (M5.7).
type BenchmarkCacheHandler struct{}

func (h *BenchmarkCacheHandler) ErrorCode() string { return "CACHE_001" }

func (h *BenchmarkCacheHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req BenchmarkCacheRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate input
	if req.PackageID == "" {
		return nil, fmt.Errorf("packageId is required")
	}
	if req.VersionRange == "" {
		return nil, fmt.Errorf("versionRange is required")
	}
	if req.TargetFramework == "" {
		return nil, fmt.Errorf("targetFramework is required")
	}
	if len(req.Sources) == 0 {
		return nil, fmt.Errorf("sources is required")
	}
	if req.ConcurrentRequests <= 0 {
		req.ConcurrentRequests = 10
	}

	// Create counting metadata client to track actual fetches
	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	v3Client := v3.NewMetadataClient(httpClient, serviceIndexClient)
	countingClient := &countingMetadataClient{
		v3Client: v3Client,
	}

	// Create resolver with caching enabled
	r := resolver.NewResolver(countingClient, req.Sources, req.TargetFramework)

	// Make concurrent requests
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < req.ConcurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = r.Resolve(context.Background(), req.PackageID, req.VersionRange)
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	return BenchmarkCacheResponse{
		TotalRequests:       req.ConcurrentRequests,
		ActualFetches:       countingClient.fetchCount,
		DurationMs:          duration.Milliseconds(),
		DeduplicationWorked: countingClient.fetchCount == 1,
	}, nil
}

// ResolveWithTTLHandler resolves package with cache TTL (M5.7).
type ResolveWithTTLHandler struct{}

func (h *ResolveWithTTLHandler) ErrorCode() string { return "TTL_001" }

func (h *ResolveWithTTLHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ResolveWithTTLRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate input
	if req.PackageID == "" {
		return nil, fmt.Errorf("packageId is required")
	}
	if req.VersionRange == "" {
		return nil, fmt.Errorf("versionRange is required")
	}
	if req.TargetFramework == "" {
		return nil, fmt.Errorf("targetFramework is required")
	}
	if len(req.Sources) == 0 {
		return nil, fmt.Errorf("sources is required")
	}

	// Create client
	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	v3Client := v3.NewMetadataClient(httpClient, serviceIndexClient)
	client := &v3MetadataClientAdapter{v3Client: v3Client}

	// Create resolver with TTL
	r := resolver.NewResolver(client, req.Sources, req.TargetFramework)
	// TODO: Set TTL on walker's operation cache

	// Resolve
	result, err := r.Resolve(context.Background(), req.PackageID, req.VersionRange)
	if err != nil {
		return nil, fmt.Errorf("resolve: %w", err)
	}

	// Convert packages
	packages := make([]ResolvedPackage, len(result.Packages))
	for i, pkg := range result.Packages {
		packages[i] = ResolvedPackage{
			PackageID: pkg.ID,
			Version:   pkg.Version,
			Depth:     0,
		}
	}

	return ResolveWithTTLResponse{
		Packages:  packages,
		WasCached: false, // TODO: Track cache hits
	}, nil
}

// BenchmarkParallelHandler benchmarks parallel resolution (M5.8).
type BenchmarkParallelHandler struct{}

func (h *BenchmarkParallelHandler) ErrorCode() string { return "PARALLEL_001" }

func (h *BenchmarkParallelHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req BenchmarkParallelRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate input
	if len(req.PackageSpecs) == 0 {
		return nil, fmt.Errorf("packageSpecs is required")
	}
	if req.TargetFramework == "" {
		return nil, fmt.Errorf("targetFramework is required")
	}
	if len(req.Sources) == 0 {
		return nil, fmt.Errorf("sources is required")
	}

	// Create client
	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	v3Client := v3.NewMetadataClient(httpClient, serviceIndexClient)
	client := &v3MetadataClientAdapter{v3Client: v3Client}

	// Create resolver
	r := resolver.NewResolver(client, req.Sources, req.TargetFramework)

	// Convert request to PackageDependency slice
	deps := make([]resolver.PackageDependency, len(req.PackageSpecs))
	for i, pkg := range req.PackageSpecs {
		deps[i] = resolver.PackageDependency{
			ID:           pkg.ID,
			VersionRange: pkg.VersionRange,
		}
	}

	// Determine if recursive resolution is needed (default: true)
	recursive := true
	if req.Recursive != nil {
		recursive = *req.Recursive
	}

	// Resolve
	start := time.Now()
	var results []*resolver.ResolutionResult
	var err error

	if req.Sequential {
		// Resolve sequentially
		results = make([]*resolver.ResolutionResult, len(deps))
		for i, dep := range deps {
			if recursive {
				results[i], err = r.Resolve(context.Background(), dep.ID, dep.VersionRange)
			} else {
				results[i], err = r.ResolveNonRecursive(context.Background(), dep.ID, dep.VersionRange)
			}
			if err != nil {
				return nil, fmt.Errorf("resolve %s: %w", dep.ID, err)
			}
		}
	} else {
		// Resolve in parallel
		if recursive {
			results, err = r.ResolveMultiple(context.Background(), deps)
		} else {
			// For non-recursive parallel resolution, resolve each package independently
			results = make([]*resolver.ResolutionResult, len(deps))
			errChan := make(chan error, len(deps))
			for i, dep := range deps {
				go func(index int, d resolver.PackageDependency) {
					result, resolveErr := r.ResolveNonRecursive(context.Background(), d.ID, d.VersionRange)
					if resolveErr != nil {
						errChan <- resolveErr
					} else {
						results[index] = result
						errChan <- nil
					}
				}(i, dep)
			}
			// Wait for all
			for range deps {
				if e := <-errChan; e != nil {
					err = e
					break
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("resolve multiple: %w", err)
		}
	}

	duration := time.Since(start)

	// Count total packages
	totalPackages := 0
	for _, result := range results {
		totalPackages += len(result.Packages)
	}

	return BenchmarkParallelResponse{
		PackageCount: totalPackages,
		DurationMs:   duration.Milliseconds(),
		WasParallel:  !req.Sequential,
	}, nil
}

// ResolveWithWorkerLimitHandler resolves packages with worker pool limit (M5.8).
type ResolveWithWorkerLimitHandler struct{}

func (h *ResolveWithWorkerLimitHandler) ErrorCode() string { return "WORKER_001" }

func (h *ResolveWithWorkerLimitHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ResolveWithWorkerLimitRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate input
	if len(req.PackageSpecs) == 0 {
		return nil, fmt.Errorf("packageSpecs is required")
	}
	if req.TargetFramework == "" {
		return nil, fmt.Errorf("targetFramework is required")
	}
	if len(req.Sources) == 0 {
		return nil, fmt.Errorf("sources is required")
	}
	if req.MaxWorkers <= 0 {
		req.MaxWorkers = 10
	}

	// Create clients
	httpClient := nugethttp.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	v3Client := v3.NewMetadataClient(httpClient, serviceIndexClient)
	adapter := &v3MetadataClientAdapter{v3Client: v3Client}

	// Create resolver
	r := resolver.NewResolver(adapter, req.Sources, req.TargetFramework)

	// Create parallel resolver with concurrency tracker
	tracker := &concurrencyTracker{}
	parallelResolver := resolver.NewParallelResolver(r, req.MaxWorkers).WithTracker(tracker)
	r.ReplaceParallelResolver(parallelResolver)

	// Convert request to PackageDependency slice
	deps := make([]resolver.PackageDependency, len(req.PackageSpecs))
	for i, pkg := range req.PackageSpecs {
		deps[i] = resolver.PackageDependency{
			ID:           pkg.ID,
			VersionRange: pkg.VersionRange,
		}
	}

	// Resolve in parallel
	results, err := r.ResolveMultiple(context.Background(), deps)
	if err != nil {
		return nil, fmt.Errorf("resolve multiple: %w", err)
	}

	// Convert results
	resolveResults := make([]ResolveResult, len(results))
	for i, result := range results {
		packages := make([]ResolvedPackage, len(result.Packages))
		for j, pkg := range result.Packages {
			packages[j] = ResolvedPackage{
				PackageID: pkg.ID,
				Version:   pkg.Version,
				Depth:     0,
			}
		}

		resolveResults[i] = ResolveResult{
			PackageID: deps[i].ID,
			Packages:  packages,
		}
	}

	return ResolveWithWorkerLimitResponse{
		Results:       resolveResults,
		MaxConcurrent: tracker.maxConcurrent,
	}, nil
}

// concurrencyTracker implements resolver.ConcurrencyTracker to track concurrent operations.
type concurrencyTracker struct {
	mu            sync.Mutex
	concurrent    int
	maxConcurrent int
}

func (t *concurrencyTracker) Enter() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.concurrent++
	if t.concurrent > t.maxConcurrent {
		t.maxConcurrent = t.concurrent
	}
}

func (t *concurrencyTracker) Exit() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.concurrent--
}

// countingMetadataClient wraps v3MetadataClientAdapter to count fetches and track concurrency.
type countingMetadataClient struct {
	v3Client      *v3.MetadataClient
	fetchCount    int
	mu            sync.Mutex
	concurrent    int
	maxConcurrent int
}

func (c *countingMetadataClient) GetPackageMetadata(ctx context.Context, source string, packageID string) ([]*resolver.PackageDependencyInfo, error) {
	// Track concurrency
	c.mu.Lock()
	c.concurrent++
	c.fetchCount++
	if c.concurrent > c.maxConcurrent {
		c.maxConcurrent = c.concurrent
	}
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.concurrent--
		c.mu.Unlock()
	}()

	// Delegate to real client
	adapter := &v3MetadataClientAdapter{v3Client: c.v3Client}
	return adapter.GetPackageMetadata(ctx, source, packageID)
}
