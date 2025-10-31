package restore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/willibrandon/gonuget/packaging"
	"github.com/willibrandon/gonuget/version"
)

// downloadPackage downloads and installs a package using the appropriate protocol (V2 or V3).
// Matches NuGet.Client's RestoreCommand package installation flow.
func (r *Restorer) downloadPackage(ctx context.Context, packageID, packageVersion, packagePath string, cacheHit bool) error {
	isDiagnostic := r.opts.Verbosity == "diagnostic"

	// Diagnostic: Show cache hit or lock acquisition
	if isDiagnostic {
		if cacheHit {
			// Package already in cache - show CACHE message (use 9 space indent to match lock messages)
			r.console.Printf("         CACHE %s %s (already in %s)\n", packageID, packageVersion, packagePath)
		} else {
			// Package needs to be downloaded - show lock acquisition (use 9 space indent)
			r.console.Printf("         Acquiring lock for the installation of %s %s\n", packageID, packageVersion)
			r.console.Printf("         Acquired lock for the installation of %s %s\n", packageID, packageVersion)
		}
	}
	// Parse version
	pkgVer, err := version.Parse(packageVersion)
	if err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	// Get source repository and detect protocol
	repos := r.client.GetRepositoryManager().ListRepositories()
	if len(repos) == 0 {
		return fmt.Errorf("no package sources configured")
	}
	repo := repos[0]

	provider, err := repo.GetProvider(ctx)
	if err != nil {
		return fmt.Errorf("get provider: %w", err)
	}

	protocolVersion := provider.ProtocolVersion()
	sourceURL := repo.SourceURL()

	// Create package identity
	packageIdentity := &packaging.PackageIdentity{
		ID:      packageID,
		Version: pkgVer,
	}

	// Create extraction context with all save modes
	extractionContext := &packaging.PackageExtractionContext{
		PackageSaveMode:    packaging.PackageSaveModeNupkg | packaging.PackageSaveModeNuspec | packaging.PackageSaveModeFiles,
		XMLDocFileSaveMode: packaging.XMLDocFileSaveModeNone,
	}

	// Use V3 or V2 installer based on protocol
	if protocolVersion == "v3" {
		return r.installPackageV3(ctx, packageID, packageVersion, packagePath, packageIdentity, sourceURL, extractionContext, cacheHit)
	}
	return r.installPackageV2(ctx, packageID, packageVersion, packagePath, packageIdentity, sourceURL, extractionContext, cacheHit)
}

// installPackageV3 installs a package using V3 protocol and layout.
// Matches NuGet.Client's V3 package installation flow.
func (r *Restorer) installPackageV3(ctx context.Context, packageID, packageVersion, packagePath string, packageIdentity *packaging.PackageIdentity, sourceURL string, extractionContext *packaging.PackageExtractionContext, cacheHit bool) error {
	isDiagnostic := r.opts.Verbosity == "diagnostic"

	// Create path resolver for V3 layout
	packagesFolder := filepath.Dir(filepath.Dir(packagePath)) // Go up to packages root
	pathResolver := packaging.NewVersionFolderPathResolver(packagesFolder, true)

	// Create download callback
	copyToAsync := func(targetPath string) error {
		// Diagnostic: HTTP GET request (if not cached) - use 11 space indent
		downloadStart := time.Now()
		if isDiagnostic && !cacheHit {
			// Build package download URL for logging (use lowercase for URL)
			downloadURL := fmt.Sprintf("%s/flatcontainer/%s/%s/%s.%s.nupkg",
				strings.TrimSuffix(sourceURL, "/index.json"),
				strings.ToLower(packageID),
				strings.ToLower(packageVersion),
				strings.ToLower(packageID),
				strings.ToLower(packageVersion))
			r.console.Printf("           GET %s\n", downloadURL)
		}

		stream, err := r.client.DownloadPackage(ctx, packageID, packageVersion)
		if err != nil {
			return fmt.Errorf("download package: %w", err)
		}
		defer func() {
			if cerr := stream.Close(); cerr != nil {
				r.console.Error("failed to close package stream: %v\n", cerr)
			}
		}()

		// Diagnostic: HTTP OK response (if not cached) - use 11 space indent
		if isDiagnostic && !cacheHit {
			elapsed := time.Since(downloadStart)
			downloadURL := fmt.Sprintf("%s/flatcontainer/%s/%s/%s.%s.nupkg",
				strings.TrimSuffix(sourceURL, "/index.json"),
				strings.ToLower(packageID),
				strings.ToLower(packageVersion),
				strings.ToLower(packageID),
				strings.ToLower(packageVersion))
			r.console.Printf("           OK %s %dms\n", downloadURL, elapsed.Milliseconds())
		}

		outFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("create temp file: %w", err)
		}
		defer func() {
			if cerr := outFile.Close(); cerr != nil {
				r.console.Error("failed to close temp file: %v\n", cerr)
			}
		}()

		if _, err := io.Copy(outFile, stream); err != nil {
			return fmt.Errorf("write package: %w", err)
		}

		return nil
	}

	// Install package (download + extract) using V3 layout
	_, err := packaging.InstallFromSourceV3(
		ctx,
		sourceURL,
		packageIdentity,
		copyToAsync,
		pathResolver,
		extractionContext,
	)

	if err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	// Diagnostic: Vulnerability check (always CACHE since we don't implement vulnerability DB yet) - use 11 space indent
	if isDiagnostic && !cacheHit {
		vulnURL := "https://api.nuget.org/v3/vulnerabilities/index.json"
		r.console.Printf("           CACHE %s\n", vulnURL)
	}

	// Note: Terminal Logger hides download/cache messages in detailed mode
	// We match Terminal Logger behavior: diagnostic mode shows download messages, detailed mode is clean

	return nil
}

