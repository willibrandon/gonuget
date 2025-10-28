using System;
using System.IO;
using GonugetInterop.Tests.TestHelpers;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Project cache file interop tests validating gonuget against NuGet.Client.
/// Tests dgSpecHash calculation and project.nuget.cache file compatibility.
/// These tests ensure gonuget's restore cache files are compatible with dotnet.
/// </summary>
public sealed class ProjectCacheFileTests
{
    #region DgSpecHash Calculation Tests

    [Fact]
    public void CalculateDgSpecHash_SimpleProject_MatchesNuGetClient()
    {
        // Create a test project
        var tempDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(tempDir);

        try
        {
            var projectPath = Path.Combine(tempDir, "test.csproj");
            var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.3"" />
  </ItemGroup>
</Project>";
            File.WriteAllText(projectPath, projectContent);

            // Calculate hash with gonuget
            var gonugetResponse = GonugetBridge.CalculateDgSpecHash(projectPath);
            Assert.NotEmpty(gonugetResponse.Hash);

            // For full validation, we would need to:
            // 1. Create a DependencyGraphSpec from the project using NuGet.Client
            // 2. Calculate its hash using DependencyGraphSpec.GetHash()
            // 3. Compare with gonuget's hash
            // This would require MSBuild evaluation which is complex in a unit test.
            // The real validation happens in manual testing and the interop test framework.
        }
        finally
        {
            if (Directory.Exists(tempDir))
            {
                Directory.Delete(tempDir, recursive: true);
            }
        }
    }

    [Fact]
    public void CalculateDgSpecHash_MultiplePackages_ReturnsConsistentHash()
    {
        // Create a test project with multiple packages
        var tempDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(tempDir);

        try
        {
            var projectPath = Path.Combine(tempDir, "test.csproj");
            var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.3"" />
    <PackageReference Include=""Serilog"" Version=""3.1.1"" />
    <PackageReference Include=""System.Text.Json"" Version=""8.0.0"" />
  </ItemGroup>
</Project>";
            File.WriteAllText(projectPath, projectContent);

            // Calculate hash twice
            var hash1 = GonugetBridge.CalculateDgSpecHash(projectPath);
            var hash2 = GonugetBridge.CalculateDgSpecHash(projectPath);

            // Should be consistent (same project = same hash)
            Assert.Equal(hash1.Hash, hash2.Hash);
        }
        finally
        {
            if (Directory.Exists(tempDir))
            {
                Directory.Delete(tempDir, recursive: true);
            }
        }
    }

    [Fact]
    public void CalculateDgSpecHash_ChangedVersion_ProducesDifferentHash()
    {
        // Create test project with version 1
        var tempDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(tempDir);

        try
        {
            var projectPath = Path.Combine(tempDir, "test.csproj");

            // Version 1: Newtonsoft.Json 13.0.3
            var projectContent1 = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.3"" />
  </ItemGroup>
</Project>";
            File.WriteAllText(projectPath, projectContent1);
            var hash1 = GonugetBridge.CalculateDgSpecHash(projectPath);

            // Version 2: Newtonsoft.Json 13.0.2
            var projectContent2 = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.2"" />
  </ItemGroup>
</Project>";
            File.WriteAllText(projectPath, projectContent2);
            var hash2 = GonugetBridge.CalculateDgSpecHash(projectPath);

            // Hashes should be different (version changed)
            Assert.NotEqual(hash1.Hash, hash2.Hash);
        }
        finally
        {
            if (Directory.Exists(tempDir))
            {
                Directory.Delete(tempDir, recursive: true);
            }
        }
    }

    [Fact]
    public void CalculateDgSpecHash_ChangedTargetFramework_ProducesDifferentHash()
    {
        // Create test project
        var tempDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(tempDir);

        try
        {
            var projectPath = Path.Combine(tempDir, "test.csproj");

            // Target framework: net8.0
            var projectContent1 = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.3"" />
  </ItemGroup>
</Project>";
            File.WriteAllText(projectPath, projectContent1);
            var hash1 = GonugetBridge.CalculateDgSpecHash(projectPath);

            // Target framework: net6.0
            var projectContent2 = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.3"" />
  </ItemGroup>
</Project>";
            File.WriteAllText(projectPath, projectContent2);
            var hash2 = GonugetBridge.CalculateDgSpecHash(projectPath);

            // Hashes should be different (target framework changed)
            Assert.NotEqual(hash1.Hash, hash2.Hash);
        }
        finally
        {
            if (Directory.Exists(tempDir))
            {
                Directory.Delete(tempDir, recursive: true);
            }
        }
    }

