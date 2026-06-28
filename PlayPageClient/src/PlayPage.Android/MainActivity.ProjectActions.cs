using System.Collections.Generic;
using System.Linq;
using System.Text.Encodings.Web;
using System.Text.Json;
using Android.App;
using Android.Content;
using Android.Database;
using Android.Provider;
using Android.Text;
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
        var actions = new[] { "打开作品", "查看详情", "作品设置", "切换广场显示", "上传新版本", "下载作品源码", "管理互动数据", "导出数据表", "申请修复", "查看修复申请", "申请独立网址", "查看独立网址", "查看历史版本", "统计数据", "删除作品" };
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
                            ShowMessage("作品详情", $"地址：{detail.Project.PublicUrl}\n互动功能：{PlayPageDisplay.YesNo(detail.Project.Interactive)}\n访问统计：{PlayPageDisplay.YesNo(detail.Project.AnalyticsEnabled)}\n广场显示：{PlayPageDisplay.Visibility(detail.Project.Visibility)}");
                            break;
                        case 2:
                            await EditSettingsAsync(project);
                            break;
                        case 3:
                            await _client.UpdateProjectVisibilityAsync(project.Id, project.Visibility == "public" ? "unlisted" : "public");
                            await LoadProjectsAsync();
                            break;
                        case 4:
                            PickReleaseFile(project);
                            break;
                        case 5:
                            await DownloadSourceAsync(project);
                            break;
                        case 6:
                            await ManageInteractiveDataAsync(project);
                            break;
                        case 7:
                            await ExportDataAsync(project);
                            break;
                        case 8:
                            await CreateRepairAsync(project);
                            break;
                        case 9:
                            await ShowRepairRequestsAsync(project);
                            break;
                        case 10:
                            await CreateDomainAsync(project);
                            break;
                        case 11:
                            await ShowDomainsAsync(project);
                            break;
                        case 12:
                            await ShowReleasesAsync(project);
                            break;
                        case 13:
                            var stats = await _client.GetProjectStatsAsync(project.Id);
                            ShowMessage("统计数据", $"访问量：{stats.TotalPageViews}\nAPI 请求：{stats.TotalApiRequests}\n成功：{stats.TotalApiSuccesses}\n失败：{stats.TotalApiFailures}\n成功率：{stats.ApiSuccessRate:P2}");
                            break;
                        case 14:
                            await DeleteProjectAsync(project);
                            break;
                    }
                }
                catch (System.Exception ex) { ShowError(ex); }
            })
            .Show();
    }


    private System.Threading.Tasks.Task<bool> EditSettingsAsync(ProjectSummary project)
    {
        var tcs = new System.Threading.Tasks.TaskCompletionSource<bool>();
        var layout = new LinearLayout(this) { Orientation = Orientation.Vertical };
        layout.SetPadding(32, 12, 32, 0);

        var name = new EditText(this) { Hint = "作品名称", Text = project.Name };
        name.ContentDescription = "作品名称";
        layout.AddView(name);

        var slug = new EditText(this) { Hint = "作品链接名", Text = project.Slug };
        slug.ContentDescription = "作品链接名";
        layout.AddView(slug);

        var interactive = new CheckBox(this) { Text = "启用互动功能", Checked = project.Interactive };
        interactive.ContentDescription = "启用互动功能";
        layout.AddView(interactive);

        var analytics = new CheckBox(this) { Text = "启用访问量统计", Checked = project.AnalyticsEnabled };
        analytics.ContentDescription = "启用访问量统计";
        layout.AddView(analytics);

        var status = new TextView(this) { Text = "修改后点击保存。" };
        layout.AddView(status);

        var dialog = new AlertDialog.Builder(this)
            .SetTitle("作品设置")
            .SetView(layout)
            .SetPositiveButton("保存", (sender, _) => { })
            .SetNegativeButton("取消", (_, _) => tcs.TrySetResult(false))
            .Create();

        dialog.SetOnShowListener(new DialogShowListener(() =>
        {
            var ok = dialog.GetButton((int)DialogButtonType.Positive);
            ok.Click += async (_, _) =>
            {
                var nextName = name.Text?.Trim() ?? "";
                var nextSlug = NormalizeSlug(slug.Text ?? "");
                if (nextName.Length == 0)
                {
                    status.Text = "请填写作品名称。";
                    return;
                }
                if (nextSlug.Length == 0)
                {
                    status.Text = "请填写作品链接名。";
                    return;
                }
                try
                {
                    ok.Enabled = false;
                    status.Text = "正在保存。";
                    await _client.UpdateProjectSettingsAsync(project.Id, new ProjectSettingsRequest
                    {
                        Name = nextName,
                        Slug = nextSlug,
                        Interactive = interactive.Checked,
                        AnalyticsEnabled = analytics.Checked
                    });
                    dialog.Dismiss();
                    SetStatus("作品设置已保存。");
                    await LoadProjectsAsync();
                    tcs.TrySetResult(true);
                }
                catch (System.Exception ex)
                {
                    ok.Enabled = true;
                    status.Text = "保存失败：" + ex.Message;
                    Toast.MakeText(this, ex.Message, ToastLength.Long)?.Show();
                }
            };
        }));
        dialog.Show();
        return tcs.Task;
    }

    private async System.Threading.Tasks.Task CreateProjectAsync()
    {
        if (_currentUser == null)
        {
            await ShowLoginDialogAsync();
            return;
        }

        var input = await PromptCreateProjectAsync();
        if (input == null) return;

        SetStatus("正在创建作品。");
        var project = await _client.CreateProjectAsync(new ProjectCreateRequest
        {
            Name = input.Name,
            Slug = input.Slug,
            Interactive = input.Interactive,
            AnalyticsEnabled = input.AnalyticsEnabled
        });

        if (input.Mode == AndroidCreateUploadMode.File)
        {
            SetStatus("请选择要上传的 HTML 或 ZIP 文件。");
            PickReleaseFile(project);
        }
        else if (input.Mode == AndroidCreateUploadMode.HtmlText)
        {
            SetStatus("正在发布粘贴的 HTML。");
            await _client.UploadReleaseHtmlTextAsync(project.Id, input.HtmlText, input.ChangeNote);
            await LoadProjectsAsync();
        }
        else
        {
            await LoadProjectsAsync();
        }
    }


    private enum AndroidCreateUploadMode
    {
        Empty,
        File,
        HtmlText
    }

    private sealed class AndroidCreateProjectInput
    {
        public string Name { get; set; } = "";
        public string Slug { get; set; } = "";
        public bool Interactive { get; set; }
        public bool AnalyticsEnabled { get; set; }
        public AndroidCreateUploadMode Mode { get; set; }
        public string HtmlText { get; set; } = "";
        public string ChangeNote { get; set; } = "";
    }

    private System.Threading.Tasks.Task<AndroidCreateProjectInput?> PromptCreateProjectAsync()
    {
        var tcs = new System.Threading.Tasks.TaskCompletionSource<AndroidCreateProjectInput?>();
        var scroll = new ScrollView(this);
        var layout = new LinearLayout(this) { Orientation = Orientation.Vertical };
        layout.SetPadding(32, 12, 32, 0);
        scroll.AddView(layout);

        var name = new EditText(this) { Hint = "作品名称" };
        name.ContentDescription = "作品名称";
        layout.AddView(name);

        var slug = new EditText(this) { Hint = "作品链接名" };
        slug.ContentDescription = "作品链接名";
        layout.AddView(slug);
        var slugTouched = false;
        name.TextChanged += (_, _) =>
        {
            if (!slugTouched) slug.Text = NormalizeSlug(name.Text ?? "");
        };
        slug.TextChanged += (_, _) => slugTouched = true;

        var interactive = new CheckBox(this) { Text = "启用互动功能" };
        interactive.ContentDescription = "启用互动功能";
        layout.AddView(interactive);
        layout.AddView(new TextView(this) { Text = "互动功能会启用作品数据接口，适合评论、留言、论坛、云存档等作品。" });

        var analytics = new CheckBox(this) { Text = "启用访问量统计" };
        analytics.ContentDescription = "启用访问量统计";
        layout.AddView(analytics);
        layout.AddView(new TextView(this) { Text = "访问量统计用于每日访问量和互动 API 请求统计。" });

        var modeGroup = new RadioGroup(this) { Orientation = Orientation.Vertical };
        var modeFile = new RadioButton(this) { Text = "创建后选择 HTML 或 ZIP 文件上传" };
        var modeText = new RadioButton(this) { Text = "直接粘贴 HTML 代码" };
        var modeEmpty = new RadioButton(this) { Text = "先只创建空作品，以后再上传" };
        modeGroup.AddView(modeFile);
        modeGroup.AddView(modeText);
        modeGroup.AddView(modeEmpty);
        modeFile.Checked = true;
        layout.AddView(modeGroup);

        var html = new EditText(this) { Hint = "HTML 代码" };
        html.ContentDescription = "HTML 代码";
        html.SetSingleLine(false);
        html.SetMinLines(8);
        html.InputType = InputTypes.ClassText | InputTypes.TextFlagMultiLine | InputTypes.TextFlagNoSuggestions;
        html.Text = "<!doctype html>\n<html lang=\"zh-CN\">\n<head>\n  <meta charset=\"utf-8\">\n  <title>我的作品</title>\n</head>\n<body>\n  <h1>你好，PlayPage</h1>\n</body>\n</html>";
        layout.AddView(html);

        var changeNote = new EditText(this) { Hint = "更新内容，可留空" };
        changeNote.ContentDescription = "更新内容";
        layout.AddView(changeNote);

        void UpdateHtmlEnabled()
        {
            html.Enabled = modeText.Checked;
        }
        modeGroup.CheckedChange += (_, _) => UpdateHtmlEnabled();
        UpdateHtmlEnabled();

        var status = new TextView(this) { Text = "先填写作品信息，再选择上传方式。" };
        layout.AddView(status);

        var dialog = new AlertDialog.Builder(this)
            .SetTitle("创建作品")
            .SetView(scroll)
            .SetPositiveButton("创建作品", (sender, _) => { })
            .SetNegativeButton("取消", (_, _) => tcs.TrySetResult(null))
            .Create();

        dialog.SetOnShowListener(new DialogShowListener(() =>
        {
            var ok = dialog.GetButton((int)DialogButtonType.Positive);
            ok.Click += (_, _) =>
            {
                var projectName = name.Text?.Trim() ?? "";
                var projectSlug = NormalizeSlug(slug.Text ?? "");
                if (projectName.Length == 0)
                {
                    status.Text = "请先填写作品名称。";
                    return;
                }
                if (projectSlug.Length == 0)
                {
                    status.Text = "请先填写作品链接名。";
                    return;
                }
                if (modeText.Checked && string.IsNullOrWhiteSpace(html.Text))
                {
                    status.Text = "请先粘贴 HTML 代码。";
                    return;
                }

                var mode = modeText.Checked ? AndroidCreateUploadMode.HtmlText : modeEmpty.Checked ? AndroidCreateUploadMode.Empty : AndroidCreateUploadMode.File;
                tcs.TrySetResult(new AndroidCreateProjectInput
                {
                    Name = projectName,
                    Slug = projectSlug,
                    Interactive = interactive.Checked,
                    AnalyticsEnabled = analytics.Checked,
                    Mode = mode,
                    HtmlText = html.Text ?? "",
                    ChangeNote = changeNote.Text?.Trim() ?? ""
                });
                dialog.Dismiss();
            };
        }));
        dialog.Show();
        return tcs.Task;
    }

    private static string NormalizeSlug(string value)
    {
        var text = (value ?? "").Trim().ToLowerInvariant();
        var builder = new System.Text.StringBuilder(text.Length);
        var lastDash = false;
        foreach (var ch in text)
        {
            if (char.IsLetterOrDigit(ch) || ch == '_' || ch == '.')
            {
                builder.Append(ch);
                lastDash = false;
            }
            else if (!lastDash)
            {
                builder.Append('-');
                lastDash = true;
            }
        }
        return builder.ToString().Trim('-', '.', '_');
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


    private async System.Threading.Tasks.Task DeleteProjectAsync(ProjectSummary project)
    {
        var domains = await _client.ListDomainsAsync(project.Id);
        var activeDomains = domains.Where(d => d.Status == "active").ToList();
        var message = $"确认删除作品“{project.Name}”吗？";
        if (activeDomains.Count > 0) message += $"\n这个作品有 {activeDomains.Count} 个已通过的独立网址，会先自动提交删除独立网址申请。";
        new AlertDialog.Builder(this)
            .SetTitle("删除作品")
            .SetMessage(message)
            .SetPositiveButton("删除", async (_, _) =>
            {
                try
                {
                    foreach (var domain in activeDomains)
                    {
                        try { await _client.CreateDomainDeleteRequestAsync(project.Id, domain.Id, "删除作品时自动申请删除独立网址"); }
                        catch { }
                    }
                    await _client.DeleteProjectAsync(project.Id);
                    SetStatus("作品已删除。 ");
                    await LoadProjectsAsync();
                }
                catch (System.Exception ex) { ShowError(ex); }
            })
            .SetNegativeButton("取消", (_, _) => { })
            .Show();
    }

    private async System.Threading.Tasks.Task ShowReleasesAsync(ProjectSummary project)
    {
        var releases = await _client.ListReleasesAsync(project.Id);
        if (releases.Count == 0)
        {
            ShowMessage("历史版本", "暂无历史版本。");
            return;
        }

        var ordered = releases.OrderByDescending(r => r.CreatedAt).ToList();
        var labels = ordered.Select(r =>
        {
            var note = string.IsNullOrWhiteSpace(r.ChangeNote) ? "无更新说明" : r.ChangeNote;
            return $"{r.CreatedAt.LocalDateTime:yyyy-MM-dd HH:mm:ss}\n{note}";
        }).ToArray();

        new AlertDialog.Builder(this)
            .SetTitle("选择要回滚的版本")
            .SetItems(labels, (sender, args) =>
            {
                var selected = ordered[args.Which];
                new AlertDialog.Builder(this)
                    .SetTitle("确认回滚")
                    .SetMessage("确认回滚到这个版本吗？\n" + labels[args.Which])
                    .SetPositiveButton("回滚", async (_, _) =>
                    {
                        try
                        {
                            await _client.RollbackReleaseAsync(project.Id, selected.Id);
                            SetStatus("作品已经回滚到所选版本。");
                            await LoadProjectsAsync();
                        }
                        catch (System.Exception ex) { ShowError(ex); }
                    })
                    .SetNegativeButton("取消", (_, _) => { })
                    .Show();
            })
            .SetNegativeButton("关闭", (_, _) => { })
            .Show();
    }

    private static readonly JsonSerializerOptions InteractiveJsonOptions = new JsonSerializerOptions
    {
        WriteIndented = true,
        Encoder = JavaScriptEncoder.UnsafeRelaxedJsonEscaping,
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
        PropertyNameCaseInsensitive = true
    };

    private async System.Threading.Tasks.Task ManageInteractiveDataAsync(ProjectSummary project)
    {
        if (!project.Interactive)
        {
            ShowMessage("管理互动数据", "这个作品没有开启互动功能。可以先在作品设置里开启互动功能。");
            return;
        }
        var collections = await _client.ListCollectionsAsync(project.Id);
        var actions = collections.Select(c => $"{c.Name}（{c.Fields.Count} 个字段）").Concat(new[] { "新建数据集合" }).ToArray();
        new AlertDialog.Builder(this)
            .SetTitle("管理互动数据")
            .SetItems(actions, async (_, args) =>
            {
                try
                {
                    if (args.Which == collections.Count) await CreateCollectionFromJsonAsync(project);
                    else await ShowCollectionActionsAsync(project, collections[args.Which]);
                }
                catch (System.Exception ex) { ShowError(ex); }
            })
            .SetNegativeButton("关闭", (_, _) => { })
            .Show();
    }

    private System.Threading.Tasks.Task ShowCollectionActionsAsync(ProjectSummary project, CollectionInfo collection)
    {
        var tcs = new System.Threading.Tasks.TaskCompletionSource<bool>();
        var actions = new[] { "查看集合结构", "修改集合结构", "查看记录", "新增记录" };
        new AlertDialog.Builder(this)
            .SetTitle(collection.Name)
            .SetItems(actions, async (_, args) =>
            {
                try
                {
                    switch (args.Which)
                    {
                        case 0:
                            ShowMessage("集合结构", JsonSerializer.Serialize(collection, InteractiveJsonOptions));
                            break;
                        case 1:
                            await UpdateCollectionFromJsonAsync(project, collection);
                            break;
                        case 2:
                            await ShowRecordsAsync(project, collection);
                            break;
                        case 3:
                            await CreateRecordFromJsonAsync(project, collection);
                            break;
                    }
                }
                catch (System.Exception ex) { ShowError(ex); }
                finally { tcs.TrySetResult(true); }
            })
            .SetNegativeButton("返回", (_, _) => tcs.TrySetResult(false))
            .Show();
        return tcs.Task;
    }

    private async System.Threading.Tasks.Task CreateCollectionFromJsonAsync(ProjectSummary project)
    {
        var preset = "{\n  \"name\": \"messages\",\n  \"permissions\": { \"publicRead\": true, \"publicWrite\": true },\n  \"fields\": [\n    { \"name\": \"nickname\", \"type\": \"string\", \"required\": true, \"isList\": false },\n    { \"name\": \"content\", \"type\": \"text\", \"required\": true, \"isList\": false }\n  ]\n}";
        var json = await PromptMultilineAsync("新建数据集合", "集合 JSON", preset);
        if (string.IsNullOrWhiteSpace(json)) return;
        var request = JsonSerializer.Deserialize<CollectionCreateRequest>(json, InteractiveJsonOptions);
        if (request == null || string.IsNullOrWhiteSpace(request.Name)) { ShowMessage("新建数据集合", "JSON 里必须包含 name。"); return; }
        await _client.CreateCollectionAsync(project.Id, request);
        SetStatus("数据集合已创建。 ");
    }

    private async System.Threading.Tasks.Task UpdateCollectionFromJsonAsync(ProjectSummary project, CollectionInfo collection)
    {
        var preset = JsonSerializer.Serialize(new CollectionUpdateRequest { Permissions = collection.Permissions, Fields = collection.Fields }, InteractiveJsonOptions);
        var json = await PromptMultilineAsync("修改集合结构", "permissions 和 fields JSON", preset);
        if (string.IsNullOrWhiteSpace(json)) return;
        var request = JsonSerializer.Deserialize<CollectionUpdateRequest>(json, InteractiveJsonOptions);
        if (request == null) { ShowMessage("修改集合结构", "JSON 格式无效。"); return; }
        await _client.UpdateCollectionAsync(project.Id, collection.Name, request);
        SetStatus("集合结构已保存。 ");
    }

    private async System.Threading.Tasks.Task ShowRecordsAsync(ProjectSummary project, CollectionInfo collection)
    {
        var records = await _client.ListRecordsAsync(project.Id, collection.Name);
        if (records.Count == 0) { ShowMessage("记录", "这个集合还没有记录。"); return; }
        var text = string.Join("\n\n", records.Take(30).Select(r => $"ID：{r.Id}\n状态：{r.Status}\n创建：{r.CreatedAt.LocalDateTime:yyyy-MM-dd HH:mm:ss}\n数据：{JsonSerializer.Serialize(r.Data, InteractiveJsonOptions)}"));
        if (records.Count > 30) text += $"\n\n只显示前 30 条，共 {records.Count} 条。";
        ShowMessage("记录 - " + collection.Name, text);
    }

    private async System.Threading.Tasks.Task CreateRecordFromJsonAsync(ProjectSummary project, CollectionInfo collection)
    {
        var json = await PromptMultilineAsync("新增记录", "data JSON，不要外包 data", "{\n  \"nickname\": \"访客\",\n  \"content\": \"你好，PlayPage！\"\n}");
        if (string.IsNullOrWhiteSpace(json)) return;
        var data = JsonSerializer.Deserialize<Dictionary<string, object?>>(json, InteractiveJsonOptions);
        if (data == null) { ShowMessage("新增记录", "JSON 格式无效。"); return; }
        await _client.CreateRecordAsync(project.Id, collection.Name, data);
        SetStatus("记录已新增。 ");
    }

    private System.Threading.Tasks.Task<string> PromptMultilineAsync(string title, string hint, string initialText)
    {
        var tcs = new System.Threading.Tasks.TaskCompletionSource<string>();
        var input = new EditText(this) { Hint = hint, Text = initialText };
        input.ContentDescription = hint;
        input.SetMinLines(8);
        input.SetSingleLine(false);
        input.InputType = InputTypes.ClassText | InputTypes.TextFlagMultiLine | InputTypes.TextFlagNoSuggestions;
        input.SetHorizontallyScrolling(false);
        new AlertDialog.Builder(this)
            .SetTitle(title)
            .SetView(input)
            .SetPositiveButton("确定", (_, _) => tcs.TrySetResult(input.Text ?? ""))
            .SetNegativeButton("取消", (_, _) => tcs.TrySetResult(""))
            .Show();
        return tcs.Task;
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
