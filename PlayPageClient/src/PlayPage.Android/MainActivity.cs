using Android.App;
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
        var actions = new[] { "打开作品", "查看详情", "切换广场显示", "查看修复申请", "查看独立网址", "查看历史版本" };
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
                            var repairs = await _client.ListRepairRequestsAsync(project.Id);
                            ShowMessage("修复申请", repairs.Count == 0 ? "暂无修复申请。" : $"共有 {repairs.Count} 条修复申请。");
                            break;
                        case 4:
                            var domains = await _client.ListDomainsAsync(project.Id);
                            ShowMessage("独立网址", domains.Count == 0 ? "暂无独立网址申请。" : $"共有 {domains.Count} 条独立网址申请。");
                            break;
                        case 5:
                            var releases = await _client.ListReleasesAsync(project.Id);
                            ShowMessage("历史版本", releases.Count == 0 ? "暂无历史版本。" : $"共有 {releases.Count} 个版本。");
                            break;
                    }
                }
                catch (System.Exception ex) { ShowError(ex); }
            })
            .Show();
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
