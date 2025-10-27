package restore

// Options holds restore configuration.
type Options struct {
	Sources        []string
	PackagesFolder string
	ConfigFile     string
	Force          bool
	NoCache        bool
	NoDependencies bool
	Verbosity      string
}
