using System.Diagnostics;
using System.Text.Json;

namespace GonugetCliInterop.Tests.TestHelpers;

/// <summary>
/// Bridge for executing gonuget CLI commands via JSON-RPC protocol to compare behavior with dotnet nuget.
/// </summary>
public class GonugetCliBridge
{
    private readonly string _executablePath;

    /// <summary>
    /// Initializes a new instance of the GonugetCliBridge class.
    /// </summary>
    /// <exception cref="FileNotFoundException">Thrown when the gonuget-cli-interop-test executable cannot be found.</exception>
    public GonugetCliBridge()
    {
        _executablePath = FindExecutable();
    }

    /// <summary>
    /// Executes a command pair, running both dotnet nuget and gonuget with the same operation.
    /// </summary>
    /// <param name="dotnetCommand">The command to execute with dotnet nuget.</param>
    /// <param name="gonugetCommand">The command to execute with gonuget.</param>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="configFile">Optional path to a specific NuGet.config file.</param>
    /// <param name="timeout">Command timeout in seconds (default: 30).</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteCommandPairResponse ExecuteCommandPair(
        string dotnetCommand,
        string gonugetCommand,
        string workingDir,
        string? configFile = null,
        int timeout = 30)
    {
        var request = new
        {
            action = "execute_command_pair",
            data = new
            {
                dotnetCommand,
                gonugetCommand,
                workingDir,
                configFile,
                timeout
            }
        };

        return Execute<ExecuteCommandPairResponse>(request);
    }

    /// <summary>
    /// Executes the 'config get' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="key">The config key to retrieve.</param>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="showPath">Whether to show the config file path with the value.</param>
    /// <param name="workingDirFlag">Optional --working-directory flag value.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteConfigGetResponse ExecuteConfigGet(
        string key,
        string workingDir,
        bool showPath = false,
        string? workingDirFlag = null)
    {
        var request = new
        {
            action = "execute_config_get",
            data = new
            {
                key,
                workingDir,
                showPath,
                workingDirFlag
            }
        };

        return Execute<ExecuteConfigGetResponse>(request);
    }

    /// <summary>
    /// Executes the 'config set' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="key">The config key to set.</param>
    /// <param name="value">The value to set for the key.</param>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteConfigSetResponse ExecuteConfigSet(
        string key,
        string value,
        string workingDir)
    {
        var request = new
        {
            action = "execute_config_set",
            data = new
            {
                key,
                value,
                workingDir
            }
        };

        return Execute<ExecuteConfigSetResponse>(request);
    }

    /// <summary>
    /// Executes the 'config unset' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="key">The config key to remove.</param>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteConfigUnsetResponse ExecuteConfigUnset(
        string key,
        string workingDir)
    {
        var request = new
        {
            action = "execute_config_unset",
            data = new
            {
                key,
                workingDir
            }
        };

        return Execute<ExecuteConfigUnsetResponse>(request);
    }

    /// <summary>
    /// Executes the 'config paths' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="workingDirFlag">Optional --working-directory flag value.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteConfigPathsResponse ExecuteConfigPaths(
        string workingDir,
        string? workingDirFlag = null)
    {
        var request = new
        {
            action = "execute_config_paths",
            data = new
            {
                workingDir,
                workingDirFlag
            }
        };

        return Execute<ExecuteConfigPathsResponse>(request);
    }

    /// <summary>
    /// Executes the 'version' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteVersionResponse ExecuteVersion(string workingDir)
    {
        var request = new
        {
            action = "execute_version",
            data = new
            {
                workingDir
            }
        };

        return Execute<ExecuteVersionResponse>(request);
    }

    /// <summary>
    /// Executes the 'help' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="command">Optional command to get help for (empty for general help).</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteHelpResponse ExecuteHelp(string workingDir, string? command = null)
    {
        var request = new
        {
            action = "execute_help",
            data = new
            {
                workingDir,
                command
            }
        };

        return Execute<ExecuteHelpResponse>(request);
    }

    /// <summary>
    /// Executes the 'list source' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="configFile">Optional path to a specific NuGet.config file.</param>
    /// <param name="format">The format of the list command output: Detailed (default) or Short.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteSourceListResponse ExecuteSourceList(
        string workingDir,
        string? configFile = null,
        string? format = null)
    {
        var request = new
        {
            action = "execute_source_list",
            data = new
            {
                workingDir,
                configFile,
                format
            }
        };

        return Execute<ExecuteSourceListResponse>(request);
    }

    /// <summary>
    /// Executes the 'add source' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="name">Name of the source.</param>
    /// <param name="source">Path to the package source.</param>
    /// <param name="configFile">Optional path to a specific NuGet.config file.</param>
    /// <param name="username">Optional username for authenticated source.</param>
    /// <param name="password">Optional password for authenticated source.</param>
    /// <param name="storePasswordInClearText">Whether to store password in clear text.</param>
    /// <param name="validAuthenticationTypes">Comma-separated list of valid authentication types.</param>
    /// <param name="protocolVersion">The NuGet server protocol version (2 or 3).</param>
    /// <param name="allowInsecureConnections">Whether to allow HTTP connections.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteSourceAddResponse ExecuteSourceAdd(
        string workingDir,
        string name,
        string source,
        string? configFile = null,
        string? username = null,
        string? password = null,
        bool storePasswordInClearText = false,
        string? validAuthenticationTypes = null,
        string? protocolVersion = null,
        bool allowInsecureConnections = false)
    {
        var request = new
        {
            action = "execute_source_add",
            data = new
            {
                workingDir,
                configFile,
                name,
                source,
                username,
                password,
                storePasswordInClearText,
                validAuthenticationTypes,
                protocolVersion,
                allowInsecureConnections
            }
        };

        return Execute<ExecuteSourceAddResponse>(request);
    }

    /// <summary>
    /// Executes the 'remove source' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="name">Name of the source to remove.</param>
    /// <param name="configFile">Optional path to a specific NuGet.config file.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteSourceRemoveResponse ExecuteSourceRemove(
        string workingDir,
        string name,
        string? configFile = null)
    {
        var request = new
        {
            action = "execute_source_remove",
            data = new
            {
                workingDir,
                configFile,
                name
            }
        };

        return Execute<ExecuteSourceRemoveResponse>(request);
    }

    /// <summary>
    /// Executes the 'enable source' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="name">Name of the source to enable.</param>
    /// <param name="configFile">Optional path to a specific NuGet.config file.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteSourceEnableResponse ExecuteSourceEnable(
        string workingDir,
        string name,
        string? configFile = null)
    {
        var request = new
        {
            action = "execute_source_enable",
            data = new
            {
                workingDir,
                configFile,
                name
            }
        };

        return Execute<ExecuteSourceEnableResponse>(request);
    }

    /// <summary>
    /// Executes the 'disable source' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="name">Name of the source to disable.</param>
    /// <param name="configFile">Optional path to a specific NuGet.config file.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteSourceDisableResponse ExecuteSourceDisable(
        string workingDir,
        string name,
        string? configFile = null)
    {
        var request = new
        {
            action = "execute_source_disable",
            data = new
            {
                workingDir,
                configFile,
                name
            }
        };

        return Execute<ExecuteSourceDisableResponse>(request);
    }

    /// <summary>
    /// Executes the 'update source' command on both dotnet nuget and gonuget.
    /// </summary>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="name">Name of the source to update.</param>
    /// <param name="source">Optional new path to the package source.</param>
    /// <param name="configFile">Optional path to a specific NuGet.config file.</param>
    /// <param name="username">Optional username for authenticated source.</param>
    /// <param name="password">Optional password for authenticated source.</param>
    /// <param name="storePasswordInClearText">Whether to store password in clear text.</param>
    /// <param name="validAuthenticationTypes">Comma-separated list of valid authentication types.</param>
    /// <param name="protocolVersion">The NuGet server protocol version (2 or 3).</param>
    /// <param name="allowInsecureConnections">Whether to allow HTTP connections.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteSourceUpdateResponse ExecuteSourceUpdate(
        string workingDir,
        string name,
        string? source = null,
        string? configFile = null,
        string? username = null,
        string? password = null,
        bool storePasswordInClearText = false,
        string? validAuthenticationTypes = null,
        string? protocolVersion = null,
        bool allowInsecureConnections = false)
    {
        var request = new
        {
            action = "execute_source_update",
            data = new
            {
                workingDir,
                configFile,
                name,
                source,
                username,
                password,
                storePasswordInClearText,
                validAuthenticationTypes,
                protocolVersion,
                allowInsecureConnections
            }
        };

        return Execute<ExecuteSourceUpdateResponse>(request);
    }

    /// <summary>
    /// Executes the 'add package' command on both dotnet add and gonuget add.
    /// </summary>
    /// <param name="projectPath">Path to the project file.</param>
    /// <param name="packageId">Package ID to add.</param>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="version">Optional version to install.</param>
    /// <param name="source">Optional package source URL.</param>
    /// <param name="noRestore">Whether to skip restoring packages.</param>
    /// <param name="prerelease">Whether to allow prerelease versions.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteAddPackageResponse ExecuteAddPackage(
        string projectPath,
        string packageId,
        string workingDir,
        string? version = null,
        string? source = null,
        bool noRestore = false,
        bool prerelease = false)
    {
        var request = new
        {
            action = "execute_add_package",
            data = new
            {
                projectPath,
                packageId,
                workingDir,
                version,
                source,
                noRestore,
                prerelease
            }
        };

        return Execute<ExecuteAddPackageResponse>(request);
    }

    /// <summary>
    /// Executes the 'restore' command on both dotnet restore and gonuget restore.
    /// </summary>
    /// <param name="projectPath">Path to the project file.</param>
    /// <param name="workingDir">The working directory for command execution.</param>
    /// <param name="source">Optional package source URL.</param>
    /// <param name="packages">Optional custom global packages folder.</param>
    /// <param name="force">Force re-download even if packages exist.</param>
    /// <param name="noCache">Don't use HTTP cache.</param>
    /// <returns>Response containing exit codes and output from both commands.</returns>
    public ExecuteRestoreResponse ExecuteRestore(
        string projectPath,
        string workingDir,
        string? source = null,
        string? packages = null,
        bool force = false,
        bool noCache = false)
    {
        var request = new
        {
            action = "execute_restore",
            data = new
            {
                projectPath,
                workingDir,
                source,
                packages,
                force,
                noCache
            }
        };

        return Execute<ExecuteRestoreResponse>(request);
    }

    /// <summary>
    /// Executes a JSON-RPC request by piping it to the gonuget-cli-interop-test process.
    /// </summary>
    /// <typeparam name="T">The expected response data type.</typeparam>
    /// <param name="request">The request object to serialize and send.</param>
    /// <returns>The deserialized response data.</returns>
    /// <exception cref="TimeoutException">Thrown when the bridge process times out.</exception>
    /// <exception cref="GonugetCliException">Thrown when the bridge returns an error response.</exception>
    private T Execute<T>(object request)
    {
        var requestJson = JsonSerializer.Serialize(request);

        var startInfo = new ProcessStartInfo
        {
            FileName = _executablePath,
            RedirectStandardInput = true,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
            CreateNoWindow = true
        };

        using var process = new Process { StartInfo = startInfo };
        process.Start();

        // Write request to stdin
        process.StandardInput.WriteLine(requestJson);
        process.StandardInput.Close();

        // Read response from stdout
        var responseJson = process.StandardOutput.ReadToEnd();
        var stderr = process.StandardError.ReadToEnd();

        if (!process.WaitForExit(30000))
        {
            process.Kill();
            throw new TimeoutException("Bridge process timed out");
        }

        // Parse response
        var response = JsonSerializer.Deserialize<BridgeResponse<T>>(responseJson);

        if (response == null)
        {
            throw new Exception($"Failed to parse bridge response. StdOut: {responseJson}\nStdErr: {stderr}");
        }

        if (!response.Success)
        {
            var errorDetails = response.Error?.Details ?? $"StdOut: {responseJson}\nStdErr: {stderr}";
            var errorMessage = (response.Error?.Message ?? "Unknown error") + "\n" + errorDetails;
            throw new GonugetCliException(
                response.Error?.Code ?? "UNKNOWN",
                errorMessage,
                errorDetails);
        }

        return response.Data!;
    }

    /// <summary>
    /// Locates the gonuget-cli-interop-test executable in the test output directory or repository root.
    /// </summary>
    /// <returns>The full path to the executable.</returns>
    /// <exception cref="FileNotFoundException">Thrown when the executable cannot be found.</exception>
    private static string FindExecutable()
    {
        const string executableName = "gonuget-cli-interop-test";

        // Check test output directory first
        var testDir = AppContext.BaseDirectory;
        var exePath = FindExecutableInDirectory(testDir, executableName);
        if (exePath != null)
            return exePath;

        // Check relative to repository root (for local development)
        // From bin/Debug/net9.0/ we need 6 levels up to reach repo root
        var repoRoot = Path.GetFullPath(Path.Combine(testDir, "../../../../../../"));
        exePath = FindExecutableInDirectory(repoRoot, executableName);
        if (exePath != null)
            return exePath;

        throw new FileNotFoundException(
            "gonuget-cli-interop-test executable not found. " +
            "Run 'make build-cli-interop' or 'go build -o gonuget-cli-interop-test ./cmd/gonuget-cli-interop-test' before running tests.");
    }

    /// <summary>
    /// Searches for an executable in a directory, checking both with and without .exe extension.
    /// </summary>
    /// <param name="directory">The directory to search in.</param>
    /// <param name="executableName">The base name of the executable (without extension).</param>
    /// <returns>The full path if found, otherwise null.</returns>
    private static string? FindExecutableInDirectory(string directory, string executableName)
    {
        // Check without extension (Linux/macOS)
        var exePath = Path.Combine(directory, executableName);
        if (File.Exists(exePath))
            return exePath;

        // Check with .exe extension (Windows)
        exePath = Path.Combine(directory, executableName + ".exe");
        if (File.Exists(exePath))
            return exePath;

        return null;
    }
}
