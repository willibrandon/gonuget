package solution

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SlnxParser parses XML-based .slnx files (introduced in .NET 9)
type SlnxParser struct{}

// NewSlnxParser creates a new .slnx file parser
func NewSlnxParser() *SlnxParser {
	return &SlnxParser{}
}

// CanParse checks if this parser supports the given file
func (p *SlnxParser) CanParse(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".slnx"
}

// slnxDocument represents the root element of a .slnx file
type slnxDocument struct {
	XMLName xml.Name      `xml:"Solution"`
	Folders []slnxFolder  `xml:"Folder"`
	Props   slnxProps     `xml:"Properties"`
}

// slnxFolder represents a folder element in .slnx
type slnxFolder struct {
	Name     string         `xml:"Name,attr"`
	Projects []slnxProject  `xml:"Project"`
	Folders  []slnxFolder   `xml:"Folder"` // Nested folders
	Files    []slnxFile     `xml:"File"`   // Solution items
}

// slnxProject represents a project reference in .slnx
type slnxProject struct {
	Path string `xml:"Path,attr"`
	Name string `xml:"Name,attr,omitempty"` // Optional display name
}

// slnxFile represents a file reference in a solution folder
type slnxFile struct {
	Path string `xml:"Path,attr"`
}

// slnxProps represents solution properties
type slnxProps struct {
	Properties []slnxProperty `xml:"Property"`
}

// slnxProperty represents a single property
type slnxProperty struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:"Value,attr"`
}

// Parse reads and parses a .slnx file
func (p *SlnxParser) Parse(path string) (*Solution, error) {
	if !p.CanParse(path) {
		return nil, &ParseError{
			FilePath: path,
			Message:  "not a .slnx file",
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, &ParseError{
			FilePath: path,
			Message:  fmt.Sprintf("cannot open file: %v", err),
		}
	}
	defer file.Close()

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Parse XML
	decoder := xml.NewDecoder(file)
	var doc slnxDocument
	if err := decoder.Decode(&doc); err != nil {
		// Try to provide a better error message
		if syntaxErr, ok := err.(*xml.SyntaxError); ok {
			return nil, &ParseError{
				FilePath: absPath,
				Line:     syntaxErr.Line,
				Message:  fmt.Sprintf("XML syntax error: %v", syntaxErr.Msg),
			}
		}
		return nil, &ParseError{
			FilePath: absPath,
			Message:  fmt.Sprintf("failed to parse XML: %v", err),
		}
	}

	// Convert to Solution structure
	sol := &Solution{
		FilePath:        absPath,
		SolutionDir:     filepath.Dir(absPath),
		Projects:        []Project{},
		SolutionFolders: []SolutionFolder{},
		FormatVersion:   "12.00", // .slnx is equivalent to modern .sln format
	}

	// Extract properties
	for _, prop := range doc.Props.Properties {
		switch prop.Name {
		case "VisualStudioVersion":
			sol.VisualStudioVersion = prop.Value
		case "MinimumVisualStudioVersion":
			sol.MinimumVisualStudioVersion = prop.Value
		}
	}

	// Process folders recursively
	guidCounter := 1
	for _, folder := range doc.Folders {
		p.processFolder(&folder, sol, "", &guidCounter)
	}

	return sol, nil
}

// processFolder recursively processes a folder and its contents
func (p *SlnxParser) processFolder(folder *slnxFolder, sol *Solution, parentGUID string, guidCounter *int) {
	// Create a GUID for this folder
	folderGUID := p.generateGUID(*guidCounter)
	*guidCounter++

	// Add the solution folder
	solutionFolder := SolutionFolder{
		Name:             folder.Name,
		GUID:             folderGUID,
		ParentFolderGUID: parentGUID,
		Items:            []string{},
	}

	// Add solution item files
	for _, file := range folder.Files {
		solutionFolder.Items = append(solutionFolder.Items, NormalizePath(file.Path))
	}

	if folder.Name != "" { // Only add named folders
		sol.SolutionFolders = append(sol.SolutionFolders, solutionFolder)
	}

	// Process projects in this folder
	for _, proj := range folder.Projects {
		project := p.convertProject(proj, folderGUID, guidCounter)
		if folder.Name != "" {
			project.ParentFolderGUID = folderGUID
		}
		sol.Projects = append(sol.Projects, project)
	}

	// Recursively process nested folders
	for _, nestedFolder := range folder.Folders {
		p.processFolder(&nestedFolder, sol, folderGUID, guidCounter)
	}
}

// convertProject converts a slnxProject to a Project
func (p *SlnxParser) convertProject(proj slnxProject, parentGUID string, guidCounter *int) Project {
	// Normalize the path
	projectPath := NormalizePath(proj.Path)

	// Extract project name from path if not provided
	projectName := proj.Name
	if projectName == "" {
		projectName = strings.TrimSuffix(filepath.Base(projectPath), filepath.Ext(projectPath))
	}

	// Generate GUID
	projectGUID := p.generateGUID(*guidCounter)
	*guidCounter++

	// Determine project type from extension
	typeGUID := p.getProjectTypeFromPath(projectPath)

	return Project{
		Name:             projectName,
		Path:             projectPath,
		GUID:             projectGUID,
		TypeGUID:         typeGUID,
		ParentFolderGUID: parentGUID,
	}
}

// getProjectTypeFromPath determines the project type GUID based on file extension
func (p *SlnxParser) getProjectTypeFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csproj":
		// Use SDK-style project GUID for modern projects
		return ProjectTypeCSProjectSDK
	case ".vbproj":
		return ProjectTypeVBProject
	case ".fsproj":
		return ProjectTypeFSProject
	default:
		// Default to SDK-style C# project for unknown extensions
		return ProjectTypeCSProjectSDK
	}
}

// generateGUID generates a placeholder GUID for items that don't have one
func (p *SlnxParser) generateGUID(counter int) string {
	// Generate a deterministic GUID based on counter
	return fmt.Sprintf("{%08X-%04X-%04X-%04X-%012X}", counter, counter, counter, counter, counter)
}