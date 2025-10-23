package packaging

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/willibrandon/gonuget/packaging/signatures"
)

// InstallFromSourceV3 installs package using V3 (PackageReference) layout with file locking.
// Reference: PackageExtractor.InstallFromSourceAsync in NuGet.Packaging
func InstallFromSourceV3(
	ctx context.Context,
	source string,
	packageIdentity *PackageIdentity,
	copyToAsync func(string) error, // Callback to download nupkg to temp location
	versionFolderPathResolver *VersionFolderPathResolver,
	extractionContext *PackageExtractionContext,
) (bool, error) {
	// Get target paths
	targetNupkg := versionFolderPathResolver.GetPackageFilePath(
		packageIdentity.ID, packageIdentity.Version)
	metadataPath := versionFolderPathResolver.GetNupkgMetadataPath(
		packageIdentity.ID, packageIdentity.Version)

	// Fast path: Check if already installed (completion marker exists)
	if _, err := os.Stat(metadataPath); err == nil {
		return false, nil // Already installed, no-op
	}

	// Acquire file lock for concurrent safety
	// Reference: ConcurrencyUtilities.ExecuteWithFileLockedAsync
	unlock, err := acquireFileLock(ctx, targetNupkg)
	if err != nil {
		return false, fmt.Errorf("acquire file lock: %w", err)
	}
	defer unlock()

	// Double-check after acquiring lock
	if _, err := os.Stat(metadataPath); err == nil {
		return false, nil // Already installed by another process
	}

	// Get target directory
	targetPath := versionFolderPathResolver.GetInstallPath(
		packageIdentity.ID, packageIdentity.Version)

	// Clean target if exists (handles broken restores)
	if err := cleanDirectory(targetPath); err != nil {
		return false, fmt.Errorf("clean target directory: %w", err)
	}

	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return false, fmt.Errorf("create target directory: %w", err)
	}

	// Generate temporary file names
	targetTempNupkg := filepath.Join(targetPath, generateTempFileName()+".nupkg")
	tempHashPath := filepath.Join(targetPath, generateTempFileName()+".sha512")
	tempMetadataPath := filepath.Join(targetPath, generateTempFileName()+".metadata")

	var packageHash string
	var contentHash string

	// Download package to temp location
	if err := copyToAsync(targetTempNupkg); err != nil {
		cleanupPartialInstall(targetPath, targetTempNupkg)
		return false, fmt.Errorf("download package: %w", err)
	}

	// Open package reader
	reader, err := OpenPackage(targetTempNupkg)
	if err != nil {
		cleanupPartialInstall(targetPath, targetTempNupkg)
		return false, fmt.Errorf("open package: %w", err)
	}

	// Verify signature (if configured)
	// Note: After this point, we do NOT stop based on cancellation
	// to ensure atomic package installation
	if extractionContext.SignatureVerifier != nil &&
		(extractionContext.PackageSaveMode.HasFlag(PackageSaveModeNuspec) ||
			extractionContext.PackageSaveMode.HasFlag(PackageSaveModeFiles)) {

		if err := extractionContext.SignatureVerifier.VerifySignatureAsync(ctx, reader); err != nil {
			_ = reader.Close()
			cleanupPartialInstall(targetPath, targetTempNupkg)
			return false, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Extract .nuspec if requested
	if extractionContext.PackageSaveMode.HasFlag(PackageSaveModeNuspec) {
		targetNuspec := versionFolderPathResolver.GetManifestFilePath(
			packageIdentity.ID, packageIdentity.Version)

		nuspecFile, err := reader.GetNuspecFile()
		if err != nil {
			_ = reader.Close()
			cleanupPartialInstall(targetPath, targetTempNupkg)
			return false, fmt.Errorf("get nuspec: %w", err)
		}

		stream, err := nuspecFile.Open()
		if err != nil {
			_ = reader.Close()
			cleanupPartialInstall(targetPath, targetTempNupkg)
			return false, fmt.Errorf("open nuspec stream: %w", err)
		}

		if _, err := CopyToFile(stream, targetNuspec); err != nil {
			_ = stream.Close()
			_ = reader.Close()
			cleanupPartialInstall(targetPath, targetTempNupkg)
			return false, fmt.Errorf("extract nuspec: %w", err)
		}
		_ = stream.Close()
	}

	// Extract files if requested
	if extractionContext.PackageSaveMode.HasFlag(PackageSaveModeFiles) {
		packageFiles := reader.GetPackageFiles()

		// Build file list for XML doc extractor
		var fileNames []string
		for _, f := range packageFiles {
			fileNames = append(fileNames, f.Name)
		}

		fileExtractor := NewPackageFileExtractor(fileNames, extractionContext.XMLDocFileSaveMode)

		for _, file := range packageFiles {
			// Skip excluded files
			if shouldExcludeFile(file.Name) || isMetadataFile(file.Name) {
				continue
			}

			targetFilePath := filepath.Join(targetPath, filepath.FromSlash(file.Name))
			stream, err := file.Open()
			if err != nil {
				continue
			}

			_, err = fileExtractor.ExtractPackageFile(file.Name, targetFilePath, stream)
			_ = stream.Close()

			if err != nil && extractionContext.Logger != nil {
				extractionContext.Logger.Warning("Failed to extract %s: %v", file.Name, err)
			}
		}
	}

	// Calculate package hash (SHA512 of entire nupkg) before closing reader
	hash, err := calculateFileHash(targetTempNupkg)
	if err != nil {
		_ = reader.Close()
		cleanupPartialInstall(targetPath, targetTempNupkg)
		return false, fmt.Errorf("calculate hash: %w", err)
	}
	packageHash = base64.StdEncoding.EncodeToString(hash)

	// Get content hash (for signed packages, hash of content excluding signature)
	// Pass the temp nupkg path to calculate signed content hash if needed
	contentHash, err = getContentHash(reader, targetTempNupkg, packageHash)
	if err != nil {
		_ = reader.Close()
		cleanupPartialInstall(targetPath, targetTempNupkg)
		return false, fmt.Errorf("calculate content hash: %w", err)
	}

	// Close reader after all hash calculations
	_ = reader.Close()

	// Write hash file
	hashFilePath := versionFolderPathResolver.GetHashPath(
		packageIdentity.ID, packageIdentity.Version)
	if err := os.WriteFile(tempHashPath, []byte(packageHash), 0644); err != nil {
		cleanupPartialInstall(targetPath, targetTempNupkg)
		return false, fmt.Errorf("write hash file: %w", err)
	}

	// Write metadata file
	metadata := NewNupkgMetadataFile(contentHash, source)
	if err := metadata.WriteToFile(tempMetadataPath); err != nil {
		cleanupPartialInstall(targetPath, targetTempNupkg)
		return false, fmt.Errorf("write metadata: %w", err)
	}

	// Atomic operations: Rename temp files to final locations
	// Order matters: nupkg, hash, then metadata (completion marker)

	if extractionContext.PackageSaveMode.HasFlag(PackageSaveModeNupkg) {
		if err := os.Rename(targetTempNupkg, targetNupkg); err != nil {
			cleanupPartialInstall(targetPath, targetTempNupkg)
			return false, fmt.Errorf("rename nupkg: %w", err)
		}
	} else {
		// Delete temp nupkg if not saving
		_ = os.Remove(targetTempNupkg)
	}

	// Rename hash file (completion signal for PackageRepository)
	if err := os.Rename(tempHashPath, hashFilePath); err != nil {
		return false, fmt.Errorf("rename hash file: %w", err)
	}

	// Rename metadata file (TRUE completion marker)
	if err := os.Rename(tempMetadataPath, metadataPath); err != nil {
		return false, fmt.Errorf("rename metadata file: %w", err)
	}

	if extractionContext.Logger != nil {
		extractionContext.Logger.Info("Installed %s %s",
			packageIdentity.ID, packageIdentity.Version.String())
	}

	return true, nil
}

// isMetadataFile checks if file is package metadata.
func isMetadataFile(path string) bool {
	lowerPath := strings.ToLower(path)
	return strings.HasSuffix(lowerPath, ".nupkg.sha512") ||
		strings.HasSuffix(lowerPath, ".nupkg.metadata")
}

// generateTempFileName generates random temp filename.
func generateTempFileName() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}
	return hex.EncodeToString(b)
}

