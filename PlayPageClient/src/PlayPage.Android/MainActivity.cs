using System.Linq;
using Android.App;
using Android.Content;
using Android.Database;
using Android.Provider;
using Android.OS;
using Android.Views;
using Android.Widget;
using PlayPage.Core;

namespace PlayPage.Client.Android;

[Activity(Label = "PlayPage", MainLauncher = true, Exported = true)]
public sealed class MainActivity : Activity
{
    private readonly PlayPageApiClient _client = new PlayPageApiClient(PlayPageOptions.Production);
    private EditText? _email;
    private EditText? _username;
    private EditText? _code;
    private Button? _sendCode;
    private Button? _login;
    private Button? _refresh;
    private TextView? _status;
    private ListView? _projects;
    private ArrayAdapter<string>? _adapter;
    private readonly System.Collections.Generic.List<ProjectSummary> _projectItems = new System.Collections.Generic.List<ProjectSummary>();
    private const int PickReleaseFileRequest = 3001;
    private const int CreateDownloadFileRequest = 3002;
    private ProjectSummary? _pendingUploadProject;
    private byte[]? _pendingDownloadBytes;
    private string _pendingDownloadName = "download";

    protected override void OnCreate(Bundle? savedInstanceState)
    {
        base.OnCreate(savedInstanceState);
        Title = "PlayPage 客户端";

        var root = new LinearLayout(this) { Orientation = Orientation.Vertical };
        root.SetPadding(32, 32, 32, 32);

        _email = new EditText(this) { Hint = "邮箱" };
        _email.InputType = global::Android.Text.InputTypes.TextVariationEmailAddress;
        _email.ContentDescription = "邮箱";
        root.AddView(_email, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _username = new EditText(this) { Hint = "用户名，新账号注册时填写" };
        _username.ContentDescription = "用户名，新账号注册时填写";
        root.AddView(_username, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _code = new EditText(this) { Hint = "验证码" };
        _code.InputType = global::Android.Text.InputTypes.ClassNumber;
        _code.ContentDescription = "邮箱验证码";
        root.AddView(_code, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _sendCode = new Button(this) { Text = "发送验证码" };
        _sendCode.ContentDescription = "发送邮箱验证码";
        _sendCode.Click += async (_, _) => await SendCodeAsync();
        root.AddView(_sendCode, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _login = new Button(this) { Text = "登录" };
        _login.ContentDescription = "用邮箱验证码登录 PlayPage";
        _login.Click += async (_, _) => await LoginAsync();
        root.AddView(_login, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _refresh = new Button(this) { Text = "刷新作品" };
        _refresh.ContentDescription = "刷新我的作品列表";
        _refresh.Click += async (_, _) => await LoadProjectsAsync();
        root.AddView(_refresh, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _status = new TextView(this) { Text = "请输入邮箱，发送验证码后登录。" };
        _status.ContentDescription = "状态：请输入邮箱，发送验证码后登录。";
        root.AddView(_status, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _projects = new ListView(this);
        _projects.ContentDescription = "我的作品列表";
        _adapter = new ArrayAdapter<string>(this, global::Android.Resource.Layout.SimpleListItem1);
        _projects.Adapter = _adapter;
        _projects.ItemClick += (_, e) => ShowProjectActions(e.Position);
        root.AddView(_projects, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, 0, 1));

        SetContentView(root);
        _ = RestoreSessionAsync();
    }

    public override bool OnCreateOptionsMenu(IMenu? menu)
    {
        if (menu == null) return base.OnCreateOptionsMenu(menu);
        menu.Add("新建作品");
        menu.Add("模板市场");
        menu.Add("管理摘要");
        return true;
    }

    public override bool OnOptionsItemSelected(IMenuItem item)
    {
        _ = HandleMenuAsync(item.TitleFormatted?.ToString() ?? item.TitleCondensedFormatted?.ToString() ?? "");
        return true;
    }

    private async System.Threading.Tasks.Task HandleMenuAsync(string title)
    {
        try
        {
            if (title == "新建作品") await CreateProjectAsync();
            else if (title == "模板市场") await ShowTemplatesAsync();
            else if (title == "管理摘要") await ShowAdminSummaryAsync();
        }
        catch (System.Exception ex) { ShowError(ex); }
    }

    private async System.Threading.Tasks.Task RestoreSessionAsync()
    {
        try
        {
            var token = LoadToken();
            if (string.IsNullOrWhiteSpace(token)) return;
            _client.SetToken(token);
            var user = await _client.GetMeAsync();
            if (_email != null) _email.Text = user.Email;
            if (_username != null) _username.Text = user.Username;
            SetStatus($"已自动登录：{user.Username}。");
            await LoadProjectsAsync();
        }
        catch
        {
            SaveToken("");
            _client.SetToken(null);
            SetStatus("登录已过期，请重新发送验证码登录。");
        }
    }

    private string LoadToken() => GetSharedPreferences("playpage", FileCreationMode.Private)?.GetString("accessToken", "") ?? "";
    private void SaveToken(string token)
    {
        var editor = GetSharedPreferences("playpage", FileCreationMode.Private)?.Edit();
        if (editor == null) return;
        if (string.IsNullOrWhiteSpace(token)) editor.Remove("accessToken"); else editor.PutString("accessToken", token.Trim());
        editor.Apply();
    }

    private async System.Threading.Tasks.Task SendCodeAsync()
    {
        try
        {
            if (_sendCode != null) _sendCode.Enabled = false;
            SetStatus("正在发送验证码。");
            var result = await _client.RequestCodeAsync(_email?.Text ?? "", _username?.Text ?? "");
            SetStatus($"验证码已发送到 {result.Email}。");
        }
        catch (System.Exception ex) { ShowError(ex); }
        finally { if (_sendCode != null) _sendCode.Enabled = true; }
    }

    private async System.Threading.Tasks.Task LoginAsync()
    {
        try
        {
            if (_login != null) _login.Enabled = false;
            SetStatus("正在登录。");
            var result = await _client.VerifyCodeAsync(_email?.Text ?? "", _code?.Text ?? "");
            SaveToken(result.AccessToken);
            SetStatus($"已登录：{result.User.Username}，正在读取作品。");
            await LoadProjectsAsync();
        }
        catch (System.Exception ex) { ShowError(ex); }
        finally { if (_login != null) _login.Enabled = true; }
    }

    private async System.Threading.Tasks.Task LoadProjectsAsync()
    {
        try
        {
            if (_refresh != null) _refresh.Enabled = false;
            var projects = await _client.GetProjectsAsync();
            _projectItems.Clear();
            _projectItems.AddRange(projects);
            _adapter?.Clear();
            foreach (var project in _projectItems) _adapter?.Add($"{project.Name}（{project.Slug}，{project.Visibility}）");
            SetStatus($"已读取 {_projectItems.Count} 个作品。点击作品可操作。");
        }
        catch (System.Exception ex) { ShowError(ex); }
        finally { if (_refresh != null) _refresh.Enabled = true; }
    }

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
                            if (!string.IsNullOrWhiteSpace(project.PublicUrl)) StartActivity(new global::Android.Content.Intent(global::Android.Content.Intent.ActionView, global::Android.Net.Uri.Parse(project.PublicUrl)));
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
        var name = await PromptAsync("新建作品", "作品名称");
        if (string.IsNullOrWhiteSpace(name)) return;
        var slug = await PromptAsync("新建作品", "作品链接名，可留空");
        await _client.CreateProjectAsync(new ProjectCreateRequest { Name = name, Slug = slug, Interactive = true, AnalyticsEnabled = false });
        await LoadProjectsAsync();
    }

    private async System.Threading.Tasks.Task ShowTemplatesAsync()
    {
        var items = await _client.ListTemplatesAsync();
        var text = items.Count == 0 ? "暂无模板。" : string.Join("\n", items.Select(t => $"{t.Name} - {t.Summary}"));
        ShowMessage("模板市场", text);
    }

    private async System.Threading.Tasks.Task ShowAdminSummaryAsync()
    {
        var users = await _client.AdminListUsersAsync();
        var projects = await _client.AdminListProjectsAsync();
        var repairs = await _client.AdminListRepairRequestsAsync();
        ShowMessage("管理摘要", $"用户：{users.Count}\n作品：{projects.Count}\n修复申请：{repairs.Count}");
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

    private async System.Threading.Tasks.Task CreateRepairAsync(ProjectSummary project)
    {
        var desc = await PromptAsync("申请修复", "请描述遇到的问题");
        if (string.IsNullOrWhiteSpace(desc)) return;
        await _client.CreateRepairRequestAsync(project.Id, new RepairRequestCreateRequest { IssueType = "other", Description = desc, AllowAdminEdit = true, Contact = _email?.Text ?? "" });
        SetStatus("修复申请已提交。");
    }

    private async System.Threading.Tasks.Task ShowRepairRequestsAsync(ProjectSummary project)
    {
        var repairs = await _client.ListRepairRequestsAsync(project.Id);
        ShowMessage("修复申请", repairs.Count == 0 ? "暂无修复申请。" : string.Join("\n\n", repairs.Select(r => $"{r.Status}\n{r.Description}")));
    }

    private async System.Threading.Tasks.Task StartOrShowRepairAIAsync(ProjectSummary project)
    {
        var repairs = await _client.ListRepairRequestsAsync(project.Id);
        if (repairs.Count == 0) { ShowMessage("AI 圆桌", "这个作品还没有修复申请。"); return; }
        try
        {
            var state = await _client.GetLatestRepairAIAsync(project.Id, repairs[0].Id);
            ShowAIState(state);
        }
        catch
        {
            var state = await _client.StartRepairAIAsync(project.Id, repairs[0].Id);
            ShowAIState(state);
        }
    }

    private async System.Threading.Tasks.Task CreateDomainAsync(ProjectSummary project)
    {
        var subdomain = await PromptAsync("申请独立网址", "子域名，例如 my-game");
        if (string.IsNullOrWhiteSpace(subdomain)) return;
        var created = await _client.CreateDomainRequestAsync(project.Id, subdomain);
        SetStatus("独立网址申请已提交：" + created.Domain);
    }

    private async System.Threading.Tasks.Task ShowDomainsAsync(ProjectSummary project)
    {
        var domains = await _client.ListDomainsAsync(project.Id);
        ShowMessage("独立网址", domains.Count == 0 ? "暂无独立网址申请。" : string.Join("\n", domains.Select(d => $"{d.Domain} - {d.Status}")));
    }

    private void ShowAIState(RepairAIState state)
    {
        var lines = new System.Collections.Generic.List<string> { $"状态：{state.Job.Status}；第 {state.Job.Round} 轮" };
        foreach (var message in state.Messages)
        {
            lines.Add($"{message.MessageSeq}. {message.AgentName}");
            lines.Add(message.Content);
        }
        ShowMessage("AI 圆桌", string.Join("\n", lines));
    }

    private System.Threading.Tasks.Task<string> PromptAsync(string title, string hint)
    {
        var tcs = new System.Threading.Tasks.TaskCompletionSource<string>();
        var input = new EditText(this) { Hint = hint };
        input.ContentDescription = hint;
        new AlertDialog.Builder(this)
            .SetTitle(title)
            .SetView(input)
            .SetPositiveButton("确定", (_, _) => tcs.TrySetResult(input.Text ?? ""))
            .SetNegativeButton("取消", (_, _) => tcs.TrySetResult(""))
            .Show();
        return tcs.Task;
    }

    private void ShowMessage(string title, string message) => new AlertDialog.Builder(this).SetTitle(title).SetMessage(message).SetPositiveButton("确定", (_, _) => { }).Show();
    private void ShowError(System.Exception ex)
    {
        SetStatus("操作失败：" + ex.Message);
        Toast.MakeText(this, ex.Message, ToastLength.Long)?.Show();
    }

    private void SetStatus(string text)
    {
        if (_status == null) return;
        _status.Text = text;
        _status.ContentDescription = "状态：" + text;
        _status.SendAccessibilityEvent(global::Android.Views.Accessibility.EventTypes.Announcement);
    }
}


