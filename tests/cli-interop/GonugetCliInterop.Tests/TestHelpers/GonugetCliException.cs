namespace GonugetCliInterop.Tests.TestHelpers;

/// <summary>
/// Exception thrown when the gonuget CLI interop test bridge returns an error.
/// </summary>
public class GonugetCliException : Exception
{
    /// <summary>
    /// Gets the error code identifying the type of error.
    /// </summary>
    public string Code { get; }

    /// <summary>
    /// Gets additional error details, such as stack traces or command output.
    /// </summary>
    public string? Details { get; }

    /// <summary>
    /// Initializes a new instance of the GonugetCliException class.
    /// </summary>
    /// <param name="code">The error code identifying the type of error.</param>
    /// <param name="message">The error message describing what went wrong.</param>
    /// <param name="details">Optional additional error details.</param>
    public GonugetCliException(string code, string message, string? details = null)
        : base(message)
    {
        Code = code;
        Details = details;
    }
}
