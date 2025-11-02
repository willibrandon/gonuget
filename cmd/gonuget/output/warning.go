package output

import (
	"fmt"
	"io"
	"os"
)

// WarningWriter provides methods for writing warning messages to stderr
type WarningWriter struct {
	writer io.Writer
}

// NewWarningWriter creates a new warning writer that outputs to stderr
func NewWarningWriter() *WarningWriter {
	return &WarningWriter{
		writer: os.Stderr,
	}
}

// NewWarningWriterWithOutput creates a warning writer with a custom output
func NewWarningWriterWithOutput(w io.Writer) *WarningWriter {
	return &WarningWriter{
		writer: w,
	}
}

// Warning writes a warning message with the standard format
// Format: "Warning: <message>"
func (w *WarningWriter) Warning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(w.writer, "Warning: %s\n", message)
}

// Warn is an alias for Warning
func (w *WarningWriter) Warn(format string, args ...interface{}) {
	w.Warning(format, args...)
}

// WriteProjectWarning writes a warning for a specific project
func (w *WarningWriter) WriteProjectWarning(projectPath string, message string) {
	w.Warning("Project '%s': %s", projectPath, message)
}

// WriteMissingProjectWarning writes a warning for a missing project file
func (w *WarningWriter) WriteMissingProjectWarning(projectPath string) {
	w.Warning("Project file not found: %s", projectPath)
}

// WriteSolutionWarning writes a warning related to solution file operations
func (w *WarningWriter) WriteSolutionWarning(solutionPath string, message string) {
	w.Warning("Solution '%s': %s", solutionPath, message)
}

// Global warning writer instance for convenience
var globalWarningWriter = NewWarningWriter()

// Warning writes a warning message to stderr using the global writer
func Warning(format string, args ...interface{}) {
	globalWarningWriter.Warning(format, args...)
}

// ProjectWarning writes a project-specific warning using the global writer
func ProjectWarning(projectPath string, message string) {
	globalWarningWriter.WriteProjectWarning(projectPath, message)
}

// MissingProjectWarning writes a missing project warning using the global writer
func MissingProjectWarning(projectPath string) {
	globalWarningWriter.WriteMissingProjectWarning(projectPath)
}

// SolutionWarning writes a solution-specific warning using the global writer
func SolutionWarning(solutionPath string, message string) {
	globalWarningWriter.WriteSolutionWarning(solutionPath, message)
}