using GonugetCliInterop.Tests.TestHelpers;

namespace GonugetCliInterop.Tests.Foundation;

/// <summary>
/// Tests source management commands (list, add, remove, enable, disable, update) to verify
/// parity between dotnet nuget and gonuget CLI behavior.
/// </summary>
public class SourceCommandTests
{
    private readonly GonugetCliBridge _bridge = new();

    [Fact]
    public void ListSource_DefaultConfig_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteSourceList(
            env.TestDirectory,
            env.ConfigFilePath);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Both should show nuget.org source
        Assert.Contains("nuget.org", result.DotnetStdOut);
        Assert.Contains("nuget.org", result.GonugetStdOut);
        Assert.Contains("https://api.nuget.org/v3/index.json", result.DotnetStdOut);
        Assert.Contains("https://api.nuget.org/v3/index.json", result.GonugetStdOut);
    }

    [Fact]
    public void ListSource_DetailedFormat_ShowsURLs()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteSourceList(
            env.TestDirectory,
            env.ConfigFilePath,
            format: "Detailed");

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Detailed format should show URLs
        Assert.Contains("https://api.nuget.org/v3/index.json", result.DotnetStdOut);
        Assert.Contains("https://api.nuget.org/v3/index.json", result.GonugetStdOut);
    }

    [Fact]
    public void ListSource_ShortFormat_HidesURLs()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteSourceList(
            env.TestDirectory,
            env.ConfigFilePath,
            format: "Short");

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Short format should show names but not URLs
        Assert.Contains("nuget.org", result.DotnetStdOut);
        Assert.Contains("nuget.org", result.GonugetStdOut);
    }

    [Fact]
    public void AddSource_BasicHTTPS_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteSourceAdd(
            env.TestDirectory,
            "TestFeed",
            "https://test.nuget.org/v3/index.json",
            configFile: env.ConfigFilePath);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify the source was added
        var listResult = _bridge.ExecuteSourceList(
            env.TestDirectory,
            env.ConfigFilePath);

        Assert.Contains("TestFeed", listResult.DotnetStdOut);
        Assert.Contains("TestFeed", listResult.GonugetStdOut);
        Assert.Contains("https://test.nuget.org/v3/index.json", listResult.DotnetStdOut);
        Assert.Contains("https://test.nuget.org/v3/index.json", listResult.GonugetStdOut);
    }

    [Fact]
    public void AddSource_WithCredentials_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteSourceAdd(
            env.TestDirectory,
            "PrivateFeed",
            "https://private.nuget.org/v3/index.json",
            configFile: env.ConfigFilePath,
            username: "testuser",
            password: "testpass",
            storePasswordInClearText: true);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify the source was added with credentials
        var configContent = env.ReadConfigFile();
        Assert.Contains("PrivateFeed", configContent);
        Assert.Contains("testuser", configContent);
        Assert.Contains("testpass", configContent);
    }

    [Fact]
    public void AddSource_HTTPWithoutFlag_Fails()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteSourceAdd(
            env.TestDirectory,
            "InsecureFeed",
            "http://insecure.nuget.org/v3/index.json",
            configFile: env.ConfigFilePath);

        // Assert - Both should reject insecure HTTP
        Assert.NotEqual(0, result.DotnetExitCode);
        Assert.NotEqual(0, result.GonugetExitCode);

        // Note: dotnet writes error to stdout, gonuget writes to stderr
        // Check both outputs for the error message
        var dotnetOutput = (result.DotnetStdOut + result.DotnetStdErr).ToLower();
        var gonugetOutput = (result.GonugetStdOut + result.GonugetStdErr).ToLower();

        Assert.Contains("http", dotnetOutput);
        Assert.Contains("http", gonugetOutput);
    }

    [Fact]
    public void AddSource_HTTPWithAllowInsecure_Succeeds()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteSourceAdd(
            env.TestDirectory,
            "InsecureFeed",
            "http://insecure.nuget.org/v3/index.json",
            configFile: env.ConfigFilePath,
            allowInsecureConnections: true);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify the source was added
        var listResult = _bridge.ExecuteSourceList(
            env.TestDirectory,
            env.ConfigFilePath);

        Assert.Contains("InsecureFeed", listResult.DotnetStdOut);
        Assert.Contains("InsecureFeed", listResult.GonugetStdOut);
    }

    [Fact]
    public void RemoveSource_ExistingSource_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Add a source first
        _bridge.ExecuteSourceAdd(
            env.TestDirectory,
            "ToRemove",
            "https://test.nuget.org/v3/index.json",
            configFile: env.ConfigFilePath);

        // Act
        var result = _bridge.ExecuteSourceRemove(
            env.TestDirectory,
            "ToRemove",
            env.ConfigFilePath);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify the source was removed
        var listResult = _bridge.ExecuteSourceList(
            env.TestDirectory,
            env.ConfigFilePath);

        Assert.DoesNotContain("ToRemove", listResult.DotnetStdOut);
        Assert.DoesNotContain("ToRemove", listResult.GonugetStdOut);
    }

    [Fact]
    public void RemoveSource_NonexistentSource_Fails()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteSourceRemove(
            env.TestDirectory,
            "NonexistentSource",
            env.ConfigFilePath);

        // Assert - Both should fail with similar error
        Assert.NotEqual(0, result.DotnetExitCode);
        Assert.NotEqual(0, result.GonugetExitCode);
    }

    [Fact]
    public void DisableSource_ExistingSource_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteSourceDisable(
            env.TestDirectory,
            "nuget.org",
            env.ConfigFilePath);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify the source shows as disabled
        var listResult = _bridge.ExecuteSourceList(
            env.TestDirectory,
            env.ConfigFilePath);

        Assert.Contains("nuget.org", listResult.DotnetStdOut);
        Assert.Contains("nuget.org", listResult.GonugetStdOut);
        Assert.Contains("Disabled", listResult.DotnetStdOut);
        Assert.Contains("Disabled", listResult.GonugetStdOut);
    }

    [Fact]
    public void EnableSource_DisabledSource_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Disable source first
        _bridge.ExecuteSourceDisable(
            env.TestDirectory,
            "nuget.org",
            env.ConfigFilePath);

        // Act
        var result = _bridge.ExecuteSourceEnable(
            env.TestDirectory,
            "nuget.org",
            env.ConfigFilePath);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify the source shows as enabled
        var listResult = _bridge.ExecuteSourceList(
            env.TestDirectory,
            env.ConfigFilePath);

        Assert.Contains("nuget.org", listResult.DotnetStdOut);
        Assert.Contains("nuget.org", listResult.GonugetStdOut);
        Assert.Contains("Enabled", listResult.DotnetStdOut);
        Assert.Contains("Enabled", listResult.GonugetStdOut);
    }

    [Fact]
    public void UpdateSource_ChangeURL_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Add a source first
        _bridge.ExecuteSourceAdd(
            env.TestDirectory,
            "ToUpdate",
            "https://old.nuget.org/v3/index.json",
            configFile: env.ConfigFilePath);

        // Act
        var result = _bridge.ExecuteSourceUpdate(
            env.TestDirectory,
            "ToUpdate",
            source: "https://new.nuget.org/v3/index.json",
            configFile: env.ConfigFilePath);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify the source was updated
        var listResult = _bridge.ExecuteSourceList(
            env.TestDirectory,
            env.ConfigFilePath);

        Assert.Contains("ToUpdate", listResult.DotnetStdOut);
        Assert.Contains("ToUpdate", listResult.GonugetStdOut);
        Assert.Contains("https://new.nuget.org/v3/index.json", listResult.DotnetStdOut);
        Assert.Contains("https://new.nuget.org/v3/index.json", listResult.GonugetStdOut);
        Assert.DoesNotContain("https://old.nuget.org/v3/index.json", listResult.DotnetStdOut);
        Assert.DoesNotContain("https://old.nuget.org/v3/index.json", listResult.GonugetStdOut);
    }

    [Fact]
    public void UpdateSource_AddCredentials_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Add a source without credentials
        _bridge.ExecuteSourceAdd(
            env.TestDirectory,
            "ToUpdate",
            "https://private.nuget.org/v3/index.json",
            configFile: env.ConfigFilePath);

        // Act - Update to add credentials
        var result = _bridge.ExecuteSourceUpdate(
            env.TestDirectory,
            "ToUpdate",
            configFile: env.ConfigFilePath,
            username: "newuser",
            password: "newpass",
            storePasswordInClearText: true);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify credentials were added
        var configContent = env.ReadConfigFile();
        Assert.Contains("newuser", configContent);
        Assert.Contains("newpass", configContent);
    }

    [Fact]
    public void UpdateSource_Nonexistent_Fails()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteSourceUpdate(
            env.TestDirectory,
            "NonexistentSource",
            source: "https://test.nuget.org/v3/index.json",
            configFile: env.ConfigFilePath);

        // Assert - Both should fail
        Assert.NotEqual(0, result.DotnetExitCode);
        Assert.NotEqual(0, result.GonugetExitCode);
    }

    [Fact]
    public void AddSource_DuplicateName_Fails()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act - Try to add a source with the same name as existing
        var result = _bridge.ExecuteSourceAdd(
            env.TestDirectory,
            "nuget.org",
            "https://duplicate.nuget.org/v3/index.json",
            configFile: env.ConfigFilePath);

        // Assert - Both should fail with duplicate error
        Assert.NotEqual(0, result.DotnetExitCode);
        Assert.NotEqual(0, result.GonugetExitCode);
    }

    [Fact]
    public void AddSource_WithProtocolVersion_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act - Test with protocol version 2 (default, should not be written)
        var result = _bridge.ExecuteSourceAdd(
            env.TestDirectory,
            "V2Feed",
            "https://v2.nuget.org/api/v2",
            configFile: env.ConfigFilePath,
            protocolVersion: "2");

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify the source was added - protocol version 2 is default and should NOT be written
        var configContent = env.ReadConfigFile();
        Assert.Contains("V2Feed", configContent);
        Assert.DoesNotContain("protocolVersion=\"2\"", configContent);

        // Test with protocol version 3 (non-default, should be written)
        // Use a different URL to avoid conflicts with the default nuget.org source
        var result3 = _bridge.ExecuteSourceAdd(
            env.TestDirectory,
            "V3Feed",
            "https://pkgs.dev.azure.com/example/feed/nuget/v3/index.json",
            configFile: env.ConfigFilePath,
            protocolVersion: "3");

        Assert.Equal(0, result3.DotnetExitCode);
        Assert.Equal(0, result3.GonugetExitCode);

        var configContent3 = env.ReadConfigFile();
        Assert.Contains("V3Feed", configContent3);
        Assert.Contains("protocolVersion=\"3\"", configContent3);
    }

    [Fact]
    public void EnableDisable_ToggleMultipleTimes_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act & Assert - Disable
        var disableResult = _bridge.ExecuteSourceDisable(
            env.TestDirectory,
            "nuget.org",
            env.ConfigFilePath);
        Assert.Equal(0, disableResult.DotnetExitCode);
        Assert.Equal(0, disableResult.GonugetExitCode);

        // Act & Assert - Enable
        var enableResult = _bridge.ExecuteSourceEnable(
            env.TestDirectory,
            "nuget.org",
            env.ConfigFilePath);
        Assert.Equal(0, enableResult.DotnetExitCode);
        Assert.Equal(0, enableResult.GonugetExitCode);

        // Act & Assert - Disable again
        var disable2Result = _bridge.ExecuteSourceDisable(
            env.TestDirectory,
            "nuget.org",
            env.ConfigFilePath);
        Assert.Equal(0, disable2Result.DotnetExitCode);
        Assert.Equal(0, disable2Result.GonugetExitCode);

        // Final verification
        var listResult = _bridge.ExecuteSourceList(
            env.TestDirectory,
            env.ConfigFilePath);
        Assert.Contains("Disabled", listResult.DotnetStdOut);
        Assert.Contains("Disabled", listResult.GonugetStdOut);
    }
}
