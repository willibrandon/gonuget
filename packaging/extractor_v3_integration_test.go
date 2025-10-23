package packaging

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/version"
)

// TestInstallFromSourceV3_Integration tests V3 installation with atomic operations
func TestInstallFromSourceV3_Integration(t *testing.T) {
	packagePath := "testdata/nuget.versioning.5.0.0.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	globalPackages := filepath.Join(tempDir, "global-packages")

	// Open package to get identity
	pkg, err := OpenPackage(packagePath)
	if err != nil {
		t.Fatalf("OpenPackage() error = %v", err)
	}
	identity, err := pkg.GetIdentity()
	_ = pkg.Close()
	if err != nil {
		t.Fatalf("GetIdentity() error = %v", err)
	}

	// Create V3 path resolver
	resolver := NewVersionFolderPathResolver(globalPackages, true)

	// Create extraction context
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV3,
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		Logger:             nil,
	}

	// Create copyToAsync callback that copies the package file
	copyToAsync := func(targetPath string) error {
		src, err := os.Open(packagePath)
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()

		dst, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer func() { _ = dst.Close() }()

		_, err = io.Copy(dst, src)
		return err
	}

	// Install package
	installed, err := InstallFromSourceV3(context.Background(), packagePath, identity, copyToAsync, resolver, ctx)
	if err != nil {
		t.Fatalf("InstallFromSourceV3() error = %v", err)
	}

	if !installed {
		t.Error("InstallFromSourceV3() returned false, expected true for first install")
	}

	// Verify installation
	packageDir := resolver.GetPackageDirectory(identity.ID, identity.Version)

	// Verify .nupkg file
	nupkgPath := resolver.GetPackageFilePath(identity.ID, identity.Version)
	if _, err := os.Stat(nupkgPath); os.IsNotExist(err) {
		t.Errorf("Expected .nupkg file not found: %s", nupkgPath)
	}

	// Verify .nuspec file
	nuspecPath := resolver.GetManifestFilePath(identity.ID, identity.Version)
	if _, err := os.Stat(nuspecPath); os.IsNotExist(err) {
		t.Errorf("Expected .nuspec file not found: %s", nuspecPath)
	}

	// Verify .nupkg.metadata file
	metadataPath := resolver.GetNupkgMetadataPath(identity.ID, identity.Version)
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Errorf("Expected .nupkg.metadata file not found: %s", metadataPath)
	}

	// Verify .sha512 hash file
	hashPath := resolver.GetHashPath(identity.ID, identity.Version)
	if _, err := os.Stat(hashPath); os.IsNotExist(err) {
		t.Errorf("Expected .sha512 file not found: %s", hashPath)
	}

	// Verify lib files extracted
	dllFile := filepath.Join(packageDir, "lib", "net472", "NuGet.Versioning.dll")
	if _, err := os.Stat(dllFile); os.IsNotExist(err) {
		t.Errorf("Expected DLL file not found: %s", dllFile)
	}

	t.Logf("Installed package to %s", packageDir)
}

// TestInstallFromSourceV3_SignedPackage tests installation with signed package and content hash
func TestInstallFromSourceV3_SignedPackage(t *testing.T) {
	packagePath := "testdata/TestPackage.AuthorSigned.1.0.0.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	globalPackages := filepath.Join(tempDir, "global-packages")

	pkg, err := OpenPackage(packagePath)
	if err != nil {
		t.Fatalf("OpenPackage() error = %v", err)
	}
	identity, err := pkg.GetIdentity()
	_ = pkg.Close()
	if err != nil {
		t.Fatalf("GetIdentity() error = %v", err)
	}

	resolver := NewVersionFolderPathResolver(globalPackages, true)
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV3,
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		Logger:             nil,
	}

	copyToAsync := func(targetPath string) error {
		src, err := os.Open(packagePath)
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()

		dst, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer func() { _ = dst.Close() }()

		_, err = io.Copy(dst, src)
		return err
	}

	installed, err := InstallFromSourceV3(context.Background(), packagePath, identity, copyToAsync, resolver, ctx)
	if err != nil {
		t.Fatalf("InstallFromSourceV3() error = %v", err)
	}

	if !installed {
		t.Error("InstallFromSourceV3() returned false for signed package")
	}

	// Verify metadata file contains content hash for signed package
	metadataPath := resolver.GetNupkgMetadataPath(identity.ID, identity.Version)
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to read metadata file: %v", err)
	}

	metadataStr := string(metadataBytes)
	if !strings.Contains(metadataStr, "contentHash") {
		t.Error("Metadata file should contain contentHash for signed package")
	}

	// Verify content hash is not empty
	if strings.Contains(metadataStr, `"contentHash":""`) {
		t.Error("Content hash should not be empty for signed package")
	}

	t.Logf("Installed signed package with content hash")
}

