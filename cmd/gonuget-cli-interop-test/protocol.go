package main

// ExecuteCommandPairRequest executes both dotnet nuget and gonuget commands
type ExecuteCommandPairRequest struct {
	DotnetCommand  string `json:"dotnetCommand"`  // e.g., "config get all"
	GonugetCommand string `json:"gonugetCommand"` // e.g., "config get all"
	WorkingDir     string `json:"workingDir"`
	Timeout        int    `json:"timeout,omitempty"` // seconds, default 30
}

// ExecuteCommandPairResponse contains execution results from both commands
type ExecuteCommandPairResponse struct {
	// dotnet nuget results
	DotnetExitCode int    `json:"dotnetExitCode"`
	DotnetStdOut   string `json:"dotnetStdOut"`
	DotnetStdErr   string `json:"dotnetStdErr"`
	DotnetSuccess  bool   `json:"dotnetSuccess"`

	// gonuget results
	GonugetExitCode int    `json:"gonugetExitCode"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	GonugetSuccess  bool   `json:"gonugetSuccess"`

	// Normalized comparison
	NormalizedDotnetStdOut  string `json:"normalizedDotnetStdOut"`
	NormalizedGonugetStdOut string `json:"normalizedGonugetStdOut"`
	OutputMatches           bool   `json:"outputMatches"`
}

// ExecuteConfigGetRequest for config get command
type ExecuteConfigGetRequest struct {
	Key            string `json:"key"`
	WorkingDir     string `json:"workingDir"`
	ShowPath       bool   `json:"showPath,omitempty"`
	WorkingDirFlag string `json:"workingDirFlag,omitempty"` // For --working-directory flag
}

// ExecuteConfigGetResponse contains config get results
type ExecuteConfigGetResponse struct {
	DotnetExitCode  int    `json:"dotnetExitCode"`
	GonugetExitCode int    `json:"gonugetExitCode"`
	DotnetStdOut    string `json:"dotnetStdOut"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	DotnetStdErr    string `json:"dotnetStdErr"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	OutputMatches   bool   `json:"outputMatches"`
}

// ExecuteConfigSetRequest for config set command
type ExecuteConfigSetRequest struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	WorkingDir string `json:"workingDir"`
}

// ExecuteConfigSetResponse contains config set results
type ExecuteConfigSetResponse struct {
	DotnetExitCode  int    `json:"dotnetExitCode"`
	GonugetExitCode int    `json:"gonugetExitCode"`
	DotnetStdOut    string `json:"dotnetStdOut"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	DotnetStdErr    string `json:"dotnetStdErr"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	OutputMatches   bool   `json:"outputMatches"`
}

// ExecuteConfigUnsetRequest for config unset command
type ExecuteConfigUnsetRequest struct {
	Key        string `json:"key"`
	WorkingDir string `json:"workingDir"`
}

// ExecuteConfigUnsetResponse contains config unset results
type ExecuteConfigUnsetResponse struct {
	DotnetExitCode  int    `json:"dotnetExitCode"`
	GonugetExitCode int    `json:"gonugetExitCode"`
	DotnetStdOut    string `json:"dotnetStdOut"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	DotnetStdErr    string `json:"dotnetStdErr"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	OutputMatches   bool   `json:"outputMatches"`
}

// ExecuteConfigPathsRequest for config paths command
type ExecuteConfigPathsRequest struct {
	WorkingDir     string `json:"workingDir"`
	WorkingDirFlag string `json:"workingDirFlag,omitempty"` // For --working-directory flag
}

// ExecuteConfigPathsResponse contains config paths results
type ExecuteConfigPathsResponse struct {
	DotnetExitCode  int    `json:"dotnetExitCode"`
	GonugetExitCode int    `json:"gonugetExitCode"`
	DotnetStdOut    string `json:"dotnetStdOut"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	DotnetStdErr    string `json:"dotnetStdErr"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	OutputMatches   bool   `json:"outputMatches"`
}

// ExecuteVersionRequest for version command
type ExecuteVersionRequest struct {
	WorkingDir string `json:"workingDir"`
}

// ExecuteVersionResponse contains version command results
type ExecuteVersionResponse struct {
	DotnetExitCode         int    `json:"dotnetExitCode"`
	GonugetExitCode        int    `json:"gonugetExitCode"`
	DotnetStdOut           string `json:"dotnetStdOut"`
	GonugetStdOut          string `json:"gonugetStdOut"`
	DotnetStdErr           string `json:"dotnetStdErr"`
	GonugetStdErr          string `json:"gonugetStdErr"`
	ExitCodesMatch         bool   `json:"exitCodesMatch"`
	OutputFormatSimilar    bool   `json:"outputFormatSimilar"` // Format may differ but both show version
}

// ExecuteSourceListRequest for list source command
type ExecuteSourceListRequest struct {
	WorkingDir string `json:"workingDir"`
	ConfigFile string `json:"configFile,omitempty"`
	Format     string `json:"format,omitempty"` // "Detailed" or "Short"
}

// ExecuteSourceListResponse contains list source results
type ExecuteSourceListResponse struct {
	DotnetExitCode  int    `json:"dotnetExitCode"`
	GonugetExitCode int    `json:"gonugetExitCode"`
	DotnetStdOut    string `json:"dotnetStdOut"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	DotnetStdErr    string `json:"dotnetStdErr"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	OutputMatches   bool   `json:"outputMatches"`
}

// ExecuteSourceAddRequest for add source command
type ExecuteSourceAddRequest struct {
	WorkingDir               string `json:"workingDir"`
	ConfigFile               string `json:"configFile,omitempty"`
	Name                     string `json:"name"`
	Source                   string `json:"source"`
	Username                 string `json:"username,omitempty"`
	Password                 string `json:"password,omitempty"`
	StorePasswordInClearText bool   `json:"storePasswordInClearText,omitempty"`
	ValidAuthenticationTypes string `json:"validAuthenticationTypes,omitempty"`
	ProtocolVersion          string `json:"protocolVersion,omitempty"`
	AllowInsecureConnections bool   `json:"allowInsecureConnections,omitempty"`
}

// ExecuteSourceAddResponse contains add source results
type ExecuteSourceAddResponse struct {
	DotnetExitCode  int    `json:"dotnetExitCode"`
	GonugetExitCode int    `json:"gonugetExitCode"`
	DotnetStdOut    string `json:"dotnetStdOut"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	DotnetStdErr    string `json:"dotnetStdErr"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	OutputMatches   bool   `json:"outputMatches"`
}

// ExecuteSourceRemoveRequest for remove source command
type ExecuteSourceRemoveRequest struct {
	WorkingDir string `json:"workingDir"`
	ConfigFile string `json:"configFile,omitempty"`
	Name       string `json:"name"`
}

// ExecuteSourceRemoveResponse contains remove source results
type ExecuteSourceRemoveResponse struct {
	DotnetExitCode  int    `json:"dotnetExitCode"`
	GonugetExitCode int    `json:"gonugetExitCode"`
	DotnetStdOut    string `json:"dotnetStdOut"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	DotnetStdErr    string `json:"dotnetStdErr"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	OutputMatches   bool   `json:"outputMatches"`
}

// ExecuteSourceEnableRequest for enable source command
type ExecuteSourceEnableRequest struct {
	WorkingDir string `json:"workingDir"`
	ConfigFile string `json:"configFile,omitempty"`
	Name       string `json:"name"`
}

// ExecuteSourceEnableResponse contains enable source results
type ExecuteSourceEnableResponse struct {
	DotnetExitCode  int    `json:"dotnetExitCode"`
	GonugetExitCode int    `json:"gonugetExitCode"`
	DotnetStdOut    string `json:"dotnetStdOut"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	DotnetStdErr    string `json:"dotnetStdErr"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	OutputMatches   bool   `json:"outputMatches"`
}

// ExecuteSourceDisableRequest for disable source command
type ExecuteSourceDisableRequest struct {
	WorkingDir string `json:"workingDir"`
	ConfigFile string `json:"configFile,omitempty"`
	Name       string `json:"name"`
}

// ExecuteSourceDisableResponse contains disable source results
type ExecuteSourceDisableResponse struct {
	DotnetExitCode  int    `json:"dotnetExitCode"`
	GonugetExitCode int    `json:"gonugetExitCode"`
	DotnetStdOut    string `json:"dotnetStdOut"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	DotnetStdErr    string `json:"dotnetStdErr"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	OutputMatches   bool   `json:"outputMatches"`
}

// ExecuteSourceUpdateRequest for update source command
type ExecuteSourceUpdateRequest struct {
	WorkingDir               string `json:"workingDir"`
	ConfigFile               string `json:"configFile,omitempty"`
	Name                     string `json:"name"`
	Source                   string `json:"source,omitempty"`
	Username                 string `json:"username,omitempty"`
	Password                 string `json:"password,omitempty"`
	StorePasswordInClearText bool   `json:"storePasswordInClearText,omitempty"`
	ValidAuthenticationTypes string `json:"validAuthenticationTypes,omitempty"`
	ProtocolVersion          string `json:"protocolVersion,omitempty"`
	AllowInsecureConnections bool   `json:"allowInsecureConnections,omitempty"`
}

// ExecuteSourceUpdateResponse contains update source results
type ExecuteSourceUpdateResponse struct {
	DotnetExitCode  int    `json:"dotnetExitCode"`
	GonugetExitCode int    `json:"gonugetExitCode"`
	DotnetStdOut    string `json:"dotnetStdOut"`
	GonugetStdOut   string `json:"gonugetStdOut"`
	DotnetStdErr    string `json:"dotnetStdErr"`
	GonugetStdErr   string `json:"gonugetStdErr"`
	OutputMatches   bool   `json:"outputMatches"`
}
