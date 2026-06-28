using System.Linq;
using Android.App;
using Android.OS;
using Android.Views;
using Android.Widget;
using PlayPage.Core;

namespace PlayPage.Client.Android;

[Activity(Label = "PlayPage", MainLauncher = true, Exported = true)]
public sealed partial class MainActivity : Activity
{
    private readonly PlayPageApiClient _client = new PlayPageApiClient(PlayPageOptions.Production);
    private TextView? _account;
    private TextView? _status;
    private Button? _createProject;
    private Button? _refresh;
    private Button? _logout;
    private ListView? _projects;
    private ArrayAdapter<string>? _adapter;
    private UserProfile? _currentUser;
    private readonly System.Collections.Generic.List<ProjectSummary> _projectItems = new System.Collections.Generic.List<ProjectSummary>();

    protected override void OnCreate(Bundle? savedInstanceState)
    {
        base.OnCreate(savedInstanceState);
        Title = "PlayPage";
        BuildMainView();
        _ = InitializeSessionAsync();
    }

    private void BuildMainView()
    {
        var root = new LinearLayout(this) { Orientation = Orientation.Vertical };
        root.SetPadding(32, 32, 32, 32);

        _account = new TextView(this) { Text = "正在检查登录状态。", TextSize = 18 };
        _account.ContentDescription = "当前登录状态";
        root.AddView(_account, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        var actions = new LinearLayout(this) { Orientation = Orientation.Horizontal };
        _createProject = new Button(this) { Text = "创建作品" };
        _createProject.ContentDescription = "创建作品";
        _createProject.Click += async (_, _) => await CreateProjectAsync();
        actions.AddView(_createProject, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WrapContent, 1));

        _refresh = new Button(this) { Text = "刷新" };
        _refresh.ContentDescription = "刷新作品列表";
        _refresh.Click += async (_, _) => await LoadProjectsAsync();
        actions.AddView(_refresh, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WrapContent, 1));

