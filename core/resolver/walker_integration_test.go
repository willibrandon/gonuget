package resolver

import (
	"context"
	"testing"

	"github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/protocol/v3"
)

// v3MetadataClientAdapter adapts v3.MetadataClient to PackageMetadataClient interface
type v3MetadataClientAdapter struct {
	v3Client *v3.MetadataClient
}

func (a *v3MetadataClientAdapter) GetPackageMetadata(ctx context.Context, source string, packageID string) ([]*PackageDependencyInfo, error) {
	// Fetch registration index from V3 API
	index, err := a.v3Client.GetPackageMetadata(ctx, source, packageID)
	if err != nil {
		return nil, err
	}

	// Convert all versions to PackageDependencyInfo
	var packages []*PackageDependencyInfo
	for _, page := range index.Items {
		for _, leaf := range page.Items {
			if leaf.CatalogEntry == nil {
				continue
			}

			pkg := &PackageDependencyInfo{
				ID:               leaf.CatalogEntry.PackageID,
				Version:          leaf.CatalogEntry.Version,
				DependencyGroups: make([]DependencyGroup, 0, len(leaf.CatalogEntry.DependencyGroups)),
			}

			// Convert dependency groups
			for _, v3Group := range leaf.CatalogEntry.DependencyGroups {
				group := DependencyGroup{
					TargetFramework: normalizeFramework(v3Group.TargetFramework),
					Dependencies:    make([]PackageDependency, 0, len(v3Group.Dependencies)),
				}

				// Convert dependencies
				for _, v3Dep := range v3Group.Dependencies {
					dep := PackageDependency{
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

func normalizeFramework(fw string) string {
	if fw == "" {
		return ""
	}
	// Simple normalization for now - M5.2 will implement full FrameworkReducer
	return fw
}

// TestWalkRealPackage_NewtonsoftJson walks a real package graph from NuGet.org
func TestWalkRealPackage_NewtonsoftJson(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup real V3 client
	httpClient := http.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	metadataClient := v3.NewMetadataClient(httpClient, serviceIndexClient)
	adapter := &v3MetadataClientAdapter{v3Client: metadataClient}

	walker := NewDependencyWalker(adapter, []string{"https://api.nuget.org/v3/index.json"}, "net8.0")

	// Walk Newtonsoft.Json 13.0.1 (has no dependencies)
	node, err := walker.Walk(context.Background(), "Newtonsoft.Json", "[13.0.1]", "net8.0", true)

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Verify root node
	if node.Item.ID != "Newtonsoft.Json" {
		t.Errorf("Expected root package Newtonsoft.Json, got %s", node.Item.ID)
	}

	if node.Item.Version != "13.0.1" {
		t.Errorf("Expected version 13.0.1, got %s", node.Item.Version)
	}

	// Verify Disposition is Acceptable
	if node.Disposition != DispositionAcceptable {
		t.Errorf("Expected root Disposition=Acceptable, got %v", node.Disposition)
	}

	// Newtonsoft.Json 13.0.1 has no dependencies
	if len(node.InnerNodes) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(node.InnerNodes))
	}

	// Verify depth
	if node.Depth != 0 {
		t.Errorf("Expected root depth 0, got %d", node.Depth)
	}

	// Root should have no OuterEdge
	if node.OuterEdge != nil {
		t.Error("Expected root to have nil OuterEdge")
	}
}

// TestWalkRealPackage_WithDependencies walks a package with dependencies
func TestWalkRealPackage_WithDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup real V3 client
	httpClient := http.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	metadataClient := v3.NewMetadataClient(httpClient, serviceIndexClient)
	adapter := &v3MetadataClientAdapter{v3Client: metadataClient}

	walker := NewDependencyWalker(adapter, []string{"https://api.nuget.org/v3/index.json"}, "net8.0")

	// Walk Microsoft.Extensions.Logging 8.0.0 (has dependencies)
	node, err := walker.Walk(context.Background(), "Microsoft.Extensions.Logging", "[8.0.0]", "net8.0", true)

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Verify root node
	if node.Item.ID != "Microsoft.Extensions.Logging" {
		t.Errorf("Expected root package Microsoft.Extensions.Logging, got %s", node.Item.ID)
	}

	// Verify Disposition states are set correctly
	if node.Disposition != DispositionAcceptable {
		t.Errorf("Expected root Disposition=Acceptable, got %v", node.Disposition)
	}

	// Should have dependencies
	if len(node.InnerNodes) == 0 {
		t.Error("Expected dependencies for Microsoft.Extensions.Logging 8.0.0")
	}

	// Verify child nodes have correct dispositions
	for _, child := range node.InnerNodes {
		if child.Disposition != DispositionAcceptable && child.Disposition != DispositionAccepted {
			t.Errorf("Expected child %s to have Acceptable/Accepted disposition, got %v",
				child.Item.ID, child.Disposition)
		}

		// Verify depth
		if child.Depth != 1 {
			t.Errorf("Expected child %s depth 1, got %d", child.Item.ID, child.Depth)
		}

		// Verify GraphEdge parent chain
		if child.OuterEdge == nil {
			t.Errorf("Expected child %s to have OuterEdge", child.Item.ID)
			continue
		}

		if child.OuterEdge.Item.ID != "Microsoft.Extensions.Logging" {
			t.Errorf("Expected child %s OuterEdge.Item to be Microsoft.Extensions.Logging, got %s",
				child.Item.ID, child.OuterEdge.Item.ID)
		}

		// Verify path from root
		path := child.PathFromRoot()
		if len(path) != 2 {
			t.Errorf("Expected child %s path length 2, got %d", child.Item.ID, len(path))
		}
		if len(path) > 0 && path[0] != "Microsoft.Extensions.Logging 8.0.0" {
			t.Errorf("Expected path[0] = Microsoft.Extensions.Logging 8.0.0, got %s", path[0])
		}
	}
}

// TestWalkRealPackage_DeepGraph walks a package with deep dependency tree
func TestWalkRealPackage_DeepGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup real V3 client
	httpClient := http.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	metadataClient := v3.NewMetadataClient(httpClient, serviceIndexClient)
	adapter := &v3MetadataClientAdapter{v3Client: metadataClient}

	walker := NewDependencyWalker(adapter, []string{"https://api.nuget.org/v3/index.json"}, "net8.0")

	// Walk a package with a deeper dependency tree
	node, err := walker.Walk(context.Background(), "Microsoft.Extensions.DependencyInjection", "[8.0.0]", "net8.0", true)

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Verify all nodes in graph have correct disposition states
	var validateNode func(*GraphNode, int)
	validateNode = func(n *GraphNode, expectedDepth int) {
		if n.Depth != expectedDepth {
			t.Errorf("Node %s expected depth %d, got %d", n.Item.ID, expectedDepth, n.Depth)
		}

		// All nodes should be Acceptable or Accepted (no cycles or conflicts in this graph)
		if n.Disposition != DispositionAcceptable && n.Disposition != DispositionAccepted && n.Disposition != DispositionCycle {
			t.Errorf("Node %s has unexpected disposition %v", n.Item.ID, n.Disposition)
		}

		// Verify OuterEdge chain is correct (except for root)
		if expectedDepth > 0 {
			if n.OuterEdge == nil {
				t.Errorf("Node %s at depth %d should have OuterEdge", n.Item.ID, expectedDepth)
			} else {
				// Verify parent chain depth
				chainDepth := 0
				edge := n.OuterEdge
				for edge != nil {
					chainDepth++
					edge = edge.OuterEdge
				}
				if chainDepth != expectedDepth {
					t.Errorf("Node %s expected edge chain depth %d, got %d", n.Item.ID, expectedDepth, chainDepth)
				}
			}
		}

		// Recursively validate children
		for _, child := range n.InnerNodes {
			if child.Item != nil {
				validateNode(child, expectedDepth+1)
			}
		}
	}

	validateNode(node, 0)
}

// TestWalkRealPackage_VerifyNoCycles ensures real package graphs don't have cycles
func TestWalkRealPackage_VerifyNoCycles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup real V3 client
	httpClient := http.NewClient(nil)
	serviceIndexClient := v3.NewServiceIndexClient(httpClient)
	metadataClient := v3.NewMetadataClient(httpClient, serviceIndexClient)
	adapter := &v3MetadataClientAdapter{v3Client: metadataClient}

	walker := NewDependencyWalker(adapter, []string{"https://api.nuget.org/v3/index.json"}, "net8.0")

	// Walk a real package - NuGet packages should not have cycles
	node, err := walker.Walk(context.Background(), "Microsoft.Extensions.Logging", "[8.0.0]", "net8.0", true)

	if err != nil {
		t.Fatalf("Walk() failed: %v", err)
	}

	// Count cycle nodes
	cycleCount := 0
	var countCycles func(*GraphNode)
	countCycles = func(n *GraphNode) {
		if n.Disposition == DispositionCycle {
			cycleCount++
		}
		for _, child := range n.InnerNodes {
			countCycles(child)
		}
	}

	countCycles(node)

	// Real NuGet packages should not have cycles
	if cycleCount > 0 {
		t.Errorf("Expected 0 cycle nodes in real package graph, found %d", cycleCount)
	}
}
