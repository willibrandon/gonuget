package packaging

import (
	"path"
	"strings"
)

// Common package folder constants
const (
	LibFolder             = "lib/"
	RefFolder             = "ref/"
	RuntimesFolder        = "runtimes/"
	ContentFolder         = "content/"
	ContentFilesFolder    = "contentFiles/"
	BuildFolder           = "build/"
	BuildTransitiveFolder = "buildTransitive/"
	ToolsFolder           = "tools/"
	NativeFolder          = "native/"
	AnalyzersFolder       = "analyzers/"
	EmbedFolder           = "embed/"
)

// Package metadata files
const (
	ManifestExtension       = ".nuspec"
	SignatureFile           = ".signature.p7s"
	PackageRelationshipFile = "_rels/.rels"
	ContentTypesFile        = "[Content_Types].xml"
	PSMDCPFile              = "package/services/metadata/core-properties/"
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
	return strings.HasPrefix(lower, strings.ToLower(ContentFolder)) ||
		strings.HasPrefix(lower, strings.ToLower(ContentFilesFolder))
}

// IsBuildFile checks if a file is in the build/ folder
func IsBuildFile(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.HasPrefix(lower, strings.ToLower(BuildFolder)) ||
		strings.HasPrefix(lower, strings.ToLower(BuildTransitiveFolder))
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

// IsManifestFile checks if a file is a .nuspec manifest
func IsManifestFile(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ManifestExtension)
}

// IsPackageMetadataFile checks if a file is package metadata
func IsPackageMetadataFile(filePath string) bool {
	lower := strings.ToLower(filePath)
	return lower == SignatureFile ||
		strings.HasPrefix(lower, "_rels/") ||
		lower == strings.ToLower(ContentTypesFile) ||
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
