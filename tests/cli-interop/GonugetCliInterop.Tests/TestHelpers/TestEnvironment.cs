namespace GonugetCliInterop.Tests.TestHelpers;

/// <summary>
/// Provides a disposable test environment for CLI interop tests with temporary directories and config files.
/// </summary>
public class TestEnvironment : IDisposable
{
    /// <summary>
    /// Gets the temporary directory path created for this test environment.
    /// </summary>
    public string TestDirectory { get; private set; }

    /// <summary>
    /// Gets the path to the NuGet.config file within the test directory.
    /// </summary>
    public string ConfigFilePath { get; private set; }

    /// <summary>
    /// Initializes a new instance of the TestEnvironment class.
    /// Creates a unique temporary directory for the test.
    /// </summary>
    public TestEnvironment()
    {
        // Create unique test directory
        TestDirectory = Path.Combine(Path.GetTempPath(), $"gonuget-cli-test-{Guid.NewGuid():N}");
        Directory.CreateDirectory(TestDirectory);

        // Set default config file path
        ConfigFilePath = Path.Combine(TestDirectory, "NuGet.config");
    }

    /// <summary>
    /// Creates a test NuGet.config file with optional custom content.
    /// </summary>
    /// <param name="content">Optional custom XML content. If null, uses default config with nuget.org source.</param>
    public void CreateTestConfig(string? content = null)
    {
        var defaultContent = @"<?xml version=""1.0"" encoding=""utf-8""?>
<configuration>
  <packageSources>
    <add key=""nuget.org"" value=""https://api.nuget.org/v3/index.json"" protocolVersion=""3"" />
  </packageSources>
</configuration>";

        File.WriteAllText(ConfigFilePath, content ?? defaultContent);
    }

    /// <summary>
    /// Creates a test NuGet.config file with specific config section key-value pairs.
    /// </summary>
    /// <param name="configValues">Dictionary of config keys and values to add to the config section.</param>
    public void CreateTestConfigWithValues(Dictionary<string, string> configValues)
    {
        var config = @"<?xml version=""1.0"" encoding=""utf-8""?>
<configuration>
  <packageSources>
    <add key=""nuget.org"" value=""https://api.nuget.org/v3/index.json"" protocolVersion=""3"" />
  </packageSources>
  <config>";

        foreach (var kvp in configValues)
        {
            config += $"\n    <add key=\"{kvp.Key}\" value=\"{kvp.Value}\" />";
        }

        config += @"
  </config>
</configuration>";

        File.WriteAllText(ConfigFilePath, config);
    }

    /// <summary>
    /// Reads the current config file content.
    /// </summary>
    /// <returns>The full text content of the config file.</returns>
    /// <exception cref="FileNotFoundException">Thrown when the config file does not exist.</exception>
    public string ReadConfigFile()
    {
        if (!File.Exists(ConfigFilePath))
            throw new FileNotFoundException($"Config file not found: {ConfigFilePath}");

        return File.ReadAllText(ConfigFilePath);
    }

    /// <summary>
    /// Checks if the config file contains a specific key-value pair.
    /// </summary>
    /// <param name="key">The config key to search for.</param>
    /// <param name="value">The config value to search for.</param>
    /// <returns>True if both the key and value are found in the config file; otherwise false.</returns>
    public bool ConfigContains(string key, string value)
    {
        if (!File.Exists(ConfigFilePath))
            return false;

        var content = File.ReadAllText(ConfigFilePath);
        return content.Contains($"key=\"{key}\"") && content.Contains($"value=\"{value}\"");
    }

    /// <summary>
    /// Checks if the config file contains a specific key.
    /// </summary>
    /// <param name="key">The config key to search for.</param>
    /// <returns>True if the key is found in the config file; otherwise false.</returns>
    public bool ConfigContainsKey(string key)
    {
        if (!File.Exists(ConfigFilePath))
            return false;

        var content = File.ReadAllText(ConfigFilePath);
        return content.Contains($"key=\"{key}\"");
    }

    /// <summary>
    /// Disposes the test environment by deleting the temporary test directory and all its contents.
    /// </summary>
    public void Dispose()
    {
        try
        {
            if (Directory.Exists(TestDirectory))
            {
                Directory.Delete(TestDirectory, recursive: true);
            }
        }
        catch
        {
            // Ignore cleanup errors
        }

        GC.SuppressFinalize(this);
    }
}
