package main

import (
	"encoding/json"
	"fmt"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/packaging/assets"
)

// FindRuntimeAssembliesHandler finds runtime assemblies from package paths.
type FindRuntimeAssembliesHandler struct{}

func (h *FindRuntimeAssembliesHandler) ErrorCode() string { return "ASSET_RT_001" }

func (h *FindRuntimeAssembliesHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req FindAssembliesRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if len(req.Paths) == 0 {
		return nil, fmt.Errorf("paths is required")
	}

	// Create conventions
	conventions := assets.NewManagedCodeConventions()

	// Match paths against runtime assembly patterns
	var items []ContentItemData
	for _, path := range req.Paths {
		for _, expr := range conventions.RuntimeAssemblies.PathExpressions {
			if item := expr.Match(path, conventions.Properties); item != nil {
				// Filter by target framework if specified
				if req.TargetFramework != "" {
					if !matchesTargetFramework(item, req.TargetFramework) {
						continue
					}
				}

				items = append(items, contentItemToData(item))
				break // Only match first pattern
			}
		}
	}

	return FindAssembliesResponse{Items: items}, nil
}

// FindCompileAssembliesHandler finds compile reference assemblies from package paths.
type FindCompileAssembliesHandler struct{}

func (h *FindCompileAssembliesHandler) ErrorCode() string { return "ASSET_CP_001" }

func (h *FindCompileAssembliesHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req FindAssembliesRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if len(req.Paths) == 0 {
		return nil, fmt.Errorf("paths is required")
	}

	// Create conventions
	conventions := assets.NewManagedCodeConventions()

	// Match paths against compile assembly patterns (ref/ + lib/)
	var items []ContentItemData
	for _, path := range req.Paths {
		// Try ref/ patterns first
		matched := false
		for _, expr := range conventions.CompileRefAssemblies.PathExpressions {
			if item := expr.Match(path, conventions.Properties); item != nil {
				// Filter by target framework if specified
				if req.TargetFramework != "" {
					if !matchesTargetFramework(item, req.TargetFramework) {
						continue
					}
				}

				items = append(items, contentItemToData(item))
				matched = true
				break
			}
		}

		// If not matched, try lib/ patterns
		if !matched {
			for _, expr := range conventions.CompileLibAssemblies.PathExpressions {
				if item := expr.Match(path, conventions.Properties); item != nil {
					// Filter by target framework if specified
					if req.TargetFramework != "" {
						if !matchesTargetFramework(item, req.TargetFramework) {
							continue
						}
					}

					items = append(items, contentItemToData(item))
					break
				}
			}
		}
	}

	return FindAssembliesResponse{Items: items}, nil
}

// ParseAssetPathHandler parses a single asset path and extracts properties.
type ParseAssetPathHandler struct{}

func (h *ParseAssetPathHandler) ErrorCode() string { return "ASSET_PARSE_001" }

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

// Helper functions

// matchesTargetFramework checks if a content item's framework matches the target.
func matchesTargetFramework(item *assets.ContentItem, targetFramework string) bool {
	// Get the item's framework
	tfmValue, hasTfm := item.Properties["tfm"]
	if !hasTfm {
		return false
	}

	itemFw, ok := tfmValue.(*frameworks.NuGetFramework)
	if !ok {
		return false
	}

	// Parse target framework
	targetFw, err := frameworks.ParseFramework(targetFramework)
	if err != nil {
		return false
	}

	// Check compatibility
	return frameworks.IsCompatible(itemFw, targetFw)
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
