# Milestone 3: Package Operations - Implementation Guide

**Status**: Not Started
**Chunks**: 14 total (1-4 in this file)
**Estimated Time**: 32 hours total (10 hours for chunks 1-4)

---

## Overview

Milestone 3 implements NuGet package operations including reading, creating, validating, and extracting packages. This milestone builds on the foundation and protocol layers to provide comprehensive package manipulation capabilities.

**Key Features**:
- Package reading from .nupkg (ZIP) files
- Nuspec (package manifest) parsing and validation
- Package creation with OPC compliance
- Package signature verification (PKCS#7)
- Asset selection based on target framework and RID
- Package extraction with integrity validation

**Reference Implementation**: NuGet.Client/src/NuGet.Core/NuGet.Packaging/

---

## M3.1: Package Reader - ZIP Access

**Estimated Time**: 2 hours
**Dependencies**: M1 (Foundation), M2 (Protocol)

### Overview

Implement a package reader that provides access to .nupkg files (ZIP archives) with signature detection, content validation, and efficient file access patterns.

### Files to Create/Modify

- `packaging/reader.go` - Core package reader implementation
- `packaging/reader_test.go` - Reader tests
- `packaging/errors.go` - Packaging-specific errors

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging/PackageArchiveReader.cs` (580 lines)
- `NuGet.Packaging/Signing/SignedPackageArchiveUtility.cs` (lines 28-69 for signature detection)

**Key Patterns**:
```csharp
// Signature detection from PackageArchiveReader.cs
public override async Task<bool> IsSignedAsync(CancellationToken token) {
    using (var zip = new ZipArchive(ZipReadStream, ZipArchiveMode.Read, leaveOpen: true)) {
        var signatureEntry = zip.GetEntry(SigningSpecifications.SignaturePath);
        if (signatureEntry != null &&
            string.Equals(signatureEntry.Name, SigningSpecifications.SignaturePath, StringComparison.Ordinal)) {
            _isSigned = true;
        }
    }
    return _isSigned;
}
```

### Implementation Details

**1. Package Reader Structure**:

```go
package packaging

import (
    "archive/zip"
    "fmt"
    "io"
    "path"
    "strings"

    "github.com/willibrandon/gonuget/version"
)

// PackageReader provides read access to .nupkg files
type PackageReader struct {
    zipReader     *zip.ReadCloser
    zipReaderAt   *zip.Reader // For in-memory ZIPs
    isClosable    bool

    // Cached values
    isSigned      *bool
    identity      *PackageIdentity
    nuspecEntry   *zip.File
}

// PackageIdentity represents a package ID and version
type PackageIdentity struct {
    ID      string
    Version *version.NuGetVersion
}

// String returns "ID Version" format
func (p *PackageIdentity) String() string {
    return fmt.Sprintf("%s %s", p.ID, p.Version.String())
}

// OpenPackage opens a .nupkg file from a file path
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

// OpenPackageFromReaderAt opens a package from a ReaderAt
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

// Close closes the package reader
func (r *PackageReader) Close() error {
    if !r.isClosable || r.zipReader == nil {
        return nil
    }
    return r.zipReader.Close()
}

// Files returns the list of files in the ZIP
func (r *PackageReader) Files() []*zip.File {
    if r.zipReader != nil {
        return r.zipReader.File
    }
    return r.zipReaderAt.File
}
```

**2. Signature Detection**:

```go
// SignaturePath is the path to the signature file in a signed package
// Reference: SigningSpecificationsV1.cs line 11
const SignaturePath = ".signature.p7s"

// IsSigned checks if the package contains a signature file
// Reference: PackageArchiveReader.cs lines 45-60
func (r *PackageReader) IsSigned() bool {
    if r.isSigned != nil {
        return *r.isSigned
    }

    signed := false
    for _, file := range r.Files() {
        // Exact match on signature path
        // Reference: SignedPackageArchiveUtility.cs lines 141-156
        if file.Name == SignaturePath {
            signed = true
            break
        }
    }

    r.isSigned = &signed
    return signed
}

// GetSignatureFile returns the signature file if package is signed
func (r *PackageReader) GetSignatureFile() (*zip.File, error) {
    for _, file := range r.Files() {
        if file.Name == SignaturePath {
            return file, nil
        }
    }
    return nil, fmt.Errorf("package is not signed")
}
```

**3. Nuspec File Access**:

```go
// GetNuspecFile finds and returns the .nuspec file entry
// Nuspec should be at the root level with .nuspec extension
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
        return nil, fmt.Errorf("no .nuspec file found in package")
    }

    if len(candidates) > 1 {
        return nil, fmt.Errorf("multiple .nuspec files found in package")
    }

    r.nuspecEntry = candidates[0]
    return r.nuspecEntry, nil
}

// OpenNuspec opens the .nuspec file for reading
func (r *PackageReader) OpenNuspec() (io.ReadCloser, error) {
    nuspecFile, err := r.GetNuspecFile()
    if err != nil {
        return nil, err
    }

    return nuspecFile.Open()
}
```

**4. File Access Utilities**:

```go
// GetFile finds a file by path (case-insensitive)
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

// HasFile checks if a file exists in the package
func (r *PackageReader) HasFile(filePath string) bool {
    _, err := r.GetFile(filePath)
    return err == nil
}

// GetFiles returns files matching a path prefix
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

// IsPackageFile checks if a file is the .nupkg itself (used during extraction)
func IsPackageFile(fileName string) bool {
    ext := strings.ToLower(path.Ext(fileName))
    return ext == ".nupkg"
}

// IsManifestFile checks if a file is a .nuspec manifest
func IsManifestFile(fileName string) bool {
    ext := strings.ToLower(path.Ext(fileName))
    return ext == ".nuspec" && !strings.Contains(fileName, "/")
}
```

**5. Path Validation**:

```go
// ValidatePackagePath checks for path traversal attacks
// Reference: PackageBuilder validation logic
func ValidatePackagePath(filePath string) error {
    // Normalize separators
    normalized := strings.ReplaceAll(filePath, "\\", "/")

    // Check for path traversal
    if strings.Contains(normalized, "..") {
        return fmt.Errorf("invalid package path: contains '..'")
    }

    // Check for absolute paths
    if strings.HasPrefix(normalized, "/") {
        return fmt.Errorf("invalid package path: absolute path not allowed")
    }

    // Check for empty path
    if strings.TrimSpace(normalized) == "" {
        return fmt.Errorf("invalid package path: empty path")
    }

    return nil
}
```

**6. Error Definitions**:

```go
// packaging/errors.go

