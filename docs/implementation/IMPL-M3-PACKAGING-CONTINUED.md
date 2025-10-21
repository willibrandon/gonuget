# Milestone 3: Package Operations - Continued (Chunks 5-7)

**Status**: Not Started
**Chunks**: 5-7 (OPC Compliance & Validation)
**Estimated Time**: 8 hours

---

## M3.5: Package Builder - OPC Compliance

**Estimated Time**: 3 hours
**Dependencies**: M3.4

### Overview

Implement Open Packaging Conventions (OPC) compliance for .nupkg files including [Content_Types].xml generation, package relationships (_rels/.rels), and core properties metadata.

### Files to Create/Modify

- `packaging/opc.go` - OPC compliance implementation
- `packaging/opc_test.go` - OPC tests
- `packaging/builder.go` - Add OPC methods to builder

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging/PackageBuilder.cs` (OPC methods)
- OPC specification: ISO/IEC 29500-2:2008

**Critical Pattern** (PackageBuilder.cs WriteOpcContentTypes):
```csharp
private void WriteOpcContentTypes(ZipArchive package, SortedSet<string> extensions,
                                   SortedSet<string> filesWithoutExtensions) {
    ZipArchiveEntry relsEntry = CreateEntry(package, "[Content_Types].xml", CompressionLevel.Optimal);

    XNamespace content = "http://schemas.openxmlformats.org/package/2006/content-types";
    XElement element = new XElement(content + "Types",
        new XElement(content + "Default",
            new XAttribute("Extension", "rels"),
            new XAttribute("ContentType", "application/vnd.openxmlformats-package.relationships+xml")),
        new XElement(content + "Default",
            new XAttribute("Extension", "psmdcp"),
            new XAttribute("ContentType", "application/vnd.openxmlformats-package.core-properties+xml"))
    );

    foreach (var extension in extensions) {
        element.Add(
            new XElement(content + "Default",
                new XAttribute("Extension", extension),
                new XAttribute("ContentType", "application/octet"))
        );
    }

    foreach (var file in filesWithoutExtensions) {
        element.Add(
            new XElement(content + "Override",
                new XAttribute("PartName", "/" + file.Replace('\\', '/')),
                new XAttribute("ContentType", "application/octet"))
        );
    }
}
```

### Implementation Details

**1. OPC Constants and Structures**:

```go
// packaging/opc.go

package packaging

import (
    "archive/zip"
    "encoding/xml"
    "fmt"
    "io"
    "path"
    "sort"
    "strings"
    "time"
)

// OPC file paths
const (
    OPCContentTypesPath    = "[Content_Types].xml"
    OPCRelationshipsPath   = "_rels/.rels"
    OPCCorePropertiesPath  = "package/services/metadata/core-properties/"
    OPCManifestRelType     = "http://schemas.microsoft.com/packaging/2010/07/manifest"
)

// OPC namespaces
// Reference: http://schemas.openxmlformats.org/package/2006/
const (
    OPCContentTypesNamespace   = "http://schemas.openxmlformats.org/package/2006/content-types"
    OPCRelationshipsNamespace  = "http://schemas.openxmlformats.org/package/2006/relationships"
    OPCCorePropertiesNamespace = "http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
    DCNamespace               = "http://purl.org/dc/elements/1.1/"
    DCTermsNamespace          = "http://purl.org/dc/terms/"
    XSINamespace              = "http://www.w3.org/2001/XMLSchema-instance"
)

// OPC content types
const (
    RelationshipContentType    = "application/vnd.openxmlformats-package.relationships+xml"
    CorePropertiesContentType  = "application/vnd.openxmlformats-package.core-properties+xml"
    DefaultContentType         = "application/octet"
)

// ContentTypesXML represents [Content_Types].xml structure
type ContentTypesXML struct {
    XMLName  xml.Name               `xml:"Types"`
    Xmlns    string                 `xml:"xmlns,attr"`
    Defaults []ContentTypeDefault   `xml:"Default"`
    Overrides []ContentTypeOverride `xml:"Override"`
}

// ContentTypeDefault maps file extension to content type
type ContentTypeDefault struct {
    Extension   string `xml:"Extension,attr"`
    ContentType string `xml:"ContentType,attr"`
}

// ContentTypeOverride maps specific file to content type
type ContentTypeOverride struct {
    PartName    string `xml:"PartName,attr"`
    ContentType string `xml:"ContentType,attr"`
}

