using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Helper class to create temporary .NET test projects for restore testing.
/// Provides a fluent builder API for creating test projects with specific target frameworks
/// and package references, then writing them to disk in a temporary directory.
/// </summary>
/// <remarks>
/// Implements IDisposable to automatically clean up temporary directories.
/// Use with 'using' statements to ensure cleanup after tests complete.
/// </remarks>
/// <example>
/// <code>
/// using var project = TestProject.Create("MyTest")
///     .WithFramework("net8.0")
///     .AddPackage("Newtonsoft.Json", "13.0.3")
///     .Build();
/// // Use project.Path for restore operations
/// </code>
/// </example>
public class TestProject : IDisposable
{
    private readonly string _tempDir;
    private readonly List<PackageReference> _packageReferences = new();

    /// <summary>
    /// Gets the name of the test project.
    /// </summary>
    public string Name { get; }

    /// <summary>
    /// Gets the full path to the generated .csproj file. Empty until Build() is called.
    /// </summary>
    public string Path { get; private set; } = string.Empty;

    /// <summary>
    /// Gets or sets the target framework moniker (TFM) for the project (e.g., "net9.0", "net8.0").
    /// Defaults to "net9.0" if not explicitly set.
    /// </summary>
    public string TargetFramework { get; private set; }

    /// <summary>
    /// Gets the list of package references configured for this project.
    /// </summary>
    public IReadOnlyList<PackageReference> PackageReferences => _packageReferences.AsReadOnly();

    private TestProject(string name)
    {
        Name = name;
        _tempDir = System.IO.Path.Combine(System.IO.Path.GetTempPath(), "gonuget-tests", Guid.NewGuid().ToString());
        TargetFramework = "net9.0"; // Default framework
        Directory.CreateDirectory(_tempDir);
    }

    /// <summary>
    /// Creates a new test project builder.
    /// </summary>
    /// <param name="name">Project name</param>
    /// <returns>Test project builder instance</returns>
    public static TestProject Create(string name)
    {
        if (string.IsNullOrWhiteSpace(name))
        {
            throw new ArgumentException("Project name cannot be null or whitespace", nameof(name));
        }

        return new TestProject(name);
    }

    /// <summary>
    /// Sets the target framework for the project.
    /// </summary>
    /// <param name="targetFramework">Target framework moniker (e.g., "net9.0", "net8.0", "net6.0")</param>
    /// <returns>Test project builder instance for chaining</returns>
    public TestProject WithFramework(string targetFramework)
    {
        if (string.IsNullOrWhiteSpace(targetFramework))
        {
            throw new ArgumentException("Target framework cannot be null or whitespace", nameof(targetFramework));
        }

        TargetFramework = targetFramework;
        return this;
    }

    /// <summary>
    /// Adds a package reference to the project.
    /// </summary>
    /// <param name="packageId">NuGet package identifier</param>
    /// <param name="version">Package version or version range</param>
    /// <param name="includePrerelease">Whether to include prerelease versions</param>
    /// <returns>Test project builder instance for chaining</returns>
    public TestProject AddPackage(string packageId, string version, bool includePrerelease = false)
    {
        if (string.IsNullOrWhiteSpace(packageId))
        {
            throw new ArgumentException("Package ID cannot be null or whitespace", nameof(packageId));
        }

        if (string.IsNullOrWhiteSpace(version))
        {
            throw new ArgumentException("Version cannot be null or whitespace", nameof(version));
        }

        _packageReferences.Add(new PackageReference(packageId, version, includePrerelease));
        return this;
    }