package packaging

import "errors"

var (
    // ErrPackageNotSigned indicates the package does not contain a signature
    ErrPackageNotSigned = errors.New("package is not signed")

    // ErrInvalidPackage indicates the package structure is invalid
    ErrInvalidPackage = errors.New("invalid package structure")

    // ErrNuspecNotFound indicates no .nuspec file was found
    ErrNuspecNotFound = errors.New("nuspec file not found")

    // ErrMultipleNuspecs indicates multiple .nuspec files were found
    ErrMultipleNuspecs = errors.New("multiple nuspec files found")

    // ErrInvalidPath indicates an invalid file path (e.g., path traversal)
    ErrInvalidPath = errors.New("invalid file path")
)
```

### Verification Steps

```bash
# 1. Run tests
go test ./packaging -v -run TestPackageReader

# 2. Test with real NuGet package
go test ./packaging -v -run TestOpenRealPackage

# 3. Benchmark file access
go test ./packaging -bench=BenchmarkFileAccess -benchmem

# 4. Check test coverage
go test ./packaging -cover
```

### Acceptance Criteria

- [ ] Can open .nupkg files from file path
- [ ] Can open .nupkg from io.ReaderAt
- [ ] Correctly detects signed packages via .signature.p7s
- [ ] Finds .nuspec file at package root
- [ ] Rejects packages with multiple .nuspec files
- [ ] Validates file paths for traversal attacks
- [ ] Case-insensitive file lookup
- [ ] Efficient file access (lazy loading)
- [ ] Proper resource cleanup (Close)
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement package reader with ZIP access

Add PackageReader for reading .nupkg files with:
- ZIP archive access via archive/zip
- Signature detection (.signature.p7s)
- Nuspec file discovery and validation
- Path traversal protection
- Case-insensitive file lookup

Reference: NuGet.Packaging/PackageArchiveReader.cs
```

---

## M3.2: Package Reader - Nuspec Parser

**Estimated Time**: 3 hours
**Dependencies**: M3.1

### Overview

Implement XML parsing for .nuspec files with support for multiple schema versions, metadata extraction, dependency parsing, and framework reference handling.

### Files to Create/Modify

- `packaging/nuspec.go` - Nuspec parsing and data structures
- `packaging/nuspec_test.go` - Nuspec parser tests
- `packaging/reader.go` - Add nuspec parsing methods to PackageReader

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging/NuspecReader.cs` (450+ lines)
- `NuGet.Packaging.Core/NuspecCoreReaderBase.cs` (227 lines)
- `NuGet.Packaging/Authoring/Manifest.cs` (338 lines)
- `NuGet.Packaging/Authoring/ManifestMetadata.cs` (381 lines)

**Schema Namespace Detection** (Manifest.cs:164-173):
```csharp
private static string GetSchemaNamespace(XDocument document) {
    string schemaNamespace = ManifestSchemaUtility.SchemaVersionV1;
    var rootNameSpace = document.Root.Name.Namespace;
    if (rootNameSpace != null && !String.IsNullOrEmpty(rootNameSpace.NamespaceName)) {
        schemaNamespace = rootNameSpace.NamespaceName;
    }
    return schemaNamespace;
}
```

### Implementation Details

**1. Nuspec Data Structures**:

```go
package packaging

import (
    "encoding/xml"
    "io"
    "strings"

    "github.com/willibrandon/gonuget/frameworks"
    "github.com/willibrandon/gonuget/version"
)

// Nuspec represents a parsed .nuspec manifest
type Nuspec struct {
    XMLName  xml.Name         `xml:"package"`
    Metadata NuspecMetadata   `xml:"metadata"`
    Files    []NuspecFile     `xml:"files>file"`
}

// NuspecMetadata represents the metadata section
type NuspecMetadata struct {
    // Required fields
    ID          string               `xml:"id"`
    Version     string               `xml:"version"`
    Description string               `xml:"description"`
    Authors     string               `xml:"authors"`

    // Optional fields
    Title                    string                      `xml:"title"`
    Owners                   string                      `xml:"owners"`
    ProjectURL              string                      `xml:"projectUrl"`
    IconURL                 string                      `xml:"iconUrl"`
    Icon                    string                      `xml:"icon"`
    LicenseURL              string                      `xml:"licenseUrl"`
    License                 *LicenseMetadata            `xml:"license"`
    RequireLicenseAcceptance bool                        `xml:"requireLicenseAcceptance"`
    DevelopmentDependency    bool                        `xml:"developmentDependency"`
    Summary                 string                      `xml:"summary"`
    ReleaseNotes            string                      `xml:"releaseNotes"`
    Copyright               string                      `xml:"copyright"`
    Language                string                      `xml:"language"`
    Tags                    string                      `xml:"tags"`
    Serviceable             bool                        `xml:"serviceable"`
    Readme                  string                      `xml:"readme"`

    // Version constraints
    MinClientVersion string `xml:"minClientVersion,attr"`

    // Complex elements
    Dependencies         *DependenciesElement         `xml:"dependencies"`
    FrameworkReferences  *FrameworkReferencesElement  `xml:"frameworkReferences"`
    FrameworkAssemblies  []FrameworkAssembly          `xml:"frameworkAssemblies>frameworkAssembly"`
    References           *ReferencesElement           `xml:"references"`
    ContentFiles         []ContentFilesEntry          `xml:"contentFiles>files"`
    PackageTypes         []PackageType                `xml:"packageTypes>packageType"`
    Repository           *RepositoryMetadata          `xml:"repository"`
}

// LicenseMetadata represents license information
type LicenseMetadata struct {
    Type    string `xml:"type,attr"`    // "expression" or "file"
    Version string `xml:"version,attr"` // SPDX license version
    Text    string `xml:",chardata"`    // License expression or file path
}

