using System;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using System.Threading.Tasks;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

public sealed class MainForm : Form
{
    private readonly PlayPageApiClient _client = new PlayPageApiClient(PlayPageOptions.Production);
    private readonly TextBox _email = new TextBox();
    private readonly TextBox _username = new TextBox();
    private readonly TextBox _code = new TextBox();
    private readonly Button _sendCode = new Button();
    private readonly Button _verify = new Button();
    private readonly Button _refresh = new Button();
    private readonly ListBox _projects = new ListBox();
    private readonly StatusStrip _status = new StatusStrip();
    private readonly ToolStripStatusLabel _statusText = new ToolStripStatusLabel();

    public MainForm()
    {
        Text = "PlayPage 客户端";
        StartPosition = FormStartPosition.CenterScreen;
        MinimumSize = new Size(900, 560);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "PlayPage 客户端主窗口";

        var root = new TableLayoutPanel { Dock = DockStyle.Fill, ColumnCount = 1, RowCount = 3, Padding = new Padding(12) };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        var top = new TableLayoutPanel { Dock = DockStyle.Top, ColumnCount = 9, AutoSize = true };
        top.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        top.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 30));
        top.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        top.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 25));
        top.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        top.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 20));
        top.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        top.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        top.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));

        _email.AccessibleName = "邮箱";
        _username.AccessibleName = "用户名，新账号注册时使用";
        _code.AccessibleName = "邮箱验证码";
        _sendCode.Text = "发送验证码(&S)";
        _verify.Text = "登录(&L)";
        _refresh.Text = "刷新作品(&R)";
        _sendCode.AccessibleName = "发送验证码";
        _verify.AccessibleName = "登录";
        _refresh.AccessibleName = "刷新作品";
        _sendCode.Click += async (_, _) => await SendCodeAsync();
        _verify.Click += async (_, _) => await VerifyCodeAsync();
        _refresh.Click += async (_, _) => await LoadProjectsAsync();

        top.Controls.Add(new Label { Text = "邮箱(&E)：", AutoSize = true }, 0, 0);
        top.Controls.Add(_email, 1, 0);
        top.Controls.Add(new Label { Text = "用户名(&U)：", AutoSize = true }, 2, 0);
        top.Controls.Add(_username, 3, 0);
        top.Controls.Add(new Label { Text = "验证码(&C)：", AutoSize = true }, 4, 0);
        top.Controls.Add(_code, 5, 0);
        top.Controls.Add(_sendCode, 6, 0);
        top.Controls.Add(_verify, 7, 0);
        top.Controls.Add(_refresh, 8, 0);

        _projects.Dock = DockStyle.Fill;
        _projects.AccessibleName = "我的作品列表";
        _projects.AccessibleDescription = "登录后显示当前账号下的 PlayPage 作品";
        _projects.DoubleClick += async (_, _) => await ShowProjectDetailsAsync();

        var menu = new ContextMenuStrip();
        menu.Items.Add("打开作品", null, (_, _) => OpenSelectedProject());
        menu.Items.Add("查看详情", null, async (_, _) => await ShowProjectDetailsAsync());
        menu.Items.Add("显示/隐藏到广场", null, async (_, _) => await ToggleVisibilityAsync());
        menu.Items.Add("查看修复申请", null, async (_, _) => await ShowRepairRequestsAsync());
        menu.Items.Add("查看独立网址申请", null, async (_, _) => await ShowDomainsAsync());
        menu.Items.Add("查看历史版本", null, async (_, _) => await ShowReleasesAsync());
        menu.Items.Add("上传新版本", null, async (_, _) => await UploadReleaseAsync());
        menu.Items.Add("下载作品源码", null, async (_, _) => await DownloadSourceAsync());
        menu.Items.Add("作品设置", null, async (_, _) => await EditSettingsAsync());
        menu.Items.Add("申请修复", null, async (_, _) => await CreateRepairAsync());
        menu.Items.Add("申请独立网址", null, async (_, _) => await CreateDomainAsync());
        menu.Items.Add("统计数据", null, async (_, _) => await ShowStatsAsync());
        menu.Items.Add("导出数据表", null, async (_, _) => await ExportDataAsync());
        _projects.ContextMenuStrip = menu;

        _status.Items.Add(_statusText);
        SetStatus("请输入邮箱，发送验证码后登录。", false);

        root.Controls.Add(top, 0, 0);
        root.Controls.Add(_projects, 0, 1);
        root.Controls.Add(_status, 0, 2);
        var mainMenu = new MenuStrip();
        var fileMenu = new ToolStripMenuItem("文件(&F)");
        fileMenu.DropDownItems.Add("新建作品(&N)", null, async (_, _) => await CreateProjectAsync());
        fileMenu.DropDownItems.Add("刷新作品(&R)", null, async (_, _) => await LoadProjectsAsync());
        fileMenu.DropDownItems.Add("模板市场(&T)", null, async (_, _) => await ShowTemplatesAsync());
        fileMenu.DropDownItems.Add("投稿模板(&M)", null, async (_, _) => await SubmitTemplateAsync());
        fileMenu.DropDownItems.Add("退出(&X)", null, (_, _) => Close());
        var projectMenu = new ToolStripMenuItem("作品(&P)");
        projectMenu.DropDownItems.Add("上传新版本(&U)", null, async (_, _) => await UploadReleaseAsync());
        projectMenu.DropDownItems.Add("下载作品源码(&D)", null, async (_, _) => await DownloadSourceAsync());
        projectMenu.DropDownItems.Add("作品设置(&S)", null, async (_, _) => await EditSettingsAsync());
        projectMenu.DropDownItems.Add("申请修复(&R)", null, async (_, _) => await CreateRepairAsync());
        projectMenu.DropDownItems.Add("启动/查看 AI 圆桌(&A)", null, async (_, _) => await StartOrShowRepairAIAsync());
        projectMenu.DropDownItems.Add("AI 修复预览(&V)", null, async (_, _) => await PreviewRepairAIAsync());
        projectMenu.DropDownItems.Add("发布 AI 修复(&P)", null, async (_, _) => await PublishRepairAIAsync());
        projectMenu.DropDownItems.Add("申请独立网址(&I)", null, async (_, _) => await CreateDomainAsync());
        projectMenu.DropDownItems.Add("统计数据(&T)", null, async (_, _) => await ShowStatsAsync());
        projectMenu.DropDownItems.Add("导出数据表(&E)", null, async (_, _) => await ExportDataAsync());
        mainMenu.Items.Add(fileMenu);
        var adminMenu = new ToolStripMenuItem("管理(&A)");
        adminMenu.DropDownItems.Add("管理摘要(&S)", null, async (_, _) => await ShowAdminSummaryAsync());
        adminMenu.DropDownItems.Add("升级申请(&U)", null, async (_, _) => await ShowAdminUpgradeRequestsAsync());
        adminMenu.DropDownItems.Add("独立网址申请(&D)", null, async (_, _) => await ShowAdminDomainRequestsAsync());
        adminMenu.DropDownItems.Add("修复申请(&R)", null, async (_, _) => await ShowAdminRepairRequestsAsync());
        adminMenu.DropDownItems.Add("模板投稿(&T)", null, async (_, _) => await ShowAdminTemplateSubmissionsAsync());
        mainMenu.Items.Add(projectMenu);
        mainMenu.Items.Add(adminMenu);
        MainMenuStrip = mainMenu;
        Controls.Add(root);
        Controls.Add(mainMenu);
    }

    private async Task SendCodeAsync()
    {
        try
        {
            _sendCode.Enabled = false;
            SetStatus("正在发送验证码。", false);
            var result = await _client.RequestCodeAsync(_email.Text, _username.Text);
            SetStatus($"验证码已发送到 {result.Email}，10 分钟内有效。", false);
            _code.Focus();
        }
        catch (Exception ex) { ShowError(ex); }
        finally { _sendCode.Enabled = true; }
    }

    private async Task VerifyCodeAsync()
    {
        try
        {
            _verify.Enabled = false;
            SetStatus("正在登录。", false);
            var result = await _client.VerifyCodeAsync(_email.Text, _code.Text);
            SetStatus($"已登录：{result.User.Username}。正在读取作品。", false);
            await LoadProjectsAsync();
        }
        catch (Exception ex) { ShowError(ex); }
        finally { _verify.Enabled = true; }
    }

    private async Task LoadProjectsAsync()
    {
        try
        {
            _refresh.Enabled = false;
            var projects = await _client.GetProjectsAsync();
            _projects.Items.Clear();
            foreach (var project in projects) _projects.Items.Add(new ProjectListItem(project));
            SetStatus($"已读取 {projects.Count} 个作品。右键作品可操作。", false);
            _projects.Focus();
        }
        catch (Exception ex) { ShowError(ex); }
        finally { _refresh.Enabled = true; }
    }

    private ProjectSummary? SelectedProject() => _projects.SelectedItem is ProjectListItem item ? item.Project : null;

    private void OpenSelectedProject()
    {
        var p = SelectedProject();
        if (p == null || string.IsNullOrWhiteSpace(p.PublicUrl)) return;
        System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo { FileName = p.PublicUrl, UseShellExecute = true });
    }

    private async Task ShowProjectDetailsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var detail = await _client.GetProjectAsync(p.Id);
        MessageBox.Show(this, $"作品名：{detail.Project.Name}\n地址：{detail.Project.PublicUrl}\n互动功能：{(detail.Project.Interactive ? "开" : "关")}\n统计功能：{(detail.Project.AnalyticsEnabled ? "开" : "关")}\n可见性：{detail.Project.Visibility}", "作品详情");
    }

    private async Task ToggleVisibilityAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var next = p.Visibility == "public" ? "unlisted" : "public";
        await _client.UpdateProjectVisibilityAsync(p.Id, next);
        await LoadProjectsAsync();
    }

    private async Task ShowRepairRequestsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var items = await _client.ListRepairRequestsAsync(p.Id);
        MessageBox.Show(this, items.Count == 0 ? "暂无修复申请。" : string.Join("\n\n", items), "修复申请");
    }

    private async Task ShowDomainsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var items = await _client.ListDomainsAsync(p.Id);
        MessageBox.Show(this, items.Count == 0 ? "暂无独立网址申请。" : string.Join("\n", items), "独立网址");
    }

    private async Task ShowReleasesAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var items = await _client.ListReleasesAsync(p.Id);
        MessageBox.Show(this, items.Count == 0 ? "暂无历史版本。" : string.Join("\n", items), "历史版本");
    }

    private async Task CreateProjectAsync()
    {
        var name = Prompt.Show(this, "新建作品", "作品名称：");
        if (string.IsNullOrWhiteSpace(name)) return;
        var slug = Prompt.Show(this, "新建作品", "作品链接名，只能用字母数字短横线，留空则自动生成：");
        var interactive = MessageBox.Show(this, "是否开启互动功能？", "新建作品", MessageBoxButtons.YesNo) == DialogResult.Yes;
        var analytics = MessageBox.Show(this, "是否开启访问量统计？", "新建作品", MessageBoxButtons.YesNo) == DialogResult.Yes;
        await _client.CreateProjectAsync(new ProjectCreateRequest { Name = name, Slug = slug, Interactive = interactive, AnalyticsEnabled = analytics });
        await LoadProjectsAsync();
    }

    private async Task UploadReleaseAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        using var dialog = new OpenFileDialog { Title = "选择 HTML 或 ZIP 文件", Filter = "网页文件|*.html;*.htm;*.zip|所有文件|*.*" };
        if (dialog.ShowDialog(this) != DialogResult.OK) return;
        var ext = System.IO.Path.GetExtension(dialog.FileName).ToLowerInvariant();
        var mode = ext == ".zip" ? "zip" : "html";
        var note = Prompt.Show(this, "上传新版本", "更新内容，可留空：");
        await _client.UploadReleaseFileAsync(p.Id, dialog.FileName, mode, note);
        SetStatus("新版本已上传。", false);
        await LoadProjectsAsync();
    }

    private async Task DownloadSourceAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var file = await _client.DownloadProjectSourceAsync(p.Id);
        using var dialog = new SaveFileDialog { Title = "保存作品源码", FileName = file.FileName };
        if (dialog.ShowDialog(this) != DialogResult.OK) return;
        await System.IO.File.WriteAllBytesAsync(dialog.FileName, file.Content);
        SetStatus("作品源码已保存。", false);
    }

    private async Task EditSettingsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var name = Prompt.Show(this, "作品设置", "作品名称：", p.Name);
        if (string.IsNullOrWhiteSpace(name)) return;
        var slug = Prompt.Show(this, "作品设置", "链接名：", p.Slug);
        if (string.IsNullOrWhiteSpace(slug)) return;
        var interactive = MessageBox.Show(this, "是否开启互动功能？", "作品设置", MessageBoxButtons.YesNo) == DialogResult.Yes;
        var analytics = MessageBox.Show(this, "是否开启统计功能？", "作品设置", MessageBoxButtons.YesNo) == DialogResult.Yes;
        await _client.UpdateProjectSettingsAsync(p.Id, new ProjectSettingsRequest { Name = name, Slug = slug, Interactive = interactive, AnalyticsEnabled = analytics });
        await LoadProjectsAsync();
    }

    private async Task CreateRepairAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var desc = Prompt.Show(this, "申请修复", "请描述遇到的问题：");
        if (string.IsNullOrWhiteSpace(desc)) return;
        var expected = Prompt.Show(this, "申请修复", "你期望修成什么样，可留空：");
        await _client.CreateRepairRequestAsync(p.Id, new RepairRequestCreateRequest { IssueType = "other", Description = desc, Expected = expected, AllowAdminEdit = true, Contact = _email.Text });
        SetStatus("修复申请已提交。", false);
    }

    private async Task CreateDomainAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var subdomain = Prompt.Show(this, "申请独立网址", "请输入子域名，例如 my-game：");
        if (string.IsNullOrWhiteSpace(subdomain)) return;
        var created = await _client.CreateDomainRequestAsync(p.Id, subdomain);
        SetStatus($"独立网址申请已提交：{created.Domain}", false);
    }

    private async Task ShowStatsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var stats = await _client.GetProjectStatsAsync(p.Id);
        MessageBox.Show(this, $"访问量：{stats.TotalPageViews}\nAPI 请求：{stats.TotalApiRequests}\n成功：{stats.TotalApiSuccesses}\n失败：{stats.TotalApiFailures}\n成功率：{stats.ApiSuccessRate:P2}", "统计数据");
    }

    private async Task ExportDataAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var collections = await _client.ListCollectionsAsync(p.Id);
        if (collections.Count == 0)
        {
            MessageBox.Show(this, "这个作品还没有数据表。", "导出数据表");
            return;
        }
        var names = new List<string>();
        foreach (var item in collections) names.Add(item.Name);
        var format = MessageBox.Show(this, "是否导出为 Word 表格？点“否”则导出 JSON。", "导出数据表", MessageBoxButtons.YesNo) == DialogResult.Yes ? "word" : "json";
        var file = await _client.ExportProjectDataAsync(p.Id, names, format);
        using var dialog = new SaveFileDialog { Title = "保存数据表导出", FileName = file.FileName };
        if (dialog.ShowDialog(this) != DialogResult.OK) return;
        await System.IO.File.WriteAllBytesAsync(dialog.FileName, file.Content);
        SetStatus("数据表已导出。", false);
    }
    private async Task<RepairRequestInfo?> FirstRepairRequestAsync(ProjectSummary project)
    {
        var repairs = await _client.ListRepairRequestsAsync(project.Id);
        if (repairs.Count == 0)
        {
            MessageBox.Show(this, "这个作品还没有修复申请。", "AI 圆桌");
            return null;
        }
        return repairs[0];
    }

    private async Task StartOrShowRepairAIAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var repair = await FirstRepairRequestAsync(p);
        if (repair == null) return;
        try
        {
            var latest = await _client.GetLatestRepairAIAsync(p.Id, repair.Id);
            ShowAIState(latest);
        }
        catch
        {
            if (MessageBox.Show(this, "还没有 AI 圆桌记录，是否立即启动？", "AI 圆桌", MessageBoxButtons.YesNo) != DialogResult.Yes) return;
            var started = await _client.StartRepairAIAsync(p.Id, repair.Id);
            ShowAIState(started);
        }
    }

    private async Task PreviewRepairAIAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var repair = await FirstRepairRequestAsync(p);
        if (repair == null) return;
        var state = await _client.CreateRepairAIPreviewAsync(p.Id, repair.Id);
        if (!string.IsNullOrWhiteSpace(state.Url))
        {
            System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo { FileName = state.Url, UseShellExecute = true });
        }
        ShowAIState(state);
    }

    private async Task PublishRepairAIAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var repair = await FirstRepairRequestAsync(p);
        if (repair == null) return;
        if (MessageBox.Show(this, "确认发布 AI 修复版本？", "发布 AI 修复", MessageBoxButtons.YesNo) != DialogResult.Yes) return;
        var state = await _client.PublishRepairAIAsync(p.Id, repair.Id);
        ShowAIState(state);
        await LoadProjectsAsync();
    }

    private void ShowAIState(RepairAIState state)
    {
        var lines = new List<string>();
        lines.Add($"状态：{state.Job.Status}；第 {state.Job.Round} 轮");
        if (!string.IsNullOrWhiteSpace(state.Job.ErrorMessage)) lines.Add("错误：" + state.Job.ErrorMessage);
        if (!string.IsNullOrWhiteSpace(state.Job.PreviewUrl)) lines.Add("预览：" + state.Job.PreviewUrl);
        if (!string.IsNullOrWhiteSpace(state.Message)) lines.Add(state.Message);
        lines.Add("");
        foreach (var message in state.Messages.TakeLast(20))
        {
            lines.Add($"{message.MessageSeq}. {message.AgentName}");
            lines.Add(message.Content);
            lines.Add("");
        }
        MessageBox.Show(this, string.Join("\n", lines), "AI 圆桌");
    }
    private async Task ShowTemplatesAsync()
    {
        var items = await _client.ListTemplatesAsync();
        MessageBox.Show(this, items.Count == 0 ? "暂无模板。" : string.Join("\n", items.Select(t => $"{t.Name} - {t.Summary}")), "模板市场");
    }

    private async Task SubmitTemplateAsync()
    {
        var name = Prompt.Show(this, "投稿模板", "模板名称：");
        if (string.IsNullOrWhiteSpace(name)) return;
        var slug = Prompt.Show(this, "投稿模板", "模板链接名：");
        if (string.IsNullOrWhiteSpace(slug)) return;
        var summary = Prompt.Show(this, "投稿模板", "一句话简介：");
        var description = Prompt.Show(this, "投稿模板", "详细说明：");
        using var dialog = new OpenFileDialog { Title = "选择模板 HTML 文件", Filter = "HTML 文件|*.html;*.htm|所有文件|*.*" };
        if (dialog.ShowDialog(this) != DialogResult.OK) return;
        var html = await System.IO.File.ReadAllTextAsync(dialog.FileName);
        await _client.CreateTemplateSubmissionAsync(new TemplateSubmissionCreateRequest
        {
            Name = name,
            Slug = slug,
            Summary = summary,
            Description = description,
            HtmlSource = html,
            SourceType = "html",
            Category = "community",
            CategoryLabel = "社区投稿"
        });
        SetStatus("模板投稿已提交，等待管理员审核。", false);
    }

    private async Task ShowAdminSummaryAsync()
    {
        var users = await _client.AdminListUsersAsync();
        var projects = await _client.AdminListProjectsAsync();
        var upgrades = await _client.AdminListUpgradeRequestsAsync();
        var repairs = await _client.AdminListRepairRequestsAsync();
        MessageBox.Show(this, $"用户：{users.Count}\n作品：{projects.Count}\n升级申请：{upgrades.Count}\n修复申请：{repairs.Count}", "管理摘要");
    }

    private async Task ShowAdminUpgradeRequestsAsync()
    {
        var items = await _client.AdminListUpgradeRequestsAsync();
        var pending = items.Where(x => x.Status == "pending").ToList();
        if (pending.Count == 0) { MessageBox.Show(this, "没有待处理升级申请。", "升级申请"); return; }
        var first = pending[0];
        var approve = MessageBox.Show(this, $"处理第一条待审核申请？\n用户：{first.UserEmail}\n目标套餐：{first.TargetPlan}\n备注：{first.PayerNote}\n\n点是通过，点否取消。", "升级申请", MessageBoxButtons.YesNo) == DialogResult.Yes;
        if (!approve) return;
        await _client.AdminReviewUpgradeRequestAsync(first.Id, new UpgradeRequestReviewRequest { Status = "approved", TargetPlan = first.TargetPlan, AdminNote = "客户端审核通过" });
        SetStatus("升级申请已通过。", false);
    }

    private async Task ShowAdminDomainRequestsAsync()
    {
        var items = await _client.AdminListProjectDomainsAsync("pending");
        if (items.Count == 0) { MessageBox.Show(this, "没有待处理独立网址申请。", "独立网址申请"); return; }
        MessageBox.Show(this, string.Join("\n", items.Select(x => $"{x.Domain} - {x.OwnerEmail} - {x.ProjectName}")), "独立网址申请");
    }

    private async Task ShowAdminRepairRequestsAsync()
    {
        var items = await _client.AdminListRepairRequestsAsync("pending");
        if (items.Count == 0) { MessageBox.Show(this, "没有待处理修复申请。", "修复申请"); return; }
        MessageBox.Show(this, string.Join("\n\n", items.Select(x => $"{x.ProjectName}\n{x.OwnerEmail}\n{x.Description}")), "修复申请");
    }

    private async Task ShowAdminTemplateSubmissionsAsync()
    {
        var items = await _client.AdminListTemplateSubmissionsAsync("pending");
        if (items.Count == 0) { MessageBox.Show(this, "没有待审核模板投稿。", "模板投稿"); return; }
        var first = items[0];
        var publish = MessageBox.Show(this, $"发布第一条模板投稿？\n{first.Name}\n作者：{first.AuthorName}\n{first.Summary}", "模板投稿", MessageBoxButtons.YesNo) == DialogResult.Yes;
        if (!publish) return;
        await _client.AdminReviewTemplateSubmissionAsync(first.Id, new TemplateSubmissionReviewRequest { Status = "published", AdminNote = "客户端审核发布" });
        SetStatus("模板投稿已发布。", false);
    }
    private void ShowError(Exception ex)
    {
        SetStatus("操作失败：" + ex.Message, true);
        MessageBox.Show(this, ex.Message, "PlayPage", MessageBoxButtons.OK, MessageBoxIcon.Error);
    }

    private void SetStatus(string text, bool assertive)
    {
        _statusText.Text = text;
        _status.AccessibleName = text;
        if (assertive) System.Media.SystemSounds.Exclamation.Play();
    }

    private sealed class ProjectListItem
    {
        public ProjectSummary Project { get; }
        public ProjectListItem(ProjectSummary project) { Project = project; }
        public override string ToString() => $"{Project.Name}（{Project.Slug}，{Project.Visibility}）";
    }
}
internal static class Prompt
{
    public static string Show(IWin32Window owner, string title, string label, string defaultValue = "")
    {
        using var form = new Form { Text = title, StartPosition = FormStartPosition.CenterParent, Width = 520, Height = 160, MinimizeBox = false, MaximizeBox = false };
        var textLabel = new Label { Text = label, Left = 12, Top = 12, Width = 480, AutoSize = true };
        var input = new TextBox { Left = 12, Top = 40, Width = 480, Text = defaultValue, AccessibleName = label };
        var ok = new Button { Text = "确定", Left = 320, Width = 80, Top = 76, DialogResult = DialogResult.OK };
        var cancel = new Button { Text = "取消", Left = 412, Width = 80, Top = 76, DialogResult = DialogResult.Cancel };
        form.Controls.AddRange(new Control[] { textLabel, input, ok, cancel });
        form.AcceptButton = ok;
        form.CancelButton = cancel;
        return form.ShowDialog(owner) == DialogResult.OK ? input.Text.Trim() : "";
    }
}




