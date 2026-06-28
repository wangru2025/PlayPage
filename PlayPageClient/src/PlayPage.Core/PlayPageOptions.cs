using System;

namespace PlayPage.Core;

public sealed class PlayPageOptions
{
    public Uri BaseUri { get; }

    public PlayPageOptions(string baseUrl)
    {
        if (string.IsNullOrWhiteSpace(baseUrl)) throw new ArgumentException("API 地址不能为空。", nameof(baseUrl));
        BaseUri = new Uri(baseUrl.TrimEnd('/') + "/", UriKind.Absolute);
    }

    public static PlayPageOptions Production { get; } = new PlayPageOptions("https://web.wangru.net");
}