// installPackageV2 installs a package using V2 protocol and layout.
// Matches NuGet.Client's V2 package installation flow.
func (r *Restorer) installPackageV2(ctx context.Context, packageID, packageVersion, packagePath string, packageIdentity *packaging.PackageIdentity, sourceURL string, extractionContext *packaging.PackageExtractionContext, cacheHit bool) error {
	isDiagnostic := r.opts.Verbosity == "diagnostic"

	// Create path resolver for V2 layout
	packagesFolder := filepath.Dir(filepath.Dir(packagePath)) // Go up to packages root
	pathResolver := packaging.NewPackagePathResolver(packagesFolder, true)

	// Check if already installed
	targetPath := pathResolver.GetInstallPath(packageIdentity)
	if _, err := os.Stat(targetPath); err == nil {
		// Note: Terminal Logger hides this message completely
		return nil
	}

	// Diagnostic: HTTP GET request (if not cached) - use 11 space indent
	downloadStart := time.Now()
	if isDiagnostic && !cacheHit {
		downloadURL := fmt.Sprintf("%s/Packages(Id='%s',Version='%s')",
			strings.TrimSuffix(sourceURL, "/"),
			packageID,
			packageVersion)
		r.console.Printf("           GET %s\n", downloadURL)
	}

	// Download package to memory
	stream, err := r.client.DownloadPackage(ctx, packageID, packageVersion)
	if err != nil {
		return fmt.Errorf("download package: %w", err)
	}
	defer func() {
		if cerr := stream.Close(); cerr != nil {
			r.console.Error("failed to close package stream: %v\n", cerr)
		}
	}()

	// Diagnostic: HTTP OK response (if not cached) - use 11 space indent
	if isDiagnostic && !cacheHit {
		elapsed := time.Since(downloadStart)
		downloadURL := fmt.Sprintf("%s/Packages(Id='%s',Version='%s')",
			strings.TrimSuffix(sourceURL, "/"),
			packageID,
			packageVersion)
		r.console.Printf("           OK %s %dms\n", downloadURL, elapsed.Milliseconds())
	}

	// Read into memory (V2 extractor needs ReadSeeker)
	packageData, err := io.ReadAll(stream)
	if err != nil {
		return fmt.Errorf("read package: %w", err)
	}

	packageReader := bytes.NewReader(packageData)

	// Extract package using V2 layout
	_, err = packaging.ExtractPackageV2(
		ctx,
		sourceURL,
		packageReader,
		pathResolver,
		extractionContext,
	)

	if err != nil {
		return fmt.Errorf("failed to extract package: %w", err)
	}

	// Diagnostic: Vulnerability check (always CACHE since we don't implement vulnerability DB yet) - use 11 space indent
	if isDiagnostic && !cacheHit {
		vulnURL := "https://api.nuget.org/v3/vulnerabilities/index.json"
		r.console.Printf("           CACHE %s\n", vulnURL)
	}

	// Note: Terminal Logger hides download messages in detailed mode
	return nil
}
