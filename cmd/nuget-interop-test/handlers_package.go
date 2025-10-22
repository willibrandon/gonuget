package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/willibrandon/gonuget/packaging"
	"github.com/willibrandon/gonuget/version"
)

// ReadPackageHandler reads package metadata from a .nupkg file.
type ReadPackageHandler struct{}

func (h *ReadPackageHandler) ErrorCode() string { return "PKG_READ_001" }

func (h *ReadPackageHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ReadPackageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	if len(req.PackageBytes) == 0 {
		return nil, fmt.Errorf("packageBytes is required")
	}

	// Use the real packaging library
	pkgReader, err := packaging.OpenPackageFromReaderAt(bytes.NewReader(req.PackageBytes), int64(len(req.PackageBytes)))
	if err != nil {
		return nil, fmt.Errorf("open package: %w", err)
	}
	defer pkgReader.Close()

	// Get nuspec using library
	nuspec, err := pkgReader.GetNuspec()
	if err != nil {
		return nil, fmt.Errorf("get nuspec: %w", err)
	}

	// Build response
	resp := ReadPackageResponse{
		ID:           nuspec.Metadata.ID,
		Version:      nuspec.Metadata.Version,
		Description:  nuspec.Metadata.Description,
		Authors:      []string{}, // Initialize to empty array, not nil
		FileCount:    len(pkgReader.Files()),
		HasSignature: pkgReader.IsSigned(),
	}

	// Add authors if present
	if nuspec.Metadata.Authors != "" {
		resp.Authors = strings.Split(nuspec.Metadata.Authors, ",")
		for i := range resp.Authors {
			resp.Authors[i] = strings.TrimSpace(resp.Authors[i])
		}
	}

	// Add dependencies
	if nuspec.Metadata.Dependencies != nil && len(nuspec.Metadata.Dependencies.Groups) > 0 {
		for _, group := range nuspec.Metadata.Dependencies.Groups {
			for _, dep := range group.Dependencies {
				depStr := fmt.Sprintf("%s:%s", dep.ID, dep.Version)
				resp.Dependencies = append(resp.Dependencies, depStr)
			}
		}
	}

	if resp.HasSignature {
		resp.SignatureType = "Unknown"
	}

	return resp, nil
}

// BuildPackageHandler creates a minimal .nupkg package.
type BuildPackageHandler struct{}

func (h *BuildPackageHandler) ErrorCode() string { return "PKG_BUILD_001" }

func (h *BuildPackageHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req BuildPackageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	if req.ID == "" {
		return nil, fmt.Errorf("id is required")
	}
	if req.Version == "" {
		return nil, fmt.Errorf("version is required")
	}

	// Parse version string
	ver, err := version.Parse(req.Version)
	if err != nil {
		return nil, fmt.Errorf("parse version: %w", err)
	}

	// Use the real packaging library
	builder := packaging.NewPackageBuilder()
	builder.SetID(req.ID)
	builder.SetVersion(ver)

	if req.Description != "" {
		builder.SetDescription(req.Description)
	}

	if len(req.Authors) > 0 {
		builder.SetAuthors(req.Authors...)
	}

	// Add files
	for path, content := range req.Files {
		if err := builder.AddFileFromBytes(path, content); err != nil {
			return nil, fmt.Errorf("add file %s: %w", path, err)
		}
	}

	// Build package to bytes
	var buf bytes.Buffer
	if err := builder.Save(&buf); err != nil {
		return nil, fmt.Errorf("save package: %w", err)
	}

	return BuildPackageResponse{
		PackageBytes: buf.Bytes(),
	}, nil
}
