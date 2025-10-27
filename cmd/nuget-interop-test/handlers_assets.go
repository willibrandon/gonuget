package main

import (
	"encoding/json"
	"fmt"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/packaging/assets"
)

// FindRuntimeAssembliesHandler finds runtime assemblies from package paths.
// If targetFramework is provided, uses asset selection (best match only).
// If targetFramework is empty, uses pattern matching (all matches).
type FindRuntimeAssembliesHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *FindRuntimeAssembliesHandler) ErrorCode() string { return "ASSET_RT_001" }

// Handle processes the request.
func (h *FindRuntimeAssembliesHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req FindAssembliesRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if len(req.Paths) == 0 {
		return nil, fmt.Errorf("paths is required")
	}

	conventions := assets.NewManagedCodeConventions()
	resp := NewFindAssembliesResponse()

	if req.TargetFramework != "" {
		// Asset selection mode: Find best matching assemblies for target framework
		targetFw, err := frameworks.ParseFramework(req.TargetFramework)
		if err != nil {
			return nil, fmt.Errorf("parse target framework: %w", err)
		}

		paths := assets.GetLibItems(req.Paths, targetFw, conventions)
		for _, path := range paths {
			resp.Items = append(resp.Items, ContentItemData{
				Path:       path,
				Properties: make(map[string]interface{}),
			})
		}
	} else {
		// Pattern matching mode: Match all paths against runtime assembly patterns
		for _, path := range req.Paths {
			for _, expr := range conventions.RuntimeAssemblies.PathExpressions {
				if item := expr.Match(path, conventions.Properties); item != nil {
					resp.Items = append(resp.Items, contentItemToData(item))
					break // Only match first pattern
				}
			}
		}
	}

	return resp, nil
}

// FindCompileAssembliesHandler finds compile reference assemblies from package paths.
// If targetFramework is provided, uses asset selection (best match only, ref/ takes precedence over lib/).
// If targetFramework is empty, uses pattern matching (all matches from both ref/ and lib/).
type FindCompileAssembliesHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *FindCompileAssembliesHandler) ErrorCode() string { return "ASSET_CP_001" }

// Handle processes the request.
func (h *FindCompileAssembliesHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req FindAssembliesRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if len(req.Paths) == 0 {
		return nil, fmt.Errorf("paths is required")
	}

	conventions := assets.NewManagedCodeConventions()
	resp := NewFindAssembliesResponse()

	if req.TargetFramework != "" {
		// Asset selection mode: Find best matching assemblies (ref/ takes precedence)
		targetFw, err := frameworks.ParseFramework(req.TargetFramework)
		if err != nil {
			return nil, fmt.Errorf("parse target framework: %w", err)
		}

		paths := assets.GetRefItems(req.Paths, targetFw, conventions)
		for _, path := range paths {
			resp.Items = append(resp.Items, ContentItemData{
				Path:       path,
				Properties: make(map[string]interface{}),
			})
		}
	} else {
		// Pattern matching mode: Match all paths against compile assembly patterns (ref/ + lib/)
		for _, path := range req.Paths {
			// Try ref/ patterns first
			matched := false
			for _, expr := range conventions.CompileRefAssemblies.PathExpressions {
				if item := expr.Match(path, conventions.Properties); item != nil {
					resp.Items = append(resp.Items, contentItemToData(item))
					matched = true
					break
				}
			}

			// If not matched, try lib/ patterns
			if !matched {
				for _, expr := range conventions.CompileLibAssemblies.PathExpressions {
					if item := expr.Match(path, conventions.Properties); item != nil {
						resp.Items = append(resp.Items, contentItemToData(item))
						break
					}
				}
			}
		}
	}

	return resp, nil
}

// ParseAssetPathHandler parses a single asset path and extracts properties.
type ParseAssetPathHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *ParseAssetPathHandler) ErrorCode() string { return "ASSET_PARSE_001" }

// Handle processes the request.
func (h *ParseAssetPathHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ParseAssetPathRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Create conventions
	conventions := assets.NewManagedCodeConventions()

	// Try to match against all pattern sets
	var item *assets.ContentItem

	// Try each pattern set in order
	patternSets := []*assets.PatternSet{
		conventions.RuntimeAssemblies,
		conventions.CompileRefAssemblies,
		conventions.CompileLibAssemblies,
		conventions.NativeLibraries,
		conventions.ResourceAssemblies,
		conventions.MSBuildFiles,
		conventions.MSBuildMultiTargeting,
		conventions.ContentFiles,
		conventions.ToolsAssemblies,
	}

	for _, ps := range patternSets {
		if ps == nil {
			continue
		}
		for _, expr := range ps.PathExpressions {
			if expr == nil {
				continue
			}
			if matched := expr.Match(req.Path, conventions.Properties); matched != nil {
				item = matched
				break
			}
		}
		if item != nil {
			break
		}
	}

	// Build response
	resp := ParseAssetPathResponse{}
	if item != nil {
		itemData := contentItemToData(item)
		resp.Item = &itemData
	}

	return resp, nil
}

// contentItemToData converts a ContentItem to ContentItemData for JSON serialization.
func contentItemToData(item *assets.ContentItem) ContentItemData {
	data := ContentItemData{
		Path:       item.Path,
		Properties: make(map[string]interface{}),
	}

	// Convert properties to JSON-friendly format
	for key, value := range item.Properties {
		switch v := value.(type) {
		case *frameworks.NuGetFramework:
			// Serialize framework as TFM string (e.g., "net6.0")
			data.Properties[key] = v.String()
		case string:
			data.Properties[key] = v
		default:
			// For other types, include as-is (they'll be JSON serialized)
			data.Properties[key] = v
		}
	}

	return data
}

// ExpandRuntimeHandler expands a runtime identifier to all compatible RIDs.
type ExpandRuntimeHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *ExpandRuntimeHandler) ErrorCode() string { return "RID_EXP_001" }

// Handle processes the request.
func (h *ExpandRuntimeHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ExpandRuntimeRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.RID == "" {
		return nil, fmt.Errorf("rid is required")
	}

	// Load default runtime graph
	graph := assets.LoadDefaultRuntimeGraph()

	// Expand the RID
	expandedRIDs := graph.ExpandRuntime(req.RID)

	return ExpandRuntimeResponse{
		ExpandedRuntimes: expandedRIDs,
	}, nil
}

// AreRuntimesCompatibleHandler checks if two runtime identifiers are compatible.
type AreRuntimesCompatibleHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *AreRuntimesCompatibleHandler) ErrorCode() string { return "RID_COMPAT_001" }

// Handle processes the request.
func (h *AreRuntimesCompatibleHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req AreRuntimesCompatibleRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.TargetRID == "" {
		return nil, fmt.Errorf("targetRid is required")
	}
	if req.PackageRID == "" {
		return nil, fmt.Errorf("packageRid is required")
	}

	// Load default runtime graph
	graph := assets.LoadDefaultRuntimeGraph()

	// Check compatibility
	compatible := graph.AreCompatible(req.TargetRID, req.PackageRID)

	return AreRuntimesCompatibleResponse{
		Compatible: compatible,
	}, nil
}
