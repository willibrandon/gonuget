using System;
using System.Collections.Generic;
using System.IO;
using System.IO.Compression;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using GonugetInterop.Tests.TestHelpers;
using NuGet.Packaging;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Tests package reading functionality by comparing gonuget's package reader
/// against NuGet.Client's PackageArchiveReader.
/// </summary>
public class PackageReaderTests
{
    #region Basic Package Reading

    [Fact]
    public void ReadPackage_MinimalPackage_ReturnsBasicMetadata()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        Assert.Equal("Test.Package", readResult.Id);
        Assert.Equal("1.0.0", readResult.Version);
    }

    [Fact]
    public void ReadPackage_WithDescription_ReturnsDescription()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            description: "Test description");

        var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        Assert.Equal("Test description", readResult.Description);
    }

    [Fact]
    public void ReadPackage_NullDescription_ReturnsEmptyString()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            description: null);

        var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        Assert.NotNull(readResult.Description);
        Assert.Equal("", readResult.Description);
    }

    [Fact]
    public void ReadPackage_WithAuthors_ReturnsAuthors()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            authors: ["Author1", "Author2"]);

        var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        Assert.NotNull(readResult.Authors);
        Assert.Equal(2, readResult.Authors.Length);
        Assert.Contains("Author1", readResult.Authors);
        Assert.Contains("Author2", readResult.Authors);
    }

    [Fact]
    public void ReadPackage_EmptyAuthors_ReturnsEmptyArray()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            authors: Array.Empty<string>());

        var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        Assert.NotNull(readResult.Authors);
        Assert.Empty(readResult.Authors);
    }

    [Fact]
    public void ReadPackage_WithFiles_CountsFiles()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/test.dll"] = [0x4D, 0x5A], // MZ header
            ["content/readme.txt"] = "Hello"u8.ToArray(),
            ["build/test.targets"] = "<Project />"u8.ToArray()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        Assert.True(readResult.FileCount >= 4); // At minimum: nuspec + 3 files
    }

    [Fact]
    public void ReadPackage_NoSignature_ReportsNotSigned()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        Assert.False(readResult.HasSignature);
    }

    [Fact]
    public void ReadPackage_InvalidPackage_ThrowsException()
    {
        var invalidBytes = new byte[] { 0x00, 0x01, 0x02 };

        var ex = Assert.Throws<GonugetException>(() =>
            GonugetBridge.ReadPackage(invalidBytes));

        Assert.NotNull(ex.Message);
    }

    [Fact]
    public void ReadPackage_EmptyBytes_ThrowsException()
    {
        var ex = Assert.Throws<GonugetException>(() =>
            GonugetBridge.ReadPackage(Array.Empty<byte>()));

        Assert.Contains("required", ex.Message, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public void ReadPackage_CorruptedZip_ThrowsException()
    {
        var corruptedZip = new byte[100];
        new Random(42).NextBytes(corruptedZip);

        var ex = Assert.Throws<GonugetException>(() =>
            GonugetBridge.ReadPackage(corruptedZip));

        Assert.NotNull(ex.Message);
    }

    [Fact]
    public void ReadPackage_MissingNuspec_ThrowsException()
    {
        using var ms = new MemoryStream();
        using (var zip = new ZipArchive(ms, ZipArchiveMode.Create, true))
        {
            var entry = zip.CreateEntry("content/file.txt");
            using var entryStream = entry.Open();
            entryStream.Write("test"u8.ToArray());
        }

        var ex = Assert.Throws<GonugetException>(() =>
            GonugetBridge.ReadPackage(ms.ToArray()));

        Assert.NotNull(ex.Message);
    }

    #endregion

    #region Round-Trip Tests (Go write → C# read)

    [Fact]
    public void RoundTrip_MinimalPackage_CSharpReaderCanRead()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var identity = reader.GetIdentity();
        Assert.Equal("Test.Package", identity.Id);
        Assert.Equal("1.0.0", identity.Version.ToString());
    }

    [Fact]
    public void RoundTrip_WithDescription_CSharpReaderGetsDescription()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            description: "Test description");

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var nuspec = reader.NuspecReader;
        Assert.Equal("Test description", nuspec.GetDescription());
    }

    [Fact]
    public void RoundTrip_WithAuthors_CSharpReaderGetsAuthors()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            authors: ["Author1", "Author2"]);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var authors = reader.NuspecReader.GetAuthors();
        Assert.Equal("Author1, Author2", authors);
    }

    [Fact]
    public void RoundTrip_WithFiles_CSharpReaderListsFiles()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/test.dll"] = [0x4D, 0x5A],
            ["content/readme.txt"] = "Hello"u8.ToArray()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var packageFiles = reader.GetFiles().ToList();
        Assert.Contains(packageFiles, f => f.EndsWith("test.dll"));
        Assert.Contains(packageFiles, f => f.EndsWith("readme.txt"));
    }

    [Fact]
    public void RoundTrip_FileContent_CSharpReaderGetsCorrectBytes()
    {
        var expectedContent = "Hello, NuGet!"u8.ToArray();
        var files = new Dictionary<string, byte[]>
        {
            ["content/test.txt"] = expectedContent
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("content/test.txt");
        Assert.NotNull(entry);

        using var entryStream = entry.Open();
        using var ms = new MemoryStream();
        entryStream.CopyTo(ms);

        Assert.Equal(expectedContent, ms.ToArray());
    }

    [Fact]
    public void RoundTrip_MultipleFiles_CSharpReaderGetsAllFiles()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/test.dll"] = [0x4D, 0x5A],
            ["lib/netstandard2.0/test.dll"] = [0x4D, 0x5A],
            ["content/readme.txt"] = "Readme"u8.ToArray(),
            ["build/test.targets"] = "<Project />"u8.ToArray(),
            ["tools/install.ps1"] = "Write-Host 'Installing'"u8.ToArray()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var packageFiles = reader.GetFiles().ToList();
        Assert.True(packageFiles.Count >= 5);
    }

    [Fact]
    public void RoundTrip_EmptyFiles_HandlesCorrectly()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["content/empty.txt"] = Array.Empty<byte>()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("content/empty.txt");
        Assert.NotNull(entry);

        using var entryStream = entry.Open();
        using var ms = new MemoryStream();
        entryStream.CopyTo(ms);

        Assert.Equal(0, ms.Length);
    }

    [Fact]
    public void RoundTrip_LargeFile_CSharpReaderHandles()
    {
        var largeContent = new byte[1024 * 1024]; // 1MB
        new Random(42).NextBytes(largeContent);

        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/large.dll"] = largeContent
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("lib/net6.0/large.dll");
        Assert.NotNull(entry);

        using var entryStream = entry.Open();
        using var ms = new MemoryStream();
        entryStream.CopyTo(ms);

        Assert.Equal(largeContent.Length, ms.Length);
    }

    [Fact]
    public void RoundTrip_SpecialCharactersInFilename_CSharpReaderHandles()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["content/file with spaces.txt"] = "Content"u8.ToArray(),
            ["content/file-with-dashes.txt"] = "Content"u8.ToArray(),
            ["content/file_with_underscores.txt"] = "Content"u8.ToArray()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var packageFiles = reader.GetFiles().ToList();
        Assert.Contains(packageFiles, f => f.Contains("file with spaces.txt"));
        Assert.Contains(packageFiles, f => f.Contains("file-with-dashes.txt"));
        Assert.Contains(packageFiles, f => f.Contains("file_with_underscores.txt"));
    }

    [Fact]
    public void RoundTrip_DeepDirectory_CSharpReaderHandles()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/subfolder/deep/nested/file.dll"] = [0x4D, 0x5A]
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("lib/net6.0/subfolder/deep/nested/file.dll");
        Assert.NotNull(entry);
    }

    [Fact]
    public void RoundTrip_BinaryContent_CSharpReaderGetsExactBytes()
    {
        var binaryContent = new byte[] { 0x00, 0xFF, 0x01, 0xFE, 0x7F, 0x80 };
        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/binary.dat"] = binaryContent
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("lib/net6.0/binary.dat");
        using var entryStream = entry.Open();
        using var ms = new MemoryStream();
        entryStream.CopyTo(ms);

        Assert.Equal(binaryContent, ms.ToArray());
    }

    [Fact]
    public void RoundTrip_GoReader_AgreesWithCSharpReader()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/test.dll"] = [0x4D, 0x5A],
            ["content/readme.txt"] = "Hello"u8.ToArray()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.2.3",
            description: "Test description",
            authors: ["TestAuthor"],
            files: files);

        var gonugetResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var csharpReader = new PackageArchiveReader(stream);

        Assert.Equal("Test.Package", gonugetResult.Id);
        Assert.Equal(csharpReader.GetIdentity().Id, gonugetResult.Id);

        Assert.Equal("1.2.3", gonugetResult.Version);
        Assert.Equal(csharpReader.GetIdentity().Version.ToString(), gonugetResult.Version);

        Assert.Equal("Test description", gonugetResult.Description);
        Assert.Equal(csharpReader.NuspecReader.GetDescription(), gonugetResult.Description);
    }

    [Fact]
    public async Task RoundTrip_NoSignature_BothReadersAgree()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        var gonugetResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var csharpReader = new PackageArchiveReader(stream);
        var isSigned = await csharpReader.IsSignedAsync(CancellationToken.None);

        Assert.False(gonugetResult.HasSignature);
        Assert.False(isSigned);
    }

    #endregion

    #region OPC Compliance

    [Fact]
    public void OPC_ContentTypesXml_Exists()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var zip = new ZipArchive(stream, ZipArchiveMode.Read);

        var contentTypes = zip.GetEntry("[Content_Types].xml");
        Assert.NotNull(contentTypes);
    }

    [Fact]
    public void OPC_RelationshipsFile_Exists()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var zip = new ZipArchive(stream, ZipArchiveMode.Read);

        var rels = zip.GetEntry("_rels/.rels");
        Assert.NotNull(rels);
    }

    [Fact]
    public void OPC_NuspecRelationship_PointsToNuspec()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var nuspecFile = reader.GetNuspecFile();
        Assert.NotNull(nuspecFile);
        Assert.EndsWith(".nuspec", nuspecFile);
    }

    [Fact]
    public void OPC_PackageMetadata_IsValid()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var metadata = reader.NuspecReader.GetMetadata().ToList();
        Assert.NotEmpty(metadata);
    }

    [Fact]
    public void OPC_ZipStructure_IsValid()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        using var stream = new MemoryStream(buildResult.PackageBytes);

        // Should not throw
        using var zip = new ZipArchive(stream, ZipArchiveMode.Read);
        Assert.NotEmpty(zip.Entries);
    }

    [Fact]
    public void OPC_AllEntriesHaveValidPaths()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/test.dll"] = [0x4D, 0x5A],
            ["content/readme.txt"] = "Hello"u8.ToArray()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var zip = new ZipArchive(stream, ZipArchiveMode.Read);

        foreach (var entry in zip.Entries)
        {
            Assert.NotNull(entry.FullName);
            Assert.NotEmpty(entry.FullName);
        }
    }

    [Fact]
    public void OPC_NoBackslashesInPaths()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/test.dll"] = [0x4D, 0x5A]
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var zip = new ZipArchive(stream, ZipArchiveMode.Read);

        foreach (var entry in zip.Entries)
        {
            Assert.DoesNotContain("\\", entry.FullName);
        }
    }

    [Fact]
    public void OPC_EntriesAreCompressed()
    {
        var largeContent = new byte[10000];
        Array.Fill<byte>(largeContent, 0x41); // Fill with 'A'

        var files = new Dictionary<string, byte[]>
        {
            ["content/large.txt"] = largeContent
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        // Compressed package should be smaller than uncompressed content
        Assert.True(buildResult.PackageBytes.Length < largeContent.Length);
    }

    [Fact]
    public void OPC_CanRoundTripThroughZipArchive()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        using var stream1 = new MemoryStream(buildResult.PackageBytes);
        using var zip1 = new ZipArchive(stream1, ZipArchiveMode.Read);

        using var stream2 = new MemoryStream();
        using (var zip2 = new ZipArchive(stream2, ZipArchiveMode.Create, true))
        {
            foreach (var entry in zip1.Entries)
            {
                var newEntry = zip2.CreateEntry(entry.FullName);
                using var sourceStream = entry.Open();
                using var destStream = newEntry.Open();
                sourceStream.CopyTo(destStream);
            }
        }

        stream2.Position = 0;
        var readResult = GonugetBridge.ReadPackage(stream2.ToArray());
        Assert.Equal("Test.Package", readResult.Id);
    }

    #endregion

    #region Signature Detection

    [Fact]
    public async Task Signature_CSharpReader_AgreesOnNoSignature()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);
        Assert.False(readResult.HasSignature);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);
        var isSigned = await reader.IsSignedAsync(CancellationToken.None);

        Assert.False(isSigned);
    }

    [Fact]
    public void Signature_NoSignatureFile_ReportsNotSigned()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var zip = new ZipArchive(stream, ZipArchiveMode.Read);

        var sigFile = zip.GetEntry("package/services/digital-signature/xml-signature/signature.p7s");
        Assert.Null(sigFile);
    }

    [Fact]
    public void Signature_EmptySignatureType_WhenNoSignature()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        Assert.False(readResult.HasSignature);
        Assert.True(string.IsNullOrEmpty(readResult.SignatureType) || readResult.SignatureType == "Unknown");
    }

    [Fact]
    public void Signature_MultiplePackages_AllReportCorrectly()
    {
        for (int i = 0; i < 5; i++)
        {
            var buildResult = GonugetBridge.BuildPackage(
                id: $"Test.Package{i}",
                version: "1.0.0");

            var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);
            Assert.False(readResult.HasSignature);
        }
    }

    [Fact]
    public void Signature_WithFiles_StillReportsNotSigned()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/test.dll"] = [0x4D, 0x5A]
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);
        Assert.False(readResult.HasSignature);
    }

    [Fact]
    public void Signature_LargePackage_StillReportsNotSigned()
    {
        var files = new Dictionary<string, byte[]>();
        for (int i = 0; i < 10; i++)
        {
            files[$"lib/net6.0/assembly{i}.dll"] = new byte[1024];
        }

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        var readResult = GonugetBridge.ReadPackage(buildResult.PackageBytes);
        Assert.False(readResult.HasSignature);
    }

    [Fact]
    public void Signature_ReadTwice_ConsistentResults()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        var read1 = GonugetBridge.ReadPackage(buildResult.PackageBytes);
        var read2 = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        Assert.Equal(read1.HasSignature, read2.HasSignature);
        Assert.Equal(read1.SignatureType, read2.SignatureType);
    }

    [Fact]
    public void Signature_DifferentPackages_IndependentResults()
    {
        var build1 = GonugetBridge.BuildPackage("Package1", "1.0.0");
        var build2 = GonugetBridge.BuildPackage("Package2", "2.0.0");

        var read1 = GonugetBridge.ReadPackage(build1.PackageBytes);
        var read2 = GonugetBridge.ReadPackage(build2.PackageBytes);

        Assert.False(read1.HasSignature);
        Assert.False(read2.HasSignature);
    }

    [Fact]
    public void Signature_AfterRoundTrip_PreservesSignatureState()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        var read1 = GonugetBridge.ReadPackage(buildResult.PackageBytes);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var zip = new ZipArchive(stream, ZipArchiveMode.Read);

        using var stream2 = new MemoryStream();
        using (var zip2 = new ZipArchive(stream2, ZipArchiveMode.Create, true))
        {
            foreach (var entry in zip.Entries)
            {
                var newEntry = zip2.CreateEntry(entry.FullName);
                using var source = entry.Open();
                using var dest = newEntry.Open();
                source.CopyTo(dest);
            }
        }

        var read2 = GonugetBridge.ReadPackage(stream2.ToArray());
        Assert.Equal(read1.HasSignature, read2.HasSignature);
    }

    #endregion

    #region Content Extraction

    [Fact]
    public void Extract_SingleFile_ReturnsCorrectContent()
    {
        var expectedContent = "Test content"u8.ToArray();
        var files = new Dictionary<string, byte[]>
        {
            ["content/test.txt"] = expectedContent
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("content/test.txt");
        using var entryStream = entry.Open();
        using var ms = new MemoryStream();
        entryStream.CopyTo(ms);

        Assert.Equal(expectedContent, ms.ToArray());
    }

    [Fact]
    public void Extract_EmptyFile_ReturnsZeroBytes()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["content/empty.txt"] = Array.Empty<byte>()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("content/empty.txt");
        using var entryStream = entry.Open();
        using var ms = new MemoryStream();
        entryStream.CopyTo(ms);

        Assert.Equal(0, ms.Length);
    }

    [Fact]
    public void Extract_BinaryFile_PreservesBytes()
    {
        var binaryData = new byte[] { 0x00, 0x01, 0xFF, 0xFE, 0x7F, 0x80 };
        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/binary.dll"] = binaryData
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("lib/net6.0/binary.dll");
        using var entryStream = entry.Open();
        using var ms = new MemoryStream();
        entryStream.CopyTo(ms);

        Assert.Equal(binaryData, ms.ToArray());
    }

    [Fact]
    public void Extract_TextFile_PreservesEncoding()
    {
        var utf8Text = "Hello, 世界! 🌍"u8.ToArray();
        var files = new Dictionary<string, byte[]>
        {
            ["content/utf8.txt"] = utf8Text
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("content/utf8.txt");
        using var entryStream = entry.Open();
        using var ms = new MemoryStream();
        entryStream.CopyTo(ms);

        Assert.Equal(utf8Text, ms.ToArray());
    }

    [Fact]
    public void Extract_LargeFile_HandlesCorrectly()
    {
        var largeData = new byte[1024 * 1024]; // 1MB
        new Random(42).NextBytes(largeData);

        var files = new Dictionary<string, byte[]>
        {
            ["lib/net6.0/large.dll"] = largeData
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("lib/net6.0/large.dll");
        using var entryStream = entry.Open();
        using var ms = new MemoryStream();
        entryStream.CopyTo(ms);

        Assert.Equal(largeData.Length, ms.Length);
    }

    [Fact]
    public void Extract_MultipleFiles_AllAccessible()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["file1.txt"] = "Content1"u8.ToArray(),
            ["file2.txt"] = "Content2"u8.ToArray(),
            ["file3.txt"] = "Content3"u8.ToArray()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        foreach (var (path, expectedContent) in files)
        {
            var entry = reader.GetEntry(path);
            using var entryStream = entry.Open();
            using var ms = new MemoryStream();
            entryStream.CopyTo(ms);
            Assert.Equal(expectedContent, ms.ToArray());
        }
    }

    [Fact]
    public void Extract_NestedDirectory_AccessibleByPath()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["a/b/c/d/file.txt"] = "Deep content"u8.ToArray()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("a/b/c/d/file.txt");
        Assert.NotNull(entry);

        using var entryStream = entry.Open();
        using var ms = new MemoryStream();
        entryStream.CopyTo(ms);

        Assert.Equal("Deep content"u8.ToArray(), ms.ToArray());
    }

    [Fact]
    public void Extract_FileWithSpaces_AccessibleByPath()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["content/file with spaces.txt"] = "Spaced content"u8.ToArray()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        var entry = reader.GetEntry("content/file with spaces.txt");
        using var entryStream = entry.Open();
        using var ms = new MemoryStream();
        entryStream.CopyTo(ms);

        Assert.Equal("Spaced content"u8.ToArray(), ms.ToArray());
    }

    [Fact]
    public void Extract_NonExistentFile_ThrowsException()
    {
        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0");

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        // NuGet.Client throws FileNotFoundException for missing files
        Assert.Throws<FileNotFoundException>(() => reader.GetEntry("does/not/exist.txt"));
    }

    [Fact]
    public void Extract_CaseSensitivePath_HandlesCorrectly()
    {
        var files = new Dictionary<string, byte[]>
        {
            ["Content/File.txt"] = "Content"u8.ToArray()
        };

        var buildResult = GonugetBridge.BuildPackage(
            id: "Test.Package",
            version: "1.0.0",
            files: files);

        using var stream = new MemoryStream(buildResult.PackageBytes);
        using var reader = new PackageArchiveReader(stream);

        // Exact match should work
        var entry = reader.GetEntry("Content/File.txt");
        Assert.NotNull(entry);
    }

    #endregion
}
