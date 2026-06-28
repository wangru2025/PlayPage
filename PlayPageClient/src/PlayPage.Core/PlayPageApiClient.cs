using System;
using System.Collections.Generic;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;

namespace PlayPage.Core;

public sealed class PlayPageApiClient
{
    private readonly HttpClient _http;
    private readonly JsonSerializerOptions _jsonOptions;
    private string? _token;

    public PlayPageApiClient(PlayPageOptions options, HttpClient? httpClient = null)
    {
        _http = httpClient ?? new HttpClient();
        _http.BaseAddress = options.BaseUri;
        _jsonOptions = new JsonSerializerOptions { PropertyNameCaseInsensitive = true };
    }

    public void SetToken(string? token)
    {
        _token = string.IsNullOrWhiteSpace(token) ? null : token;
    }

    public async Task<AuthResult> LoginAsync(string email, string password, CancellationToken cancellationToken = default)
    {
        var result = await SendJsonAsync<AuthResult>(HttpMethod.Post, "api/v1/auth/login", new LoginRequest { Email = email, Password = password }, cancellationToken).ConfigureAwait(false);
        SetToken(result.Token);
        return result;
    }

    public async Task<IReadOnlyList<ProjectSummary>> GetProjectsAsync(CancellationToken cancellationToken = default)
    {
        var envelope = await SendAsync<ProjectListEnvelope>(HttpMethod.Get, "api/v1/projects", cancellationToken).ConfigureAwait(false);
        return envelope.Items;
    }

    private async Task<T> SendJsonAsync<T>(HttpMethod method, string path, object body, CancellationToken cancellationToken)
    {
        using var content = new StringContent(JsonSerializer.Serialize(body, _jsonOptions), Encoding.UTF8, "application/json");
        return await SendAsync<T>(method, path, cancellationToken, content).ConfigureAwait(false);
    }

    private async Task<T> SendAsync<T>(HttpMethod method, string path, CancellationToken cancellationToken, HttpContent? content = null)
    {
        using var request = new HttpRequestMessage(method, path);
        if (content != null) request.Content = content;
        if (!string.IsNullOrWhiteSpace(_token)) request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", _token);

        using var response = await _http.SendAsync(request, cancellationToken).ConfigureAwait(false);
        var text = await response.Content.ReadAsStringAsync().ConfigureAwait(false);
        if (!response.IsSuccessStatusCode)
        {
            var message = TryReadError(text) ?? $"请求失败，状态码：{(int)response.StatusCode}";
            throw new PlayPageApiException(message, response.StatusCode);
        }
        if (string.IsNullOrWhiteSpace(text)) throw new PlayPageApiException("服务器返回了空内容。", response.StatusCode);
        var value = JsonSerializer.Deserialize<T>(text, _jsonOptions);
        if (value == null) throw new PlayPageApiException("服务器返回内容格式不正确。", response.StatusCode);
        return value;
    }

    private string? TryReadError(string text)
    {
        try
        {
            using var doc = JsonDocument.Parse(text);
            if (doc.RootElement.TryGetProperty("error", out var error)) return error.GetString();
            if (doc.RootElement.TryGetProperty("message", out var message)) return message.GetString();
        }
        catch { }
        return null;
    }

    private sealed class ProjectListEnvelope
    {
        public List<ProjectSummary> Items { get; set; } = new List<ProjectSummary>();
    }
}
