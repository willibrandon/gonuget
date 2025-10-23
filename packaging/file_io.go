package packaging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// FileIOMode represents Unix file permissions.
type FileIOMode uint32

const (
	// UnixFileMode is the default permission for extracted files (0766 octal = rwxrw-rw-)
	// Reference: NuGetExtractionFileIO.cs DefaultFileMode
	// Matches .NET Core 1.x behavior for backward compatibility
	UnixFileMode FileIOMode = 0766
)

// CreateFile creates a file with platform-specific permissions.
// Reference: NuGetExtractionFileIO.CreateFile in NuGet.Packaging
func CreateFile(path string) (*os.File, error) {
	// Create parent directories if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	// Platform-specific file creation
	if runtime.GOOS == "windows" {
		// Windows: Standard file creation
		return os.Create(path)
	}

	// Unix/Linux/macOS: Create with specific permissions
	// Apply umask by opening with 0666, OS applies umask
	// Then chmod to desired permissions
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}

	// Set executable bit (important for tools in packages)
	// 0766 = rwxrw-rw- (owner execute, group/other read/write)
	if err := os.Chmod(path, os.FileMode(UnixFileMode)); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("chmod: %w", err)
	}

	return file, nil
}

// CopyToFile copies stream to file with optimizations.
// Reference: StreamExtensions.CopyToFile in NuGet.Packaging
func CopyToFile(stream io.Reader, fileFullPath string) (string, error) {
	// Check if this is a directory entry (path ends with slash or base is ".")
	base := filepath.Base(fileFullPath)
	isDirectory := base == "" || base == "." || base == string(filepath.Separator) ||
		strings.HasSuffix(fileFullPath, "/") || strings.HasSuffix(fileFullPath, "\\")

	if isDirectory {
		// Clean the path to ensure directory format
		dirPath := filepath.Clean(fileFullPath)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return "", fmt.Errorf("create directory: %w", err)
		}
		// Return original path format to match test expectations
		return fileFullPath, nil
	}

	// Skip if file already exists
	if _, err := os.Stat(fileFullPath); err == nil {
		return fileFullPath, nil
	}

	// Create file with platform-specific permissions
	file, err := CreateFile(fileFullPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Copy stream to file
	// Note: Could add memory-mapped file optimization for small files (<10MB)
	// Environment variable: GONUGET_ENABLE_MMAP
	if _, err := io.Copy(file, stream); err != nil {
		return "", fmt.Errorf("copy stream: %w", err)
	}

	return fileFullPath, nil
}

// UpdateFileTimeFromEntry updates file timestamp with retry logic.
// Reference: ZipArchiveExtensions.UpdateFileTimeFromEntry in NuGet.Packaging
func UpdateFileTimeFromEntry(fileFullPath string, modTime time.Time, logger Logger) error {
	// Validate timestamp
	if modTime.IsZero() || modTime.Year() < 1980 || modTime.Year() > 2100 {
		// Skip invalid or future timestamps
		return nil
	}

	// Retry logic with exponential backoff
	maxRetries := 9 // Configurable via environment variable
	if retryStr := os.Getenv("GONUGET_UPDATEFILETIME_MAXRETRIES"); retryStr != "" {
		_, _ = fmt.Sscanf(retryStr, "%d", &maxRetries)
	}

	var lastErr error
	for retry := 0; retry <= maxRetries; retry++ {
		if err := os.Chtimes(fileFullPath, modTime, modTime); err == nil {
			return nil
		} else {
			lastErr = err
			if retry < maxRetries {
				// Exponential backoff: 1ms, 2ms, 4ms, 8ms, ...
				time.Sleep(time.Duration(1<<uint(retry)) * time.Millisecond)
			}
		}
	}

	// Log warning on failure (don't fail extraction)
	if logger != nil {
		logger.Warning("Failed to update file time for %s after %d retries: %v",
			fileFullPath, maxRetries+1, lastErr)
	}

	return nil
}
