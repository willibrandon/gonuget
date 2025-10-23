package packaging

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/willibrandon/gonuget/version"
)

// PackagePathResolver resolves paths for V2 (packages.config) layout.
// Reference: PackagePathResolver class in NuGet.Packaging
type PackagePathResolver struct {
	rootDirectory      string
	useSideBySidePaths bool // Include version in directory name
}

// NewPackagePathResolver creates a V2 path resolver.
func NewPackagePathResolver(rootDirectory string, useSideBySidePaths bool) *PackagePathResolver {
	return &PackagePathResolver{
		rootDirectory:      rootDirectory,
		useSideBySidePaths: useSideBySidePaths,
	}
}

// GetPackageDirectoryName returns the directory name for a package.
// Format: {ID}.{Version} (if useSideBySidePaths) or {ID} (otherwise)
func (r *PackagePathResolver) GetPackageDirectoryName(identity *PackageIdentity) string {
	if r.useSideBySidePaths {
		return fmt.Sprintf("%s.%s", identity.ID, identity.Version.String())
	}
	return identity.ID
}

// GetInstallPath returns full installation path.
// Format: {rootDirectory}/{ID}.{Version} or {rootDirectory}/{ID}
func (r *PackagePathResolver) GetInstallPath(identity *PackageIdentity) string {
	return filepath.Join(r.rootDirectory, r.GetPackageDirectoryName(identity))
}

// GetPackageFileName returns the .nupkg file name.
// Format: {ID}.{Version}.nupkg
func (r *PackagePathResolver) GetPackageFileName(identity *PackageIdentity) string {
	return fmt.Sprintf("%s.%s.nupkg", identity.ID, identity.Version.String())
}

// GetPackageFilePath returns full path to .nupkg file.
func (r *PackagePathResolver) GetPackageFilePath(identity *PackageIdentity) string {
	return filepath.Join(r.GetInstallPath(identity), r.GetPackageFileName(identity))
}

// GetManifestFileName returns the .nuspec file name.
// Format: {ID}.nuspec (preserves original casing)
func (r *PackagePathResolver) GetManifestFileName(identity *PackageIdentity) string {
	return fmt.Sprintf("%s.nuspec", identity.ID)
}

// GetPackageDownloadMarkerFileName returns download marker filename.
// Format: {ID}.packagedownload.marker
func (r *PackagePathResolver) GetPackageDownloadMarkerFileName(identity *PackageIdentity) string {
	return fmt.Sprintf("%s.packagedownload.marker", identity.ID)
}

// VersionFolderPathResolver resolves paths for V3 (PackageReference) layout.
// Reference: VersionFolderPathResolver class in NuGet.Packaging
type VersionFolderPathResolver struct {
	rootPath    string
	isLowercase bool // Lowercase package IDs and versions
}

// NewVersionFolderPathResolver creates a V3 path resolver.
func NewVersionFolderPathResolver(rootPath string, isLowercase bool) *VersionFolderPathResolver {
	return &VersionFolderPathResolver{
		rootPath:    rootPath,
		isLowercase: isLowercase,
	}
}

// normalize applies lowercase if configured.
func (r *VersionFolderPathResolver) normalize(s string) string {
	if r.isLowercase {
		return strings.ToLower(s)
	}
	return s
}

// GetVersionListDirectory returns package ID folder.
// Format: {rootPath}/{id} (lowercase if configured)
func (r *VersionFolderPathResolver) GetVersionListDirectory(packageID string) string {
	return filepath.Join(r.rootPath, r.normalize(packageID))
}

// GetPackageDirectory returns package version folder.
// Format: {rootPath}/{id}/{version}
func (r *VersionFolderPathResolver) GetPackageDirectory(packageID string, ver *version.NuGetVersion) string {
	return filepath.Join(r.rootPath, r.normalize(packageID), r.normalize(ver.ToNormalizedString()))
}

// GetInstallPath returns full installation path.
func (r *VersionFolderPathResolver) GetInstallPath(packageID string, ver *version.NuGetVersion) string {
	return r.GetPackageDirectory(packageID, ver)
}

// GetPackageFilePath returns full path to .nupkg file.
// Format: {rootPath}/{id}/{version}/{id}.{version}.nupkg
func (r *VersionFolderPathResolver) GetPackageFilePath(packageID string, ver *version.NuGetVersion) string {
	dir := r.GetPackageDirectory(packageID, ver)
	normalizedID := r.normalize(packageID)
	normalizedVer := r.normalize(ver.ToNormalizedString())
	return filepath.Join(dir, fmt.Sprintf("%s.%s.nupkg", normalizedID, normalizedVer))
}

// GetManifestFilePath returns full path to .nuspec file.
// Format: {rootPath}/{id}/{version}/{id}.nuspec
func (r *VersionFolderPathResolver) GetManifestFilePath(packageID string, ver *version.NuGetVersion) string {
	dir := r.GetPackageDirectory(packageID, ver)
	return filepath.Join(dir, fmt.Sprintf("%s.nuspec", r.normalize(packageID)))
}

// GetHashPath returns full path to hash file.
// Format: {rootPath}/{id}/{version}/{id}.{version}.nupkg.sha512
func (r *VersionFolderPathResolver) GetHashPath(packageID string, ver *version.NuGetVersion) string {
	nupkgPath := r.GetPackageFilePath(packageID, ver)
	return nupkgPath + ".sha512"
}

// GetNupkgMetadataPath returns full path to metadata file.
// Format: {rootPath}/{id}/{version}/.nupkg.metadata
func (r *VersionFolderPathResolver) GetNupkgMetadataPath(packageID string, ver *version.NuGetVersion) string {
	dir := r.GetPackageDirectory(packageID, ver)
	return filepath.Join(dir, ".nupkg.metadata")
}
