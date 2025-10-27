package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/willibrandon/gonuget/packaging"
	"github.com/willibrandon/gonuget/version"
)

// ReadPackageHandler reads package metadata from a .nupkg file.
type ReadPackageHandler struct{}

// ErrorCode returns the error code for this handler.
func (h *ReadPackageHandler) ErrorCode() string { return "PKG_READ_001" }

// Handle processes the request.
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

// ErrorCode returns the error code for this handler.
func (h *BuildPackageHandler) ErrorCode() string { return "PKG_BUILD_001" }

// Handle processes the request.
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

// ExtractPackageV2Handler extracts a package using V2 (packages.config) layout.
type ExtractPackageV2Handler struct{}

// ErrorCode returns the error code for this handler.
func (h *ExtractPackageV2Handler) ErrorCode() string { return "PKG_EXTRACT_V2_001" }

// Handle processes the request.
func (h *ExtractPackageV2Handler) Handle(data json.RawMessage) (interface{}, error) {
	var req ExtractPackageV2Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	if len(req.PackageBytes) == 0 {
		return nil, fmt.Errorf("packageBytes is required")
	}
	if req.InstallPath == "" {
		return nil, fmt.Errorf("installPath is required")
	}

	// Open package from bytes
	packageReader := bytes.NewReader(req.PackageBytes)

	// Create path resolver
	resolver := packaging.NewPackagePathResolver(req.InstallPath, req.UseSideBySideLayout)

	// Create extraction context
	ctx := &packaging.PackageExtractionContext{
		PackageSaveMode:    packaging.PackageSaveMode(req.PackageSaveMode),
		XMLDocFileSaveMode: packaging.XMLDocFileSaveMode(req.XMLDocFileSaveMode),
		Logger:             nil, // No logging for tests
	}

	// Extract package
	extractedFiles, err := packaging.ExtractPackageV2(
		context.Background(),
		"source", // Source string (not used in V2)
		packageReader,
		resolver,
		ctx,
	)
	if err != nil {
		return nil, fmt.Errorf("extract package: %w", err)
	}

	return ExtractPackageV2Response{
		ExtractedFiles: extractedFiles,
		FileCount:      len(extractedFiles),
	}, nil
}

// InstallFromSourceV3Handler installs a package using V3 (PackageReference) layout.
type InstallFromSourceV3Handler struct{}

// ErrorCode returns the error code for this handler.
func (h *InstallFromSourceV3Handler) ErrorCode() string { return "PKG_INSTALL_V3_001" }

// Handle processes the request.
func (h *InstallFromSourceV3Handler) Handle(data json.RawMessage) (interface{}, error) {
	var req InstallFromSourceV3Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	if len(req.PackageBytes) == 0 {
		return nil, fmt.Errorf("packageBytes is required")
	}
	if req.ID == "" {
		return nil, fmt.Errorf("id is required")
	}
	if req.Version == "" {
		return nil, fmt.Errorf("version is required")
	}
	if req.GlobalPackagesFolder == "" {
		return nil, fmt.Errorf("globalPackagesFolder is required")
	}

	// Parse version
	ver, err := version.Parse(req.Version)
	if err != nil {
		return nil, fmt.Errorf("parse version: %w", err)
	}

	// Create package identity
	identity := &packaging.PackageIdentity{
		ID:      req.ID,
		Version: ver,
	}

	// Create version folder path resolver
	resolver := packaging.NewVersionFolderPathResolver(req.GlobalPackagesFolder, true)

	// Create extraction context
	ctx := &packaging.PackageExtractionContext{
		PackageSaveMode:    packaging.PackageSaveMode(req.PackageSaveMode),
		XMLDocFileSaveMode: packaging.XMLDocFileSaveMode(req.XMLDocFileSaveMode),
		Logger:             nil, // No logging for tests
	}

	// Create copyToAsync function that writes packageBytes to target
	copyToAsync := func(targetPath string) error {
		return os.WriteFile(targetPath, req.PackageBytes, 0644)
	}

	// Install package
	installed, err := packaging.InstallFromSourceV3(
		context.Background(),
		"source", // Source string
		identity,
		copyToAsync,
		resolver,
		ctx,
	)
	if err != nil {
		return nil, fmt.Errorf("install package: %w", err)
	}

	// Build response
	resp := InstallFromSourceV3Response{
		Installed:        installed,
		PackageDirectory: resolver.GetPackageDirectory(req.ID, ver),
		NuspecPath:       resolver.GetManifestFilePath(req.ID, ver),
		HashPath:         resolver.GetHashPath(req.ID, ver),
		MetadataPath:     resolver.GetNupkgMetadataPath(req.ID, ver),
	}

	// Include nupkg path if saved
	if req.PackageSaveMode&int(packaging.PackageSaveModeNupkg) != 0 {
		resp.NupkgPath = resolver.GetPackageFilePath(req.ID, ver)
	}

	return resp, nil
}