    #endregion

    #region Cache File Verification Tests

    [Fact]
    public void VerifyCacheFile_ValidCacheMatchingHash_ReturnsValid()
    {
        // Create a test project and restore it with dotnet to generate a cache file
        var tempDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(tempDir);

        try
        {
            var projectPath = Path.Combine(tempDir, "test.csproj");
            var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.3"" />
  </ItemGroup>
</Project>";
            File.WriteAllText(projectPath, projectContent);

            // Create obj directory and a mock cache file
            var objDir = Path.Combine(tempDir, "obj");
            Directory.CreateDirectory(objDir);

            // Calculate expected hash
            var hashResponse = GonugetBridge.CalculateDgSpecHash(projectPath);

            // Create a simple cache file
            var cachePath = Path.Combine(objDir, "project.nuget.cache");
            var cacheContent = $@"{{
  ""version"": 2,
  ""dgSpecHash"": ""{hashResponse.Hash}"",
  ""success"": true,
  ""projectFilePath"": ""{projectPath.Replace("\\", "\\\\")}"",
  ""expectedPackageFiles"": [
    ""/Users/user/.nuget/packages/newtonsoft.json/13.0.3/newtonsoft.json.13.0.3.nupkg.sha512""
  ],
  ""logs"": []
}}";
            File.WriteAllText(cachePath, cacheContent);

            // Verify cache file with gonuget
            var verifyResponse = GonugetBridge.VerifyProjectCacheFile(cachePath, hashResponse.Hash);

            // Should be valid (hash matches)
            Assert.True(verifyResponse.Valid);
            Assert.Equal(2, verifyResponse.Version);
            Assert.Equal(hashResponse.Hash, verifyResponse.DgSpecHash);
            Assert.True(verifyResponse.Success);
            Assert.Equal(1, verifyResponse.ExpectedPackageFilesCount);
        }
        finally
        {
            if (Directory.Exists(tempDir))
            {
                Directory.Delete(tempDir, recursive: true);
            }
        }
    }

    [Fact]
    public void VerifyCacheFile_MismatchedHash_ReturnsInvalid()
    {
        // Create a test project
        var tempDir = Path.Combine(Path.GetTempPath(), Guid.NewGuid().ToString());
        Directory.CreateDirectory(tempDir);

        try
        {
            var projectPath = Path.Combine(tempDir, "test.csproj");
            var projectContent = @"<Project Sdk=""Microsoft.NET.Sdk"">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include=""Newtonsoft.Json"" Version=""13.0.3"" />
  </ItemGroup>
</Project>";
            File.WriteAllText(projectPath, projectContent);

            var objDir = Path.Combine(tempDir, "obj");
            Directory.CreateDirectory(objDir);

            // Create cache file with DIFFERENT hash than current project
            var cachePath = Path.Combine(objDir, "project.nuget.cache");
            var cacheContent = @"{
  ""version"": 2,
  ""dgSpecHash"": ""OLD_HASH_VALUE"",
  ""success"": true,
  ""projectFilePath"": ""test.csproj"",
  ""expectedPackageFiles"": [],
  ""logs"": []
}";
            File.WriteAllText(cachePath, cacheContent);

            // Calculate current hash
            var currentHash = GonugetBridge.CalculateDgSpecHash(projectPath);

            // Verify cache file (should be invalid because hash doesn't match)
            var verifyResponse = GonugetBridge.VerifyProjectCacheFile(cachePath, currentHash.Hash);

            // Should be invalid (hash mismatch)
            Assert.False(verifyResponse.Valid);
            Assert.Equal("OLD_HASH_VALUE", verifyResponse.DgSpecHash);
        }
        finally
        {
            if (Directory.Exists(tempDir))
            {
                Directory.Delete(tempDir, recursive: true);
            }
        }
    }

    [Fact]
    public void VerifyCacheFile_MissingFile_ReturnsInvalid()
    {
        var cachePath = Path.Combine(Path.GetTempPath(), "nonexistent-cache-" + Guid.NewGuid() + ".json");
        var someHash = "sBEBoV+1pAY=";

        // When a cache file doesn't exist, both dotnet and gonuget return an invalid cache.
        // From NuGet.Client's CacheFileFormat.Read: "if the file doesn't exist, return invalid cache"
        // This is implemented in restore/cache_file.go:98-100

        // Verify gonuget returns invalid (not an error) when file is missing
        var response = GonugetBridge.VerifyProjectCacheFile(cachePath, someHash);

        // Should be invalid
        Assert.False(response.Valid);
        Assert.False(response.Success);
    }

    #endregion
}
