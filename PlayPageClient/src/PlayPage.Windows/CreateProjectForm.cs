using System;
using System.Drawing;
using System.IO;
using System.Windows.Forms;

namespace PlayPage.Windows;

internal enum CreateProjectUploadMode
{
    None,
    File,
    HtmlText
}

internal sealed class CreateProjectFormResult
{
    public string Name { get; set; } = "";
    public string Slug { get; set; } = "";
    public bool Interactive { get; set; }
    public bool AnalyticsEnabled { get; set; }
    public CreateProjectUploadMode UploadMode { get; set; }
    public string FilePath { get; set; } = "";
    public string HtmlText { get; set; } = "";
    public string ChangeNote { get; set; } = "";
}

internal sealed class CreateProjectForm : Form
{
    private readonly TextBox _name = new TextBox();
    private readonly TextBox _slug = new TextBox();
    private readonly CheckBox _interactive = new CheckBox();
    private readonly CheckBox _analytics = new CheckBox();
    private readonly RadioButton _modeNone = new RadioButton();
    private readonly RadioButton _modeFile = new RadioButton();
    private readonly RadioButton _modeText = new RadioButton();
    private readonly TextBox _filePath = new TextBox();
    private readonly Button _chooseFile = new Button();
    private readonly TextBox _htmlText = new TextBox();
    private readonly TextBox _changeNote = new TextBox();
    private readonly Label _status = new Label();
    private bool _slugTouched;

    public CreateProjectFormResult? Result { get; private set; }

    public CreateProjectForm()
    {
        Text = "创建作品";
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(760, 680);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "创建作品窗口";

        var root = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            Padding = new Padding(14),
            ColumnCount = 1,
            RowCount = 8,
            AutoScroll = true
        };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        var basic = new TableLayoutPanel { Dock = DockStyle.Top, ColumnCount = 2, AutoSize = true };
        basic.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        basic.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        AddLabeledText(basic, 0, "作品名称(&N)：", _name, "作品名称，例如：我的小站");
        AddLabeledText(basic, 1, "作品链接名(&S)：", _slug, "作品链接名，只能用字母、数字、短横线等");

        _name.TextChanged += (_, _) =>
        {
            if (!_slugTouched) _slug.Text = NormalizeSlug(_name.Text);
        };
        _slug.TextChanged += (_, _) => _slugTouched = true;

        _interactive.Text = "启用互动功能(&I)";
        _interactive.AccessibleName = "启用互动功能";
        _interactive.AutoSize = true;
        _analytics.Text = "启用访问量统计(&A)";
        _analytics.AccessibleName = "启用访问量统计";
        _analytics.AutoSize = true;

