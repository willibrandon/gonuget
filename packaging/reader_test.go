package packaging

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/willibrandon/gonuget/version"
)

// createTestPackage creates a minimal test .nupkg in memory
func createTestPackage(t *testing.T, files map[string]string, includeSignature bool) *bytes.Reader {
	t.Helper()

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Add files
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("Failed to write file %s: %v", name, err)
		}
	}

	// Add signature if requested
	if includeSignature {
		f, err := w.Create(SignaturePath)
		if err != nil {
			t.Fatalf("Failed to create signature: %v", err)
		}
		if _, err := f.Write([]byte("signature data")); err != nil {
			t.Fatalf("Failed to write signature: %v", err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close zip: %v", err)
	}

	return bytes.NewReader(buf.Bytes())
}

// createTestPackageBytes creates a minimal test .nupkg and returns the bytes
func createTestPackageBytes(t *testing.T, files map[string]string, includeSignature bool) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Add files
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("Failed to write file %s: %v", name, err)
		}
	}

	// Add signature if requested
	if includeSignature {
		f, err := w.Create(SignaturePath)
		if err != nil {
			t.Fatalf("Failed to create signature: %v", err)
		}
		if _, err := f.Write([]byte("signature data")); err != nil {
			t.Fatalf("Failed to write signature: %v", err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close zip: %v", err)
	}

	return buf.Bytes()
}

func TestOpenPackage(t *testing.T) {
	// Create a temporary .nupkg file
	files := map[string]string{
		"test.nuspec":  `<?xml version="1.0"?><package></package>`,
		"lib/test.dll": "binary content",
	}

	pkgBytes := createTestPackageBytes(t, files, false)
	tmpFile := t.TempDir() + "/test.nupkg"

	err := os.WriteFile(tmpFile, pkgBytes, 0644)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	pkg, err := OpenPackage(tmpFile)
	if err != nil {
		t.Fatalf("OpenPackage failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	if pkg.zipReader == nil {
		t.Error("zipReader should not be nil")
	}

	if !pkg.isClosable {
		t.Error("Package opened from file should be closable")
	}
}

func TestOpenPackage_InvalidFile(t *testing.T) {
	_, err := OpenPackage("/nonexistent/package.nupkg")
	if err == nil {
		t.Error("OpenPackage should fail for non-existent file")
	}
}

func TestOpenPackageFromReaderAt(t *testing.T) {
	files := map[string]string{
		"test.nuspec":  `<?xml version="1.0"?><package></package>`,
		"lib/test.dll": "binary content",
	}

	reader := createTestPackage(t, files, false)

	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	if pkg.zipReaderAt == nil {
		t.Error("zipReaderAt should not be nil")
	}

	if pkg.isClosable {
		t.Error("Package opened from ReaderAt should not be closable")
	}
}

func TestPackageReader_Files(t *testing.T) {
	files := map[string]string{
		"test.nuspec":  "content",
		"lib/test.dll": "binary",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	zipFiles := pkg.Files()
	if len(zipFiles) != 2 {
		t.Errorf("Expected 2 files, got %d", len(zipFiles))
	}
}

func TestPackageReader_IsSigned(t *testing.T) {
	tests := []struct {
		name             string
		includeSignature bool
		want             bool
	}{
		{
			name:             "unsigned package",
			includeSignature: false,
			want:             false,
		},
		{
			name:             "signed package",
			includeSignature: true,
			want:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"test.nuspec": "content",
			}

			reader := createTestPackage(t, files, tt.includeSignature)
			pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
			if err != nil {
				t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
			}
			defer func() { _ = pkg.Close() }()

			got := pkg.IsSigned()
			if got != tt.want {
				t.Errorf("IsSigned() = %v, want %v", got, tt.want)
			}

			// Test caching - should return same result
			got2 := pkg.IsSigned()
			if got2 != got {
				t.Error("IsSigned() should return cached result")
			}
		})
	}
}

func TestPackageReader_GetSignatureFile(t *testing.T) {
	t.Run("signed package", func(t *testing.T) {
		files := map[string]string{
			"test.nuspec": "content",
		}

		reader := createTestPackage(t, files, true)
		pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
		if err != nil {
			t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
		}
		defer func() { _ = pkg.Close() }()

		sigFile, err := pkg.GetSignatureFile()
		if err != nil {
			t.Errorf("GetSignatureFile() error = %v", err)
		}

		if sigFile.Name != SignaturePath {
			t.Errorf("Signature file name = %s, want %s", sigFile.Name, SignaturePath)
		}
	})

	t.Run("unsigned package", func(t *testing.T) {
		files := map[string]string{
			"test.nuspec": "content",
		}

		reader := createTestPackage(t, files, false)
		pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
		if err != nil {
			t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
		}
		defer func() { _ = pkg.Close() }()

		_, err = pkg.GetSignatureFile()
		if err != ErrPackageNotSigned {
			t.Errorf("GetSignatureFile() error = %v, want %v", err, ErrPackageNotSigned)
		}
	})
}

func TestPackageReader_GetNuspecFile(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		wantErr error
	}{
		{
			name: "single nuspec at root",
			files: map[string]string{
				"test.nuspec": "content",
			},
			wantErr: nil,
		},
		{
			name: "case insensitive extension",
			files: map[string]string{
				"test.NUSPEC": "content",
			},
			wantErr: nil,
		},
		{
			name: "no nuspec",
			files: map[string]string{
				"lib/test.dll": "content",
			},
			wantErr: ErrNuspecNotFound,
		},
		{
			name: "multiple nuspecs",
			files: map[string]string{
				"test1.nuspec": "content",
				"test2.nuspec": "content",
			},
			wantErr: ErrMultipleNuspecs,
		},
		{
			name: "nuspec in subdirectory ignored",
			files: map[string]string{
				"test.nuspec":     "content",
				"lib/test.nuspec": "ignored",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := createTestPackage(t, tt.files, false)
			pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
			if err != nil {
				t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
			}
			defer func() { _ = pkg.Close() }()

			nuspecFile, err := pkg.GetNuspecFile()
			if err != tt.wantErr {
				t.Errorf("GetNuspecFile() error = %v, want %v", err, tt.wantErr)
				return
			}

			if tt.wantErr == nil && nuspecFile == nil {
				t.Error("GetNuspecFile() returned nil file with no error")
			}

			// Test caching
			if tt.wantErr == nil {
				nuspecFile2, _ := pkg.GetNuspecFile()
				if nuspecFile2 != nuspecFile {
					t.Error("GetNuspecFile() should return cached result")
				}
			}
		})
	}
}

