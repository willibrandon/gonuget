// Package packaging provides read and write access to NuGet packages (.nupkg files).
package packaging

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/willibrandon/gonuget/packaging/signatures"
	"github.com/willibrandon/gonuget/version"
)

// SignaturePath is the path to the signature file in a signed package.
// Reference: SigningSpecificationsV1.cs
const SignaturePath = ".signature.p7s"

// PackageReader provides read access to .nupkg files.
type PackageReader struct {
	zipReader   *zip.ReadCloser
	zipReaderAt *zip.Reader // For in-memory ZIPs
	isClosable  bool

	// Cached values
	isSigned    *bool
	identity    *PackageIdentity
	nuspecEntry *zip.File
}

// PackageIdentity represents a package ID and version.
type PackageIdentity struct {
	ID      string
	Version *version.NuGetVersion
}

// String returns "ID Version" format.
func (p *PackageIdentity) String() string {
	return fmt.Sprintf("%s %s", p.ID, p.Version.String())
}

// OpenPackage opens a .nupkg file from a file path.
func OpenPackage(path string) (*PackageReader, error) {
	zipReader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open package: %w", err)
	}

	return &PackageReader{
		zipReader:  zipReader,
		isClosable: true,
	}, nil
}

// OpenPackageFromReaderAt opens a package from a ReaderAt.
func OpenPackageFromReaderAt(r io.ReaderAt, size int64) (*PackageReader, error) {
	zipReader, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("open package from reader: %w", err)
	}

	return &PackageReader{
		zipReaderAt: zipReader,
		isClosable:  false,
	}, nil
}

// Close closes the package reader.
func (r *PackageReader) Close() error {
	if !r.isClosable || r.zipReader == nil {
		return nil
	}
	return r.zipReader.Close()
}

// Files returns the list of files in the ZIP.
func (r *PackageReader) Files() []*zip.File {
	if r.zipReader != nil {
		return r.zipReader.File
	}
	return r.zipReaderAt.File
}

// IsSigned checks if the package contains a signature file.
// Reference: PackageArchiveReader.cs
func (r *PackageReader) IsSigned() bool {
	if r.isSigned != nil {
		return *r.isSigned
	}

	signed := false
	for _, file := range r.Files() {
		// Exact match on signature path
		// Reference: SignedPackageArchiveUtility.cs
		if file.Name == SignaturePath {
			signed = true
			break
		}
	}

	r.isSigned = &signed
	return signed
}

// GetSignatureFile returns the signature file if package is signed.
func (r *PackageReader) GetSignatureFile() (*zip.File, error) {
	for _, file := range r.Files() {
		if file.Name == SignaturePath {
			return file, nil
		}
	}
	return nil, ErrPackageNotSigned
}

// GetNuspecFile finds and returns the .nuspec file entry.
// Nuspec should be at the root level with .nuspec extension.
func (r *PackageReader) GetNuspecFile() (*zip.File, error) {
	if r.nuspecEntry != nil {
		return r.nuspecEntry, nil
	}

	var candidates []*zip.File
	for _, file := range r.Files() {
		// Nuspec must be at root (no directory separator)
		if !strings.Contains(file.Name, "/") && strings.HasSuffix(strings.ToLower(file.Name), ".nuspec") {
			candidates = append(candidates, file)
		}
	}

	if len(candidates) == 0 {
		return nil, ErrNuspecNotFound
	}

	if len(candidates) > 1 {
		return nil, ErrMultipleNuspecs
	}

	r.nuspecEntry = candidates[0]
	return r.nuspecEntry, nil
}

// OpenNuspec opens the .nuspec file for reading.
func (r *PackageReader) OpenNuspec() (io.ReadCloser, error) {
	nuspecFile, err := r.GetNuspecFile()
	if err != nil {
		return nil, err
	}

	return nuspecFile.Open()
}

