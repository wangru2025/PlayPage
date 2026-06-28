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
    private readonly TextBox _password = new TextBox();
    private readonly Button _login = new Button();
    private readonly ListBox _projects = new ListBox();
    private readonly StatusStrip _status = new StatusStrip();
    private readonly ToolStripStatusLabel _statusText = new ToolStripStatusLabel();

    public MainForm()
    {
        Text = "PlayPage 客户端";
        StartPosition = FormStartPosition.CenterScreen;
        MinimumSize = new Size(760, 520);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "PlayPage 客户端主窗口";

        var root = new TableLayoutPanel { Dock = DockStyle.Fill, ColumnCount = 1, RowCount = 3, Padding = new Padding(12) };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        var loginPanel = new TableLayoutPanel { Dock = DockStyle.Top, ColumnCount = 5, AutoSize = true };
        loginPanel.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        loginPanel.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));
        loginPanel.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        loginPanel.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));
        loginPanel.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));

        var emailLabel = new Label { Text = "邮箱(&E)：", AutoSize = true, TextAlign = ContentAlignment.MiddleLeft };
        _email.Dock = DockStyle.Fill;
        _email.AccessibleName = "邮箱";
        var passwordLabel = new Label { Text = "密码(&P)：", AutoSize = true, TextAlign = ContentAlignment.MiddleLeft };
        _password.Dock = DockStyle.Fill;
        _password.UseSystemPasswordChar = true;
        _password.AccessibleName = "密码";
        _login.Text = "登录(&L)";
        _login.AutoSize = true;
        _login.AccessibleName = "登录";
        _login.AccessibleDescription = "登录 PlayPage 并读取我的作品列表";
        _login.Click += async (_, _) => await LoginAsync();

        loginPanel.Controls.Add(emailLabel, 0, 0);
        loginPanel.Controls.Add(_email, 1, 0);
        loginPanel.Controls.Add(passwordLabel, 2, 0);
        loginPanel.Controls.Add(_password, 3, 0);
        loginPanel.Controls.Add(_login, 4, 0);

        _projects.Dock = DockStyle.Fill;
        _projects.AccessibleName = "我的作品列表";
        _projects.AccessibleDescription = "登录后显示当前账号下的 PlayPage 作品";

        _status.Items.Add(_statusText);
        SetStatus("请输入账号密码登录。", false);

        root.Controls.Add(loginPanel, 0, 0);
        root.Controls.Add(_projects, 0, 1);
        root.Controls.Add(_status, 0, 2);
        Controls.Add(root);
    }

    private async Task LoginAsync()
    {
        try
        {
            _login.Enabled = false;
            SetStatus("正在登录。", false);
            await _client.LoginAsync(_email.Text, _password.Text);
            SetStatus("登录成功，正在读取作品。", false);
            var projects = await _client.GetProjectsAsync();
            _projects.Items.Clear();
            foreach (var project in projects) _projects.Items.Add($"{project.Name}（{project.Slug}）");
            SetStatus($"已读取 {projects.Count} 个作品。", false);
            _projects.Focus();
        }
        catch (Exception ex)
        {
            SetStatus("操作失败：" + ex.Message, true);
            MessageBox.Show(this, ex.Message, "PlayPage", MessageBoxButtons.OK, MessageBoxIcon.Error);
        }
        finally
        {
            _login.Enabled = true;
        }
    }

    private void SetStatus(string text, bool assertive)
    {
        _statusText.Text = text;
        _status.AccessibleName = text;
        if (assertive) System.Media.SystemSounds.Exclamation.Play();
    }
}