// DependenciesElement represents the dependencies container
type DependenciesElement struct {
    Groups []DependencyGroup `xml:"group"`
    // Legacy: dependencies without groups (applies to all frameworks)
    Dependencies []Dependency `xml:"dependency"`
}

// DependencyGroup represents dependencies for a specific framework
type DependencyGroup struct {
    TargetFramework string       `xml:"targetFramework,attr"`
    Dependencies    []Dependency `xml:"dependency"`
}

// Dependency represents a package dependency
type Dependency struct {
    ID      string `xml:"id,attr"`
    Version string `xml:"version,attr"` // Version range string
    Include string `xml:"include,attr"` // Asset include filter
    Exclude string `xml:"exclude,attr"` // Asset exclude filter
}

// FrameworkReferencesElement represents framework references container
type FrameworkReferencesElement struct {
    Groups []FrameworkReferenceGroup `xml:"group"`
}

// FrameworkReferenceGroup represents framework references for a TFM
type FrameworkReferenceGroup struct {
    TargetFramework string               `xml:"targetFramework,attr"`
    References      []FrameworkReference `xml:"frameworkReference"`
}

// FrameworkReference represents a reference to a framework assembly
type FrameworkReference struct {
    Name string `xml:"name,attr"`
}

// FrameworkAssembly represents a legacy framework assembly reference
type FrameworkAssembly struct {
    AssemblyName    string `xml:"assemblyName,attr"`
    TargetFramework string `xml:"targetFramework,attr"`
}

// ReferencesElement represents package assembly references
type ReferencesElement struct {
    Groups []ReferenceGroup `xml:"group"`
}

// ReferenceGroup represents references for a specific framework
type ReferenceGroup struct {
    TargetFramework string      `xml:"targetFramework,attr"`
    References      []Reference `xml:"reference"`
}

// Reference represents a reference to an assembly in the package
type Reference struct {
    File string `xml:"file,attr"`
}

// ContentFilesEntry represents content files metadata
type ContentFilesEntry struct {
    Include       string `xml:"include,attr"`
    Exclude       string `xml:"exclude,attr"`
    BuildAction   string `xml:"buildAction,attr"`
    CopyToOutput  string `xml:"copyToOutput,attr"`
    Flatten       string `xml:"flatten,attr"`
}

// PackageType represents the type of package
type PackageType struct {
    Name    string `xml:"name,attr"`
    Version string `xml:"version,attr"`
}

// RepositoryMetadata represents repository information
type RepositoryMetadata struct {
    Type   string `xml:"type,attr"`
    URL    string `xml:"url,attr"`
    Branch string `xml:"branch,attr"`
    Commit string `xml:"commit,attr"`
}

// NuspecFile represents a file entry in the nuspec
type NuspecFile struct {
    Source  string `xml:"src,attr"`
    Target  string `xml:"target,attr"`
    Exclude string `xml:"exclude,attr"`
}
```

**2. Nuspec Parser**:

```go
// ParseNuspec parses a .nuspec XML document
func ParseNuspec(r io.Reader) (*Nuspec, error) {
    decoder := xml.NewDecoder(r)

    var nuspec Nuspec
    if err := decoder.Decode(&nuspec); err != nil {
        return nil, fmt.Errorf("parse nuspec: %w", err)
    }

    return &nuspec, nil
}

// GetParsedIdentity returns the package identity from nuspec
func (n *Nuspec) GetParsedIdentity() (*PackageIdentity, error) {
    ver, err := version.Parse(n.Metadata.Version)
    if err != nil {
        return nil, fmt.Errorf("parse version: %w", err)
    }

    return &PackageIdentity{
        ID:      n.Metadata.ID,
        Version: ver,
    }, nil
}

// GetAuthors returns the list of authors
func (n *Nuspec) GetAuthors() []string {
    if n.Metadata.Authors == "" {
        return []string{}
    }

    // Authors are comma-separated
    authors := strings.Split(n.Metadata.Authors, ",")
    for i := range authors {
        authors[i] = strings.TrimSpace(authors[i])
    }

    return authors
}

// GetOwners returns the list of owners
func (n *Nuspec) GetOwners() []string {
    if n.Metadata.Owners == "" {
        return []string{}
    }

    owners := strings.Split(n.Metadata.Owners, ",")
    for i := range owners {
        owners[i] = strings.TrimSpace(owners[i])
    }

    return owners
}

// GetTags returns the list of tags
func (n *Nuspec) GetTags() []string {
    if n.Metadata.Tags == "" {
        return []string{}
    }

    // Tags are space-separated
    tags := strings.Fields(n.Metadata.Tags)
    return tags
}
```

**3. Dependency Parsing**:

```go
// GetDependencyGroups returns all dependency groups with parsed frameworks
func (n *Nuspec) GetDependencyGroups() ([]ParsedDependencyGroup, error) {
    if n.Metadata.Dependencies == nil {
        return []ParsedDependencyGroup{}, nil
    }

    var groups []ParsedDependencyGroup

    // Handle legacy dependencies (no groups)
    if len(n.Metadata.Dependencies.Dependencies) > 0 {
        // Dependencies without group apply to all frameworks
        anyFramework := frameworks.AnyFramework

        deps, err := parseDependencies(n.Metadata.Dependencies.Dependencies)
        if err != nil {
            return nil, err
        }

        groups = append(groups, ParsedDependencyGroup{
            TargetFramework: &anyFramework,
            Dependencies:    deps,
        })
    }

    // Handle grouped dependencies
    for _, group := range n.Metadata.Dependencies.Groups {
        var targetFramework *frameworks.NuGetFramework

        if group.TargetFramework != "" {
            fw, err := frameworks.ParseFramework(group.TargetFramework)
            if err != nil {
                return nil, fmt.Errorf("parse target framework %q: %w", group.TargetFramework, err)
            }
            targetFramework = fw
        } else {
            // Empty target framework means "any"
            anyFramework := frameworks.AnyFramework
            targetFramework = &anyFramework
        }

        deps, err := parseDependencies(group.Dependencies)
        if err != nil {
            return nil, err
        }

        groups = append(groups, ParsedDependencyGroup{
            TargetFramework: targetFramework,
            Dependencies:    deps,
        })
    }

    return groups, nil
}

