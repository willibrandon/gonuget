using GonugetCliInterop.Tests.TestHelpers;

namespace GonugetCliInterop.Tests.Foundation;

public class HelpCommandTests
{
    private readonly GonugetCliBridge _bridge = new();

    [Fact]
    public void HelpCommand_GeneralHelp_ShowsCommandList()
    {
        // Arrange
        using var env = new TestEnvironment();

        // Act
        var result = _bridge.ExecuteHelp(env.TestDirectory);

        // Assert
        Assert.True(result.ExitCodesMatch, "Exit codes should match");
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        Assert.True(result.BothShowCommands, "Both outputs should show command list");
        Assert.Contains("version", result.DotnetStdOut.ToLower());
        Assert.Contains("version", result.GonugetStdOut.ToLower());
        Assert.Contains("config", result.DotnetStdOut.ToLower());
        Assert.Contains("config", result.GonugetStdOut.ToLower());
    }

    [Fact]
    public void HelpCommand_GeneralHelp_ShowsUsageInformation()
    {
        // Arrange
        using var env = new TestEnvironment();

        // Act
        var result = _bridge.ExecuteHelp(env.TestDirectory);

        // Assert
        Assert.True(result.BothShowUsage, "Both outputs should show usage information");
        Assert.Contains("usage", result.DotnetStdOut.ToLower());
        Assert.Contains("usage", result.GonugetStdOut.ToLower());
    }

    [Fact]
    public void HelpCommand_CommandSpecificHelp_ShowsUsage()
    {
        // Arrange
        using var env = new TestEnvironment();

        // Act - Use 'source' command which exists in both CLIs
        var result = _bridge.ExecuteHelp(env.TestDirectory, "source");

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Both should show usage information for source command
        Assert.True(result.BothShowUsage, "Both outputs should show usage information");
        Assert.Contains("usage", result.DotnetStdOut.ToLower());
        Assert.Contains("usage", result.GonugetStdOut.ToLower());
    }

    [Fact]
    public void HelpCommand_VersionCommandHelp_ShowsVersionUsage()
    {
        // Arrange
        using var env = new TestEnvironment();

        // Act
        var result = _bridge.ExecuteHelp(env.TestDirectory, "version");

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        Assert.True(result.BothShowUsage, "Both outputs should show usage information");
    }

    [Fact]
    public void HelpCommand_ConfigCommandHelp_ShowsConfigUsage()
    {
        // Arrange
        using var env = new TestEnvironment();

        // Act
        var result = _bridge.ExecuteHelp(env.TestDirectory, "config");

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        Assert.True(result.BothShowUsage, "Both outputs should show usage information");
        Assert.Contains("config", result.DotnetStdOut.ToLower());
        Assert.Contains("config", result.GonugetStdOut.ToLower());
    }

    [Fact]
    public void HelpCommand_FormatIsSimilar()
    {
        // Arrange
        using var env = new TestEnvironment();

        // Act
        var result = _bridge.ExecuteHelp(env.TestDirectory);

        // Assert
        Assert.True(result.OutputFormatSimilar, "Output formats should be similar");
    }
}
