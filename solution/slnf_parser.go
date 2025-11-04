package solution

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SlnfParser parses JSON-based .slnf solution filter files
type SlnfParser struct{}

// NewSlnfParser creates a new .slnf file parser
func NewSlnfParser() *SlnfParser {
	return &SlnfParser{}
}

// CanParse checks if this parser supports the given file
func (p *SlnfParser) CanParse(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".slnf"
}

// slnfDocument represents the JSON structure of a .slnf file
type slnfDocument struct {
	Solution slnfSolution `json:"solution"`
}

// slnfSolution contains the solution reference and filtered projects
type slnfSolution struct {
	Path     string   `json:"path"`
	Projects []string `json:"projects"`
}

// Parse reads and parses a .slnf file
func (p *SlnfParser) Parse(path string) (*Solution, error) {
	if !p.CanParse(path) {
		return nil, &ParseError{
			FilePath: path,
			Message:  "not a .slnf file",
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, &ParseError{
			FilePath: path,
			Message:  fmt.Sprintf("cannot open file: %v", err),
		}
	}
	defer func() { _ = file.Close() }()

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	filterDir := filepath.Dir(absPath)

	// Parse JSON
	decoder := json.NewDecoder(file)
	var doc slnfDocument
	if err := decoder.Decode(&doc); err != nil {
		return nil, &ParseError{
			FilePath: absPath,
			Message:  fmt.Sprintf("failed to parse JSON: %v", err),
		}
	}

	// Validate the filter document
	if doc.Solution.Path == "" {
		return nil, &ParseError{
			FilePath: absPath,
			Message:  "missing solution path in filter file",
		}
	}

	// Resolve the parent solution path
	solutionPath := NormalizePath(doc.Solution.Path)
	if !filepath.IsAbs(solutionPath) {
		solutionPath = filepath.Join(filterDir, solutionPath)
	}
	solutionPath = filepath.Clean(solutionPath)

	// Check if the parent solution exists
	if _, err := os.Stat(solutionPath); err != nil {
		if os.IsNotExist(err) {
			return nil, &ParseError{
				FilePath: absPath,
				Message:  fmt.Sprintf("parent solution file not found: %s", solutionPath),
			}
		}
		return nil, &ParseError{
			FilePath: absPath,
			Message:  fmt.Sprintf("cannot access parent solution: %v", err),
		}
	}

	// Parse the parent solution
	parentParser, err := GetParser(solutionPath)
	if err != nil {
		return nil, &ParseError{
			FilePath: absPath,
			Message:  fmt.Sprintf("unsupported parent solution format: %v", err),
		}
	}

	parentSolution, err := parentParser.Parse(solutionPath)
	if err != nil {
		return nil, &ParseError{
			FilePath: absPath,
			Message:  fmt.Sprintf("failed to parse parent solution: %v", err),
		}
	}

	// Create a filtered solution
	filteredSolution := &Solution{
		FilePath:                   absPath, // Use the filter file path
		SolutionDir:                parentSolution.SolutionDir,
		FormatVersion:              parentSolution.FormatVersion,
		VisualStudioVersion:        parentSolution.VisualStudioVersion,
		MinimumVisualStudioVersion: parentSolution.MinimumVisualStudioVersion,
		Projects:                   []Project{},
		SolutionFolders:            parentSolution.SolutionFolders, // Keep all folders
	}

	// Build a map of filtered project paths for quick lookup
	filterMap := make(map[string]bool)
	for _, projPath := range doc.Solution.Projects {
		// Normalize the project path
		normalized := NormalizePath(projPath)
		filterMap[normalized] = true
	}

	// Filter projects from the parent solution
	for _, project := range parentSolution.Projects {
		// Check if this project is in the filter
		if filterMap[project.Path] {
			filteredSolution.Projects = append(filteredSolution.Projects, project)
		}
	}

	return filteredSolution, nil
}