// ParsedDependencyGroup represents a dependency group with parsed framework
type ParsedDependencyGroup struct {
    TargetFramework *frameworks.NuGetFramework
    Dependencies    []ParsedDependency
}

// ParsedDependency represents a dependency with parsed version range
type ParsedDependency struct {
    ID           string
    VersionRange *version.VersionRange
    Include      []string // Asset include patterns
    Exclude      []string // Asset exclude patterns
}

func parseDependencies(deps []Dependency) ([]ParsedDependency, error) {
    var parsed []ParsedDependency

    for _, dep := range deps {
        var versionRange *version.VersionRange

        if dep.Version != "" {
            vr, err := version.ParseVersionRange(dep.Version)
            if err != nil {
                return nil, fmt.Errorf("parse version range %q for %q: %w", dep.Version, dep.ID, err)
            }
            versionRange = vr
        }

        parsedDep := ParsedDependency{
            ID:           dep.ID,
            VersionRange: versionRange,
        }

        if dep.Include != "" {
            parsedDep.Include = strings.Split(dep.Include, ";")
        }

        if dep.Exclude != "" {
            parsedDep.Exclude = strings.Split(dep.Exclude, ";")
        }

        parsed = append(parsed, parsedDep)
    }

    return parsed, nil
}
```

**4. Framework Reference Parsing**:

```go
// GetFrameworkReferenceGroups returns all framework reference groups
func (n *Nuspec) GetFrameworkReferenceGroups() ([]ParsedFrameworkReferenceGroup, error) {
    if n.Metadata.FrameworkReferences == nil {
        return []ParsedFrameworkReferenceGroup{}, nil
    }

    var groups []ParsedFrameworkReferenceGroup

    for _, group := range n.Metadata.FrameworkReferences.Groups {
        fw, err := frameworks.ParseFramework(group.TargetFramework)
        if err != nil {
            return nil, fmt.Errorf("parse target framework %q: %w", group.TargetFramework, err)
        }

        var refs []string
        for _, ref := range group.References {
            refs = append(refs, ref.Name)
        }

        groups = append(groups, ParsedFrameworkReferenceGroup{
            TargetFramework: fw,
            References:      refs,
        })
    }

    return groups, nil
}

// ParsedFrameworkReferenceGroup represents framework references with parsed TFM
type ParsedFrameworkReferenceGroup struct {
    TargetFramework *frameworks.NuGetFramework
    References      []string
}
```

**5. Add to PackageReader**:

```go
// packaging/reader.go additions

// GetIdentity returns the package identity from the nuspec
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

// GetNuspec reads and parses the .nuspec file
func (r *PackageReader) GetNuspec() (*Nuspec, error) {
    nuspecReader, err := r.OpenNuspec()
    if err != nil {
        return nil, err
    }
    defer nuspecReader.Close()

    return ParseNuspec(nuspecReader)
}
```

### Verification Steps

```bash
# 1. Run nuspec parser tests
go test ./packaging -v -run TestNuspecParser

# 2. Test with various schema versions
go test ./packaging -v -run TestNuspecSchemaVersions

# 3. Test dependency parsing
go test ./packaging -v -run TestDependencyParsing

# 4. Validate with real nuspec files
go test ./packaging -v -run TestParseRealNuspec

# 5. Check test coverage
go test ./packaging -cover
```

### Acceptance Criteria

- [ ] Parse .nuspec XML with standard Go encoding/xml
- [ ] Extract all required metadata (ID, Version, Description, Authors)
- [ ] Extract optional metadata fields
- [ ] Parse dependency groups with target frameworks
- [ ] Parse legacy dependencies (no groups)
- [ ] Parse version ranges for dependencies
- [ ] Parse framework reference groups
- [ ] Parse framework assembly references
- [ ] Parse package types
- [ ] Parse repository metadata
- [ ] Handle missing optional fields gracefully
- [ ] Return structured ParsedDependencyGroup with NuGetFramework
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement nuspec XML parser

Add nuspec parsing with:
- XML schema support for all .nuspec versions
- Metadata extraction (ID, version, authors, etc.)
- Dependency group parsing with target frameworks
- Framework reference parsing
- Version range parsing for dependencies
- Legacy dependency format support

Reference: NuGet.Packaging/NuspecReader.cs
```

---

## M3.3: Package Reader - File Access

**Estimated Time**: 2 hours
**Dependencies**: M3.1, M3.2

### Overview

Implement high-level file access methods for common package operations including lib/ folder access, content file enumeration, and efficient file extraction.

### Files to Create/Modify

- `packaging/reader.go` - Add file access methods
- `packaging/files.go` - File filtering and enumeration utilities
- `packaging/reader_test.go` - File access tests

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging/PackageReaderBase.cs` (GetLibItems, GetContentItems, GetFiles)
- `NuGet.Packaging/PackageHelper.cs` (file type detection)

### Implementation Details

**1. File Type Detection**:

```go
// packaging/files.go

package packaging

import (
    "path"
    "strings"
)

// Common package folder constants
const (
    LibFolder              = "lib/"
    RefFolder              = "ref/"
    RuntimesFolder         = "runtimes/"
    ContentFolder          = "content/"
    ContentFilesFolder     = "contentFiles/"
    BuildFolder            = "build/"
    BuildTransitiveFolder  = "buildTransitive/"
    ToolsFolder            = "tools/"
    NativeFolder           = "native/"
    AnalyzersFolder        = "analyzers/"
    EmbedFolder            = "embed/"
)

// Package metadata files
const (
    ManifestExtension       = ".nuspec"
    SignatureFile          = ".signature.p7s"
    PackageRelationshipFile = "_rels/.rels"
    ContentTypesFile       = "[Content_Types].xml"
    PSMDCPFile            = "package/services/metadata/core-properties/"
)

// IsLibFile checks if a file is in the lib/ folder
func IsLibFile(filePath string) bool {
    return strings.HasPrefix(strings.ToLower(filePath), LibFolder)
}

