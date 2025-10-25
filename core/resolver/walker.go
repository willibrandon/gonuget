package resolver

import (
	"context"
	"fmt"
	"strings"

	"github.com/willibrandon/gonuget/version"
)

// DependencyWalker builds dependency graphs using stack-based traversal.
// Matches NuGet.Client's RemoteDependencyWalker.
type DependencyWalker struct {
	client            PackageMetadataClient
	sources           []string
	cache             *WalkerCache
	targetFramework   string
	frameworkSelector *FrameworkSelector
}

// PackageMetadataClient interface for fetching package metadata
type PackageMetadataClient interface {
	GetPackageMetadata(ctx context.Context, source string, packageID string) ([]*PackageDependencyInfo, error)
}

// NewDependencyWalker creates a new dependency walker
func NewDependencyWalker(client PackageMetadataClient, sources []string, targetFramework string) *DependencyWalker {
	return &DependencyWalker{
		client:            client,
		sources:           sources,
		cache:             NewWalkerCache(),
		targetFramework:   targetFramework,
		frameworkSelector: NewFrameworkSelector(),
	}
}

// Walk builds the complete dependency graph starting from the given package.
// Uses manual stack-based traversal matching NuGet.Client for performance.
func (w *DependencyWalker) Walk(
	ctx context.Context,
	packageID string,
	versionRange string,
	targetFramework string,
) (*GraphNode, error) {
	// Fetch root package
	rootInfo, err := w.fetchDependency(ctx, PackageDependency{
		ID:           packageID,
		VersionRange: versionRange,
	}, targetFramework)
	if err != nil {
		return nil, fmt.Errorf("fetch root package: %w", err)
	}

	if rootInfo == nil {
		return nil, fmt.Errorf("package not found: %s %s", packageID, versionRange)
	}

	// Create root node
	rootNode := &GraphNode{
		Key:         rootInfo.Key(),
		Item:        rootInfo,
		OuterNode:   nil,
		InnerNodes:  make([]*GraphNode, 0),
		ParentNodes: make([]*GraphNode, 0),
		Disposition: DispositionAcceptable,
		Depth:       0,
		OuterEdge:   nil,
	}

	// Use manual stack-based traversal (performance-critical)
	// This avoids recursive goroutine overhead for large graphs
	if err := w.walkStackBased(ctx, rootNode, targetFramework); err != nil {
		return nil, err
	}

	return rootNode, nil
}

// walkStackBased performs manual stack-based graph traversal.
// Matches RemoteDependencyWalker.CreateGraphNodeAsync behavior.
func (w *DependencyWalker) walkStackBased(
	ctx context.Context,
	rootNode *GraphNode,
	targetFramework string,
) error {
	// Initialize stack with root state
	stack := []*WalkerStackState{
		{
			Node:            rootNode,
			DependencyTasks: make([]*DependencyFetchTask, 0),
			Index:           0,
			OuterEdge:       nil,
		},
	}

	for len(stack) > 0 {
		// Pop current state
		state := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		node := state.Node
		index := state.Index

		// Phase 1: Start all dependency fetch operations in parallel
		// This matches NuGet.Client's approach of starting all async operations before awaiting
		if index == 0 && node.Item != nil {
			// Check if this node has SuppressParent=All, which means its dependencies should not be walked
			if state.OuterEdge != nil && state.OuterEdge.Edge.SuppressParent == LibraryIncludeFlagsAll {
				// Skip walking this node's dependencies (PrivateAssets="All")
				continue
			}

			deps := w.getDependenciesForFramework(node.Item, targetFramework)

			for _, dep := range deps {
				// Check if dependency should be processed
				result, _ := w.calculateDependencyResult(state.OuterEdge, dep, node.Key)

				if result == DependencyResultCycle {
					// Add cycle node
					cycleNode := &GraphNode{
						Key:         w.makeDependencyKey(dep),
						Item:        nil,
						OuterNode:   node,
						InnerNodes:  make([]*GraphNode, 0),
						ParentNodes: make([]*GraphNode, 0),
						Disposition: DispositionCycle,
						Depth:       node.Depth + 1,
						OuterEdge:   state.OuterEdge,
					}
					node.InnerNodes = append(node.InnerNodes, cycleNode)
					continue
				}

				if result == DependencyResultPotentiallyDowngraded {
					// Add downgrade node
					downgradeNode := &GraphNode{
						Key:         w.makeDependencyKey(dep),
						Item:        nil,
						OuterNode:   node,
						InnerNodes:  make([]*GraphNode, 0),
						ParentNodes: make([]*GraphNode, 0),
						Disposition: DispositionPotentiallyDowngraded,
						Depth:       node.Depth + 1,
						OuterEdge:   state.OuterEdge,
					}
					node.InnerNodes = append(node.InnerNodes, downgradeNode)
					continue
				}

				if result == DependencyResultEclipsed {
					// Skip eclipsed dependencies
					continue
				}

				// Create fetch task (start fetch operation but don't wait)
				task := &DependencyFetchTask{
					Dependency: dep,
					ResultChan: make(chan *DependencyFetchResult, 1),
					InnerEdge: &GraphEdge{
						OuterEdge: state.OuterEdge,
						Item:      node.Item,
						Edge:      dep,
					},
				}

				// Start fetch in background
				go func(t *DependencyFetchTask) {
					info, err := w.fetchDependency(ctx, t.Dependency, targetFramework)
					t.ResultChan <- &DependencyFetchResult{Info: info, Error: err}
				}(task)

				state.DependencyTasks = append(state.DependencyTasks, task)
			}
		}

		// Phase 2: Process one dependency at a time
		if index < len(state.DependencyTasks) {
			task := state.DependencyTasks[index]

			// Wait for fetch result
			select {
			case <-ctx.Done():
				return ctx.Err()
			case result := <-task.ResultChan:
				if result.Error != nil {
					return result.Error
				}

				if result.Info == nil {
					// Dependency not found - skip
					// Push parent state with index+1 to continue
					stack = append(stack, &WalkerStackState{
						Node:            node,
						DependencyTasks: state.DependencyTasks,
						Index:           index + 1,
						OuterEdge:       state.OuterEdge,
					})
					continue
				}

				// Create child node
				childNode := &GraphNode{
					Key:         result.Info.Key(),
					Item:        result.Info,
					OuterNode:   node,
					InnerNodes:  make([]*GraphNode, 0),
					ParentNodes: make([]*GraphNode, 0),
					Disposition: DispositionAcceptable,
					Depth:       node.Depth + 1,
					OuterEdge:   task.InnerEdge,
				}

				node.InnerNodes = append(node.InnerNodes, childNode)

				// Push parent state (with index+1 to continue siblings)
				stack = append(stack, &WalkerStackState{
					Node:            node,
					DependencyTasks: state.DependencyTasks,
					Index:           index + 1,
					OuterEdge:       state.OuterEdge,
				})

				// Push child state (with index=0 to start child's dependencies)
				stack = append(stack, &WalkerStackState{
					Node:            childNode,
					DependencyTasks: make([]*DependencyFetchTask, 0),
					Index:           0,
					OuterEdge:       task.InnerEdge,
				})
			}
		}
	}

	return nil
}

