package restore

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LockFile represents project.assets.json structure.
// Ported from NuGet.ProjectModel/LockFile/LockFile.cs
type LockFile struct {
	Version                     int                      `json:"version"`
	Targets                     map[string]Target        `json:"targets"`
	Libraries                   map[string]Library       `json:"libraries"`
	ProjectFileDependencyGroups map[string][]string      `json:"projectFileDependencyGroups"`
	PackageFolders              map[string]PackageFolder `json:"packageFolders"`
	Project                     ProjectInfo              `json:"project"`
}

// Target represents a target framework's dependency graph.
type Target struct {
	// Map of package ID to package info
	// Key format: "PackageID/Version"
}

// Library represents a package library entry.
type Library struct {
	Type  string   `json:"type"`
	Path  string   `json:"path,omitempty"`
	Files []string `json:"files,omitempty"`
}

// PackageFolder represents a package folder location.
type PackageFolder struct {
	// Empty struct for now
}

// ProjectInfo represents project metadata in lock file.
type ProjectInfo struct {
	Version    string                          `json:"version"`
	Restore    Info                            `json:"restore"`
	Frameworks map[string]ProjectFrameworkInfo `json:"frameworks"`
}

// Info represents restore metadata.
type Info struct {
	ProjectUniqueName        string                   `json:"projectUniqueName"`
	ProjectName              string                   `json:"projectName"`
	ProjectPath              string                   `json:"projectPath"`
	PackagesPath             string                   `json:"packagesPath"`
	OutputPath               string                   `json:"outputPath"`
	ProjectStyle             string                   `json:"projectStyle"`
	Sources                  map[string]SourceInfo    `json:"sources"`
	FallbackFolders          []string                 `json:"fallbackFolders"`
	ConfigFilePaths          []string                 `json:"configFilePaths"`
	OriginalTargetFrameworks []string                 `json:"originalTargetFrameworks"`
	Frameworks               map[string]FrameworkInfo `json:"frameworks"`
}

// SourceInfo represents a package source.
type SourceInfo struct {
	// Empty struct for now
}

// FrameworkInfo represents framework-specific restore info (project references and restore metadata).
// Named FrameworkInfo rather than RestoreFrameworkInfo to avoid package name stuttering.
// Distinct from ProjectFrameworkInfo which holds package dependencies.
type FrameworkInfo struct {
	TargetAlias       string         `json:"targetAlias"`
	ProjectReferences map[string]any `json:"projectReferences"`
}

// ProjectFrameworkInfo represents framework-specific project info (package dependencies).
// Named ProjectFrameworkInfo to distinguish from FrameworkInfo (restore metadata) and avoid conflicts.
type ProjectFrameworkInfo struct {
	TargetAlias  string                    `json:"targetAlias"`
	Dependencies map[string]DependencyInfo `json:"dependencies"`
}

// DependencyInfo represents a package dependency.
type DependencyInfo struct {
	Target  string `json:"target"`
	Version string `json:"version"`
}

// Save writes the lock file to disk.
func (lf *LockFile) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(path, data, 0644)
}
