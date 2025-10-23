package packaging

import (
	"encoding/json"
	"fmt"
	"os"
)

// NupkgMetadataFile represents .nupkg.metadata file content.
// Reference: NupkgMetadataFile class in NuGet.Packaging
type NupkgMetadataFile struct {
	Version     int    `json:"version"`     // Format version (currently 2)
	ContentHash string `json:"contentHash"` // Base64-encoded SHA512 hash
	Source      string `json:"source"`      // Source URL
}

// NewNupkgMetadataFile creates metadata with current version.
func NewNupkgMetadataFile(contentHash, source string) *NupkgMetadataFile {
	return &NupkgMetadataFile{
		Version:     2, // Current format version
		ContentHash: contentHash,
		Source:      source,
	}
}

// WriteToFile writes metadata as JSON.
func (m *NupkgMetadataFile) WriteToFile(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write metadata file: %w", err)
	}

	return nil
}

// ReadNupkgMetadataFile reads metadata from JSON file.
func ReadNupkgMetadataFile(path string) (*NupkgMetadataFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata file: %w", err)
	}

	var metadata NupkgMetadataFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &metadata, nil
}
