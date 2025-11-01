package restore

import (
	"os"
	"path/filepath"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// LockFileBuilder builds project.assets.json from restore results.
// Ported from NuGet.Commands/RestoreCommand/LockFileBuilder.cs
type LockFileBuilder struct {
}

// NewLockFileBuilder creates a new lock file builder.
func NewLockFileBuilder() *LockFileBuilder {
	return &LockFileBuilder{}
}

// Build creates a LockFile from project and restore results.
func (b *LockFileBuilder) Build(proj *project.Project, result *Result) *LockFile {
	// Get packages folder
	home, _ := os.UserHomeDir()
	packagesPath := filepath.Join(home, ".nuget", "packages")

	// Get all target frameworks
	targetFrameworks := proj.GetTargetFrameworks()
	if len(targetFrameworks) == 0 {
		// Fallback to single TFM if none found
		if proj.TargetFramework != "" {
			targetFrameworks = []string{proj.TargetFramework}
		} else if len(proj.TargetFrameworks) > 0 {
			targetFrameworks = proj.TargetFrameworks
		}
	}

	// Build lock file
	lf := &LockFile{
		Version:                     3,
		Targets:                     make(map[string]Target),
		Libraries:                   make(map[string]Library),
		ProjectFileDependencyGroups: make(map[string][]string),
		PackageFolders: map[string]PackageFolder{
			packagesPath: {},
		},
		Project: ProjectInfo{
			Version: "1.0.0",
			Restore: Info{
				ProjectUniqueName:        proj.Path,
				ProjectName:              filepath.Base(proj.Path),
				ProjectPath:              proj.Path,
				PackagesPath:             packagesPath,
				OutputPath:               filepath.Join(filepath.Dir(proj.Path), "obj"),
				ProjectStyle:             "PackageReference",
				Sources:                  make(map[string]SourceInfo),
				FallbackFolders:          []string{},
				ConfigFilePaths:          []string{},
				OriginalTargetFrameworks: targetFrameworks,
				Frameworks:               make(map[string]FrameworkInfo),
			},
			Frameworks: make(map[string]ProjectFrameworkInfo),
		},
	}

	// Get package references once
	packageRefs := proj.GetPackageReferences()

	// Build dependencies list for ProjectFileDependencyGroups
	dependencies := make([]string, 0, len(packageRefs))
	for _, pkgRef := range packageRefs {
		dependencies = append(dependencies, pkgRef.Include+" >= "+pkgRef.Version)
	}

	// Add entries for each target framework
	for _, tfm := range targetFrameworks {
		// Add to Restore.Frameworks
		lf.Project.Restore.Frameworks[tfm] = FrameworkInfo{
			TargetAlias:       tfm,
			ProjectReferences: make(map[string]any),
		}

		// Add to Project.Frameworks
		frameworkDeps := make(map[string]DependencyInfo)
		for _, pkgRef := range packageRefs {
			frameworkDeps[pkgRef.Include] = DependencyInfo{
				Target:  "Package",
				Version: pkgRef.Version,
			}
		}
		lf.Project.Frameworks[tfm] = ProjectFrameworkInfo{
			TargetAlias:  tfm,
			Dependencies: frameworkDeps,
		}

		// Add to ProjectFileDependencyGroups (per-framework)
		lf.ProjectFileDependencyGroups[tfm] = dependencies

		// Add target for this framework
		// For multi-TFM, each framework gets its own target section
		lf.Targets[tfm] = Target{}
	}

	// Add global ProjectFileDependencyGroups entry (for all frameworks)
	lf.ProjectFileDependencyGroups[""] = dependencies

	// Add libraries (direct + transitive) - this is the UNION across all frameworks
	// Matches NuGet.Client BuildAssetsFile line 265
	allPackages := result.AllPackages()
	for _, pkg := range allPackages {
		key := pkg.ID + "/" + pkg.Version
		lf.Libraries[key] = Library{
			Type:  "package",
			Path:  pkg.Path,
			Files: []string{},
		}
	}

	return lf
}