// IsRefFile checks if a file is in the ref/ folder
func IsRefFile(filePath string) bool {
    return strings.HasPrefix(strings.ToLower(filePath), RefFolder)
}

// IsContentFile checks if a file is in the content/ folder
func IsContentFile(filePath string) bool {
    lower := strings.ToLower(filePath)
    return strings.HasPrefix(lower, ContentFolder) || strings.HasPrefix(lower, ContentFilesFolder)
}

// IsBuildFile checks if a file is in the build/ folder
func IsBuildFile(filePath string) bool {
    lower := strings.ToLower(filePath)
    return strings.HasPrefix(lower, BuildFolder) || strings.HasPrefix(lower, BuildTransitiveFolder)
}

// IsToolsFile checks if a file is in the tools/ folder
func IsToolsFile(filePath string) bool {
    return strings.HasPrefix(strings.ToLower(filePath), ToolsFolder)
}

// IsRuntimesFile checks if a file is in the runtimes/ folder
func IsRuntimesFile(filePath string) bool {
    return strings.HasPrefix(strings.ToLower(filePath), RuntimesFolder)
}

// IsAnalyzerFile checks if a file is in the analyzers/ folder
func IsAnalyzerFile(filePath string) bool {
    return strings.HasPrefix(strings.ToLower(filePath), AnalyzersFolder)
}

// IsPackageMetadataFile checks if a file is package metadata
func IsPackageMetadataFile(filePath string) bool {
    lower := strings.ToLower(filePath)
    return lower == SignatureFile ||
           strings.HasPrefix(lower, "_rels/") ||
           lower == ContentTypesFile ||
           strings.HasPrefix(lower, PSMDCPFile) ||
           IsManifestFile(filePath)
}

// GetFileExtension returns the file extension (lowercase, with dot)
func GetFileExtension(filePath string) string {
    ext := path.Ext(filePath)
    return strings.ToLower(ext)
}

// IsDllOrExe checks if file is a .dll or .exe
func IsDllOrExe(filePath string) bool {
    ext := GetFileExtension(filePath)
    return ext == ".dll" || ext == ".exe"
}

// IsAssembly checks if file is a managed assembly
func IsAssembly(filePath string) bool {
    ext := GetFileExtension(filePath)
    switch ext {
    case ".dll", ".exe", ".winmd":
        return true
    default:
        return false
    }
}
```

**2. File Enumeration**:

```go
// packaging/reader.go additions

// GetPackageFiles returns all files in the package excluding metadata
func (r *PackageReader) GetPackageFiles() []string {
    var files []string

    for _, file := range r.Files() {
        // Skip directories
        if strings.HasSuffix(file.Name, "/") {
            continue
        }

        // Skip package metadata files
        if IsPackageMetadataFile(file.Name) {
            continue
        }

        files = append(files, file.Name)
    }

    return files
}

// GetLibFiles returns all files in lib/ folder
func (r *PackageReader) GetLibFiles() []string {
    var files []string

    for _, file := range r.Files() {
        if IsLibFile(file.Name) && !strings.HasSuffix(file.Name, "/") {
            files = append(files, file.Name)
        }
    }

    return files
}

// GetRefFiles returns all files in ref/ folder
func (r *PackageReader) GetRefFiles() []string {
    var files []string

    for _, file := range r.Files() {
        if IsRefFile(file.Name) && !strings.HasSuffix(file.Name, "/") {
            files = append(files, file.Name)
        }
    }

    return files
}

// GetContentFiles returns all files in content/ folder
func (r *PackageReader) GetContentFiles() []string {
    var files []string

    for _, file := range r.Files() {
        if IsContentFile(file.Name) && !strings.HasSuffix(file.Name, "/") {
            files = append(files, file.Name)
        }
    }

    return files
}

// GetBuildFiles returns all files in build/ folder
func (r *PackageReader) GetBuildFiles() []string {
    var files []string

    for _, file := range r.Files() {
        if IsBuildFile(file.Name) && !strings.HasSuffix(file.Name, "/") {
            files = append(files, file.Name)
        }
    }

    return files
}

// GetToolsFiles returns all files in tools/ folder
func (r *PackageReader) GetToolsFiles() []string {
    var files []string

    for _, file := range r.Files() {
        if IsToolsFile(file.Name) && !strings.HasSuffix(file.Name, "/") {
            files = append(files, file.Name)
        }
    }

    return files
}
```

**3. File Extraction**:

```go
import (
    "io"
    "os"
    "path/filepath"
)

// ExtractFile extracts a single file from the package to the specified path
func (r *PackageReader) ExtractFile(zipPath, destPath string) error {
    zipFile, err := r.GetFile(zipPath)
    if err != nil {
        return err
    }

    // Validate destination path
    if err := ValidatePackagePath(destPath); err != nil {
        return fmt.Errorf("invalid destination path: %w", err)
    }

    // Create parent directories
    if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
        return fmt.Errorf("create parent directory: %w", err)
    }

    // Open source file
    srcReader, err := zipFile.Open()
    if err != nil {
        return fmt.Errorf("open zip file: %w", err)
    }
    defer srcReader.Close()

    // Create destination file
    destFile, err := os.Create(destPath)
    if err != nil {
        return fmt.Errorf("create destination file: %w", err)
    }
    defer destFile.Close()

    // Copy contents
    if _, err := io.Copy(destFile, srcReader); err != nil {
        return fmt.Errorf("copy file contents: %w", err)
    }

    return nil
}

// ExtractFiles extracts multiple files to a destination directory
func (r *PackageReader) ExtractFiles(files []string, destDir string) error {
    for _, file := range files {
        destPath := filepath.Join(destDir, file)
        if err := r.ExtractFile(file, destPath); err != nil {
            return fmt.Errorf("extract %s: %w", file, err)
        }
    }

    return nil
}

