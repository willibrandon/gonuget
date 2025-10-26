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
