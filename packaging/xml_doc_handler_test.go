package packaging

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestNewPackageFileExtractor(t *testing.T) {
	tests := []struct {
		name       string
		files      []string
		saveMode   XMLDocFileSaveMode
		expectDocs map[string]bool
	}{
		{
			name: "identify XML docs in lib/",
			files: []string{
				"lib/net45/MyLib.dll",
				"lib/net45/MyLib.xml",
				"lib/net45/Other.xml",
			},
			saveMode: XMLDocFileSaveModeCompress,
			expectDocs: map[string]bool{
				"lib/net45/MyLib.xml": true,
				"lib/net45/Other.xml": false,
			},
		},
		{
			name: "identify XML docs in ref/",
			files: []string{
				"ref/net6.0/MyLib.dll",
				"ref/net6.0/MyLib.xml",
			},
			saveMode: XMLDocFileSaveModeCompress,
			expectDocs: map[string]bool{
				"ref/net6.0/MyLib.xml": true,
			},
		},
		{
			name: "XML docs with exe",
			files: []string{
				"lib/net45/MyApp.exe",
				"lib/net45/MyApp.xml",
			},
			saveMode: XMLDocFileSaveModeCompress,
			expectDocs: map[string]bool{
				"lib/net45/MyApp.xml": true,
			},
		},
		{
			name: "no XML docs - saveMode none",
			files: []string{
				"lib/net45/MyLib.dll",
				"lib/net45/MyLib.xml",
			},
			saveMode:   XMLDocFileSaveModeNone,
			expectDocs: map[string]bool{},
		},
		{
			name: "XML in content/ - not a doc",
			files: []string{
				"content/data.xml",
			},
			saveMode: XMLDocFileSaveModeCompress,
			expectDocs: map[string]bool{
				"content/data.xml": false,
			},
		},
		{
			name: "resource assembly XML docs",
			files: []string{
				"lib/net6.0/MyLib.dll",
				"lib/net6.0/ja-jp/MyLib.resources.dll",
				"lib/net6.0/ja-jp/MyLib.xml",
			},
			saveMode: XMLDocFileSaveModeCompress,
			expectDocs: map[string]bool{
				"lib/net6.0/ja-jp/MyLib.xml": true,
			},
		},
		{
			name: "multiple frameworks",
			files: []string{
				"lib/net45/MyLib.dll",
				"lib/net45/MyLib.xml",
				"lib/net6.0/MyLib.dll",
				"lib/net6.0/MyLib.xml",
			},
			saveMode: XMLDocFileSaveModeCompress,
			expectDocs: map[string]bool{
				"lib/net45/MyLib.xml":  true,
				"lib/net6.0/MyLib.xml": true,
			},
		},
		{
			name: "case insensitive matching",
			files: []string{
				"lib/net45/MyLib.DLL",
				"lib/net45/mylib.xml",
			},
			saveMode: XMLDocFileSaveModeCompress,
			expectDocs: map[string]bool{
				"lib/net45/mylib.xml": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewPackageFileExtractor(tt.files, tt.saveMode)

			if extractor.xmlDocFileSaveMode != tt.saveMode {
				t.Errorf("saveMode = %v, want %v", extractor.xmlDocFileSaveMode, tt.saveMode)
			}

			// Check identified XML docs
			for file, expected := range tt.expectDocs {
				got := extractor.xmlDocFiles[file]
				if got != expected {
					t.Errorf("xmlDocFiles[%q] = %v, want %v", file, got, expected)
				}
			}

			// Verify empty when saveMode is None
			if tt.saveMode == XMLDocFileSaveModeNone && len(extractor.xmlDocFiles) > 0 {
				t.Errorf("xmlDocFiles should be empty when saveMode is None, got %d entries", len(extractor.xmlDocFiles))
			}
		})
	}
}