// GetFile finds a file by path (case-insensitive).
func (r *PackageReader) GetFile(filePath string) (*zip.File, error) {
	// Normalize path separators
	normalizedPath := strings.ReplaceAll(filePath, "\\", "/")

	for _, file := range r.Files() {
		if strings.EqualFold(file.Name, normalizedPath) {
			return file, nil
		}
	}

	return nil, fmt.Errorf("file not found: %s", filePath)
}

// HasFile checks if a file exists in the package.
func (r *PackageReader) HasFile(filePath string) bool {
	_, err := r.GetFile(filePath)
	return err == nil
}

// GetFiles returns files matching a path prefix.
func (r *PackageReader) GetFiles(prefix string) []*zip.File {
	normalizedPrefix := strings.ReplaceAll(prefix, "\\", "/")

	var matches []*zip.File
	for _, file := range r.Files() {
		if strings.HasPrefix(strings.ToLower(file.Name), strings.ToLower(normalizedPrefix)) {
			matches = append(matches, file)
		}
	}

	return matches
}

// GetNuspec reads and parses the .nuspec file.
func (r *PackageReader) GetNuspec() (*Nuspec, error) {
	nuspecReader, err := r.OpenNuspec()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = nuspecReader.Close()
	}()

	return ParseNuspec(nuspecReader)
}

// GetIdentity returns the package identity from the nuspec.
// Reference: PackageArchiveReader.GetIdentity
func (r *PackageReader) GetIdentity() (*PackageIdentity, error) {
	if r.identity != nil {
		return r.identity, nil
	}

	nuspec, err := r.GetNuspec()
	if err != nil {
		return nil, err
	}

	identity, err := nuspec.GetParsedIdentity()
	if err != nil {
		return nil, err
	}

	r.identity = identity
	return identity, nil
}

// ValidatePackagePath checks for path traversal attacks.
// Reference: PackageBuilder validation logic
func ValidatePackagePath(filePath string) error {
	// Normalize separators
	normalized := strings.ReplaceAll(filePath, "\\", "/")

	// Check for path traversal
	if strings.Contains(normalized, "..") {
		return ErrInvalidPath
	}

	// Check for absolute paths
	if strings.HasPrefix(normalized, "/") {
		return ErrInvalidPath
	}

	// Check for empty path
	if strings.TrimSpace(normalized) == "" {
		return ErrInvalidPath
	}

	return nil
}

// GetPackageFiles returns all files excluding package metadata.
// This filters out .nuspec, signatures, and OPC files.
func (r *PackageReader) GetPackageFiles() []*zip.File {
	var packageFiles []*zip.File

	for _, file := range r.Files() {
		if !IsPackageMetadataFile(file.Name) {
			packageFiles = append(packageFiles, file)
		}
	}

	return packageFiles
}

// GetLibFiles returns all files in the lib/ folder.
func (r *PackageReader) GetLibFiles() []*zip.File {
	var libFiles []*zip.File

	for _, file := range r.Files() {
		if IsLibFile(file.Name) {
			libFiles = append(libFiles, file)
		}
	}

	return libFiles
}

// GetRefFiles returns all files in the ref/ folder.
func (r *PackageReader) GetRefFiles() []*zip.File {
	var refFiles []*zip.File

	for _, file := range r.Files() {
		if IsRefFile(file.Name) {
			refFiles = append(refFiles, file)
		}
	}

	return refFiles
}

// GetContentFiles returns all files in content/ folders.
func (r *PackageReader) GetContentFiles() []*zip.File {
	var contentFiles []*zip.File

	for _, file := range r.Files() {
		if IsContentFile(file.Name) {
			contentFiles = append(contentFiles, file)
		}
	}

	return contentFiles
}

// GetBuildFiles returns all files in build/ folders.
func (r *PackageReader) GetBuildFiles() []*zip.File {
	var buildFiles []*zip.File

	for _, file := range r.Files() {
		if IsBuildFile(file.Name) {
			buildFiles = append(buildFiles, file)
		}
	}

	return buildFiles
}

