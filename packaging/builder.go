package packaging

import (
	"archive/zip"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

// PackageBuilder builds .nupkg files.
type PackageBuilder struct {
	metadata PackageMetadata
	files    []PackageFile

	// Internal tracking
	filePaths   map[string]bool // For duplicate detection
	createdTime time.Time
}

// PackageFile represents a file to be added to the package.
type PackageFile struct {
	SourcePath string    // Path on disk (or empty for in-memory)
	TargetPath string    // Path in .nupkg
	Content    []byte    // In-memory content (if SourcePath is empty)
	Reader     io.Reader // Stream content (if SourcePath is empty and Content is nil)
}

// NewPackageBuilder creates a new package builder.
func NewPackageBuilder() *PackageBuilder {
	return &PackageBuilder{
		filePaths:   make(map[string]bool),
		createdTime: time.Now().UTC(),
	}
}

// NewPackageBuilderFromNuspec creates a builder from a nuspec file.
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

// ParseNuspecFile parses a nuspec from a file path.
func ParseNuspecFile(path string) (*Nuspec, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	return ParseNuspec(file)
}

// SetID sets the package ID.
func (b *PackageBuilder) SetID(id string) *PackageBuilder {
	b.metadata.ID = id
	return b
}

// SetVersion sets the package version.
func (b *PackageBuilder) SetVersion(ver *version.NuGetVersion) *PackageBuilder {
	b.metadata.Version = ver
	return b
}

// SetDescription sets the package description.
func (b *PackageBuilder) SetDescription(description string) *PackageBuilder {
	b.metadata.Description = description
	return b
}

// SetAuthors sets the package authors.
func (b *PackageBuilder) SetAuthors(authors ...string) *PackageBuilder {
	b.metadata.Authors = authors
	return b
}

// SetTitle sets the package title.
func (b *PackageBuilder) SetTitle(title string) *PackageBuilder {
	b.metadata.Title = title
	return b
}

// SetOwners sets the package owners.
func (b *PackageBuilder) SetOwners(owners ...string) *PackageBuilder {
	b.metadata.Owners = owners
	return b
}

// SetProjectURL sets the project URL.
func (b *PackageBuilder) SetProjectURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid project URL: %w", err)
	}
	b.metadata.ProjectURL = u
	return nil
}

// SetIconURL sets the icon URL.
func (b *PackageBuilder) SetIconURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid icon URL: %w", err)
	}
	b.metadata.IconURL = u
	return nil
}

// SetLicenseURL sets the license URL.
func (b *PackageBuilder) SetLicenseURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid license URL: %w", err)
	}
	b.metadata.LicenseURL = u
	return nil
}

// SetTags sets the package tags.
func (b *PackageBuilder) SetTags(tags ...string) *PackageBuilder {
	b.metadata.Tags = tags
	return b
}

// SetCopyright sets the copyright.
func (b *PackageBuilder) SetCopyright(copyright string) *PackageBuilder {
	b.metadata.Copyright = copyright
	return b
}

// SetSummary sets the summary.
func (b *PackageBuilder) SetSummary(summary string) *PackageBuilder {
	b.metadata.Summary = summary
	return b
}

// SetReleaseNotes sets the release notes.
func (b *PackageBuilder) SetReleaseNotes(releaseNotes string) *PackageBuilder {
	b.metadata.ReleaseNotes = releaseNotes
	return b
}

// SetLanguage sets the language.
func (b *PackageBuilder) SetLanguage(language string) *PackageBuilder {
	b.metadata.Language = language
	return b
}

// SetRequireLicenseAcceptance sets whether license acceptance is required.
func (b *PackageBuilder) SetRequireLicenseAcceptance(required bool) *PackageBuilder {
	b.metadata.RequireLicenseAcceptance = required
	return b
}

// SetDevelopmentDependency sets whether this is a development dependency.
func (b *PackageBuilder) SetDevelopmentDependency(isDev bool) *PackageBuilder {
	b.metadata.DevelopmentDependency = isDev
	return b
}

// SetServiceable sets whether the package is serviceable.
func (b *PackageBuilder) SetServiceable(serviceable bool) *PackageBuilder {
	b.metadata.Serviceable = serviceable
	return b
}

// SetIcon sets the icon file path in the package.
func (b *PackageBuilder) SetIcon(icon string) *PackageBuilder {
	b.metadata.Icon = icon
	return b
}

// SetReadme sets the readme file path in the package.
func (b *PackageBuilder) SetReadme(readme string) *PackageBuilder {
	b.metadata.Readme = readme
	return b
}

// SetMinClientVersion sets the minimum client version.
func (b *PackageBuilder) SetMinClientVersion(ver *version.NuGetVersion) *PackageBuilder {
	b.metadata.MinClientVersion = ver
	return b
}

// SetRepository sets repository metadata.
func (b *PackageBuilder) SetRepository(repo *PackageRepositoryMetadata) *PackageBuilder {
	b.metadata.Repository = repo
	return b
}

// SetLicenseMetadata sets license metadata.
func (b *PackageBuilder) SetLicenseMetadata(license *LicenseMetadata) *PackageBuilder {
	b.metadata.LicenseMetadata = license
	return b
}

// AddDependencyGroup adds a dependency group.
func (b *PackageBuilder) AddDependencyGroup(group PackageDependencyGroup) *PackageBuilder {
	b.metadata.DependencyGroups = append(b.metadata.DependencyGroups, group)
	return b
}

