package packaging

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PackageFileExtractor handles file extraction with XML doc compression.
// Reference: PackageFileExtractor class in NuGet.Packaging
type PackageFileExtractor struct {
	packageFiles       []string
	xmlDocFileSaveMode XMLDocFileSaveMode
	xmlDocFiles        map[string]bool // Files identified as XML docs
}

// NewPackageFileExtractor creates file extractor.
func NewPackageFileExtractor(packageFiles []string, saveMode XMLDocFileSaveMode) *PackageFileExtractor {
	extractor := &PackageFileExtractor{
		packageFiles:       packageFiles,
		xmlDocFileSaveMode: saveMode,
		xmlDocFiles:        make(map[string]bool),
	}

	// Identify XML documentation files
	if saveMode != XMLDocFileSaveModeNone {
		extractor.identifyXMLDocFiles()
	}

	return extractor
}

// identifyXMLDocFiles scans package to find XML documentation files.
// Reference: PackageFileExtractor.IsXmlDocFile
func (e *PackageFileExtractor) identifyXMLDocFiles() {
	// Build map of assemblies (DLL/EXE files) with lowercase keys for case-insensitive lookup
	assemblies := make(map[string]bool)
	for _, file := range e.packageFiles {
		lower := strings.ToLower(file)
		if strings.HasSuffix(lower, ".dll") || strings.HasSuffix(lower, ".exe") {
			assemblies[lower] = true
		}
	}

	// Find XML files in lib/ and ref/ folders with corresponding assemblies
	for _, file := range e.packageFiles {
		lower := strings.ToLower(file)

		// Must be XML file
		if !strings.HasSuffix(lower, ".xml") {
			continue
		}

		// Must be in lib/ or ref/ folder
		if !strings.HasPrefix(lower, "lib/") && !strings.HasPrefix(lower, "ref/") {
			continue
		}

		// Check if corresponding assembly exists (case-insensitive)
		baseName := strings.TrimSuffix(lower, ".xml")
		dllPath := baseName + ".dll"
		exePath := baseName + ".exe"

		if assemblies[dllPath] || assemblies[exePath] {
			e.xmlDocFiles[file] = true
		}

		// Check for resource assemblies (culture-specific)
		// Pattern: lib/net6.0/ja-jp/MyLib.resources.dll -> lib/net6.0/ja-jp/MyLib.xml
		if strings.Contains(lower, "/") {
			parts := strings.Split(lower, "/")
			if len(parts) >= 2 {
				// Check if parent directory has assembly
				parentDir := filepath.Dir(lower)
				parentBase := filepath.Join(filepath.Dir(parentDir), filepath.Base(baseName))
				parentDll := parentBase + ".dll"
				parentExe := parentBase + ".exe"

				if assemblies[parentDll] || assemblies[parentExe] {
					e.xmlDocFiles[file] = true
				}
			}
		}
	}
}

// ExtractPackageFile extracts a single file with special handling.
// Returns extracted path or empty string if skipped.
func (e *PackageFileExtractor) ExtractPackageFile(
	source string,
	target string,
	stream io.Reader,
) (string, error) {
	// Check if this is an XML doc file
	isXmlDoc := e.xmlDocFiles[source]

	// Handle XML doc save modes
	if isXmlDoc {
		switch e.xmlDocFileSaveMode {
		case XMLDocFileSaveModeSkip:
			// Skip extraction
			return "", nil

		case XMLDocFileSaveModeCompress:
			// Compress to .xml.zip
			return e.compressXmlDoc(target, stream)
		}
	}

	// Normal extraction
	return CopyToFile(stream, target)
}

// compressXmlDoc compresses XML file to .xml.zip.
func (e *PackageFileExtractor) compressXmlDoc(target string, stream io.Reader) (string, error) {
	// Change extension to .xml.zip
	zipTarget := target + ".zip"

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(zipTarget), 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	// Create ZIP file
	zipFile, err := os.Create(zipTarget)
	if err != nil {
		return "", fmt.Errorf("create zip: %w", err)
	}
	defer func() { _ = zipFile.Close() }()

	zipWriter := zip.NewWriter(zipFile)
	defer func() { _ = zipWriter.Close() }()

	// Create entry with original XML filename
	entryName := filepath.Base(target)
	entry, err := zipWriter.Create(entryName)
	if err != nil {
		return "", fmt.Errorf("create zip entry: %w", err)
	}

	// Copy XML content to ZIP entry
	if _, err := io.Copy(entry, stream); err != nil {
		return "", fmt.Errorf("write zip entry: %w", err)
	}

	return zipTarget, nil
}