func TestPackageReader_OpenNuspec(t *testing.T) {
	files := map[string]string{
		"test.nuspec": "<?xml version=\"1.0\"?><package></package>",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	rc, err := pkg.OpenNuspec()
	if err != nil {
		t.Fatalf("OpenNuspec() error = %v", err)
	}
	defer func() { _ = rc.Close() }()

	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("Failed to read nuspec: %v", err)
	}

	expected := files["test.nuspec"]
	if string(content) != expected {
		t.Errorf("Nuspec content = %s, want %s", string(content), expected)
	}
}

func TestPackageReader_GetFile(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		lookFor  string
		wantErr  bool
		wantName string
	}{
		{
			name: "exact match",
			files: map[string]string{
				"lib/net6.0/test.dll": "content",
			},
			lookFor:  "lib/net6.0/test.dll",
			wantErr:  false,
			wantName: "lib/net6.0/test.dll",
		},
		{
			name: "case insensitive",
			files: map[string]string{
				"lib/net6.0/test.dll": "content",
			},
			lookFor:  "LIB/NET6.0/TEST.DLL",
			wantErr:  false,
			wantName: "lib/net6.0/test.dll",
		},
		{
			name: "backslash conversion",
			files: map[string]string{
				"lib/net6.0/test.dll": "content",
			},
			lookFor:  "lib\\net6.0\\test.dll",
			wantErr:  false,
			wantName: "lib/net6.0/test.dll",
		},
		{
			name: "file not found",
			files: map[string]string{
				"lib/test.dll": "content",
			},
			lookFor: "lib/missing.dll",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := createTestPackage(t, tt.files, false)
			pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
			if err != nil {
				t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
			}
			defer func() { _ = pkg.Close() }()

			file, err := pkg.GetFile(tt.lookFor)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && file.Name != tt.wantName {
				t.Errorf("GetFile() file name = %s, want %s", file.Name, tt.wantName)
			}
		})
	}
}