// CopyFileTo copies a file from the package to a writer
func (r *PackageReader) CopyFileTo(zipPath string, writer io.Writer) error {
    zipFile, err := r.GetFile(zipPath)
    if err != nil {
        return err
    }

    srcReader, err := zipFile.Open()
    if err != nil {
        return fmt.Errorf("open zip file: %w", err)
    }
    defer srcReader.Close()

    if _, err := io.Copy(writer, srcReader); err != nil {
        return fmt.Errorf("copy contents: %w", err)
    }

    return nil
}
```

**4. Framework-Specific File Filtering**:

```go
// GetLibFilesForFramework returns lib files for a specific framework
// This is a simple implementation; M3.11-M3.12 will implement full asset selection
func (r *PackageReader) GetLibFilesForFramework(targetFramework string) []string {
    var matches []string

    libFiles := r.GetLibFiles()
    prefix := LibFolder + targetFramework + "/"

    for _, file := range libFiles {
        if strings.HasPrefix(strings.ToLower(file), strings.ToLower(prefix)) {
            matches = append(matches, file)
        }
    }

    return matches
}
```

### Verification Steps

```bash
# 1. Run file access tests
go test ./packaging -v -run TestFileAccess

# 2. Test file type detection
go test ./packaging -v -run TestFileTypeDetection

# 3. Test file extraction
go test ./packaging -v -run TestExtractFile

# 4. Verify with real packages
go test ./packaging -v -run TestExtractRealPackage

# 5. Check test coverage
go test ./packaging -cover
```

### Acceptance Criteria

- [ ] Enumerate all package files excluding metadata
- [ ] Get files by folder (lib/, ref/, content/, build/, tools/)
- [ ] Detect file types correctly (assembly, metadata, etc.)
- [ ] Extract single file to destination path
- [ ] Extract multiple files to directory
- [ ] Copy file contents to writer
- [ ] Validate extraction paths for security
- [ ] Create parent directories during extraction
- [ ] Handle file permissions correctly
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): add file access and extraction methods

Add package file operations:
- File enumeration by folder (lib/, ref/, content/, etc.)
- File type detection (assembly, metadata, etc.)
- Single and batch file extraction
- Stream-based file copying
- Path validation for security

Reference: NuGet.Packaging/PackageReaderBase.cs
```

---

## M3.4: Package Builder - Core API

**Estimated Time**: 3 hours
**Dependencies**: M1, M2

### Overview

Implement the core package builder API for creating .nupkg files with fluent configuration, file addition tracking, and basic validation.

### Files to Create/Modify

- `packaging/builder.go` - Core PackageBuilder implementation
- `packaging/builder_test.go` - Builder tests
- `packaging/package.go` - Package metadata structures

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging/PackageBuilder.cs` (1100+ lines)
- `NuGet.Packaging/IPackageMetadata.cs` - Metadata interface

**Builder Pattern** (PackageBuilder.cs:150-200):
```csharp
public class PackageBuilder : IPackageBuilder {
    public PackageBuilder(string path, string basePath) {
        using (Stream stream = File.OpenRead(path)) {
            ReadManifest(stream, basePath, path);
        }
    }

    public void Save(Stream stream) {
        // Validation
        PackageIdValidator.ValidatePackageId(Id);
        ValidateDependencies(Version, DependencyGroups);
        ValidateFilesUnique(Files);
        // ... more validations

        // Create ZIP
        using (var package = new ZipArchive(stream, ZipArchiveMode.Create, leaveOpen: true)) {
            WriteManifest(package, DetermineMinimumSchemaVersion(Files, DependencyGroups), psmdcpPath);
            WriteFiles(package);
            WriteOpcContentTypes(package, extensions, filesWithoutExtensions);
            WriteOpcPackageProperties(package, psmdcpPath);
        }
    }
}
```

### Implementation Details

**1. Package Metadata Structures**:

```go
// packaging/package.go

package packaging

import (
    "net/url"
    "time"

    "github.com/willibrandon/gonuget/frameworks"
    "github.com/willibrandon/gonuget/version"
)

// PackageMetadata represents package metadata for building
type PackageMetadata struct {
    // Required
    ID          string
    Version     *version.NuGetVersion
    Description string
    Authors     []string

    // Optional
    Title                    string
    Owners                   []string
    ProjectURL              *url.URL
    IconURL                 *url.URL
    Icon                    string
    LicenseURL              *url.URL
    LicenseMetadata         *LicenseMetadata
    RequireLicenseAcceptance bool
    DevelopmentDependency    bool
    Summary                 string
    ReleaseNotes            string
    Copyright               string
    Language                string
    Tags                    []string
    Serviceable             bool
    Readme                  string

    // Version constraints
    MinClientVersion *version.NuGetVersion

    // Complex elements
    DependencyGroups         []PackageDependencyGroup
    FrameworkReferenceGroups []PackageFrameworkReferenceGroup
    FrameworkAssemblies      []PackageFrameworkAssembly
    PackageTypes             []PackageTypeInfo
    Repository               *PackageRepositoryMetadata
}

// PackageDependencyGroup represents dependencies for a target framework
type PackageDependencyGroup struct {
    TargetFramework *frameworks.NuGetFramework
    Dependencies    []PackageDependency
}

// PackageDependency represents a single package dependency
type PackageDependency struct {
    ID           string
    VersionRange *version.VersionRange
    Include      []string // Asset include filters
    Exclude      []string // Asset exclude filters
}

// PackageFrameworkReferenceGroup represents framework references for a TFM
type PackageFrameworkReferenceGroup struct {
    TargetFramework *frameworks.NuGetFramework
    References      []string
}

// PackageFrameworkAssembly represents a framework assembly reference
type PackageFrameworkAssembly struct {
    AssemblyName     string
    TargetFrameworks []*frameworks.NuGetFramework
}

// PackageTypeInfo represents a package type
type PackageTypeInfo struct {
    Name    string
    Version *version.NuGetVersion
}

// PackageRepositoryMetadata represents repository metadata
type PackageRepositoryMetadata struct {
    Type   string
    URL    string
    Branch string
    Commit string
}

// LicenseMetadata represents license information
type LicenseMetadata struct {
    Type       string // "expression" or "file"
    License    string // SPDX expression or file path
    Version    string // SPDX license version
    Expression string // Parsed SPDX expression (if type is "expression")
}
```

**2. Package Builder Structure**:

```go
// packaging/builder.go

