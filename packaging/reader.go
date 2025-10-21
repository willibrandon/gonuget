// Package packaging provides read and write access to NuGet packages (.nupkg files).
package packaging

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"

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
