package main

// SignPackageRequest represents a request to create a package signature.
// This matches the NuGet.Client SignPackageRequest semantics.
type SignPackageRequest struct {
	// PackageHash is the hash of the package content (ZIP archive).
	// This is the value that gets signed (detached signature).
	PackageHash []byte `json:"packageHash"`

	// CertPath is the filesystem path to the signing certificate.
	// Supports PEM, DER, or PFX formats.
	CertPath string `json:"certPath"`

	// CertPassword is optional password for encrypted certificates (PFX).
	CertPassword string `json:"certPassword,omitempty"`

	// KeyPath is the filesystem path to the private key (PEM format).
	// Not needed if CertPath points to PFX with embedded key.
	KeyPath string `json:"keyPath,omitempty"`

	// SignatureType is "Author" or "Repository".
	SignatureType string `json:"signatureType"`

	// HashAlgorithm is "SHA256", "SHA384", or "SHA512".
	HashAlgorithm string `json:"hashAlgorithm"`

	// TimestampURL is optional RFC 3161 timestamp authority URL.
	// If empty, signature is created without timestamp.
	TimestampURL string `json:"timestampURL,omitempty"`
}

// SignPackageResponse contains the created signature bytes.
type SignPackageResponse struct {
	// Signature is the DER-encoded PKCS#7 SignedData structure.
	// This can be loaded by NuGet.Client's PrimarySignature.Load().
	Signature []byte `json:"signature"`
}

// ParseSignatureRequest asks gonuget to parse a signature.
type ParseSignatureRequest struct {
	// Signature is DER-encoded PKCS#7 SignedData to parse.
	Signature []byte `json:"signature"`
}

// ParseSignatureResponse contains parsed signature metadata.
// This allows C# tests to verify gonuget extracts the same data as NuGet.Client.
type ParseSignatureResponse struct {
	// Type is "Author", "Repository", or "Unknown".
	Type string `json:"type"`

	// HashAlgorithm is "SHA256", "SHA384", "SHA512", or "Unknown".
	HashAlgorithm string `json:"hashAlgorithm"`

	// SignerCertHash is hex-encoded SubjectKeyId or certificate hash.
	SignerCertHash string `json:"signerCertHash"`

	// TimestampCount is the number of timestamps in the signature.
	TimestampCount int `json:"timestampCount"`

	// TimestampTimes are RFC3339-formatted timestamp generation times.
	TimestampTimes []string `json:"timestampTimes,omitempty"`

	// Certificates is the number of certificates in the chain.
	Certificates int `json:"certificates"`
}

// VerifySignatureRequest validates a signature.
type VerifySignatureRequest struct {
	// Signature is the signature to verify.
	Signature []byte `json:"signature"`

	// TrustedRoots are DER-encoded root CA certificates.
	TrustedRoots [][]byte `json:"trustedRoots,omitempty"`

	// AllowUntrustedRoot allows signatures with untrusted roots (for testing).
	AllowUntrustedRoot bool `json:"allowUntrustedRoot"`

	// RequireTimestamp requires the signature to be timestamped.
	RequireTimestamp bool `json:"requireTimestamp"`
}

// VerifySignatureResponse contains verification results.
type VerifySignatureResponse struct {
	// Valid is true if signature verification succeeded.
	Valid bool `json:"valid"`

	// Errors contains verification failure messages.
	Errors []string `json:"errors,omitempty"`

	// Warnings contains non-fatal verification warnings.
	Warnings []string `json:"warnings,omitempty"`

	// SignerSubject is the signer certificate's subject DN.
	SignerSubject string `json:"signerSubject,omitempty"`
}

// CompareVersionsRequest compares two NuGet version strings.
type CompareVersionsRequest struct {
	Version1 string `json:"version1"`
	Version2 string `json:"version2"`
}

// CompareVersionsResponse returns version comparison result.
type CompareVersionsResponse struct {
	// Result is -1 (v1 < v2), 0 (equal), or 1 (v1 > v2).
	Result int `json:"result"`
}

// ParseVersionRequest parses a NuGet version string.
type ParseVersionRequest struct {
	Version string `json:"version"`
}

// ParseVersionResponse contains parsed version components.
// This matches NuGet.Versioning.NuGetVersion structure.
type ParseVersionResponse struct {
	Major        int    `json:"major"`
	Minor        int    `json:"minor"`
	Patch        int    `json:"patch"`
	Revision     int    `json:"revision"`
	Release      string `json:"release"`      // Pre-release label (e.g., "beta.1")
	Metadata     string `json:"metadata"`     // Build metadata (e.g., "20130313144700")
	IsPrerelease bool   `json:"isPrerelease"` // True if has pre-release label
	IsLegacy     bool   `json:"isLegacy"`     // True if uses legacy 4-part format
}

