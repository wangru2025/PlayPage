using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

internal sealed class TemplateCreateProjectResult
{
    public TemplateInfo Template { get; set; } = new TemplateInfo();
    public string Name { get; set; } = "";
    public string Slug { get; set; } = "";
    public bool Interactive { get; set; }
    public bool AnalyticsEnabled { get; set; }
    public string ChangeNote { get; set; } = "";
    public Dictionary<string, string> Params { get; set; } = new Dictionary<string, string>();
}

internal sealed class TemplateCreateProjectForm : Form
{
    private readonly TemplateInfo _template;
    private readonly TextBox _name = new TextBox();
    private readonly TextBox _slug = new TextBox();
    private readonly CheckBox _interactive = new CheckBox();
    private readonly CheckBox _analytics = new CheckBox();
    private readonly TextBox _changeNote = new TextBox();
    private readonly Panel _paramPanel = new Panel();
    private readonly Label _status = new Label();
    private readonly Dictionary<string, Control> _paramControls = new Dictionary<string, Control>();
    private bool _slugTouched;

    public TemplateCreateProjectResult? Result { get; private set; }

    public TemplateCreateProjectForm(TemplateInfo template)
    {
        _template = template;
        Text = "用模板创建作品 - " + template.Name;
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(860, 680);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "用模板创建作品窗口";
        Controls.Add(BuildLayout());
        RenderTemplateDefaults();
    }

    private Control BuildLayout()
    {
        var root = new TableLayoutPanel { Dock = DockStyle.Fill, Padding = new Padding(12), ColumnCount = 1, RowCount = 6 };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        root.Controls.Add(new Label { Text = $"已选择模板：{_template.Name}\n{_template.Summary}", AutoSize = true, MaximumSize = new Size(780, 0) }, 0, 0);

        var project = new TableLayoutPanel { Dock = DockStyle.Top, ColumnCount = 2, AutoSize = true };
        project.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        project.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        AddText(project, 0, "作品名称：", _name, "作品名称");
        AddText(project, 1, "作品链接名：", _slug, "作品链接名");
        AddText(project, 2, "更新内容：", _changeNote, "更新内容，可留空");
        _name.TextChanged += (_, _) => { if (!_slugTouched) _slug.Text = NormalizeSlug(_name.Text); };
        _slug.TextChanged += (_, _) => _slugTouched = true;
        root.Controls.Add(project, 0, 1);

        _interactive.Text = _template.InteractiveRequired ? "启用互动功能（此模板需要，不能关闭）" : "启用互动功能";
        _interactive.AutoSize = true;
        _interactive.Enabled = !_template.InteractiveRequired;
        _analytics.Text = "启用访问量统计";
        _analytics.AutoSize = true;
        var features = new FlowLayoutPanel { Dock = DockStyle.Top, AutoSize = true, FlowDirection = FlowDirection.TopDown };
        features.Controls.Add(_interactive);
        features.Controls.Add(_analytics);
        root.Controls.Add(features, 0, 2);

        root.Controls.Add(new Label { Text = "模板参数", AutoSize = true, Font = new Font(Font, FontStyle.Bold) }, 0, 3);
        _paramPanel.Dock = DockStyle.Fill;
        _paramPanel.AutoScroll = true;
        root.Controls.Add(_paramPanel, 0, 4);

        var bottom = new TableLayoutPanel { Dock = DockStyle.Bottom, ColumnCount = 2, AutoSize = true };
        bottom.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        bottom.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        _status.AutoSize = true;
        _status.AccessibleName = "模板创建状态";
        bottom.Controls.Add(_status, 0, 0);
        var buttons = new FlowLayoutPanel { AutoSize = true, FlowDirection = FlowDirection.RightToLeft };
        var create = new Button { Text = "创建作品", AutoSize = true };
        var close = new Button { Text = "取消", AutoSize = true, DialogResult = DialogResult.Cancel };
        create.Click += (_, _) => Accept();
        buttons.Controls.Add(close);
        buttons.Controls.Add(create);
        bottom.Controls.Add(buttons, 1, 0);
        root.Controls.Add(bottom, 0, 5);
        AcceptButton = create;
        CancelButton = close;
        return root;
    }

    private static void AddText(TableLayoutPanel panel, int row, string label, TextBox box, string accessibleName)
    {
        panel.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        panel.Controls.Add(new Label { Text = label, AutoSize = true, Anchor = AnchorStyles.Left, Margin = new Padding(0, 6, 8, 6) }, 0, row);
        box.Anchor = AnchorStyles.Left | AnchorStyles.Right;
        box.AccessibleName = accessibleName;
        panel.Controls.Add(box, 1, row);
    }

