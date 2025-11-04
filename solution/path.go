package solution

import (
	"path/filepath"
	"strings"
)

// PathResolver handles cross-platform path resolution
type PathResolver struct {
	// SolutionDir is the directory containing the solution file
	SolutionDir string
}

// NewPathResolver creates a new path resolver
func NewPathResolver(solutionDir string) *PathResolver {
	return &PathResolver{
		SolutionDir: solutionDir,
	}
}

// NormalizePath converts Windows-style paths to forward slash format
func NormalizePath(path string) string {
	if path == "" {
		return ""
	}

	// Check if this is a UNC path (starts with \\ or //)
	isUNC := strings.HasPrefix(path, "\\\\") || strings.HasPrefix(path, "//")

	// Convert backslashes to forward slashes
	normalized := strings.ReplaceAll(path, "\\", "/")

	// For UNC paths, preserve the leading double slash
	if isUNC {
		// Ensure it starts with exactly two slashes
		normalized = strings.TrimLeft(normalized, "/")
		normalized = "//" + normalized
	}

	// Remove duplicate slashes (but preserve UNC prefix)
	if isUNC && len(normalized) > 2 {
		// For UNC paths, clean everything after the initial //
		prefix := "//"
		remainder := normalized[2:]
		for strings.Contains(remainder, "//") {
			remainder = strings.ReplaceAll(remainder, "//", "/")
		}
		normalized = prefix + remainder
	} else if !isUNC {
		// For non-UNC paths, remove all duplicate slashes
		for strings.Contains(normalized, "//") {
			normalized = strings.ReplaceAll(normalized, "//", "/")
		}
	}

	return normalized
}

// ResolvePath resolves a project path relative to the solution directory
func (r *PathResolver) ResolvePath(projectPath string) string {
	// Normalize the path first
	normalized := NormalizePath(projectPath)

	// If already absolute, return as-is
	if filepath.IsAbs(normalized) {
		return normalized
	}

	// Resolve relative to solution directory
	resolved := filepath.Join(r.SolutionDir, normalized)
	return filepath.Clean(resolved)
}

// ConvertToSystemPath converts a path to the current OS format
func ConvertToSystemPath(path string) string {
	// First normalize to forward slashes
	normalized := NormalizePath(path)

	// Then convert to OS-specific separators
	return filepath.FromSlash(normalized)
}

// ResolveProjectPath resolves a project path from a solution file
func ResolveProjectPath(solutionDir, projectPath string) string {
	if projectPath == "" {
		return ""
	}

	// Normalize the project path
	normalized := NormalizePath(projectPath)

	// If the path is already absolute, clean and return it
	if filepath.IsAbs(normalized) {
		return filepath.Clean(normalized)
	}

	// Resolve relative to solution directory
	resolved := filepath.Join(solutionDir, normalized)
	return filepath.Clean(resolved)
}
