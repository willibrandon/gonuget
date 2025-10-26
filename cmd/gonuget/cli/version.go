// cmd/gonuget/cli/version.go
package cli

// Version information (set by main)
var (
	Version = "0.0.0-dev"
	Commit  = "unknown"
	Date    = "unknown"
	BuiltBy = "unknown"
)

// GetVersion returns formatted version information
func GetVersion() string {
	return Version
}

// GetFullVersion returns detailed version information
func GetFullVersion() string {
	return "gonuget version " + Version + "\n" +
		"commit: " + Commit + "\n" +
		"built: " + Date + "\n" +
		"built by: " + BuiltBy
}
