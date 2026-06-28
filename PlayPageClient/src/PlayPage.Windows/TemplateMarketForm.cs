using System;
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

internal sealed class TemplateMarketForm : Form
{
    private readonly IReadOnlyList<TemplateInfo> _templates;
    private readonly ListBox _templateList = new ListBox();
    private readonly TextBox _name = new TextBox();
    private readonly TextBox _slug = new TextBox();
    private readonly CheckBox _interactive = new CheckBox();
    private readonly CheckBox _analytics = new CheckBox();
    private readonly TextBox _changeNote = new TextBox();
    private readonly Panel _paramPanel = new Panel();
    private readonly Label _description = new Label();
    private readonly Label _status = new Label();
    private readonly Dictionary<string, Control> _paramControls = new Dictionary<string, Control>();
    private bool _slugTouched;

    public TemplateCreateProjectResult? Result { get; private set; }

    public TemplateMarketForm(IReadOnlyList<TemplateInfo> templates)
    {
        _templates = templates;
        Text = "模板市场";
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(980, 720);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "模板市场窗口";

        var root = new TableLayoutPanel { Dock = DockStyle.Fill, Padding = new Padding(12), ColumnCount = 2, RowCount = 2 };
        root.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 36));
        root.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 64));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        _templateList.Dock = DockStyle.Fill;
        _templateList.AccessibleName = "模板列表";
        foreach (var template in templates.OrderBy(t => t.CategoryLabel).ThenBy(t => t.Name))
        {
            _templateList.Items.Add(new TemplateListItem(template));
        }
        _templateList.SelectedIndexChanged += (_, _) => RenderSelectedTemplate();
        root.Controls.Add(_templateList, 0, 0);

        var right = new TableLayoutPanel { Dock = DockStyle.Fill, ColumnCount = 1, RowCount = 7, AutoScroll = true };
        right.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        right.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        right.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        right.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        right.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        right.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        right.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        _description.AutoSize = true;
        _description.MaximumSize = new Size(560, 0);
        right.Controls.Add(_description, 0, 0);

        var project = new TableLayoutPanel { Dock = DockStyle.Top, ColumnCount = 2, AutoSize = true };
        project.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        project.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        AddText(project, 0, "作品名称：", _name, "作品名称");
        AddText(project, 1, "作品链接名：", _slug, "作品链接名");
        AddText(project, 2, "更新内容：", _changeNote, "更新内容，可留空");
        _name.TextChanged += (_, _) => { if (!_slugTouched) _slug.Text = NormalizeSlug(_name.Text); };
        _slug.TextChanged += (_, _) => _slugTouched = true;
        right.Controls.Add(project, 0, 1);

        _interactive.Text = "启用互动功能";
        _interactive.AutoSize = true;
        _analytics.Text = "启用访问量统计";
        _analytics.AutoSize = true;
        var features = new FlowLayoutPanel { Dock = DockStyle.Top, AutoSize = true, FlowDirection = FlowDirection.TopDown };
        features.Controls.Add(_interactive);
        features.Controls.Add(_analytics);
        right.Controls.Add(features, 0, 2);

        right.Controls.Add(new Label { Text = "模板参数", AutoSize = true, Font = new Font(Font, FontStyle.Bold) }, 0, 3);
        _paramPanel.Dock = DockStyle.Fill;
        _paramPanel.AutoScroll = true;
        right.Controls.Add(_paramPanel, 0, 5);

        _status.AutoSize = true;
        _status.AccessibleName = "模板创建状态";
        right.Controls.Add(_status, 0, 6);
        root.Controls.Add(right, 1, 0);

        var buttons = new FlowLayoutPanel { Dock = DockStyle.Bottom, AutoSize = true, FlowDirection = FlowDirection.RightToLeft };
        var create = new Button { Text = "使用模板创建作品", AutoSize = true };
        var close = new Button { Text = "关闭", AutoSize = true, DialogResult = DialogResult.Cancel };
        create.Click += (_, _) => Accept();
        buttons.Controls.Add(create);
        buttons.Controls.Add(close);
        root.Controls.Add(buttons, 0, 1);
        root.SetColumnSpan(buttons, 2);

        Controls.Add(root);
        CancelButton = close;
        if (_templateList.Items.Count > 0) _templateList.SelectedIndex = 0;
    }

    private static void AddText(TableLayoutPanel panel, int row, string label, TextBox box, string accessibleName)
    {
        panel.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        panel.Controls.Add(new Label { Text = label, AutoSize = true, Anchor = AnchorStyles.Left, Margin = new Padding(0, 6, 8, 6) }, 0, row);
        box.Anchor = AnchorStyles.Left | AnchorStyles.Right;
        box.AccessibleName = accessibleName;
        panel.Controls.Add(box, 1, row);
    }

    private TemplateInfo? SelectedTemplate() => _templateList.SelectedItem is TemplateListItem item ? item.Template : null;

    private void RenderSelectedTemplate()
    {
        var template = SelectedTemplate();
        _paramControls.Clear();
        _paramPanel.Controls.Clear();
        if (template == null)
        {
            _description.Text = "请选择模板。";
            return;
        }

        _description.Text = $"{template.Name}\n分类：{template.CategoryLabel}\n作者：{template.AuthorName}\n简介：{template.Summary}\n{template.Description}";
        if (string.IsNullOrWhiteSpace(_name.Text)) _name.Text = template.Name;
        _interactive.Checked = template.InteractiveRequired || _interactive.Checked;
        _interactive.Enabled = !template.InteractiveRequired;
        _analytics.Checked = template.AnalyticsRecommended || _analytics.Checked;

        var table = new TableLayoutPanel { Dock = DockStyle.Top, AutoSize = true, ColumnCount = 2 };
        table.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        table.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        var row = 0;
        foreach (var field in template.ConfigFields)
        {
            table.RowStyles.Add(new RowStyle(SizeType.AutoSize));
            var label = new Label { Text = field.Label + (field.Required ? " *" : "") + "：", AutoSize = true, Anchor = AnchorStyles.Left, Margin = new Padding(0, 6, 8, 6) };
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
            else
            {
                editor = new TextBox { Text = field.Default, Anchor = AnchorStyles.Left | AnchorStyles.Right };
            }
            editor.AccessibleName = field.Label;
            table.Controls.Add(label, 0, row);
            table.Controls.Add(editor, 1, row);
            row++;
            if (!string.IsNullOrWhiteSpace(field.Help))
            {
                table.RowStyles.Add(new RowStyle(SizeType.AutoSize));
                var help = new Label { Text = field.Help, AutoSize = true, MaximumSize = new Size(520, 0) };
                table.Controls.Add(help, 1, row++);
            }
            _paramControls[field.Name] = editor;
        }
        _paramPanel.Controls.Add(table);
    }

    private void Accept()
    {
        var template = SelectedTemplate();
        if (template == null)
        {
            SetStatus("请先选择模板。");
            return;
        }
        if (string.IsNullOrWhiteSpace(_name.Text))
        {
            SetStatus("请填写作品名称。");
            _name.Focus();
            return;
        }
        var slug = NormalizeSlug(_slug.Text);
        if (string.IsNullOrWhiteSpace(slug))
        {
            SetStatus("请填写作品链接名。");
            _slug.Focus();
            return;
        }

        var values = new Dictionary<string, string>();
        foreach (var field in template.ConfigFields)
        {
            var value = _paramControls.TryGetValue(field.Name, out var control) ? ControlValue(control) : "";
            if (field.Required && string.IsNullOrWhiteSpace(value))
            {
                SetStatus("请填写模板参数：" + field.Label);
                control?.Focus();
                return;
            }
            if (field.Type == "color" && !string.IsNullOrWhiteSpace(value) && !Regex.IsMatch(value.Trim(), "^#[0-9a-fA-F]{6}$"))
            {
                SetStatus("颜色参数必须是 #RRGGBB 格式：" + field.Label);
                control?.Focus();
                return;
            }
            values[field.Name] = value;
        }

        Result = new TemplateCreateProjectResult
        {
            Template = template,
            Name = _name.Text.Trim(),
            Slug = slug,
            Interactive = template.InteractiveRequired || _interactive.Checked,
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
        _ => control.Text.Trim()
    };

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

    private sealed class TemplateListItem
    {
        public TemplateInfo Template { get; }
        public TemplateListItem(TemplateInfo template) => Template = template;
        public override string ToString() => $"{Template.CategoryLabel}｜{Template.Name}｜{Template.Summary}";
    }
}
