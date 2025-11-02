package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Verb-first patterns that should be detected and rejected
var verbFirstPatterns = map[string]string{
	// Package namespace
	"add package":    "gonuget package add",
	"list package":   "gonuget package list",
	"remove package": "gonuget package remove",
	"search package": "gonuget package search",

	// Source namespace
	"add source":    "gonuget source add",
	"list source":   "gonuget source list",
	"remove source": "gonuget source remove",

	// Top-level verbs that imply source (backward compatibility detection)
	"enable":  "gonuget source enable",
	"disable": "gonuget source disable",
	"update":  "gonuget source update",
}

// SetupCustomErrorHandler configures verb-first pattern detection
func SetupCustomErrorHandler(rootCmd *cobra.Command) {
	rootCmd.SilenceErrors = true // Prevent Cobra's default error output

	// Set custom error handler for FlagErrorFunc
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if err == nil {
			return nil
		}

		// Check if this looks like a verb-first pattern
		if suggestion := detectVerbFirstPattern(cmd); suggestion != "" {
			return fmt.Errorf("the verb-first form is not supported. Try: %s", suggestion)
		}

		// Default error handling
		return err
	})
}

// detectVerbFirstPattern checks if command looks like verb-first and suggests alternative
func detectVerbFirstPattern(cmd *cobra.Command) string {
	// Build command path (e.g., "add package")
	parts := []string{}
	for c := cmd; c != nil && c.Parent() != nil; c = c.Parent() {
		parts = append([]string{c.Name()}, parts...)
	}
	commandPath := strings.Join(parts, " ")

	// Check against known patterns (case-insensitive)
	commandPathLower := strings.ToLower(commandPath)
	for pattern, suggestion := range verbFirstPatterns {
		if strings.Contains(commandPathLower, pattern) {
			return suggestion
		}
	}

	// Check if it's a top-level verb that should be under source
	if len(parts) > 0 {
		firstArg := strings.ToLower(parts[0])
		if suggestion, found := verbFirstPatterns[firstArg]; found {
			return suggestion
		}
	}

	return ""
}

// HandleUnknownCommand provides suggestions for unknown commands
func HandleUnknownCommand(cmd *cobra.Command, args []string) error {
	// Check for verb-first patterns first
	if suggestion := detectVerbFirstPattern(cmd); suggestion != "" {
		return fmt.Errorf("the verb-first form is not supported. Try: %s", suggestion)
	}

	// Default unknown command handling (Cobra will provide suggestions via Levenshtein distance)
	return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
}

// Solution file error messages matching dotnet CLI exactly
const (
	// ErrNoProjectForAdd is the error message when trying to add a package to a solution file
	ErrNoProjectForAdd = "Couldn't find a project to run. Ensure a project exists in %s, or pass the path to the project using --project"

	// ErrInvalidProjectFile is the error message when trying to remove a package from a solution file
	ErrInvalidProjectFile = "Missing or invalid project file: %s"

	// ErrSolutionNotSupported is a generic error for unsupported solution operations
	ErrSolutionNotSupported = "Operation not supported for solution files"
)

// SolutionNotSupportedError represents an error when a solution file is provided for add operation
type SolutionNotSupportedError struct {
	Directory string
}

// Error implements the error interface
func (e *SolutionNotSupportedError) Error() string {
	return fmt.Sprintf(ErrNoProjectForAdd, e.Directory)
}

// InvalidProjectFileError represents an error when a solution file is provided for remove operation
type InvalidProjectFileError struct {
	Path string
}

// Error implements the error interface
func (e *InvalidProjectFileError) Error() string {
	return fmt.Sprintf(ErrInvalidProjectFile, e.Path)
}