// CheckFrameworkCompatRequest checks framework compatibility.
type CheckFrameworkCompatRequest struct {
	// PackageFramework is the framework the package supports (e.g., "net6.0").
	PackageFramework string `json:"packageFramework"`

	// ProjectFramework is the project's target framework (e.g., "net8.0").
	ProjectFramework string `json:"projectFramework"`
}

// CheckFrameworkCompatResponse returns compatibility result.
type CheckFrameworkCompatResponse struct {
	// Compatible is true if project can use package.
	Compatible bool `json:"compatible"`

	// Reason explains why incompatible (empty if compatible).
	Reason string `json:"reason,omitempty"`
}

// ParseFrameworkRequest parses a framework identifier.
type ParseFrameworkRequest struct {
	Framework string `json:"framework"`
}

// ParseFrameworkResponse contains parsed framework components.
type ParseFrameworkResponse struct {
	// Identifier is the framework identifier (e.g., ".NETCoreApp", ".NETFramework").
	Identifier string `json:"identifier"`

	// Version is the framework version (e.g., "6.0", "4.7.2").
	Version string `json:"version"`

	// Profile is optional profile (e.g., "Client" for ".NETFramework,Profile=Client").
	Profile string `json:"profile,omitempty"`

	// Platform is optional platform (e.g., "windows7.0").
	Platform string `json:"platform,omitempty"`
}

// FormatFrameworkRequest formats a framework to its short folder name.
type FormatFrameworkRequest struct {
	// Framework is the framework to format (e.g., "net6.0-windows", "portable-net45+win8").
	Framework string `json:"framework"`
}

// FormatFrameworkResponse contains the formatted short folder name.
type FormatFrameworkResponse struct {
	// ShortFolderName is the formatted name (e.g., "net6.0-windows", "portable-net45+win8").
	// This matches NuGet.Client's GetShortFolderName() output.
	ShortFolderName string `json:"shortFolderName"`
}

// ReadPackageRequest reads package structure and metadata.
type ReadPackageRequest struct {
	// PackageBytes is the ZIP package content.
	PackageBytes []byte `json:"packageBytes"`
}

// ReadPackageResponse contains package metadata.
type ReadPackageResponse struct {
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	Authors       []string `json:"authors"`                // Always serialize, even if empty
	Description   string   `json:"description"`            // Always serialize, even if empty
	Dependencies  []string `json:"dependencies,omitempty"` // Formatted as "id:version"
	FileCount     int      `json:"fileCount"`
	HasSignature  bool     `json:"hasSignature"`
	SignatureType string   `json:"signatureType,omitempty"` // "Author", "Repository", or empty
}

// BuildPackageRequest creates a minimal NuGet package.
type BuildPackageRequest struct {
	ID          string            `json:"id"`
	Version     string            `json:"version"`
	Authors     []string          `json:"authors,omitempty"`
	Description string            `json:"description,omitempty"`
	Files       map[string][]byte `json:"files"` // Relative path -> content
}

// BuildPackageResponse contains the built package.
type BuildPackageResponse struct {
	// PackageBytes is the ZIP package content.
	PackageBytes []byte `json:"packageBytes"`
}

// ContentItemData represents a content item with path and properties.
type ContentItemData struct {
	Path       string                 `json:"path"`
	Properties map[string]interface{} `json:"properties"`
}

// FindAssembliesRequest finds assemblies matching patterns and framework.
type FindAssembliesRequest struct {
	// Paths are package file paths to match (e.g., "lib/net6.0/MyLib.dll").
	Paths []string `json:"paths"`

	// TargetFramework is optional framework filter (e.g., "net8.0").
	TargetFramework string `json:"targetFramework,omitempty"`
}

// FindAssembliesResponse contains matched assembly items.
type FindAssembliesResponse struct {
	Items []ContentItemData `json:"items"`
}

// NewFindAssembliesResponse creates a response with an empty items array.
func NewFindAssembliesResponse() FindAssembliesResponse {
	return FindAssembliesResponse{Items: []ContentItemData{}}
}

// ParseAssetPathRequest parses a single asset path.
type ParseAssetPathRequest struct {
	Path string `json:"path"`
}

// ParseAssetPathResponse contains the parsed path properties.
type ParseAssetPathResponse struct {
	// Item is nil if path didn't match any pattern.
	Item *ContentItemData `json:"item"`
}

// ExpandRuntimeRequest expands a runtime identifier to compatible RIDs.
type ExpandRuntimeRequest struct {
	// RID is the runtime identifier to expand (e.g., "win10-x64").
	RID string `json:"rid"`
}

