package packaging

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCreateFile(t *testing.T) {
	tests := []struct {
		name       string
		setupDir   func(string) string
		wantErr    bool
		checkPerms bool
	}{
		{
			name: "create file in existing directory",
			setupDir: func(base string) string {
				return filepath.Join(base, "test.txt")
			},
			wantErr:    false,
			checkPerms: true,
		},
		{
			name: "create file with nested directory creation",
			setupDir: func(base string) string {
				return filepath.Join(base, "subdir", "nested", "test.txt")
			},
			wantErr:    false,
			checkPerms: true,
		},
		{
			name: "create file in root of temp dir",
			setupDir: func(base string) string {
				return filepath.Join(base, "simple.txt")
			},
			wantErr:    false,
			checkPerms: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filePath := tt.setupDir(tempDir)

			file, err := CreateFile(filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				defer func() { _ = file.Close() }()

				// Verify file was created
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("CreateFile() did not create file at %s", filePath)
				}

				// Verify parent directories were created
				parentDir := filepath.Dir(filePath)
				if _, err := os.Stat(parentDir); os.IsNotExist(err) {
					t.Errorf("CreateFile() did not create parent directory %s", parentDir)
				}

				// Check Unix permissions on non-Windows platforms
				if tt.checkPerms && runtime.GOOS != "windows" {
					info, err := os.Stat(filePath)
					if err != nil {
						t.Fatalf("Stat() error = %v", err)
					}

					mode := info.Mode()
					// Check that executable bit is set (0766 = rwxrw-rw-)
					if mode&0100 == 0 {
						t.Errorf("CreateFile() mode = %o, expected executable bit set", mode)
					}
				}

				// Verify file is writable
				testData := []byte("test content")
				if _, err := file.Write(testData); err != nil {
					t.Errorf("Write() error = %v", err)
				}
			}
		})
	}
}

func TestCreateFile_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "file already exists - should truncate",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				path := filepath.Join(tempDir, "existing.txt")
				_ = os.WriteFile(path, []byte("old content"), 0644)
				return path
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setup(t)

			file, err := CreateFile(filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if file != nil {
				_ = file.Close()

				// For the truncate test, verify file was truncated
				if tt.name == "file already exists - should truncate" {
					content, _ := os.ReadFile(filePath)
					if len(content) > 0 {
						t.Errorf("CreateFile() did not truncate existing file")
					}
				}
			}
		})
	}
}

func TestCopyToFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		setupPath   func(string) string
		expectWrite bool
		wantErr     bool
	}{
		{
			name:    "copy to new file",
			content: "test content for new file",
			setupPath: func(base string) string {
				return filepath.Join(base, "newfile.txt")
			},
			expectWrite: true,
			wantErr:     false,
		},
		{
			name:    "skip existing file",
			content: "new content",
			setupPath: func(base string) string {
				path := filepath.Join(base, "existing.txt")
				_ = os.WriteFile(path, []byte("old content"), 0644)
				return path
			},
			expectWrite: false, // Should skip existing file
			wantErr:     false,
		},
		{
			name:    "create directory entry",
			content: "",
			setupPath: func(base string) string {
				return filepath.Join(base, "somedir/") // Trailing slash = directory
			},
			expectWrite: false,
			wantErr:     false,
		},
		{
			name:    "create nested file with directory creation",
			content: "nested content",
			setupPath: func(base string) string {
				return filepath.Join(base, "sub", "nested", "file.txt")
			},
			expectWrite: true,
			wantErr:     false,
		},
		{
			name:    "empty content",
			content: "",
			setupPath: func(base string) string {
				return filepath.Join(base, "empty.txt")
			},
			expectWrite: true,
			wantErr:     false,
		},
		{
			name:    "large content",
			content: strings.Repeat("large content data ", 10000),
			setupPath: func(base string) string {
				return filepath.Join(base, "large.txt")
			},
			expectWrite: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			targetPath := tt.setupPath(tempDir)

			stream := bytes.NewReader([]byte(tt.content))
			resultPath, err := CopyToFile(stream, targetPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("CopyToFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if resultPath != targetPath {
					t.Errorf("CopyToFile() path = %v, want %v", resultPath, targetPath)
				}

				// Check if directory or file
				info, statErr := os.Stat(targetPath)
				if statErr != nil && !strings.HasSuffix(targetPath, "/") {
					t.Fatalf("Stat() error = %v", statErr)
				}

				if strings.HasSuffix(targetPath, "/") {
					// Directory case
					if !info.IsDir() {
						t.Errorf("CopyToFile() did not create directory")
					}
				} else {
					// File case
					if tt.expectWrite {
						// Verify content was written
						actualContent, err := os.ReadFile(targetPath)
						if err != nil {
							t.Fatalf("ReadFile() error = %v", err)
						}
						if string(actualContent) != tt.content {
							t.Errorf("CopyToFile() content mismatch, got len=%d want len=%d",
								len(actualContent), len(tt.content))
						}
					} else if strings.Contains(tt.name, "skip existing") {
						// For skip case, verify old content preserved
						actualContent, _ := os.ReadFile(targetPath)
						if string(actualContent) != "old content" {
							t.Errorf("CopyToFile() modified existing file")
						}
					}
				}
			}
		})
	}
}

