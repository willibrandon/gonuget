using NuGet.Frameworks;
using Xunit;
using Xunit.Abstractions;

namespace GonugetInterop.Tests;

public class NuGetClientDirectTest
{
    private readonly ITestOutputHelper _output;

    public NuGetClientDirectTest(ITestOutputHelper output)
    {
        _output = output;
    }

    [Fact]
    public void WhatDoesNuGetClientActuallyReturn()
    {
        var fw1 = NuGetFramework.Parse("net6.0-windows10.0.19041.0");
        _output.WriteLine($"net6.0-windows10.0.19041.0 -> {fw1.GetShortFolderName()}");
    }
}
