package restore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheFile(t *testing.T) {
	cache := NewCacheFile("abc123")

	assert.Equal(t, CacheFileVersion, cache.Version)
	assert.Equal(t, "abc123", cache.DgSpecHash)
	assert.False(t, cache.Success)
	assert.Empty(t, cache.ExpectedPackageFiles)
	assert.Empty(t, cache.Logs)
}

func TestCacheFile_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		cache *CacheFile
		valid bool
	}{
		{
			name: "valid cache",
			cache: &CacheFile{
				Version:    CacheFileVersion,
				DgSpecHash: "abc123",
				Success:    true,
			},
			valid: true,
		},
		{
			name: "wrong version",
			cache: &CacheFile{
				Version:    1,
				DgSpecHash: "abc123",
				Success:    true,
			},
			valid: false,
		},
		{
			name: "empty hash",
			cache: &CacheFile{
				Version:    CacheFileVersion,
				DgSpecHash: "",
				Success:    true,
			},
			valid: false,
		},
		{
			name: "restore failed",
			cache: &CacheFile{
				Version:    CacheFileVersion,
				DgSpecHash: "abc123",
				Success:    false,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.cache.IsValid())
		})
	}
}

func TestCacheFile_Save_Load(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "obj", "project.nuget.cache")

	// Create cache file
	original := &CacheFile{
		Version:         CacheFileVersion,
		DgSpecHash:      "testHash123",
		Success:         true,
		ProjectFilePath: "/path/to/project.csproj",
		ExpectedPackageFiles: []string{
			"/packages/foo/1.0.0/foo.1.0.0.nupkg.sha512",
		},
		Logs: []LogMessage{},
	}

	// Save
	err := original.Save(cachePath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(cachePath)
	require.NoError(t, err)

	// Load
	loaded, err := LoadCacheFile(cachePath)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.Version, loaded.Version)
	assert.Equal(t, original.DgSpecHash, loaded.DgSpecHash)
	assert.Equal(t, original.Success, loaded.Success)
	assert.Equal(t, original.ProjectFilePath, loaded.ProjectFilePath)
	assert.Equal(t, original.ExpectedPackageFiles, loaded.ExpectedPackageFiles)
}

func TestLoadCacheFile_NotExists(t *testing.T) {
	cache, err := LoadCacheFile("/nonexistent/path/cache.json")
	require.NoError(t, err)
	assert.False(t, cache.IsValid())
}

func TestLoadCacheFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Write invalid JSON
	err := os.WriteFile(cachePath, []byte("not json"), 0644)
	require.NoError(t, err)

	// Should return invalid cache (not error)
	cache, err := LoadCacheFile(cachePath)
	require.NoError(t, err)
	assert.False(t, cache.IsValid())
}

func TestGetCacheFilePath(t *testing.T) {
	projectPath := "/path/to/MyProject/MyProject.csproj"
	expected := "/path/to/MyProject/obj/project.nuget.cache"

	result := GetCacheFilePath(projectPath)
	assert.Equal(t, expected, result)
}

func TestCacheFile_VerifyPackageFilesExist(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real package file
	pkgPath := filepath.Join(tmpDir, "packages", "foo", "1.0.0", "foo.1.0.0.nupkg.sha512")
	err := os.MkdirAll(filepath.Dir(pkgPath), 0755)
	require.NoError(t, err)
	err = os.WriteFile(pkgPath, []byte("dummy"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name   string
		files  []string
		exists bool
	}{
		{
			name:   "all files exist",
			files:  []string{pkgPath},
			exists: true,
		},
		{
			name:   "no files",
			files:  []string{},
			exists: true,
		},
		{
			name:   "file missing",
			files:  []string{"/nonexistent/package.sha512"},
			exists: false,
		},
		{
			name:   "some exist some missing",
			files:  []string{pkgPath, "/nonexistent/package.sha512"},
			exists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &CacheFile{
				ExpectedPackageFiles: tt.files,
			}
			assert.Equal(t, tt.exists, cache.VerifyPackageFilesExist())
		})
	}
}

func TestIsCacheValid(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Create package file
	pkgPath := filepath.Join(tmpDir, "packages", "foo.sha512")
	err := os.MkdirAll(filepath.Dir(pkgPath), 0755)
	require.NoError(t, err)
	err = os.WriteFile(pkgPath, []byte("dummy"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		cache       *CacheFile
		currentHash string
		valid       bool
	}{
		{
			name: "valid cache",
			cache: &CacheFile{
				Version:              CacheFileVersion,
				DgSpecHash:           "abc123",
				Success:              true,
				ExpectedPackageFiles: []string{pkgPath},
			},
			currentHash: "abc123",
			valid:       true,
		},
		{
			name: "hash mismatch",
			cache: &CacheFile{
				Version:              CacheFileVersion,
				DgSpecHash:           "abc123",
				Success:              true,
				ExpectedPackageFiles: []string{pkgPath},
			},
			currentHash: "xyz789",
			valid:       false,
		},
		{
			name: "package missing",
			cache: &CacheFile{
				Version:              CacheFileVersion,
				DgSpecHash:           "abc123",
				Success:              true,
				ExpectedPackageFiles: []string{"/nonexistent/package.sha512"},
			},
			currentHash: "abc123",
			valid:       false,
		},
		{
			name: "restore failed",
			cache: &CacheFile{
				Version:              CacheFileVersion,
				DgSpecHash:           "abc123",
				Success:              false,
				ExpectedPackageFiles: []string{pkgPath},
			},
			currentHash: "abc123",
			valid:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save cache
			err := tt.cache.Save(cachePath)
			require.NoError(t, err)

			// Validate
			valid, _, err := IsCacheValid(cachePath, tt.currentHash)
			require.NoError(t, err)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

func TestIsCacheValid_CacheNotExists(t *testing.T) {
	valid, cache, err := IsCacheValid("/nonexistent/cache.json", "hash123")
	require.NoError(t, err)
	assert.False(t, valid)
	assert.NotNil(t, cache)
	assert.False(t, cache.IsValid())
}
