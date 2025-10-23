package signatures

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"strings"
	"testing"
)

func TestGetPackageContentHash(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		signed    bool
		wantEmpty bool
		wantErr   bool
	}{
		{
			name: "unsigned package returns empty",
			files: map[string]string{
				"test.nuspec":  "<?xml version='1.0'?><package></package>",
				"lib/test.dll": "binary content",
			},
			signed:    false,
			wantEmpty: true,
			wantErr:   false,
		},
		{
			name: "signed package returns hash",
			files: map[string]string{
				"test.nuspec":    "<?xml version='1.0'?><package></package>",
				"lib/test.dll":   "binary content",
				".signature.p7s": "signature data",
			},
			signed:    true,
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name: "signed package with multiple files",
			files: map[string]string{
				"test.nuspec":         "<?xml version='1.0'?><package></package>",
				"lib/net45/lib1.dll":  "library 1",
				"lib/net45/lib2.dll":  "library 2",
				"lib/net6.0/lib1.dll": "library 1 for net6",
				".signature.p7s":      "signature data",
			},
			signed:    true,
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name: "signed package with case variations",
			files: map[string]string{
				"Test.nuspec":    "<?xml version='1.0'?><package></package>",
				"lib/TEST.dll":   "binary content",
				".SIGNATURE.P7S": "signature data",
			},
			signed:    true,
			wantEmpty: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := createTestPackageReader(t, tt.files)

			hash, err := GetPackageContentHash(pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPackageContentHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantEmpty {
				if hash != "" {
					t.Errorf("GetPackageContentHash() = %q, want empty for unsigned package", hash)
				}
			} else {
				if hash == "" {
					t.Errorf("GetPackageContentHash() returned empty, want hash for signed package")
				}

				// Verify hash is valid base64
				_, err := base64.StdEncoding.DecodeString(hash)
				if err != nil {
					t.Errorf("GetPackageContentHash() returned invalid base64: %v", err)
				}
			}
		})
	}
}

func TestGetPackageContentHash_Deterministic(t *testing.T) {
	// Hash should be deterministic for the same package file
	files := map[string]string{
		"test.nuspec":    "<?xml version='1.0'?><package></package>",
		"lib/test.dll":   "binary content",
		".signature.p7s": "signature data",
	}

	// Create package once
	pkg := createTestPackageReader(t, files)
	hash1, err := GetPackageContentHash(pkg)
	if err != nil {
		t.Fatalf("GetPackageContentHash() error = %v", err)
	}

	// Read same package again (seek to start)
	if _, err := pkg.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek error: %v", err)
	}
	hash2, err := GetPackageContentHash(pkg)
	if err != nil {
		t.Fatalf("GetPackageContentHash() error = %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("GetPackageContentHash() not deterministic:\n  hash1 = %s\n  hash2 = %s", hash1, hash2)
	}
}

func TestGetPackageContentHash_DifferentContent(t *testing.T) {
	// Different package content should produce different hashes
	files1 := map[string]string{
		"test.nuspec":    "<?xml version='1.0'?><package></package>",
		"lib/test.dll":   "content 1",
		".signature.p7s": "signature data",
	}

	files2 := map[string]string{
		"test.nuspec":    "<?xml version='1.0'?><package></package>",
		"lib/test.dll":   "content 2", // Different content
		".signature.p7s": "signature data",
	}

	pkg1 := createTestPackageReader(t, files1)
	hash1, err := GetPackageContentHash(pkg1)
	if err != nil {
		t.Fatalf("GetPackageContentHash() error = %v", err)
	}

	pkg2 := createTestPackageReader(t, files2)
	hash2, err := GetPackageContentHash(pkg2)
	if err != nil {
		t.Fatalf("GetPackageContentHash() error = %v", err)
	}

	if hash1 == hash2 {
		t.Errorf("GetPackageContentHash() produced same hash for different content")
	}
}

