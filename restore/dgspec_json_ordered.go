package restore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// OrderedJSONWriter writes JSON with exact field order matching NuGet.Client.
// This is CRITICAL for hash compatibility - different field order = different hash.
type OrderedJSONWriter struct {
	buf *bytes.Buffer
}

// NewOrderedJSONWriter creates a new ordered JSON writer.
func NewOrderedJSONWriter() *OrderedJSONWriter {
	return &OrderedJSONWriter{
		buf: &bytes.Buffer{},
	}
}

// Bytes returns the generated JSON bytes.
func (w *OrderedJSONWriter) Bytes() []byte {
	return w.buf.Bytes()
}

// writeString writes a raw string to the buffer.
func (w *OrderedJSONWriter) writeString(s string) {
	w.buf.WriteString(s)
}

// writeEscapedString writes a JSON-escaped string.
func (w *OrderedJSONWriter) writeEscapedString(s string) {
	// Use json.Marshal to get proper escaping
	b, _ := json.Marshal(s)
	w.buf.Write(b)
}

// writeStringField writes a string field: "key":"value"
func (w *OrderedJSONWriter) writeStringField(key, value string) {
	w.writeEscapedString(key)
	w.writeString(":")
	w.writeEscapedString(value)
}

// writeIntField writes an integer field: "key":123
func (w *OrderedJSONWriter) writeIntField(key string, value int) {
	w.writeEscapedString(key)
	w.writeString(":")
	w.writeString(fmt.Sprintf("%d", value))
}

// writeBoolField writes a boolean field: "key":true
func (w *OrderedJSONWriter) writeBoolField(key string, value bool) {
	w.writeEscapedString(key)
	w.writeString(":")
	if value {
		w.writeString("true")
	} else {
		w.writeString("false")
	}
}

// writeArrayField writes a string array field: "key":["a","b"]
func (w *OrderedJSONWriter) writeArrayField(key string, values []string) {
	w.writeEscapedString(key)
	w.writeString(":[")
	for i, v := range values {
		w.writeEscapedString(v)
		if i < len(values)-1 {
			w.writeString(",")
		}
	}
	w.writeString("]")
}

// WriteDgSpec writes the complete DgSpec JSON matching NuGet.Client's exact structure.
// Reference: DependencyGraphSpec.cs Write() method (lines 359-389)
func (w *OrderedJSONWriter) WriteDgSpec(hasher *DgSpecHasher) {
	proj := hasher.proj
	projectPath := proj.Path

	w.writeString("{")

	// 1. format (line 362)
	w.writeIntField("format", 1)

	// 2. restore (lines 364-373)
	w.writeString(",")
	w.writeEscapedString("restore")
	w.writeString(":{")
	w.writeEscapedString(projectPath)
	w.writeString(":{}")
	w.writeString("}")

	// 3. projects (lines 375-388)
	w.writeString(",")
	w.writeEscapedString("projects")
	w.writeString(":{")
	w.writeEscapedString(projectPath)
	w.writeString(":")

	// Write PackageSpec
	w.writePackageSpec(hasher)

	w.writeString("}")

	w.writeString("}")
}

// writePackageSpec writes a PackageSpec object.
// Reference: PackageSpecWriter.cs Write() method (lines 35-57)
func (w *OrderedJSONWriter) writePackageSpec(hasher *DgSpecHasher) {
	w.writeString("{")

	// 1. version (lines 47-50) - always write for SDK-style projects
	w.writeStringField("version", "1.0.0")

	// 2. restore metadata (line 52)
	w.writeString(",")
	w.writeRestoreMetadata(hasher)

	// 3. frameworks (line 54)
	w.writeString(",")
	w.writeFrameworks(hasher)

	// 4. RuntimeGraph (line 56) - usually omitted, skip for now

	w.writeString("}")
}