// ExpandRuntimeResponse contains the expanded runtime identifiers.
type ExpandRuntimeResponse struct {
	// ExpandedRuntimes is the array of compatible RIDs in priority order.
	// The first element is the original RID, followed by compatible RIDs (nearest first).
	ExpandedRuntimes []string `json:"expandedRuntimes"`
}

// AreRuntimesCompatibleRequest checks if two RIDs are compatible.
type AreRuntimesCompatibleRequest struct {
	// TargetRID is the target runtime (criteria).
	TargetRID string `json:"targetRid"`

	// PackageRID is the package runtime (provided).
	PackageRID string `json:"packageRid"`
}

// AreRuntimesCompatibleResponse contains the compatibility result.
type AreRuntimesCompatibleResponse struct {
	// Compatible is true if the package RID is compatible with the target RID.
	Compatible bool `json:"compatible"`
}

// ExtractPackageV2Request extracts a package using V2 (packages.config) layout.
type ExtractPackageV2Request struct {
	// PackageBytes is the ZIP package content.
	PackageBytes []byte `json:"packageBytes"`

	// InstallPath is the target directory for extraction.
	InstallPath string `json:"installPath"`

	// PackageSaveMode controls what to extract (nuspec, files, nupkg).
	// Bitmask: 1=Nuspec, 2=Nupkg, 4=Files
	PackageSaveMode int `json:"packageSaveMode"`

	// UseSideBySideLayout controls directory naming (ID.Version vs ID).
	UseSideBySideLayout bool `json:"useSideBySideLayout"`

	// XMLDocFileSaveMode controls XML doc compression (0=None, 1=Skip, 2=Compress).
	XMLDocFileSaveMode int `json:"xmlDocFileSaveMode"`
}

// ExtractPackageV2Response contains extraction results.
type ExtractPackageV2Response struct {
	// ExtractedFiles are paths to all extracted files.
	ExtractedFiles []string `json:"extractedFiles"`

	// FileCount is the number of files extracted.
	FileCount int `json:"fileCount"`
}

// InstallFromSourceV3Request installs a package using V3 (PackageReference) layout.
type InstallFromSourceV3Request struct {
	// PackageBytes is the ZIP package content.
	PackageBytes []byte `json:"packageBytes"`

	// ID is the package ID.
	ID string `json:"id"`

	// Version is the package version.
	Version string `json:"version"`

	// GlobalPackagesFolder is the target global packages directory.
	GlobalPackagesFolder string `json:"globalPackagesFolder"`

	// PackageSaveMode controls what to save (nuspec, files, nupkg).
	// Bitmask: 1=Nuspec, 2=Nupkg, 4=Files
	PackageSaveMode int `json:"packageSaveMode"`

	// XMLDocFileSaveMode controls XML doc compression (0=None, 1=Skip, 2=Compress).
	XMLDocFileSaveMode int `json:"xmlDocFileSaveMode"`
}

// InstallFromSourceV3Response contains installation results.
type InstallFromSourceV3Response struct {
	// Installed is true if package was installed (false if already existed).
	Installed bool `json:"installed"`

	// PackageDirectory is the final package directory path.
	PackageDirectory string `json:"packageDirectory"`

	// NupkgPath is the path to the .nupkg file (if saved).
	NupkgPath string `json:"nupkgPath,omitempty"`

	// NuspecPath is the path to the .nuspec file.
	NuspecPath string `json:"nuspecPath"`

	// HashPath is the path to the .sha512 hash file.
	HashPath string `json:"hashPath"`

	// MetadataPath is the path to the .nupkg.metadata file.
	MetadataPath string `json:"metadataPath"`
}

// ComputeCacheHashRequest asks gonuget to compute a cache hash.
// This validates CachingUtility.ComputeHash() compatibility.
type ComputeCacheHashRequest struct {
	// Value is the string to hash (usually a URL or package ID).
	Value string `json:"value"`

	// AddIdentifiableCharacters appends trailing portion for readability.
	AddIdentifiableCharacters bool `json:"addIdentifiableCharacters"`
}

// ComputeCacheHashResponse contains the computed hash string.
type ComputeCacheHashResponse struct {
	// Hash is the computed cache hash (40-char hex + optional trailing chars).
	Hash string `json:"hash"`
}

// SanitizeCacheFilenameRequest asks gonuget to sanitize a filename.
// This validates CachingUtility.RemoveInvalidFileNameChars() compatibility.
type SanitizeCacheFilenameRequest struct {
	// Value is the filename or path to sanitize.
	Value string `json:"value"`
}

// SanitizeCacheFilenameResponse contains the sanitized filename.
type SanitizeCacheFilenameResponse struct {
	// Sanitized is the filename with invalid chars replaced and collapsed.
	Sanitized string `json:"sanitized"`
}

