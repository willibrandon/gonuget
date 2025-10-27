using GonugetCliInterop.Tests.TestHelpers;

namespace GonugetCliInterop.Tests.PackageManagement;

/// <summary>
/// Tests add package command to verify parity between dotnet add package and gonuget add package.
/// </summary>
public class AddPackageTests
{
    private readonly GonugetCliBridge _bridge = new();

    [Fact]
    public void AddPackage_Basic_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");

        // Act
        var result = _bridge.ExecuteAddPackage(
            projectPath,
            "Newtonsoft.Json",
            env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Both commands should succeed
        Assert.Contains("Newtonsoft.Json", result.DotnetStdOut);
        Assert.Contains("Newtonsoft.Json", result.GonugetStdOut);
    }

    [Fact]
    public void AddPackage_WithVersion_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");

        // Act
        var result = _bridge.ExecuteAddPackage(
            projectPath,
            "Newtonsoft.Json",
            env.TestDirectory,
            version: "13.0.1");

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify version is in output
        Assert.Contains("13.0.1", result.DotnetStdOut);
        Assert.Contains("13.0.1", result.GonugetStdOut);
    }

    [Fact]
    public void AddPackage_WithPrerelease_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");

        // Act
        var result = _bridge.ExecuteAddPackage(
            projectPath,
            "Newtonsoft.Json",
            env.TestDirectory,
            prerelease: true);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
    }

    [Fact]
    public void AddPackage_NoRestore_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");

        // Act
        var result = _bridge.ExecuteAddPackage(
            projectPath,
            "Newtonsoft.Json",
            env.TestDirectory,
            noRestore: true);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Should not restore when --no-restore is specified
        Assert.DoesNotContain("Restored", result.DotnetStdOut);
        Assert.DoesNotContain("Restored", result.GonugetStdOut);
    }

    [Fact]
    public void AddPackage_UpdateExisting_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");

        // Add package first time with old version
        _bridge.ExecuteAddPackage(
            projectPath,
            "Newtonsoft.Json",
            env.TestDirectory,
            version: "12.0.1",
            noRestore: true);

        // Act - Add same package with newer version
        var result = _bridge.ExecuteAddPackage(
            projectPath,
            "Newtonsoft.Json",
            env.TestDirectory,
            version: "13.0.1",
            noRestore: true);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Should update to new version
        Assert.Contains("13.0.1", result.DotnetStdOut);
        Assert.Contains("13.0.1", result.GonugetStdOut);
    }

    [Fact]
    public void AddPackage_InvalidPackage_SameError()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");

        // Act - Don't use --no-restore, so both tools attempt to resolve the package
        var result = _bridge.ExecuteAddPackage(
            projectPath,
            "NonExistentPackage12345678",
            env.TestDirectory);

        // Assert
        // Both should fail with non-zero exit code
        Assert.NotEqual(0, result.DotnetExitCode);
        Assert.NotEqual(0, result.GonugetExitCode);

        // Both should report error about package not found (dotnet writes to stdout, not stderr)
        var dotnetOutput = result.DotnetStdOut + result.DotnetStdErr;
        var gonugetOutput = result.GonugetStdOut + result.GonugetStdErr;
        Assert.Contains("error", dotnetOutput, StringComparison.OrdinalIgnoreCase);
        Assert.Contains("error", gonugetOutput, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public void AddPackage_NoProject_SameError()
    {
        // Arrange
        using var env = new TestEnvironment();
        // Don't create a project

        // Act
        var result = _bridge.ExecuteAddPackage(
            "NonExistent.csproj",
            "Newtonsoft.Json",
            env.TestDirectory);

        // Assert
        // Both should fail with non-zero exit code
        Assert.NotEqual(0, result.DotnetExitCode);
        Assert.NotEqual(0, result.GonugetExitCode);

        // Both should report error about missing project file
        // dotnet: "Could not find project or directory"
        // gonuget: "Error: failed to load project"
        Assert.Contains("could not find", result.DotnetStdErr, StringComparison.OrdinalIgnoreCase);
        Assert.Contains("failed to load project", result.GonugetStdErr, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public void AddPackage_WithSource_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");

        // Act
        var result = _bridge.ExecuteAddPackage(
            projectPath,
            "Newtonsoft.Json",
            env.TestDirectory,
            source: "https://api.nuget.org/v3/index.json");

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
    }
}
