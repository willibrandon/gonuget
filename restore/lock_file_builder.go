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

	// Get target framework
	tfm := proj.TargetFramework
	if tfm == "" && len(proj.TargetFrameworks) > 0 {
		tfm = proj.TargetFrameworks[0]
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
				OriginalTargetFrameworks: []string{tfm},
				Frameworks: map[string]FrameworkInfo{
					tfm: {
						TargetAlias:       tfm,
						ProjectReferences: make(map[string]any),
					},
				},
			},
			Frameworks: map[string]ProjectFrameworkInfo{
				tfm: {
					TargetAlias:  tfm,
					Dependencies: make(map[string]DependencyInfo),
				},
			},
		},
	}

	// Add package references to project dependencies
	packageRefs := proj.GetPackageReferences()
	dependencies := make([]string, 0, len(packageRefs))
	for _, pkgRef := range packageRefs {
		dependencies = append(dependencies, pkgRef.Include+" >= "+pkgRef.Version)
		lf.Project.Frameworks[tfm].Dependencies[pkgRef.Include] = DependencyInfo{
			Target:  "Package",
			Version: pkgRef.Version,
		}
	}
	lf.ProjectFileDependencyGroups[tfm] = dependencies
	lf.ProjectFileDependencyGroups[""] = dependencies

	// Add libraries (direct + transitive)
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

	// Add target
	lf.Targets[tfm] = Target{}

	return lf
}
