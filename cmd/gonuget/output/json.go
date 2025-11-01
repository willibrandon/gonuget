package output

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// JSON output types matching the schema contract

// SourceListOutput represents the JSON output for source list command
type SourceListOutput struct {
	SchemaVersion string          `json:"schemaVersion"`
	ConfigFile    string          `json:"configFile"`
	Sources       []PackageSource `json:"sources"`
	ElapsedMs     int64           `json:"elapsedMs"`
}

// PackageSource represents a package source in JSON output
type PackageSource struct {
	Name            string `json:"name"`
	Source          string `json:"source"`
	Enabled         bool   `json:"enabled"`
	ProtocolVersion string `json:"protocolVersion,omitempty"`
}

// PackageListOutput represents the JSON output for package list command
type PackageListOutput struct {
	SchemaVersion string             `json:"schemaVersion"`
	Project       string             `json:"project"`
	Framework     string             `json:"framework"`
	Packages      []PackageReference `json:"packages"`
	Warnings      []string           `json:"warnings"`
	ElapsedMs     int64              `json:"elapsedMs"`
}

// PackageReference represents a package reference in JSON output
type PackageReference struct {
	ID              string `json:"id"`
	Version         string `json:"version"`
	Type            string `json:"type"` // "direct" or "transitive"
	ResolvedVersion string `json:"resolvedVersion"`
	Framework       string `json:"framework,omitempty"`
}

// PackageSearchOutput represents the JSON output for package search command
type PackageSearchOutput struct {
	SchemaVersion string         `json:"schemaVersion"`
	SearchTerm    string         `json:"searchTerm"`
	Sources       []string       `json:"sources"`
	Items         []SearchResult `json:"items"`
	Total         int            `json:"total"`
	ElapsedMs     int64          `json:"elapsedMs"`
}

// SearchResult represents a package search result in JSON output
type SearchResult struct {
	ID             string   `json:"id"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	Authors        string   `json:"authors"`
	TotalDownloads int64    `json:"totalDownloads,omitempty"`
	Verified       bool     `json:"verified,omitempty"`
	IconURL        string   `json:"iconUrl,omitempty"`
	ProjectURL     string   `json:"projectUrl,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

// WriteJSON writes a JSON object to the specified writer (typically stdout)
// When --format json is used, ALL JSON goes to stdout and ALL messages go to stderr
func WriteJSON(w io.Writer, v any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// MeasureElapsed returns elapsed time in milliseconds since start
func MeasureElapsed(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}

// CurrentSchemaVersion is the schema version for all JSON outputs
const CurrentSchemaVersion = "1.0.0"

// NewSourceListOutput creates a new SourceListOutput with schema version and start time
func NewSourceListOutput(configFile string, start time.Time) *SourceListOutput {
	return &SourceListOutput{
		SchemaVersion: CurrentSchemaVersion,
		ConfigFile:    configFile,
		Sources:       []PackageSource{},
		ElapsedMs:     MeasureElapsed(start),
	}
}

// NewPackageListOutput creates a new PackageListOutput with schema version
func NewPackageListOutput(project, framework string, start time.Time) *PackageListOutput {
	return &PackageListOutput{
		SchemaVersion: CurrentSchemaVersion,
		Project:       project,
		Framework:     framework,
		Packages:      []PackageReference{},
		Warnings:      []string{},
		ElapsedMs:     MeasureElapsed(start),
	}
}

// NewPackageSearchOutput creates a new PackageSearchOutput with schema version
func NewPackageSearchOutput(searchTerm string, sources []string, start time.Time) *PackageSearchOutput {
	return &PackageSearchOutput{
		SchemaVersion: CurrentSchemaVersion,
		SearchTerm:    searchTerm,
		Sources:       sources,
		Items:         []SearchResult{},
		Total:         0,
		ElapsedMs:     MeasureElapsed(start),
	}
}

// JSONOutputWriter writes JSON to stdout and messages to stderr
type JSONOutputWriter struct {
	stdout io.Writer
	stderr io.Writer
}

// NewJSONOutputWriter creates a new JSON output writer
func NewJSONOutputWriter(stdout, stderr io.Writer) *JSONOutputWriter {
	return &JSONOutputWriter{
		stdout: stdout,
		stderr: stderr,
	}
}

// WriteJSON writes JSON to stdout
func (w *JSONOutputWriter) WriteJSON(v any) error {
	return WriteJSON(w.stdout, v)
}

// WriteError writes an error message to stderr
func (w *JSONOutputWriter) WriteError(format string, args ...any) {
	_, _ = fmt.Fprintf(w.stderr, "Error: "+format+"\n", args...)
}

// WriteWarning writes a warning message to stderr
func (w *JSONOutputWriter) WriteWarning(format string, args ...any) {
	_, _ = fmt.Fprintf(w.stderr, "Warning: "+format+"\n", args...)
}
