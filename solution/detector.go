package solution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Detector helps detect and identify solution files
type Detector struct {
	// SearchDir is the directory to search for solution files
	SearchDir string
}

// NewDetector creates a new solution file detector
func NewDetector(searchDir string) *Detector {
	if searchDir == "" {
		searchDir = "."
	}
	return &Detector{SearchDir: searchDir}
}

// IsSolutionFile checks if a file path has a solution file extension
func IsSolutionFile(path string) bool {
	if path == "" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".sln" || ext == ".slnx" || ext == ".slnf"
}

// IsProjectFile checks if a file path has a project file extension
func IsProjectFile(path string) bool {
	if path == "" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".csproj" || ext == ".vbproj" || ext == ".fsproj"
}

// GetSolutionFormat returns the solution format based on file extension
func GetSolutionFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".sln":
		return "sln"
	case ".slnx":
		return "slnx"
	case ".slnf":
		return "slnf"
	default:
		return ""
	}
}

// DetectionResult contains the result of solution file detection
type DetectionResult struct {
	// Found indicates if any solution file was found
	Found bool

	// Ambiguous indicates if multiple solution files were found
	Ambiguous bool

	// SolutionPath is the path to the found solution file
	SolutionPath string

	// FoundFiles lists all solution files found
	FoundFiles []string

	// Format is the detected solution format
	Format string
}

// DetectSolution searches for solution files in the configured directory
func (d *Detector) DetectSolution() (*DetectionResult, error) {
	result := &DetectionResult{
		FoundFiles: []string{},
	}

	// Search for solution files
	err := filepath.Walk(d.SearchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't read
			if os.IsPermission(err) {
				return nil
			}
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Don't recurse into hidden directories or build directories
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "bin" || name == "obj" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a solution file
		if IsSolutionFile(path) {
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			result.FoundFiles = append(result.FoundFiles, absPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error searching for solution files: %w", err)
	}

	// Analyze results
	switch len(result.FoundFiles) {
	case 0:
		result.Found = false
		return result, nil
	case 1:
		result.Found = true
		result.SolutionPath = result.FoundFiles[0]
		result.Format = GetSolutionFormat(result.SolutionPath)
		return result, nil
	default:
		result.Found = true
		result.Ambiguous = true
		return result, nil
	}
}

// ValidateSolutionFile checks if a solution file exists and is readable
func ValidateSolutionFile(path string) error {
	if !IsSolutionFile(path) {
		return fmt.Errorf("not a solution file (must have .sln, .slnx, or .slnf extension): %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("solution file not found: %s", path)
		}
		return fmt.Errorf("cannot access solution file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a solution file: %s", path)
	}

	// Try to open the file to ensure it's readable
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot read solution file: %w", err)
	}
	_ = file.Close()

	return nil
}
