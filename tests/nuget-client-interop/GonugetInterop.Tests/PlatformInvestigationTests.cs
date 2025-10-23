using System;
using System.IO;
using Xunit;
using Xunit.Abstractions;

namespace GonugetInterop.Tests;

/// <summary>
/// Scratch pad for investigating platform-specific behavior and .NET API characteristics.
/// Useful for quick debugging and understanding how NuGet.Client behaves on different platforms.
/// </summary>
public sealed class PlatformInvestigationTests
{
    private readonly ITestOutputHelper _output;

    public PlatformInvestigationTests(ITestOutputHelper output)
    {
        _output = output;
    }

    [Fact]
    public void InvestigatePlatformInvalidFileNameChars()
    {
        var invalid = Path.GetInvalidFileNameChars();

        _output.WriteLine($"Platform: {Environment.OSVersion}");
        _output.WriteLine($"Invalid filename chars count: {invalid.Length}");
        _output.WriteLine("Invalid chars:");

        foreach (var ch in invalid)
        {
            if (ch == '\0')
                _output.WriteLine("  '\\0' (null)");
            else if (ch < 32)
                _output.WriteLine($"  (control char 0x{((int)ch):X2})");
            else
                _output.WriteLine($"  '{ch}'");
        }

        _output.WriteLine($"\nKey character tests:");
        _output.WriteLine($"  Contains ':' ? {Array.IndexOf(invalid, ':') >= 0}");
        _output.WriteLine($"  Contains '/' ? {Array.IndexOf(invalid, '/') >= 0}");
        _output.WriteLine($"  Contains '\\' ? {Array.IndexOf(invalid, '\\') >= 0}");
        _output.WriteLine($"  Contains '|' ? {Array.IndexOf(invalid, '|') >= 0}");
        _output.WriteLine($"  Contains '<' ? {Array.IndexOf(invalid, '<') >= 0}");
        _output.WriteLine($"  Contains '>' ? {Array.IndexOf(invalid, '>') >= 0}");
        _output.WriteLine($"  Contains '*' ? {Array.IndexOf(invalid, '*') >= 0}");
        _output.WriteLine($"  Contains '?' ? {Array.IndexOf(invalid, '?') >= 0}");
        _output.WriteLine($"  Contains '\"' ? {Array.IndexOf(invalid, '\"') >= 0}");
    }
}