// writeRestoreMetadata writes the restore metadata object.
// Reference: PackageSpecWriter.cs SetMSBuildMetadata() (lines 111-177)
func (w *OrderedJSONWriter) writeRestoreMetadata(hasher *DgSpecHasher) {
	proj := hasher.proj
	projectPath := proj.Path

	w.writeEscapedString("restore")
	w.writeString(":{")

	// 1. projectUniqueName (line 125)
	w.writeStringField("projectUniqueName", projectPath)

	// 2. projectName (line 126)
	// Use filepath.Base for cross-platform compatibility (Windows uses backslashes)
	projectName := strings.TrimSuffix(filepath.Base(projectPath), ".csproj")
	w.writeString(",")
	w.writeStringField("projectName", projectName)

	// 3. projectPath (line 127)
	w.writeString(",")
	w.writeStringField("projectPath", projectPath)

	// 4. projectJsonPath (line 128) - skip if empty

	// 5. packagesPath (line 129)
	if hasher.packagesPath != "" {
		w.writeString(",")
		packagesPath := hasher.packagesPath
		// Add trailing separator (native to platform)
		if !strings.HasSuffix(packagesPath, string(filepath.Separator)) {
			packagesPath += string(filepath.Separator)
		}
		w.writeStringField("packagesPath", packagesPath)
	}

	// 6. outputPath (line 130)
	// Use native path separators to match dotnet's behavior
	objDir := filepath.Join(filepath.Dir(projectPath), "obj") + string(filepath.Separator)
	w.writeString(",")
	w.writeStringField("outputPath", objDir)

	// 7. projectStyle (lines 132-135)
	w.writeString(",")
	w.writeStringField("projectStyle", "PackageReference")

	// 8. Booleans (line 137) - WriteMetadataBooleans - skip if all false

	// 9. fallbackFolders (lines 146-153)
	if len(hasher.fallbackFolders) > 0 {
		w.writeString(",")
		w.writeArrayField("fallbackFolders", hasher.fallbackFolders)
	}

	// 10. configFilePaths (lines 146-153)
	if len(hasher.configPaths) > 0 {
		w.writeString(",")
		w.writeArrayField("configFilePaths", hasher.configPaths)
	}

	// 11. originalTargetFrameworks (line 156)
	frameworks := proj.GetTargetFrameworks()
	if len(frameworks) > 0 {
		sorted := make([]string, len(frameworks))
		copy(sorted, frameworks)
		sort.Strings(sorted) // OrderBy StringComparer.Ordinal
		w.writeString(",")
		w.writeArrayField("originalTargetFrameworks", sorted)
	}

	// 12. sources (line 158) - WriteMetadataSources
	if len(hasher.sources) > 0 {
		w.writeString(",")
		w.writeEscapedString("sources")
		w.writeString(":{")

		// Sort sources
		sortedSources := make([]string, len(hasher.sources))
		copy(sortedSources, hasher.sources)
		sort.Strings(sortedSources)

		for i, src := range sortedSources {
			w.writeEscapedString(src)
			w.writeString(":{}")
			if i < len(sortedSources)-1 {
				w.writeString(",")
			}
		}
		w.writeString("}")
	}

	// 13. files (line 159) - WriteMetadataFiles - skip for now

	// 14. frameworks (line 160) - WriteMetadataTargetFrameworks
	w.writeString(",")
	w.writeRestoreFrameworks(hasher)

	// 15. warningProperties (line 161) - SetWarningProperties
	w.writeString(",")
	w.writeWarningProperties()

	// 16. restoreLockProperties (line 163) - skip if empty

	// 17. restoreAuditProperties (line 164)
	w.writeString(",")
	w.writeRestoreAuditProperties()

	// 18. packagesConfigPath (lines 166-169) - skip for PackageReference

	// 19. SdkAnalysisLevel (lines 171-174)
	if hasher.sdkAnalysisLevel != "" {
		w.writeString(",")
		w.writeStringField("SdkAnalysisLevel", hasher.sdkAnalysisLevel) // Last field
	}

	w.writeString("}")
}

// writeRestoreFrameworks writes the frameworks object in restore metadata.
// Reference: PackageSpecWriter.cs WriteMetadataTargetFrameworks() (lines 245-312)
func (w *OrderedJSONWriter) writeRestoreFrameworks(hasher *DgSpecHasher) {
	frameworks := hasher.proj.GetTargetFrameworks()

	w.writeEscapedString("frameworks")
	w.writeString(":{")

	for i, tfm := range frameworks {
		w.writeEscapedString(tfm)
		w.writeString(":{")

		// targetAlias (line 263)
		w.writeStringField("targetAlias", tfm)

		// projectReferences (lines 265-297) - empty for our case
		w.writeString(",")
		w.writeEscapedString("projectReferences")
		w.writeString(":{}")

		w.writeString("}")

		if i < len(frameworks)-1 {
			w.writeString(",")
		}
	}

	w.writeString("}")
}

// writeWarningProperties writes warning properties.
// Reference: PackageSpecWriter.cs SetWarningProperties() (lines 331-383)
func (w *OrderedJSONWriter) writeWarningProperties() {
	w.writeEscapedString("warningProperties")
	w.writeString(":{")

	// For default .NET projects, warnAsError includes NU1605
	w.writeArrayField("warnAsError", []string{"NU1605"})

	w.writeString("}")
}

// writeRestoreAuditProperties writes restore audit properties.
// Reference: PackageSpecWriter.cs WriteNuGetAuditProperties() (lines 220-243)
func (w *OrderedJSONWriter) writeRestoreAuditProperties() {
	w.writeEscapedString("restoreAuditProperties")
	w.writeString(":{")

	// Order from WriteNuGetAuditProperties: enableAudit, auditLevel, auditMode
	w.writeStringField("enableAudit", "true")
	w.writeString(",")
	w.writeStringField("auditLevel", "low")
	w.writeString(",")
	w.writeStringField("auditMode", "direct")

	w.writeString("}")
}

