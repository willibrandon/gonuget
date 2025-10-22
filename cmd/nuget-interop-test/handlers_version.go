package main

import (
	"encoding/json"
	"fmt"

	"github.com/willibrandon/gonuget/version"
)

// CompareVersionsHandler compares two NuGet version strings.
type CompareVersionsHandler struct{}

func (h *CompareVersionsHandler) ErrorCode() string { return "VER_CMP_001" }

func (h *CompareVersionsHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req CompareVersionsRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.Version1 == "" {
		return nil, fmt.Errorf("version1 is required")
	}
	if req.Version2 == "" {
		return nil, fmt.Errorf("version2 is required")
	}

	// Parse version 1
	v1, err := version.Parse(req.Version1)
	if err != nil {
		return nil, fmt.Errorf("parse version1 '%s': %w", req.Version1, err)
	}

	// Parse version 2
	v2, err := version.Parse(req.Version2)
	if err != nil {
		return nil, fmt.Errorf("parse version2 '%s': %w", req.Version2, err)
	}

	// Compare versions
	result := v1.Compare(v2)

	return CompareVersionsResponse{
		Result: result,
	}, nil
}

// ParseVersionHandler parses a NuGet version string into its components.
type ParseVersionHandler struct{}

func (h *ParseVersionHandler) ErrorCode() string { return "VER_PARSE_001" }

func (h *ParseVersionHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ParseVersionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.Version == "" {
		return nil, fmt.Errorf("version is required")
	}

	// Parse version
	v, err := version.Parse(req.Version)
	if err != nil {
		return nil, fmt.Errorf("parse version '%s': %w", req.Version, err)
	}

	// Extract components from parsed version
	resp := ParseVersionResponse{
		Major:        v.Major,
		Minor:        v.Minor,
		Patch:        v.Patch,
		Revision:     v.Revision,
		IsPrerelease: v.IsPrerelease(),
		IsLegacy:     v.IsLegacyVersion,
		Metadata:     v.Metadata,
	}

	// Extract pre-release label if present
	// Join ReleaseLabels with '.' to match NuGet.Client format
	if len(v.ReleaseLabels) > 0 {
		release := ""
		for i, label := range v.ReleaseLabels {
			if i > 0 {
				release += "."
			}
			release += label
		}
		resp.Release = release
	}

	return resp, nil
}
