package packaging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExtractPackageV2_Integration tests V2 extraction with real package
func TestExtractPackageV2_Integration(t *testing.T) {
	packagePath := "testdata/TestUpdatePackage.1.0.1.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	installPath := filepath.Join(tempDir, "packages")

	// Open package file as ReadSeeker
	packageFile, err := os.Open(packagePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = packageFile.Close() }()

	// Create path resolver
	resolver := NewPackagePathResolver(installPath, true)

	// Create extraction context
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV2,
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		Logger:             nil, // Use nil logger for tests
	}

	// Extract package
	files, err := ExtractPackageV2(context.Background(), packagePath, packageFile, resolver, ctx)
	if err != nil {
		t.Fatalf("ExtractPackageV2() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("ExtractPackageV2() returned no files")
	}

	// Open package to get identity for verification
	pkg, err := OpenPackage(packagePath)
	if err != nil {
		t.Fatalf("OpenPackage() error = %v", err)
	}
	defer func() { _ = pkg.Close() }()

	identity, err := pkg.GetIdentity()
	if err != nil {
		t.Fatalf("GetIdentity() error = %v", err)
	}

	packageDir := resolver.GetInstallPath(identity)

	// Verify .nupkg file exists
	nupkgPath := filepath.Join(packageDir, identity.ID+"."+identity.Version.String()+".nupkg")
	if _, err := os.Stat(nupkgPath); os.IsNotExist(err) {
		t.Errorf("Expected .nupkg file not found: %s", nupkgPath)
	}

	// Verify content files extracted
	contentFile := filepath.Join(packageDir, "content", "readme.txt")
	if _, err := os.Stat(contentFile); os.IsNotExist(err) {
		t.Errorf("Expected content file not found: %s", contentFile)
	}

	// Verify tools files extracted
	toolsFile := filepath.Join(packageDir, "tools", "init.ps1")
	if _, err := os.Stat(toolsFile); os.IsNotExist(err) {
		t.Errorf("Expected tools file not found: %s", toolsFile)
	}

	t.Logf("Extracted %d files to %s", len(files), packageDir)
}

// TestExtractPackageV2_WithXMLDocCompression tests XML doc compression
func TestExtractPackageV2_WithXMLDocCompression(t *testing.T) {
	packagePath := "testdata/nuget.versioning.5.0.0.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	installPath := filepath.Join(tempDir, "packages")

	packageFile, err := os.Open(packagePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = packageFile.Close() }()

	resolver := NewPackagePathResolver(installPath, true)
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV2,
		XMLDocFileSaveMode: XMLDocFileSaveModeCompress,
		Logger:             nil,
	}

	files, err := ExtractPackageV2(context.Background(), packagePath, packageFile, resolver, ctx)
	if err != nil {
		t.Fatalf("ExtractPackageV2() error = %v", err)
	}

	pkg, _ := OpenPackage(packagePath)
	defer func() { _ = pkg.Close() }()
	identity, _ := pkg.GetIdentity()
	packageDir := resolver.GetInstallPath(identity)

	// Verify DLL extracted
	dllFile := filepath.Join(packageDir, "lib", "net472", "NuGet.Versioning.dll")
	if _, err := os.Stat(dllFile); os.IsNotExist(err) {
		t.Errorf("Expected DLL file not found: %s", dllFile)
	}

	// Verify XML doc compressed to .xml.zip
	xmlZipFile := filepath.Join(packageDir, "lib", "net472", "NuGet.Versioning.xml.zip")
	if _, err := os.Stat(xmlZipFile); os.IsNotExist(err) {
		t.Errorf("Expected compressed XML doc not found: %s", xmlZipFile)
	}

	// Verify original .xml does NOT exist
	xmlFile := filepath.Join(packageDir, "lib", "net472", "NuGet.Versioning.xml")
	if _, err := os.Stat(xmlFile); err == nil {
		t.Errorf("XML file should have been compressed but exists: %s", xmlFile)
	}

	t.Logf("Extracted %d files with XML compression", len(files))
}