// TestInstallFromSourceV3_IdempotentInstall tests reinstalling does not fail
func TestInstallFromSourceV3_IdempotentInstall(t *testing.T) {
	packagePath := "testdata/TestUpdatePackage.1.0.1.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	globalPackages := filepath.Join(tempDir, "global-packages")

	pkg, err := OpenPackage(packagePath)
	if err != nil {
		t.Fatalf("OpenPackage() error = %v", err)
	}
	identity, err := pkg.GetIdentity()
	_ = pkg.Close()
	if err != nil {
		t.Fatalf("GetIdentity() error = %v", err)
	}

	resolver := NewVersionFolderPathResolver(globalPackages, true)
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV3,
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		Logger:             nil,
	}

	copyToAsync := func(targetPath string) error {
		src, err := os.Open(packagePath)
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()

		dst, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer func() { _ = dst.Close() }()

		_, err = io.Copy(dst, src)
		return err
	}

	// First installation
	installed1, err := InstallFromSourceV3(context.Background(), packagePath, identity, copyToAsync, resolver, ctx)
	if err != nil {
		t.Fatalf("First InstallFromSourceV3() error = %v", err)
	}

	if !installed1 {
		t.Error("First installation should return true")
	}

	// Second installation (should detect existing and skip)
	installed2, err := InstallFromSourceV3(context.Background(), packagePath, identity, copyToAsync, resolver, ctx)
	if err != nil {
		t.Fatalf("Second InstallFromSourceV3() error = %v", err)
	}

	if installed2 {
		t.Error("Second installation should return false (already installed)")
	}

	t.Logf("Idempotent installation verified")
}

// TestInstallFromSourceV3_ConcurrentInstall tests concurrent installation safety with file locking
func TestInstallFromSourceV3_ConcurrentInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	packagePath := "testdata/TestUpdatePackage.1.0.1.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	globalPackages := filepath.Join(tempDir, "global-packages")

	pkg, err := OpenPackage(packagePath)
	if err != nil {
		t.Fatalf("OpenPackage() error = %v", err)
	}
	identity, err := pkg.GetIdentity()
	_ = pkg.Close()
	if err != nil {
		t.Fatalf("GetIdentity() error = %v", err)
	}

	// Launch 10 concurrent installations
	const numInstalls = 10
	results := make(chan error, numInstalls)

	for i := range numInstalls {
		go func(id int) {
			resolver := NewVersionFolderPathResolver(globalPackages, true)
			ctx := &PackageExtractionContext{
				PackageSaveMode:    PackageSaveModeDefaultV3,
				XMLDocFileSaveMode: XMLDocFileSaveModeNone,
				Logger:             nil,
			}

			copyToAsync := func(targetPath string) error {
				src, err := os.Open(packagePath)
				if err != nil {
					return err
				}
				defer func() { _ = src.Close() }()

				dst, err := os.Create(targetPath)
				if err != nil {
					return err
				}
				defer func() { _ = dst.Close() }()

				_, err = io.Copy(dst, src)
				return err
			}

			_, err := InstallFromSourceV3(context.Background(), packagePath, identity, copyToAsync, resolver, ctx)
			results <- err
		}(i)
	}

	// Collect results
	successCount := 0
	for i := range numInstalls {
		err := <-results
		if err == nil {
			successCount++
		} else {
			t.Logf("Concurrent install %d error: %v", i, err)
		}
	}

	if successCount == 0 {
		t.Error("All concurrent installations failed")
	}

	// Verify package is properly installed
	resolver := NewVersionFolderPathResolver(globalPackages, true)
	nupkgPath := resolver.GetPackageFilePath(identity.ID, identity.Version)

	if _, err := os.Stat(nupkgPath); os.IsNotExist(err) {
		t.Error("Package not properly installed after concurrent attempts")
	}

	t.Logf("Concurrent installations: %d succeeded out of %d", successCount, numInstalls)
}

