using System;
using System.Collections.Generic;
using System.Drawing;
using System.Threading.Tasks;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

public sealed partial class MainForm : Form
{
    private readonly PlayPageApiClient _client = new PlayPageApiClient(PlayPageOptions.Production);
    private readonly Button _createProject = new Button();
    private readonly Button _refresh = new Button();
    private readonly Button _logout = new Button();
    private readonly Label _userLabel = new Label();
    private readonly ListView _projects = new ListView();
    private readonly StatusStrip _status = new StatusStrip();
    private readonly ToolStripStatusLabel _statusText = new ToolStripStatusLabel();
    private UserProfile? _currentUser;

    public MainForm()
    {
        Text = "PlayPage";
        StartPosition = FormStartPosition.CenterScreen;
        MinimumSize = new Size(980, 620);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "PlayPage 主窗口";
        Load += async (_, _) => await InitializeSessionAsync();

        var mainMenu = BuildMenu();
        var root = BuildRootLayout();

        MainMenuStrip = mainMenu;
        Controls.Add(root);
        Controls.Add(mainMenu);
    }

    private MenuStrip BuildMenu()
    {
        var mainMenu = new MenuStrip();

        var fileMenu = new ToolStripMenuItem("文件(&F)");
        fileMenu.DropDownItems.Add("新建作品(&N)", null, async (_, _) => await CreateProjectAsync());
        fileMenu.DropDownItems.Add("刷新作品(&R)", null, async (_, _) => await LoadProjectsAsync());
        fileMenu.DropDownItems.Add("模板市场(&T)", null, async (_, _) => await ShowTemplatesAsync());
        fileMenu.DropDownItems.Add("投稿模板(&M)", null, async (_, _) => await SubmitTemplateAsync());
        fileMenu.DropDownItems.Add("退出登录(&L)", null, async (_, _) => await LogoutAsync());
        fileMenu.DropDownItems.Add("退出(&X)", null, (_, _) => Close());

        var projectMenu = new ToolStripMenuItem("作品(&P)");
        projectMenu.DropDownItems.Add("打开作品(&O)", null, (_, _) => OpenSelectedProject());
        projectMenu.DropDownItems.Add("查看详情(&D)", null, async (_, _) => await ShowProjectDetailsAsync());
        projectMenu.DropDownItems.Add("上传新版本(&U)", null, async (_, _) => await UploadReleaseAsync());
        projectMenu.DropDownItems.Add("下载作品源码(&S)", null, async (_, _) => await DownloadSourceAsync());
        projectMenu.DropDownItems.Add("作品设置(&G)", null, async (_, _) => await EditSettingsAsync());
        projectMenu.DropDownItems.Add("显示/隐藏到广场(&V)", null, async (_, _) => await ToggleVisibilityAsync());
        projectMenu.DropDownItems.Add("申请修复(&R)", null, async (_, _) => await CreateRepairAsync());
        projectMenu.DropDownItems.Add("查看修复申请(&Q)", null, async (_, _) => await ShowRepairRequestsAsync());
        projectMenu.DropDownItems.Add("申请独立网址(&I)", null, async (_, _) => await CreateDomainAsync());
        projectMenu.DropDownItems.Add("查看独立网址申请(&W)", null, async (_, _) => await ShowDomainsAsync());
        projectMenu.DropDownItems.Add("查看历史版本(&H)", null, async (_, _) => await ShowReleasesAsync());
        projectMenu.DropDownItems.Add("统计数据(&T)", null, async (_, _) => await ShowStatsAsync());
        projectMenu.DropDownItems.Add("管理互动数据(&M)", null, async (_, _) => await ManageInteractiveDataAsync());
        projectMenu.DropDownItems.Add("导出数据表(&E)", null, async (_, _) => await ExportDataAsync());
        projectMenu.DropDownItems.Add("删除作品(&X)", null, async (_, _) => await DeleteProjectAsync());

        var adminMenu = new ToolStripMenuItem("管理(&A)");
        adminMenu.DropDownItems.Add("管理摘要(&S)", null, async (_, _) => await ShowAdminSummaryAsync());
        adminMenu.DropDownItems.Add("升级申请(&U)", null, async (_, _) => await ShowAdminUpgradeRequestsAsync());
        adminMenu.DropDownItems.Add("独立网址申请(&D)", null, async (_, _) => await ShowAdminDomainRequestsAsync());
        adminMenu.DropDownItems.Add("独立网址删除申请(&X)", null, async (_, _) => await ShowAdminDomainDeleteRequestsAsync());
        adminMenu.DropDownItems.Add("修复申请(&R)", null, async (_, _) => await ShowAdminRepairRequestsAsync());
        adminMenu.DropDownItems.Add("模板投稿(&T)", null, async (_, _) => await ShowAdminTemplateSubmissionsAsync());

        mainMenu.Items.Add(fileMenu);
        mainMenu.Items.Add(projectMenu);
        mainMenu.Items.Add(adminMenu);
        return mainMenu;
    }