// calculateDependencyResult walks parent chain to check for cycles and downgrades.
// Matches RemoteDependencyWalker.WalkParentsAndCalculateDependencyResult.
func (w *DependencyWalker) calculateDependencyResult(
	edge *GraphEdge,
	dependency PackageDependency,
	currentKey string,
) (DependencyResult, *PackageDependency) {
	// Check for direct cycle (A -> B -> A)
	if strings.HasPrefix(currentKey, dependency.ID+"|") {
		return DependencyResultCycle, &dependency
	}

	// Walk up parent chain
	currentEdge := edge
	for currentEdge != nil {
		if currentEdge.Item == nil {
			currentEdge = currentEdge.OuterEdge
			continue
		}

		// Check for cycle
		if strings.HasPrefix(currentEdge.Item.Key(), dependency.ID+"|") {
			return DependencyResultCycle, &dependency
		}

		// TODO(M5.3): Implement full downgrade and eclipse detection
		// NuGet.Client's CalculateDependencyResult checks each dependency in currentEdge.Item.Dependencies:
		//   1. Check if childDependencyLibrary.IsEclipsedBy(d.LibraryRange)
		//   2. If eclipsed, check if IsGreaterThanOrEqualTo(d.LibraryRange.VersionRange, childDependencyLibrary.VersionRange)
		//   3. If not greater or equal, return DependencyResultPotentiallyDowngraded
		//   4. Otherwise return DependencyResultEclipsed
		// This will be implemented in M5.3: Version Conflict Detection

		currentEdge = currentEdge.OuterEdge
	}

	return DependencyResultAcceptable, nil
}

// getDependenciesForFramework returns dependencies applicable to target framework
func (w *DependencyWalker) getDependenciesForFramework(
	info *PackageDependencyInfo,
	targetFramework string,
) []PackageDependency {
	if len(info.DependencyGroups) > 0 {
		return w.frameworkSelector.SelectDependencies(info.DependencyGroups, targetFramework)
	}
	return info.Dependencies
}

// makeDependencyKey creates a key for a dependency
func (w *DependencyWalker) makeDependencyKey(dep PackageDependency) string {
	return fmt.Sprintf("%s|%s", dep.ID, dep.VersionRange)
}

// fetchDependency fetches metadata for a dependency
func (w *DependencyWalker) fetchDependency(
	ctx context.Context,
	dep PackageDependency,
	targetFramework string,
) (*PackageDependencyInfo, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s|%s|%s", dep.ID, dep.VersionRange, targetFramework)
	if cached := w.cache.Get(cacheKey); cached != nil {
		return cached, nil
	}

	// Parse version range
	versionRange, err := version.ParseVersionRange(dep.VersionRange)
	if err != nil {
		return nil, fmt.Errorf("parse version range %q: %w", dep.VersionRange, err)
	}

	// Try all sources
	for _, source := range w.sources {
		packages, err := w.client.GetPackageMetadata(ctx, source, dep.ID)
		if err != nil {
			continue
		}

		// Find best match for version range (return highest version that satisfies range)
		var bestMatch *PackageDependencyInfo
		for _, pkg := range packages {
			pkgVersion, err := version.Parse(pkg.Version)
			if err != nil {
				continue // Skip invalid versions
			}

			// Check if this version satisfies the range
			if versionRange.Satisfies(pkgVersion) {
				// Keep the highest satisfying version
				if bestMatch == nil {
					bestMatch = pkg
				} else {
					bestVersion, _ := version.Parse(bestMatch.Version)
					if pkgVersion.Compare(bestVersion) > 0 {
						bestMatch = pkg
					}
				}
			}
		}

		if bestMatch != nil {
			w.cache.Set(cacheKey, bestMatch)
			return bestMatch, nil
		}
	}

	return nil, nil // Not found
}