// TestExtractPackageV2_SkipXMLDocs tests skipping XML documentation files
func TestExtractPackageV2_SkipXMLDocs(t *testing.T) {
	packagePath := "testdata/nuget.versioning.5.0.0.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	installPath := filepath.Join(tempDir, "packages")

	packageFile, err := os.Open(packagePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = packageFile.Close() }()

	resolver := NewPackagePathResolver(installPath, true)
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV2,
		XMLDocFileSaveMode: XMLDocFileSaveModeSkip,
		Logger:             nil,
	}

	files, err := ExtractPackageV2(context.Background(), packagePath, packageFile, resolver, ctx)
	if err != nil {
		t.Fatalf("ExtractPackageV2() error = %v", err)
	}

	pkg, _ := OpenPackage(packagePath)
	defer func() { _ = pkg.Close() }()
	identity, _ := pkg.GetIdentity()
	packageDir := resolver.GetInstallPath(identity)

	// Verify DLL extracted
	dllFile := filepath.Join(packageDir, "lib", "net472", "NuGet.Versioning.dll")
	if _, err := os.Stat(dllFile); os.IsNotExist(err) {
		t.Errorf("Expected DLL file not found: %s", dllFile)
	}

	// Verify XML files skipped (do not exist)
	xmlFile := filepath.Join(packageDir, "lib", "net472", "NuGet.Versioning.xml")
	if _, err := os.Stat(xmlFile); err == nil {
		t.Errorf("XML file should have been skipped but exists: %s", xmlFile)
	}

	xmlZipFile := filepath.Join(packageDir, "lib", "net472", "NuGet.Versioning.xml.zip")
	if _, err := os.Stat(xmlZipFile); err == nil {
		t.Errorf("XML.zip file should not exist: %s", xmlZipFile)
	}

	t.Logf("Extracted %d files, skipped XML docs", len(files))
}

// TestExtractPackageV2_SignedPackage tests extracting signed package
func TestExtractPackageV2_SignedPackage(t *testing.T) {
	packagePath := "testdata/TestPackage.AuthorSigned.1.0.0.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	installPath := filepath.Join(tempDir, "packages")

	packageFile, err := os.Open(packagePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = packageFile.Close() }()

	resolver := NewPackagePathResolver(installPath, true)
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV2,
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		Logger:             nil,
	}

	files, err := ExtractPackageV2(context.Background(), packagePath, packageFile, resolver, ctx)
	if err != nil {
		t.Fatalf("ExtractPackageV2() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("ExtractPackageV2() returned no files for signed package")
	}

	pkg, _ := OpenPackage(packagePath)
	defer func() { _ = pkg.Close() }()
	identity, _ := pkg.GetIdentity()
	packageDir := resolver.GetInstallPath(identity)

	// Verify .nupkg file exists
	nupkgPath := filepath.Join(packageDir, identity.ID+"."+identity.Version.String()+".nupkg")
	if _, err := os.Stat(nupkgPath); os.IsNotExist(err) {
		t.Errorf("Expected .nupkg file not found: %s", nupkgPath)
	}

	t.Logf("Extracted %d files from signed package", len(files))
}

// TestExtractPackageV2_NonSideBySide tests non-side-by-side layout
func TestExtractPackageV2_NonSideBySide(t *testing.T) {
	packagePath := "testdata/TestUpdatePackage.1.0.1.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	installPath := filepath.Join(tempDir, "packages")

	packageFile, err := os.Open(packagePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = packageFile.Close() }()

	resolver := NewPackagePathResolver(installPath, false) // Non-side-by-side
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV2,
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		Logger:             nil,
	}

	files, err := ExtractPackageV2(context.Background(), packagePath, packageFile, resolver, ctx)
	if err != nil {
		t.Fatalf("ExtractPackageV2() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("ExtractPackageV2() returned no files")
	}

	pkg, _ := OpenPackage(packagePath)
	defer func() { _ = pkg.Close() }()
	identity, _ := pkg.GetIdentity()
	packageDir := resolver.GetInstallPath(identity)

	// For non-side-by-side, directory is just package ID without version
	expectedDir := filepath.Join(installPath, identity.ID)
	if packageDir != expectedDir {
		t.Errorf("Package dir = %q, want %q", packageDir, expectedDir)
	}

	t.Logf("Extracted %d files to non-side-by-side layout", len(files))
}

// TestExtractPackageV2_WithLogger tests extraction with logger
func TestExtractPackageV2_WithLogger(t *testing.T) {
	packagePath := "testdata/TestUpdatePackage.1.0.1.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	installPath := filepath.Join(tempDir, "packages")

	packageFile, err := os.Open(packagePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = packageFile.Close() }()

	resolver := NewPackagePathResolver(installPath, true)

	// Create logger to capture log messages
	logger := &extractorTestLogger{messages: make([]string, 0)}

	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV2,
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		Logger:             logger,
	}

	files, err := ExtractPackageV2(context.Background(), packagePath, packageFile, resolver, ctx)
	if err != nil {
		t.Fatalf("ExtractPackageV2() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("ExtractPackageV2() returned no files")
	}

	// Verify logger was called
	if len(logger.messages) == 0 {
		t.Error("Expected logger messages, got none")
	}

	t.Logf("Logger messages: %v", logger.messages)
}