        var featureBox = new GroupBox { Text = "功能选项", Dock = DockStyle.Top, AutoSize = true };
        var featureLayout = new FlowLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, FlowDirection = FlowDirection.TopDown };
        featureLayout.Controls.Add(_interactive);
        featureLayout.Controls.Add(new Label { Text = "互动功能会启用作品数据接口，适合评论、留言、论坛、云存档等作品。", AutoSize = true });
        featureLayout.Controls.Add(_analytics);
        featureLayout.Controls.Add(new Label { Text = "访问量统计会在作品中加入统计代码，用于访问量和互动 API 请求统计。", AutoSize = true });
        featureBox.Controls.Add(featureLayout);

        var uploadBox = new GroupBox { Text = "上传方式", Dock = DockStyle.Top, AutoSize = true };
        var uploadLayout = new TableLayoutPanel { Dock = DockStyle.Top, ColumnCount = 1, AutoSize = true };
        _modeFile.Text = "上传 HTML 或 ZIP 文件(&F)";
        _modeText.Text = "直接粘贴 HTML 代码(&T)";
        _modeNone.Text = "先只创建空作品，以后再上传(&E)";
        _modeFile.Checked = true;
        _modeFile.CheckedChanged += (_, _) => UpdateUploadControls();
        _modeText.CheckedChanged += (_, _) => UpdateUploadControls();
        _modeNone.CheckedChanged += (_, _) => UpdateUploadControls();
        uploadLayout.Controls.Add(_modeFile);

        var fileRow = new TableLayoutPanel { Dock = DockStyle.Top, ColumnCount = 2, AutoSize = true };
        fileRow.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        fileRow.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        _filePath.ReadOnly = true;
        _filePath.AccessibleName = "选择的作品文件";
        _chooseFile.Text = "选择文件(&B)";
        _chooseFile.AutoSize = true;
        _chooseFile.Click += (_, _) => ChooseFile();
        fileRow.Controls.Add(_filePath, 0, 0);
        fileRow.Controls.Add(_chooseFile, 1, 0);
        uploadLayout.Controls.Add(fileRow);

        uploadLayout.Controls.Add(_modeText);
        _htmlText.Multiline = true;
        _htmlText.ScrollBars = ScrollBars.Both;
        _htmlText.WordWrap = false;
        _htmlText.Height = 220;
        _htmlText.AccessibleName = "HTML 代码";
        _htmlText.Text = "<!doctype html>\r\n<html lang=\"zh-CN\">\r\n<head>\r\n  <meta charset=\"utf-8\">\r\n  <title>我的作品</title>\r\n</head>\r\n<body>\r\n  <h1>你好，PlayPage</h1>\r\n</body>\r\n</html>";
        uploadLayout.Controls.Add(_htmlText);
        uploadLayout.Controls.Add(_modeNone);
        uploadBox.Controls.Add(uploadLayout);

        var notePanel = new TableLayoutPanel { Dock = DockStyle.Top, ColumnCount = 2, AutoSize = true };
        notePanel.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        notePanel.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        AddLabeledText(notePanel, 0, "更新内容：", _changeNote, "可选，例如：首次发布");

        _status.AutoSize = true;
        _status.Dock = DockStyle.Fill;
        _status.AccessibleName = "创建状态";

        var buttons = new FlowLayoutPanel { Dock = DockStyle.Bottom, AutoSize = true, FlowDirection = FlowDirection.RightToLeft };
        var ok = new Button { Text = "创建作品(&C)", AutoSize = true, DialogResult = DialogResult.None };
        var cancel = new Button { Text = "取消", AutoSize = true, DialogResult = DialogResult.Cancel };
        ok.Click += (_, _) => Accept();
        buttons.Controls.Add(ok);
        buttons.Controls.Add(cancel);

        root.Controls.Add(new Label { Text = "先填写作品信息，再选择上传方式。", AutoSize = true }, 0, 0);
        root.Controls.Add(basic, 0, 1);
        root.Controls.Add(featureBox, 0, 2);
        root.Controls.Add(uploadBox, 0, 3);
        root.Controls.Add(notePanel, 0, 4);
        root.Controls.Add(new Panel(), 0, 5);
        root.Controls.Add(_status, 0, 6);
        root.Controls.Add(buttons, 0, 7);
        Controls.Add(root);
        CancelButton = cancel;
        UpdateUploadControls();
    }

    protected override void OnShown(EventArgs e)
    {
        base.OnShown(e);
        _name.Focus();
    }

    private static void AddLabeledText(TableLayoutPanel panel, int row, string labelText, TextBox textBox, string accessibleName)
    {
        panel.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        var label = new Label { Text = labelText, AutoSize = true, Anchor = AnchorStyles.Left, Margin = new Padding(0, 6, 8, 6) };
        textBox.Anchor = AnchorStyles.Left | AnchorStyles.Right;
        textBox.AccessibleName = accessibleName;
        panel.Controls.Add(label, 0, row);
        panel.Controls.Add(textBox, 1, row);
    }

    private void ChooseFile()
    {
        using var dialog = new OpenFileDialog
        {
            Title = "选择 HTML 或 ZIP 文件",
            Filter = "网页文件|*.html;*.htm;*.zip|所有文件|*.*"
        };
        if (dialog.ShowDialog(this) == DialogResult.OK) _filePath.Text = dialog.FileName;
    }

    private void UpdateUploadControls()
    {
        var file = _modeFile.Checked;
        var text = _modeText.Checked;
        _filePath.Enabled = file;
        _chooseFile.Enabled = file;
        _htmlText.Enabled = text;
    }

    private void Accept()
    {
        if (string.IsNullOrWhiteSpace(_name.Text))
        {
            SetStatus("请先填写作品名称。");
            _name.Focus();
            return;
        }
        if (string.IsNullOrWhiteSpace(_slug.Text))
        {
            SetStatus("请先填写作品链接名。");
            _slug.Focus();
            return;
        }
        if (_modeFile.Checked && string.IsNullOrWhiteSpace(_filePath.Text))
        {
            SetStatus("请先选择 HTML 或 ZIP 文件。");
            _chooseFile.Focus();
            return;
        }
        if (_modeText.Checked && string.IsNullOrWhiteSpace(_htmlText.Text))
        {
            SetStatus("请先粘贴 HTML 代码。");
            _htmlText.Focus();
            return;
        }

        Result = new CreateProjectFormResult
        {
            Name = _name.Text.Trim(),
            Slug = NormalizeSlug(_slug.Text),
            Interactive = _interactive.Checked,
            AnalyticsEnabled = _analytics.Checked,
            UploadMode = _modeFile.Checked ? CreateProjectUploadMode.File : _modeText.Checked ? CreateProjectUploadMode.HtmlText : CreateProjectUploadMode.None,
            FilePath = _filePath.Text.Trim(),
            HtmlText = _htmlText.Text,
            ChangeNote = _changeNote.Text.Trim()
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

    private static string NormalizeSlug(string value)
    {
        var text = value.Trim().ToLowerInvariant();
        var chars = new System.Text.StringBuilder(text.Length);
        var lastDash = false;
        foreach (var ch in text)
        {
            var keep = char.IsLetterOrDigit(ch) || ch == '_' || ch == '.';
            if (keep)
            {
                chars.Append(ch);
                lastDash = false;
            }
            else if (!lastDash)
            {
                chars.Append('-');
                lastDash = true;
            }
        }
        return chars.ToString().Trim('-', '.', '_');
    }
}
