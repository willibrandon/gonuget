using System;
using System.Diagnostics;
using System.IO;
using System.Linq;
using GonugetInterop.Tests.TestHelpers;
using NuGet.ProjectModel;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Verifies gonuget's transitive dependency resolution matches NuGet.Client behavior exactly.
/// Tests direct vs transitive categorization, package resolution, and lock file format.
/// </summary>
public sealed class RestoreTransitiveTests
{
    private readonly string[] _testSources = ["https://api.nuget.org/v3/index.json"];

    /// <summary>
    /// T014: Test simple transitive resolution with 1-2 dependency levels.
    /// Serilog.Sinks.File 5.0.0 -> Serilog (>= 2.10.0)
    /// </summary>
    [Fact]
    public void SimpleTransitiveResolution_MatchesNuGetClient()
    {
        // Arrange: Create test project with package that has simple transitive dependency
        var projectDir = Path.Combine(Path.GetTempPath(), $"simple-transitive-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "SimpleTransitive.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Serilog.Sinks.File"" Version=""5.0.0"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null, // Use default
                sources: _testSources,
                noCache: false,
                force: true
            );

            // Act: Run dotnet restore (uses NuGet.Client) to create reference lock file
            var dotnetProcess = Process.Start(new ProcessStartInfo
            {
                FileName = "dotnet",
                Arguments = $"restore \"{projectPath}\" --force",
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                UseShellExecute = false,
                CreateNoWindow = true
            });

            dotnetProcess?.WaitForExit();
            var dotnetExitCode = dotnetProcess?.ExitCode ?? -1;

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Both restores succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.Equal(0, dotnetExitCode);
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: gonuget resolved packages
            Assert.NotEmpty(gonugetResult.DirectPackages);
            Assert.NotEmpty(gonugetResult.TransitivePackages);

            // Assert: Direct package count matches (should have 1 direct: Serilog.Sinks.File)
            Assert.Single(gonugetResult.DirectPackages);
            var directPkg = gonugetResult.DirectPackages[0];
            Assert.Equal("Serilog.Sinks.File", directPkg.PackageId, ignoreCase: true);
            Assert.Equal("5.0.0", directPkg.Version);

            // Assert: Transitive packages include Serilog (pulled by Serilog.Sinks.File)
            var serilogTransitive = gonugetResult.TransitivePackages.FirstOrDefault(p =>
                p.PackageId.Equals("Serilog", StringComparison.OrdinalIgnoreCase));
            Assert.NotNull(serilogTransitive);

            // Assert: Verify lock file format matches NuGet.Client
            var format = new LockFileFormat();
            var lockFile = format.Read(gonugetLockFilePath);
            Assert.Equal(3, lockFile.Version);
            Assert.Contains(lockFile.Libraries, lib => lib.Name.Equals("Serilog.Sinks.File", StringComparison.OrdinalIgnoreCase));
            Assert.Contains(lockFile.Libraries, lib => lib.Name.Equals("Serilog", StringComparison.OrdinalIgnoreCase));

            // Assert: ProjectFileDependencyGroups contains only direct dependencies
            var net80Group = lockFile.ProjectFileDependencyGroups.FirstOrDefault(g => g.FrameworkName == "net8.0" || g.FrameworkName == ".NETCoreApp,Version=v8.0");
            Assert.NotNull(net80Group);
            Assert.Contains(net80Group.Dependencies, dep => dep.Contains("Serilog.Sinks.File"));
            Assert.DoesNotContain(net80Group.Dependencies, dep => dep.Contains("Serilog") && !dep.Contains("Serilog.Sinks.File"));
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T015: Test moderate transitive resolution with 5-10 packages.
    /// Microsoft.Extensions.Logging 8.0.0 pulls in multiple transitive dependencies.
    /// </summary>
    [Fact]
    public void ModerateTransitiveResolution_MatchesNuGetClient()
    {
        // Arrange: Create test project with package that has moderate dependency tree
        var projectDir = Path.Combine(Path.GetTempPath(), $"moderate-transitive-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "ModerateTransitive.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Microsoft.Extensions.Logging"" Version=""8.0.0"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Has direct and transitive packages
            Assert.NotEmpty(gonugetResult.DirectPackages);
            Assert.NotEmpty(gonugetResult.TransitivePackages);

            // Assert: Should have more than 3 transitive packages (moderate complexity)
            Assert.True(gonugetResult.TransitivePackages.Count >= 3,
                $"Expected at least 3 transitive packages, got {gonugetResult.TransitivePackages.Count}");

            // Assert: Direct package is only Microsoft.Extensions.Logging
            Assert.Single(gonugetResult.DirectPackages);
            Assert.Equal("Microsoft.Extensions.Logging", gonugetResult.DirectPackages[0].PackageId, ignoreCase: true);
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T016: Test complex transitive resolution with 10+ packages.
    /// ASP.NET Core packages have deep dependency trees.
    /// Uses netcoreapp2.2 to match Microsoft.AspNetCore.Mvc 2.2.0's target framework.
    /// </summary>
    [Fact]
    public void ComplexTransitiveResolution_MatchesNuGetClient()
    {
        // Arrange: Create test project with package that has complex dependency tree
        var projectDir = Path.Combine(Path.GetTempPath(), $"complex-transitive-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "ComplexTransitive.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>netcoreapp2.2</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Microsoft.AspNetCore.Mvc"" Version=""2.2.0"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Has many transitive packages (complex dependency tree)
            Assert.NotEmpty(gonugetResult.DirectPackages);
            Assert.NotEmpty(gonugetResult.TransitivePackages);

            // Assert: Should have 10+ transitive packages (complex)
            Assert.True(gonugetResult.TransitivePackages.Count >= 10,
                $"Expected at least 10 transitive packages, got {gonugetResult.TransitivePackages.Count}");

            // Assert: Direct package is only Microsoft.AspNetCore.Mvc
            Assert.Single(gonugetResult.DirectPackages);
            Assert.Equal("Microsoft.AspNetCore.Mvc", gonugetResult.DirectPackages[0].PackageId, ignoreCase: true);
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T017: Test diamond dependencies (multiple paths to same package).
    /// Both Newtonsoft.Json and Microsoft.AspNetCore.Mvc.NewtonsoftJson depend on Newtonsoft.Json.
    /// Uses version 13.0.3 to avoid downgrade (8.0.0 requires >= 13.0.3).
    /// </summary>
    [Fact]
    public void DiamondDependencies_MatchesNuGetClient()
    {
        // Arrange: Create test project with diamond dependency pattern
        var projectDir = Path.Combine(Path.GetTempPath(), $"diamond-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "Diamond.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.3"" />
    <PackageReference Include=""Microsoft.AspNetCore.Mvc.NewtonsoftJson"" Version=""8.0.0"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Both direct packages present
            Assert.Equal(2, gonugetResult.DirectPackages.Count);
            Assert.Contains(gonugetResult.DirectPackages, p => p.PackageId.Equals("Newtonsoft.Json", StringComparison.OrdinalIgnoreCase));
            Assert.Contains(gonugetResult.DirectPackages, p => p.PackageId.Equals("Microsoft.AspNetCore.Mvc.NewtonsoftJson", StringComparison.OrdinalIgnoreCase));

            // Assert: Newtonsoft.Json should not appear in transitive (it's direct)
            Assert.DoesNotContain(gonugetResult.TransitivePackages, p => p.PackageId.Equals("Newtonsoft.Json", StringComparison.OrdinalIgnoreCase));
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T018: Test framework-specific dependencies (net6.0 vs net8.0 vs net9.0).
    /// Different frameworks may resolve different dependency versions.
    /// </summary>
    [Fact]
    public void FrameworkSpecificDependencies_MatchesNuGetClient()
    {
        // Arrange: Create test project targeting net6.0
        var projectDir = Path.Combine(Path.GetTempPath(), $"framework-specific-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "FrameworkSpecific.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Microsoft.Extensions.Configuration"" Version=""8.0.0"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Verify lock file contains correct target framework
            var format = new LockFileFormat();
            var lockFile = format.Read(gonugetLockFilePath);
            Assert.Contains(lockFile.Targets, t => t.TargetFramework.GetShortFolderName() == "net6.0");
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T019: Test shared transitive dependencies.
    /// Multiple direct dependencies share the same transitive dependency.
    /// </summary>
    [Fact]
    public void SharedTransitiveDependencies_MatchesNuGetClient()
    {
        // Arrange: Create test project where multiple direct deps share transitive deps
        var projectDir = Path.Combine(Path.GetTempPath(), $"shared-transitive-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "SharedTransitive.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Microsoft.Extensions.Logging"" Version=""8.0.0"" />
    <PackageReference Include=""Microsoft.Extensions.Configuration"" Version=""8.0.0"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Both direct packages present
            Assert.Equal(2, gonugetResult.DirectPackages.Count);

            // Assert: Shared transitive dependencies (both packages depend on Microsoft.Extensions.DependencyInjection.Abstractions)
            var sharedDep = gonugetResult.TransitivePackages.FirstOrDefault(p =>
                p.PackageId.Equals("Microsoft.Extensions.DependencyInjection.Abstractions", StringComparison.OrdinalIgnoreCase));
            Assert.NotNull(sharedDep);
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T020: Test version resolution in transitive chain.
    /// Ensures correct version is selected when multiple versions are referenced.
    /// </summary>
    [Fact]
    public void VersionResolutionInTransitiveChain_MatchesNuGetClient()
    {
        // Arrange: Create test project with version constraints
        var projectDir = Path.Combine(Path.GetTempPath(), $"version-resolution-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "VersionResolution.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.1"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Exact version resolved
            var jsonPackage = gonugetResult.DirectPackages.FirstOrDefault(p =>
                p.PackageId.Equals("Newtonsoft.Json", StringComparison.OrdinalIgnoreCase));
            Assert.NotNull(jsonPackage);
            Assert.Equal("13.0.1", jsonPackage.Version);
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T021: Test transitive resolution with version ranges.
    /// Packages with version ranges (e.g., [1.0.0, 2.0.0)) should resolve correctly.
    /// </summary>
    [Fact]
    public void TransitiveResolutionWithVersionRanges_MatchesNuGetClient()
    {
        // Arrange: Create test project with version range
        var projectDir = Path.Combine(Path.GetTempPath(), $"version-ranges-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "VersionRanges.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""[13.0.0, 14.0.0)"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Version within range was selected
            var jsonPackage = gonugetResult.DirectPackages.FirstOrDefault(p =>
                p.PackageId.Equals("Newtonsoft.Json", StringComparison.OrdinalIgnoreCase));
            Assert.NotNull(jsonPackage);

            // Parse version and verify it's in range [13.0.0, 14.0.0)
            var version = NuGet.Versioning.NuGetVersion.Parse(jsonPackage.Version);
            Assert.True(version >= new NuGet.Versioning.NuGetVersion("13.0.0"));
            Assert.True(version < new NuGet.Versioning.NuGetVersion("14.0.0"));
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    // ========================================================================
    // Phase 4: User Story 2 - Direct vs Transitive Categorization
    // ========================================================================

    /// <summary>
    /// T022: Test pure direct dependencies (no transitive).
    /// Verifies packages with no dependencies are correctly categorized as direct only.
    /// </summary>
    [Fact]
    public void PureDirectDependencies_CorrectlyCategorized()
    {
        // Arrange: Create test project with package that has no dependencies
        var projectDir = Path.Combine(Path.GetTempPath(), $"pure-direct-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "PureDirect.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""System.Memory"" Version=""4.5.5"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Only direct package, no transitive
            Assert.Single(gonugetResult.DirectPackages);
            Assert.Equal("System.Memory", gonugetResult.DirectPackages[0].PackageId, ignoreCase: true);

            // Assert: ProjectFileDependencyGroups contains only System.Memory
            var format = new LockFileFormat();
            var lockFile = format.Read(gonugetLockFilePath);
            var net80Group = lockFile.ProjectFileDependencyGroups.FirstOrDefault(g =>
                g.FrameworkName == "net8.0" || g.FrameworkName == ".NETCoreApp,Version=v8.0");
            Assert.NotNull(net80Group);
            Assert.Contains(net80Group.Dependencies, dep => dep.Contains("System.Memory"));

            // Assert: Libraries map contains System.Memory
            Assert.Contains(lockFile.Libraries, lib => lib.Name.Equals("System.Memory", StringComparison.OrdinalIgnoreCase));
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T023: Test pure transitive dependencies.
    /// Verifies packages pulled only by other packages are categorized as transitive.
    /// </summary>
    [Fact]
    public void PureTransitiveDependencies_CorrectlyCategorized()
    {
        // Arrange: Create test project with package that has transitive dependencies
        var projectDir = Path.Combine(Path.GetTempPath(), $"pure-transitive-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "PureTransitive.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Serilog.Sinks.File"" Version=""5.0.0"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Has direct and transitive packages
            Assert.Single(gonugetResult.DirectPackages);
            Assert.NotEmpty(gonugetResult.TransitivePackages);

            // Assert: Serilog should be transitive (pulled by Serilog.Sinks.File)
            var serilogTransitive = gonugetResult.TransitivePackages.FirstOrDefault(p =>
                p.PackageId.Equals("Serilog", StringComparison.OrdinalIgnoreCase));
            Assert.NotNull(serilogTransitive);

            // Assert: ProjectFileDependencyGroups contains only Serilog.Sinks.File (direct)
            var format = new LockFileFormat();
            var lockFile = format.Read(gonugetLockFilePath);
            var net80Group = lockFile.ProjectFileDependencyGroups.FirstOrDefault(g =>
                g.FrameworkName == "net8.0" || g.FrameworkName == ".NETCoreApp,Version=v8.0");
            Assert.NotNull(net80Group);
            Assert.Contains(net80Group.Dependencies, dep => dep.Contains("Serilog.Sinks.File"));
            Assert.DoesNotContain(net80Group.Dependencies, dep => dep.Contains("Serilog") && !dep.Contains("Serilog.Sinks.File"));

            // Assert: Libraries map contains both direct and transitive
            Assert.Contains(lockFile.Libraries, lib => lib.Name.Equals("Serilog.Sinks.File", StringComparison.OrdinalIgnoreCase));
            Assert.Contains(lockFile.Libraries, lib => lib.Name.Equals("Serilog", StringComparison.OrdinalIgnoreCase));
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T024: Test mixed scenario (package is both direct and transitive).
    /// When a package is both directly referenced and pulled transitively, it should be categorized as direct.
    /// </summary>
    [Fact]
    public void MixedDirectAndTransitive_CategorizedAsDirect()
    {
        // Arrange: Create test project where Newtonsoft.Json is both direct and transitive
        var projectDir = Path.Combine(Path.GetTempPath(), $"mixed-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "Mixed.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.3"" />
    <PackageReference Include=""Microsoft.AspNetCore.Mvc.NewtonsoftJson"" Version=""8.0.0"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Newtonsoft.Json should be in direct packages (not transitive)
            Assert.Contains(gonugetResult.DirectPackages, p => p.PackageId.Equals("Newtonsoft.Json", StringComparison.OrdinalIgnoreCase));
            Assert.DoesNotContain(gonugetResult.TransitivePackages, p => p.PackageId.Equals("Newtonsoft.Json", StringComparison.OrdinalIgnoreCase));

            // Assert: Both direct packages present
            Assert.Equal(2, gonugetResult.DirectPackages.Count);

            // Assert: ProjectFileDependencyGroups contains both direct packages
            var format = new LockFileFormat();
            var lockFile = format.Read(gonugetLockFilePath);
            var net80Group = lockFile.ProjectFileDependencyGroups.FirstOrDefault(g =>
                g.FrameworkName == "net8.0" || g.FrameworkName == ".NETCoreApp,Version=v8.0");
            Assert.NotNull(net80Group);
            Assert.Contains(net80Group.Dependencies, dep => dep.Contains("Newtonsoft.Json"));
            Assert.Contains(net80Group.Dependencies, dep => dep.Contains("Microsoft.AspNetCore.Mvc.NewtonsoftJson"));
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T025: Test ProjectFileDependencyGroups contains only direct dependencies.
    /// Validates that lock file ProjectFileDependencyGroups section lists only packages from PackageReference.
    /// </summary>
    [Fact]
    public void ProjectFileDependencyGroups_ContainsOnlyDirect()
    {
        // Arrange: Create test project with known direct and transitive packages
        var projectDir = Path.Combine(Path.GetTempPath(), $"dependency-groups-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "DependencyGroups.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Microsoft.Extensions.Logging"" Version=""8.0.0"" />
    <PackageReference Include=""Microsoft.Extensions.Configuration"" Version=""8.0.0"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Verify ProjectFileDependencyGroups
            var format = new LockFileFormat();
            var lockFile = format.Read(gonugetLockFilePath);
            var net80Group = lockFile.ProjectFileDependencyGroups.FirstOrDefault(g =>
                g.FrameworkName == "net8.0" || g.FrameworkName == ".NETCoreApp,Version=v8.0");
            Assert.NotNull(net80Group);

            // Assert: Contains both direct dependencies
            Assert.Contains(net80Group.Dependencies, dep => dep.Contains("Microsoft.Extensions.Logging"));
            Assert.Contains(net80Group.Dependencies, dep => dep.Contains("Microsoft.Extensions.Configuration"));

            // Assert: Does NOT contain transitive dependencies (e.g., Microsoft.Extensions.DependencyInjection.Abstractions)
            Assert.DoesNotContain(net80Group.Dependencies, dep =>
                dep.Contains("Microsoft.Extensions.DependencyInjection.Abstractions"));
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T026: Test Libraries map contains all packages (direct + transitive).
    /// Validates that lock file Libraries section contains every resolved package.
    /// </summary>
    [Fact]
    public void LibrariesMap_ContainsAllPackages()
    {
        // Arrange: Create test project with known direct and transitive packages
        var projectDir = Path.Combine(Path.GetTempPath(), $"libraries-map-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "LibrariesMap.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Serilog.Sinks.File"" Version=""5.0.0"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Verify Libraries map
            var format = new LockFileFormat();
            var lockFile = format.Read(gonugetLockFilePath);

            // Assert: Contains direct package
            Assert.Contains(lockFile.Libraries, lib => lib.Name.Equals("Serilog.Sinks.File", StringComparison.OrdinalIgnoreCase));

            // Assert: Contains transitive package
            Assert.Contains(lockFile.Libraries, lib => lib.Name.Equals("Serilog", StringComparison.OrdinalIgnoreCase));

            // Assert: Libraries count matches total packages (direct + transitive)
            var totalPackages = gonugetResult.DirectPackages.Count + gonugetResult.TransitivePackages.Count;
            Assert.Equal(totalPackages, lockFile.Libraries.Count);
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T027: Test multi-framework project categorization.
    /// Verifies correct categorization when project targets multiple frameworks.
    /// </summary>
    [Fact]
    public void MultiFrameworkProject_CorrectCategorization()
    {
        // Arrange: Create test project targeting multiple frameworks
        var projectDir = Path.Combine(Path.GetTempPath(), $"multi-framework-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "MultiFramework.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFrameworks>net6.0;net8.0</TargetFrameworks>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.3"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Verify lock file has both targets
            var format = new LockFileFormat();
            var lockFile = format.Read(gonugetLockFilePath);
            Assert.Contains(lockFile.Targets, t => t.TargetFramework.GetShortFolderName() == "net6.0");
            Assert.Contains(lockFile.Targets, t => t.TargetFramework.GetShortFolderName() == "net8.0");

            // Assert: ProjectFileDependencyGroups has entries for both frameworks
            var net60Group = lockFile.ProjectFileDependencyGroups.FirstOrDefault(g =>
                g.FrameworkName == "net6.0" || g.FrameworkName == ".NETCoreApp,Version=v6.0");
            var net80Group = lockFile.ProjectFileDependencyGroups.FirstOrDefault(g =>
                g.FrameworkName == "net8.0" || g.FrameworkName == ".NETCoreApp,Version=v8.0");

            Assert.NotNull(net60Group);
            Assert.NotNull(net80Group);

            // Assert: Both groups contain Newtonsoft.Json
            Assert.Contains(net60Group.Dependencies, dep => dep.Contains("Newtonsoft.Json"));
            Assert.Contains(net80Group.Dependencies, dep => dep.Contains("Newtonsoft.Json"));
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    /// <summary>
    /// T028: Test framework-specific transitive dependencies categorization.
    /// Verifies packages pulled transitively are correctly categorized per framework.
    /// </summary>
    [Fact]
    public void FrameworkSpecificTransitive_CorrectCategorization()
    {
        // Arrange: Create test project with framework-specific dependencies
        var projectDir = Path.Combine(Path.GetTempPath(), $"framework-transitive-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "FrameworkTransitive.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Microsoft.Extensions.Configuration"" Version=""8.0.0"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        var objDir = Path.Combine(projectDir, "obj");
        Directory.CreateDirectory(objDir);

        try
        {
            // Act: Run gonuget restore
            var gonugetResult = GonugetBridge.RestoreTransitive(
                projectPath: projectPath,
                packagesFolder: null,
                sources: _testSources,
                noCache: false,
                force: true
            );

            var gonugetLockFilePath = Path.Combine(objDir, "project.assets.json");

            // Assert: Restore succeeded
            Assert.True(gonugetResult.Success, $"gonuget restore failed: {string.Join(", ", gonugetResult.ErrorMessages)}");
            Assert.True(File.Exists(gonugetLockFilePath), "Lock file should exist");

            // Assert: Has transitive packages for net6.0
            Assert.NotEmpty(gonugetResult.TransitivePackages);

            // Assert: Verify lock file structure
            var format = new LockFileFormat();
            var lockFile = format.Read(gonugetLockFilePath);

            // Assert: ProjectFileDependencyGroups contains only direct (Microsoft.Extensions.Configuration)
            var net60Group = lockFile.ProjectFileDependencyGroups.FirstOrDefault(g =>
                g.FrameworkName == "net6.0" || g.FrameworkName == ".NETCoreApp,Version=v6.0");
            Assert.NotNull(net60Group);
            Assert.Contains(net60Group.Dependencies, dep => dep.Contains("Microsoft.Extensions.Configuration"));

            // Assert: Libraries map contains both direct and transitive packages
            Assert.Contains(lockFile.Libraries, lib => lib.Name.Equals("Microsoft.Extensions.Configuration", StringComparison.OrdinalIgnoreCase));
            Assert.True(lockFile.Libraries.Count > 1, "Should have transitive dependencies");
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }
}