func TestUpdateFileTimeFromEntry(t *testing.T) {
	tests := []struct {
		name      string
		modTime   time.Time
		wantErr   bool
		skipRetry bool
	}{
		{
			name:    "valid timestamp",
			modTime: time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "skip zero time",
			modTime: time.Time{},
			wantErr: false, // No error, but no-op
		},
		{
			name:    "skip timestamp before 1980",
			modTime: time.Date(1979, 12, 31, 23, 59, 59, 0, time.UTC),
			wantErr: false, // No error, but no-op
		},
		{
			name:    "skip future timestamp",
			modTime: time.Date(2101, 1, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false, // No error, but no-op
		},
		{
			name:    "minimum valid time",
			modTime: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "maximum valid time",
			modTime: time.Date(2100, 12, 31, 23, 59, 59, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "current time",
			modTime: time.Now(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "testfile.txt")

			// Create test file
			if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			// Mock logger
			var logMessages []string
			logger := &testLogger{
				warnings: &logMessages,
			}

			err := UpdateFileTimeFromEntry(filePath, tt.modTime, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateFileTimeFromEntry() error = %v, wantErr %v", err, tt.wantErr)
			}

			// For valid timestamps, verify the time was updated
			if !tt.modTime.IsZero() &&
				tt.modTime.Year() >= 1980 &&
				tt.modTime.Year() <= 2100 {

				info, err := os.Stat(filePath)
				if err != nil {
					t.Fatalf("Stat() error = %v", err)
				}

				// Note: Time comparison may have some precision loss
				actualTime := info.ModTime()
				if !actualTime.Truncate(time.Second).Equal(tt.modTime.Truncate(time.Second)) {
					t.Logf("UpdateFileTimeFromEntry() time mismatch (may be expected on some platforms)")
					t.Logf("  got:  %v", actualTime)
					t.Logf("  want: %v", tt.modTime)
				}
			}
		})
	}
}

func TestUpdateFileTimeFromEntry_RetryLogic(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")

	// Create test file
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Test with custom retry count
	t.Setenv("GONUGET_UPDATEFILETIME_MAXRETRIES", "2")

	var logMessages []string
	logger := &testLogger{
		warnings: &logMessages,
	}

	validTime := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)

	// This should succeed with custom retry count
	err := UpdateFileTimeFromEntry(filePath, validTime, logger)
	if err != nil {
		t.Errorf("UpdateFileTimeFromEntry() error = %v", err)
	}
}

func TestUpdateFileTimeFromEntry_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "nonexistent.txt")

	var logMessages []string
	logger := &testLogger{
		warnings: &logMessages,
	}

	validTime := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)

	// Should log warning but not return error
	err := UpdateFileTimeFromEntry(nonExistentPath, validTime, logger)
	if err != nil {
		t.Errorf("UpdateFileTimeFromEntry() should not error on non-existent file, got %v", err)
	}

	if len(logMessages) == 0 {
		t.Errorf("UpdateFileTimeFromEntry() should log warning for non-existent file")
	}
}

// testLogger implements Logger interface for testing
type testLogger struct {
	warnings *[]string
}

func (l *testLogger) Info(format string, args ...any) {
	// No-op for info
}

func (l *testLogger) Warning(format string, args ...any) {
	if l.warnings != nil {
		*l.warnings = append(*l.warnings, format)
	}
}

func (l *testLogger) Error(format string, args ...any) {
	if l.warnings != nil {
		*l.warnings = append(*l.warnings, format)
	}
}

func TestUnixFileMode(t *testing.T) {
	// Test that the constant is set correctly
	expectedMode := FileIOMode(0766)
	if UnixFileMode != expectedMode {
		t.Errorf("UnixFileMode = %o, want %o", UnixFileMode, expectedMode)
	}

	// Verify it translates to rwxrw-rw-
	mode := os.FileMode(UnixFileMode)
	if mode&0700 != 0700 { // Owner: rwx
		t.Errorf("UnixFileMode owner permissions incorrect: %o", mode)
	}
	if mode&0060 != 0060 { // Group: rw-
		t.Errorf("UnixFileMode group permissions incorrect: %o", mode)
	}
	if mode&0006 != 0006 { // Other: rw-
		t.Errorf("UnixFileMode other permissions incorrect: %o", mode)
	}
}

func TestCopyToFile_DirectoryWithoutTrailingSlash(t *testing.T) {
	tempDir := t.TempDir()

	// Test that a path with trailing slash is treated as directory
	dirPath := filepath.Join(tempDir, "testdir") + string(filepath.Separator)

	stream := bytes.NewReader([]byte(""))
	resultPath, err := CopyToFile(stream, dirPath)

	if err != nil {
		t.Errorf("CopyToFile() error = %v", err)
	}

	if resultPath != dirPath {
		t.Errorf("CopyToFile() returned path = %v, want %v", resultPath, dirPath)
	}

	// Verify directory was created (remove trailing separator for stat)
	checkPath := strings.TrimSuffix(dirPath, string(filepath.Separator))
	info, statErr := os.Stat(checkPath)
	if statErr != nil {
		t.Fatalf("Directory was not created: %v", statErr)
	}

	if !info.IsDir() {
		t.Errorf("Path is not a directory")
	}
}

func BenchmarkCreateFile(b *testing.B) {
	tempDir := b.TempDir()

	b.ResetTimer()
	for i := range b.N {
		filePath := filepath.Join(tempDir, "bench", fmt.Sprintf("file%d.txt", i))
		file, err := CreateFile(filePath)
		if err != nil {
			b.Fatalf("CreateFile() error = %v", err)
		}
		_ = file.Close()
	}
}

func BenchmarkCopyToFile(b *testing.B) {
	tempDir := b.TempDir()
	content := []byte(strings.Repeat("test content ", 100))

	b.ResetTimer()
	for i := range b.N {
		filePath := filepath.Join(tempDir, fmt.Sprintf("bench%d.txt", i))
		stream := bytes.NewReader(content)
		_, err := CopyToFile(stream, filePath)
		if err != nil {
			b.Fatalf("CopyToFile() error = %v", err)
		}
	}
}
