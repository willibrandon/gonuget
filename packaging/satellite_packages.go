package packaging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/willibrandon/gonuget/version"
)

// IsSatellitePackage checks if a package is a satellite package based on NuGet rules:
// 1. Has a <language> element in .nuspec metadata
// 2. Package ID ends with ".<language>" suffix (e.g., "MyLib.ja-jp")
// 3. Has exactly one dependency with exact version match to the runtime package
//
// Reference: NuGet.Client PackageExtractor.cs IsSatellitePackage
func IsSatellitePackage(reader *PackageReader, identity *PackageIdentity) (bool, *PackageIdentity, error) {
	nuspec, err := reader.GetNuspec()
	if err != nil {
		return false, nil, fmt.Errorf("get nuspec: %w", err)
	}

	// Check for language element
	if nuspec.Metadata.Language == "" {
		return false, nil, nil
	}

	// Check dependencies
	deps := nuspec.Metadata.Dependencies
	if len(deps.Groups) != 1 {
		return false, nil, nil // Must have exactly one dependency group
	}

	packages := deps.Groups[0].Dependencies
	if len(packages) != 1 {
		return false, nil, nil // Must have exactly one dependency
	}

	runtimePkg := packages[0]

	// Parse version range
	versionRange, err := version.ParseVersionRange(runtimePkg.Version)
	if err != nil {
		return false, nil, nil
	}

	// Verify exact version match (satellite must match runtime package version)
	// An exact version has both min and max versions equal and both are inclusive
	if versionRange.MinVersion == nil || versionRange.MaxVersion == nil {
		return false, nil, nil
	}
	if !versionRange.MinInclusive || !versionRange.MaxInclusive {
		return false, nil, nil
	}
	if !versionRange.MinVersion.Equals(versionRange.MaxVersion) {
		return false, nil, nil
	}

	exactVersion := versionRange.MinVersion
	if !exactVersion.Equals(identity.Version) {
		return false, nil, nil
	}

	// Verify ID suffix matches language (e.g., "MyLib" + ".ja-jp")
	language := nuspec.Metadata.Language
	expectedID := runtimePkg.ID + "." + language
	if !strings.EqualFold(identity.ID, expectedID) {
		return false, nil, nil
	}

	// This is a valid satellite package
	runtimeIdentity := &PackageIdentity{
		ID:      runtimePkg.ID,
		Version: exactVersion,
	}

	return true, runtimeIdentity, nil
}

// extractSatelliteFilesV2 extracts satellite package files to runtime package directory (V2 layout).
// Reference: NuGet.Client PackageExtractor.CopySatelliteFilesAsync
func extractSatelliteFilesV2(
	packageReader *PackageReader,
	runtimeIdentity *PackageIdentity,
	pathResolver *PackagePathResolver,
	saveMode PackageSaveMode,
	logger Logger,
) error {
	// Get runtime package directory
	runtimePackageDir := pathResolver.GetInstallPath(runtimeIdentity)

	// Check if runtime package exists
	if _, err := os.Stat(runtimePackageDir); os.IsNotExist(err) {
		return fmt.Errorf("runtime package not installed: %s %s", runtimeIdentity.ID, runtimeIdentity.Version)
	}

	// Extract satellite package files to runtime package directory
	// IMPORTANT: OPC metadata files are excluded (same as regular extraction)
	files := packageReader.GetPackageFiles()

	// Build file list for XML doc extractor
	var fileNames []string
	for _, f := range files {
		fileNames = append(fileNames, f.Name)
	}

	extractor := NewPackageFileExtractor(fileNames, XMLDocFileSaveModeNone) // Satellites don't use XML doc compression

	for _, file := range files {
		// Filter files based on PackageSaveMode (same as NuGet.Client)
		// Reference: NuGet.Client PackageExtractor.CopySatelliteFilesAsync uses IsPackageFile
		if !isPackageFile(file.Name, saveMode) {
			continue
		}

		// Skip .nuspec and .nupkg in root (satellite already has its own)
		if isRootMetadata(file.Name) {
			continue
		}

		// Determine target path in runtime package directory
		targetPath := filepath.Join(runtimePackageDir, filepath.FromSlash(file.Name))

		// Ensure target directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("create directory for %s: %w", file.Name, err)
		}

		// Extract file from satellite package
		stream, err := file.Open()
		if err != nil {
			return fmt.Errorf("open stream for %s: %w", file.Name, err)
		}

		// Copy to runtime package directory
		if _, err := extractor.ExtractPackageFile(file.Name, targetPath, stream); err != nil {
			_ = stream.Close()
			return fmt.Errorf("extract %s: %w", file.Name, err)
		}
		_ = stream.Close()

		if logger != nil {
			logger.Info("Satellite file copied: %s", file.Name)
		}
	}

	return nil
}

