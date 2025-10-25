# gonuget Packaging Design

**Component**: `pkg/gonuget/pack/` and `pkg/gonuget/signature/`
**Version**: 1.0.0
**Status**: Draft

---

## Table of Contents

1. [Overview](#overview)
2. [Package Structure](#package-structure)
3. [Package Reading](#package-reading)
4. [Package Creation](#package-creation)
5. [Package Validation](#package-validation)
6. [Package Signing](#package-signing)
7. [Signature Verification](#signature-verification)
8. [Implementation Details](#implementation-details)

---

## Overview

NuGet packages (.nupkg files) are ZIP archives with a specific structure. gonuget must support:

- **Reading packages**: Extract metadata and files
- **Creating packages**: Build .nupkg from files and manifest
- **Validating packages**: Check structure and metadata
- **Signing packages**: Add digital signatures (X.509)
- **Verifying signatures**: Validate package authenticity

---

## Package Structure

### .nupkg File Format

```
package.nupkg (ZIP archive)
├── package.nuspec                    # Package manifest (XML)
├── _rels/
│   └── .rels                         # Relationships (OPC format)
├── [Content_Types].xml               # Content types (OPC format)
├── package/
│   └── services/
│       └── metadata/
│           └── core-properties/
│               └── *.psmdcp         # Package metadata
├── lib/                              # Libraries
│   ├── net8.0/
│   │   └── MyLib.dll
│   ├── netstandard2.0/
│   │   └── MyLib.dll
│   └── net462/
│       └── MyLib.dll
├── ref/                              # Reference assemblies
│   └── net8.0/
│       └── MyLib.dll
├── content/                          # Content files (legacy)
│   └── README.md
├── build/                            # MSBuild targets
│   └── MyPackage.targets
├── buildTransitive/                  # Transitive MSBuild targets
│   └── MyPackage.targets
├── tools/                            # Tools and scripts
│   └── install.ps1
├── icon.png                          # Package icon (optional)
└── .signature.p7s                    # Package signature (optional)
```

### Nuspec Format

```xml
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>MyPackage</id>
    <version>1.0.0</version>
    <authors>Author Name</authors>
    <owners>Owner Name</owners>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <license type="expression">MIT</license>
    <licenseUrl>https://licenses.nuget.org/MIT</licenseUrl>
    <projectUrl>https://github.com/user/project</projectUrl>
    <icon>icon.png</icon>
    <description>Package description</description>
    <releaseNotes>Release notes</releaseNotes>
    <copyright>Copyright 2025</copyright>
    <tags>tag1 tag2 tag3</tags>
    <dependencies>
      <group targetFramework="net8.0">
        <dependency id="Newtonsoft.Json" version="13.0.3" />
      </group>
      <group targetFramework="netstandard2.0">
        <dependency id="Newtonsoft.Json" version="12.0.3" />
      </group>
    </dependencies>
  </metadata>
  <files>
    <file src="bin\Release\net8.0\MyLib.dll" target="lib\net8.0\" />
    <file src="bin\Release\netstandard2.0\MyLib.dll" target="lib\netstandard2.0\" />
    <file src="icon.png" target="" />
  </files>
</package>
```

---

## Package Reading

### PackageReader

**File**: `pkg/gonuget/pack/reader.go`

```go
package pack

import (
    "archive/zip"
    "encoding/xml"
    "fmt"
    "io"
    "path"
    "strings"
)

// PackageReader reads .nupkg files
type PackageReader struct {
    zipReader *zip.ReadCloser
    nuspec    *Nuspec
    files     map[string]*zip.File
}

// OpenPackage opens a .nupkg file for reading
func OpenPackage(path string) (*PackageReader, error) {
    zipReader, err := zip.OpenReader(path)
    if err != nil {
        return nil, fmt.Errorf("failed to open package: %w", err)
    }

    pr := &PackageReader{
        zipReader: zipReader,
        files:     make(map[string]*zip.File),
    }

    // Build file index
    for _, f := range zipReader.File {
        // Normalize path (forward slashes only)
        normalizedPath := filepath.ToSlash(f.Name)
        pr.files[normalizedPath] = f
    }

    // Find and parse nuspec
    if err := pr.loadNuspec(); err != nil {
        zipReader.Close()
        return nil, err
    }

    return pr, nil
}

// loadNuspec finds and parses the .nuspec file
func (pr *PackageReader) loadNuspec() error {
    // Nuspec is at root, ends with .nuspec
    var nuspecFile *zip.File
    for name, f := range pr.files {
        if strings.HasSuffix(name, ".nuspec") && !strings.Contains(name, "/") {
            nuspecFile = f
            break
        }
    }

    if nuspecFile == nil {
        return fmt.Errorf(".nuspec file not found in package")
    }

    // Open and parse nuspec
    rc, err := nuspecFile.Open()
    if err != nil {
        return fmt.Errorf("failed to open .nuspec: %w", err)
    }
    defer rc.Close()

    var nuspec Nuspec
    if err := xml.NewDecoder(rc).Decode(&nuspec); err != nil {
        return fmt.Errorf("failed to parse .nuspec: %w", err)
    }

    pr.nuspec = &nuspec
    return nil
}

// GetNuspec returns the package nuspec
func (pr *PackageReader) GetNuspec() *Nuspec {
    return pr.nuspec
}

// GetIdentity returns the package identity
func (pr *PackageReader) GetIdentity() *PackageIdentity {
    return &PackageIdentity{
        ID:      pr.nuspec.Metadata.ID,
        Version: pr.nuspec.Metadata.Version,
    }
}

// GetFiles returns all files in the package
func (pr *PackageReader) GetFiles() []string {
    files := make([]string, 0, len(pr.files))
    for name := range pr.files {
        files = append(files, name)
    }
    return files
}

// GetFile opens a file from the package
func (pr *PackageReader) GetFile(path string) (io.ReadCloser, error) {
    f, ok := pr.files[path]
    if !ok {
        return nil, fmt.Errorf("file not found: %s", path)
    }

    return f.Open()
}

// GetLibFiles returns library files for a framework
func (pr *PackageReader) GetLibFiles(framework string) []string {
    prefix := "lib/" + framework + "/"
    var files []string

    for name := range pr.files {
        if strings.HasPrefix(name, prefix) {
            files = append(files, name)
        }
    }

    return files
}

// GetSupportedFrameworks returns frameworks with lib files
func (pr *PackageReader) GetSupportedFrameworks() []string {
    frameworks := make(map[string]bool)

    for name := range pr.files {
        if strings.HasPrefix(name, "lib/") {
            parts := strings.Split(name, "/")
            if len(parts) >= 2 {
                frameworks[parts[1]] = true
            }
        }
    }

    result := make([]string, 0, len(frameworks))
    for fw := range frameworks {
        result = append(result, fw)
    }

    return result
}

// Close closes the package reader
func (pr *PackageReader) Close() error {
    return pr.zipReader.Close()
}

// Nuspec represents the package manifest
type Nuspec struct {
    XMLName  xml.Name        `xml:"package"`
    Metadata NuspecMetadata  `xml:"metadata"`
}

// NuspecMetadata contains package metadata
type NuspecMetadata struct {
    ID                       string              `xml:"id"`
    Version                  string              `xml:"version"`
    Authors                  string              `xml:"authors"`
    Owners                   string              `xml:"owners"`
    Title                    string              `xml:"title,omitempty"`
    Description              string              `xml:"description"`
    Summary                  string              `xml:"summary,omitempty"`
    ReleaseNotes             string              `xml:"releaseNotes,omitempty"`
    Copyright                string              `xml:"copyright,omitempty"`
    Language                 string              `xml:"language,omitempty"`
    ProjectURL               string              `xml:"projectUrl,omitempty"`
    IconURL                  string              `xml:"iconUrl,omitempty"`
    Icon                     string              `xml:"icon,omitempty"`
    LicenseURL               string              `xml:"licenseUrl,omitempty"`
    License                  NuspecLicense       `xml:"license,omitempty"`
    RequireLicenseAcceptance bool                `xml:"requireLicenseAcceptance"`
    Tags                     string              `xml:"tags,omitempty"`
    Dependencies             NuspecDependencies  `xml:"dependencies"`
}

// NuspecLicense represents license information
type NuspecLicense struct {
    Type    string `xml:"type,attr,omitempty"`
    Version string `xml:"version,attr,omitempty"`
    Value   string `xml:",chardata"`
}

// NuspecDependencies represents dependency groups
type NuspecDependencies struct {
    Groups []NuspecDependencyGroup `xml:"group"`
}

// NuspecDependencyGroup represents dependencies for a framework
type NuspecDependencyGroup struct {
    TargetFramework string              `xml:"targetFramework,attr,omitempty"`
    Dependencies    []NuspecDependency  `xml:"dependency"`
}

// NuspecDependency represents a single dependency
type NuspecDependency struct {
    ID      string `xml:"id,attr"`
    Version string `xml:"version,attr,omitempty"`
    Include string `xml:"include,attr,omitempty"`
    Exclude string `xml:"exclude,attr,omitempty"`
}
```

---

## Package Creation

### PackageBuilder

**File**: `pkg/gonuget/pack/builder.go`

```go
package pack

import (
    "archive/zip"
    "encoding/xml"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
)

// PackageBuilder builds .nupkg packages
type PackageBuilder struct {
    metadata *NuspecMetadata
    files    []PackageFile
}

// PackageFile represents a file to include in the package
type PackageFile struct {
    SourcePath string // Path to source file
    TargetPath string // Path within package (e.g., "lib/net8.0/MyLib.dll")
}

// NewPackageBuilder creates a new package builder
func NewPackageBuilder() *PackageBuilder {
    return &PackageBuilder{
        metadata: &NuspecMetadata{},
        files:    make([]PackageFile, 0),
    }
}

// ID sets the package ID
func (pb *PackageBuilder) ID(id string) *PackageBuilder {
    pb.metadata.ID = id
    return pb
}

// Version sets the package version
func (pb *PackageBuilder) Version(version string) *PackageBuilder {
    pb.metadata.Version = version
    return pb
}

// Authors sets the package authors
func (pb *PackageBuilder) Authors(authors ...string) *PackageBuilder {
    pb.metadata.Authors = strings.Join(authors, ",")
    return pb
}

// Description sets the package description
func (pb *PackageBuilder) Description(description string) *PackageBuilder {
    pb.metadata.Description = description
    return pb
}

// ProjectURL sets the project URL
func (pb *PackageBuilder) ProjectURL(url string) *PackageBuilder {
    pb.metadata.ProjectURL = url
    return pb
}

// LicenseExpression sets the SPDX license expression
func (pb *PackageBuilder) LicenseExpression(expression string) *PackageBuilder {
    pb.metadata.License = NuspecLicense{
        Type:  "expression",
        Value: expression,
    }
    return pb
}

// AddFile adds a file to the package
func (pb *PackageBuilder) AddFile(source, target string) *PackageBuilder {
    pb.files = append(pb.files, PackageFile{
        SourcePath: source,
        TargetPath: target,
    })
    return pb
}

// AddDependency adds a package dependency
func (pb *PackageBuilder) AddDependency(framework, id, versionRange string) *PackageBuilder {
    // Find or create dependency group for framework
    var group *NuspecDependencyGroup
    for i := range pb.metadata.Dependencies.Groups {
        if pb.metadata.Dependencies.Groups[i].TargetFramework == framework {
            group = &pb.metadata.Dependencies.Groups[i]
            break
        }
    }

    if group == nil {
        pb.metadata.Dependencies.Groups = append(pb.metadata.Dependencies.Groups, NuspecDependencyGroup{
            TargetFramework: framework,
        })
        group = &pb.metadata.Dependencies.Groups[len(pb.metadata.Dependencies.Groups)-1]
    }

    // Add dependency
    group.Dependencies = append(group.Dependencies, NuspecDependency{
        ID:      id,
        Version: versionRange,
    })

    return pb
}

// Build creates the .nupkg package
func (pb *PackageBuilder) Build(outputPath string) error {
    // Validate metadata
    if err := pb.validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    // Create output file
    outFile, err := os.Create(outputPath)
    if err != nil {
        return fmt.Errorf("failed to create output file: %w", err)
    }
    defer outFile.Close()

    // Create ZIP writer
    zipWriter := zip.NewWriter(outFile)
    defer zipWriter.Close()

    // Write nuspec
    if err := pb.writeNuspec(zipWriter); err != nil {
        return fmt.Errorf("failed to write nuspec: %w", err)
    }

    // Write OPC files
    if err := pb.writeOPCFiles(zipWriter); err != nil {
        return fmt.Errorf("failed to write OPC files: %w", err)
    }

    // Write package files
    for _, file := range pb.files {
        if err := pb.writeFile(zipWriter, file); err != nil {
            return fmt.Errorf("failed to write file %s: %w", file.SourcePath, err)
        }
    }

    return nil
}

// validate checks that all required metadata is present
func (pb *PackageBuilder) validate() error {
    if pb.metadata.ID == "" {
        return fmt.Errorf("package ID is required")
    }
    if pb.metadata.Version == "" {
        return fmt.Errorf("package version is required")
    }
    if pb.metadata.Authors == "" {
        return fmt.Errorf("package authors are required")
    }
    if pb.metadata.Description == "" {
        return fmt.Errorf("package description is required")
    }
    return nil
}

// writeNuspec writes the .nuspec file to the package
func (pb *PackageBuilder) writeNuspec(zipWriter *zip.Writer) error {
    nuspecName := pb.metadata.ID + ".nuspec"

    // Create nuspec file in ZIP
    w, err := zipWriter.Create(nuspecName)
    if err != nil {
        return err
    }

    // Create nuspec XML
    nuspec := Nuspec{
        Metadata: *pb.metadata,
    }

    // Write XML header
    if _, err := w.Write([]byte(xml.Header)); err != nil {
        return err
    }

    // Encode nuspec
    encoder := xml.NewEncoder(w)
    encoder.Indent("", "  ")
    if err := encoder.Encode(nuspec); err != nil {
        return err
    }

    return nil
}

// writeOPCFiles writes OPC (Open Packaging Conventions) files
func (pb *PackageBuilder) writeOPCFiles(zipWriter *zip.Writer) error {
    // Write [Content_Types].xml
    if err := pb.writeContentTypes(zipWriter); err != nil {
        return err
    }

    // Write _rels/.rels
    if err := pb.writeRels(zipWriter); err != nil {
        return err
    }

    return nil
}

// writeContentTypes writes [Content_Types].xml
func (pb *PackageBuilder) writeContentTypes(zipWriter *zip.Writer) error {
    w, err := zipWriter.Create("[Content_Types].xml")
    if err != nil {
        return err
    }

    contentTypes := `<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml" />
  <Default Extension="nuspec" ContentType="application/octet-stream" />
  <Default Extension="dll" ContentType="application/octet-stream" />
  <Default Extension="exe" ContentType="application/octet-stream" />
  <Default Extension="psmdcp" ContentType="application/vnd.openxmlformats-package.core-properties+xml" />
</Types>`

    _, err = w.Write([]byte(contentTypes))
    return err
}

// writeRels writes _rels/.rels
func (pb *PackageBuilder) writeRels(zipWriter *zip.Writer) error {
    w, err := zipWriter.Create("_rels/.rels")
    if err != nil {
        return err
    }

    rels := `<?xml version="1.0" encoding="utf-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Type="http://schemas.microsoft.com/packaging/2010/07/manifest" Target="/` + pb.metadata.ID + `.nuspec" Id="R0" />
</Relationships>`

    _, err = w.Write([]byte(rels))
    return err
}

// writeFile writes a file to the package
func (pb *PackageBuilder) writeFile(zipWriter *zip.Writer, file PackageFile) error {
    // Open source file
    sourceFile, err := os.Open(file.SourcePath)
    if err != nil {
        return err
    }
    defer sourceFile.Close()

    // Get file info
    info, err := sourceFile.Stat()
    if err != nil {
        return err
    }

    // Create file in ZIP
    header := &zip.FileHeader{
        Name:     file.TargetPath,
        Method:   zip.Deflate,
        Modified: time.Now(),
    }
    header.SetMode(info.Mode())

    w, err := zipWriter.CreateHeader(header)
    if err != nil {
        return err
    }

    // Copy file contents
    if _, err := io.Copy(w, sourceFile); err != nil {
        return err
    }

    return nil
}
```

---

## Package Validation

### PackageValidator

**File**: `pkg/gonuget/pack/validator.go`

```go
package pack

import (
    "fmt"
    "strings"
)

// ValidationIssue represents a package validation problem
type ValidationIssue struct {
    Severity Severity
    Code     string
    Message  string
}

// Severity levels
type Severity int

const (
    SeverityError Severity = iota
    SeverityWarning
    SeverityInfo
)

func (s Severity) String() string {
    switch s {
    case SeverityError:
        return "ERROR"
    case SeverityWarning:
        return "WARNING"
    case SeverityInfo:
        return "INFO"
    default:
        return "UNKNOWN"
    }
}

// PackageValidator validates package structure and metadata
type PackageValidator struct {
    issues []ValidationIssue
}

// NewPackageValidator creates a new validator
func NewPackageValidator() *PackageValidator {
    return &PackageValidator{
        issues: make([]ValidationIssue, 0),
    }
}

// Validate validates a package
func (pv *PackageValidator) Validate(pkg *PackageReader) []ValidationIssue {
    pv.issues = make([]ValidationIssue, 0)

    // Validate metadata
    pv.validateMetadata(pkg.GetNuspec().Metadata)

    // Validate structure
    pv.validateStructure(pkg)

    // Validate dependencies
    pv.validateDependencies(pkg.GetNuspec().Metadata.Dependencies)

    return pv.issues
}

// validateMetadata validates package metadata
func (pv *PackageValidator) validateMetadata(metadata NuspecMetadata) {
    // Required fields
    if metadata.ID == "" {
        pv.addError("NU1001", "Package ID is required")
    }

    if metadata.Version == "" {
        pv.addError("NU1002", "Package version is required")
    }

    if metadata.Authors == "" {
        pv.addError("NU1003", "Package authors are required")
    }

    if metadata.Description == "" {
        pv.addError("NU1004", "Package description is required")
    }

    // ID naming rules
    if strings.ContainsAny(metadata.ID, " !@#$%^&*()") {
        pv.addError("NU1005", "Package ID contains invalid characters")
    }

    // License validation
    if metadata.License.Type == "" && metadata.LicenseURL == "" {
        pv.addWarning("NU1006", "Package should specify a license")
    }

    // Icon validation
    if metadata.Icon != "" && metadata.IconURL != "" {
        pv.addWarning("NU1007", "Package specifies both icon file and iconUrl (icon file takes precedence)")
    }
}

// validateStructure validates package file structure
func (pv *PackageValidator) validateStructure(pkg *PackageReader) {
    files := pkg.GetFiles()

    // Check for required files
    hasNuspec := false
    hasContentTypes := false
    hasRels := false

    for _, file := range files {
        if strings.HasSuffix(file, ".nuspec") {
            hasNuspec = true
        }
        if file == "[Content_Types].xml" {
            hasContentTypes = true
        }
        if file == "_rels/.rels" {
            hasRels = true
        }
    }

    if !hasNuspec {
        pv.addError("NU1008", ".nuspec file is missing")
    }
    if !hasContentTypes {
        pv.addWarning("NU1009", "[Content_Types].xml is missing")
    }
    if !hasRels {
        pv.addWarning("NU1010", "_rels/.rels is missing")
    }

    // Check for lib files
    hasLibFiles := false
    for _, file := range files {
        if strings.HasPrefix(file, "lib/") {
            hasLibFiles = true
            break
        }
    }

    if !hasLibFiles {
        pv.addInfo("NU1011", "Package contains no lib files")
    }
}

// validateDependencies validates package dependencies
func (pv *PackageValidator) validateDependencies(deps NuspecDependencies) {
    for _, group := range deps.Groups {
        for _, dep := range group.Dependencies {
            // Check dependency ID
            if dep.ID == "" {
                pv.addError("NU1012", "Dependency has empty ID")
            }

            // Check version range
            if dep.Version != "" {
                // TODO: Parse and validate version range
            }
        }
    }
}

// Helper methods
func (pv *PackageValidator) addError(code, message string) {
    pv.issues = append(pv.issues, ValidationIssue{
        Severity: SeverityError,
        Code:     code,
        Message:  message,
    })
}

func (pv *PackageValidator) addWarning(code, message string) {
    pv.issues = append(pv.issues, ValidationIssue{
        Severity: SeverityWarning,
        Code:     code,
        Message:  message,
    })
}

func (pv *PackageValidator) addInfo(code, message string) {
    pv.issues = append(pv.issues, ValidationIssue{
        Severity: SeverityInfo,
        Code:     code,
        Message:  message,
    })
}
```

---

## Package Signing

### PackageSigner

**File**: `pkg/gonuget/signature/signer.go`

```go
package signature

import (
    "crypto"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "fmt"
    "io/ioutil"
    "os"

    "go.mozilla.org/pkcs7"
)

// PackageSigner signs NuGet packages
type PackageSigner struct {
    cert       *x509.Certificate
    privateKey *rsa.PrivateKey
    timestamp  string // RFC 3161 timestamp server URL
    logger     Logger
}

// NewPackageSigner creates a new package signer
func NewPackageSigner(certPath, keyPath, timestamp string, logger Logger) (*PackageSigner, error) {
    // Load certificate
    certPEM, err := ioutil.ReadFile(certPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read certificate: %w", err)
    }

    block, _ := pem.Decode(certPEM)
    if block == nil {
        return nil, fmt.Errorf("failed to decode certificate PEM")
    }

    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
        return nil, fmt.Errorf("failed to parse certificate: %w", err)
    }

    // Load private key
    keyPEM, err := ioutil.ReadFile(keyPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read private key: %w", err)
    }

    keyBlock, _ := pem.Decode(keyPEM)
    if keyBlock == nil {
        return nil, fmt.Errorf("failed to decode private key PEM")
    }

    privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
    if err != nil {
        return nil, fmt.Errorf("failed to parse private key: %w", err)
    }

    return &PackageSigner{
        cert:       cert,
        privateKey: privateKey,
        timestamp:  timestamp,
        logger:     logger,
    }, nil
}

// SignPackage signs a package and writes .signature.p7s
func (ps *PackageSigner) SignPackage(packagePath string) error {
    ps.logger.Info("Signing package {Path}", packagePath)

    // Read package file
    packageData, err := ioutil.ReadFile(packagePath)
    if err != nil {
        return fmt.Errorf("failed to read package: %w", err)
    }

    // Create PKCS#7 signature
    signedData, err := pkcs7.NewSignedData(packageData)
    if err != nil {
        return fmt.Errorf("failed to create signed data: %w", err)
    }

    // Add signer
    if err := signedData.AddSigner(ps.cert, ps.privateKey, pkcs7.SignerInfoConfig{
        Hash: crypto.SHA256,
    }); err != nil {
        return fmt.Errorf("failed to add signer: %w", err)
    }

    // Finalize signature
    signature, err := signedData.Finish()
    if err != nil {
        return fmt.Errorf("failed to finalize signature: %w", err)
    }

    // Write signature to .signature.p7s (next to package)
    signaturePath := packagePath + ".signature.p7s"
    if err := ioutil.WriteFile(signaturePath, signature, 0644); err != nil {
        return fmt.Errorf("failed to write signature: %w", err)
    }

    ps.logger.Info("Package signed successfully: {SignaturePath}", signaturePath)

    return nil
}
```

---

## Signature Verification

### SignatureVerifier

**File**: `pkg/gonuget/signature/verifier.go`

```go
package signature

import (
    "crypto/x509"
    "encoding/pem"
    "fmt"
    "io/ioutil"

    "go.mozilla.org/pkcs7"
)

// SignatureVerifier verifies package signatures
type SignatureVerifier struct {
    trustedCerts []*x509.Certificate
    logger       Logger
}

// VerificationResult contains verification results
type VerificationResult struct {
    IsValid   bool
    SignedBy  string
    Timestamp string
    Issues    []string
}

// NewSignatureVerifier creates a new signature verifier
func NewSignatureVerifier(trustedCertsPath string, logger Logger) (*SignatureVerifier, error) {
    // Load trusted certificates
    certsPEM, err := ioutil.ReadFile(trustedCertsPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read trusted certs: %w", err)
    }

    var certs []*x509.Certificate
    for {
        block, rest := pem.Decode(certsPEM)
        if block == nil {
            break
        }

        cert, err := x509.ParseCertificate(block.Bytes)
        if err != nil {
            logger.Warn("Failed to parse certificate: {Error}", err)
            certsPEM = rest
            continue
        }

        certs = append(certs, cert)
        certsPEM = rest
    }

    return &SignatureVerifier{
        trustedCerts: certs,
        logger:       logger,
    }, nil
}

// Verify verifies a package signature
func (sv *SignatureVerifier) Verify(packagePath string) (*VerificationResult, error) {
    result := &VerificationResult{
        Issues: make([]string, 0),
    }

    // Read signature file
    signaturePath := packagePath + ".signature.p7s"
    signatureData, err := ioutil.ReadFile(signaturePath)
    if err != nil {
        result.Issues = append(result.Issues, "Signature file not found")
        return result, nil
    }

    // Parse PKCS#7 signature
    p7, err := pkcs7.Parse(signatureData)
    if err != nil {
        result.Issues = append(result.Issues, fmt.Sprintf("Failed to parse signature: %v", err))
        return result, nil
    }

    // Read package file
    packageData, err := ioutil.ReadFile(packagePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read package: %w", err)
    }

    // Verify signature
    if err := p7.Verify(); err != nil {
        result.Issues = append(result.Issues, fmt.Sprintf("Signature verification failed: %v", err))
        return result, nil
    }

    // Check signer certificate against trusted certs
    signerCert := p7.GetOnlySigner()
    if signerCert == nil {
        result.Issues = append(result.Issues, "No signer certificate found")
        return result, nil
    }

    // Verify certificate chain
    roots := x509.NewCertPool()
    for _, cert := range sv.trustedCerts {
        roots.AddCert(cert)
    }

    opts := x509.VerifyOptions{
        Roots: roots,
    }

    if _, err := signerCert.Verify(opts); err != nil {
        result.Issues = append(result.Issues, fmt.Sprintf("Certificate verification failed: %v", err))
        return result, nil
    }

    // Success
    result.IsValid = true
    result.SignedBy = signerCert.Subject.CommonName

    return result, nil
}
```

---

## Implementation Details

### Dependencies

```go
require (
    // ZIP handling
    "archive/zip"

    // XML parsing
    "encoding/xml"

    // Crypto
    "crypto"
    "crypto/rsa"
    "crypto/x509"

    // PKCS#7 for signatures
    "go.mozilla.org/pkcs7" v0.0.0-20210826202110-33d05740a352
)
```

### Path Normalization

Always use forward slashes in package paths:

```go
// Good
"lib/net8.0/MyLib.dll"

// Bad
"lib\\net8.0\\MyLib.dll"
```

### Security Considerations

1. **Path traversal prevention**: Validate all file paths
2. **ZIP bombs**: Limit extraction size
3. **Signature verification**: Always verify before installing
4. **Certificate validation**: Check expiration and trust chain

---

**Document Status**: Draft v1.0
**Last Updated**: 2025-01-19
**Next Review**: After implementation
