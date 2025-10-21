package packaging

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
)

// OPC file paths
const (
	OPCContentTypesPath   = "[Content_Types].xml"
	OPCRelationshipsPath  = "_rels/.rels"
	OPCCorePropertiesPath = "package/services/metadata/core-properties/"
	OPCManifestRelType    = "http://schemas.microsoft.com/packaging/2010/07/manifest"
)

// OPC namespaces
const (
	OPCContentTypesNamespace   = "http://schemas.openxmlformats.org/package/2006/content-types"
	OPCRelationshipsNamespace  = "http://schemas.openxmlformats.org/package/2006/relationships"
	OPCCorePropertiesNamespace = "http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
	DCNamespace                = "http://purl.org/dc/elements/1.1/"
	DCTermsNamespace           = "http://purl.org/dc/terms/"
	XSINamespace               = "http://www.w3.org/2001/XMLSchema-instance"
)

// OPC content types
const (
	RelationshipContentType   = "application/vnd.openxmlformats-package.relationships+xml"
	CorePropertiesContentType = "application/vnd.openxmlformats-package.core-properties+xml"
	DefaultContentType        = "application/octet"
)

// ContentTypesXML represents [Content_Types].xml structure.
type ContentTypesXML struct {
	XMLName   xml.Name              `xml:"Types"`
	Xmlns     string                `xml:"xmlns,attr"`
	Defaults  []ContentTypeDefault  `xml:"Default"`
	Overrides []ContentTypeOverride `xml:"Override,omitempty"`
}

// ContentTypeDefault maps file extension to content type.
type ContentTypeDefault struct {
	Extension   string `xml:"Extension,attr"`
	ContentType string `xml:"ContentType,attr"`
}

// ContentTypeOverride maps specific file to content type.
type ContentTypeOverride struct {
	PartName    string `xml:"PartName,attr"`
	ContentType string `xml:"ContentType,attr"`
}

// RelationshipsXML represents _rels/.rels structure.
type RelationshipsXML struct {
	XMLName       xml.Name       `xml:"Relationships"`
	Xmlns         string         `xml:"xmlns,attr"`
	Relationships []Relationship `xml:"Relationship"`
}

// Relationship represents a single relationship.
type Relationship struct {
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
	ID     string `xml:"Id,attr"`
}

// CorePropertiesXML represents package/services/metadata/core-properties/*.psmdcp.
type CorePropertiesXML struct {
	XMLName        xml.Name `xml:"http://schemas.openxmlformats.org/package/2006/metadata/core-properties coreProperties"`
	XmlnsDC        string   `xml:"xmlns:dc,attr"`
	XmlnsDCTerms   string   `xml:"xmlns:dcterms,attr"`
	XmlnsXSI       string   `xml:"xmlns:xsi,attr"`
	Creator        string   `xml:"http://purl.org/dc/elements/1.1/ creator,omitempty"`
	Description    string   `xml:"http://purl.org/dc/elements/1.1/ description,omitempty"`
	Identifier     string   `xml:"http://purl.org/dc/elements/1.1/ identifier,omitempty"`
	Version        string   `xml:"version,omitempty"`
	Keywords       string   `xml:"keywords,omitempty"`
	LastModifiedBy string   `xml:"lastModifiedBy,omitempty"`
}

// GenerateContentTypes generates [Content_Types].xml based on package files.
func GenerateContentTypes(files []PackageFile) (*ContentTypesXML, error) {
	contentTypes := &ContentTypesXML{
		Xmlns: OPCContentTypesNamespace,
	}

	// Required defaults for OPC compliance
	contentTypes.Defaults = []ContentTypeDefault{
		{
			Extension:   "rels",
			ContentType: RelationshipContentType,
		},
		{
			Extension:   "psmdcp",
			ContentType: CorePropertiesContentType,
		},
	}

	// Collect extensions and files without extensions
	extensions := make(map[string]bool)
	var filesWithoutExtension []string

	for _, file := range files {
		ext := getFileExtension(file.TargetPath)
		if ext == "" {
			filesWithoutExtension = append(filesWithoutExtension, file.TargetPath)
		} else {
			// Remove leading dot
			ext = strings.TrimPrefix(ext, ".")
			extensions[ext] = true
		}
	}

	// Add extension defaults
	// Convert map to sorted slice for deterministic output
	var sortedExtensions []string
	for ext := range extensions {
		// Skip extensions that are already defined
		if ext != "rels" && ext != "psmdcp" {
			sortedExtensions = append(sortedExtensions, ext)
		}
	}
	sort.Strings(sortedExtensions)

	for _, ext := range sortedExtensions {
		contentTypes.Defaults = append(contentTypes.Defaults, ContentTypeDefault{
			Extension:   ext,
			ContentType: DefaultContentType,
		})
	}

	// Add overrides for files without extensions
	sort.Strings(filesWithoutExtension)
	for _, file := range filesWithoutExtension {
		// Part names must start with /
		partName := "/" + normalizePackagePath(file)
		contentTypes.Overrides = append(contentTypes.Overrides, ContentTypeOverride{
			PartName:    partName,
			ContentType: DefaultContentType,
		})
	}

	return contentTypes, nil
}

