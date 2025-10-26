package main

import (
	"regexp"
	"strings"
)

// NormalizeOutput normalizes command output for comparison
func NormalizeOutput(output string) string {
	if output == "" {
		return ""
	}

	normalized := output

	// Normalize line endings
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")

	// Normalize path separators
	normalized = strings.ReplaceAll(normalized, "\\", "/")

	// Remove timestamps (e.g., "2025-01-25 14:30:45")
	re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}`)
	normalized = re.ReplaceAllString(normalized, "<TIMESTAMP>")

	// Remove version numbers from tool output
	re = regexp.MustCompile(`(NuGet|gonuget)\s+\d+\.\d+\.\d+(\.\d+)?`)
	normalized = re.ReplaceAllString(normalized, "$1 <VERSION>")

	// Normalize absolute paths to relative
	normalized = normalizePaths(normalized)

	// Trim trailing whitespace from each line
	lines := strings.Split(normalized, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	normalized = strings.Join(lines, "\n")

	return strings.TrimSpace(normalized)
}

// normalizePaths normalizes absolute paths to relative paths
func normalizePaths(output string) string {
	// Remove drive letters (Windows)
	re := regexp.MustCompile(`[A-Z]:[/\\]`)
	output = re.ReplaceAllString(output, "")

	// Replace home directory patterns
	re = regexp.MustCompile(`/Users/[^/]+/`)
	output = re.ReplaceAllString(output, "~/")

	re = regexp.MustCompile(`/home/[^/]+/`)
	output = re.ReplaceAllString(output, "~/")

	re = regexp.MustCompile(`C:\\Users\\[^\\]+\\`)
	output = re.ReplaceAllString(output, "~/")

	// Normalize /private/tmp to /tmp on macOS
	output = strings.ReplaceAll(output, "/private/tmp", "/tmp")

	return output
}
