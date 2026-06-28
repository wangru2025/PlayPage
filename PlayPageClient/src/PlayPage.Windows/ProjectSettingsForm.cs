using System.Drawing;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

internal sealed class ProjectSettingsFormResult
{
    public string Name { get; set; } = "";
    public string Slug { get; set; } = "";
    public bool Interactive { get; set; }
    public bool AnalyticsEnabled { get; set; }
}

internal sealed class ProjectSettingsForm : Form
{
    private readonly TextBox _name = new TextBox();
    private readonly TextBox _slug = new TextBox();
    private readonly CheckBox _interactive = new CheckBox();
    private readonly CheckBox _analytics = new CheckBox();
    private readonly Label _status = new Label();

    public ProjectSettingsFormResult? Result { get; private set; }

    public ProjectSettingsForm(ProjectSummary project)
    {
        Text = "作品设置";
        StartPosition = FormStartPosition.CenterParent;
        FormBorderStyle = FormBorderStyle.FixedDialog;
        MinimizeBox = false;
        MaximizeBox = false;
        ClientSize = new Size(560, 260);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "作品设置窗口";

        var root = new TableLayoutPanel { Dock = DockStyle.Fill, Padding = new Padding(14), ColumnCount = 2, RowCount = 6 };
        root.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        root.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        for (var i = 0; i < 6; i++) root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        _name.Text = project.Name;
        _name.AccessibleName = "作品名称";
        _slug.Text = project.Slug;
        _slug.AccessibleName = "作品链接名";
        _interactive.Text = "启用互动功能";
        _interactive.Checked = project.Interactive;
        _interactive.AutoSize = true;
        _analytics.Text = "启用访问量统计";
        _analytics.Checked = project.AnalyticsEnabled;
        _analytics.AutoSize = true;
        _status.AutoSize = true;
        _status.AccessibleName = "设置状态";

        root.Controls.Add(new Label { Text = "作品名称：", AutoSize = true }, 0, 0);
        root.Controls.Add(_name, 1, 0);
        root.Controls.Add(new Label { Text = "作品链接名：", AutoSize = true }, 0, 1);
        root.Controls.Add(_slug, 1, 1);
        root.Controls.Add(_interactive, 1, 2);
        root.Controls.Add(_analytics, 1, 3);
        root.Controls.Add(_status, 0, 4);
        root.SetColumnSpan(_status, 2);

        var buttons = new FlowLayoutPanel { AutoSize = true, FlowDirection = FlowDirection.RightToLeft, Dock = DockStyle.Fill };
        var ok = new Button { Text = "保存", AutoSize = true };
        var cancel = new Button { Text = "取消", AutoSize = true, DialogResult = DialogResult.Cancel };
        ok.Click += (_, _) => Accept();
        buttons.Controls.Add(ok);
        buttons.Controls.Add(cancel);
        root.Controls.Add(buttons, 1, 5);

        Controls.Add(root);
        AcceptButton = ok;
        CancelButton = cancel;
    }

    private void Accept()
    {
        if (string.IsNullOrWhiteSpace(_name.Text))
        {
            SetStatus("请填写作品名称。");
            _name.Focus();
            return;
        }
        if (string.IsNullOrWhiteSpace(_slug.Text))
        {
            SetStatus("请填写作品链接名。");
            _slug.Focus();
            return;
        }
        Result = new ProjectSettingsFormResult
        {
            Name = _name.Text.Trim(),
            Slug = _slug.Text.Trim(),
            Interactive = _interactive.Checked,
            AnalyticsEnabled = _analytics.Checked
        };
        DialogResult = DialogResult.OK;
        Close();
    }

    private void SetStatus(string text)
    {
        _status.Text = text;
        _status.AccessibleName = text;
        System.Media.SystemSounds.Exclamation.Play();
    }
}
