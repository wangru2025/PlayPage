using System.Drawing;
using System.Linq;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

internal sealed class TemplateDetailForm : Form
{
    private readonly TemplateInfo _template;
    public bool UseTemplate { get; private set; }

    public TemplateDetailForm(TemplateInfo template)
    {
        _template = template;
        Text = "模板详情 - " + template.Name;
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(820, 620);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "模板详情窗口";
        Controls.Add(BuildLayout());
    }

    private Control BuildLayout()
    {
        var root = new TableLayoutPanel { Dock = DockStyle.Fill, Padding = new Padding(12), ColumnCount = 1, RowCount = 3 };
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        var text = new TextBox
        {
            Dock = DockStyle.Fill,
            Multiline = true,
            ReadOnly = true,
            ScrollBars = ScrollBars.Vertical,
            Text = BuildText(),
            AccessibleName = "模板详情"
        };
        root.Controls.Add(text, 0, 0);

        var note = new Label
        {
            AutoSize = true,
            Text = "点“使用这个模板”后，会进入单独的创建页面填写作品名、链接名和模板参数。",
            AccessibleName = "操作说明"
        };
        root.Controls.Add(note, 0, 1);

        var buttons = new FlowLayoutPanel { Dock = DockStyle.Bottom, AutoSize = true, FlowDirection = FlowDirection.RightToLeft };
        var close = new Button { Text = "返回模板市场", AutoSize = true, DialogResult = DialogResult.Cancel };
        var use = new Button { Text = "使用这个模板", AutoSize = true };
        use.Click += (_, _) => { UseTemplate = true; DialogResult = DialogResult.OK; Close(); };
        buttons.Controls.Add(close);
        buttons.Controls.Add(use);
        root.Controls.Add(buttons, 0, 2);
        CancelButton = close;
        AcceptButton = use;
        return root;
    }

    private string BuildText()
    {
        var fields = _template.ConfigFields.Count == 0
            ? "无"
            : string.Join("\r\n", _template.ConfigFields.Select(f => $"- {f.Label}（{FieldType(f.Type)}{(f.Required ? "，必填" : "，可选")}）{(string.IsNullOrWhiteSpace(f.Help) ? "" : "：" + f.Help)}"));
        var collections = _template.Collections.Count == 0
            ? "无"
            : string.Join("\r\n", _template.Collections.Select(c => $"- {c.Name}，{c.Fields.Count} 个字段，公开读取：{PlayPageDisplay.YesNo(c.Permissions.PublicRead)}，公开写入：{PlayPageDisplay.YesNo(c.Permissions.PublicWrite)}"));
        return $"模板名称：{_template.Name}\r\n分类：{_template.CategoryLabel}\r\n作者：{_template.AuthorName}\r\n使用次数：{_template.UsageCount}\r\n需要互动功能：{PlayPageDisplay.YesNo(_template.InteractiveRequired)}\r\n建议开启统计：{PlayPageDisplay.YesNo(_template.AnalyticsRecommended)}\r\n标签：{(_template.Tags.Count == 0 ? "无" : string.Join("、", _template.Tags))}\r\n\r\n简介：\r\n{_template.Summary}\r\n\r\n详细说明：\r\n{_template.Description}\r\n\r\n模板参数：\r\n{fields}\r\n\r\n需要的数据集合：\r\n{collections}";
    }

    private static string FieldType(string type) => type switch
    {
        "color" => "颜色",
        "select" => "下拉选择",
        "text" => "多行文本",
        "number" => "数字",
        _ => "文本"
    };
}
