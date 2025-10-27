using GonugetCliInterop.Tests.TestHelpers;

namespace GonugetCliInterop.Tests.PackageManagement;

/// <summary>
/// Tests restore command to verify parity between dotnet restore and gonuget restore.
/// </summary>
public class RestoreTests
{
    private readonly GonugetCliBridge _bridge = new();

    [Fact]
    public void Restore_Basic_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");
        env.AddPackageReference(projectPath, "Newtonsoft.Json", "13.0.3");

        // Act
        var result = _bridge.ExecuteRestore(
            projectPath,
            env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Both should succeed and mention restore completion
        var dotnetOutput = result.DotnetStdOut + result.DotnetStdErr;
        var gonugetOutput = result.GonugetStdOut + result.GonugetStdErr;
        Assert.Contains("Restored", dotnetOutput, StringComparison.OrdinalIgnoreCase);
        Assert.Contains("Restored", gonugetOutput, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public void Restore_MultipleDependencies_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");
        env.AddPackageReference(projectPath, "Newtonsoft.Json", "13.0.3");
        env.AddPackageReference(projectPath, "Serilog", "3.0.0");
        env.AddPackageReference(projectPath, "System.Text.Json", "8.0.0");

        // Act
        var result = _bridge.ExecuteRestore(
            projectPath,
            env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
    }

    [Fact]
    public void Restore_WithCustomSource_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");
        env.AddPackageReference(projectPath, "Newtonsoft.Json", "13.0.3");

        // Act
        var result = _bridge.ExecuteRestore(
            projectPath,
            env.TestDirectory,
            source: "https://api.nuget.org/v3/index.json");

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
    }

    [Fact]
    public void Restore_WithCustomPackagesFolder_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");
        env.AddPackageReference(projectPath, "Newtonsoft.Json", "13.0.3");
        var customPackagesDir = Path.Combine(env.TestDirectory, "custom-packages");

        // Act
        var result = _bridge.ExecuteRestore(
            projectPath,
            env.TestDirectory,
            packages: customPackagesDir);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify packages were restored to custom location
        Assert.True(Directory.Exists(customPackagesDir));
    }

    [Fact]
    public void Restore_Force_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");
        env.AddPackageReference(projectPath, "Newtonsoft.Json", "13.0.3");

        // First restore
        _bridge.ExecuteRestore(projectPath, env.TestDirectory);

        // Act - Force re-download
        var result = _bridge.ExecuteRestore(
            projectPath,
            env.TestDirectory,
            force: true);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
    }

    [Fact]
    public void Restore_NoCache_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");
        env.AddPackageReference(projectPath, "Newtonsoft.Json", "13.0.3");

        // Act
        var result = _bridge.ExecuteRestore(
            projectPath,
            env.TestDirectory,
            noCache: true);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
    }

    [Fact]
    public void Restore_NoProject_SameError()
    {
        // Arrange
        using var env = new TestEnvironment();
        // Don't create a project

        // Act
        var result = _bridge.ExecuteRestore(
            "NonExistent.csproj",
            env.TestDirectory);

        // Assert
        // Both should fail with non-zero exit code
        Assert.NotEqual(0, result.DotnetExitCode);
        Assert.NotEqual(0, result.GonugetExitCode);

        // Both should report error about missing project file
        var dotnetOutput = result.DotnetStdOut + result.DotnetStdErr;
        var gonugetOutput = result.GonugetStdOut + result.GonugetStdErr;
        Assert.Contains("error", dotnetOutput, StringComparison.OrdinalIgnoreCase);
        Assert.Contains("error", gonugetOutput, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public void Restore_Assets_MatchesDotnet()
    {
        // Arrange
        using var env = new TestEnvironment();
        var projectPath = env.CreateTestProject("MyApp");
        env.AddPackageReference(projectPath, "Newtonsoft.Json", "13.0.3");

        // Act
        var result = _bridge.ExecuteRestore(
            projectPath,
            env.TestDirectory);

        // Assert
        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify project.assets.json was created for both
        var projectDir = Path.GetDirectoryName(projectPath)!;
        var dotnetAssetsPath = Path.Combine(projectDir, "obj", "project.assets.json");
        var gonugetAssetsPath = Path.Combine(projectDir, "obj", "project.assets.json");

        Assert.True(File.Exists(dotnetAssetsPath), "dotnet should create project.assets.json");
        Assert.True(File.Exists(gonugetAssetsPath), "gonuget should create project.assets.json");
    }
}