func TestGetPackageContentHash_SignatureExcluded(t *testing.T) {
	// Use real signed package from testdata
	path := "../testdata/TestPackage.AuthorSigned.1.0.0.nupkg"
	f, err := os.Open(path)
	if err != nil {
		t.Skipf("Test package not found: %v", err)
	}
	defer func() { _ = f.Close() }()

	// Get hash of real signed package
	hash, err := GetPackageContentHash(f)
	if err != nil {
		t.Fatalf("GetPackageContentHash() error = %v", err)
	}

	if hash == "" {
		t.Errorf("GetPackageContentHash() returned empty hash for signed package")
	}

	// Hash should be deterministic - read same package again
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek error: %v", err)
	}

	hash2, err := GetPackageContentHash(f)
	if err != nil {
		t.Fatalf("GetPackageContentHash() second call error = %v", err)
	}

	if hash != hash2 {
		t.Errorf("GetPackageContentHash() not deterministic:\n  hash1 = %s\n  hash2 = %s", hash, hash2)
	}
}

func TestReadSignedArchiveMetadata(t *testing.T) {
	files := map[string]string{
		"test.nuspec":    "<?xml version='1.0'?><package></package>",
		"lib/test.dll":   "binary content",
		".signature.p7s": "signature data",
	}

	pkg := createTestPackageReader(t, files)
	metadata, err := readSignedArchiveMetadata(pkg)
	if err != nil {
		t.Fatalf("readSignedArchiveMetadata() error = %v", err)
	}

	// Should have 3 central directory headers
	if len(metadata.CentralDirectoryHeaders) != 3 {
		t.Errorf("CentralDirectoryHeaders length = %d, want 3", len(metadata.CentralDirectoryHeaders))
	}

	// Should identify signature file
	if metadata.SignatureCentralDirectoryHeaderIndex < 0 {
		t.Errorf("SignatureCentralDirectoryHeaderIndex = %d, want >= 0", metadata.SignatureCentralDirectoryHeaderIndex)
	}

	// Verify signature header is marked correctly
	sigHeader := metadata.CentralDirectoryHeaders[metadata.SignatureCentralDirectoryHeaderIndex]
	if !sigHeader.IsPackageSignatureFile {
		t.Errorf("Signature header not marked as signature file")
	}

	// Verify other headers are not marked as signature
	for i, header := range metadata.CentralDirectoryHeaders {
		if i != metadata.SignatureCentralDirectoryHeaderIndex {
			if header.IsPackageSignatureFile {
				t.Errorf("Non-signature header %d marked as signature file", i)
			}
		}
	}
}

func TestReadSignedArchiveMetadata_UnsignedPackage(t *testing.T) {
	files := map[string]string{
		"test.nuspec":  "<?xml version='1.0'?><package></package>",
		"lib/test.dll": "binary content",
	}

	pkg := createTestPackageReader(t, files)
	metadata, err := readSignedArchiveMetadata(pkg)
	if err != nil {
		t.Fatalf("readSignedArchiveMetadata() error = %v", err)
	}

	// Should not find signature
	if metadata.SignatureCentralDirectoryHeaderIndex != -1 {
		t.Errorf("SignatureCentralDirectoryHeaderIndex = %d, want -1 for unsigned", metadata.SignatureCentralDirectoryHeaderIndex)
	}
}

func TestRemoveSignatureAndSortByOffset(t *testing.T) {
	metadata := &SignedPackageArchiveMetadata{
		CentralDirectoryHeaders: []CentralDirectoryHeaderMetadata{
			{OffsetToLocalFileHeader: 100, IsPackageSignatureFile: false},
			{OffsetToLocalFileHeader: 50, IsPackageSignatureFile: false},
			{OffsetToLocalFileHeader: 200, IsPackageSignatureFile: true}, // Signature
			{OffsetToLocalFileHeader: 150, IsPackageSignatureFile: false},
		},
		SignatureCentralDirectoryHeaderIndex: 2,
	}

	result := removeSignatureAndSortByOffset(metadata)

	// Should have 3 entries (signature removed)
	if len(result) != 3 {
		t.Errorf("removeSignatureAndSortByOffset() length = %d, want 3", len(result))
	}

	// Verify sorted by offset
	expectedOffsets := []int64{50, 100, 150}
	for i, entry := range result {
		if entry.OffsetToLocalFileHeader != expectedOffsets[i] {
			t.Errorf("Entry %d offset = %d, want %d", i, entry.OffsetToLocalFileHeader, expectedOffsets[i])
		}

		// Verify no signature entries
		if entry.IsPackageSignatureFile {
			t.Errorf("Entry %d is signature file, should be removed", i)
		}
	}
}

