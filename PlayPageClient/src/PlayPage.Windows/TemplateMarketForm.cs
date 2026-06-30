using System;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

internal sealed class TemplateMarketForm : Form
{
    private readonly IReadOnlyList<TemplateInfo> _templates;
    private readonly ComboBox _category = new ComboBox();
    private readonly TextBox _search = new TextBox();
    private readonly ListBox _templateList = new ListBox();
    private readonly Label _summary = new Label();
    private readonly Label _status = new Label();

    public TemplateInfo? SelectedTemplate { get; private set; }

    public TemplateMarketForm(IReadOnlyList<TemplateInfo> templates)
    {
        _templates = templates;
        Text = "模板市场";
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(900, 620);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "模板市场窗口";
        Controls.Add(BuildLayout());
        RenderCategories();
        RenderList();
    }

    private Control BuildLayout()
    {
        var root = new TableLayoutPanel { Dock = DockStyle.Fill, Padding = new Padding(12), ColumnCount = 1, RowCount = 4 };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        var filters = new TableLayoutPanel { Dock = DockStyle.Top, ColumnCount = 4, AutoSize = true };
        filters.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        filters.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 45));
        filters.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        filters.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 55));
        filters.Controls.Add(new Label { Text = "分类：", AutoSize = true, Anchor = AnchorStyles.Left, Margin = new Padding(0, 6, 8, 6) }, 0, 0);
        _category.DropDownStyle = ComboBoxStyle.DropDownList;
        _category.Anchor = AnchorStyles.Left | AnchorStyles.Right;
        _category.SelectedIndexChanged += (_, _) => RenderList();
        filters.Controls.Add(_category, 1, 0);
        filters.Controls.Add(new Label { Text = "搜索：", AutoSize = true, Anchor = AnchorStyles.Left, Margin = new Padding(12, 6, 8, 6) }, 2, 0);
        _search.Anchor = AnchorStyles.Left | AnchorStyles.Right;
        _search.AccessibleName = "搜索模板";
        _search.TextChanged += (_, _) => RenderList();
        filters.Controls.Add(_search, 3, 0);
        root.Controls.Add(filters, 0, 0);

        var middle = new SplitContainer { Dock = DockStyle.Fill, Orientation = Orientation.Vertical, SplitterDistance = 420 };
        _templateList.Dock = DockStyle.Fill;
        _templateList.AccessibleName = "模板列表";
        _templateList.SelectedIndexChanged += (_, _) => RenderSummary();
        _templateList.DoubleClick += (_, _) => OpenDetail();
        middle.Panel1.Controls.Add(_templateList);
        _summary.Dock = DockStyle.Fill;
        _summary.AutoSize = false;
        _summary.MaximumSize = new Size(420, 0);
        _summary.AccessibleName = "模板摘要";
        middle.Panel2.Controls.Add(_summary);
        root.Controls.Add(middle, 0, 1);

        _status.AutoSize = true;
        _status.AccessibleName = "模板市场状态";
        root.Controls.Add(_status, 0, 2);

        var buttons = new FlowLayoutPanel { Dock = DockStyle.Bottom, AutoSize = true, FlowDirection = FlowDirection.RightToLeft };
        var close = new Button { Text = "关闭", AutoSize = true, DialogResult = DialogResult.Cancel };
        var detail = new Button { Text = "查看模板详情", AutoSize = true };
        detail.Click += (_, _) => OpenDetail();
        buttons.Controls.Add(close);
        buttons.Controls.Add(detail);
        root.Controls.Add(buttons, 0, 3);
        CancelButton = close;
        return root;
    }

    private void RenderCategories()
    {
        _category.Items.Clear();
        _category.Items.Add(new CategoryItem("", "全部分类"));
        foreach (var group in _templates.GroupBy(t => new { t.Category, t.CategoryLabel }).OrderBy(g => g.Key.CategoryLabel))
        {
            _category.Items.Add(new CategoryItem(group.Key.Category, string.IsNullOrWhiteSpace(group.Key.CategoryLabel) ? group.Key.Category : group.Key.CategoryLabel));
        }
        _category.SelectedIndex = 0;
    }

    private void RenderList()
    {
        var selectedCategory = _category.SelectedItem is CategoryItem item ? item.Category : "";
        var query = _search.Text.Trim();
        var items = _templates.Where(t => string.IsNullOrWhiteSpace(selectedCategory) || t.Category == selectedCategory);
        if (!string.IsNullOrWhiteSpace(query))
        {
            items = items.Where(t => Contains(t.Name, query) || Contains(t.Summary, query) || Contains(t.Description, query) || t.Tags.Any(tag => Contains(tag, query)));
        }
        _templateList.Items.Clear();
        foreach (var template in items.OrderBy(t => t.CategoryLabel).ThenBy(t => t.Name)) _templateList.Items.Add(new TemplateListItem(template));
        if (_templateList.Items.Count > 0) _templateList.SelectedIndex = 0;
        else RenderSummary();
        _status.Text = $"找到 {_templateList.Items.Count} 个模板。选择模板后点“查看模板详情”。";
    }

    private void RenderSummary()
    {
        var template = CurrentTemplate();
        if (template == null)
        {
            _summary.Text = "没有符合条件的模板。";
            return;
        }
        _summary.Text = $"{template.Name}\n分类：{template.CategoryLabel}\n作者：{template.AuthorName}\n使用次数：{template.UsageCount}\n简介：{template.Summary}\n\n标签：{(template.Tags.Count == 0 ? "无" : string.Join("、", template.Tags))}\n\n双击或点击“查看模板详情”进入详情页。";
    }

    private void OpenDetail()
    {
        var template = CurrentTemplate();
        if (template == null)
        {
            _status.Text = "请先选择模板。";
            return;
        }
        SelectedTemplate = template;
        DialogResult = DialogResult.OK;
        Close();
    }

    private TemplateInfo? CurrentTemplate() => _templateList.SelectedItem is TemplateListItem item ? item.Template : null;
    private static bool Contains(string value, string query) => (value ?? "").IndexOf(query, StringComparison.OrdinalIgnoreCase) >= 0;

    private sealed class CategoryItem
    {
        public string Category { get; }
        private readonly string _label;
        public CategoryItem(string category, string label) { Category = category; _label = label; }
        public override string ToString() => _label;
    }

    private sealed class TemplateListItem
    {
        public TemplateInfo Template { get; }
        public TemplateListItem(TemplateInfo template) => Template = template;
        public override string ToString() => $"{Template.CategoryLabel}｜{Template.Name}｜{Template.Summary}";
    }
}