package packaging

import (
    "archive/zip"
    "fmt"
    "io"
    "path/filepath"
    "strings"
    "time"
)

// PackageBuilder builds .nupkg files
type PackageBuilder struct {
    metadata PackageMetadata
    files    []PackageFile

    // Internal tracking
    filePaths    map[string]bool // For duplicate detection
    createdTime  time.Time
}

// PackageFile represents a file to be added to the package
type PackageFile struct {
    SourcePath string      // Path on disk (or empty for in-memory)
    TargetPath string      // Path in .nupkg
    Content    []byte      // In-memory content (if SourcePath is empty)
    Reader     io.Reader   // Stream content (if SourcePath is empty and Content is nil)
}

// NewPackageBuilder creates a new package builder
func NewPackageBuilder() *PackageBuilder {
    return &PackageBuilder{
        filePaths:   make(map[string]bool),
        createdTime: time.Now().UTC(),
    }
}

// NewPackageBuilderFromNuspec creates a builder from a nuspec file
func NewPackageBuilderFromNuspec(nuspecPath string) (*PackageBuilder, error) {
    nuspec, err := ParseNuspecFile(nuspecPath)
    if err != nil {
        return nil, fmt.Errorf("parse nuspec: %w", err)
    }

    builder := NewPackageBuilder()
    if err := builder.PopulateFromNuspec(nuspec); err != nil {
        return nil, err
    }

    return builder, nil
}

// ParseNuspecFile is a helper to parse nuspec from file path
func ParseNuspecFile(path string) (*Nuspec, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    return ParseNuspec(file)
}
```

**3. Metadata Configuration (Fluent API)**:

```go
// SetID sets the package ID
func (b *PackageBuilder) SetID(id string) *PackageBuilder {
    b.metadata.ID = id
    return b
}

// SetVersion sets the package version
func (b *PackageBuilder) SetVersion(ver *version.NuGetVersion) *PackageBuilder {
    b.metadata.Version = ver
    return b
}

// SetDescription sets the package description
func (b *PackageBuilder) SetDescription(description string) *PackageBuilder {
    b.metadata.Description = description
    return b
}

// SetAuthors sets the package authors
func (b *PackageBuilder) SetAuthors(authors ...string) *PackageBuilder {
    b.metadata.Authors = authors
    return b
}

// SetTitle sets the package title
func (b *PackageBuilder) SetTitle(title string) *PackageBuilder {
    b.metadata.Title = title
    return b
}

// SetOwners sets the package owners
func (b *PackageBuilder) SetOwners(owners ...string) *PackageBuilder {
    b.metadata.Owners = owners
    return b
}

// SetProjectURL sets the project URL
func (b *PackageBuilder) SetProjectURL(urlStr string) error {
    u, err := url.Parse(urlStr)
    if err != nil {
        return fmt.Errorf("invalid project URL: %w", err)
    }
    b.metadata.ProjectURL = u
    return nil
}

// SetLicenseURL sets the license URL
func (b *PackageBuilder) SetLicenseURL(urlStr string) error {
    u, err := url.Parse(urlStr)
    if err != nil {
        return fmt.Errorf("invalid license URL: %w", err)
    }
    b.metadata.LicenseURL = u
    return nil
}

// SetTags sets the package tags
func (b *PackageBuilder) SetTags(tags ...string) *PackageBuilder {
    b.metadata.Tags = tags
    return b
}

// AddDependencyGroup adds a dependency group
func (b *PackageBuilder) AddDependencyGroup(group PackageDependencyGroup) *PackageBuilder {
    b.metadata.DependencyGroups = append(b.metadata.DependencyGroups, group)
    return b
}

// AddDependency adds a dependency to a specific framework (or nil for "any")
func (b *PackageBuilder) AddDependency(fw *frameworks.NuGetFramework, id string, versionRange *version.VersionRange) *PackageBuilder {
    // Find existing group for this framework
    for i := range b.metadata.DependencyGroups {
        if frameworksEqual(b.metadata.DependencyGroups[i].TargetFramework, fw) {
            b.metadata.DependencyGroups[i].Dependencies = append(
                b.metadata.DependencyGroups[i].Dependencies,
                PackageDependency{ID: id, VersionRange: versionRange},
            )
            return b
        }
    }

    // Create new group
    b.metadata.DependencyGroups = append(b.metadata.DependencyGroups, PackageDependencyGroup{
        TargetFramework: fw,
        Dependencies:    []PackageDependency{{ID: id, VersionRange: versionRange}},
    })

    return b
}

func frameworksEqual(a, b *frameworks.NuGetFramework) bool {
    if a == nil && b == nil {
        return true
    }
    if a == nil || b == nil {
        return false
    }
    return a.Framework == b.Framework && a.Version == b.Version
}
```

**4. File Addition**:

```go
// AddFile adds a file from disk to the package
func (b *PackageBuilder) AddFile(sourcePath, targetPath string) error {
    // Validate target path
    if err := ValidatePackagePath(targetPath); err != nil {
        return fmt.Errorf("invalid target path: %w", err)
    }

    // Normalize target path
    normalizedTarget := normalizePackagePath(targetPath)

    // Check for duplicates
    if b.filePaths[normalizedTarget] {
        return fmt.Errorf("duplicate file path: %s", targetPath)
    }

    b.files = append(b.files, PackageFile{
        SourcePath: sourcePath,
        TargetPath: normalizedTarget,
    })

    b.filePaths[normalizedTarget] = true
    return nil
}

// AddFileFromBytes adds a file from in-memory bytes
func (b *PackageBuilder) AddFileFromBytes(targetPath string, content []byte) error {
    if err := ValidatePackagePath(targetPath); err != nil {
        return fmt.Errorf("invalid target path: %w", err)
    }

    normalizedTarget := normalizePackagePath(targetPath)

    if b.filePaths[normalizedTarget] {
        return fmt.Errorf("duplicate file path: %s", targetPath)
    }

    b.files = append(b.files, PackageFile{
        TargetPath: normalizedTarget,
        Content:    content,
    })

    b.filePaths[normalizedTarget] = true
    return nil
}