// calculateFileHash calculates SHA512 hash of file.
func calculateFileHash(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	hash := sha512.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

// cleanDirectory removes all contents of directory but keeps directory.
func cleanDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			_ = os.RemoveAll(entryPath)
		} else {
			_ = os.Remove(entryPath)
		}
	}

	return nil
}

// cleanupPartialInstall cleans up after failed installation.
// For concurrent safety, only removes the temp nupkg and the target directory
// if it's empty. This prevents deleting files from other concurrent installations.
// Reference: NuGet.Client PackageExtractor.DeleteTargetAndTempPaths
func cleanupPartialInstall(targetPath, tempNupkg string) {
	// Remove temp nupkg file only
	if tempNupkg != "" {
		_ = os.Remove(tempNupkg)
	}

	// DO NOT remove directories during cleanup.
	// Reason: In concurrent scenarios, multiple goroutines may be writing
	// temp files to the same version directory. Removing the directory
	// would break other in-progress installations.
	//
	// Empty directories are safe to leave and will be reused or cleaned
	// up by the successful installation.
}

// getContentHash returns the content hash for a package.
// For signed packages, this calculates the hash excluding the signature file.
// For unsigned packages, returns the provided packageHash.
// Reference: NuGet.Client PackageReaderBase.GetContentHash and SignedPackageArchiveUtility.GetPackageContentHash
func getContentHash(reader *PackageReader, nupkgPath string, packageHash string) (string, error) {
	// Check if package is signed
	if !reader.IsSigned() {
		// Unsigned package: content hash equals package hash
		return packageHash, nil
	}

	// Signed package: calculate hash excluding signature
	// Open the nupkg file to get a ReadSeeker for the signed content hash calculation
	file, err := os.Open(nupkgPath)
	if err != nil {
		return "", fmt.Errorf("open nupkg for content hash: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Calculate signed package content hash using signatures package
	// Import: "github.com/willibrandon/gonuget/packaging/signatures"
	contentHash, err := signatures.GetPackageContentHash(file)
	if err != nil {
		return "", fmt.Errorf("calculate signed content hash: %w", err)
	}

	// If contentHash is empty (shouldn't happen for signed package), fall back to package hash
	if contentHash == "" {
		return packageHash, nil
	}

	return contentHash, nil
}
