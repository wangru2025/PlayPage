using System.Linq;
using Android.App;
using Android.Content;
using Android.Database;
using Android.Provider;
using Android.Widget;
using PlayPage.Core;

namespace PlayPage.Client.Android;

public sealed partial class MainActivity
{
    private const int PickReleaseFileRequest = 3001;
    private const int CreateDownloadFileRequest = 3002;
    private ProjectSummary? _pendingUploadProject;
    private byte[]? _pendingDownloadBytes;
    private string _pendingDownloadName = "download";

    private void ShowProjectActions(int position)
    {
        if (position < 0 || position >= _projectItems.Count) return;
        var project = _projectItems[position];
        var actions = new[] { "打开作品", "查看详情", "切换广场显示", "上传新版本", "下载作品源码", "导出数据表", "申请修复", "查看修复申请", "AI 圆桌", "申请独立网址", "查看独立网址", "查看历史版本", "统计数据" };
        new AlertDialog.Builder(this)
            .SetTitle(project.Name)
            .SetItems(actions, async (_, args) =>
            {
                try
                {
                    switch (args.Which)
                    {
                        case 0:
                            if (!string.IsNullOrWhiteSpace(project.PublicUrl)) StartActivity(new Intent(Intent.ActionView, global::Android.Net.Uri.Parse(project.PublicUrl)));
                            break;
                        case 1:
                            var detail = await _client.GetProjectAsync(project.Id);
                            ShowMessage("作品详情", $"地址：{detail.Project.PublicUrl}\n互动：{detail.Project.Interactive}\n统计：{detail.Project.AnalyticsEnabled}\n可见性：{detail.Project.Visibility}");
                            break;
                        case 2:
                            await _client.UpdateProjectVisibilityAsync(project.Id, project.Visibility == "public" ? "unlisted" : "public");
                            await LoadProjectsAsync();
                            break;
                        case 3:
                            PickReleaseFile(project);
                            break;
                        case 4:
                            await DownloadSourceAsync(project);
                            break;
                        case 5:
                            await ExportDataAsync(project);
                            break;
                        case 6:
                            await CreateRepairAsync(project);
                            break;
                        case 7:
                            await ShowRepairRequestsAsync(project);
                            break;
                        case 8:
                            await StartOrShowRepairAIAsync(project);
                            break;
                        case 9:
                            await CreateDomainAsync(project);
                            break;
                        case 10:
                            await ShowDomainsAsync(project);
                            break;
                        case 11:
                            var releases = await _client.ListReleasesAsync(project.Id);
                            ShowMessage("历史版本", releases.Count == 0 ? "暂无历史版本。" : $"共有 {releases.Count} 个版本。");
                            break;
                        case 12:
                            var stats = await _client.GetProjectStatsAsync(project.Id);
                            ShowMessage("统计数据", $"访问量：{stats.TotalPageViews}\nAPI 请求：{stats.TotalApiRequests}\n成功：{stats.TotalApiSuccesses}\n失败：{stats.TotalApiFailures}");
                            break;
                    }
                }
                catch (System.Exception ex) { ShowError(ex); }
            })
            .Show();
    }

    private async System.Threading.Tasks.Task CreateProjectAsync()
    {
        if (_currentUser == null)
        {
            await ShowLoginDialogAsync();
            return;
        }
        var name = await PromptAsync("新建作品", "作品名称");
        if (string.IsNullOrWhiteSpace(name)) return;
        var slug = await PromptAsync("新建作品", "作品链接名，可留空");
        await _client.CreateProjectAsync(new ProjectCreateRequest { Name = name, Slug = slug, Interactive = true, AnalyticsEnabled = false });
        await LoadProjectsAsync();
    }

    private void PickReleaseFile(ProjectSummary project)
    {
        _pendingUploadProject = project;
        var intent = new Intent(Intent.ActionOpenDocument);
        intent.AddCategory(Intent.CategoryOpenable);
        intent.SetType("*/*");
        intent.PutExtra(Intent.ExtraMimeTypes, new[] { "text/html", "application/zip", "application/x-zip-compressed" });
        StartActivityForResult(intent, PickReleaseFileRequest);
    }

    protected override async void OnActivityResult(int requestCode, Result resultCode, Intent? data)
    {
        base.OnActivityResult(requestCode, resultCode, data);
        if (resultCode != Result.Ok || data?.Data == null) return;
        if (requestCode == PickReleaseFileRequest && _pendingUploadProject != null)
        {
            try
            {
                var uri = data.Data;
                var name = GetDisplayName(uri) ?? "upload.html";
                var mode = name.EndsWith(".zip", System.StringComparison.OrdinalIgnoreCase) ? "zip" : "html";
                using var input = ContentResolver?.OpenInputStream(uri);
                if (input == null) throw new System.InvalidOperationException("无法读取选择的文件。");
                await _client.UploadReleaseStreamAsync(_pendingUploadProject.Id, input, name, mode);
                SetStatus("新版本已上传。");
                await LoadProjectsAsync();
            }
            catch (System.Exception ex) { ShowError(ex); }
            finally { _pendingUploadProject = null; }
            return;
        }
        if (requestCode == CreateDownloadFileRequest && _pendingDownloadBytes != null)
        {
            try
            {
                using var output = ContentResolver?.OpenOutputStream(data.Data);
                if (output == null) throw new System.InvalidOperationException("无法写入选择的文件。");
                await output.WriteAsync(_pendingDownloadBytes, 0, _pendingDownloadBytes.Length);
                SetStatus("文件已保存：" + _pendingDownloadName);
            }
            catch (System.Exception ex) { ShowError(ex); }
            finally { _pendingDownloadBytes = null; _pendingDownloadName = "download"; }
        }
    }

    private string? GetDisplayName(global::Android.Net.Uri uri)
    {
        ICursor? cursor = null;
        try
        {
            cursor = ContentResolver?.Query(uri, null, null, null, null);
            if (cursor != null && cursor.MoveToFirst())
            {
                var index = cursor.GetColumnIndex(OpenableColumns.DisplayName);
                if (index >= 0) return cursor.GetString(index);
            }
        }
        finally { cursor?.Close(); }
        return uri.LastPathSegment;
    }

    private async System.Threading.Tasks.Task DownloadSourceAsync(ProjectSummary project)
    {
        var file = await _client.DownloadProjectSourceAsync(project.Id);
        SaveDownloadedFile(file);
    }

    private async System.Threading.Tasks.Task ExportDataAsync(ProjectSummary project)
    {
        var collections = await _client.ListCollectionsAsync(project.Id);
        if (collections.Count == 0) { ShowMessage("导出数据表", "这个作品还没有数据表。"); return; }
        var names = collections.Select(c => c.Name).ToList();
        var file = await _client.ExportProjectDataAsync(project.Id, names, "json");
        SaveDownloadedFile(file);
    }

    private void SaveDownloadedFile(DownloadedFile file)
    {
        _pendingDownloadBytes = file.Content;
        _pendingDownloadName = string.IsNullOrWhiteSpace(file.FileName) ? "download" : file.FileName;
        var intent = new Intent(Intent.ActionCreateDocument);
        intent.AddCategory(Intent.CategoryOpenable);
        intent.SetType(string.IsNullOrWhiteSpace(file.ContentType) ? "application/octet-stream" : file.ContentType);
        intent.PutExtra(Intent.ExtraTitle, _pendingDownloadName);
        StartActivityForResult(intent, CreateDownloadFileRequest);
    }
}