func TestFindEndOfCentralDirectory(t *testing.T) {
	files := map[string]string{
		"test.nuspec":  "<?xml version='1.0'?><package></package>",
		"lib/test.dll": "binary content",
	}

	pkg := createTestPackageReader(t, files)
	eocdr, offset, err := findEndOfCentralDirectory(pkg)
	if err != nil {
		t.Fatalf("findEndOfCentralDirectory() error = %v", err)
	}

	if eocdr == nil {
		t.Fatal("findEndOfCentralDirectory() returned nil EOCDR")
	}

	if eocdr.Signature != 0x06054b50 {
		t.Errorf("EOCDR signature = 0x%08x, want 0x06054b50", eocdr.Signature)
	}

	if offset < 0 {
		t.Errorf("EOCDR offset = %d, want >= 0", offset)
	}

	// Verify entry count matches files
	if eocdr.NumEntries != 2 {
		t.Errorf("EOCDR NumEntries = %d, want 2", eocdr.NumEntries)
	}
}

func TestReadCentralDirectoryHeader(t *testing.T) {
	files := map[string]string{
		"test.nuspec": "<?xml version='1.0'?><package></package>",
	}

	pkg := createTestPackageReader(t, files)

	// Find EOCDR first
	eocdr, _, err := findEndOfCentralDirectory(pkg)
	if err != nil {
		t.Fatalf("findEndOfCentralDirectory() error = %v", err)
	}

	// Seek to central directory
	if _, err := pkg.Seek(int64(eocdr.CentralDirectoryOffset), io.SeekStart); err != nil {
		t.Fatalf("Seek() error = %v", err)
	}

	// Read first central directory header
	header, err := readCentralDirectoryHeader(pkg)
	if err != nil {
		t.Fatalf("readCentralDirectoryHeader() error = %v", err)
	}

	if header == nil {
		t.Fatal("readCentralDirectoryHeader() returned nil")
	}

	if header.Signature != 0x02014b50 {
		t.Errorf("Header signature = 0x%08x, want 0x02014b50", header.Signature)
	}

	if header.FileName != "test.nuspec" {
		t.Errorf("Header FileName = %q, want %q", header.FileName, "test.nuspec")
	}
}

func TestReadLocalFileHeader(t *testing.T) {
	files := map[string]string{
		"test.nuspec": "<?xml version='1.0'?><package></package>",
	}

	pkg := createTestPackageReader(t, files)

	// Local file header is at the start
	if _, err := pkg.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek() error = %v", err)
	}

	header, err := readLocalFileHeader(pkg)
	if err != nil {
		t.Fatalf("readLocalFileHeader() error = %v", err)
	}

	if header == nil {
		t.Fatal("readLocalFileHeader() returned nil")
	}

	if header.Signature != 0x04034b50 {
		t.Errorf("Header signature = 0x%08x, want 0x04034b50", header.Signature)
	}
}

