using GonugetCliInterop.Tests.TestHelpers;
using Xunit;

namespace GonugetCliInterop.Tests.Foundation;

public class VersionCommandTests
{
    private readonly GonugetCliBridge _bridge = new();

    [Fact]
    public void VersionCommand_ReturnsVersionInfo()
    {
        // Arrange
        using var env = new TestEnvironment();

        // Act
        var result = _bridge.ExecuteVersion(env.TestDirectory);

        // Assert
        Assert.True(result.ExitCodesMatch, "Exit codes should match");
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        Assert.True(result.OutputFormatSimilar, "Both outputs should contain version information");
        Assert.Contains("nuget", result.DotnetStdOut.ToLower());
        Assert.Contains("version", result.GonugetStdOut.ToLower());
    }

    [Fact]
    public void VersionCommand_ExitCodeIsZero()
    {
        // Arrange
        using var env = new TestEnvironment();

        // Act
        var result = _bridge.ExecuteVersion(env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
    }

    [Fact]
    public void VersionCommand_OutputContainsVersionNumber()
    {
        // Arrange
        using var env = new TestEnvironment();

        // Act
        var result = _bridge.ExecuteVersion(env.TestDirectory);

        // Assert
        // Both should output version information (numbers)
        Assert.Matches(@"\d+\.\d+", result.DotnetStdOut);
        Assert.Matches(@"\d+\.\d+", result.GonugetStdOut);
    }
}
