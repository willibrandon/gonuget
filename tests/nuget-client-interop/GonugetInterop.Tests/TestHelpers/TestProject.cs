using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Helper class to create temporary .NET test projects for restore testing.
/// </summary>
public class TestProject : IDisposable
{
    private readonly string _tempDir;
    private readonly List<PackageReference> _packageReferences = new();

    public string Name { get; }
    public string Path { get; private set; } = string.Empty;
    public string TargetFramework { get; private set; }
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
    /// </summary>
    /// <returns>Test project instance with Path set to the created .csproj file</returns>
    public TestProject Build()
    {
        var projectFileName = $"{Name}.csproj";
        Path = System.IO.Path.Combine(_tempDir, projectFileName);

        var csprojContent = GenerateCsprojContent();
        File.WriteAllText(Path, csprojContent, Encoding.UTF8);

        return this;
    }

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
    /// Cleans up the temporary project directory.
    /// </summary>
    public void Dispose()
    {
        try
        {
            if (Directory.Exists(_tempDir))
            {
                Directory.Delete(_tempDir, recursive: true);
            }
        }
        catch
        {
            // Best effort cleanup - don't throw in Dispose
        }
    }
}

/// <summary>
/// Represents a NuGet package reference in a project.
/// </summary>
public record PackageReference(string PackageId, string Version, bool IncludePrerelease = false);