    /// <summary>
    /// Builds the test project by writing the .csproj file to disk.
    /// This method must be called after configuring the project with WithFramework() and/or AddPackage().
    /// </summary>
    /// <returns>Test project instance with Path property set to the created .csproj file path</returns>
    /// <exception cref="IOException">Thrown if the .csproj file cannot be written to disk</exception>
    public TestProject Build()
    {
        var projectFileName = $"{Name}.csproj";
        Path = System.IO.Path.Combine(_tempDir, projectFileName);

        try
        {
            var csprojContent = GenerateCsprojContent();
            File.WriteAllText(Path, csprojContent, Encoding.UTF8);
        }
        catch (Exception ex) when (ex is IOException || ex is UnauthorizedAccessException || ex is System.Security.SecurityException)
        {
            throw new IOException(
                $"Failed to create test project file.\n" +
                $"Project name: {Name}\n" +
                $"Target path: {Path}\n" +
                $"Temp directory: {_tempDir}\n" +
                $"Target framework: {TargetFramework}\n" +
                $"Package count: {_packageReferences.Count}\n" +
                $"Error: {ex.Message}", ex);
        }

        // Verify file was created successfully
        if (!File.Exists(Path))
        {
            throw new IOException(
                $"Test project file was not created.\n" +
                $"Expected path: {Path}\n" +
                $"Directory exists: {Directory.Exists(_tempDir)}\n" +
                $"This may indicate a filesystem permissions issue.");
        }

        return this;
    }

    /// <summary>
    /// Generates the .csproj file content as XML based on configured target framework and package references.
    /// </summary>
    /// <returns>Complete .csproj file content as a string</returns>
    private string GenerateCsprojContent()
    {
        var sb = new StringBuilder();
        sb.AppendLine("<Project Sdk=\"Microsoft.NET.Sdk\">");
        sb.AppendLine();
        sb.AppendLine("  <PropertyGroup>");
        sb.AppendLine("    <OutputType>Exe</OutputType>");
        sb.AppendLine($"    <TargetFramework>{TargetFramework}</TargetFramework>");
        sb.AppendLine("    <ImplicitUsings>enable</ImplicitUsings>");
        sb.AppendLine("    <Nullable>enable</Nullable>");
        sb.AppendLine("  </PropertyGroup>");

        if (_packageReferences.Any())
        {
            sb.AppendLine();
            sb.AppendLine("  <ItemGroup>");
            foreach (var pkgRef in _packageReferences)
            {
                sb.AppendLine($"    <PackageReference Include=\"{pkgRef.PackageId}\" Version=\"{pkgRef.Version}\" />");
            }
            sb.AppendLine("  </ItemGroup>");
        }

        sb.AppendLine();
        sb.AppendLine("</Project>");

        return sb.ToString();
    }

    /// <summary>
    /// Cleans up the temporary project directory and all its contents.
    /// This method is called automatically when the object is disposed (via using statement or explicit Dispose()).
    /// </summary>
    /// <remarks>
    /// Uses best-effort cleanup - exceptions during cleanup are logged to console but not thrown
    /// to prevent exceptions from Dispose() which could hide test failures.
    /// If cleanup fails, the temporary directory path is written to console for manual cleanup.
    /// </remarks>
    public void Dispose()
    {
        try
        {
            if (Directory.Exists(_tempDir))
            {
                Directory.Delete(_tempDir, recursive: true);
            }
        }
        catch (Exception ex)
        {
            // Best effort cleanup - don't throw in Dispose
            // Write diagnostic info to console for debugging
            Console.WriteLine(
                $"WARNING: Failed to clean up test project temporary directory.\n" +
                $"Project: {Name}\n" +
                $"Path: {_tempDir}\n" +
                $"Error: {ex.GetType().Name}: {ex.Message}\n" +
                $"Manual cleanup may be required.\n");
        }
    }
}

/// <summary>
/// Represents a NuGet package reference in a project (.csproj PackageReference element).
/// </summary>
/// <param name="PackageId">The NuGet package identifier (e.g., "Newtonsoft.Json")</param>
/// <param name="Version">The package version or version range (e.g., "13.0.3" or "[13.0.0, 14.0.0)")</param>
/// <param name="IncludePrerelease">Whether to include prerelease versions when resolving this package. Defaults to false.</param>
public record PackageReference(string PackageId, string Version, bool IncludePrerelease = false);
