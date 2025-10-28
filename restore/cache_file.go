package restore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CacheFile represents project.nuget.cache file structure.
// Matches NuGet.ProjectModel.CacheFile in NuGet.Client.
type CacheFile struct {
	// Version is always 2 (current cache format version)
	Version int `json:"version"`

	// DgSpecHash is base64-encoded hash of dependency graph spec
	DgSpecHash string `json:"dgSpecHash"`

	// Success indicates whether restore succeeded
	Success bool `json:"success"`

	// ProjectFilePath is absolute path to project file
	ProjectFilePath string `json:"projectFilePath"`

	// ExpectedPackageFiles lists .nupkg.sha512 paths that must exist
	ExpectedPackageFiles []string `json:"expectedPackageFiles"`

	// Logs contains warnings/errors from restore (for replay on cache hit)
	Logs []LogMessage `json:"logs"`
}

// LogMessage represents a log entry in the cache file.
type LogMessage struct {
	Level   string `json:"level"`   // "Warning", "Error", etc.
	Code    string `json:"code"`    // "NU1001", etc.
	Message string `json:"message"` // Full message text
}

const (
	// CacheFileVersion matches NuGet.ProjectModel.CacheFile.CurrentVersion
	CacheFileVersion = 2

	// CacheFileName matches NoOpRestoreUtilities.NoOpCacheFileName
	CacheFileName = "project.nuget.cache"
)

// NewCacheFile creates a new cache file with the given hash.
func NewCacheFile(dgSpecHash string) *CacheFile {
	return &CacheFile{
		Version:              CacheFileVersion,
		DgSpecHash:           dgSpecHash,
		Success:              false,
		ExpectedPackageFiles: []string{},
		Logs:                 []LogMessage{},
	}
}

// IsValid returns true if cache file is valid (version matches and restore succeeded).
// Matches CacheFile.IsValid property in NuGet.Client.
func (c *CacheFile) IsValid() bool {
	return c.Version == CacheFileVersion && c.Success && c.DgSpecHash != ""
}

// Save writes cache file to disk at the given path.
// Matches CacheFileFormat.Write in NuGet.Client.
func (c *CacheFile) Save(path string) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	// Marshal to JSON with indentation (matches dotnet format)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache file: %w", err)
	}

	// Write atomically
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Clean up on failure
		return fmt.Errorf("rename cache file: %w", err)
	}

	return nil
}

// LoadCacheFile reads cache file from disk.
// Matches CacheFileFormat.Read in NuGet.Client.
func LoadCacheFile(path string) (*CacheFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return invalid cache if file doesn't exist
			return NewCacheFile(""), nil
		}
		return nil, fmt.Errorf("read cache file: %w", err)
	}

	var cache CacheFile
	if err := json.Unmarshal(data, &cache); err != nil {
		// Return invalid cache on parse error (matches dotnet behavior)
		return NewCacheFile(""), nil
	}

	return &cache, nil
}

// GetCacheFilePath returns path to project.nuget.cache for a project.
// Matches NoOpRestoreUtilities.GetProjectCacheFilePath.
func GetCacheFilePath(projectPath string) string {
	objDir := filepath.Join(filepath.Dir(projectPath), "obj")
	return filepath.Join(objDir, CacheFileName)
}

// VerifyPackageFilesExist checks if all expected package files exist on disk.
// Matches NoOpRestoreUtilities.VerifyRestoreOutput in NuGet.Client.
func (c *CacheFile) VerifyPackageFilesExist() bool {
	for _, pkgPath := range c.ExpectedPackageFiles {
		if _, err := os.Stat(pkgPath); err != nil {
			return false
		}
	}
	return true
}

// IsCacheValid checks if cache can be used (hash matches + files exist).
// Matches the logic in RestoreCommand.EvaluateCacheFile (line 1360).
func IsCacheValid(cachePath string, currentHash string) (bool, *CacheFile, error) {
	// Load cache file
	cache, err := LoadCacheFile(cachePath)
	if err != nil {
		return false, nil, err
	}

	// Check if cache is structurally valid
	if !cache.IsValid() {
		return false, cache, nil
	}

	// Check if hash matches
	if cache.DgSpecHash != currentHash {
		return false, cache, nil
	}

	// Check if all package files exist
	if !cache.VerifyPackageFilesExist() {
		return false, cache, nil
	}

	return true, cache, nil
}