func TestHashUntilPosition(t *testing.T) {
	content := []byte("test content for hashing")
	reader := bytes.NewReader(content)
	hashWriter := new(bytes.Buffer)

	err := hashUntilPosition(reader, hashWriter, 10)
	if err != nil {
		t.Fatalf("hashUntilPosition() error = %v", err)
	}

	hashed := hashWriter.Bytes()
	if len(hashed) != 10 {
		t.Errorf("hashUntilPosition() hashed %d bytes, want 10", len(hashed))
	}

	if string(hashed) != "test conte" {
		t.Errorf("hashUntilPosition() content = %q, want %q", hashed, "test conte")
	}
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		name string
		a    int64
		b    int64
		want int64
	}{
		{"a smaller", 5, 10, 5},
		{"b smaller", 10, 5, 5},
		{"equal", 7, 7, 7},
		{"negative", -5, 3, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := min(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCentralDirectoryHeaderGetSizeInBytes(t *testing.T) {
	tests := []struct {
		name           string
		fileNameLen    uint16
		extraFieldLen  uint16
		fileCommentLen uint16
		expectedSize   uint32
	}{
		{"no extras", 10, 0, 0, 56},    // 46 + 10
		{"with extra", 10, 5, 0, 61},   // 46 + 10 + 5
		{"with comment", 10, 0, 3, 59}, // 46 + 10 + 3
		{"all fields", 15, 5, 8, 74},   // 46 + 15 + 5 + 8
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &centralDirectoryHeader{
				FileNameLength:    tt.fileNameLen,
				ExtraFieldLength:  tt.extraFieldLen,
				FileCommentLength: tt.fileCommentLen,
			}

			size := header.GetSizeInBytes()
			if size != tt.expectedSize {
				t.Errorf("GetSizeInBytes() = %d, want %d", size, tt.expectedSize)
			}
		})
	}
}

func TestGetPackageContentHash_LargePackage(t *testing.T) {
	// Test with larger content
	files := map[string]string{
		"test.nuspec":     "<?xml version='1.0'?><package></package>",
		"lib/large.dll":   strings.Repeat("A", 10000),
		"lib/another.dll": strings.Repeat("B", 10000),
		".signature.p7s":  "signature data",
	}

	pkg := createTestPackageReader(t, files)
	hash, err := GetPackageContentHash(pkg)
	if err != nil {
		t.Fatalf("GetPackageContentHash() error = %v", err)
	}

	if hash == "" {
		t.Errorf("GetPackageContentHash() returned empty for large package")
	}

	// Verify base64
	decoded, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		t.Errorf("Invalid base64 hash: %v", err)
	}

	// SHA512 produces 64 bytes
	if len(decoded) != 64 {
		t.Errorf("Hash length = %d bytes, want 64 (SHA512)", len(decoded))
	}
}

func TestGetPackageContentHash_EmptyFiles(t *testing.T) {
	files := map[string]string{
		"test.nuspec":    "<?xml version='1.0'?><package></package>",
		"lib/empty.dll":  "",
		".signature.p7s": "signature data",
	}

	pkg := createTestPackageReader(t, files)
	hash, err := GetPackageContentHash(pkg)
	if err != nil {
		t.Fatalf("GetPackageContentHash() error = %v", err)
	}

	if hash == "" {
		t.Errorf("GetPackageContentHash() returned empty for package with empty files")
	}
}

func BenchmarkGetPackageContentHash(b *testing.B) {
	files := map[string]string{
		"test.nuspec":    "<?xml version='1.0'?><package></package>",
		"lib/test.dll":   strings.Repeat("A", 1000),
		".signature.p7s": "signature data",
	}

	b.ResetTimer()
	for b.Loop() {
		pkg := createTestPackageReader(b, files)
		_, err := GetPackageContentHash(pkg)
		if err != nil {
			b.Fatalf("GetPackageContentHash() error = %v", err)
		}
	}
}

func BenchmarkReadSignedArchiveMetadata(b *testing.B) {
	files := map[string]string{
		"test.nuspec":    "<?xml version='1.0'?><package></package>",
		"lib/test1.dll":  "content 1",
		"lib/test2.dll":  "content 2",
		"lib/test3.dll":  "content 3",
		".signature.p7s": "signature data",
	}

	b.ResetTimer()
	for b.Loop() {
		pkg := createTestPackageReader(b, files)
		_, err := readSignedArchiveMetadata(pkg)
		if err != nil {
			b.Fatalf("readSignedArchiveMetadata() error = %v", err)
		}
	}
}

// Helper functions

func createTestPackageReader(tb testing.TB, files map[string]string) *bytes.Reader {
	tb.Helper()

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			tb.Fatalf("Failed to create file %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			tb.Fatalf("Failed to write file %s: %v", name, err)
		}
	}

	if err := w.Close(); err != nil {
		tb.Fatalf("Failed to close zip: %v", err)
	}

	return bytes.NewReader(buf.Bytes())
}