// GetToolsFiles returns all files in the tools/ folder.
func (r *PackageReader) GetToolsFiles() []*zip.File {
	var toolsFiles []*zip.File

	for _, file := range r.Files() {
		if IsToolsFile(file.Name) {
			toolsFiles = append(toolsFiles, file)
		}
	}

	return toolsFiles
}

// ExtractFile extracts a single file from the package to the destination path.
// The ZIP path is validated to prevent directory traversal attacks.
func (r *PackageReader) ExtractFile(zipPath, destPath string) error {
	// Validate ZIP path to prevent malicious packages
	if err := ValidatePackagePath(zipPath); err != nil {
		return err
	}

	// Find file in ZIP
	zipFile, err := r.GetFile(zipPath)
	if err != nil {
		return err
	}

	// Create destination directory
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Open ZIP file
	rc, err := zipFile.Open()
	if err != nil {
		return fmt.Errorf("open zip file: %w", err)
	}
	defer func() {
		_ = rc.Close()
	}()

	// Create destination file
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = outFile.Close()
	}()

	// Copy contents
	if _, err := io.Copy(outFile, rc); err != nil {
		return fmt.Errorf("copy file contents: %w", err)
	}

	return nil
}

// ExtractFiles extracts multiple files to a destination directory.
// File paths are preserved relative to the package root.
func (r *PackageReader) ExtractFiles(files []*zip.File, destDir string) error {
	for _, file := range files {
		// Skip directories
		if strings.HasSuffix(file.Name, "/") {
			continue
		}

		// Construct destination path
		destPath := filepath.Join(destDir, file.Name)

		// Validate path
		if err := ValidatePackagePath(file.Name); err != nil {
			return fmt.Errorf("invalid path %q: %w", file.Name, err)
		}

		// Extract file
		if err := r.ExtractFile(file.Name, destPath); err != nil {
			return fmt.Errorf("extract %q: %w", file.Name, err)
		}
	}

	return nil
}

// CopyFileTo copies a file from the package to the provided writer.
func (r *PackageReader) CopyFileTo(zipPath string, writer io.Writer) error {
	// Find file in ZIP
	zipFile, err := r.GetFile(zipPath)
	if err != nil {
		return err
	}

	// Open ZIP file
	rc, err := zipFile.Open()
	if err != nil {
		return fmt.Errorf("open zip file: %w", err)
	}
	defer func() {
		_ = rc.Close()
	}()

	// Copy to writer
	if _, err := io.Copy(writer, rc); err != nil {
		return fmt.Errorf("copy file contents: %w", err)
	}

	return nil
}

// GetPrimarySignature returns the primary signature if package is signed
func (r *PackageReader) GetPrimarySignature() (*signatures.PrimarySignature, error) {
	if !r.IsSigned() {
		return nil, ErrPackageNotSigned
	}

	// Get signature file
	sigFile, err := r.GetSignatureFile()
	if err != nil {
		return nil, err
	}

	// Open and read signature data
	reader, err := sigFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open signature file: %w", err)
	}
	defer func() { _ = reader.Close() }()

	sigData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read signature data: %w", err)
	}

	// Parse signature
	return signatures.ReadSignature(sigData)
}

// IsRepositorySigned checks if package has a repository signature
func (r *PackageReader) IsRepositorySigned() (bool, error) {
	sig, err := r.GetPrimarySignature()
	if err != nil {
		if err == ErrPackageNotSigned {
			return false, nil
		}
		return false, err
	}

	return sig.Type == signatures.SignatureTypeRepository, nil
}

// IsAuthorSigned checks if package has an author signature
func (r *PackageReader) IsAuthorSigned() (bool, error) {
	sig, err := r.GetPrimarySignature()
	if err != nil {
		if err == ErrPackageNotSigned {
			return false, nil
		}
		return false, err
	}

	return sig.Type == signatures.SignatureTypeAuthor, nil
}
