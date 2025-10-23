using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using GonugetInterop.Tests.TestHelpers;
using NuGet.Common;
using NuGet.Packaging;
using NuGet.Packaging.Core;
using NuGet.Versioning;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Tests package extraction functionality by comparing gonuget against NuGet.Client.
/// Verifies that V2 (packages.config) and V3 (PackageReference) extraction match exactly.
/// </summary>
public class PackageExtractionTests : IDisposable
{
    private readonly List<string> _tempDirectories = new();

    #region V2 Package Extraction (packages.config layout)

    [Fact]
    public void ExtractPackageV2_MinimalPackage_ExtractsAllFiles()
    {
        // Arrange
        var packageBytes = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: new Dictionary<string, byte[]>
            {
                ["lib/net6.0/test.dll"] = new byte[] { 0x4D, 0x5A }, // MZ header
                ["content/readme.txt"] = "Hello"u8.ToArray()
            }).PackageBytes;

        var installPath = CreateTempDirectory();

        // Act - Extract with gonuget
        var gonugetResult = GonugetBridge.ExtractPackageV2(
            packageBytes,
            installPath,
            packageSaveMode: 6, // Defaultv2: Nupkg | Files (no Nuspec)
            useSideBySideLayout: true);

        // Act - Extract with NuGet.Client
        var nugetInstallPath = CreateTempDirectory();
        var nugetExtracted = ExtractWithNuGetClient(packageBytes, nugetInstallPath, useSideBySide: true);

        // Assert - Both should extract the same files
        Assert.NotEmpty(gonugetResult.ExtractedFiles);

        // Debug: Print what each extracted with full relative paths
        Console.WriteLine($"Gonuget extracted files ({gonugetResult.ExtractedFiles.Length}):");
        foreach (var f in gonugetResult.ExtractedFiles.OrderBy(x => x))
        {
            Console.WriteLine($"  {f}");
        }

        Console.WriteLine($"\nNuGet.Client extracted files ({nugetExtracted.Count}):");
        foreach (var f in nugetExtracted.OrderBy(x => x))
        {
            Console.WriteLine($"  {f}");
        }

        if (nugetExtracted.Count != gonugetResult.FileCount)
        {
            var gonugetFiles = gonugetResult.ExtractedFiles
                .Select(f => Path.GetFileName(f))
                .OrderBy(f => f)
                .ToList();
            var nugetFiles = nugetExtracted
                .Select(f => Path.GetFileName(f))
                .OrderBy(f => f)
                .ToList();

            var message = $"File count mismatch!\n" +
                         $"Gonuget files ({gonugetFiles.Count}): {string.Join(", ", gonugetFiles)}\n" +
                         $"NuGet files ({nugetFiles.Count}): {string.Join(", ", nugetFiles)}";
            Assert.Fail(message);
        }

        Assert.Equal(nugetExtracted.Count, gonugetResult.FileCount);

