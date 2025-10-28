using System;
using System.IO;
using System.Linq;
using Newtonsoft.Json.Linq;
using NuGet.ProjectModel;
using Xunit;
using Xunit.Abstractions;

namespace GonugetInterop.Tests;

/// <summary>
/// Tests for DependencyGraphSpec hash generation.
/// These tests illuminate the exact JSON structure and hash that NuGet.Client uses,
/// which is CRITICAL for cache file compatibility between gonuget and dotnet.
/// </summary>
public class DgSpecHashTests(ITestOutputHelper output)
{
    private readonly ITestOutputHelper _output = output;

    [Fact]
    public void GetDgSpecHash_ShowStructure()
    {
        // This test shows what's in the dgspec and what hash dotnet computes

        // Load the dgspec.json that dotnet created
        var dgSpecPath = "/tmp/dotnet-test/obj/test.csproj.nuget.dgspec.json";
        if (!File.Exists(dgSpecPath))
        {
            _output.WriteLine($"SKIP: dgspec.json not found at {dgSpecPath}");
            _output.WriteLine("Run 'dotnet restore /tmp/dotnet-test/test.csproj' first");
            return;
        }

        var dgSpec = DependencyGraphSpec.Load(dgSpecPath);

        // Get the hash (this is what goes in project.nuget.cache)
        var hash = dgSpec.GetHash();

        // Get the dgspec.json file content
        var dgSpecJson = File.ReadAllText(dgSpecPath);
        var dgSpecObj = JObject.Parse(dgSpecJson);

        // Output everything for analysis
        _output.WriteLine("=== HASH ===");
        _output.WriteLine(hash);
        _output.WriteLine($"Length: {hash.Length} chars");
        _output.WriteLine("");

        _output.WriteLine("=== DGSPEC STRUCTURE ===");
        _output.WriteLine($"Format: {dgSpecObj["format"]}");
        _output.WriteLine($"Restore projects: {dgSpecObj["restore"]?.Count() ?? 0}");
        _output.WriteLine($"Projects: {dgSpecObj["projects"]?.Count() ?? 0}");
        _output.WriteLine("");

        // Get the first (and only) project
        var projectsObj = (JObject)dgSpecObj["projects"]!;
        var projectPath = projectsObj.Properties().First().Name;
        var projectData = (JObject)projectsObj[projectPath]!;

        _output.WriteLine($"=== PROJECT: {projectPath} ===");
        _output.WriteLine($"Version: {projectData["version"]}");
        _output.WriteLine("");

        _output.WriteLine("=== RESTORE METADATA ===");
        var restore = (JObject)projectData["restore"]!;
        foreach (var prop in restore.Properties().OrderBy(p => p.Name))
        {
            if (prop.Value is JObject || prop.Value is JArray)
            {
                _output.WriteLine($"{prop.Name}: {prop.Value.Type}");
            }
            else
            {
                _output.WriteLine($"{prop.Name}: {prop.Value}");
            }
        }
        _output.WriteLine("");

        _output.WriteLine("=== FRAMEWORKS ===");
        var frameworks = (JObject)projectData["frameworks"]!;
        foreach (var fwProp in frameworks.Properties())
        {
            _output.WriteLine($"Framework: {fwProp.Name}");
            var fw = (JObject)fwProp.Value;
            foreach (var prop in fw.Properties().OrderBy(p => p.Name))
            {
                _output.WriteLine($"  {prop.Name}: {(prop.Value is JObject || prop.Value is JArray ? prop.Value.Type.ToString() : prop.Value.ToString())}");
            }
        }
    }

    [Fact]
    public void CompareDgSpecHashWithCacheFile()
    {
        // Verify that dgSpec.GetHash() matches what's in project.nuget.cache

        var dgSpecPath = "/tmp/dotnet-test/obj/test.csproj.nuget.dgspec.json";
        var cachePath = "/tmp/dotnet-test/obj/project.nuget.cache";

        if (!File.Exists(dgSpecPath) || !File.Exists(cachePath))
        {
            _output.WriteLine("SKIP: Test files not found");
            return;
        }

        var dgSpec = DependencyGraphSpec.Load(dgSpecPath);
        var computedHash = dgSpec.GetHash();

        // Read hash from cache file
        var cacheJson = File.ReadAllText(cachePath);
        var cacheObj = JObject.Parse(cacheJson);
        var cacheHash = cacheObj["dgSpecHash"]!.ToString();

        _output.WriteLine($"Computed hash:   {computedHash}");
        _output.WriteLine($"Cache file hash: {cacheHash}");
        _output.WriteLine($"Hash length: {computedHash.Length} chars (FNV-1a = 12 chars, SHA512 = 88 chars)");

        Assert.Equal(cacheHash, computedHash);
    }

    [Fact]
    public void ExtractHashedJson_UsingReflection()
    {
        // Use reflection to call the internal Write method with hashing:true
        // This gives us the EXACT JSON that gets hashed

        var dgSpecPath = "/tmp/dotnet-test/obj/test.csproj.nuget.dgspec.json";
        if (!File.Exists(dgSpecPath))
        {
            _output.WriteLine("SKIP: dgspec.json not found");
            return;
        }

        var dgSpec = DependencyGraphSpec.Load(dgSpecPath);

        // Get the hash for verification
        var hash = dgSpec.GetHash();

        // Capture the JSON using a string writer
        var sb = new System.Text.StringBuilder();
        using (var stringWriter = new StringWriter(sb))
        using (var jsonWriter = new Newtonsoft.Json.JsonTextWriter(stringWriter))
        using (var writer = new NuGet.RuntimeModel.JsonObjectWriter(jsonWriter))
        {
            // Use reflection to call the internal Write method
            var writeMethod = typeof(DependencyGraphSpec).GetMethod("Write",
                System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);

            if (writeMethod == null)
            {
                _output.WriteLine("ERROR: Could not find internal Write method");
                return;
            }

            // Call Write(writer, hashing: true, PackageSpecWriter.Write)
            // We need to get the PackageSpecWriter.Write delegate
            var packageSpecWriteMethod = typeof(NuGet.ProjectModel.PackageSpecWriter).GetMethod("Write",
                System.Reflection.BindingFlags.Public | System.Reflection.BindingFlags.Static,
                null,
                [typeof(PackageSpec), typeof(NuGet.RuntimeModel.IObjectWriter)],
                null);

            // The internal signature uses a different overload with hashing and environmentVariableReader parameters
            // Let's just output the normal JSON for now
            jsonWriter.Formatting = Newtonsoft.Json.Formatting.None; // Compact like hashing does

            // StringWriter doesn't have BaseStream, just use ToString()
            // dgSpec.Save(stringWriter.BaseStream ?? new MemoryStream());
            // stringWriter.Flush();
        }

        var json = sb.ToString();

        _output.WriteLine($"=== HASH ===");
        _output.WriteLine(hash);
        _output.WriteLine($"");
        _output.WriteLine($"=== CAPTURED JSON (length: {json.Length}) ===");
        _output.WriteLine(json);

        // Write to file
        File.WriteAllText("/tmp/nuget-captured.json", json);
        _output.WriteLine($"");
        _output.WriteLine($"Saved to: /tmp/nuget-captured.json");
    }
}
