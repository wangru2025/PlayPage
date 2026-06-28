using System;
using System.Drawing;
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
        _projects.ContextMenuStrip = menu;

        _status.Items.Add(_statusText);
        SetStatus("请输入邮箱，发送验证码后登录。", false);

        root.Controls.Add(top, 0, 0);
        root.Controls.Add(_projects, 0, 1);
        root.Controls.Add(_status, 0, 2);
        Controls.Add(root);
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