func TestExtractPackageFile(t *testing.T) {
	tests := []struct {
		name           string
		files          []string
		saveMode       XMLDocFileSaveMode
		sourceFile     string
		fileContent    string
		expectExtract  bool
		expectCompress bool
	}{
		{
			name: "extract normal file",
			files: []string{
				"lib/net45/MyLib.dll",
			},
			saveMode:       XMLDocFileSaveModeNone,
			sourceFile:     "lib/net45/MyLib.dll",
			fileContent:    "DLL content",
			expectExtract:  true,
			expectCompress: false,
		},
		{
			name: "skip XML doc with skip mode",
			files: []string{
				"lib/net45/MyLib.dll",
				"lib/net45/MyLib.xml",
			},
			saveMode:       XMLDocFileSaveModeSkip,
			sourceFile:     "lib/net45/MyLib.xml",
			fileContent:    "<doc>XML content</doc>",
			expectExtract:  false,
			expectCompress: false,
		},
		{
			name: "compress XML doc",
			files: []string{
				"lib/net45/MyLib.dll",
				"lib/net45/MyLib.xml",
			},
			saveMode:       XMLDocFileSaveModeCompress,
			sourceFile:     "lib/net45/MyLib.xml",
			fileContent:    "<doc>XML content</doc>",
			expectExtract:  true,
			expectCompress: true,
		},
		{
			name: "extract XML doc with none mode",
			files: []string{
				"lib/net45/MyLib.dll",
				"lib/net45/MyLib.xml",
			},
			saveMode:       XMLDocFileSaveModeNone,
			sourceFile:     "lib/net45/MyLib.xml",
			fileContent:    "<doc>XML content</doc>",
			expectExtract:  true,
			expectCompress: false,
		},
		{
			name: "extract non-doc XML with compress mode",
			files: []string{
				"content/data.xml",
			},
			saveMode:       XMLDocFileSaveModeCompress,
			sourceFile:     "content/data.xml",
			fileContent:    "<data>Content</data>",
			expectExtract:  true,
			expectCompress: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			targetPath := filepath.Join(tempDir, filepath.FromSlash(tt.sourceFile))

			extractor := NewPackageFileExtractor(tt.files, tt.saveMode)
			stream := bytes.NewReader([]byte(tt.fileContent))

			resultPath, err := extractor.ExtractPackageFile(tt.sourceFile, targetPath, stream)

			if err != nil {
				t.Fatalf("ExtractPackageFile() error = %v", err)
			}

			if tt.expectExtract {
				if resultPath == "" {
					t.Errorf("ExtractPackageFile() returned empty path, expected extraction")
				}

				if tt.expectCompress {
					// Verify compressed file
					expectedZipPath := targetPath + ".zip"
					if resultPath != expectedZipPath {
						t.Errorf("ExtractPackageFile() path = %v, want %v", resultPath, expectedZipPath)
					}

					// Verify ZIP file exists and contains XML
					if _, err := os.Stat(expectedZipPath); os.IsNotExist(err) {
						t.Errorf("Compressed XML not created at %s", expectedZipPath)
					}

					// Verify ZIP contents
					verifyCompressedXML(t, expectedZipPath, tt.sourceFile, tt.fileContent)
				} else {
					// Verify normal file
					if resultPath != targetPath {
						t.Errorf("ExtractPackageFile() path = %v, want %v", resultPath, targetPath)
					}

					if _, err := os.Stat(targetPath); os.IsNotExist(err) {
						t.Errorf("File not created at %s", targetPath)
					}

					// Verify content
					content, err := os.ReadFile(targetPath)
					if err != nil {
						t.Fatalf("ReadFile() error = %v", err)
					}
					if string(content) != tt.fileContent {
						t.Errorf("File content = %q, want %q", content, tt.fileContent)
					}
				}
			} else {
				if resultPath != "" {
					t.Errorf("ExtractPackageFile() returned path %q, expected empty for skipped file", resultPath)
				}

				// Verify file was not created
				if _, err := os.Stat(targetPath); err == nil {
					t.Errorf("File should not be created when skipped")
				}
			}
		})
	}
}

