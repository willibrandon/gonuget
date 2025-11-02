package solution

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SlnParser parses text-based .sln files (MSBuild format)
type SlnParser struct{}

// NewSlnParser creates a new .sln file parser
func NewSlnParser() *SlnParser {
	return &SlnParser{}
}

// CanParse checks if this parser supports the given file
func (p *SlnParser) CanParse(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".sln"
}

// Parse reads and parses a .sln file
func (p *SlnParser) Parse(path string) (*Solution, error) {
	if !p.CanParse(path) {
		return nil, &ParseError{
			FilePath: path,
			Message:  "not a .sln file",
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

	sol := &Solution{
		FilePath:        absPath,
		SolutionDir:     filepath.Dir(absPath),
		Projects:        []Project{},
		SolutionFolders: []SolutionFolder{},
	}

	// Regular expressions for parsing
	formatVersionRegex := regexp.MustCompile(`^Microsoft Visual Studio Solution File, Format Version (\S+)`)
	vsVersionRegex := regexp.MustCompile(`^VisualStudioVersion = (\S+)`)
	minVSVersionRegex := regexp.MustCompile(`^MinimumVisualStudioVersion = (\S+)`)

	// Project line: Project("{GUID}") = "Name", "Path", "{GUID}"
	projectRegex := regexp.MustCompile(
		`(?i)^Project\("\{([A-F0-9-]+)\}"\)\s*=\s*"([^"]+)",\s*"([^"]+)",\s*"\{([A-F0-9-]+)\}"`,
	)

	// Nested projects in GlobalSection
	nestedProjectRegex := regexp.MustCompile(`(?i)^\s*\{([A-F0-9-]+)\}\s*=\s*\{([A-F0-9-]+)\}`)

	scanner := bufio.NewScanner(file)
	lineNum := 0
	inGlobalSection := false
	inNestedProjects := false
	currentProject := (*Project)(nil)
	currentFolder := (*SolutionFolder)(nil)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Parse format version
		if matches := formatVersionRegex.FindStringSubmatch(line); matches != nil {
			sol.FormatVersion = matches[1]
			continue
		}

		// Parse Visual Studio version
		if matches := vsVersionRegex.FindStringSubmatch(line); matches != nil {
			sol.VisualStudioVersion = matches[1]
			continue
		}

		// Parse minimum Visual Studio version
		if matches := minVSVersionRegex.FindStringSubmatch(line); matches != nil {
			sol.MinimumVisualStudioVersion = matches[1]
			continue
		}

		// Parse project definition
		if matches := projectRegex.FindStringSubmatch(line); matches != nil {
			typeGUID := "{" + strings.ToUpper(matches[1]) + "}"
			projectGUID := "{" + strings.ToUpper(matches[4]) + "}"
			projectName := matches[2]
			projectPath := matches[3]

			// Normalize the project path (convert backslashes to forward slashes)
			projectPath = NormalizePath(projectPath)

			// Check if it's a solution folder
			if typeGUID == ProjectTypeSolutionFolder {
				folder := SolutionFolder{
					Name:  projectName,
					GUID:  projectGUID,
					Items: []string{},
				}
				currentFolder = &folder
			} else {
				// It's a regular project
				project := Project{
					Name:     projectName,
					Path:     projectPath,
					GUID:     projectGUID,
					TypeGUID: typeGUID,
				}
				currentProject = &project
			}
			continue
		}

		// Handle EndProject
		if trimmedLine == "EndProject" {
			if currentProject != nil {
				sol.Projects = append(sol.Projects, *currentProject)
				currentProject = nil
			} else if currentFolder != nil {
				sol.SolutionFolders = append(sol.SolutionFolders, *currentFolder)
				currentFolder = nil
			}
			continue
		}

		// Handle ProjectSection for SolutionItems
		if currentFolder != nil && strings.Contains(line, "ProjectSection(SolutionItems)") {
			// Read solution items until EndProjectSection
			for scanner.Scan() {
				lineNum++
				itemLine := strings.TrimSpace(scanner.Text())
				if strings.Contains(itemLine, "EndProjectSection") {
					break
				}
				// Parse solution item (e.g., "README.md = README.md")
				parts := strings.Split(itemLine, "=")
				if len(parts) >= 1 {
					item := strings.TrimSpace(parts[0])
					if item != "" {
						currentFolder.Items = append(currentFolder.Items, item)
					}
				}
			}
			continue
		}

		// Handle Global sections
		if trimmedLine == "Global" {
			inGlobalSection = true
			continue
		}

		if trimmedLine == "EndGlobal" {
			inGlobalSection = false
			continue
		}

		// Handle nested projects section
		if inGlobalSection {
			if strings.Contains(line, "GlobalSection(NestedProjects)") {
				inNestedProjects = true
				continue
			}
			if strings.Contains(line, "EndGlobalSection") {
				inNestedProjects = false
				continue
			}

			// Parse nested project relationships
			if inNestedProjects {
				if matches := nestedProjectRegex.FindStringSubmatch(line); matches != nil {
					childGUID := "{" + strings.ToUpper(matches[1]) + "}"
					parentGUID := "{" + strings.ToUpper(matches[2]) + "}"

					// Find the child and set its parent
					for i := range sol.Projects {
						if sol.Projects[i].GUID == childGUID {
							sol.Projects[i].ParentFolderGUID = parentGUID
							break
						}
					}
					for i := range sol.SolutionFolders {
						if sol.SolutionFolders[i].GUID == childGUID {
							sol.SolutionFolders[i].ParentFolderGUID = parentGUID
							break
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, &ParseError{
			FilePath: path,
			Message:  fmt.Sprintf("error reading file: %v", err),
		}
	}

	// Check if we have an unclosed project
	if currentProject != nil || currentFolder != nil {
		return nil, &ParseError{
			FilePath: path,
			Line:     lineNum,
			Message:  "unexpected end of file: missing EndProject",
		}
	}

	return sol, nil
}