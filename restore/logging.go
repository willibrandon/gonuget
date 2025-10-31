package restore

import (
	"strings"

	"github.com/fatih/color"
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// addLog adds a log message to the collector for cache file persistence.
// Matches MSBuildRestoreUtility.CollectMessage in NuGet.Client.
func (r *Restorer) addLog(log LogMessage) {
	r.logs = append(r.logs, log)
}

// addErrorLog creates and adds an error log from a NuGetError.
// Matches NuGet.Client's error logging in RestoreCommand.
func (r *Restorer) addErrorLog(err *NuGetError, targetFramework string) {
	log := LogMessage{
		Code:         err.Code,
		Level:        "Error",
		Message:      err.Message,
		ProjectPath:  err.ProjectPath,
		FilePath:     err.ProjectPath,
		LibraryID:    err.PackageID,
		TargetGraphs: []string{targetFramework},
	}
	r.addLog(log)
}

// replayLogs outputs cached logs to console (on cache hit).
// Matches MSBuildRestoreUtility.ReplayWarningsAndErrorsAsync in NuGet.Client.
func (r *Restorer) replayLogs(logs []LogMessage) {
	for _, log := range logs {
		level := strings.ToLower(log.Level)
		switch level {
		case "error":
			// Format: "    /path/to/project.csproj : error NU1101: message"
			// Use ANSI colors only if colors are enabled (TTY mode)
			if !color.NoColor {
				const (
					red   = "\033[1;31m"
					reset = "\033[0m"
				)
				r.console.Printf("    %s : %serror %s%s: %s\n",
					log.ProjectPath, red, log.Code, reset, log.Message)
			} else {
				r.console.Printf("    %s : error %s: %s\n",
					log.ProjectPath, log.Code, log.Message)
			}
		case "warning":
			// Format warnings similarly (yellow color in TTY mode)
			if !color.NoColor {
				const (
					yellow = "\033[1;33m"
					reset  = "\033[0m"
				)
				r.console.Printf("    %s : %swarning %s%s: %s\n",
					log.ProjectPath, yellow, log.Code, reset, log.Message)
			} else {
				r.console.Printf("    %s : warning %s: %s\n",
					log.ProjectPath, log.Code, log.Message)
			}
		}
	}
}

// writeCacheFileOnError writes a cache file when restore fails early.
// Matches NuGet.Client behavior of writing cache even on failure (with success=false).
func (r *Restorer) writeCacheFileOnError(proj *project.Project, dgSpecHash, cachePath string) {
	cacheFile := &CacheFile{
		Version:              CacheFileVersion,
		DgSpecHash:           dgSpecHash,
		Success:              false, // Restore failed
		ProjectFilePath:      proj.Path,
		ExpectedPackageFiles: []string{}, // No packages resolved
		Logs:                 r.logs,     // Collected error logs
	}

	// Don't fail if cache write fails (just log warning)
	if err := cacheFile.Save(cachePath); err != nil {
		r.console.Warning("Failed to write cache file: %v\n", err)
	}
}