func TestCompressXmlDoc(t *testing.T) {
	tests := []struct {
		name       string
		xmlContent string
		targetPath string
		wantErr    bool
	}{
		{
			name:       "compress simple XML",
			xmlContent: "<doc><assembly><name>MyLib</name></assembly></doc>",
			targetPath: "lib/net45/MyLib.xml",
			wantErr:    false,
		},
		{
			name:       "compress large XML",
			xmlContent: strings.Repeat("<member name='Test'><summary>Doc</summary></member>", 1000),
			targetPath: "lib/net45/Large.xml",
			wantErr:    false,
		},
		{
			name:       "compress empty XML",
			xmlContent: "",
			targetPath: "lib/net45/Empty.xml",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			targetPath := filepath.Join(tempDir, filepath.FromSlash(tt.targetPath))

			extractor := &PackageFileExtractor{
				xmlDocFileSaveMode: XMLDocFileSaveModeCompress,
			}

			stream := bytes.NewReader([]byte(tt.xmlContent))
			resultPath, err := extractor.compressXMLDoc(targetPath, stream)

			if (err != nil) != tt.wantErr {
				t.Errorf("compressXMLDoc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				expectedZipPath := targetPath + ".zip"
				if resultPath != expectedZipPath {
					t.Errorf("compressXMLDoc() path = %v, want %v", resultPath, expectedZipPath)
				}

				// Verify ZIP file exists
				if _, err := os.Stat(expectedZipPath); os.IsNotExist(err) {
					t.Errorf("ZIP file not created at %s", expectedZipPath)
				}

				// Verify ZIP contents
				verifyCompressedXML(t, expectedZipPath, tt.targetPath, tt.xmlContent)
			}
		})
	}
}

func TestCompressXmlDoc_DirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	// Deep nested path
	targetPath := filepath.Join(tempDir, "deep", "nested", "path", "MyLib.xml")

	extractor := &PackageFileExtractor{
		xmlDocFileSaveMode: XMLDocFileSaveModeCompress,
	}

	xmlContent := "<doc>Test</doc>"
	stream := bytes.NewReader([]byte(xmlContent))

	resultPath, err := extractor.compressXMLDoc(targetPath, stream)
	if err != nil {
		t.Fatalf("compressXMLDoc() error = %v", err)
	}

	// Verify all directories were created
	expectedZipPath := targetPath + ".zip"
	if _, err := os.Stat(filepath.Dir(expectedZipPath)); os.IsNotExist(err) {
		t.Errorf("Parent directories not created")
	}

	if resultPath != expectedZipPath {
		t.Errorf("compressXMLDoc() path = %v, want %v", resultPath, expectedZipPath)
	}
}

func TestXMLDocFileSaveMode(t *testing.T) {
	tests := []struct {
		name string
		mode XMLDocFileSaveMode
		want XMLDocFileSaveMode
	}{
		{"None", XMLDocFileSaveModeNone, 0},
		{"Skip", XMLDocFileSaveModeSkip, 1},
		{"Compress", XMLDocFileSaveModeCompress, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mode != tt.want {
				t.Errorf("XMLDocFileSaveMode %s = %v, want %v", tt.name, tt.mode, tt.want)
			}
		})
	}
}

func TestExtractPackageFile_StreamConsumed(t *testing.T) {
	// Verify that ExtractPackageFile properly consumes the stream
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "test.dll")

	files := []string{"test.dll"}
	extractor := NewPackageFileExtractor(files, XMLDocFileSaveModeNone)

	content := "test content"
	stream := bytes.NewReader([]byte(content))

	_, err := extractor.ExtractPackageFile("test.dll", targetPath, stream)
	if err != nil {
		t.Fatalf("ExtractPackageFile() error = %v", err)
	}

	// Verify stream was fully consumed
	remaining := stream.Len()
	if remaining != 0 {
		t.Errorf("Stream not fully consumed, %d bytes remaining", remaining)
	}
}

func TestIdentifyXMLDocFiles_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		files      []string
		expectDocs []string
	}{
		{
			name:       "empty file list",
			files:      []string{},
			expectDocs: []string{},
		},
		{
			name: "only XML files no assemblies",
			files: []string{
				"lib/net45/MyLib.xml",
				"lib/net45/Other.xml",
			},
			expectDocs: []string{},
		},
		{
			name: "only assemblies no XML",
			files: []string{
				"lib/net45/MyLib.dll",
				"lib/net45/Other.dll",
			},
			expectDocs: []string{},
		},
		{
			name: "XML in tools/ - not a doc",
			files: []string{
				"tools/install.ps1",
				"tools/config.xml",
			},
			expectDocs: []string{},
		},
		{
			name: "XML in root - not a doc",
			files: []string{
				"MyPackage.nuspec",
				"readme.xml",
			},
			expectDocs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewPackageFileExtractor(tt.files, XMLDocFileSaveModeCompress)

			for file := range extractor.xmlDocFiles {
				if !slices.Contains(tt.expectDocs, file) {
					t.Errorf("Unexpected XML doc identified: %s", file)
				}
			}

			if len(tt.expectDocs) != len(extractor.xmlDocFiles) {
				t.Errorf("Expected %d XML docs, got %d", len(tt.expectDocs), len(extractor.xmlDocFiles))
			}
		})
	}
}

