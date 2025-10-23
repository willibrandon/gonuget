using GonugetInterop.Tests.TestHelpers;
using NuGet.Frameworks;
using Xunit;

namespace GonugetInterop.Tests;

/// <summary>
/// Tests gonuget's framework formatting (GetShortFolderName) against NuGet.Client.
/// These tests validate that gonuget produces identical folder names for all framework types.
/// </summary>
public class FrameworkFormattingTests
{
    /// <summary>
    /// Tests .NET Framework compact version formatting (2-4 digits).
    /// NuGet uses compact notation: net45, net472, net4721 (without dots).
    /// </summary>
    [Theory]
    [InlineData("net45", "net45")]
    [InlineData("net40", "net40")]
    [InlineData("net35", "net35")]
    [InlineData("net20", "net20")]
    [InlineData("net11", "net11")]
    [InlineData("net472", "net472")]
    [InlineData("net471", "net471")]
    [InlineData("net47", "net47")]
    [InlineData("net463", "net463")]
    [InlineData("net462", "net462")]
    [InlineData("net461", "net461")]
    [InlineData("net46", "net46")]
    [InlineData("net452", "net452")]
    [InlineData("net451", "net451")]
    [InlineData("net403", "net403")]
    [InlineData("net48", "net48")]
    [InlineData("net481", "net481")]
    public void FormatFramework_NetFramework_ShouldMatchNuGetClient(string input, string expected)
    {
        // NuGet.Client (source of truth)
        var nugetFramework = NuGetFramework.Parse(input);
        var nugetFormatted = nugetFramework.GetShortFolderName();
        Assert.Equal(expected, nugetFormatted);

        // Gonuget (under test)
        var gonugetResult = GonugetBridge.FormatFramework(input);
        Assert.Equal(expected, gonugetResult.ShortFolderName);
        Assert.Equal(nugetFormatted, gonugetResult.ShortFolderName);
    }

    /// <summary>
    /// Tests .NET 5+ formatting with "net" prefix (not "netcoreapp").
    /// Critical: .NET 5+ uses dotted version (net6.0), NOT compact (net60).
    /// </summary>
    [Theory]
    [InlineData("net5.0", "net5.0")]
    [InlineData("net6.0", "net6.0")]
    [InlineData("net7.0", "net7.0")]
    [InlineData("net8.0", "net8.0")]
    [InlineData("net9.0", "net9.0")]
    [InlineData("net10.0", "net10.0")]
    public void FormatFramework_Net5Plus_ShouldMatchNuGetClient(string input, string expected)
    {
        // NuGet.Client
        var nugetFramework = NuGetFramework.Parse(input);
        var nugetFormatted = nugetFramework.GetShortFolderName();
        Assert.Equal(expected, nugetFormatted);

        // Gonuget
        var gonugetResult = GonugetBridge.FormatFramework(input);
        Assert.Equal(expected, gonugetResult.ShortFolderName);
        Assert.Equal(nugetFormatted, gonugetResult.ShortFolderName);
    }

    /// <summary>
    /// Tests .NET Standard formatting.
    /// </summary>
    [Theory]
    [InlineData("netstandard1.0", "netstandard1.0")]
    [InlineData("netstandard1.1", "netstandard1.1")]
    [InlineData("netstandard1.2", "netstandard1.2")]
    [InlineData("netstandard1.3", "netstandard1.3")]
    [InlineData("netstandard1.4", "netstandard1.4")]
    [InlineData("netstandard1.5", "netstandard1.5")]
    [InlineData("netstandard1.6", "netstandard1.6")]
    [InlineData("netstandard2.0", "netstandard2.0")]
    [InlineData("netstandard2.1", "netstandard2.1")]
    public void FormatFramework_NetStandard_ShouldMatchNuGetClient(string input, string expected)
    {
        // NuGet.Client
        var nugetFramework = NuGetFramework.Parse(input);
        var nugetFormatted = nugetFramework.GetShortFolderName();
        Assert.Equal(expected, nugetFormatted);

        // Gonuget
        var gonugetResult = GonugetBridge.FormatFramework(input);
        Assert.Equal(expected, gonugetResult.ShortFolderName);
        Assert.Equal(nugetFormatted, gonugetResult.ShortFolderName);
    }