func getFileExtension(filePath string) string {
	ext := path.Ext(filePath)
	return strings.ToLower(ext)
}

// WriteContentTypes writes [Content_Types].xml to the ZIP archive.
func WriteContentTypes(zipWriter *zip.Writer, files []PackageFile) error {
	contentTypes, err := GenerateContentTypes(files)
	if err != nil {
		return fmt.Errorf("generate content types: %w", err)
	}

	// Create ZIP entry
	writer, err := zipWriter.Create(OPCContentTypesPath)
	if err != nil {
		return fmt.Errorf("create content types entry: %w", err)
	}

	// Write XML with declaration
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("write XML header: %w", err)
	}

	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	if err := encoder.Encode(contentTypes); err != nil {
		return fmt.Errorf("encode content types: %w", err)
	}

	return nil
}

// GenerateRelationships generates _rels/.rels for the package.
func GenerateRelationships(nuspecFileName string, corePropertiesPath string) *RelationshipsXML {
	rels := &RelationshipsXML{
		Xmlns: OPCRelationshipsNamespace,
	}

	// Relationship to .nuspec (manifest)
	rels.Relationships = append(rels.Relationships, Relationship{
		Type:   OPCManifestRelType,
		Target: "/" + nuspecFileName,
		ID:     GenerateRelationshipID(),
	})

	// Relationship to core properties (if exists)
	if corePropertiesPath != "" {
		rels.Relationships = append(rels.Relationships, Relationship{
			Type:   "http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties",
			Target: "/" + corePropertiesPath,
			ID:     GenerateRelationshipID(),
		})
	}

	return rels
}

// GenerateRelationshipID generates a unique relationship ID.
func GenerateRelationshipID() string {
	// Use timestamp-based ID similar to NuGet
	// Format: R + hex timestamp
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("R%X", timestamp)
}

// WriteRelationships writes _rels/.rels to the ZIP archive.
func WriteRelationships(zipWriter *zip.Writer, nuspecFileName string, corePropertiesPath string) error {
	rels := GenerateRelationships(nuspecFileName, corePropertiesPath)

	// Create directory entry for _rels/
	if _, err := zipWriter.Create("_rels/"); err != nil {
		return fmt.Errorf("create _rels directory: %w", err)
	}

	// Create ZIP entry for .rels file
	writer, err := zipWriter.Create(OPCRelationshipsPath)
	if err != nil {
		return fmt.Errorf("create relationships entry: %w", err)
	}

	// Write XML with declaration
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("write XML header: %w", err)
	}

	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	if err := encoder.Encode(rels); err != nil {
		return fmt.Errorf("encode relationships: %w", err)
	}

	return nil
}

// GenerateCoreProperties generates core properties metadata.
func GenerateCoreProperties(metadata PackageMetadata) *CorePropertiesXML {
	props := &CorePropertiesXML{
		XmlnsDC:      DCNamespace,
		XmlnsDCTerms: DCTermsNamespace,
		XmlnsXSI:     XSINamespace,
	}

	// Creator (authors)
	if len(metadata.Authors) > 0 {
		props.Creator = strings.Join(metadata.Authors, ", ")
	}

	// Description
	props.Description = metadata.Description

	// Identifier (package ID)
	props.Identifier = metadata.ID

	// Version
	if metadata.Version != nil {
		props.Version = metadata.Version.String()
	}

	// Keywords (tags)
	if len(metadata.Tags) > 0 {
		props.Keywords = strings.Join(metadata.Tags, " ")
	}

	// Last modified by (NuGet client identifier)
	props.LastModifiedBy = "gonuget"

	return props
}

// WriteCoreProperties writes core properties to the ZIP archive.
func WriteCoreProperties(zipWriter *zip.Writer, metadata PackageMetadata) (string, error) {
	props := GenerateCoreProperties(metadata)

	// Generate unique filename with timestamp
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("%s%016x.psmdcp", OPCCorePropertiesPath, timestamp)

	// Create directory structure
	if _, err := zipWriter.Create("package/"); err != nil {
		return "", fmt.Errorf("create package directory: %w", err)
	}
	if _, err := zipWriter.Create("package/services/"); err != nil {
		return "", fmt.Errorf("create services directory: %w", err)
	}
	if _, err := zipWriter.Create("package/services/metadata/"); err != nil {
		return "", fmt.Errorf("create metadata directory: %w", err)
	}
	if _, err := zipWriter.Create(OPCCorePropertiesPath); err != nil {
		return "", fmt.Errorf("create core-properties directory: %w", err)
	}

	// Create ZIP entry for .psmdcp file
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return "", fmt.Errorf("create core properties entry: %w", err)
	}

	// Write XML with declaration
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return "", fmt.Errorf("write XML header: %w", err)
	}

	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	if err := encoder.Encode(props); err != nil {
		return "", fmt.Errorf("encode core properties: %w", err)
	}

	return filename, nil
}