        // Verify structure matches
        VerifyDirectoryStructureMatches(installPath, nugetInstallPath);
    }

    [Fact]
    public void ExtractPackageV2_OnlyNuspec_ExtractsOnlyNuspec()
    {
        // Arrange
        var packageBytes = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: new Dictionary<string, byte[]>
            {
                ["lib/net6.0/test.dll"] = new byte[] { 0x4D, 0x5A }
            }).PackageBytes;

        var installPath = CreateTempDirectory();

        // Act - Extract only nuspec with gonuget
        var gonugetResult = GonugetBridge.ExtractPackageV2(
            packageBytes,
            installPath,
            packageSaveMode: 1, // Nuspec only
            useSideBySideLayout: true);

        // Assert - Only nuspec file should be extracted
        Assert.Single(gonugetResult.ExtractedFiles);
        Assert.EndsWith(".nuspec", gonugetResult.ExtractedFiles[0]);

        // Verify DLL was NOT extracted
        var packageDir = Directory.GetDirectories(installPath).Single();
        Assert.False(File.Exists(Path.Combine(packageDir, "lib", "net6.0", "test.dll")));
    }

    [Fact]
    public void ExtractPackageV2_WithoutNupkg_DoesNotSaveNupkg()
    {
        // Arrange
        var packageBytes = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0").PackageBytes;

        var installPath = CreateTempDirectory();

        // Act - Extract without nupkg (Nuspec | Files)
        var gonugetResult = GonugetBridge.ExtractPackageV2(
            packageBytes,
            installPath,
            packageSaveMode: 5, // Nuspec | Files (no Nupkg)
            useSideBySideLayout: true);

        // Assert - No .nupkg file should exist
        var packageDir = Directory.GetDirectories(installPath).Single();
        var nupkgFiles = Directory.GetFiles(packageDir, "*.nupkg");
        Assert.Empty(nupkgFiles);

        // But nuspec should exist
        var nuspecFiles = Directory.GetFiles(packageDir, "*.nuspec");
        Assert.Single(nuspecFiles);
    }

    [Fact]
    public void ExtractPackageV2_NonSideBySide_UsesPackageIdOnly()
    {
        // Arrange
        var packageBytes = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0").PackageBytes;

        var installPath = CreateTempDirectory();

        // Act - Extract without side-by-side layout
        var gonugetResult = GonugetBridge.ExtractPackageV2(
            packageBytes,
            installPath,
            packageSaveMode: 7,
            useSideBySideLayout: false); // Package ID only, no version

        // Act - Extract with NuGet.Client using non-side-by-side
        var nugetInstallPath = CreateTempDirectory();
        var nugetExtracted = ExtractWithNuGetClient(packageBytes, nugetInstallPath, useSideBySide: false);

        // Assert - Directory should be just "Test.Package", not "Test.Package.1.0.0"
        var gonugetDirs = Directory.GetDirectories(installPath);
        Assert.Single(gonugetDirs);
        Assert.Equal("Test.Package", Path.GetFileName(gonugetDirs[0]));

        // Verify matches NuGet.Client
        var nugetDirs = Directory.GetDirectories(nugetInstallPath);
        Assert.Equal(Path.GetFileName(nugetDirs[0]), Path.GetFileName(gonugetDirs[0]));
    }

    #endregion

    #region V3 Package Installation (PackageReference layout)

    [Fact]
    public void InstallFromSourceV3_NewPackage_InstallsSuccessfully()
    {
        // Arrange
        var packageBytes = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: new Dictionary<string, byte[]>
            {
                ["lib/net6.0/test.dll"] = new byte[] { 0x4D, 0x5A }
            }).PackageBytes;

        var globalPackages = CreateTempDirectory();

        // Act - Install with gonuget
        var gonugetResult = GonugetBridge.InstallFromSourceV3(
            packageBytes,
            id: "Test.Package",
            version: "1.0.0",
            globalPackagesFolder: globalPackages);

        // Act - Install with NuGet.Client
        var nugetGlobalPackages = CreateTempDirectory();
        var nugetResult = InstallWithNuGetClient(packageBytes, "Test.Package", "1.0.0", nugetGlobalPackages);

        // Assert - Both should indicate successful installation
        Assert.True(gonugetResult.Installed);
        Assert.True(nugetResult.installed);

        // Verify file structure matches
        Assert.True(Directory.Exists(gonugetResult.PackageDirectory));
        Assert.True(File.Exists(gonugetResult.NuspecPath));
        Assert.True(File.Exists(gonugetResult.HashPath));
        Assert.True(File.Exists(gonugetResult.MetadataPath));

        // Verify lowercase normalized paths
        Assert.Contains("test.package", gonugetResult.PackageDirectory.ToLowerInvariant());
        Assert.Contains("1.0.0", gonugetResult.PackageDirectory);

        // Compare directory structures
        VerifyDirectoryStructureMatches(globalPackages, nugetGlobalPackages);
    }

    [Fact]
    public void InstallFromSourceV3_AlreadyInstalled_ReturnsFalse()
    {
        // Arrange
        var packageBytes = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0").PackageBytes;

        var globalPackages = CreateTempDirectory();

        // Act - First installation
        var firstResult = GonugetBridge.InstallFromSourceV3(
            packageBytes,
            id: "Test.Package",
            version: "1.0.0",
            globalPackagesFolder: globalPackages);

        // Act - Second installation (should detect existing)
        var secondResult = GonugetBridge.InstallFromSourceV3(
            packageBytes,
            id: "Test.Package",
            version: "1.0.0",
            globalPackagesFolder: globalPackages);

        // Assert
        Assert.True(firstResult.Installed, "First installation should return true");
        Assert.False(secondResult.Installed, "Second installation should return false (already installed)");

        // Both should point to the same directory
        Assert.Equal(firstResult.PackageDirectory, secondResult.PackageDirectory);
    }

    [Fact]
    public void InstallFromSourceV3_WithoutNupkg_DoesNotSaveNupkg()
    {
        // Arrange
        var packageBytes = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0").PackageBytes;

        var globalPackages = CreateTempDirectory();

        // Act - Install without saving nupkg (Nuspec | Files only)
        var result = GonugetBridge.InstallFromSourceV3(
            packageBytes,
            id: "Test.Package",
            version: "1.0.0",
            globalPackagesFolder: globalPackages,
            packageSaveMode: 5); // Nuspec | Files (no Nupkg)

        // Assert - NupkgPath should be null/empty
        Assert.True(string.IsNullOrEmpty(result.NupkgPath));

        // Verify .nupkg does not exist
        var expectedNupkgPath = Path.Combine(result.PackageDirectory, "test.package.1.0.0.nupkg");
        Assert.False(File.Exists(expectedNupkgPath));

        // But hash and metadata should exist
        Assert.True(File.Exists(result.HashPath));
        Assert.True(File.Exists(result.MetadataPath));
    }

    [Fact]
    public void InstallFromSourceV3_HashFile_MatchesNuGetClient()
    {
        // Arrange
        var packageBytes = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0").PackageBytes;

        var gonugetGlobalPackages = CreateTempDirectory();
        var nugetGlobalPackages = CreateTempDirectory();

        // Act
        var gonugetResult = GonugetBridge.InstallFromSourceV3(
            packageBytes,
            id: "Test.Package",
            version: "1.0.0",
            globalPackagesFolder: gonugetGlobalPackages);

        var nugetResult = InstallWithNuGetClient(packageBytes, "Test.Package", "1.0.0", nugetGlobalPackages);

        // Assert - Hash files should have identical content
        var gonugetHash = File.ReadAllText(gonugetResult.HashPath);
        var nugetHash = File.ReadAllText(nugetResult.hashPath);

        Assert.Equal(nugetHash, gonugetHash);
    }

    #endregion

    #region Helper Methods

    private string CreateTempDirectory()
    {
        var tempDir = Path.Combine(Path.GetTempPath(), "gonuget-test-" + Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(tempDir);
        _tempDirectories.Add(tempDir);
        return tempDir;
    }

    /// <summary>
    /// Extracts a package using NuGet.Client's PackageExtractor.
    /// </summary>
    private List<string> ExtractWithNuGetClient(byte[] packageBytes, string installPath, bool useSideBySide)
    {
        using var packageStream = new MemoryStream(packageBytes);
        var resolver = new PackagePathResolver(installPath, useSideBySide);
        var context = new PackageExtractionContext(
            packageSaveMode: PackageSaveMode.Defaultv2,
            xmlDocFileSaveMode: XmlDocFileSaveMode.None,
            clientPolicyContext: null,
            logger: NullLogger.Instance);

        var extractedFiles = PackageExtractor.ExtractPackageAsync(
            source: "source",
            packageStream: packageStream,
            packagePathResolver: resolver,
            packageExtractionContext: context,
            token: CancellationToken.None).GetAwaiter().GetResult();

        return extractedFiles.ToList();
    }

    /// <summary>
    /// Installs a package using NuGet.Client's InstallFromSourceAsync.
    /// </summary>
    private (bool installed, string hashPath) InstallWithNuGetClient(
        byte[] packageBytes,
        string id,
        string version,
        string globalPackagesFolder)
    {
        var identity = new PackageIdentity(id, NuGetVersion.Parse(version));
        var resolver = new VersionFolderPathResolver(globalPackagesFolder);
        var context = new PackageExtractionContext(
            packageSaveMode: PackageSaveMode.Defaultv3,
            xmlDocFileSaveMode: XmlDocFileSaveMode.None,
            clientPolicyContext: null,
            logger: NullLogger.Instance);

        Func<Stream, Task> copyToAsync = async stream =>
        {
            await stream.WriteAsync(packageBytes, 0, packageBytes.Length);
        };

        var installed = PackageExtractor.InstallFromSourceAsync(
            source: "source",
            packageIdentity: identity,
            copyToAsync: copyToAsync,
            versionFolderPathResolver: resolver,
            packageExtractionContext: context,
            token: CancellationToken.None).GetAwaiter().GetResult();

        var hashPath = resolver.GetHashPath(id, NuGetVersion.Parse(version));
        return (installed, hashPath);
    }

    /// <summary>
    /// Verifies that two directory structures match (same files, same structure).
    /// </summary>
    private void VerifyDirectoryStructureMatches(string dir1, string dir2)
    {
        var files1 = Directory.GetFiles(dir1, "*", SearchOption.AllDirectories)
            .Select(f => f.Substring(dir1.Length).TrimStart(Path.DirectorySeparatorChar))
            .OrderBy(f => f.ToLowerInvariant())
            .ToList();

        var files2 = Directory.GetFiles(dir2, "*", SearchOption.AllDirectories)
            .Select(f => f.Substring(dir2.Length).TrimStart(Path.DirectorySeparatorChar))
            .OrderBy(f => f.ToLowerInvariant())
            .ToList();

        // Compare file lists (case-insensitive since paths may differ in casing)
        Assert.Equal(files1.Count, files2.Count);

        for (int i = 0; i < files1.Count; i++)
        {
            Assert.Equal(
                files1[i].Replace('\\', '/').ToLowerInvariant(),
                files2[i].Replace('\\', '/').ToLowerInvariant());
        }
    }

    public void Dispose()
    {
        foreach (var dir in _tempDirectories)
        {
            try
            {
                if (Directory.Exists(dir))
                    Directory.Delete(dir, recursive: true);
            }
            catch
            {
                // Ignore cleanup errors
            }
        }
    }

    #endregion
}