    private void RenderTemplateDefaults()
    {
        _name.Text = _template.Name;
        _interactive.Checked = _template.InteractiveRequired;
        _analytics.Checked = _template.AnalyticsRecommended;
        _paramControls.Clear();
        _paramPanel.Controls.Clear();

        if (_template.ConfigFields.Count == 0)
        {
            _paramPanel.Controls.Add(new Label { Text = "这个模板没有需要填写的参数。", AutoSize = true });
            return;
        }

        var table = new TableLayoutPanel { Dock = DockStyle.Top, AutoSize = true, ColumnCount = 2 };
        table.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        table.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        var row = 0;
        foreach (var field in _template.ConfigFields)
        {
            table.RowStyles.Add(new RowStyle(SizeType.AutoSize));
            var label = new Label { Text = field.Label + (field.Required ? " *" : "") + "：", AutoSize = true, Anchor = AnchorStyles.Left, Margin = new Padding(0, 6, 8, 6) };
            var editor = CreateEditor(field);
            table.Controls.Add(label, 0, row);
            table.Controls.Add(editor, 1, row);
            row++;
            if (!string.IsNullOrWhiteSpace(field.Help))
            {
                table.RowStyles.Add(new RowStyle(SizeType.AutoSize));
                table.Controls.Add(new Label { Text = field.Help, AutoSize = true, MaximumSize = new Size(620, 0) }, 1, row++);
            }
            _paramControls[field.Name] = editor;
        }
        _paramPanel.Controls.Add(table);
    }

    private Control CreateEditor(TemplateConfigField field)
    {
        Control editor;
        if (field.Type == "select")
        {
            var combo = new ComboBox { DropDownStyle = ComboBoxStyle.DropDownList, Anchor = AnchorStyles.Left | AnchorStyles.Right };
            foreach (var option in field.Options) combo.Items.Add(option);
            combo.Text = field.Default;
            if (combo.SelectedIndex < 0 && combo.Items.Count > 0) combo.SelectedIndex = 0;
            editor = combo;
        }
        else if (field.Type == "text")
        {
            editor = new TextBox { Multiline = true, Height = 72, ScrollBars = ScrollBars.Vertical, Text = field.Default, Anchor = AnchorStyles.Left | AnchorStyles.Right };
        }
        else if (field.Type == "color")
        {
            var colorRow = new FlowLayoutPanel { AutoSize = true, FlowDirection = FlowDirection.LeftToRight, Anchor = AnchorStyles.Left | AnchorStyles.Right };
            var text = new TextBox { Text = string.IsNullOrWhiteSpace(field.Default) ? "#ffffff" : field.Default, Width = 120 };
            var pick = new Button { Text = "选择颜色", AutoSize = true };
            pick.Click += (_, _) =>
            {
                using var dialog = new ColorDialog { FullOpen = true };
                if (TryParseColor(text.Text, out var current)) dialog.Color = current;
                if (dialog.ShowDialog(this) == DialogResult.OK) text.Text = $"#{dialog.Color.R:X2}{dialog.Color.G:X2}{dialog.Color.B:X2}";
            };
            colorRow.Controls.Add(text);
            colorRow.Controls.Add(pick);
            editor = colorRow;
        }
        else
        {
            editor = new TextBox { Text = field.Default, Anchor = AnchorStyles.Left | AnchorStyles.Right };
        }
        editor.AccessibleName = field.Label;
        return editor;
    }

    private void Accept()
    {
        if (string.IsNullOrWhiteSpace(_name.Text)) { SetStatus("请填写作品名称。"); _name.Focus(); return; }
        var slug = NormalizeSlug(_slug.Text);
        if (string.IsNullOrWhiteSpace(slug)) { SetStatus("请填写作品链接名。"); _slug.Focus(); return; }

        var values = new Dictionary<string, string>();
        foreach (var field in _template.ConfigFields)
        {
            var value = _paramControls.TryGetValue(field.Name, out var control) ? ControlValue(control) : "";
            if (field.Required && string.IsNullOrWhiteSpace(value)) { SetStatus("请填写模板参数：" + field.Label); control?.Focus(); return; }
            if (field.Type == "color" && !string.IsNullOrWhiteSpace(value) && !Regex.IsMatch(value.Trim(), "^#[0-9a-fA-F]{6}$")) { SetStatus("颜色参数必须是 #RRGGBB 格式：" + field.Label); control?.Focus(); return; }
            values[field.Name] = value;
        }

        Result = new TemplateCreateProjectResult
        {
            Template = _template,
            Name = _name.Text.Trim(),
            Slug = slug,
            Interactive = _template.InteractiveRequired || _interactive.Checked,
            AnalyticsEnabled = _analytics.Checked,
            ChangeNote = _changeNote.Text.Trim(),
            Params = values
        };
        DialogResult = DialogResult.OK;
        Close();
    }

    private static string ControlValue(Control control) => control switch
    {
        TextBox text => text.Text.Trim(),
        ComboBox combo => combo.Text.Trim(),
        FlowLayoutPanel panel when panel.Controls.OfType<TextBox>().FirstOrDefault() is TextBox text => text.Text.Trim(),
        _ => control.Text.Trim()
    };

    private static bool TryParseColor(string value, out Color color)
    {
        color = Color.White;
        value = (value ?? "").Trim();
        if (!Regex.IsMatch(value, "^#[0-9a-fA-F]{6}$")) return false;
        color = ColorTranslator.FromHtml(value);
        return true;
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
            if (char.IsLetterOrDigit(ch) || ch == '_' || ch == '.') { chars.Append(ch); lastDash = false; }
            else if (!lastDash) { chars.Append('-'); lastDash = true; }
        }
        return chars.ToString().Trim('-', '.', '_');
    }
}