func TestPackageReader_HasFile(t *testing.T) {
	files := map[string]string{
		"lib/test.dll": "content",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	if !pkg.HasFile("lib/test.dll") {
		t.Error("HasFile() should return true for existing file")
	}

	if pkg.HasFile("lib/missing.dll") {
		t.Error("HasFile() should return false for missing file")
	}
}

func TestPackageReader_GetFiles(t *testing.T) {
	files := map[string]string{
		"lib/net6.0/test1.dll": "content",
		"lib/net6.0/test2.dll": "content",
		"lib/net48/test.dll":   "content",
		"content/file.txt":     "content",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	matches := pkg.GetFiles("lib/net6.0/")
	if len(matches) != 2 {
		t.Errorf("GetFiles('lib/net6.0/') returned %d files, want 2", len(matches))
	}

	matches = pkg.GetFiles("lib/")
	if len(matches) != 3 {
		t.Errorf("GetFiles('lib/') returned %d files, want 3", len(matches))
	}

	matches = pkg.GetFiles("missing/")
	if len(matches) != 0 {
		t.Errorf("GetFiles('missing/') returned %d files, want 0", len(matches))
	}
}

func TestValidatePackagePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid path",
			path:    "lib/net6.0/test.dll",
			wantErr: false,
		},
		{
			name:    "valid windows path",
			path:    "lib\\net6.0\\test.dll",
			wantErr: false,
		},
		{
			name:    "path traversal",
			path:    "../etc/passwd",
			wantErr: true,
		},
		{
			name:    "path traversal in middle",
			path:    "lib/../etc/passwd",
			wantErr: true,
		},
		{
			name:    "absolute path",
			path:    "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "whitespace path",
			path:    "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePackagePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePackagePath() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != ErrInvalidPath {
				t.Errorf("ValidatePackagePath() error = %v, want %v", err, ErrInvalidPath)
			}
		})
	}
}

func TestPackageIdentity_String(t *testing.T) {
	ver := version.MustParse("1.2.3")
	identity := &PackageIdentity{
		ID:      "TestPackage",
		Version: ver,
	}

	got := identity.String()
	want := "TestPackage 1.2.3"

	if got != want {
		t.Errorf("String() = %s, want %s", got, want)
	}
}

func TestPackageReader_Close(t *testing.T) {
	t.Run("closable reader", func(t *testing.T) {
		// We can't easily test file-based Close without creating a temp file
		// but we can test that non-closable readers don't error
	})

	t.Run("non-closable reader", func(t *testing.T) {
		files := map[string]string{
			"test.nuspec": "content",
		}

		reader := createTestPackage(t, files, false)
		pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
		if err != nil {
			t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
		}

		err = pkg.Close()
		if err != nil {
			t.Errorf("Close() on non-closable reader should not error: %v", err)
		}
	})
}

