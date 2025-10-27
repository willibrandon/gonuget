using System;
using System.IO;
using System.Linq;
using GonugetInterop.Tests.TestHelpers;
using NuGet.Common;
using NuGet.ProjectModel;
using NuGet.Protocol;
using NuGet.Protocol.Core.Types;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Interop tests for restore package functionality.
/// Validates gonuget restore implementation against NuGet.Client.
/// </summary>
public class RestoreTests
{
    private readonly string _testPackagesFolder;
    private readonly string[] _testSources = ["https://api.nuget.org/v3/index.json"];

    public RestoreTests()
    {
        _testPackagesFolder = Path.Combine(Path.GetTempPath(), "gonuget-test-packages", Guid.NewGuid().ToString());
        Directory.CreateDirectory(_testPackagesFolder);
    }

    [Fact]
    public async System.Threading.Tasks.Task ResolveLatestVersion_StableVersion_MatchesNuGetClient()
    {
        // Arrange
        const string packageId = "Newtonsoft.Json";
        const string source = "https://api.nuget.org/v3/index.json";

        // Act - NuGet.Client (source of truth)
        var sourceRepository = Repository.Factory.GetCoreV3(source);
        var metadataResource = sourceRepository.GetResource<MetadataResource>();
        var nugetVersions = await metadataResource.GetVersions(
            packageId,
            includePrerelease: false,
            includeUnlisted: false,
            sourceCacheContext: new SourceCacheContext(),
            log: NullLogger.Instance,
            token: System.Threading.CancellationToken.None
        );
        var expectedVersion = nugetVersions.Max()?.ToString();

        // Act - gonuget (under test)
        var gonugetResponse = GonugetBridge.ResolveLatestVersion(
            packageId,
            source,
            prerelease: false
        );

        // Assert
        Assert.NotNull(expectedVersion);
        Assert.Equal(expectedVersion, gonugetResponse.Version);
    }

    [Fact]
    public async System.Threading.Tasks.Task ResolveLatestVersion_PrereleaseVersion_MatchesNuGetClient()
    {
        // Arrange
        const string packageId = "Newtonsoft.Json";
        const string source = "https://api.nuget.org/v3/index.json";

        // Act - NuGet.Client (source of truth)
        var sourceRepository = Repository.Factory.GetCoreV3(source);
        var metadataResource = sourceRepository.GetResource<MetadataResource>();
        var nugetVersions = await metadataResource.GetVersions(
            packageId,
            includePrerelease: true,
            includeUnlisted: false,
            sourceCacheContext: new SourceCacheContext(),
            log: NullLogger.Instance,
            token: System.Threading.CancellationToken.None
        );
        var expectedVersion = nugetVersions.Max()?.ToString();

        // Act - gonuget (under test)
        var gonugetResponse = GonugetBridge.ResolveLatestVersion(
            packageId,
            source,
            prerelease: true
        );

        // Assert
        Assert.NotNull(expectedVersion);
        Assert.Equal(expectedVersion, gonugetResponse.Version);
    }

    [Fact]
    public void ParseLockFile_ValidLockFile_ParsesCorrectly()
    {
        // Arrange - Create a test lock file
        var lockFile = new LockFile
        {
            Version = 3
        };

        lockFile.Libraries.Add(new NuGet.ProjectModel.LockFileLibrary
        {
            Name = "Newtonsoft.Json",
            Version = new NuGet.Versioning.NuGetVersion("13.0.3"),
            Type = NuGet.LibraryModel.LibraryType.Package,
            Path = "newtonsoft.json/13.0.3"
        });

        lockFile.PackageFolders.Add(new LockFileItem(_testPackagesFolder));

        lockFile.ProjectFileDependencyGroups.Add(new ProjectFileDependencyGroup(
            "net8.0",
            ["Newtonsoft.Json >= 13.0.3"]
        ));

        var lockFilePath = Path.Combine(Path.GetTempPath(), $"test-{Guid.NewGuid()}.assets.json");
        try
        {
            // Write lock file
            var format = new LockFileFormat();
            format.Write(lockFilePath, lockFile);

            // Act - gonuget (under test)
            var gonugetResponse = GonugetBridge.ParseLockFile(lockFilePath);

            // Assert
            Assert.Equal(3, gonugetResponse.Version);
            Assert.Single(gonugetResponse.Libraries);
            Assert.True(gonugetResponse.Libraries.ContainsKey("Newtonsoft.Json/13.0.3"));
            Assert.Equal("package", gonugetResponse.Libraries["Newtonsoft.Json/13.0.3"].Type);
            Assert.Equal("newtonsoft.json/13.0.3", gonugetResponse.Libraries["Newtonsoft.Json/13.0.3"].Path);
        }
        finally
        {
            if (File.Exists(lockFilePath))
            {
                File.Delete(lockFilePath);
            }
        }
    }

    [Fact]
    public void RestoreDirectDependencies_SimpleProject_CreatesLockFile()
    {
        // Arrange - Create a test project
        var projectDir = Path.Combine(Path.GetTempPath(), $"test-project-{Guid.NewGuid()}");
        Directory.CreateDirectory(projectDir);

        var projectPath = Path.Combine(projectDir, "TestProject.csproj");
        var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.3"" />
  </ItemGroup>
</Project>";

        File.WriteAllText(projectPath, projectContent);

        try
        {
            // Act - gonuget (under test)
            var gonugetResponse = GonugetBridge.RestoreDirectDependencies(
                projectPath,
                _testPackagesFolder,
                _testSources,
                noCache: false,
                force: true
            );

            // Assert
            Assert.True(gonugetResponse.Success);
            Assert.NotEmpty(gonugetResponse.LockFilePath);
            Assert.True(File.Exists(gonugetResponse.LockFilePath), $"Lock file should exist at {gonugetResponse.LockFilePath}");
            Assert.Contains("Newtonsoft.Json/13.0.3", gonugetResponse.InstalledPackages);

            // Verify lock file can be parsed by NuGet.Client
            var format = new LockFileFormat();
            var lockFile = format.Read(gonugetResponse.LockFilePath);
            Assert.Equal(3, lockFile.Version);
            Assert.Contains(lockFile.Libraries, lib => lib.Name == "Newtonsoft.Json" && lib.Version.ToString() == "13.0.3");
        }
        finally
        {
            if (Directory.Exists(projectDir))
            {
                Directory.Delete(projectDir, recursive: true);
            }
        }
    }

    [Fact]
    public void ParseLockFile_EmptyLockFile_ParsesWithoutErrors()
    {
        // Arrange
        var lockFile = new LockFile
        {
            Version = 3
        };

        var lockFilePath = Path.Combine(Path.GetTempPath(), $"empty-{Guid.NewGuid()}.assets.json");
        try
        {
            var format = new LockFileFormat();
            format.Write(lockFilePath, lockFile);

            // Act
            var gonugetResponse = GonugetBridge.ParseLockFile(lockFilePath);

            // Assert
            Assert.Equal(3, gonugetResponse.Version);
            Assert.Empty(gonugetResponse.Libraries);
            Assert.Empty(gonugetResponse.Targets);
        }
        finally
        {
            if (File.Exists(lockFilePath))
            {
                File.Delete(lockFilePath);
            }
        }
    }
}