// RelationshipsXML represents _rels/.rels structure
type RelationshipsXML struct {
    XMLName       xml.Name       `xml:"Relationships"`
    Xmlns         string         `xml:"xmlns,attr"`
    Relationships []Relationship `xml:"Relationship"`
}

// Relationship represents a single relationship
type Relationship struct {
    Type   string `xml:"Type,attr"`
    Target string `xml:"Target,attr"`
    ID     string `xml:"Id,attr"`
}

// CorePropertiesXML represents package/services/metadata/core-properties/*.psmdcp
type CorePropertiesXML struct {
    XMLName     xml.Name `xml:"coreProperties"`
    Xmlns       string   `xml:"xmlns,attr"`
    XmlnsDC     string   `xml:"xmlns:dc,attr"`
    XmlnsDCTerms string  `xml:"xmlns:dcterms,attr"`
    XmlnsXSI    string   `xml:"xmlns:xsi,attr"`

    Creator     string `xml:"dc:creator"`
    Description string `xml:"dc:description"`
    Identifier  string `xml:"dc:identifier"`
    Version     string `xml:"version"`
    Keywords    string `xml:"keywords"`
    LastModifiedBy string `xml:"lastModifiedBy"`
}
```

**2. Content Types Generation**:

```go
// GenerateContentTypes generates [Content_Types].xml based on package files
func GenerateContentTypes(files []PackageFile) (*ContentTypesXML, error) {
    contentTypes := &ContentTypesXML{
        Xmlns: OPCContentTypesNamespace,
    }

    // Required defaults for OPC compliance
    // Reference: PackageBuilder.cs WriteOpcContentTypes
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

// WriteContentTypes writes [Content_Types].xml to the ZIP archive
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
```

**3. Package Relationships Generation**:

```go
// GenerateRelationships generates _rels/.rels for the package
func GenerateRelationships(nuspecFileName string, corePropertiesPath string) *RelationshipsXML {
    rels := &RelationshipsXML{
        Xmlns: OPCRelationshipsNamespace,
    }

    // Relationship to .nuspec (manifest)
    // Reference: PackageBuilder.cs WriteOpcManifestRelationship
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

// GenerateRelationshipID generates a unique relationship ID
// Reference: PackageBuilder.cs GenerateRelationshipId
func GenerateRelationshipID() string {
    // Use timestamp-based ID similar to NuGet
    // Format: R + hex timestamp
    timestamp := time.Now().UnixNano()
    return fmt.Sprintf("R%X", timestamp)
}

// WriteRelationships writes _rels/.rels to the ZIP archive
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
```

**4. Core Properties Generation**:

```go
// GenerateCoreProperties generates core properties metadata
func GenerateCoreProperties(metadata PackageMetadata) *CorePropertiesXML {
    props := &CorePropertiesXML{
        Xmlns:        OPCCorePropertiesNamespace,
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

// WriteCoreProperties writes core properties to the ZIP archive
func WriteCoreProperties(zipWriter *zip.Writer, metadata PackageMetadata) (string, error) {
    props := GenerateCoreProperties(metadata)

    // Generate unique filename with timestamp
    // Reference: PackageBuilder.cs uses GUID
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
```

**5. Add to PackageBuilder**:

```go
// packaging/builder.go additions

// writeOPCFiles writes OPC-required files to the ZIP
func (b *PackageBuilder) writeOPCFiles(zipWriter *zip.Writer, nuspecFileName string) error {
    // Write core properties
    corePropsPath, err := WriteCoreProperties(zipWriter, b.metadata)
    if err != nil {
        return fmt.Errorf("write core properties: %w", err)
    }

    // Write relationships
    if err := WriteRelationships(zipWriter, nuspecFileName, corePropsPath); err != nil {
        return fmt.Errorf("write relationships: %w", err)
    }

    // Write content types
    if err := WriteContentTypes(zipWriter, b.files); err != nil {
        return fmt.Errorf("write content types: %w", err)
    }

    return nil
}
```

### Verification Steps

```bash
# 1. Run OPC tests
go test ./packaging -v -run TestOPC

# 2. Test content types generation
go test ./packaging -v -run TestGenerateContentTypes

# 3. Test relationships generation
go test ./packaging -v -run TestGenerateRelationships

# 4. Test core properties generation
go test ./packaging -v -run TestGenerateCoreProperties

# 5. Validate XML structure
go test ./packaging -v -run TestOPCXMLStructure

# 6. Check test coverage
go test ./packaging -cover
```

### Acceptance Criteria

- [ ] Generate [Content_Types].xml with correct namespace
- [ ] Include required defaults (.rels, .psmdcp)
- [ ] Add defaults for all file extensions
- [ ] Add overrides for files without extensions
- [ ] Generate _rels/.rels with nuspec relationship
- [ ] Generate unique relationship IDs
- [ ] Create core properties with metadata
- [ ] Write core properties to unique .psmdcp file
- [ ] Create proper directory structure
- [ ] XML output is well-formed and indented
- [ ] Deterministic output (sorted extensions)
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement OPC compliance

Add Open Packaging Conventions support:
- [Content_Types].xml generation with defaults and overrides
- _rels/.rels package relationships
- Core properties metadata (.psmdcp)
- Proper XML namespaces per OPC spec
- Deterministic, sorted output

Reference: PackageBuilder.cs WriteOpcContentTypes
Spec: ISO/IEC 29500-2:2008
```

---

## M3.6: Package Builder - File Addition and Save

**Estimated Time**: 2 hours
**Dependencies**: M3.4, M3.5

### Overview

Implement the Save method that writes the complete .nupkg file including nuspec generation, file addition to ZIP, and OPC compliance.

### Files to Create/Modify

- `packaging/builder.go` - Add Save method
- `packaging/nuspec_writer.go` - Nuspec XML generation
- `packaging/builder_test.go` - Save tests

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging/PackageBuilder.cs` Save method ()
- `NuGet.Packaging/Authoring/Manifest.cs` Save method ()

**Save Pattern** (PackageBuilder.cs:400-470):
```csharp
public void Save(Stream stream) {
    // Validation (covered in M3.7)
    PackageIdValidator.ValidatePackageId(Id);
    ValidateDependencies(Version, DependencyGroups);
    ValidateFilesUnique(Files);

    using (var package = new ZipArchive(stream, ZipArchiveMode.Create, leaveOpen: true)) {
        string psmdcpPath = null;

        // Write manifest (.nuspec)
        WriteManifest(package, DetermineMinimumSchemaVersion(Files, DependencyGroups), out psmdcpPath);

        // Write files
        var extensions = new SortedSet<string>();
        var filesWithoutExtensions = new SortedSet<string>();
        WriteFiles(package, extensions, filesWithoutExtensions);

        // Write OPC files
        WriteOpcContentTypes(package, extensions, filesWithoutExtensions);
        WriteOpcPackageProperties(package, psmdcpPath);
    }
}
```

### Implementation Details

**1. Nuspec XML Generation**:

```go
// packaging/nuspec_writer.go

package packaging

import (
    "encoding/xml"
    "fmt"
    "io"
    "strings"
)

// Nuspec schema namespaces
const (
    NuspecNamespaceV5 = "http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd"
    NuspecNamespaceV6 = "http://schemas.microsoft.com/packaging/2013/01/nuspec.xsd"
)

// GenerateNuspecXML generates nuspec XML from package metadata
func GenerateNuspecXML(metadata PackageMetadata) ([]byte, error) {
    // Determine schema version based on features used
    namespace := determineNuspecNamespace(metadata)

    // Build nuspec structure
    nuspec := buildNuspecStructure(metadata, namespace)

    // Encode to XML
    var buf strings.Builder
    buf.WriteString(xml.Header)

    encoder := xml.NewEncoder(&buf)
    encoder.Indent("", "  ")

    if err := encoder.Encode(nuspec); err != nil {
        return nil, fmt.Errorf("encode nuspec: %w", err)
    }

    return []byte(buf.String()), nil
}

func determineNuspecNamespace(metadata PackageMetadata) string {
    // Use v5 namespace by default (most recent)
    // Reference: ManifestSchemaUtility in NuGet.Client
    return NuspecNamespaceV5
}

func buildNuspecStructure(metadata PackageMetadata, namespace string) *Nuspec {
    nuspec := &Nuspec{
        XMLName: xml.Name{
            Space: namespace,
            Local: "package",
        },
        Metadata: NuspecMetadata{
            ID:          metadata.ID,
            Version:     metadata.Version.String(),
            Description: metadata.Description,
        },
    }

    // Authors (required)
    if len(metadata.Authors) > 0 {
        nuspec.Metadata.Authors = strings.Join(metadata.Authors, ", ")
    }

    // Optional fields
    if metadata.Title != "" {
        nuspec.Metadata.Title = metadata.Title
    }

    if len(metadata.Owners) > 0 {
        nuspec.Metadata.Owners = strings.Join(metadata.Owners, ", ")
    }

    if metadata.ProjectURL != nil {
        nuspec.Metadata.ProjectURL = metadata.ProjectURL.String()
    }

    if metadata.IconURL != nil {
        nuspec.Metadata.IconURL = metadata.IconURL.String()
    }

    if metadata.Icon != "" {
        nuspec.Metadata.Icon = metadata.Icon
    }

    if metadata.LicenseURL != nil {
        nuspec.Metadata.LicenseURL = metadata.LicenseURL.String()
    }

    if metadata.LicenseMetadata != nil {
        nuspec.Metadata.License = &LicenseMetadata{
            Type:    metadata.LicenseMetadata.Type,
            Version: metadata.LicenseMetadata.Version,
            Text:    metadata.LicenseMetadata.License,
        }
    }

    nuspec.Metadata.RequireLicenseAcceptance = metadata.RequireLicenseAcceptance
    nuspec.Metadata.DevelopmentDependency = metadata.DevelopmentDependency

    if metadata.Summary != "" {
        nuspec.Metadata.Summary = metadata.Summary
    }

    if metadata.ReleaseNotes != "" {
        nuspec.Metadata.ReleaseNotes = metadata.ReleaseNotes
    }

    if metadata.Copyright != "" {
        nuspec.Metadata.Copyright = metadata.Copyright
    }

    if metadata.Language != "" {
        nuspec.Metadata.Language = metadata.Language
    }

    if len(metadata.Tags) > 0 {
        nuspec.Metadata.Tags = strings.Join(metadata.Tags, " ")
    }

    nuspec.Metadata.Serviceable = metadata.Serviceable

    if metadata.Readme != "" {
        nuspec.Metadata.Readme = metadata.Readme
    }

    if metadata.MinClientVersion != nil {
        nuspec.Metadata.MinClientVersion = metadata.MinClientVersion.String()
    }

    // Dependencies
    if len(metadata.DependencyGroups) > 0 {
        nuspec.Metadata.Dependencies = &DependenciesElement{}

        for _, group := range metadata.DependencyGroups {
            depGroup := DependencyGroup{}

            if group.TargetFramework != nil && !group.TargetFramework.IsAny() {
                depGroup.TargetFramework = group.TargetFramework.GetShortFolderName()
            }

            for _, dep := range group.Dependencies {
                dependency := Dependency{
                    ID: dep.ID,
                }

                if dep.VersionRange != nil {
                    dependency.Version = dep.VersionRange.String()
                }

                if len(dep.Include) > 0 {
                    dependency.Include = strings.Join(dep.Include, ";")
                }

                if len(dep.Exclude) > 0 {
                    dependency.Exclude = strings.Join(dep.Exclude, ";")
                }

                depGroup.Dependencies = append(depGroup.Dependencies, dependency)
            }

            nuspec.Metadata.Dependencies.Groups = append(nuspec.Metadata.Dependencies.Groups, depGroup)
        }
    }

    // Framework references
    if len(metadata.FrameworkReferenceGroups) > 0 {
        nuspec.Metadata.FrameworkReferences = &FrameworkReferencesElement{}

        for _, group := range metadata.FrameworkReferenceGroups {
            fwRefGroup := FrameworkReferenceGroup{
                TargetFramework: group.TargetFramework.GetShortFolderName(),
            }

            for _, ref := range group.References {
                fwRefGroup.References = append(fwRefGroup.References, FrameworkReference{
                    Name: ref,
                })
            }

            nuspec.Metadata.FrameworkReferences.Groups = append(nuspec.Metadata.FrameworkReferences.Groups, fwRefGroup)
        }
    }

    // Package types
    if len(metadata.PackageTypes) > 0 {
        for _, pt := range metadata.PackageTypes {
            pkgType := PackageType{
                Name: pt.Name,
            }

            if pt.Version != nil {
                pkgType.Version = pt.Version.String()
            }

            nuspec.Metadata.PackageTypes = append(nuspec.Metadata.PackageTypes, pkgType)
        }
    }

    // Repository metadata
    if metadata.Repository != nil {
        nuspec.Metadata.Repository = &RepositoryMetadata{
            Type:   metadata.Repository.Type,
            URL:    metadata.Repository.URL,
            Branch: metadata.Repository.Branch,
            Commit: metadata.Repository.Commit,
        }
    }

    return nuspec
}
```

**2. Save Implementation**:

```go
// packaging/builder.go

import (
    "archive/zip"
    "fmt"
    "io"
    "os"
)

// Save writes the package to a stream
func (b *PackageBuilder) Save(writer io.Writer) error {
    // Basic validation
    if err := b.validateBasic(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    // Create ZIP archive
    zipWriter := zip.NewWriter(writer)
    defer zipWriter.Close()

    // Write nuspec
    nuspecFileName, err := b.writeNuspec(zipWriter)
    if err != nil {
        return fmt.Errorf("write nuspec: %w", err)
    }

    // Write package files
    if err := b.writeFiles(zipWriter); err != nil {
        return fmt.Errorf("write files: %w", err)
    }

    // Write OPC files
    if err := b.writeOPCFiles(zipWriter, nuspecFileName); err != nil {
        return fmt.Errorf("write OPC files: %w", err)
    }

    return nil
}

// SaveToFile writes the package to a file
func (b *PackageBuilder) SaveToFile(path string) error {
    file, err := os.Create(path)
    if err != nil {
        return fmt.Errorf("create file: %w", err)
    }
    defer file.Close()

    return b.Save(file)
}

func (b *PackageBuilder) validateBasic() error {
    if b.metadata.ID == "" {
        return fmt.Errorf("package ID is required")
    }

    if b.metadata.Version == nil {
        return fmt.Errorf("package version is required")
    }

    if b.metadata.Description == "" {
        return fmt.Errorf("package description is required")
    }

    if len(b.metadata.Authors) == 0 {
        return fmt.Errorf("package authors are required")
    }

    return nil
}

func (b *PackageBuilder) writeNuspec(zipWriter *zip.Writer) (string, error) {
    // Generate nuspec XML
    nuspecXML, err := GenerateNuspecXML(b.metadata)
    if err != nil {
        return "", err
    }

    // Nuspec file name: {ID}.nuspec
    nuspecFileName := b.metadata.ID + ".nuspec"

    // Create ZIP entry
    writer, err := zipWriter.Create(nuspecFileName)
    if err != nil {
        return "", fmt.Errorf("create nuspec entry: %w", err)
    }

    // Write XML
    if _, err := writer.Write(nuspecXML); err != nil {
        return "", fmt.Errorf("write nuspec: %w", err)
    }

    return nuspecFileName, nil
}

func (b *PackageBuilder) writeFiles(zipWriter *zip.Writer) error {
    for _, file := range b.files {
        if err := b.writeFile(zipWriter, file); err != nil {
            return fmt.Errorf("write file %s: %w", file.TargetPath, err)
        }
    }

    return nil
}

func (b *PackageBuilder) writeFile(zipWriter *zip.Writer, file PackageFile) error {
    // Create ZIP entry
    writer, err := zipWriter.Create(file.TargetPath)
    if err != nil {
        return fmt.Errorf("create ZIP entry: %w", err)
    }

    // Write from source
    if file.SourcePath != "" {
        // Read from disk
        sourceFile, err := os.Open(file.SourcePath)
        if err != nil {
            return fmt.Errorf("open source file: %w", err)
        }
        defer sourceFile.Close()

        if _, err := io.Copy(writer, sourceFile); err != nil {
            return fmt.Errorf("copy from source: %w", err)
        }
    } else if file.Content != nil {
        // Write from bytes
        if _, err := writer.Write(file.Content); err != nil {
            return fmt.Errorf("write content: %w", err)
        }
    } else if file.Reader != nil {
        // Write from reader
        if _, err := io.Copy(writer, file.Reader); err != nil {
            return fmt.Errorf("copy from reader: %w", err)
        }
    } else {
        return fmt.Errorf("no content source for file")
    }

    return nil
}
```

### Verification Steps

```bash
# 1. Run Save tests
go test ./packaging -v -run TestBuilderSave

# 2. Test nuspec generation
go test ./packaging -v -run TestGenerateNuspec

# 3. Create a real package
go test ./packaging -v -run TestCreateRealPackage

# 4. Verify package structure with unzip
go test ./packaging -v -run TestVerifyPackageStructure

# 5. Check test coverage
go test ./packaging -cover
```

### Acceptance Criteria

- [ ] Generate valid nuspec XML from metadata
- [ ] Write nuspec to ZIP with correct filename
- [ ] Write all added files to ZIP
- [ ] Support files from disk, bytes, and readers
- [ ] Write OPC files ([Content_Types].xml, _rels/.rels, .psmdcp)
- [ ] Create valid .nupkg ZIP structure
- [ ] Save to io.Writer
- [ ] Save to file path
- [ ] Basic validation before save
- [ ] Proper error handling
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement package Save with nuspec generation

Add package building with:
- Nuspec XML generation from metadata
- ZIP archive creation with all files
- OPC compliance (content types, relationships, core properties)
- Save to writer or file
- Basic validation before save

Reference: PackageBuilder.cs Save method
```

---

## M3.7: Package Validation Rules

**Estimated Time**: 3 hours
**Dependencies**: M3.4, M3.5, M3.6

### Overview

Implement comprehensive package validation including ID validation, dependency validation, file path validation, license validation, and framework validation.

### Files to Create/Modify

- `packaging/validation.go` - Validation rules implementation
- `packaging/validation_test.go` - Validation tests
- `packaging/builder.go` - Add validation calls to Save

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging/PackageBuilder.cs` validation methods ()
- `NuGet.Packaging/Rules/` - Validation rule implementations
- `NuGet.Packaging.Core/PackageIdValidator.cs`

**ID Validation** (PackageIdValidator.cs:20-45):
```csharp
public static bool IsValidPackageId(string packageId) {
    if (string.IsNullOrWhiteSpace(packageId)) {
        return false;
    }

    if (packageId.Length > MaxPackageIdLength) {
        return false;
    }

    // Must start with letter or underscore
    if (!char.IsLetter(packageId[0]) && packageId[0] != '_') {
        return false;
    }

    // Can only contain letters, digits, periods, hyphens, underscores
    foreach (char c in packageId) {
        if (!char.IsLetterOrDigit(c) && c != '.' && c != '-' && c != '_') {
            return false;
        }
    }

    return true;
}
```

### Implementation Details

**1. Package ID Validation**:

```go
// packaging/validation.go

package packaging

import (
    "fmt"
    "regexp"
    "strings"
    "unicode"

    "github.com/willibrandon/gonuget/version"
)

const (
    // MaxPackageIDLength is the maximum allowed package ID length
    // Reference: PackageIdValidator.cs
    MaxPackageIDLength = 100
)

var (
    // Package ID pattern: must start with letter or underscore,
    // can contain letters, digits, periods, hyphens, underscores
    packageIDPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9._-]*$`)
)

// ValidatePackageID validates a package ID
// Reference: PackageIdValidator.cs IsValidPackageId
func ValidatePackageID(id string) error {
    if id == "" {
        return fmt.Errorf("package ID cannot be empty")
    }

    if len(id) > MaxPackageIDLength {
        return fmt.Errorf("package ID cannot exceed %d characters", MaxPackageIDLength)
    }

    // Must start with letter or underscore
    firstChar := rune(id[0])
    if !unicode.IsLetter(firstChar) && firstChar != '_' {
        return fmt.Errorf("package ID must start with a letter or underscore")
    }

    // Check pattern
    if !packageIDPattern.MatchString(id) {
        return fmt.Errorf("package ID contains invalid characters (only letters, digits, '.', '-', '_' allowed)")
    }

    return nil
}
```

**2. Dependency Validation**:

```go
// ValidateDependencies validates all dependency groups
// Reference: PackageBuilder.cs ValidateDependencies
func ValidateDependencies(packageID string, packageVersion *version.NuGetVersion, groups []PackageDependencyGroup) error {
    for _, group := range groups {
        // Check for duplicate dependencies in the same group
        seen := make(map[string]bool)
        for _, dep := range group.Dependencies {
            depKey := strings.ToLower(dep.ID)
            if seen[depKey] {
                return fmt.Errorf("duplicate dependency %q in group for %s", dep.ID, group.TargetFramework.GetShortFolderName())
            }
            seen[depKey] = true

            // Validate dependency version range
            if err := validateDependencyVersion(dep); err != nil {
                return fmt.Errorf("invalid dependency %q: %w", dep.ID, err)
            }

            // Check for self-dependency
            if strings.EqualFold(dep.ID, packageID) {
                return fmt.Errorf("package cannot depend on itself")
            }
        }
    }

    return nil
}

// validateDependencyVersion validates a dependency version range
// Reference: Manifest.cs ValidateDependencyVersion
func validateDependencyVersion(dep PackageDependency) error {
    if dep.VersionRange == nil {
        return nil
    }

    vr := dep.VersionRange

    // If both min and max are set
    if vr.MinVersion != nil && vr.MaxVersion != nil {
        // If both exclusive and versions are equal, invalid
        if !vr.IsMinInclusive && !vr.IsMaxInclusive && vr.MinVersion.Equals(vr.MaxVersion) {
            return fmt.Errorf("version range (exclusive) cannot have equal min and max versions")
        }

        // Max must be >= Min
        if vr.MaxVersion.CompareTo(vr.MinVersion) < 0 {
            return fmt.Errorf("max version must be greater than or equal to min version")
        }
    }

    return nil
}
```

**3. File Validation**:

```go
// ValidateFiles validates all files in the package
func ValidateFiles(files []PackageFile) error {
    if len(files) == 0 {
        return fmt.Errorf("package must contain at least one file")
    }

    // Check for duplicates
    seen := make(map[string]bool)
    for _, file := range files {
        normalized := strings.ToLower(normalizePackagePath(file.TargetPath))
        if seen[normalized] {
            return fmt.Errorf("duplicate file path: %s", file.TargetPath)
        }
        seen[normalized] = true

        // Validate path
        if err := ValidatePackagePath(file.TargetPath); err != nil {
            return fmt.Errorf("invalid file path %q: %w", file.TargetPath, err)
        }
    }

    return nil
}
```

**4. License Validation**:

```go
// ValidateLicense validates license metadata
// Reference: PackageBuilder.cs ValidateLicenseFile
func ValidateLicense(metadata PackageMetadata, files []PackageFile) error {
    // If RequireLicenseAcceptance is true, must have license
    if metadata.RequireLicenseAcceptance {
        if metadata.LicenseURL == nil && metadata.LicenseMetadata == nil {
            return fmt.Errorf("requireLicenseAcceptance requires either licenseUrl or license metadata")
        }
    }

    // If both licenseUrl and license metadata, they must match or one must be null
    if metadata.LicenseURL != nil && metadata.LicenseMetadata != nil {
        return fmt.Errorf("cannot specify both licenseUrl and license metadata")
    }

    // If license is a file, verify it exists
    if metadata.LicenseMetadata != nil && metadata.LicenseMetadata.Type == "file" {
        licenseFile := metadata.LicenseMetadata.License
        if !fileExists(files, licenseFile) {
            return fmt.Errorf("license file %q specified but not found in package", licenseFile)
        }
    }

    return nil
}

func fileExists(files []PackageFile, targetPath string) bool {
    normalized := strings.ToLower(normalizePackagePath(targetPath))
    for _, file := range files {
        if strings.ToLower(normalizePackagePath(file.TargetPath)) == normalized {
            return true
        }
    }
    return false
}
```

**5. Icon and Readme Validation**:

```go
// ValidateIcon validates icon file reference
// Reference: PackageBuilder.cs ValidateIconFile
func ValidateIcon(metadata PackageMetadata, files []PackageFile) error {
    if metadata.Icon == "" {
        return nil
    }

    if !fileExists(files, metadata.Icon) {
        return fmt.Errorf("icon file %q specified but not found in package", metadata.Icon)
    }

    // Icon should be in a specific folder or root
    // NuGet recommends icon/ folder or root
    normalized := strings.ToLower(metadata.Icon)
    if !strings.HasPrefix(normalized, "icon/") && strings.Contains(normalized, "/") {
        return fmt.Errorf("icon file should be in 'icon/' folder or at package root")
    }

    return nil
}

// ValidateReadme validates readme file reference
// Reference: PackageBuilder.cs ValidateReadmeFile
func ValidateReadme(metadata PackageMetadata, files []PackageFile) error {
    if metadata.Readme == "" {
        return nil
    }

    if !fileExists(files, metadata.Readme) {
        return fmt.Errorf("readme file %q specified but not found in package", metadata.Readme)
    }

    return nil
}
```

**6. Framework Validation**:

```go
// ValidateFrameworkReferences validates framework reference groups
func ValidateFrameworkReferences(groups []PackageFrameworkReferenceGroup) error {
    for _, group := range groups {
        if group.TargetFramework == nil {
            return fmt.Errorf("framework reference group must have a target framework")
        }

        if len(group.References) == 0 {
            return fmt.Errorf("framework reference group for %s has no references", group.TargetFramework.GetShortFolderName())
        }

        // Check for duplicates
        seen := make(map[string]bool)
        for _, ref := range group.References {
            refKey := strings.ToLower(ref)
            if seen[refKey] {
                return fmt.Errorf("duplicate framework reference %q in group for %s", ref, group.TargetFramework.GetShortFolderName())
            }
            seen[refKey] = true
        }
    }

    return nil
}
```

**7. Complete Validation in Builder**:

```go
// packaging/builder.go updates

// Validate performs comprehensive package validation
func (b *PackageBuilder) Validate() error {
    // Validate ID
    if err := ValidatePackageID(b.metadata.ID); err != nil {
        return fmt.Errorf("package ID validation: %w", err)
    }

    // Validate version
    if b.metadata.Version == nil {
        return fmt.Errorf("package version is required")
    }

    // Validate required metadata
    if b.metadata.Description == "" {
        return fmt.Errorf("package description is required")
    }

    if len(b.metadata.Authors) == 0 {
        return fmt.Errorf("package authors are required")
    }

    // Validate package is not empty
    if len(b.files) == 0 && len(b.metadata.DependencyGroups) == 0 && len(b.metadata.FrameworkReferenceGroups) == 0 {
        return fmt.Errorf("package must contain files, dependencies, or framework references")
    }

    // Validate dependencies
    if err := ValidateDependencies(b.metadata.ID, b.metadata.Version, b.metadata.DependencyGroups); err != nil {
        return fmt.Errorf("dependency validation: %w", err)
    }

    // Validate files
    if err := ValidateFiles(b.files); err != nil {
        return fmt.Errorf("file validation: %w", err)
    }

    // Validate license
    if err := ValidateLicense(b.metadata, b.files); err != nil {
        return fmt.Errorf("license validation: %w", err)
    }

    // Validate icon
    if err := ValidateIcon(b.metadata, b.files); err != nil {
        return fmt.Errorf("icon validation: %w", err)
    }

    // Validate readme
    if err := ValidateReadme(b.metadata, b.files); err != nil {
        return fmt.Errorf("readme validation: %w", err)
    }

    // Validate framework references
    if err := ValidateFrameworkReferences(b.metadata.FrameworkReferenceGroups); err != nil {
        return fmt.Errorf("framework reference validation: %w", err)
    }

    return nil
}

// Update Save to use comprehensive validation
func (b *PackageBuilder) Save(writer io.Writer) error {
    // Comprehensive validation
    if err := b.Validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    // ... rest of Save implementation
}
```

### Verification Steps

```bash
# 1. Run validation tests
go test ./packaging -v -run TestValidation

# 2. Test package ID validation
go test ./packaging -v -run TestValidatePackageID

# 3. Test dependency validation
go test ./packaging -v -run TestValidateDependencies

# 4. Test file validation
go test ./packaging -v -run TestValidateFiles

# 5. Test comprehensive validation
go test ./packaging -v -run TestBuilderValidate

# 6. Check test coverage
go test ./packaging -cover
```

### Acceptance Criteria

- [ ] Validate package ID format and length
- [ ] Detect duplicate dependencies in same group
- [ ] Validate dependency version ranges
- [ ] Prevent self-dependencies
- [ ] Detect duplicate file paths
- [ ] Validate file paths for security
- [ ] Enforce required metadata fields
- [ ] Validate license configuration
- [ ] Verify license file exists if type is "file"
- [ ] Verify icon file exists if specified
- [ ] Verify readme file exists if specified
- [ ] Validate framework references
- [ ] Ensure package is not empty
- [ ] Comprehensive validation before Save
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement package validation rules

Add comprehensive validation:
- Package ID format and length validation
- Dependency validation (duplicates, version ranges, self-deps)
- File validation (duplicates, path security)
- License validation (file existence, mutual exclusion)
- Icon and readme file existence
- Framework reference validation
- Required metadata enforcement

Reference: PackageBuilder.cs validation methods
Reference: PackageIdValidator.cs
```

---

## Summary - Chunks 5-7 Complete

**Total Time for This File**: 8 hours
**Files Created**: 6
**Lines of Code**: ~1,200

**Next File**: IMPL-M3-PACKAGING-CONTINUED-2.md (Chunks 8-10: Package Signatures)

**Dependencies for Next Chunks**:
- M3.8 requires M3.1 (reader for signature access)
- M3.9 requires M3.8 (signature reading)
- M3.10 requires M3.4, M3.5, M3.6 (builder with OPC)