// TestInstallFromSourceV3_WithLogger tests installation with logger
func TestInstallFromSourceV3_WithLogger(t *testing.T) {
	packagePath := "testdata/TestUpdatePackage.1.0.1.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	globalPackages := filepath.Join(tempDir, "global-packages")

	pkg, err := OpenPackage(packagePath)
	if err != nil {
		t.Fatalf("OpenPackage() error = %v", err)
	}
	identity, err := pkg.GetIdentity()
	_ = pkg.Close()
	if err != nil {
		t.Fatalf("GetIdentity() error = %v", err)
	}

	resolver := NewVersionFolderPathResolver(globalPackages, true)

	// Create logger
	logger := &v3TestLogger{messages: make([]string, 0)}

	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV3,
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		Logger:             logger,
	}

	copyToAsync := func(targetPath string) error {
		src, err := os.Open(packagePath)
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()

		dst, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer func() { _ = dst.Close() }()

		_, err = io.Copy(dst, src)
		return err
	}

	installed, err := InstallFromSourceV3(context.Background(), packagePath, identity, copyToAsync, resolver, ctx)
	if err != nil {
		t.Fatalf("InstallFromSourceV3() error = %v", err)
	}

	if !installed {
		t.Error("InstallFromSourceV3() returned false")
	}

	// Verify logger was called
	if len(logger.messages) == 0 {
		t.Error("Expected logger messages, got none")
	}

	t.Logf("Logger messages: %v", logger.messages)
}

// TestInstallFromSourceV3_WithoutNupkg tests installation without saving nupkg file
func TestInstallFromSourceV3_WithoutNupkg(t *testing.T) {
	packagePath := "testdata/TestUpdatePackage.1.0.1.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	globalPackages := filepath.Join(tempDir, "global-packages")

	pkg, err := OpenPackage(packagePath)
	if err != nil {
		t.Fatalf("OpenPackage() error = %v", err)
	}
	identity, err := pkg.GetIdentity()
	_ = pkg.Close()
	if err != nil {
		t.Fatalf("GetIdentity() error = %v", err)
	}

	resolver := NewVersionFolderPathResolver(globalPackages, true)

	// Save mode without nupkg
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeNuspec | PackageSaveModeFiles,
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		Logger:             nil,
	}

	copyToAsync := func(targetPath string) error {
		src, err := os.Open(packagePath)
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()

		dst, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer func() { _ = dst.Close() }()

		_, err = io.Copy(dst, src)
		return err
	}

	installed, err := InstallFromSourceV3(context.Background(), packagePath, identity, copyToAsync, resolver, ctx)
	if err != nil {
		t.Fatalf("InstallFromSourceV3() error = %v", err)
	}

	if !installed {
		t.Error("InstallFromSourceV3() returned false")
	}

	// Verify .nupkg file does NOT exist
	nupkgPath := resolver.GetPackageFilePath(identity.ID, identity.Version)
	if _, err := os.Stat(nupkgPath); err == nil {
		t.Errorf("Expected .nupkg file to not exist but it does: %s", nupkgPath)
	}

	// Verify .nuspec and hash files exist
	nuspecPath := resolver.GetManifestFilePath(identity.ID, identity.Version)
	if _, err := os.Stat(nuspecPath); os.IsNotExist(err) {
		t.Errorf("Expected .nuspec file not found: %s", nuspecPath)
	}

	hashPath := resolver.GetHashPath(identity.ID, identity.Version)
	if _, err := os.Stat(hashPath); os.IsNotExist(err) {
		t.Errorf("Expected .sha512 file not found: %s", hashPath)
	}

	t.Logf("Installed package without .nupkg file")
}

