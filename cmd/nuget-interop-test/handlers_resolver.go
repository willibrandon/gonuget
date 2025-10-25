package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

	// Walk the graph
	rootNode, err := walker.Walk(
		context.Background(),
		req.PackageID,
		req.VersionRange,
		req.TargetFramework,
	)
	if err != nil {
		return nil, fmt.Errorf("walk graph: %w", err)
	}

	// Convert GraphNode to response format
	resp := WalkGraphResponse{
		RootNode: convertGraphNode(rootNode),
	}

	return resp, nil
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

// convertGraphNode converts a resolver.GraphNode to the API response format.
func convertGraphNode(node *resolver.GraphNode) GraphNodeData {
	if node == nil {
		return GraphNodeData{}
	}

	result := GraphNodeData{
		Key:        node.Key,
		Depth:      node.Depth,
		InnerNodes: make([]GraphNodeData, 0, len(node.InnerNodes)),
	}

	// Set disposition
	result.Disposition = node.Disposition.String()

	// Set package info
	if node.Item != nil {
		result.PackageID = node.Item.ID
		result.Version = node.Item.Version
	}

	// Convert OuterEdge
	if node.OuterEdge != nil {
		result.OuterEdge = &GraphEdgeData{
			Dependency: DependencyData{
				ID:              node.OuterEdge.Edge.ID,
				VersionRange:    node.OuterEdge.Edge.VersionRange,
				TargetFramework: node.OuterEdge.Edge.TargetFramework,
			},
		}
		if node.OuterEdge.Item != nil {
			result.OuterEdge.ParentPackageID = node.OuterEdge.Item.ID
			result.OuterEdge.ParentVersion = node.OuterEdge.Item.Version
		}
	}

	// Convert child nodes recursively
	for _, child := range node.InnerNodes {
		result.InnerNodes = append(result.InnerNodes, convertGraphNode(child))
	}

	return result
}
