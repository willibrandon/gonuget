package packaging

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractPackageV2 extracts package using V2 (packages.config) directory layout.
// Reference: PackageExtractor.ExtractPackageAsync (Stream-based) in NuGet.Packaging
func ExtractPackageV2(
	ctx context.Context,
	source string,
	packageStream io.ReadSeeker,
	pathResolver *PackagePathResolver,
	extractionContext *PackageExtractionContext,
) ([]string, error) {
	// Check for cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Validate stream is seekable (NuGet.Client requirement)
	if _, err := packageStream.Seek(0, io.SeekCurrent); err != nil {
		return nil, fmt.Errorf("package stream must be seekable: %w", err)
	}

	// Seek to start
	if _, err := packageStream.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to start: %w", err)
	}

	// Get file size
	endPos, err := packageStream.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("get file size: %w", err)
	}
	if _, err := packageStream.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to start: %w", err)
	}

	// Open package reader
	reader, err := OpenPackageFromReaderAt(packageStream.(io.ReaderAt), endPos)
	if err != nil {
		return nil, fmt.Errorf("open package reader: %w", err)
	}
	defer func() { _ = reader.Close() }()

	// Get package identity
	identity, err := reader.GetIdentity()
	if err != nil {
		return nil, fmt.Errorf("get package identity: %w", err)
	}

	// Create installation directory
	installPath := pathResolver.GetInstallPath(identity)
	if err := os.MkdirAll(installPath, 0755); err != nil {
		return nil, fmt.Errorf("create install directory: %w", err)
	}

	// Verify package signature (if configured)
	if extractionContext.SignatureVerifier != nil {
		if err := extractionContext.SignatureVerifier.VerifySignatureAsync(ctx, reader); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	var extractedFiles []string

	// Extract .nuspec if requested
	if extractionContext.PackageSaveMode.HasFlag(PackageSaveModeNuspec) {
		nuspecPath := filepath.Join(installPath, pathResolver.GetManifestFileName(identity))

		// Find nuspec file in package
		nuspecFile, err := reader.GetNuspecFile()
		if err != nil {
			return nil, fmt.Errorf("get nuspec file: %w", err)
		}

		// Extract nuspec
		stream, err := nuspecFile.Open()
		if err != nil {
			return nil, fmt.Errorf("open nuspec stream: %w", err)
		}
		defer func() { _ = stream.Close() }()

		if _, err := CopyToFile(stream, nuspecPath); err != nil {
			return nil, fmt.Errorf("extract nuspec: %w", err)
		}

		extractedFiles = append(extractedFiles, nuspecPath)

		if extractionContext.Logger != nil {
			extractionContext.Logger.Info("Extracted %s", nuspecPath)
		}
	}

	// Extract package files if requested
	if extractionContext.PackageSaveMode.HasFlag(PackageSaveModeFiles) {
		// Get all package files
		packageFiles := reader.GetPackageFiles()

		// Build file list for XML doc extractor
		var fileNames []string
		for _, f := range packageFiles {
			fileNames = append(fileNames, f.Name)
		}

		// Create file extractor for XML doc handling
		fileExtractor := NewPackageFileExtractor(fileNames, extractionContext.XMLDocFileSaveMode)

		// Extract each file
		for _, file := range packageFiles {
			// Check for cancellation
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			// Skip OPC metadata files
			if shouldExcludeFile(file.Name) {
				continue
			}

			targetPath := filepath.Join(installPath, filepath.FromSlash(file.Name))

			// Open file stream
			stream, err := file.Open()
			if err != nil {
				if extractionContext.Logger != nil {
					extractionContext.Logger.Warning("Failed to open %s: %v", file.Name, err)
				}
				continue
			}

			// Extract file (handles XML doc compression)
			extractedPath, err := fileExtractor.ExtractPackageFile(file.Name, targetPath, stream)
			_ = stream.Close()

			if err != nil {
				if extractionContext.Logger != nil {
					extractionContext.Logger.Warning("Failed to extract %s: %v", file.Name, err)
				}
				continue
			}

			if extractedPath != "" {
				extractedFiles = append(extractedFiles, extractedPath)
			}
		}

		if extractionContext.Logger != nil {
			extractionContext.Logger.Info("Extracted %d files", len(extractedFiles))
		}
	}

	// Save .nupkg if requested (MUST BE LAST - atomic install marker)
	if extractionContext.PackageSaveMode.HasFlag(PackageSaveModeNupkg) {
		nupkgPath := pathResolver.GetPackageFilePath(identity)

		// Seek to start of stream
		if _, err := packageStream.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek package stream: %w", err)
		}

		// Copy nupkg
		nupkgFile, err := CreateFile(nupkgPath)
		if err != nil {
			return nil, fmt.Errorf("create nupkg file: %w", err)
		}

		if _, err := io.Copy(nupkgFile, packageStream); err != nil {
			_ = nupkgFile.Close()
			return nil, fmt.Errorf("copy nupkg: %w", err)
		}
		_ = nupkgFile.Close()

		extractedFiles = append(extractedFiles, nupkgPath)
	}

	// Copy satellite files if requested
	if extractionContext.CopySatelliteFiles {
		if err := copySatelliteFilesV2(reader, pathResolver, extractionContext); err != nil {
			if extractionContext.Logger != nil {
				extractionContext.Logger.Warning("Failed to copy satellite files: %v", err)
			}
		}
	}

	return extractedFiles, nil
}

// shouldExcludeFile checks if file should be excluded from extraction.
// Reference: PackageHelper.IsPackageFile and ShouldInclude
func shouldExcludeFile(path string) bool {
	lowerPath := strings.ToLower(path)

	// Exclude OPC metadata
	if strings.HasPrefix(lowerPath, "_rels/") || strings.HasPrefix(lowerPath, "package/") {
		return true
	}
	if lowerPath == "[content_types].xml" {
		return true
	}
	if strings.HasSuffix(lowerPath, ".psmdcp") {
		return true
	}

	// Exclude root-level .nupkg and .nuspec (extracted with proper names)
	parts := strings.Split(path, "/")
	if len(parts) == 1 {
		if strings.HasSuffix(lowerPath, ".nupkg") || strings.HasSuffix(lowerPath, ".nuspec") {
			return true
		}
	}

	return false
}

// copySatelliteFilesV2 copies satellite files to runtime package if applicable (V2 layout).
func copySatelliteFilesV2(reader *PackageReader, pathResolver *PackagePathResolver, ctx *PackageExtractionContext) error {
	// Get package identity
	identity, err := reader.GetIdentity()
	if err != nil {
		return fmt.Errorf("get identity: %w", err)
	}

	// Call the satellite files copy function
	_, err = CopySatelliteFilesIfApplicableV2(reader, identity, pathResolver, ctx.PackageSaveMode, ctx.Logger)
	return err
}