    /// <summary>
    /// Tests .NET Core App formatting.
    /// Note: .NET Core 1.x-3.x uses "netcoreapp", but .NET 5+ uses "net".
    /// </summary>
    [Theory]
    [InlineData("netcoreapp1.0", "netcoreapp1.0")]
    [InlineData("netcoreapp1.1", "netcoreapp1.1")]
    [InlineData("netcoreapp2.0", "netcoreapp2.0")]
    [InlineData("netcoreapp2.1", "netcoreapp2.1")]
    [InlineData("netcoreapp2.2", "netcoreapp2.2")]
    [InlineData("netcoreapp3.0", "netcoreapp3.0")]
    [InlineData("netcoreapp3.1", "netcoreapp3.1")]
    public void FormatFramework_NetCoreApp_ShouldMatchNuGetClient(string input, string expected)
    {
        // NuGet.Client
        var nugetFramework = NuGetFramework.Parse(input);
        var nugetFormatted = nugetFramework.GetShortFolderName();
        Assert.Equal(expected, nugetFormatted);

        // Gonuget
        var gonugetResult = GonugetBridge.FormatFramework(input);
        Assert.Equal(expected, gonugetResult.ShortFolderName);
        Assert.Equal(nugetFormatted, gonugetResult.ShortFolderName);
    }

    /// <summary>
    /// Tests platform-specific .NET 5+ formatting (windows, android, ios, etc.).
    /// Format: net{version}-{platform}[{platformVersion}]
    /// </summary>
    [Theory]
    [InlineData("net6.0-windows", "net6.0-windows")]
    [InlineData("net6.0-windows7.0", "net6.0-windows7.0")]
    [InlineData("net6.0-windows10.0", "net6.0-windows10.0")]
    [InlineData("net6.0-windows10.0.19041.0", "net6.0-windows10.0.19041")] // NuGet.Client drops trailing .0 on revision
    [InlineData("net6.0-android", "net6.0-android")]
    [InlineData("net6.0-android31.0", "net6.0-android31.0")]
    [InlineData("net6.0-ios", "net6.0-ios")]
    [InlineData("net6.0-ios15.0", "net6.0-ios15.0")]
    [InlineData("net7.0-windows", "net7.0-windows")]
    [InlineData("net8.0-windows", "net8.0-windows")]
    public void FormatFramework_PlatformSpecific_ShouldMatchNuGetClient(string input, string expected)
    {
        // NuGet.Client
        var nugetFramework = NuGetFramework.Parse(input);
        var nugetFormatted = nugetFramework.GetShortFolderName();
        Assert.Equal(expected, nugetFormatted);

        // Gonuget
        var gonugetResult = GonugetBridge.FormatFramework(input);
        Assert.Equal(expected, gonugetResult.ShortFolderName);
        Assert.Equal(nugetFormatted, gonugetResult.ShortFolderName);
    }

    /// <summary>
    /// Tests profile-based frameworks (.NET Framework with profiles).
    /// Common profiles: Client, Full, CompactFramework
    /// </summary>
    [Theory]
    [InlineData("net40-client", "net40-client")]
    [InlineData("net40-Client", "net40-client")]  // Case insensitive
    [InlineData("net45-cf", "net45-cf")]  // Compact Framework
    [InlineData("net45-CF", "net45-cf")]
    public void FormatFramework_Profiles_ShouldMatchNuGetClient(string input, string expected)
    {
        // NuGet.Client
        var nugetFramework = NuGetFramework.Parse(input);
        var nugetFormatted = nugetFramework.GetShortFolderName();
        Assert.Equal(expected, nugetFormatted);

        // Gonuget
        var gonugetResult = GonugetBridge.FormatFramework(input);
        Assert.Equal(expected, gonugetResult.ShortFolderName);
        Assert.Equal(nugetFormatted, gonugetResult.ShortFolderName);
    }

    /// <summary>
    /// Tests Portable Class Library (PCL) formatting with profile numbers.
    /// NuGet.Client expands profile numbers to their framework lists.
    /// </summary>
    [Theory]
    [InlineData("portable-Profile7", "portable-net45+win8")]
    [InlineData("portable-Profile31", "portable-win81+wp81")]
    [InlineData("portable-Profile32", "portable-win81+wpa81")]
    [InlineData("portable-Profile44", "portable-net451+win81")]
    [InlineData("portable-Profile49", "portable-net45+wp8")]
    [InlineData("portable-Profile78", "portable-net45+win8+wp8")]
    [InlineData("portable-Profile84", "portable-wp81+wpa81")]
    [InlineData("portable-Profile111", "portable-net45+win8+wpa81")]
    [InlineData("portable-Profile151", "portable-net451+win81+wpa81")]
    [InlineData("portable-Profile157", "portable-win81+wp81+wpa81")]
    [InlineData("portable-Profile259", "portable-net45+win8+wp8+wpa81")]
    public void FormatFramework_PCL_ProfileNumbers_ShouldMatchNuGetClient(string input, string expected)
    {
        // NuGet.Client
        var nugetFramework = NuGetFramework.Parse(input);
        var nugetFormatted = nugetFramework.GetShortFolderName();
        Assert.Equal(expected, nugetFormatted);

        // Gonuget
        var gonugetResult = GonugetBridge.FormatFramework(input);
        Assert.Equal(expected, gonugetResult.ShortFolderName);
        Assert.Equal(nugetFormatted, gonugetResult.ShortFolderName);
    }