// AddDependency adds a dependency to a specific framework (or nil for "any").
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

// AddFrameworkReferenceGroup adds a framework reference group.
func (b *PackageBuilder) AddFrameworkReferenceGroup(group PackageFrameworkReferenceGroup) *PackageBuilder {
	b.metadata.FrameworkReferenceGroups = append(b.metadata.FrameworkReferenceGroups, group)
	return b
}

// AddPackageType adds a package type.
func (b *PackageBuilder) AddPackageType(packageType PackageTypeInfo) *PackageBuilder {
	b.metadata.PackageTypes = append(b.metadata.PackageTypes, packageType)
	return b
}

func frameworksEqual(a, b *frameworks.NuGetFramework) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Framework == b.Framework && a.Version.Compare(b.Version) == 0
}

// AddFile adds a file from disk to the package.
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

// AddFileFromBytes adds a file from in-memory bytes.
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

// AddFileFromReader adds a file from an io.Reader.
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

// normalizePackagePath normalizes a package path to use forward slashes.
func normalizePackagePath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

// PopulateFromNuspec populates builder metadata from a parsed nuspec.
func (b *PackageBuilder) PopulateFromNuspec(nuspec *Nuspec) error {
	// Parse identity
	identity, err := nuspec.GetParsedIdentity()
	if err != nil {
		return fmt.Errorf("parse identity: %w", err)
	}

	b.metadata.ID = identity.ID
	b.metadata.Version = identity.Version

	// Required fields
	b.metadata.Description = nuspec.Metadata.Description
	b.metadata.Authors = nuspec.GetAuthors()

	// Optional fields
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
			return fmt.Errorf("parse project URL: %w", err)
		}
	}

	if nuspec.Metadata.IconURL != "" {
		if err := b.SetIconURL(nuspec.Metadata.IconURL); err != nil {
			return fmt.Errorf("parse icon URL: %w", err)
		}
	}

	if nuspec.Metadata.LicenseURL != "" {
		if err := b.SetLicenseURL(nuspec.Metadata.LicenseURL); err != nil {
			return fmt.Errorf("parse license URL: %w", err)
		}
	}

	// License metadata
	if nuspec.Metadata.License != nil {
		b.metadata.LicenseMetadata = &LicenseMetadata{
			Type:    nuspec.Metadata.License.Type,
			Text:    nuspec.Metadata.License.Text,
			Version: nuspec.Metadata.License.Version,
		}
	}

	// Min client version
	if nuspec.Metadata.MinClientVersion != "" {
		minVer, err := version.Parse(nuspec.Metadata.MinClientVersion)
		if err != nil {
			return fmt.Errorf("parse min client version: %w", err)
		}
		b.metadata.MinClientVersion = minVer
	}

	// Repository metadata
	if nuspec.Metadata.Repository != nil {
		b.metadata.Repository = &PackageRepositoryMetadata{
			Type:   nuspec.Metadata.Repository.Type,
			URL:    nuspec.Metadata.Repository.URL,
			Branch: nuspec.Metadata.Repository.Branch,
			Commit: nuspec.Metadata.Repository.Commit,
		}
	}

	// Package types
	for _, pt := range nuspec.Metadata.PackageTypes {
		var ptVer *version.NuGetVersion
		if pt.Version != "" {
			var err error
			ptVer, err = version.Parse(pt.Version)
			if err != nil {
				return fmt.Errorf("parse package type version: %w", err)
			}
		}

		b.metadata.PackageTypes = append(b.metadata.PackageTypes, PackageTypeInfo{
			Name:    pt.Name,
			Version: ptVer,
		})
	}

	// Parse dependencies
	depGroups, err := nuspec.GetDependencyGroups()
	if err != nil {
		return fmt.Errorf("parse dependencies: %w", err)
	}

	for _, group := range depGroups {
		b.metadata.DependencyGroups = append(b.metadata.DependencyGroups, group.ToPackageDependencyGroup())
	}

	// Parse framework references
	fwRefGroups, err := nuspec.GetFrameworkReferenceGroups()
	if err != nil {
		return fmt.Errorf("parse framework references: %w", err)
	}

	for _, group := range fwRefGroups {
		b.metadata.FrameworkReferenceGroups = append(b.metadata.FrameworkReferenceGroups, group.ToPackageFrameworkReferenceGroup())
	}

	return nil
}

// GetMetadata returns the current package metadata.
func (b *PackageBuilder) GetMetadata() PackageMetadata {
	return b.metadata
}

// GetFiles returns the current file list.
func (b *PackageBuilder) GetFiles() []PackageFile {
	return b.files
}

// writeOPCFiles writes OPC-required files to the ZIP.
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

// Save writes the package to a stream.
func (b *PackageBuilder) Save(writer io.Writer) error {
	// Comprehensive validation
	if err := b.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Create ZIP archive
	zipWriter := zip.NewWriter(writer)
	defer func() { _ = zipWriter.Close() }()

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

	// Close the ZIP writer before returning
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("close ZIP: %w", err)
	}

	return nil
}

// SaveToFile writes the package to a file.
func (b *PackageBuilder) SaveToFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if err := b.Save(file); err != nil {
		return err
	}

	return file.Close()
}

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
	if len(b.files) > 0 {
		if err := ValidateFiles(b.files); err != nil {
			return fmt.Errorf("file validation: %w", err)
		}
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
		defer func() { _ = sourceFile.Close() }()

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
