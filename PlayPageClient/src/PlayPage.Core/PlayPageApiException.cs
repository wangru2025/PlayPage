using System;
using System.Net;

namespace PlayPage.Core;

public sealed class PlayPageApiException : Exception
{
    public HttpStatusCode? StatusCode { get; }

    public PlayPageApiException(string message, HttpStatusCode? statusCode = null, Exception? innerException = null)
        : base(message, innerException)
    {
        StatusCode = statusCode;
    }
}