// v3TestLogger implements Logger interface for V3 testing
type v3TestLogger struct {
	messages []string
}

func (l *v3TestLogger) Info(format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf("INFO: "+format, args...))
}

func (l *v3TestLogger) Warning(format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf("WARN: "+format, args...))
}

func (l *v3TestLogger) Error(format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf("ERROR: "+format, args...))
}

// TestInstallFromSourceV3_ErrorPaths tests error handling
func TestInstallFromSourceV3_ErrorPaths(t *testing.T) {
	t.Run("Invalid package path", func(t *testing.T) {
		tempDir := t.TempDir()
		globalPackages := filepath.Join(tempDir, "global-packages")

		identity := &PackageIdentity{
			ID:      "TestPackage",
			Version: version.MustParse("1.0.0"),
		}

		resolver := NewVersionFolderPathResolver(globalPackages, true)
		ctx := &PackageExtractionContext{
			PackageSaveMode:    PackageSaveModeDefaultV3,
			XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		}

		copyToAsync := func(targetPath string) error {
			return os.ErrNotExist
		}

		_, err := InstallFromSourceV3(context.Background(), "nonexistent.nupkg", identity, copyToAsync, resolver, ctx)

		if err == nil {
			t.Error("Expected error for invalid package")
		}
	})

	t.Run("Copy failure triggers cleanup", func(t *testing.T) {
		packagePath := "testdata/TestUpdatePackage.1.0.1.nupkg"
		if _, err := os.Stat(packagePath); os.IsNotExist(err) {
			t.Skip("Test package not found")
		}

		pkg, _ := OpenPackage(packagePath)
		identity, _ := pkg.GetIdentity()
		_ = pkg.Close()

		tempDir := t.TempDir()
		globalPackages := filepath.Join(tempDir, "global-packages")

		resolver := NewVersionFolderPathResolver(globalPackages, true)
		ctx := &PackageExtractionContext{
			PackageSaveMode:    PackageSaveModeDefaultV3,
			XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		}

		// copyToAsync that fails - this should trigger cleanup
		copyToAsync := func(targetPath string) error {
			// Create temp file then fail
			f, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			_ = f.Close()
			return os.ErrPermission // Simulate failure
		}

		_, err := InstallFromSourceV3(context.Background(), packagePath, identity, copyToAsync, resolver, ctx)

		if err == nil {
			t.Error("Expected error from failed copy")
		}

		// Verify cleanup behavior matches NuGet.Client:
		// 1. Temp nupkg file should be removed (if possible)
		// 2. Directory cleanup may fail if not empty (concurrent-safe behavior)
		// Reference: NuGet.Client PackageExtractor.DeleteTargetAndTempPaths
		//
		// NuGet.Client uses Directory.Delete() which is non-recursive and throws
		// if directory is not empty. The cleanup is wrapped in try-catch and logs
		// warnings on failure. This is expected behavior for concurrent installations.
		targetNupkg := resolver.GetPackageFilePath(identity.ID, identity.Version)

		// Final nupkg should not exist (temp was either removed or never renamed)
		if _, err := os.Stat(targetNupkg); err == nil {
			t.Error("Final nupkg file should not exist after failed installation")
		}

		// Directory may or may not exist - both are valid:
		// - If empty: cleanup removed it (single installation)
		// - If not empty: cleanup skipped it (concurrent installations)
		// This matches NuGet.Client's behavior where Directory.Delete() throws IOException
	})
}
