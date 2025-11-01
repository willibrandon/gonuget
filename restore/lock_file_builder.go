package restore

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/packaging"
	"github.com/willibrandon/gonuget/packaging/assets"
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

	// Get all packages (direct + transitive) - needed for both Libraries and Targets
	// Matches NuGet.Client BuildAssetsFile line 265
	allPackages := result.AllPackages()

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

		// Parse framework
		framework, err := frameworks.ParseFramework(tfm)
		if err != nil {
			// If we can't parse framework, create empty target
			lf.Targets[tfm] = Target{}
			continue
		}

		// Add target for this framework with populated assemblies
		// For multi-TFM, each framework gets its own target section
		target := make(Target)

		// Populate assemblies for each package
		for _, pkg := range allPackages {
			targetLib := b.createTargetLibrary(pkg, framework, packagesPath)
			if targetLib != nil {
				key := pkg.ID + "/" + pkg.Version
				target[key] = *targetLib
			}
		}

		lf.Targets[tfm] = target
	}

	// Add global ProjectFileDependencyGroups entry (for all frameworks)
	lf.ProjectFileDependencyGroups[""] = dependencies

	// Add libraries (direct + transitive) - this is the UNION across all frameworks
	// Matches NuGet.Client BuildAssetsFile line 265
	for _, pkg := range allPackages {
		key := pkg.ID + "/" + pkg.Version
		// NuGet.Client uses lowercase package ID in path for cross-platform compatibility
		// Format: "packageid/version" (e.g., "newtonsoft.json/13.0.3")
		relativePath := strings.ToLower(pkg.ID) + "/" + pkg.Version
		lf.Libraries[key] = Library{
			Type:  "package",
			Path:  relativePath,
			Files: []string{},
		}
	}

	return lf
}

// createTargetLibrary creates a TargetLibrary with compile and runtime assemblies for a package.
// Matches NuGet.Client's LockFileUtils.CreateLockFileTargetLibrary.
func (b *LockFileBuilder) createTargetLibrary(
	pkg PackageInfo,
	framework *frameworks.NuGetFramework,
	packagesPath string,
) *TargetLibrary {
	// Build package path
	pkgPath := filepath.Join(packagesPath, strings.ToLower(pkg.ID), pkg.Version)
	nupkgPath := filepath.Join(pkgPath, strings.ToLower(pkg.ID)+"."+pkg.Version+".nupkg")

	// Check if package exists
	if _, err := os.Stat(nupkgPath); os.IsNotExist(err) {
		// Package not downloaded yet - return empty target library
		return &TargetLibrary{
			Type:    "package",
			Compile: make(map[string]map[string]string),
			Runtime: make(map[string]map[string]string),
		}
	}

	// Open package
	reader, err := packaging.OpenPackage(nupkgPath)
	if err != nil {
		// Can't read package - return empty target library
		return &TargetLibrary{
			Type:    "package",
			Compile: make(map[string]map[string]string),
			Runtime: make(map[string]map[string]string),
		}
	}
	defer func() { _ = reader.Close() }()

	// Get all files from package
	files := reader.GetFiles("")
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Name)
	}

	// Create content item collection from package files
	collection := assets.NewContentItemCollection(paths)

	// Create managed code conventions for asset selection
	conventions := assets.NewManagedCodeConventions()

	// Create selection criteria for this framework
	criteria := assets.ForFramework(framework, conventions.Properties)

	targetLib := &TargetLibrary{
		Type:    "package",
		Compile: make(map[string]map[string]string),
		Runtime: make(map[string]map[string]string),
	}

	// Select compile assemblies (ref/ takes precedence over lib/)
	compileGroup := collection.FindBestItemGroup(criteria, conventions.CompileRefAssemblies, conventions.CompileLibAssemblies)
	if compileGroup != nil {
		for _, item := range compileGroup.Items {
			// Add with empty metadata (related property would go here if we parsed it)
			targetLib.Compile[item.Path] = map[string]string{"related": ".xml"}
		}
	}

	// Select runtime assemblies (lib/ folder)
	runtimeGroup := collection.FindBestItemGroup(criteria, conventions.RuntimeAssemblies)
	if runtimeGroup != nil {
		for _, item := range runtimeGroup.Items {
			// Add with empty metadata
			targetLib.Runtime[item.Path] = map[string]string{"related": ".xml"}
		}
	}

	return targetLib
}