// TestExtractPackageV2_WithSatelliteFiles tests satellite file copying
func TestExtractPackageV2_WithSatelliteFiles(t *testing.T) {
	packagePath := "testdata/TestUpdatePackage.1.0.1.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	installPath := filepath.Join(tempDir, "packages")

	packageFile, err := os.Open(packagePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = packageFile.Close() }()

	resolver := NewPackagePathResolver(installPath, true)
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeDefaultV2,
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		CopySatelliteFiles: true, // Enable satellite file copying
		Logger:             nil,
	}

	files, err := ExtractPackageV2(context.Background(), packagePath, packageFile, resolver, ctx)
	if err != nil {
		t.Fatalf("ExtractPackageV2() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("ExtractPackageV2() returned no files")
	}

	t.Logf("Extracted %d files with satellite file copying enabled", len(files))
}

// TestExtractPackageV2_OnlyNuspec tests extracting only nuspec file
func TestExtractPackageV2_OnlyNuspec(t *testing.T) {
	packagePath := "testdata/TestUpdatePackage.1.0.1.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	installPath := filepath.Join(tempDir, "packages")

	packageFile, err := os.Open(packagePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = packageFile.Close() }()

	resolver := NewPackagePathResolver(installPath, true)
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeNuspec, // Only extract nuspec
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		Logger:             nil,
	}

	files, err := ExtractPackageV2(context.Background(), packagePath, packageFile, resolver, ctx)
	if err != nil {
		t.Fatalf("ExtractPackageV2() error = %v", err)
	}

	// Should only have nuspec file
	if len(files) != 1 {
		t.Errorf("ExtractPackageV2() extracted %d files, want 1 (nuspec only)", len(files))
	}

	// Verify it's the nuspec file
	if len(files) > 0 && !strings.HasSuffix(files[0], ".nuspec") {
		t.Errorf("Expected .nuspec file, got %s", files[0])
	}

	t.Logf("Extracted nuspec only: %v", files)
}

// TestExtractPackageV2_OnlyNupkg tests extracting only nupkg file
func TestExtractPackageV2_OnlyNupkg(t *testing.T) {
	packagePath := "testdata/TestUpdatePackage.1.0.1.nupkg"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skipf("Test package not found: %s", packagePath)
	}

	tempDir := t.TempDir()
	installPath := filepath.Join(tempDir, "packages")

	packageFile, err := os.Open(packagePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = packageFile.Close() }()

	resolver := NewPackagePathResolver(installPath, true)
	ctx := &PackageExtractionContext{
		PackageSaveMode:    PackageSaveModeNupkg, // Only save nupkg
		XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		Logger:             nil,
	}

	files, err := ExtractPackageV2(context.Background(), packagePath, packageFile, resolver, ctx)
	if err != nil {
		t.Fatalf("ExtractPackageV2() error = %v", err)
	}

	// Should only have nupkg file
	if len(files) != 1 {
		t.Errorf("ExtractPackageV2() extracted %d files, want 1 (nupkg only)", len(files))
	}

	// Verify it's the nupkg file
	if len(files) > 0 && !strings.HasSuffix(files[0], ".nupkg") {
		t.Errorf("Expected .nupkg file, got %s", files[0])
	}

	t.Logf("Extracted nupkg only: %v", files)
}

// extractorTestLogger implements Logger interface for testing
type extractorTestLogger struct {
	messages []string
}

func (l *extractorTestLogger) Info(format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf("INFO: "+format, args...))
}

func (l *extractorTestLogger) Warning(format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf("WARN: "+format, args...))
}

func (l *extractorTestLogger) Error(format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf("ERROR: "+format, args...))
}

// TestExtractPackageV2_ErrorPaths tests error handling
func TestExtractPackageV2_ErrorPaths(t *testing.T) {
	t.Run("Invalid package path", func(t *testing.T) {
		tempDir := t.TempDir()
		resolver := NewPackagePathResolver(tempDir, true)
		ctx := &PackageExtractionContext{
			PackageSaveMode:    PackageSaveModeDefaultV2,
			XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		}

		// Try to extract non-existent package
		packageFile, _ := os.Open("testdata/nonexistent.nupkg")
		_, err := ExtractPackageV2(context.Background(), "testdata/nonexistent.nupkg", packageFile, resolver, ctx)

		if err == nil {
			t.Error("Expected error for non-existent package")
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		packagePath := "testdata/nuget.versioning.5.0.0.nupkg"
		if _, err := os.Stat(packagePath); os.IsNotExist(err) {
			t.Skip("Test package not found")
		}

		tempDir := t.TempDir()
		resolver := NewPackagePathResolver(tempDir, true)
		ctx := &PackageExtractionContext{
			PackageSaveMode:    PackageSaveModeDefaultV2,
			XMLDocFileSaveMode: XMLDocFileSaveModeNone,
		}

		packageFile, _ := os.Open(packagePath)
		defer func() { _ = packageFile.Close() }()

		// Cancel context immediately
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := ExtractPackageV2(cancelCtx, packagePath, packageFile, resolver, ctx)

		if err == nil {
			t.Error("Expected error from cancelled context")
		}

		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	})
}