        _logout = new Button(this) { Text = "退出" };
        _logout.ContentDescription = "退出登录";
        _logout.Click += async (_, _) => await LogoutAsync();
        actions.AddView(_logout, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WrapContent, 1));
        root.AddView(actions, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _status = new TextView(this) { Text = "正在启动 PlayPage。" };
        _status.ContentDescription = "状态：正在启动 PlayPage。";
        root.AddView(_status, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _projects = new ListView(this);
        _projects.ContentDescription = "我的作品列表";
        _adapter = new ArrayAdapter<string>(this, global::Android.Resource.Layout.SimpleListItem1);
        _projects.Adapter = _adapter;
        _projects.ItemClick += (_, e) => ShowProjectActions(e.Position);
        root.AddView(_projects, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, 0, 1));

        SetContentView(root);
        SetMainActionsEnabled(false);
    }

    public override bool OnCreateOptionsMenu(IMenu? menu)
    {
        if (menu == null) return base.OnCreateOptionsMenu(menu);
        menu.Add("新建作品");
        menu.Add("模板市场");
        menu.Add("投稿模板");
        menu.Add("个人中心");
        menu.Add("管理摘要");
        menu.Add("退出登录");
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
            else if (title == "投稿模板") await SubmitTemplateAsync();
            else if (title == "个人中心") await ShowAccountCenterAsync();
            else if (title == "管理摘要") await ShowAdminSummaryAsync();
            else if (title == "退出登录") await LogoutAsync();
        }
        catch (System.Exception ex) { ShowError(ex); }
    }

    private async System.Threading.Tasks.Task ShowAccountCenterAsync()
    {
        if (_currentUser == null)
        {
            await ShowLoginDialogAsync();
            if (_currentUser == null) return;
        }
        _currentUser = await _client.GetMeAsync();
        UpdateAccountLabel();
        var upgrades = await _client.ListMyUpgradeRequestsAsync();
        var requestText = upgrades.Count == 0
            ? "暂无升级申请。"
            : string.Join("\n", upgrades.Take(10).Select(x => $"{x.CreatedAt.LocalDateTime:yyyy-MM-dd HH:mm}｜{x.TargetPlan}｜{x.Status}｜{x.AdminNote}"));
        var message = $"邮箱：{_currentUser.Email}\n公开名字：{_currentUser.Username}\n套餐：{_currentUser.PlanCode}\n角色：{_currentUser.Role}\n状态：{_currentUser.Status}\n\n最近升级申请：\n{requestText}";
        new AlertDialog.Builder(this)
            .SetTitle("个人中心")
            .SetMessage(message)
            .SetPositiveButton("改名字", async (_, _) =>
            {
                try
                {
                    var username = await PromptAsync("修改公开名字", "新的公开名字");
                    if (string.IsNullOrWhiteSpace(username)) return;
                    _currentUser = await _client.UpdateProfileAsync(username.Trim());
                    UpdateAccountLabel();
                    SetStatus("公开名字已更新。");
                }
                catch (System.Exception ex) { ShowError(ex); }
            })
            .SetNeutralButton("升级套餐", async (_, _) =>
            {
                try { await CreateUpgradeRequestAsync(); }
                catch (System.Exception ex) { ShowError(ex); }
            })
            .SetNegativeButton("关闭", (_, _) => { })
            .Show();
    }

    private async System.Threading.Tasks.Task CreateUpgradeRequestAsync()
    {
        var targetPlan = await PromptAsync("升级套餐", "目标套餐代码，例如 light 或 pro");
        if (string.IsNullOrWhiteSpace(targetPlan)) return;
        var payment = await PromptAsync("升级套餐", "付款方式，例如 wechat、alipay，可留空");
        var note = await PromptAsync("升级套餐", "付款备注、转账昵称或其他说明，可留空");
        var created = await _client.CreateUpgradeRequestAsync(new UpgradeRequestCreateRequest
        {
            TargetPlan = targetPlan.Trim(),
            PaymentMethod = string.IsNullOrWhiteSpace(payment) ? "wechat" : payment.Trim(),
            PayerNote = note?.Trim() ?? ""
        });
        SetStatus($"升级申请已提交：{created.TargetPlan}，状态 {created.Status}。");
    }

    private async System.Threading.Tasks.Task LoadProjectsAsync()
    {
        if (_currentUser == null) return;
        try
        {
            if (_refresh != null) _refresh.Enabled = false;
            SetStatus("正在读取你的作品。");
            var projects = await _client.GetProjectsAsync();
            _projectItems.Clear();
            _projectItems.AddRange(projects);
            _adapter?.Clear();
            foreach (var project in _projectItems)
            {
                var visibility = project.Visibility == "public" ? "公开" : "不公开";
                var interactive = project.Interactive ? "互动开" : "互动关";
                var analytics = project.AnalyticsEnabled ? "统计开" : "统计关";
                _adapter?.Add($"{project.Name}\n{project.Slug}，{visibility}，{interactive}，{analytics}");
            }
            SetStatus(_projectItems.Count == 0 ? "还没有作品。可以点击创建作品。" : $"已读取 {_projectItems.Count} 个作品。点击作品可操作。");
        }
        catch (System.Exception ex) { ShowError(ex); }
        finally { if (_refresh != null) _refresh.Enabled = _currentUser != null; }
    }

    private void UpdateAccountLabel()
    {
        if (_account == null) return;
        if (_currentUser == null)
        {
            _account.Text = "未登录";
        }
        else
        {
            var name = string.IsNullOrWhiteSpace(_currentUser.Username) ? _currentUser.Email : _currentUser.Username;
            _account.Text = "当前账号：" + name;
        }
        _account.ContentDescription = _account.Text;
    }

    private void SetMainActionsEnabled(bool enabled)
    {
        if (_createProject != null) _createProject.Enabled = enabled;
        if (_refresh != null) _refresh.Enabled = enabled;
        if (_logout != null) _logout.Enabled = enabled;
        if (_projects != null) _projects.Enabled = enabled;
    }
}
