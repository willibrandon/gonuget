package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/willibrandon/gonuget/packaging"
)

// ReadPackageHandler reads package metadata from a .nupkg file.
type ReadPackageHandler struct{}

func (h *ReadPackageHandler) ErrorCode() string { return "PKG_READ_001" }

func (h *ReadPackageHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ReadPackageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if len(req.PackageBytes) == 0 {
		return nil, fmt.Errorf("packageBytes is required")
	}

	// Create a reader from the package bytes
	reader, err := zip.NewReader(bytes.NewReader(req.PackageBytes), int64(len(req.PackageBytes)))
	if err != nil {
		return nil, fmt.Errorf("read ZIP archive: %w", err)
	}

	// Find and read the .nuspec file
	var nuspec *packaging.Nuspec
	var hasSignature bool
	fileCount := len(reader.File)

	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ".nuspec") {
			// Read nuspec file
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("open nuspec: %w", err)
			}
			defer rc.Close()

			nuspecBytes, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("read nuspec: %w", err)
			}

			nuspec = &packaging.Nuspec{}
			if err := xml.Unmarshal(nuspecBytes, nuspec); err != nil {
				return nil, fmt.Errorf("parse nuspec: %w", err)
			}
		}

		// Check for signature file
		if file.Name == ".signature.p7s" {
			hasSignature = true
		}
	}

	if nuspec == nil {
		return nil, fmt.Errorf("nuspec file not found in package")
	}

	// Build response
	resp := ReadPackageResponse{
		ID:           nuspec.Metadata.ID,
		Version:      nuspec.Metadata.Version,
		Description:  nuspec.Metadata.Description,
		FileCount:    fileCount,
		HasSignature: hasSignature,
	}

	// Add authors if present
	if nuspec.Metadata.Authors != "" {
		resp.Authors = strings.Split(nuspec.Metadata.Authors, ",")
		// Trim whitespace from each author
		for i := range resp.Authors {
			resp.Authors[i] = strings.TrimSpace(resp.Authors[i])
		}
	}

	// Add dependencies (simplified - just count for now)
	if nuspec.Metadata.Dependencies != nil && len(nuspec.Metadata.Dependencies.Groups) > 0 {
		for _, group := range nuspec.Metadata.Dependencies.Groups {
			for _, dep := range group.Dependencies {
				depStr := fmt.Sprintf("%s:%s", dep.ID, dep.Version)
				resp.Dependencies = append(resp.Dependencies, depStr)
			}
		}
	}

	// If we detected a signature, try to determine its type
	if hasSignature {
		// For now, we can't easily determine the type without parsing it
		// Set as "Unknown" - tests can improve this later
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

	// Validate required fields
	if req.ID == "" {
		return nil, fmt.Errorf("id is required")
	}
	if req.Version == "" {
		return nil, fmt.Errorf("version is required")
	}

	// Create nuspec content
	nuspec := &packaging.Nuspec{
		Metadata: packaging.NuspecMetadata{
			ID:          req.ID,
			Version:     req.Version,
			Description: req.Description,
		},
	}

	if len(req.Authors) > 0 {
		nuspec.Metadata.Authors = strings.Join(req.Authors, ", ")
	} else {
		nuspec.Metadata.Authors = "Unknown"
	}

	// Marshal nuspec to XML
	nuspecBytes, err := xml.MarshalIndent(nuspec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal nuspec: %w", err)
	}

	// Add XML header
	nuspecContent := []byte(xml.Header + string(nuspecBytes))

	// Create ZIP archive in memory
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add nuspec file
	nuspecName := req.ID + ".nuspec"
	writer, err := zipWriter.Create(nuspecName)
	if err != nil {
		return nil, fmt.Errorf("create nuspec entry: %w", err)
	}
	if _, err := writer.Write(nuspecContent); err != nil {
		return nil, fmt.Errorf("write nuspec: %w", err)
	}

	// Add additional files
	for path, content := range req.Files {
		writer, err := zipWriter.Create(path)
		if err != nil {
			return nil, fmt.Errorf("create file entry %s: %w", path, err)
		}
		if _, err := writer.Write(content); err != nil {
			return nil, fmt.Errorf("write file %s: %w", path, err)
		}
	}

	// Close ZIP writer
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close ZIP: %w", err)
	}

	return BuildPackageResponse{
		PackageBytes: buf.Bytes(),
	}, nil
}