    /// <summary>
    /// Tests PCL formatting with framework lists (resolved to profile numbers or alphabetically sorted).
    /// CRITICAL: Frameworks are sorted alphabetically, regardless of input order!
    /// </summary>
    [Theory]
    [InlineData("portable-net45+win8", "portable-net45+win8")]  // Profile7
    [InlineData("portable-win8+net45", "portable-net45+win8")]  // Same as above, but sorted!
    [InlineData("portable-net45+win8+wp8+wpa81", "portable-net45+win8+wp8+wpa81")]  // Profile259
    [InlineData("portable-wpa81+wp8+win8+net45", "portable-net45+win8+wp8+wpa81")]  // Sorted
    public void FormatFramework_PCL_FrameworkLists_ShouldMatchNuGetClient(string input, string expected)
    {
        // NuGet.Client
        var nugetFramework = NuGetFramework.Parse(input);
        var nugetFormatted = nugetFramework.GetShortFolderName();
        Assert.Equal(expected, nugetFormatted);

        // Gonuget
        var gonugetResult = GonugetBridge.FormatFramework(input);
        Assert.Equal(expected, gonugetResult.ShortFolderName);
        Assert.Equal(nugetFormatted, gonugetResult.ShortFolderName);
    }

    /// <summary>
    /// Tests legacy PCL framework formatting (Windows, WindowsPhone, etc.).
    /// These are standalone TFMs that can appear in PCL profiles.
    /// </summary>
    [Theory]
    [InlineData("win8", "win8")]
    [InlineData("win81", "win81")]
    [InlineData("wp8", "wp8")]
    [InlineData("wp81", "wp81")]
    [InlineData("wpa81", "wpa81")]
    [InlineData("sl5", "sl5")]
    public void FormatFramework_LegacyPCL_ShouldMatchNuGetClient(string input, string expected)
    {
        // NuGet.Client
        var nugetFramework = NuGetFramework.Parse(input);
        var nugetFormatted = nugetFramework.GetShortFolderName();
        Assert.Equal(expected, nugetFormatted);

        // Gonuget
        var gonugetResult = GonugetBridge.FormatFramework(input);
        Assert.Equal(expected, gonugetResult.ShortFolderName);
        Assert.Equal(nugetFormatted, gonugetResult.ShortFolderName);
    }

    /// <summary>
    /// Tests special framework identifiers (Any, Unsupported, Agnostic).
    /// </summary>
    [Theory]
    [InlineData("any", "any")]
    [InlineData("unsupported", "unsupported")]
    [InlineData("agnostic", "agnostic")]
    public void FormatFramework_Special_ShouldMatchNuGetClient(string input, string expected)
    {
        // NuGet.Client
        var nugetFramework = NuGetFramework.Parse(input);
        var nugetFormatted = nugetFramework.GetShortFolderName();
        Assert.Equal(expected, nugetFormatted);

        // Gonuget
        var gonugetResult = GonugetBridge.FormatFramework(input);
        Assert.Equal(expected, gonugetResult.ShortFolderName);
        Assert.Equal(nugetFormatted, gonugetResult.ShortFolderName);
    }

    /// <summary>
    /// Tests round-trip: parse → format → parse should produce same result.
    /// </summary>
    [Theory]
    [InlineData("net8.0")]
    [InlineData("net48")]
    [InlineData("netstandard2.1")]
    [InlineData("netcoreapp3.1")]
    [InlineData("net6.0-windows10.0")]
    [InlineData("portable-net45+win8")]
    public void FormatFramework_RoundTrip_ShouldBeIdempotent(string input)
    {
        // First format
        var formatted1 = GonugetBridge.FormatFramework(input);

        // Parse the formatted result and format again
        var formatted2 = GonugetBridge.FormatFramework(formatted1.ShortFolderName);

        // Should be identical
        Assert.Equal(formatted1.ShortFolderName, formatted2.ShortFolderName);
    }
}
