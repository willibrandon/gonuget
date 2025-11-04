package solution

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Parser defines the interface for parsing solution files
type Parser interface {
	// Parse reads and parses a solution file
	Parse(path string) (*Solution, error)

	// CanParse checks if this parser supports the given file
	CanParse(path string) bool
}

// GetParser returns the appropriate parser for a solution file
func GetParser(path string) (Parser, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".sln":
		return NewSlnParser(), nil
	case ".slnx":
		return NewSlnxParser(), nil
	case ".slnf":
		return NewSlnfParser(), nil
	default:
		return nil, fmt.Errorf("unsupported solution format: %s (supported: .sln, .slnx, .slnf)", ext)
	}
}

// ParseSolution is a convenience function that automatically selects the right parser
func ParseSolution(path string) (*Solution, error) {
	parser, err := GetParser(path)
	if err != nil {
		return nil, err
	}

	return parser.Parse(path)
}