    private Control BuildRootLayout()
    {
        var root = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            ColumnCount = 1,
            RowCount = 3,
            Padding = new Padding(12)
        };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        var top = new TableLayoutPanel { Dock = DockStyle.Top, ColumnCount = 2, AutoSize = true };
        top.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        top.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));

        _userLabel.AutoSize = true;
        _userLabel.Anchor = AnchorStyles.Left;
        _userLabel.Text = "正在检查登录状态。";
        _userLabel.AccessibleName = "当前登录状态";

        _createProject.Text = "创建作品(&N)";
        _createProject.AccessibleName = "创建作品";
        _createProject.AutoSize = true;
        _createProject.Click += async (_, _) => await CreateProjectAsync();

        _refresh.Text = "刷新作品(&R)";
        _refresh.AccessibleName = "刷新作品";
        _refresh.AutoSize = true;
        _refresh.Click += async (_, _) => await LoadProjectsAsync();

        _logout.Text = "退出登录(&L)";
        _logout.AccessibleName = "退出登录";
        _logout.AutoSize = true;
        _logout.Click += async (_, _) => await LogoutAsync();

        var actions = new FlowLayoutPanel { AutoSize = true, FlowDirection = FlowDirection.LeftToRight, Anchor = AnchorStyles.Right };
        actions.Controls.Add(_createProject);
        actions.Controls.Add(_refresh);
        actions.Controls.Add(_logout);

        top.Controls.Add(_userLabel, 0, 0);
        top.Controls.Add(actions, 1, 0);

        ConfigureProjectList();
        _status.Items.Add(_statusText);
        SetStatus("正在启动 PlayPage。", false);

        root.Controls.Add(top, 0, 0);
        root.Controls.Add(_projects, 0, 1);
        root.Controls.Add(_status, 0, 2);
        return root;
    }

    private void ConfigureProjectList()
    {
        _projects.Dock = DockStyle.Fill;
        _projects.View = View.Details;
        _projects.FullRowSelect = true;
        _projects.MultiSelect = false;
        _projects.HideSelection = false;
        _projects.AccessibleName = "我的作品列表";
        _projects.AccessibleDescription = "登录后显示当前账号下的 PlayPage 作品。选择作品后可通过菜单或右键操作。";
        _projects.Columns.Add("作品名", 220);
        _projects.Columns.Add("链接名", 160);
        _projects.Columns.Add("公开状态", 110);
        _projects.Columns.Add("互动", 80);
        _projects.Columns.Add("统计", 80);
        _projects.Columns.Add("地址", 360);
        _projects.DoubleClick += (_, _) => OpenSelectedProject();

        var menu = new ContextMenuStrip();
        menu.Items.Add("打开作品", null, (_, _) => OpenSelectedProject());
        menu.Items.Add("查看详情", null, async (_, _) => await ShowProjectDetailsAsync());
        menu.Items.Add("上传新版本", null, async (_, _) => await UploadReleaseAsync());
        menu.Items.Add("作品设置", null, async (_, _) => await EditSettingsAsync());
        menu.Items.Add("更多：修复申请", null, async (_, _) => await ShowRepairRequestsAsync());
        menu.Items.Add("更多：独立网址", null, async (_, _) => await ShowDomainsAsync());
        menu.Items.Add("更多：历史版本", null, async (_, _) => await ShowReleasesAsync());
        menu.Items.Add("更多：统计数据", null, async (_, _) => await ShowStatsAsync());
        menu.Items.Add("更多：管理互动数据", null, async (_, _) => await ManageInteractiveDataAsync());
        menu.Items.Add("更多：导出数据表", null, async (_, _) => await ExportDataAsync());
        menu.Items.Add("更多：删除作品", null, async (_, _) => await DeleteProjectAsync());
        _projects.ContextMenuStrip = menu;
    }

    private async Task InitializeSessionAsync()
    {
        SetMainActionsEnabled(false);
        try
        {
            var token = LoadToken();
            if (!string.IsNullOrWhiteSpace(token))
            {
                _client.SetToken(token);
                _currentUser = await _client.GetMeAsync();
                if (!await EnsureProfileAsync()) return;
                UpdateUserLabel();
                SetMainActionsEnabled(true);
                await LoadProjectsAsync();
                return;
            }
        }
        catch
        {
            SaveToken("");
            _client.SetToken(null);
            _currentUser = null;
        }

        await ShowLoginDialogAsync();
    }

    private async Task ShowLoginDialogAsync()
    {
        using var login = new LoginForm(_client);
        if (login.ShowDialog(this) != DialogResult.OK || login.Result == null)
        {
            SetStatus("未登录。请从“文件”菜单选择退出，或重新打开程序登录。", true);
            _userLabel.Text = "未登录";
            SetMainActionsEnabled(false);
            return;
        }

        SaveToken(login.Result.AccessToken);
        _currentUser = login.Result.User;
        if (!await EnsureProfileAsync()) return;
        UpdateUserLabel();
        SetMainActionsEnabled(true);
        await LoadProjectsAsync();
    }

    private async Task LogoutAsync()
    {
        try
        {
            await _client.LogoutAsync();
        }
        catch
        {
            // 本地退出优先，服务器退出失败不阻塞用户重新登录。
        }
        SaveToken("");
        _client.SetToken(null);
        _currentUser = null;
        _projects.Items.Clear();
        SetMainActionsEnabled(false);
        await ShowLoginDialogAsync();
    }

    private static string TokenFilePath()
    {
        var dir = System.IO.Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "PlayPageClient");
        System.IO.Directory.CreateDirectory(dir);
        return System.IO.Path.Combine(dir, "access-token.txt");
    }

    private static string LoadToken()
    {
        var path = TokenFilePath();
        return System.IO.File.Exists(path) ? System.IO.File.ReadAllText(path).Trim() : "";
    }

    private static void SaveToken(string token)
    {
        var path = TokenFilePath();
        if (string.IsNullOrWhiteSpace(token))
        {
            if (System.IO.File.Exists(path)) System.IO.File.Delete(path);
        }
        else
        {
            System.IO.File.WriteAllText(path, token.Trim());
        }
    }

    private async Task LoadProjectsAsync()
    {
        if (_currentUser == null) return;
        try
        {
            _refresh.Enabled = false;
            SetStatus("正在读取你的作品。", false);
            var projects = await _client.GetProjectsAsync();
            _projects.Items.Clear();
            foreach (var project in projects)
            {
                var item = new ListViewItem(project.Name);
                item.SubItems.Add(project.Slug);
                item.SubItems.Add(project.Visibility == "public" ? "公开" : "不公开");
                item.SubItems.Add(project.Interactive ? "开" : "关");
                item.SubItems.Add(project.AnalyticsEnabled ? "开" : "关");
                item.SubItems.Add(project.PublicUrl);
                item.Tag = project;
                _projects.Items.Add(item);
            }
            SetStatus(projects.Count == 0 ? "还没有作品。可以点击“创建作品”。" : $"已读取 {projects.Count} 个作品。", false);
            _projects.Focus();
        }
        catch (Exception ex)
        {
            ShowError(ex);
        }
        finally
        {
            _refresh.Enabled = _currentUser != null;
        }
    }

    private ProjectSummary? SelectedProject()
    {
        if (_projects.SelectedItems.Count == 0) return null;
        return _projects.SelectedItems[0].Tag as ProjectSummary;
    }

    private void OpenSelectedProject()
    {
        var p = SelectedProject();
        if (p == null || string.IsNullOrWhiteSpace(p.PublicUrl)) return;
        System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo { FileName = p.PublicUrl, UseShellExecute = true });
    }


    private async Task<bool> EnsureProfileAsync()
    {
        while (_currentUser != null && string.IsNullOrWhiteSpace(_currentUser.Username))
        {
            var username = Prompt.Show(this, "设置公开名字", "请输入你的公开名字。这个名字会显示在作品、模板和评论旁边：");
            if (string.IsNullOrWhiteSpace(username))
            {
                MessageBox.Show(this, "新账号需要先设置公开名字，才能继续使用 PlayPage。", "PlayPage", MessageBoxButtons.OK, MessageBoxIcon.Information);
                continue;
            }
            try
            {
                _currentUser = await _client.UpdateProfileAsync(username);
                SetStatus("公开名字已保存。", false);
            }
            catch (Exception ex)
            {
                ShowError(ex);
            }
        }
        return _currentUser != null && !string.IsNullOrWhiteSpace(_currentUser.Username);
    }

    private void UpdateUserLabel()
    {
        if (_currentUser == null)
        {
            _userLabel.Text = "未登录";
            return;
        }
        var name = string.IsNullOrWhiteSpace(_currentUser.Username) ? _currentUser.Email : _currentUser.Username;
        _userLabel.Text = $"当前账号：{name}";
    }

    private void SetMainActionsEnabled(bool enabled)
    {
        _createProject.Enabled = enabled;
        _refresh.Enabled = enabled;
        _logout.Enabled = enabled;
        _projects.Enabled = enabled;
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
}