// AddFileFromReader adds a file from a reader
func (b *PackageBuilder) AddFileFromReader(targetPath string, reader io.Reader) error {
    if err := ValidatePackagePath(targetPath); err != nil {
        return fmt.Errorf("invalid target path: %w", err)
    }

    normalizedTarget := normalizePackagePath(targetPath)

    if b.filePaths[normalizedTarget] {
        return fmt.Errorf("duplicate file path: %s", targetPath)
    }

    b.files = append(b.files, PackageFile{
        TargetPath: normalizedTarget,
        Reader:     reader,
    })

    b.filePaths[normalizedTarget] = true
    return nil
}

func normalizePackagePath(path string) string {
    // Replace backslashes with forward slashes
    normalized := strings.ReplaceAll(path, "\\", "/")
    // Ensure no leading slash
    normalized = strings.TrimPrefix(normalized, "/")
    return normalized
}
```

**5. Populate from Nuspec**:

```go
// PopulateFromNuspec populates builder metadata from parsed nuspec
func (b *PackageBuilder) PopulateFromNuspec(nuspec *Nuspec) error {
    // Parse version
    ver, err := version.Parse(nuspec.Metadata.Version)
    if err != nil {
        return fmt.Errorf("parse version: %w", err)
    }

    b.metadata.ID = nuspec.Metadata.ID
    b.metadata.Version = ver
    b.metadata.Description = nuspec.Metadata.Description
    b.metadata.Authors = nuspec.GetAuthors()
    b.metadata.Title = nuspec.Metadata.Title
    b.metadata.Owners = nuspec.GetOwners()
    b.metadata.Summary = nuspec.Metadata.Summary
    b.metadata.ReleaseNotes = nuspec.Metadata.ReleaseNotes
    b.metadata.Copyright = nuspec.Metadata.Copyright
    b.metadata.Language = nuspec.Metadata.Language
    b.metadata.Tags = nuspec.GetTags()
    b.metadata.RequireLicenseAcceptance = nuspec.Metadata.RequireLicenseAcceptance
    b.metadata.DevelopmentDependency = nuspec.Metadata.DevelopmentDependency
    b.metadata.Serviceable = nuspec.Metadata.Serviceable
    b.metadata.Icon = nuspec.Metadata.Icon
    b.metadata.Readme = nuspec.Metadata.Readme

    // Parse URLs
    if nuspec.Metadata.ProjectURL != "" {
        if err := b.SetProjectURL(nuspec.Metadata.ProjectURL); err != nil {
            return err
        }
    }

    if nuspec.Metadata.LicenseURL != "" {
        if err := b.SetLicenseURL(nuspec.Metadata.LicenseURL); err != nil {
            return err
        }
    }

    if nuspec.Metadata.IconURL != "" {
        u, err := url.Parse(nuspec.Metadata.IconURL)
        if err != nil {
            return fmt.Errorf("parse icon URL: %w", err)
        }
        b.metadata.IconURL = u
    }

    // Parse dependency groups
    depGroups, err := nuspec.GetDependencyGroups()
    if err != nil {
        return fmt.Errorf("parse dependency groups: %w", err)
    }

    for _, group := range depGroups {
        pkgGroup := PackageDependencyGroup{
            TargetFramework: group.TargetFramework,
        }

        for _, dep := range group.Dependencies {
            pkgGroup.Dependencies = append(pkgGroup.Dependencies, PackageDependency{
                ID:           dep.ID,
                VersionRange: dep.VersionRange,
                Include:      dep.Include,
                Exclude:      dep.Exclude,
            })
        }

        b.metadata.DependencyGroups = append(b.metadata.DependencyGroups, pkgGroup)
    }

    // Parse framework references
    fwRefGroups, err := nuspec.GetFrameworkReferenceGroups()
    if err != nil {
        return fmt.Errorf("parse framework references: %w", err)
    }

    for _, group := range fwRefGroups {
        b.metadata.FrameworkReferenceGroups = append(b.metadata.FrameworkReferenceGroups, PackageFrameworkReferenceGroup{
            TargetFramework: group.TargetFramework,
            References:      group.References,
        })
    }

    return nil
}

// GetMetadata returns the current package metadata
func (b *PackageBuilder) GetMetadata() PackageMetadata {
    return b.metadata
}

// GetFiles returns the current file list
func (b *PackageBuilder) GetFiles() []PackageFile {
    return b.files
}
```

### Verification Steps

```bash
# 1. Run builder tests
go test ./packaging -v -run TestPackageBuilder

# 2. Test fluent API
go test ./packaging -v -run TestBuilderFluentAPI

# 3. Test nuspec population
go test ./packaging -v -run TestPopulateFromNuspec

# 4. Test file addition
go test ./packaging -v -run TestBuilderAddFile

# 5. Check test coverage
go test ./packaging -cover
```

### Acceptance Criteria

- [ ] Create builder with NewPackageBuilder
- [ ] Create builder from nuspec file
- [ ] Fluent API for metadata configuration
- [ ] Add files from disk with source/target paths
- [ ] Add files from in-memory bytes
- [ ] Add files from io.Reader
- [ ] Detect duplicate file paths
- [ ] Normalize file paths (forward slashes)
- [ ] Validate file paths for security
- [ ] Populate metadata from parsed nuspec
- [ ] Parse all metadata fields correctly
- [ ] Parse dependency groups with frameworks
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement package builder core API

Add PackageBuilder with:
- Fluent API for metadata configuration
- File addition from disk, bytes, and readers
- Nuspec-based initialization
- Duplicate file detection
- Path validation and normalization
- Dependency group management

Reference: NuGet.Packaging/PackageBuilder.cs
```

---

## Summary - Chunks 1-4 Complete

**Total Time for This File**: 10 hours
**Files Created**: 8
**Lines of Code**: ~1,400

**Next File**: IMPL-M3-PACKAGING-CONTINUED.md (Chunks 5-7: OPC Compliance & Validation)

**Dependencies for Next Chunks**:
- M3.5 requires M3.4 (builder core)
- M3.6 requires M3.5 (OPC compliance)
- M3.7 requires M3.4, M3.5, M3.6 (validation of complete package)
