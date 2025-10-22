using System;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Exception thrown when gonuget returns an error.
/// </summary>
public class GonugetException : Exception
{
    public string Code { get; }
    public string? Details { get; }

    public GonugetException()
        : base()
    {
        Code = string.Empty;
    }

    public GonugetException(string message)
        : base(message)
    {
        Code = string.Empty;
    }

    public GonugetException(string message, Exception innerException)
        : base(message, innerException)
    {
        Code = string.Empty;
    }

    public GonugetException(string code, string message, string? details = null)
        : base($"[{code}] {message}")
    {
        Code = code;
        Details = details;
    }
}
