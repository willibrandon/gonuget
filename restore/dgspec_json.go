package restore

import (
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// DgSpecHasher generates the exact JSON structure that NuGet.Client uses for dgSpecHash.
// Reference: NuGet.ProjectModel/DependencyGraphSpec.cs Write() method
type DgSpecHasher struct {
	proj                    *project.Project
	packagesPath            string
	fallbackFolders         []string
	sources                 []string
	configPaths             []string
	runtimeIDPath           string
	sdkAnalysisLevel        string
	downloadDependenciesMap map[string]map[string]string // tfm -> (name -> version)
}

// NewDgSpecHasher creates a hasher for a project.
func NewDgSpecHasher(proj *project.Project) *DgSpecHasher {
	return &DgSpecHasher{
		proj: proj,
	}
}

// WithPackagesPath sets the global packages folder path.
func (h *DgSpecHasher) WithPackagesPath(path string) *DgSpecHasher {
	h.packagesPath = path
	return h
}

// WithFallbackFolders sets the fallback package folders.
func (h *DgSpecHasher) WithFallbackFolders(folders []string) *DgSpecHasher {
	h.fallbackFolders = folders
	return h
}

// WithSources sets the package sources.
func (h *DgSpecHasher) WithSources(sources []string) *DgSpecHasher {
	h.sources = sources
	return h
}

// WithConfigPaths sets the NuGet.config file paths.
func (h *DgSpecHasher) WithConfigPaths(paths []string) *DgSpecHasher {
	h.configPaths = paths
	return h
}

// WithRuntimeIDPath sets the RuntimeIdentifierGraph.json path.
func (h *DgSpecHasher) WithRuntimeIDPath(path string) *DgSpecHasher {
	h.runtimeIDPath = path
	return h
}

// WithSdkAnalysisLevel sets the SDK analysis level.
func (h *DgSpecHasher) WithSdkAnalysisLevel(level string) *DgSpecHasher {
	h.sdkAnalysisLevel = level
	return h
}

// WithDownloadDependencies sets the download dependencies map.
func (h *DgSpecHasher) WithDownloadDependencies(deps map[string]map[string]string) *DgSpecHasher {
	h.downloadDependenciesMap = deps
	return h
}

// GenerateJSON generates the dgspec JSON for hashing.
// Matches DependencyGraphSpec.Write() with hashing: true.
//
// CRITICAL: Key order must match NuGet.Client exactly for hash compatibility.
// NuGet writes: format, restore, projects (NOT alphabetical).
// Go's json.Marshal sorts keys alphabetically, so we use OrderedJSONWriter.
func (h *DgSpecHasher) GenerateJSON() ([]byte, error) {
	writer := NewOrderedJSONWriter()
	writer.WriteDgSpec(h)
	return writer.Bytes(), nil
}