// extractSatelliteFilesV3 extracts satellite package files to runtime package directory (V3 layout).
// Reference: NuGet.Client PackageExtractor.CopySatelliteFilesAsync
func extractSatelliteFilesV3(
	packageReader *PackageReader,
	runtimeIdentity *PackageIdentity,
	versionResolver *VersionFolderPathResolver,
	saveMode PackageSaveMode,
	logger Logger,
) error {
	// Get runtime package directory
	runtimePackageDir := versionResolver.GetInstallPath(runtimeIdentity.ID, runtimeIdentity.Version)

	// Check if runtime package exists
	if _, err := os.Stat(runtimePackageDir); os.IsNotExist(err) {
		return fmt.Errorf("runtime package not installed: %s %s", runtimeIdentity.ID, runtimeIdentity.Version)
	}

	// Extract satellite package files to runtime package directory
	files := packageReader.GetPackageFiles()

	// Build file list for XML doc extractor
	var fileNames []string
	for _, f := range files {
		fileNames = append(fileNames, f.Name)
	}

	extractor := NewPackageFileExtractor(fileNames, XMLDocFileSaveModeNone)

	for _, file := range files {
		// Filter files based on PackageSaveMode (same as NuGet.Client)
		// Reference: NuGet.Client PackageExtractor.CopySatelliteFilesAsync uses IsPackageFile
		if !isPackageFile(file.Name, saveMode) {
			continue
		}

		// Skip .nuspec and .nupkg in root
		if isRootMetadata(file.Name) {
			continue
		}

		// Determine target path in runtime package directory
		targetPath := filepath.Join(runtimePackageDir, filepath.FromSlash(file.Name))

		// Ensure target directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("create directory for %s: %w", file.Name, err)
		}

		// Extract file from satellite package
		stream, err := file.Open()
		if err != nil {
			return fmt.Errorf("open stream for %s: %w", file.Name, err)
		}

		// Copy to runtime package directory
		if _, err := extractor.ExtractPackageFile(file.Name, targetPath, stream); err != nil {
			_ = stream.Close()
			return fmt.Errorf("extract %s: %w", file.Name, err)
		}
		_ = stream.Close()

		if logger != nil {
			logger.Info("Satellite file copied: %s", file.Name)
		}
	}

	return nil
}

func isRootMetadata(path string) bool {
	// Check if file is in root and is .nuspec or .nupkg
	if strings.Contains(path, "/") || strings.Contains(path, "\\") {
		return false // Not in root
	}

	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".nuspec") || strings.HasSuffix(lower, ".nupkg")
}

// isPackageFile determines if a file should be extracted based on PackageSaveMode.
// Reference: NuGet.Client PackageHelper.IsPackageFile
func isPackageFile(fileName string, saveMode PackageSaveMode) bool {
	if fileName == "" || filepath.Base(fileName) == "" {
		// Ignore directory entries
		return false
	}

	// Check if file is .nuspec
	if strings.HasSuffix(strings.ToLower(fileName), ".nuspec") {
		return (saveMode & PackageSaveModeNuspec) == PackageSaveModeNuspec
	}

	// For other files, check if Files mode is enabled
	if (saveMode & PackageSaveModeFiles) == PackageSaveModeFiles {
		// Exclude files that should always be skipped
		if shouldExcludeFile(fileName) {
			return false
		}
		// Exclude NuGet-generated files (.nupkg.sha512, .nupkg.metadata)
		lower := strings.ToLower(fileName)
		if strings.HasSuffix(lower, ".nupkg.sha512") || strings.HasSuffix(lower, ".nupkg.metadata") {
			return false
		}
		return true
	}

	return false
}

// CopySatelliteFilesIfApplicableV2 checks if the package is a satellite package and copies files to runtime package (V2 layout).
// Returns true if satellite files were copied, false otherwise.
func CopySatelliteFilesIfApplicableV2(
	packageReader *PackageReader,
	identity *PackageIdentity,
	pathResolver *PackagePathResolver,
	saveMode PackageSaveMode,
	logger Logger,
) (bool, error) {
	isSatellite, runtimeIdentity, err := IsSatellitePackage(packageReader, identity)
	if err != nil {
		return false, fmt.Errorf("check satellite package: %w", err)
	}

	if !isSatellite {
		return false, nil // Not a satellite package
	}

	// Extract and merge satellite files
	if err := extractSatelliteFilesV2(packageReader, runtimeIdentity, pathResolver, saveMode, logger); err != nil {
		return false, fmt.Errorf("extract satellite files: %w", err)
	}

	if logger != nil {
		logger.Info("Satellite package %s merged into %s", identity, runtimeIdentity)
	}

	return true, nil
}

// CopySatelliteFilesIfApplicableV3 checks if the package is a satellite package and copies files to runtime package (V3 layout).
func CopySatelliteFilesIfApplicableV3(
	packageReader *PackageReader,
	identity *PackageIdentity,
	versionResolver *VersionFolderPathResolver,
	saveMode PackageSaveMode,
	logger Logger,
) (bool, error) {
	isSatellite, runtimeIdentity, err := IsSatellitePackage(packageReader, identity)
	if err != nil {
		return false, fmt.Errorf("check satellite package: %w", err)
	}

	if !isSatellite {
		return false, nil
	}

	// Extract and merge satellite files
	if err := extractSatelliteFilesV3(packageReader, runtimeIdentity, versionResolver, saveMode, logger); err != nil {
		return false, fmt.Errorf("extract satellite files: %w", err)
	}

	if logger != nil {
		logger.Info("Satellite package %s merged into %s", identity, runtimeIdentity)
	}

	return true, nil
}
