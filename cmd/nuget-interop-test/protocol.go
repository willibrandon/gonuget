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

// ParseAssetPathRequest parses a single asset path.
type ParseAssetPathRequest struct {
	Path string `json:"path"`
}

// ParseAssetPathResponse contains the parsed path properties.
type ParseAssetPathResponse struct {
	// Item is nil if path didn't match any pattern.
	Item *ContentItemData `json:"item"`
}
