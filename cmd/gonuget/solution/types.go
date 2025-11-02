// Package solution provides parsers and utilities for working with .NET solution files
package solution

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Solution represents a parsed solution file (.sln, .slnx, or .slnf)
type Solution struct {
	// FilePath is the absolute path to the solution file
	FilePath string

	// FormatVersion is the solution file format version (e.g., "12.00" for VS 2013+)
	FormatVersion string

	// VisualStudioVersion is the Visual Studio version that created the file
	VisualStudioVersion string

	// MinimumVisualStudioVersion is the minimum VS version required
	MinimumVisualStudioVersion string

	// Projects contains all projects in the solution (excludes solution folders)
	Projects []Project

	// SolutionFolders contains virtual folders for organizing projects
	SolutionFolders []SolutionFolder

	// SolutionDir is the directory containing the solution file
	SolutionDir string
}

// Project represents a project reference in a solution
type Project struct {
	// Name is the display name of the project
	Name string

	// Path is the file system path to the project file (relative or absolute)
	Path string

	// GUID is the unique identifier for this project instance
	GUID string

	// TypeGUID identifies the project type (C#, VB.NET, F#, etc.)
	TypeGUID string

	// ParentFolderGUID is the GUID of the containing solution folder (if any)
	ParentFolderGUID string
}

// SolutionFolder represents a virtual folder in the solution
type SolutionFolder struct {
	// Name is the display name of the folder
	Name string

	// GUID is the unique identifier for this folder
	GUID string

	// ParentFolderGUID is the GUID of the parent folder (for nested folders)
	ParentFolderGUID string

	// Items contains file references in SolutionItems folders
	Items []string
}

// SolutionFilter represents a .slnf filter file
type SolutionFilter struct {
	// SolutionPath is the path to the parent .sln file
	SolutionPath string

	// Projects lists the project paths to include
	Projects []string
}

// ParseError represents an error during solution file parsing
type ParseError struct {
	// FilePath is the path to the file being parsed
	FilePath string

	// Line is the line number where the error occurred
	Line int

	// Column is the column number where the error occurred
	Column int

	// Message describes what went wrong
	Message string
}

// Error implements the error interface
func (e *ParseError) Error() string {
	if e.Line > 0 && e.Column > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", e.FilePath, e.Line, e.Column, e.Message)
	}
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", e.FilePath, e.Line, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.FilePath, e.Message)
}

// ProjectType GUIDs for common project types
const (
	// ProjectTypeCSProject identifies a C# project (classic)
	ProjectTypeCSProject = "{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}"

	// ProjectTypeCSProjectSDK identifies a SDK-style C# project (.NET Core/.NET 5+)
	ProjectTypeCSProjectSDK = "{9A19103F-16F7-4668-BE54-9A1E7A4F7556}"

	// ProjectTypeVBProject identifies a VB.NET project
	ProjectTypeVBProject = "{F184B08F-C81C-45F6-A57F-5ABD9991F28F}"

	// ProjectTypeFSProject identifies an F# project
	ProjectTypeFSProject = "{F2A71F9B-5D33-465A-A702-920D77279786}"

	// ProjectTypeSolutionFolder identifies a solution folder
	ProjectTypeSolutionFolder = "{2150E333-8FDC-42A3-9474-1A3956D46DE8}"

	// ProjectTypeSharedProject identifies a shared project
	ProjectTypeSharedProject = "{D954291E-2A0B-460D-934E-DC6B0785DB48}"

	// ProjectTypeWebSite identifies a website project
	ProjectTypeWebSite = "{E24C65DC-7377-472B-9ABA-BC803B73C61A}"
)

// IsNETProject returns true if this is a .NET project type
func (p *Project) IsNETProject() bool {
	upperGUID := strings.ToUpper(p.TypeGUID)
	return upperGUID == ProjectTypeCSProject ||
		upperGUID == ProjectTypeCSProjectSDK ||
		upperGUID == ProjectTypeVBProject ||
		upperGUID == ProjectTypeFSProject
}

// IsProjectFile returns true if the path looks like a project file
func (p *Project) IsProjectFile() bool {
	ext := strings.ToLower(filepath.Ext(p.Path))
	return ext == ".csproj" || ext == ".vbproj" || ext == ".fsproj"
}

// GetAbsolutePath returns the absolute path to the project file
func (p *Project) GetAbsolutePath(solutionDir string) string {
	if filepath.IsAbs(p.Path) {
		return p.Path
	}
	return filepath.Join(solutionDir, p.Path)
}

// GetProjects returns all project paths from the solution
func (s *Solution) GetProjects() []string {
	paths := make([]string, 0, len(s.Projects))
	for _, project := range s.Projects {
		if project.IsNETProject() {
			paths = append(paths, project.GetAbsolutePath(s.SolutionDir))
		}
	}
	return paths
}

// GetProjectByName finds a project by its name
func (s *Solution) GetProjectByName(name string) (*Project, bool) {
	lowerName := strings.ToLower(name)
	for i := range s.Projects {
		if strings.ToLower(s.Projects[i].Name) == lowerName {
			return &s.Projects[i], true
		}
	}
	return nil, false
}

// GetProjectByPath finds a project by its path
func (s *Solution) GetProjectByPath(path string) (*Project, bool) {
	searchPath := filepath.Clean(path)
	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(s.SolutionDir, searchPath)
	}

	for i := range s.Projects {
		projectPath := s.Projects[i].GetAbsolutePath(s.SolutionDir)
		if filepath.Clean(projectPath) == searchPath {
			return &s.Projects[i], true
		}
	}
	return nil, false
}