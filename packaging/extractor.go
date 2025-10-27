// Package packaging provides NuGet package extraction functionality.
package packaging

import (
	"context"

	"github.com/willibrandon/gonuget/version"
)

// PackageSaveMode controls what gets extracted (flags).
// Reference: PackageSaveMode enum in NuGet.Packaging
type PackageSaveMode int

const (
	// PackageSaveModeNone indicates no save mode.
	PackageSaveModeNone PackageSaveMode = 0
	// PackageSaveModeNuspec extracts .nuspec files.
	PackageSaveModeNuspec PackageSaveMode = 1 << 0
	// PackageSaveModeNupkg saves .nupkg files.
	PackageSaveModeNupkg PackageSaveMode = 1 << 1
	// PackageSaveModeFiles extracts package files.
	PackageSaveModeFiles PackageSaveMode = 1 << 2

	// PackageSaveModeDefaultV2 is the default for packages.config projects (V2).
	PackageSaveModeDefaultV2 = PackageSaveModeNupkg | PackageSaveModeFiles

	// PackageSaveModeDefaultV3 is the default for PackageReference projects (V3).
	PackageSaveModeDefaultV3 = PackageSaveModeNuspec | PackageSaveModeNupkg | PackageSaveModeFiles
)

// HasFlag checks if a specific flag is set.
func (m PackageSaveMode) HasFlag(flag PackageSaveMode) bool {
	return m&flag != 0
}

// XMLDocFileSaveMode controls XML documentation file handling.
// Reference: XmlDocFileSaveMode enum in NuGet.Packaging
type XMLDocFileSaveMode int

const (
	// XMLDocFileSaveModeNone extracts XML documentation files normally.
	XMLDocFileSaveModeNone XMLDocFileSaveMode = 0
	// XMLDocFileSaveModeSkip skips extracting XML documentation files.
	XMLDocFileSaveModeSkip XMLDocFileSaveMode = 1
	// XMLDocFileSaveModeCompress compresses XML documentation files as .xml.zip.
	XMLDocFileSaveModeCompress XMLDocFileSaveMode = 2
)

// PackageExtractionContext configures extraction behavior.
// Reference: PackageExtractionContext class in NuGet.Packaging
type PackageExtractionContext struct {
	// PackageSaveMode controls what files to extract
	PackageSaveMode PackageSaveMode

	// XMLDocFileSaveMode controls XML documentation handling
	XMLDocFileSaveMode XMLDocFileSaveMode

	// CopySatelliteFiles enables satellite package file merging
	CopySatelliteFiles bool

	// SignatureVerifier for signed package validation (optional)
	SignatureVerifier SignatureVerifier

	// Logger for extraction progress (optional)
	Logger Logger

	// ParentID for telemetry correlation (optional)
	ParentID string
}

// SignatureVerifier interface for package signature verification.
type SignatureVerifier interface {
	VerifySignatureAsync(ctx context.Context, reader *PackageReader) error
}

// Logger interface for extraction logging.
type Logger interface {
	Info(format string, args ...any)
	Warning(format string, args ...any)
	Error(format string, args ...any)
}

// DefaultExtractionContext returns context with sensible defaults.
func DefaultExtractionContext() *PackageExtractionContext {
	return &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV3,
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		CopySatelliteFiles: true,
	}
}

// PathResolver interface for V2/V3 path resolution.
type PathResolver interface {
	GetInstallPath(packageID string, ver *version.NuGetVersion) string
}