func TestCompressXmlDoc_ZipStructure(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "lib", "net45", "MyLib.xml")

	extractor := &PackageFileExtractor{
		xmlDocFileSaveMode: XMLDocFileSaveModeCompress,
	}

	xmlContent := "<doc>Test content</doc>"
	stream := bytes.NewReader([]byte(xmlContent))

	resultPath, err := extractor.compressXMLDoc(targetPath, stream)
	if err != nil {
		t.Fatalf("compressXMLDoc() error = %v", err)
	}

	// Open ZIP and verify structure
	zipFile, err := zip.OpenReader(resultPath)
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	defer func() { _ = zipFile.Close() }()

	// Should have exactly one entry
	if len(zipFile.File) != 1 {
		t.Errorf("ZIP should have 1 entry, got %d", len(zipFile.File))
	}

	if len(zipFile.File) > 0 {
		entry := zipFile.File[0]

		// Entry name should be the base filename (MyLib.xml)
		expectedName := filepath.Base(targetPath)
		if entry.Name != expectedName {
			t.Errorf("ZIP entry name = %q, want %q", entry.Name, expectedName)
		}

		// Verify content
		rc, err := entry.Open()
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer func() { _ = rc.Close() }()

		content, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if string(content) != xmlContent {
			t.Errorf("ZIP content = %q, want %q", content, xmlContent)
		}
	}
}

func BenchmarkNewPackageFileExtractor(b *testing.B) {
	files := make([]string, 100)
	for i := range 100 {
		if i%2 == 0 {
			files[i] = filepath.Join("lib/net45", "MyLib"+string(rune(i))+".dll")
		} else {
			files[i] = filepath.Join("lib/net45", "MyLib"+string(rune(i-1))+".xml")
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_ = NewPackageFileExtractor(files, XMLDocFileSaveModeCompress)
	}
}

func BenchmarkExtractPackageFile_Normal(b *testing.B) {
	tempDir := b.TempDir()
	files := []string{"lib/net45/MyLib.dll"}
	extractor := NewPackageFileExtractor(files, XMLDocFileSaveModeNone)
	content := []byte("DLL content")

	b.ResetTimer()
	for i := range b.N {
		targetPath := filepath.Join(tempDir, fmt.Sprintf("MyLib%d.dll", i))
		stream := bytes.NewReader(content)
		_, err := extractor.ExtractPackageFile("lib/net45/MyLib.dll", targetPath, stream)
		if err != nil {
			b.Fatalf("ExtractPackageFile() error = %v", err)
		}
	}
}

func BenchmarkCompressXmlDoc(b *testing.B) {
	tempDir := b.TempDir()
	extractor := &PackageFileExtractor{
		xmlDocFileSaveMode: XMLDocFileSaveModeCompress,
	}
	xmlContent := []byte(strings.Repeat("<member name='Test'><summary>Doc</summary></member>", 100))

	b.ResetTimer()
	for i := range b.N {
		targetPath := filepath.Join(tempDir, fmt.Sprintf("MyLib%d.xml", i))
		stream := bytes.NewReader(xmlContent)
		_, err := extractor.compressXMLDoc(targetPath, stream)
		if err != nil {
			b.Fatalf("compressXMLDoc() error = %v", err)
		}
	}
}

// Helper functions

func verifyCompressedXML(t *testing.T, zipPath, sourcePath, expectedContent string) {
	t.Helper()

	zipFile, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	defer func() { _ = zipFile.Close() }()

	if len(zipFile.File) != 1 {
		t.Fatalf("ZIP should have 1 entry, got %d", len(zipFile.File))
	}

	entry := zipFile.File[0]
	expectedName := filepath.Base(sourcePath)
	if entry.Name != expectedName {
		t.Errorf("ZIP entry name = %q, want %q", entry.Name, expectedName)
	}

	rc, err := entry.Open()
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = rc.Close() }()

	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf("ZIP content = %q, want %q", content, expectedContent)
	}
}
