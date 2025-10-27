package restore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
	"github.com/willibrandon/gonuget/core"
)

// Run executes the restore operation (entry point called from CLI).
func Run(ctx context.Context, args []string, opts *Options, console Console) error {
	start := time.Now()

	// 1. Find project file
	projectPath, err := findProjectFile(args)
	if err != nil {
		return err
	}

	console.Printf("Restoring packages for %s...\n", projectPath)

	// 2. Load project
	proj, err := project.LoadProject(projectPath)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	// 3. Get package references
	packageRefs := proj.GetPackageReferences()

	if len(packageRefs) == 0 {
		console.Printf("Nothing to restore\n")
		return nil
	}

	// 4. Create restorer
	restorer := NewRestorer(opts, console)

	// 5. Execute restore
	result, err := restorer.Restore(ctx, proj, packageRefs)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	// 6. Generate lock file (project.assets.json)
	lockFile := NewLockFileBuilder().Build(proj, result)
	objDir := filepath.Join(filepath.Dir(proj.Path), "obj")
	assetsPath := filepath.Join(objDir, "project.assets.json")
	if err := lockFile.Save(assetsPath); err != nil {
		return fmt.Errorf("failed to save project.assets.json: %w", err)
	}

	// 7. Report summary
	elapsed := time.Since(start)
	console.Printf("  Restored %s (in %d ms)\n", projectPath, elapsed.Milliseconds())

	return nil
}

func findProjectFile(args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return project.FindProjectFile(cwd)
}

// Restorer executes restore operations.
type Restorer struct {
	opts    *Options
	console Console
	client  *core.Client
}

// NewRestorer creates a new restorer.
func NewRestorer(opts *Options, console Console) *Restorer {
	// Create repository manager
	repoManager := core.NewRepositoryManager()

	// Add sources
	if len(opts.Sources) > 0 {
		for _, source := range opts.Sources {
			repo := core.NewSourceRepository(core.RepositoryConfig{
				SourceURL: source,
			})
			if err := repoManager.AddRepository(repo); err != nil {
				console.Warning(fmt.Sprintf("Failed to add repository %s: %v", source, err))
			}
		}
	}

	// Create client
	client := core.NewClient(core.ClientConfig{
		RepositoryManager: repoManager,
	})

	return &Restorer{
		opts:    opts,
		console: console,
		client:  client,
	}
}

// Result holds restore results.
type Result struct {
	Packages []PackageInfo
}

// PackageInfo holds package information.
type PackageInfo struct {
	ID      string
	Version string
	Path    string
}

// Restore executes the restore operation for direct dependencies only (Chunk 5 - simplified).
func (r *Restorer) Restore(ctx context.Context, proj *project.Project, packageRefs []project.PackageReference) (*Result, error) {
	result := &Result{
		Packages: make([]PackageInfo, 0, len(packageRefs)),
	}

	// Get global packages folder
	packagesFolder := r.opts.PackagesFolder
	if packagesFolder == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		packagesFolder = filepath.Join(home, ".nuget", "packages")
	}

	// Ensure packages folder exists
	if err := os.MkdirAll(packagesFolder, 0755); err != nil {
		return nil, fmt.Errorf("failed to create packages folder: %w", err)
	}

	// Restore each package reference
	for _, pkgRef := range packageRefs {
		r.console.Printf("  Restoring %s %s...\n", pkgRef.Include, pkgRef.Version)

		// Normalize package ID to lowercase for cross-platform consistency
		// This matches NuGet.Client's VersionFolderPathResolver behavior
		normalizedPackageID := strings.ToLower(pkgRef.Include)

		// Check if package already exists in cache
		packagePath := filepath.Join(packagesFolder, normalizedPackageID, pkgRef.Version)
		if !r.opts.Force {
			if _, err := os.Stat(packagePath); err == nil {
				r.console.Printf("    Package already cached at %s\n", packagePath)
				result.Packages = append(result.Packages, PackageInfo{
					ID:      pkgRef.Include,
					Version: pkgRef.Version,
					Path:    packagePath,
				})
				continue
			}
		}

		// Download package
		if err := r.downloadPackage(ctx, normalizedPackageID, pkgRef.Version, packagePath); err != nil {
			return nil, fmt.Errorf("failed to download package %s %s: %w", pkgRef.Include, pkgRef.Version, err)
		}

		result.Packages = append(result.Packages, PackageInfo{
			ID:      pkgRef.Include,
			Version: pkgRef.Version,
			Path:    packagePath,
		})
	}

	return result, nil
}

func (r *Restorer) downloadPackage(ctx context.Context, packageID, version, packagePath string) error {
	// Create package directory
	if err := os.MkdirAll(packagePath, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Download .nupkg
	nupkgPath := filepath.Join(packagePath, fmt.Sprintf("%s.%s.nupkg", packageID, version))
	packageReader, err := r.client.DownloadPackage(ctx, packageID, version)
	if err != nil {
		return fmt.Errorf("failed to download package: %w", err)
	}
	defer func() {
		if cerr := packageReader.Close(); cerr != nil {
			r.console.Error("failed to close package reader: %v\n", cerr)
		}
	}()

	// Save .nupkg file
	nupkgFile, err := os.Create(nupkgPath)
	if err != nil {
		return fmt.Errorf("failed to create .nupkg file: %w", err)
	}
	defer func() {
		if cerr := nupkgFile.Close(); cerr != nil {
			r.console.Error("failed to close .nupkg file: %v\n", cerr)
		}
	}()

	if _, err := nupkgFile.ReadFrom(packageReader); err != nil {
		return fmt.Errorf("failed to write .nupkg file: %w", err)
	}

	r.console.Printf("    Downloaded to %s\n", nupkgPath)

	// TODO: Extract package contents (deferred to future chunk)
	// For Chunk 5, we just download the .nupkg file

	return nil
}