// writeFrameworks writes the frameworks object.
// Reference: PackageSpecWriter.cs SetFrameworks() (lines 543-569)
func (w *OrderedJSONWriter) writeFrameworks(hasher *DgSpecHasher) {
	frameworks := hasher.proj.GetTargetFrameworks()

	w.writeEscapedString("frameworks")
	w.writeString(":{")

	// Sort frameworks (line 549 - OrderBy with NuGetFrameworkSorter)
	// For simplicity, use alphabetical sort
	sorted := make([]string, len(frameworks))
	copy(sorted, frameworks)
	sort.Strings(sorted)

	for i, tfm := range sorted {
		w.writeEscapedString(tfm)
		w.writeString(":{")

		w.writeFrameworkInfo(hasher, tfm)

		w.writeString("}")

		if i < len(sorted)-1 {
			w.writeString(",")
		}
	}

	w.writeString("}")
}

// writeFrameworkInfo writes a single framework's information.
// Reference: PackageSpecWriter.cs SetFrameworks() lines 551-564
func (w *OrderedJSONWriter) writeFrameworkInfo(hasher *DgSpecHasher, tfm string) {
	// Field order from SetFrameworks:
	// 1. targetAlias (line 552)
	// 2. dependencies (line 553)
	// 3. centralPackageVersions (line 554) - skip if empty
	// 4. imports (line 555)
	// 5. assetTargetFallback (line 556)
	// 6. secondaryFramework (line 557) - skip
	// 7. warn (line 559)
	// 8. downloadDependencies (line 560) - SKIP (not included in dgspec for hash)
	// 9. frameworkReferences (line 561)
	// 10. runtimeIdentifierGraphPath (line 562)
	// 11. packagesToPrune (line 563) - skip if empty

	w.writeStringField("targetAlias", tfm)

	// dependencies
	packageRefs := hasher.proj.GetPackageReferences()
	if len(packageRefs) > 0 {
		w.writeString(",")
		w.writeEscapedString("dependencies")
		w.writeString(":{")

		// Sort by package ID
		sorted := make([]project.PackageReference, len(packageRefs))
		copy(sorted, packageRefs)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Include < sorted[j].Include
		})

		for i, pkg := range sorted {
			w.writeEscapedString(pkg.Include)
			w.writeString(":{")
			w.writeStringField("target", "Package")
			w.writeString(",")

			// Normalize version to range format
			version := pkg.Version
			if !strings.HasPrefix(version, "[") && !strings.HasPrefix(version, "(") {
				version = "[" + version + ", )"
			}
			w.writeStringField("version", version)
			w.writeString("}")

			if i < len(sorted)-1 {
				w.writeString(",")
			}
		}
		w.writeString("}")
	}

	// For .NET 6+, add imports, assetTargetFallback, warn, etc.
	if strings.HasPrefix(tfm, "net6") || strings.HasPrefix(tfm, "net7") ||
		strings.HasPrefix(tfm, "net8") || strings.HasPrefix(tfm, "net9") {

		// imports
		w.writeString(",")
		imports := []string{"net461", "net462", "net47", "net471", "net472", "net48", "net481"}
		w.writeArrayField("imports", imports)

		// assetTargetFallback
		w.writeString(",")
		w.writeBoolField("assetTargetFallback", true)

		// warn
		w.writeString(",")
		w.writeBoolField("warn", true)

		// downloadDependencies (framework reference packs)
		w.writeString(",")
		w.writeDownloadDependencies(hasher, tfm)

		// frameworkReferences
		w.writeString(",")
		w.writeFrameworkReferences()

		// runtimeIdentifierGraphPath
		if hasher.runtimeIDPath != "" {
			w.writeString(",")
			w.writeStringField("runtimeIdentifierGraphPath", hasher.runtimeIDPath)
		}
	}
}

// writeDownloadDependencies writes download dependencies (framework reference packs).
// These are automatically added by the SDK based on the target framework.
// Reference: Microsoft.NET.Sdk targets add KnownFrameworkReference items
func (w *OrderedJSONWriter) writeDownloadDependencies(hasher *DgSpecHasher, tfm string) {
	// Get download dependencies from hasher
	if hasher.downloadDependenciesMap == nil {
		return
	}

	deps, ok := hasher.downloadDependenciesMap[tfm]
	if !ok || len(deps) == 0 {
		return
	}

	// Write downloadDependencies array
	w.writeEscapedString("downloadDependencies")
	w.writeString(":[")

	// Sort keys for consistent order (dotnet sorts by name)
	names := make([]string, 0, len(deps))
	for name := range deps {
		names = append(names, name)
	}
	sort.Strings(names)

	for i, name := range names {
		version := deps[name]

		w.writeString("{")
		w.writeStringField("name", name)
		w.writeString(",")
		w.writeStringField("version", version)
		w.writeString("}")

		if i < len(names)-1 {
			w.writeString(",")
		}
	}

	w.writeString("]")
}

// writeFrameworkReferences writes framework references.
func (w *OrderedJSONWriter) writeFrameworkReferences() {
	w.writeEscapedString("frameworkReferences")
	w.writeString(":{")
	w.writeEscapedString("Microsoft.NETCore.App")
	w.writeString(":{")
	w.writeStringField("privateAssets", "all")
	w.writeString("}}")
}
