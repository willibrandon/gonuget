# gonuget - Product Requirements Document: Package Operations

**Version:** 1.0
**Status:** Draft
**Last Updated:** 2025-10-19
**Owner:** Engineering

---

## Table of Contents

1. [Overview](#overview)
2. [Package Reading](#package-reading)
3. [Package Creation](#package-creation)
4. [Package Validation](#package-validation)
5. [Package Signing](#package-signing)
6. [Nuspec Handling](#nuspec-handling)
7. [Asset Selection](#asset-selection)
8. [Acceptance Criteria](#acceptance-criteria)

---

## Overview

This document specifies requirements for package operations including reading .nupkg files, creating packages, validation, and signature verification.

**Package Format:** .nupkg files are ZIP archives following Open Packaging Conventions (OPC).

**Related Design Documents:**
- DESIGN-PACKAGING.md - Package operations design

---

## Package Reading

### Requirement PR-001: Open Package

**Priority:** P0 (Critical)
**Component:** `packaging` package

**Description:**
Open and read .nupkg files (ZIP archives).

**Functional Requirements:**

1. **Open from file:**
   - Read .nupkg as ZIP archive
   - Validate ZIP format
   - Load file index

2. **Open from stream:**
   - Support io.Reader input
   - Handle streaming reads

3. **File access:**
   - List all files in package
   - Read specific files
   - Extract files to disk

**API:**
```go
type PackageReader struct {
    files map[string]*zip.File
    nuspec *Nuspec
    // ...
}

func OpenPackage(path string) (*PackageReader, error)
func OpenPackageFromReader(r io.ReaderAt, size int64) (*PackageReader, error)

func (pr *PackageReader) GetNuspec() (*Nuspec, error)
func (pr *PackageReader) GetFiles() []string
func (pr *PackageReader) ReadFile(path string) ([]byte, error)
func (pr *PackageReader) ExtractFile(path string, dest string) error
func (pr *PackageReader) ExtractAll(dest string) error
func (pr *PackageReader) Close() error
```

**Error Handling:**
- Invalid ZIP format
- Missing .nuspec file
- Corrupted archive
- File not found

**Performance Requirements:**
- Open package: <50ms
- List files: <10ms
- Read small file (<1MB): <10ms

**Acceptance Criteria:**
- ✅ Opens valid .nupkg files
- ✅ Reads file contents
- ✅ Extracts files correctly
- ✅ Error handling complete

---

### Requirement PR-002: Nuspec Parsing

**Priority:** P0 (Critical)
**Component:** `packaging` package

**Description:**
Parse .nuspec manifest files (XML format).

**Nuspec Format:**
```xml
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>Newtonsoft.Json</id>
    <version>13.0.1</version>
    <authors>James Newton-King</authors>
    <description>Json.NET is a popular high-performance JSON framework for .NET</description>
    <dependencies>
      <group targetFramework="netstandard2.0">
        <dependency id="System.Text.Json" version="5.0.0" />
      </group>
    </dependencies>
  </metadata>
  <files>
    <file src="lib\netstandard2.0\Newtonsoft.Json.dll" target="lib\netstandard2.0\" />
  </files>
</package>
```

**Functional Requirements:**

1. **Parse metadata:**
   - ID, version, authors, description
   - License, icon, project URLs
   - Tags, release notes
   - Copyright, language

2. **Parse dependencies:**
   - Dependency groups by framework
   - Dependency ID and version range
   - Handle no dependencies

3. **Parse files section:**
   - Source and target paths
   - Exclude patterns

4. **Schema versions:**
   - Support multiple nuspec schema versions
   - Handle namespace variations

**API:**
```go
type Nuspec struct {
    Metadata *NuspecMetadata
    Files []*NuspecFile
}

type NuspecMetadata struct {
    ID string
    Version string
    Authors string // Can be comma-separated
    Owners string
    Description string
    Summary string
    LicenseURL string
    ProjectURL string
    IconURL string
    Icon string // Path to icon file in package
    Tags string
    ReleaseNotes string
    Copyright string
    Language string
    DependencyGroups []*DependencyGroup
    FrameworkAssemblies []*FrameworkAssembly
    References []*Reference
    ContentFiles []*ContentFile
}

type NuspecFile struct {
    Src string
    Target string
    Exclude string
}

func ParseNuspec(r io.Reader) (*Nuspec, error)
```

**Edge Cases:**
- Empty/missing fields
- Invalid XML
- Unknown elements (ignore gracefully)
- Multiple schema namespaces

**Acceptance Criteria:**
- ✅ Parses all standard nuspec fields
- ✅ Handles dependency groups
- ✅ Supports multiple schema versions
- ✅ Ignores unknown elements

---

### Requirement PR-003: Package Content Validation

**Priority:** P0 (Critical)
**Component:** `packaging` package

**Description:**
Validate package structure and contents.

**Validation Checks:**

1. **Required files:**
   - `.nuspec` exists at package root
   - Only one `.nuspec` file
   - `[Content_Types].xml` present (OPC requirement)

2. **Nuspec validation:**
   - Valid XML
   - Required fields present (id, version, description, authors)
   - Version format valid
   - Dependencies have valid version ranges

3. **Package structure:**
   - Valid folder structure (lib/, ref/, build/, etc.)
   - No invalid characters in paths
   - No duplicate files

4. **Content validation:**
   - ZIP not corrupted
   - File sizes match
   - No malicious paths (e.g., `../../etc/passwd`)

**API:**
```go
type ValidationResult struct {
    Valid bool
    Errors []ValidationError
    Warnings []ValidationWarning
}

type ValidationError struct {
    Code string
    Message string
    Location string
}

func (pr *PackageReader) Validate() (*ValidationResult, error)
```

**Validation Levels:**
- Strict: All errors and warnings fail
- Normal: Only errors fail
- Minimal: Only critical errors fail

**Acceptance Criteria:**
- ✅ Detects missing required files
- ✅ Validates nuspec contents
- ✅ Detects structural issues
- ✅ Security validation (path traversal, etc.)

---

## Package Creation

### Requirement PC-001: Package Builder

**Priority:** P1 (High)
**Component:** `packaging` package

**Description:**
Create .nupkg files from source files and metadata.

**Functional Requirements:**

1. **Builder API:**
   - Fluent API for construction
   - Add files from disk
   - Add files from memory
   - Set metadata programmatically

2. **Build process:**
   - Generate .nuspec
   - Create ZIP archive
   - Add `[Content_Types].xml` (OPC)
   - Add `.rels` files (OPC)

3. **Validation:**
   - Validate before building
   - Check required metadata
   - Verify file paths

**API:**
```go
type PackageBuilder struct {
    metadata *NuspecMetadata
    files map[string][]byte
}

func NewPackageBuilder(id string, version string) *PackageBuilder

func (pb *PackageBuilder) SetAuthors(authors ...string) *PackageBuilder
func (pb *PackageBuilder) SetDescription(desc string) *PackageBuilder
func (pb *PackageBuilder) AddFile(src string, target string) error
func (pb *PackageBuilder) AddFileFromBytes(name string, content []byte, target string) error
func (pb *PackageBuilder) AddDependency(framework string, id string, versionRange string) error

func (pb *PackageBuilder) Build(output string) error
func (pb *PackageBuilder) BuildToWriter(w io.Writer) error
```

**Example Usage:**
```go
builder := NewPackageBuilder("MyPackage", "1.0.0").
    SetAuthors("John Doe").
    SetDescription("My awesome package")

builder.AddFile("bin/Release/MyLib.dll", "lib/net8.0/")
builder.AddDependency("net8.0", "Newtonsoft.Json", "[13.0.1, )")

err := builder.Build("MyPackage.1.0.0.nupkg")
```

**Acceptance Criteria:**
- ✅ Creates valid .nupkg files
- ✅ Generates proper .nuspec
- ✅ OPC structure correct
- ✅ Packages installable by NuGet clients

---

### Requirement PC-002: OPC Compliance

**Priority:** P1 (High)
**Component:** `packaging` package

**Description:**
Generate Open Packaging Conventions (OPC) required files.

**Required Files:**

1. **[Content_Types].xml:**
   ```xml
   <?xml version="1.0" encoding="utf-8"?>
   <Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
     <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml" />
     <Default Extension="nuspec" ContentType="application/octet" />
     <Default Extension="dll" ContentType="application/octet" />
     <Default Extension="xml" ContentType="application/octet" />
   </Types>
   ```

2. **_rels/.rels:**
   ```xml
   <?xml version="1.0" encoding="utf-8"?>
   <Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
     <Relationship Type="http://schemas.microsoft.com/packaging/2010/07/manifest"
                   Target="/MyPackage.nuspec" Id="..." />
   </Relationships>
   ```

**Functional Requirements:**

1. **Content types:**
   - Detect file extensions in package
   - Generate content type mappings
   - Use default mappings for common types

2. **Relationships:**
   - Create relationship to .nuspec
   - Generate unique IDs

**API:**
```go
func generateContentTypes(files []string) []byte
func generateRelsFile(nuspecPath string) []byte
```

**Acceptance Criteria:**
- ✅ Generates valid `[Content_Types].xml`
- ✅ Generates valid `_rels/.rels`
- ✅ OPC validation passes

---

## Package Validation

### Requirement PV-001: Semantic Validation

**Priority:** P0 (Critical)
**Component:** `packaging` package

**Description:**
Validate package semantics beyond structure.

**Validation Rules:**

1. **Version consistency:**
   - Nuspec version matches filename
   - Dependency versions valid

2. **Framework compatibility:**
   - Framework folders use valid TFMs
   - Dependencies target valid frameworks
   - No incompatible framework combinations

3. **Content validation:**
   - DLL files in lib/ folders
   - Build files in build/ folders
   - Content files properly marked

4. **Dependency resolution:**
   - Circular dependency detection
   - Version conflicts identification

**API:**
```go
type SemanticValidator struct {
    // ...
}

func NewSemanticValidator() *SemanticValidator
func (sv *SemanticValidator) Validate(pr *PackageReader) (*ValidationResult, error)
```

**Acceptance Criteria:**
- ✅ Detects semantic errors
- ✅ Framework validation works
- ✅ Dependency issues flagged

---

## Package Signing

### Requirement PS-001: Signature Verification

**Priority:** P0 (Critical)
**Component:** `packaging/signing` package

**Description:**
Verify package signatures (author and repository signatures).

**Signature Types:**

1. **Author signature:**
   - Created by package author
   - Signs package at creation time
   - Uses author's certificate

2. **Repository signature:**
   - Added by repository (e.g., nuget.org)
   - Countersigns author signature
   - Uses repository certificate

**Signature Location:**
- `.signature.p7s` file in package root

**Functional Requirements:**

1. **Detect signature:**
   - Check for `.signature.p7s` file
   - Determine signature type

2. **Verify signature:**
   - Parse PKCS#7 signature
   - Verify certificate chain
   - Check timestamp (RFC 3161)
   - Verify package hash

3. **Certificate validation:**
   - Check expiry
   - Check revocation (optional)
   - Validate trust chain

**API:**
```go
type SignatureInfo struct {
    Type SignatureType // Author or Repository
    Certificate *x509.Certificate
    Timestamp time.Time
    Valid bool
    ValidationErrors []error
}

func (pr *PackageReader) HasSignature() bool
func (pr *PackageReader) VerifySignature(opts *VerifyOptions) (*SignatureInfo, error)
```

**VerifyOptions:**
```go
type VerifyOptions struct {
    TrustedCertificates []*x509.Certificate
    AllowUntrustedRoot bool
    AllowExpiredCertificate bool
    CheckRevocation bool
}
```

**Acceptance Criteria:**
- ✅ Detects signed packages
- ✅ Verifies PKCS#7 signatures
- ✅ Validates certificate chains
- ✅ Checks timestamps

---

### Requirement PS-002: Signature Creation

**Priority:** P2 (Medium)
**Component:** `packaging/signing` package

**Description:**
Sign packages with author signature.

**Functional Requirements:**

1. **Load certificate:**
   - Read PFX/P12 file
   - Extract private key
   - Extract certificate chain

2. **Create signature:**
   - Hash package contents
   - Create PKCS#7 signature
   - Add RFC 3161 timestamp
   - Embed in .signature.p7s

3. **Add to package:**
   - Insert .signature.p7s
   - Update [Content_Types].xml
   - Update _rels/.rels

**API:**
```go
type SigningOptions struct {
    CertificatePath string
    CertificatePassword string
    TimestampURL string // RFC 3161 timestamp server
}

func SignPackage(packagePath string, opts *SigningOptions) error
```

**Acceptance Criteria:**
- ✅ Creates valid signatures
- ✅ Includes timestamp
- ✅ Verifiable by NuGet.exe

---

## Nuspec Handling

### Requirement NH-001: Nuspec Generation

**Priority:** P1 (High)
**Component:** `packaging` package

**Description:**
Generate .nuspec files from metadata.

**Functional Requirements:**

1. **Generate XML:**
   - Marshal metadata to XML
   - Use correct namespace
   - Proper formatting/indentation

2. **Minimal nuspec:**
   - Only required fields
   - Omit empty optionals

3. **Complete nuspec:**
   - All metadata fields
   - Dependencies
   - Files section

**API:**
```go
func (md *NuspecMetadata) ToXML() ([]byte, error)
func (md *NuspecMetadata) WriteToFile(path string) error
```

**Acceptance Criteria:**
- ✅ Generates valid nuspec XML
- ✅ Parseable by NuGet.exe
- ✅ Includes all metadata

---

### Requirement NH-002: Nuspec Transformation

**Priority:** P2 (Medium)
**Component:** `packaging` package

**Description:**
Transform nuspec files (token replacement, etc.).

**Transformations:**

1. **Token replacement:**
   - `$id$` → Package ID
   - `$version$` → Package version
   - `$author$` → Authors
   - Custom tokens

2. **Property substitution:**
   - Replace from properties file
   - Environment variables

**API:**
```go
type TokenReplacer struct {
    Tokens map[string]string
}

func (tr *TokenReplacer) Transform(nuspec *Nuspec) error
```

**Acceptance Criteria:**
- ✅ Replaces standard tokens
- ✅ Supports custom tokens
- ✅ Handles missing tokens

---

## Asset Selection

### Requirement AS-001: Framework-Based Selection

**Priority:** P0 (Critical)
**Component:** `packaging` package

**Description:**
Select appropriate assets for target framework.

**Asset Folders:**
- `lib/{framework}/` - Runtime assemblies
- `ref/{framework}/` - Reference assemblies
- `build/{framework}/` - MSBuild files
- `buildTransitive/{framework}/` - Transitive build files
- `runtimes/{rid}/lib/{framework}/` - Platform-specific

**Selection Algorithm:**

1. **Find compatible folders:**
   - List all lib/ folders
   - Parse framework from folder name
   - Check compatibility with target

2. **Select nearest:**
   - Choose most specific compatible framework
   - E.g., for net8.0 target: net8.0 > net6.0 > netstandard2.1 > netstandard2.0

3. **RID-specific:**
   - Consider runtime identifier
   - Select most specific RID
   - Fallback to generic if no RID match

**API:**
```go
type AssetGroup struct {
    TargetFramework *NuGetFramework
    RuntimeIdentifier *RuntimeIdentifier
    Items []*Asset
}

type Asset struct {
    Path string
    Type AssetType // Lib, Ref, Build, etc.
}

func (pr *PackageReader) GetAssets(target *NuGetFramework, rid *RuntimeIdentifier) ([]*AssetGroup, error)
func (pr *PackageReader) GetLibAssets(target *NuGetFramework) ([]*Asset, error)
func (pr *PackageReader) GetRefAssets(target *NuGetFramework) ([]*Asset, error)
```

**Acceptance Criteria:**
- ✅ Selects correct framework assets
- ✅ Nearest framework logic works
- ✅ RID selection accurate
- ✅ Matches C# NuGet.Client behavior

---

### Requirement AS-002: Content File Selection

**Priority:** P1 (High)
**Component:** `packaging` package

**Description:**
Select content files based on rules in nuspec.

**Content Files Section:**
```xml
<contentFiles>
  <files include="any/any/config.json" buildAction="None" copyToOutput="true" flatten="false" />
  <files include="cs/net6.0/*.cs" buildAction="Compile" />
</contentFiles>
```

**Selection Rules:**

1. **Include/exclude:**
   - Pattern matching
   - Framework filtering
   - Language filtering

2. **Build action:**
   - None, Compile, Content, EmbeddedResource

3. **CopyToOutput:**
   - true, false, PreserveNewest

**API:**
```go
type ContentFile struct {
    Include string
    Exclude string
    BuildAction string
    CopyToOutput string
    Flatten bool
}

func (pr *PackageReader) GetContentFiles(target *NuGetFramework) ([]*ContentFile, error)
```

**Acceptance Criteria:**
- ✅ Parses content files rules
- ✅ Applies filters correctly
- ✅ Returns matching files

---

## Acceptance Criteria

### Package Reading

**Functional:**
- ✅ Opens and reads .nupkg files
- ✅ Parses .nuspec correctly
- ✅ Lists package contents
- ✅ Extracts files
- ✅ Validates structure

**Performance:**
- ✅ Opens package <50ms
- ✅ Parses nuspec <10ms
- ✅ Lists files <10ms

**Compatibility:**
- ✅ Reads packages created by NuGet.exe
- ✅ Reads packages from nuget.org
- ✅ Handles all nuspec schemas

### Package Creation

**Functional:**
- ✅ Creates valid .nupkg files
- ✅ Generates .nuspec
- ✅ OPC-compliant structure
- ✅ Installable by NuGet.exe

**Quality:**
- ✅ Validation before build
- ✅ Error messages actionable
- ✅ No corrupted packages

### Signature Verification

**Functional:**
- ✅ Verifies author signatures
- ✅ Verifies repository signatures
- ✅ Validates certificate chains
- ✅ Checks timestamps

**Security:**
- ✅ Rejects invalid signatures
- ✅ Rejects expired certificates (configurable)
- ✅ Checks revocation (optional)

### Asset Selection

**Functional:**
- ✅ Selects correct framework assets
- ✅ RID-specific selection works
- ✅ Content file filtering accurate

**Compatibility:**
- ✅ Selection matches C# NuGet.Client
- ✅ Handles all asset types
- ✅ Nearest framework logic correct

---

## Related Documents

- PRD-OVERVIEW.md - Product vision and goals
- PRD-CORE.md - Core library requirements
- PRD-PROTOCOL.md - Protocol implementation
- PRD-INFRASTRUCTURE.md - HTTP, caching, observability
- PRD-TESTING.md - Testing requirements
- PRD-RELEASE.md - Release criteria

---

**END OF PRD-PACKAGING.md**
