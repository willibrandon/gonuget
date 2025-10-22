package main

import (
	"encoding/json"
	"fmt"

	"github.com/willibrandon/gonuget/frameworks"
)

// CheckFrameworkCompatHandler checks if a package framework is compatible with a project framework.
type CheckFrameworkCompatHandler struct{}

func (h *CheckFrameworkCompatHandler) ErrorCode() string { return "FW_COMPAT_001" }

func (h *CheckFrameworkCompatHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req CheckFrameworkCompatRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.PackageFramework == "" {
		return nil, fmt.Errorf("packageFramework is required")
	}
	if req.ProjectFramework == "" {
		return nil, fmt.Errorf("projectFramework is required")
	}

	// Parse package framework
	pkgFw, err := frameworks.ParseFramework(req.PackageFramework)
	if err != nil {
		return nil, fmt.Errorf("parse packageFramework '%s': %w", req.PackageFramework, err)
	}

	// Parse project framework
	projFw, err := frameworks.ParseFramework(req.ProjectFramework)
	if err != nil {
		return nil, fmt.Errorf("parse projectFramework '%s': %w", req.ProjectFramework, err)
	}

	// Check compatibility
	compatible := pkgFw.IsCompatible(projFw)

	resp := CheckFrameworkCompatResponse{
		Compatible: compatible,
	}

	// Add reason if not compatible
	if !compatible {
		resp.Reason = fmt.Sprintf("Package framework %s is not compatible with project framework %s",
			req.PackageFramework, req.ProjectFramework)
	}

	return resp, nil
}

// ParseFrameworkHandler parses a framework identifier into components.
type ParseFrameworkHandler struct{}

func (h *ParseFrameworkHandler) ErrorCode() string { return "FW_PARSE_001" }

func (h *ParseFrameworkHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ParseFrameworkRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if req.Framework == "" {
		return nil, fmt.Errorf("framework is required")
	}

	// Parse framework
	fw, err := frameworks.ParseFramework(req.Framework)
	if err != nil {
		return nil, fmt.Errorf("parse framework '%s': %w", req.Framework, err)
	}

	// Build response with framework components
	resp := ParseFrameworkResponse{
		Identifier: fw.Framework,
		Profile:    fw.Profile,
		Platform:   fw.Platform,
	}

	// Format version as string using the FrameworkVersion.String() method
	resp.Version = fw.Version.String()

	return resp, nil
}
