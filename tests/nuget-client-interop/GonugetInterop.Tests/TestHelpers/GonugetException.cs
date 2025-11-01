using System;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Exception thrown when gonuget returns an error.
/// Includes comprehensive diagnostic information to help debug test failures.
/// </summary>
public class GonugetException : Exception
{
    /// <summary>
    /// Gets the error code returned by gonuget (e.g., "NU1101", "RESOLVER_ERROR").
    /// </summary>
    public string Code { get; }

    /// <summary>
    /// Gets additional error details if provided by gonuget.
    /// </summary>
    public string? Details { get; }

    /// <summary>
    /// Gets the action that was being executed when the error occurred.
    /// </summary>
    public string? Action { get; }

    /// <summary>
    /// Gets the JSON request that was sent to gonuget-interop-test.
    /// </summary>
    public string? RequestJson { get; }

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

    /// <summary>
    /// Creates a new GonugetException with comprehensive diagnostic context.
    /// </summary>
    /// <param name="code">Error code from gonuget</param>
    /// <param name="message">Error message from gonuget</param>
    /// <param name="details">Additional error details</param>
    /// <param name="action">The action that was being executed</param>
    /// <param name="requestJson">The JSON request that was sent</param>
    public GonugetException(string code, string message, string? details, string action, string requestJson)
        : base(BuildDetailedMessage(code, message, details, action, requestJson))
    {
        Code = code;
        Details = details;
        Action = action;
        RequestJson = requestJson;
    }

    private static string BuildDetailedMessage(string code, string message, string? details, string action, string requestJson)
    {
        var msg = $"[{code}] {message}\n";

        if (!string.IsNullOrEmpty(details))
        {
            msg += $"Details: {details}\n";
        }

        msg += $"Action: {action}\n";
        msg += $"Request: {requestJson}";

        return msg;
    }
}
