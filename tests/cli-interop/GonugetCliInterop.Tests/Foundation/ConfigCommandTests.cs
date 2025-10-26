using GonugetCliInterop.Tests.TestHelpers;
using Xunit;

namespace GonugetCliInterop.Tests.Foundation;

public class ConfigCommandTests
{
    private readonly GonugetCliBridge _bridge = new();

    [Fact]
    public void ConfigGet_SimpleValue_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfigWithValues(new Dictionary<string, string>
        {
            { "repositoryPath", "~/test-packages" }
        });

        // Act
        var result = _bridge.ExecuteConfigGet(
            "repositoryPath",
            env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
        Assert.Contains("~/test-packages", result.DotnetStdOut);
        Assert.Contains("~/test-packages", result.GonugetStdOut);
    }

    [Fact]
    public void ConfigGet_AllKeyword_ShowsAllSections()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfigWithValues(new Dictionary<string, string>
        {
            { "key1", "value1" },
            { "key2", "value2" }
        });

        // Act
        var result = _bridge.ExecuteConfigGet(
            "all",
            env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Both should show packageSources and config sections
        Assert.Contains("packageSources", result.DotnetStdOut);
        Assert.Contains("packageSources", result.GonugetStdOut);
        Assert.Contains("config", result.DotnetStdOut);
        Assert.Contains("config", result.GonugetStdOut);

        // Both should show the keys
        Assert.Contains("key1", result.DotnetStdOut);
        Assert.Contains("key1", result.GonugetStdOut);
        Assert.Contains("key2", result.DotnetStdOut);
        Assert.Contains("key2", result.GonugetStdOut);
    }

    [Fact]
    public void ConfigGet_NotFound_ReturnsSameError()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteConfigGet(
            "nonexistentKey",
            env.TestDirectory);

        // Assert
        // Both should fail with non-zero exit code
        Assert.NotEqual(0, result.DotnetExitCode);
        Assert.NotEqual(0, result.GonugetExitCode);

        // Exit codes should match
        Assert.Equal(result.DotnetExitCode, result.GonugetExitCode);
    }

    [Fact]
    public void ConfigSet_NewValue_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteConfigSet(
            "repositoryPath",
            "~/my-packages",
            env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify the value was set
        Assert.True(env.ConfigContains("repositoryPath", "~/my-packages"));
    }

    [Fact]
    public void ConfigSet_UpdateValue_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfigWithValues(new Dictionary<string, string>
        {
            { "repositoryPath", "~/old-packages" }
        });

        // Act
        var result = _bridge.ExecuteConfigSet(
            "repositoryPath",
            "~/new-packages",
            env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify the value was updated
        Assert.True(env.ConfigContains("repositoryPath", "~/new-packages"));
        Assert.False(env.ConfigContains("repositoryPath", "~/old-packages"));
    }

    [Fact]
    public void ConfigSet_RequiresExistingFile()
    {
        // Arrange
        using var env = new TestEnvironment();
        // Don't create a config file - both commands should fail

        // Act
        var result = _bridge.ExecuteConfigSet(
            "testKey",
            "testValue",
            env.TestDirectory);

        // Assert
        // Both should fail with non-zero exit code when no config file exists
        Assert.NotEqual(0, result.DotnetExitCode);
        Assert.NotEqual(0, result.GonugetExitCode);
    }

    [Fact]
    public void ConfigUnset_ExistingValue_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfigWithValues(new Dictionary<string, string>
        {
            { "repositoryPath", "~/test-packages" }
        });

        // Act
        var result = _bridge.ExecuteConfigUnset(
            "repositoryPath",
            env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify the value was removed
        Assert.False(env.ConfigContainsKey("repositoryPath"));
    }

    [Fact]
    public void ConfigUnset_NonExistentValue_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteConfigUnset(
            "nonexistentKey",
            env.TestDirectory);

        // Assert
        // Both should succeed (removing non-existent key is not an error)
        Assert.Equal(result.DotnetExitCode, result.GonugetExitCode);
    }

    [Fact]
    public void ConfigPaths_ShowsConfigHierarchy()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfig();

        // Act
        var result = _bridge.ExecuteConfigPaths(
            env.TestDirectory,
            workingDirFlag: env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Both should show the test config file
        Assert.Contains(env.TestDirectory, result.DotnetStdOut);
        Assert.Contains(env.TestDirectory, result.GonugetStdOut);

        // Both should show user config path
        // On Windows: %APPDATA%\NuGet (e.g., C:\Users\<user>\AppData\Roaming\NuGet)
        // On Unix: ~/.nuget/NuGet
        var expectedUserConfigMarker = OperatingSystem.IsWindows() ? "appdata" : ".nuget";
        Assert.Contains(expectedUserConfigMarker, result.DotnetStdOut.ToLower());
        Assert.Contains(expectedUserConfigMarker, result.GonugetStdOut.ToLower());
    }

    [Fact]
    public void ConfigGet_ShowPathFlag_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfigWithValues(new Dictionary<string, string>
        {
            { "relativePath", "./packages" }
        });

        // Act
        var result = _bridge.ExecuteConfigGet(
            "relativePath",
            env.TestDirectory,
            showPath: true);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Both should return format: <value><TAB>file: <path>
        Assert.Contains("./packages", result.DotnetStdOut);
        Assert.Contains("./packages", result.GonugetStdOut);
        Assert.Contains("\tfile: ", result.DotnetStdOut);
        Assert.Contains("\tfile: ", result.GonugetStdOut);
        Assert.Contains(env.TestDirectory, result.DotnetStdOut);
        Assert.Contains(env.TestDirectory, result.GonugetStdOut);
    }

    [Fact]
    public void ConfigGet_WorkingDirectoryFlag_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        env.CreateTestConfigWithValues(new Dictionary<string, string>
        {
            { "testKey", "testValue" }
        });

        // Act
        var result = _bridge.ExecuteConfigGet(
            "testKey",
            env.TestDirectory,
            workingDirFlag: env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
        Assert.Contains("testValue", result.DotnetStdOut);
        Assert.Contains("testValue", result.GonugetStdOut);
    }
}