func TestPackageReader_GetPackageFiles(t *testing.T) {
	files := map[string]string{
		"test.nuspec":               "nuspec content",
		".signature.p7s":            "signature",
		"_rels/.rels":               "rels",
		"[Content_Types].xml":       "content types",
		"lib/net6.0/test.dll":       "assembly",
		"content/readme.txt":        "readme",
		"build/TestPackage.targets": "targets",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	packageFiles := pkg.GetPackageFiles()

	// Should exclude: .nuspec, .signature.p7s, _rels/.rels, [Content_Types].xml
	// Should include: lib/*, content/*, build/*
	expectedCount := 3 // lib/net6.0/test.dll, content/readme.txt, build/TestPackage.targets
	if len(packageFiles) != expectedCount {
		t.Errorf("GetPackageFiles() returned %d files, want %d", len(packageFiles), expectedCount)
		for _, f := range packageFiles {
			t.Logf("  - %s", f.Name)
		}
	}

	// Verify metadata files are excluded
	for _, file := range packageFiles {
		if IsPackageMetadataFile(file.Name) {
			t.Errorf("GetPackageFiles() included metadata file: %s", file.Name)
		}
	}
}

func TestPackageReader_GetLibFiles(t *testing.T) {
	files := map[string]string{
		"test.nuspec":         "nuspec",
		"lib/net6.0/test.dll": "assembly",
		"lib/net48/test.dll":  "assembly",
		"ref/net6.0/test.dll": "ref assembly",
		"content/readme.txt":  "readme",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	libFiles := pkg.GetLibFiles()

	expectedCount := 2 // lib/net6.0/test.dll, lib/net48/test.dll
	if len(libFiles) != expectedCount {
		t.Errorf("GetLibFiles() returned %d files, want %d", len(libFiles), expectedCount)
	}

	// Verify all are lib files
	for _, file := range libFiles {
		if !IsLibFile(file.Name) {
			t.Errorf("GetLibFiles() included non-lib file: %s", file.Name)
		}
	}
}

func TestPackageReader_GetRefFiles(t *testing.T) {
	files := map[string]string{
		"test.nuspec":         "nuspec",
		"lib/net6.0/test.dll": "assembly",
		"ref/net6.0/test.dll": "ref assembly",
		"ref/net48/test.dll":  "ref assembly",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	refFiles := pkg.GetRefFiles()

	expectedCount := 2 // ref/net6.0/test.dll, ref/net48/test.dll
	if len(refFiles) != expectedCount {
		t.Errorf("GetRefFiles() returned %d files, want %d", len(refFiles), expectedCount)
	}

	// Verify all are ref files
	for _, file := range refFiles {
		if !IsRefFile(file.Name) {
			t.Errorf("GetRefFiles() included non-ref file: %s", file.Name)
		}
	}
}

func TestPackageReader_GetContentFiles(t *testing.T) {
	files := map[string]string{
		"test.nuspec":                     "nuspec",
		"content/readme.txt":              "readme",
		"content/images/logo.png":         "image",
		"contentFiles/any/any/app.config": "config",
		"lib/net6.0/test.dll":             "assembly",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	contentFiles := pkg.GetContentFiles()

	expectedCount := 3 // content/*, contentFiles/*
	if len(contentFiles) != expectedCount {
		t.Errorf("GetContentFiles() returned %d files, want %d", len(contentFiles), expectedCount)
	}

	// Verify all are content files
	for _, file := range contentFiles {
		if !IsContentFile(file.Name) {
			t.Errorf("GetContentFiles() included non-content file: %s", file.Name)
		}
	}
}

func TestPackageReader_GetBuildFiles(t *testing.T) {
	files := map[string]string{
		"test.nuspec":                         "nuspec",
		"build/TestPackage.targets":           "targets",
		"build/net6.0/TestPackage.props":      "props",
		"buildTransitive/TestPackage.targets": "transitive targets",
		"lib/net6.0/test.dll":                 "assembly",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	buildFiles := pkg.GetBuildFiles()

	expectedCount := 3 // build/*, buildTransitive/*
	if len(buildFiles) != expectedCount {
		t.Errorf("GetBuildFiles() returned %d files, want %d", len(buildFiles), expectedCount)
	}

	// Verify all are build files
	for _, file := range buildFiles {
		if !IsBuildFile(file.Name) {
			t.Errorf("GetBuildFiles() included non-build file: %s", file.Name)
		}
	}
}

func TestPackageReader_GetToolsFiles(t *testing.T) {
	files := map[string]string{
		"test.nuspec":           "nuspec",
		"tools/install.ps1":     "install script",
		"tools/net6.0/tool.exe": "tool",
		"lib/net6.0/test.dll":   "assembly",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	toolsFiles := pkg.GetToolsFiles()

	expectedCount := 2 // tools/*
	if len(toolsFiles) != expectedCount {
		t.Errorf("GetToolsFiles() returned %d files, want %d", len(toolsFiles), expectedCount)
	}

	// Verify all are tools files
	for _, file := range toolsFiles {
		if !IsToolsFile(file.Name) {
			t.Errorf("GetToolsFiles() included non-tools file: %s", file.Name)
		}
	}
}

func TestPackageReader_ExtractFile(t *testing.T) {
	files := map[string]string{
		"test.nuspec":         "nuspec content",
		"lib/net6.0/test.dll": "assembly content",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	// Create temp directory for extraction
	tmpDir := t.TempDir()
	destPath := tmpDir + "/extracted/test.dll"

	err = pkg.ExtractFile("lib/net6.0/test.dll", destPath)
	if err != nil {
		t.Fatalf("ExtractFile() error = %v", err)
	}

	// Verify file was extracted
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	expected := "assembly content"
	if string(content) != expected {
		t.Errorf("Extracted content = %s, want %s", string(content), expected)
	}
}

func TestPackageReader_ExtractFile_InvalidPath(t *testing.T) {
	// Create a package with files that should be rejected
	files := map[string]string{
		"test.nuspec":  "content",
		"lib/test.dll": "assembly",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		zipPath string
	}{
		{
			name:    "path traversal",
			zipPath: "../etc/passwd",
		},
		{
			name:    "absolute path",
			zipPath: "/etc/passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pkg.ExtractFile(tt.zipPath, tmpDir+"/output.txt")
			if err == nil {
				t.Error("ExtractFile() should fail for invalid ZIP path")
			}
		})
	}
}

func TestPackageReader_ExtractFiles(t *testing.T) {
	files := map[string]string{
		"test.nuspec":          "nuspec",
		"lib/net6.0/test1.dll": "assembly1",
		"lib/net6.0/test2.dll": "assembly2",
		"content/readme.txt":   "readme",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	tmpDir := t.TempDir()

	// Extract lib files
	libFiles := pkg.GetLibFiles()
	err = pkg.ExtractFiles(libFiles, tmpDir)
	if err != nil {
		t.Fatalf("ExtractFiles() error = %v", err)
	}

	// Verify files were extracted with correct paths
	expectedFiles := []string{
		"lib/net6.0/test1.dll",
		"lib/net6.0/test2.dll",
	}

	for _, expectedFile := range expectedFiles {
		fullPath := tmpDir + "/" + expectedFile
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file not found: %s", fullPath)
		}
	}
}

func TestPackageReader_CopyFileTo(t *testing.T) {
	files := map[string]string{
		"test.nuspec":         "nuspec content",
		"lib/net6.0/test.dll": "assembly content",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	var buf bytes.Buffer
	err = pkg.CopyFileTo("lib/net6.0/test.dll", &buf)
	if err != nil {
		t.Fatalf("CopyFileTo() error = %v", err)
	}

	expected := "assembly content"
	if buf.String() != expected {
		t.Errorf("Copied content = %s, want %s", buf.String(), expected)
	}
}

func TestPackageReader_CopyFileTo_FileNotFound(t *testing.T) {
	files := map[string]string{
		"test.nuspec": "content",
	}

	reader := createTestPackage(t, files, false)
	pkg, err := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("OpenPackageFromReaderAt failed: %v", err)
	}
	defer func() { _ = pkg.Close() }()

	var buf bytes.Buffer
	err = pkg.CopyFileTo("missing.dll", &buf)
	if err == nil {
		t.Error("CopyFileTo() should fail for missing file")
	}
}

// BenchmarkFileAccess benchmarks file lookup operations
func BenchmarkFileAccess(b *testing.B) {
	files := map[string]string{
		"lib/net6.0/file0.dll":  "content",
		"lib/net6.0/file50.dll": "content",
		"lib/net6.0/file99.dll": "content",
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, _ := w.Create(name)
		_, _ = f.Write([]byte(content))
	}
	_ = w.Close()

	reader := bytes.NewReader(buf.Bytes())
	pkg, _ := OpenPackageFromReaderAt(reader, int64(reader.Len()))
	defer func() { _ = pkg.Close() }()

	b.ResetTimer()
	for b.Loop() {
		_, _ = pkg.GetFile("lib/net6.0/file50.dll")
	}
}
