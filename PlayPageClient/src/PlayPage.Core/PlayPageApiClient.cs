using System;
using System.Collections.Generic;
using System.IO;
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

    public string? AccessToken => _token;

    public PlayPageApiClient(PlayPageOptions options, HttpClient? httpClient = null)
    {
        _http = httpClient ?? new HttpClient();
        _http.BaseAddress = options.BaseUri;
        _jsonOptions = new JsonSerializerOptions { PropertyNameCaseInsensitive = true };
    }

    public void SetToken(string? token) => _token = string.IsNullOrWhiteSpace(token) ? null : token;

    public Task<AuthCodeResponse> RequestCodeAsync(string email, string username = "", CancellationToken cancellationToken = default) =>
        SendJsonAsync<AuthCodeResponse>(HttpMethod.Post, "api/v1/auth/request-code", new AuthCodeRequest { Email = email, Username = username }, cancellationToken);

    public async Task<AuthResult> VerifyCodeAsync(string email, string code, CancellationToken cancellationToken = default)
    {
        var result = await SendJsonAsync<AuthResult>(HttpMethod.Post, "api/v1/auth/verify-code", new AuthCodeVerifyRequest { Email = email, Code = code }, cancellationToken).ConfigureAwait(false);
        SetToken(result.AccessToken);
        return result;
    }

    public Task<UserProfile> GetMeAsync(CancellationToken cancellationToken = default) => SendAsync<UserProfile>(HttpMethod.Get, "api/v1/me", cancellationToken);
    public Task<UserProfile> UpdateProfileAsync(string username, CancellationToken cancellationToken = default) => SendJsonAsync<UserProfile>(HttpMethod.Post, "api/v1/me/profile", new UserProfileUpdateRequest { Username = username }, cancellationToken);
    public Task<StatusEnvelope> LogoutAsync(CancellationToken cancellationToken = default) => SendJsonAsync<StatusEnvelope>(HttpMethod.Post, "api/v1/auth/logout", new { }, cancellationToken);

    public Task<IReadOnlyList<ProjectSummary>> GetProjectsAsync(CancellationToken cancellationToken = default) => GetItemsAsync<ProjectSummary>("api/v1/projects", cancellationToken);
    public Task<ProjectSummary> CreateProjectAsync(ProjectCreateRequest request, CancellationToken cancellationToken = default) => SendJsonAsync<ProjectSummary>(HttpMethod.Post, "api/v1/projects", request, cancellationToken);
    public Task<ProjectDetails> GetProjectAsync(string projectId, CancellationToken cancellationToken = default) => SendAsync<ProjectDetails>(HttpMethod.Get, $"api/v1/projects/{Uri.EscapeDataString(projectId)}", cancellationToken);
    public Task<ProjectSummary> UpdateProjectSettingsAsync(string projectId, ProjectSettingsRequest request, CancellationToken cancellationToken = default) => SendJsonAsync<ProjectSummary>(HttpMethod.Post, ProjectPath(projectId, "settings"), request, cancellationToken);
    public Task<ProjectSummary> UpdateProjectVisibilityAsync(string projectId, string visibility, CancellationToken cancellationToken = default) => SendJsonAsync<ProjectSummary>(HttpMethod.Post, ProjectPath(projectId, "visibility"), new ProjectVisibilityRequest { Visibility = visibility }, cancellationToken);
    public Task<StatusEnvelope> DeleteProjectAsync(string projectId, CancellationToken cancellationToken = default) => SendAsync<StatusEnvelope>(HttpMethod.Delete, $"api/v1/projects/{Uri.EscapeDataString(projectId)}", cancellationToken);

    public Task<IReadOnlyList<ReleaseInfo>> ListReleasesAsync(string projectId, CancellationToken cancellationToken = default) => GetItemsAsync<ReleaseInfo>(ProjectPath(projectId, "releases"), cancellationToken);
    public Task<ReleaseInfo> RollbackReleaseAsync(string projectId, string releaseId, CancellationToken cancellationToken = default) => SendJsonAsync<ReleaseInfo>(HttpMethod.Post, ProjectPath(projectId, $"releases/{Uri.EscapeDataString(releaseId)}/rollback"), new { }, cancellationToken);
    public Task<ReleaseInfo> UploadReleaseFileAsync(string projectId, string filePath, string mode, string changeNote = "", CancellationToken cancellationToken = default)
    {
        var content = new MultipartFormDataContent();
        content.Add(new StreamContent(File.OpenRead(filePath)), "file", Path.GetFileName(filePath));
        if (!string.IsNullOrWhiteSpace(changeNote)) content.Add(new StringContent(changeNote, Encoding.UTF8), "changeNote");
        return SendAsync<ReleaseInfo>(HttpMethod.Post, ProjectPath(projectId, "releases") + "?mode=" + Uri.EscapeDataString(mode), cancellationToken, content);
    }
    public Task<ReleaseInfo> UploadReleaseHtmlTextAsync(string projectId, string html, string changeNote = "", CancellationToken cancellationToken = default) =>
        SendJsonAsync<ReleaseInfo>(HttpMethod.Post, ProjectPath(projectId, "releases") + "?mode=text", new { html, changeNote }, cancellationToken);
    public Task<ReleaseInfo> CreateReleaseFromTemplateAsync(string projectId, TemplateCreateReleaseRequest request, CancellationToken cancellationToken = default) =>
        SendJsonAsync<ReleaseInfo>(HttpMethod.Post, ProjectPath(projectId, "releases") + "?mode=template", request, cancellationToken);

    public Task<string> GetInteractiveDocAsync(string projectId, CancellationToken cancellationToken = default) => SendTextAsync(HttpMethod.Get, ProjectPath(projectId, "interactive-doc"), cancellationToken);
    public Task<DownloadedFile> DownloadProjectSourceAsync(string projectId, CancellationToken cancellationToken = default) => SendFileAsync(HttpMethod.Get, ProjectPath(projectId, "source"), cancellationToken);
    public Task<DownloadedFile> ExportProjectDataAsync(string projectId, IReadOnlyList<string> collections, string format, CancellationToken cancellationToken = default) =>
        SendJsonFileAsync(HttpMethod.Post, ProjectPath(projectId, "data-export"), new DataExportRequest { Collections = new List<string>(collections), Format = format }, cancellationToken);
    public Task<ProjectStatsSummary> GetProjectStatsAsync(string projectId, string from = "", string to = "", CancellationToken cancellationToken = default)
    {
        var query = new List<string>();
        if (!string.IsNullOrWhiteSpace(from)) query.Add("from=" + Uri.EscapeDataString(from));
        if (!string.IsNullOrWhiteSpace(to)) query.Add("to=" + Uri.EscapeDataString(to));
        return SendAsync<ProjectStatsSummary>(HttpMethod.Get, ProjectPath(projectId, "stats") + (query.Count == 0 ? "" : "?" + string.Join("&", query)), cancellationToken);
    }

    public Task<IReadOnlyList<CollectionInfo>> ListCollectionsAsync(string projectId, CancellationToken cancellationToken = default) => GetItemsAsync<CollectionInfo>(ProjectPath(projectId, "collections"), cancellationToken);
    public Task<CollectionInfo> CreateCollectionAsync(string projectId, CollectionCreateRequest request, CancellationToken cancellationToken = default) => SendJsonAsync<CollectionInfo>(HttpMethod.Post, ProjectPath(projectId, "collections"), request, cancellationToken);
    public Task<CollectionInfo> UpdateCollectionAsync(string projectId, string collectionName, CollectionUpdateRequest request, CancellationToken cancellationToken = default) => SendJsonAsync<CollectionInfo>(HttpMethod.Patch, ProjectPath(projectId, $"collections/{Uri.EscapeDataString(collectionName)}"), request, cancellationToken);
    public Task<StatusEnvelope> DeleteCollectionAsync(string projectId, string collectionName, CancellationToken cancellationToken = default) => SendAsync<StatusEnvelope>(HttpMethod.Delete, ProjectPath(projectId, $"collections/{Uri.EscapeDataString(collectionName)}"), cancellationToken);
    public Task<IReadOnlyList<RecordInfo>> ListRecordsAsync(string projectId, string collectionName, CancellationToken cancellationToken = default) => GetItemsAsync<RecordInfo>(ProjectPath(projectId, $"collections/{Uri.EscapeDataString(collectionName)}/records"), cancellationToken);
    public Task<RecordInfo> CreateRecordAsync(string projectId, string collectionName, Dictionary<string, object?> data, CancellationToken cancellationToken = default) => SendJsonAsync<RecordInfo>(HttpMethod.Post, ProjectPath(projectId, $"collections/{Uri.EscapeDataString(collectionName)}/records"), new RecordWriteRequest { Data = data }, cancellationToken);

    public Task<IReadOnlyList<ProjectDomainInfo>> ListDomainsAsync(string projectId, CancellationToken cancellationToken = default) => GetItemsAsync<ProjectDomainInfo>(ProjectPath(projectId, "domains"), cancellationToken);
    public Task<ProjectDomainInfo> CreateDomainRequestAsync(string projectId, string subdomain, CancellationToken cancellationToken = default) => SendJsonAsync<ProjectDomainInfo>(HttpMethod.Post, ProjectPath(projectId, "domains"), new ProjectDomainCreateRequest { Subdomain = subdomain }, cancellationToken);
    public Task<IReadOnlyList<ProjectDomainDeleteRequestInfo>> ListDomainDeleteRequestsAsync(string projectId, CancellationToken cancellationToken = default) => GetItemsAsync<ProjectDomainDeleteRequestInfo>(ProjectPath(projectId, "domain-delete-requests"), cancellationToken);
    public Task<ProjectDomainDeleteRequestInfo> CreateDomainDeleteRequestAsync(string projectId, string domainId, string reason, CancellationToken cancellationToken = default) => SendJsonAsync<ProjectDomainDeleteRequestInfo>(HttpMethod.Post, ProjectPath(projectId, "domain-delete-requests"), new ProjectDomainDeleteCreateRequest { DomainId = domainId, Reason = reason }, cancellationToken);

    public Task<IReadOnlyList<RepairRequestInfo>> ListRepairRequestsAsync(string projectId, CancellationToken cancellationToken = default) => GetItemsAsync<RepairRequestInfo>(ProjectPath(projectId, "repair-requests"), cancellationToken);
    public Task<RepairRequestInfo> CreateRepairRequestAsync(string projectId, RepairRequestCreateRequest request, CancellationToken cancellationToken = default) => SendJsonAsync<RepairRequestInfo>(HttpMethod.Post, ProjectPath(projectId, "repair-requests"), request, cancellationToken);
    public Task<RepairRequestInfo> ReplyRepairRequestAsync(string projectId, string requestId, string reply, CancellationToken cancellationToken = default) => SendJsonAsync<RepairRequestInfo>(HttpMethod.Post, ProjectPath(projectId, $"repair-requests/{Uri.EscapeDataString(requestId)}/reply"), new RepairReplyRequest { Reply = reply }, cancellationToken);
    public Task<RepairAIJobInfo> StartRepairAIAsync(string projectId, string requestId, CancellationToken cancellationToken = default) => SendJsonAsync<RepairAIJobInfo>(HttpMethod.Post, ProjectPath(projectId, $"repair-requests/{Uri.EscapeDataString(requestId)}/ai/start"), new { }, cancellationToken);
    public Task<RepairAIJobInfo> StopRepairAIAsync(string projectId, string requestId, CancellationToken cancellationToken = default) => SendJsonAsync<RepairAIJobInfo>(HttpMethod.Post, ProjectPath(projectId, $"repair-requests/{Uri.EscapeDataString(requestId)}/ai/stop"), new { }, cancellationToken);
    public Task<RepairAIJobInfo> GetLatestRepairAIAsync(string projectId, string requestId, CancellationToken cancellationToken = default) => SendAsync<RepairAIJobInfo>(HttpMethod.Get, ProjectPath(projectId, $"repair-requests/{Uri.EscapeDataString(requestId)}/ai/latest"), cancellationToken);
    public Task<RepairAIJobInfo> SendRepairAIFeedbackAsync(string projectId, string requestId, string feedback, CancellationToken cancellationToken = default) => SendJsonAsync<RepairAIJobInfo>(HttpMethod.Post, ProjectPath(projectId, $"repair-requests/{Uri.EscapeDataString(requestId)}/ai/feedback"), new RepairAIFeedbackRequest { Feedback = feedback }, cancellationToken);
    public Task<RepairAIJobInfo> CreateRepairAIPreviewAsync(string projectId, string requestId, CancellationToken cancellationToken = default) => SendJsonAsync<RepairAIJobInfo>(HttpMethod.Post, ProjectPath(projectId, $"repair-requests/{Uri.EscapeDataString(requestId)}/ai/preview"), new { }, cancellationToken);
    public Task<ReleaseInfo> PublishRepairAIAsync(string projectId, string requestId, CancellationToken cancellationToken = default) => SendJsonAsync<ReleaseInfo>(HttpMethod.Post, ProjectPath(projectId, $"repair-requests/{Uri.EscapeDataString(requestId)}/ai/publish"), new { }, cancellationToken);

    public Task<IReadOnlyList<TemplateInfo>> ListTemplatesAsync(CancellationToken cancellationToken = default) => GetItemsAsync<TemplateInfo>("api/v1/templates", cancellationToken);
    public Task<TemplateInfo> GetTemplateAsync(string templateId, CancellationToken cancellationToken = default) => SendAsync<TemplateInfo>(HttpMethod.Get, $"api/v1/templates/{Uri.EscapeDataString(templateId)}", cancellationToken);
    public Task<IReadOnlyList<UpgradeRequestInfo>> ListMyUpgradeRequestsAsync(CancellationToken cancellationToken = default) => GetItemsAsync<UpgradeRequestInfo>("api/v1/me/upgrade-requests", cancellationToken);
    public Task<UpgradeRequestInfo> CreateUpgradeRequestAsync(UpgradeRequestCreateRequest request, CancellationToken cancellationToken = default) => SendJsonAsync<UpgradeRequestInfo>(HttpMethod.Post, "api/v1/me/upgrade-requests", request, cancellationToken);

    private static string ProjectPath(string projectId, string suffix) => $"api/v1/projects/{Uri.EscapeDataString(projectId)}/{suffix}";
    private async Task<IReadOnlyList<T>> GetItemsAsync<T>(string path, CancellationToken cancellationToken) => (await SendAsync<ListEnvelope<T>>(HttpMethod.Get, path, cancellationToken).ConfigureAwait(false)).Items;

    private async Task<T> SendJsonAsync<T>(HttpMethod method, string path, object body, CancellationToken cancellationToken)
    {
        using var content = new StringContent(JsonSerializer.Serialize(body, _jsonOptions), Encoding.UTF8, "application/json");
        return await SendAsync<T>(method, path, cancellationToken, content).ConfigureAwait(false);
    }

    private async Task<DownloadedFile> SendJsonFileAsync(HttpMethod method, string path, object body, CancellationToken cancellationToken)
    {
        using var content = new StringContent(JsonSerializer.Serialize(body, _jsonOptions), Encoding.UTF8, "application/json");
        return await SendFileAsync(method, path, cancellationToken, content).ConfigureAwait(false);
    }

    private async Task<T> SendAsync<T>(HttpMethod method, string path, CancellationToken cancellationToken, HttpContent? content = null)
    {
        var text = await SendTextAsync(method, path, cancellationToken, content).ConfigureAwait(false);
        if (string.IsNullOrWhiteSpace(text)) throw new PlayPageApiException("服务器返回了空内容。");
        var value = JsonSerializer.Deserialize<T>(text, _jsonOptions);
        if (value == null) throw new PlayPageApiException("服务器返回内容格式不正确。");
        return value;
    }

    private async Task<string> SendTextAsync(HttpMethod method, string path, CancellationToken cancellationToken, HttpContent? content = null)
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
        return text;
    }

    private async Task<DownloadedFile> SendFileAsync(HttpMethod method, string path, CancellationToken cancellationToken, HttpContent? content = null)
    {
        using var request = new HttpRequestMessage(method, path);
        if (content != null) request.Content = content;
        if (!string.IsNullOrWhiteSpace(_token)) request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", _token);
        using var response = await _http.SendAsync(request, cancellationToken).ConfigureAwait(false);
        var bytes = await response.Content.ReadAsByteArrayAsync().ConfigureAwait(false);
        if (!response.IsSuccessStatusCode)
        {
            var text = Encoding.UTF8.GetString(bytes);
            var message = TryReadError(text) ?? $"请求失败，状态码：{(int)response.StatusCode}";
            throw new PlayPageApiException(message, response.StatusCode);
        }
        return new DownloadedFile
        {
            Content = bytes,
            ContentType = response.Content.Headers.ContentType?.ToString() ?? "application/octet-stream",
            FileName = ParseDownloadFileName(response.Content.Headers.ContentDisposition?.ToString()) ?? "download"
        };
    }

    private static string? ParseDownloadFileName(string? disposition)
    {
        if (string.IsNullOrWhiteSpace(disposition)) return null;
        foreach (var part in disposition.Split(';'))
        {
            var trimmed = part.Trim();
            if (trimmed.StartsWith("filename*=", StringComparison.OrdinalIgnoreCase))
            {
                var value = trimmed.Substring("filename*=".Length);
                const string prefix = "UTF-8''";
                if (value.StartsWith(prefix, StringComparison.OrdinalIgnoreCase)) value = value.Substring(prefix.Length);
                return Uri.UnescapeDataString(value.Trim('"'));
            }
            if (trimmed.StartsWith("filename=", StringComparison.OrdinalIgnoreCase))
            {
                return trimmed.Substring("filename=".Length).Trim('"');
            }
        }
        return null;
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
}