// GenerateCachePathsRequest asks gonuget to generate cache file paths.
// This validates HttpCacheUtility.InitializeHttpCacheResult() compatibility.
type GenerateCachePathsRequest struct {
	// CacheDirectory is the root cache directory path.
	CacheDirectory string `json:"cacheDirectory"`

	// SourceURL is the source URL to hash for the folder name.
	SourceURL string `json:"sourceURL"`

	// CacheKey is the cache key for the file name.
	CacheKey string `json:"cacheKey"`
}

// GenerateCachePathsResponse contains the generated cache paths.
type GenerateCachePathsResponse struct {
	// BaseFolderName is the hash-based folder name.
	BaseFolderName string `json:"baseFolderName"`

	// CacheFile is the full path to the cache file.
	CacheFile string `json:"cacheFile"`

	// NewFile is the full path to the temporary file during atomic writes.
	NewFile string `json:"newFile"`
}

// ValidateCacheFileRequest asks gonuget to validate cache file TTL expiration.
// This validates CachingUtility.ReadCacheFile() TTL logic compatibility.
type ValidateCacheFileRequest struct {
	// CacheDirectory is the root cache directory path.
	CacheDirectory string `json:"cacheDirectory"`

	// SourceURL is the source URL for the cached resource.
	SourceURL string `json:"sourceURL"`

	// CacheKey is the cache key for the file.
	CacheKey string `json:"cacheKey"`

	// MaxAgeSeconds is the maximum age in seconds before the cache is considered expired.
	MaxAgeSeconds int64 `json:"maxAgeSeconds"`
}

// ValidateCacheFileResponse indicates whether the cache file is valid (not expired).
type ValidateCacheFileResponse struct {
	// Valid is true if the file exists and is within the TTL, false if missing or expired.
	Valid bool `json:"valid"`
}

// WalkGraphRequest walks the dependency graph for a package.
type WalkGraphRequest struct {
	// PackageID is the package identifier (e.g., "Newtonsoft.Json").
	PackageID string `json:"packageId"`

	// VersionRange is the version constraint (e.g., "[13.0.1]", "[1.0.0,2.0.0)").
	VersionRange string `json:"versionRange"`

	// TargetFramework is the target framework (e.g., "net8.0").
	TargetFramework string `json:"targetFramework"`

	// Sources is the list of package sources to query.
	Sources []string `json:"sources"`
}

// WalkGraphResponse contains the dependency graph in flat array format.
type WalkGraphResponse struct {
	// Nodes is the flat array of all graph nodes.
	Nodes []GraphNodeData `json:"nodes"`

	// Cycles is the array of detected circular dependencies (package IDs).
	Cycles []string `json:"cycles"`

	// Downgrades is the array of detected version downgrades.
	Downgrades []DowngradeInfo `json:"downgrades"`
}

// GraphNodeData represents a node in the dependency graph.
// Matches the C# GraphNodeData structure.
type GraphNodeData struct {
	// PackageID is the package identifier.
	PackageID string `json:"packageId"`

	// Version is the package version.
	Version string `json:"version"`

	// Disposition is the node state (Acceptable, Rejected, Accepted, PotentiallyDowngraded, Cycle).
	Disposition string `json:"disposition"`

	// Depth is the distance from root (0 for root).
	Depth int `json:"depth"`

	// Dependencies are the package IDs of direct dependencies (not full node data).
	Dependencies []string `json:"dependencies"`
}

// DowngradeInfo represents a detected version downgrade.
type DowngradeInfo struct {
	// PackageID is the package being downgraded.
	PackageID string `json:"packageId"`

	// FromVersion is the current (higher) version.
	FromVersion string `json:"fromVersion"`

	// ToVersion is the target (lower) version.
	ToVersion string `json:"toVersion"`
}

// ResolveConflictsRequest resolves version conflicts in a dependency set.
type ResolveConflictsRequest struct {
	// PackageIDs are the package identifiers to resolve.
	PackageIDs []string `json:"packageIds"`

	// VersionRanges are the version constraints for each package.
	VersionRanges []string `json:"versionRanges"`

	// TargetFramework is the target framework.
	TargetFramework string `json:"targetFramework"`
}

// ResolveConflictsResponse contains the resolved packages.
type ResolveConflictsResponse struct {
	// Packages are the resolved packages after conflict resolution.
	Packages []ResolvedPackage `json:"packages"`
}

// ResolvedPackage represents a package after conflict resolution.
type ResolvedPackage struct {
	// PackageID is the package identifier.
	PackageID string `json:"packageId"`

	// Version is the selected version.
	Version string `json:"version"`

	// Depth is the depth in the dependency graph.
	Depth int `json:"depth"`
}
