using System;
using System.Collections.Generic;
using System.Drawing;
using System.Text.Json;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

internal sealed class TemplateSubmissionForm : Form
{
    private readonly TextBox _name = new TextBox();
    private readonly TextBox _slug = new TextBox();
    private readonly TextBox _category = new TextBox();
    private readonly TextBox _categoryLabel = new TextBox();
    private readonly TextBox _summary = new TextBox();
    private readonly TextBox _description = new TextBox();
    private readonly TextBox _tags = new TextBox();
    private readonly CheckBox _interactiveRequired = new CheckBox();
    private readonly CheckBox _analyticsRecommended = new CheckBox();
    private readonly TextBox _configFields = new TextBox();
    private readonly TextBox _collections = new TextBox();
    private readonly TextBox _htmlSource = new TextBox();
    private readonly Label _status = new Label();
    private bool _slugTouched;

    public TemplateSubmissionCreateRequest? Result { get; private set; }

    public TemplateSubmissionForm()
    {
        Text = "投稿模板";
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(980, 760);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "投稿模板窗口";

        var root = new TableLayoutPanel { Dock = DockStyle.Fill, Padding = new Padding(12), ColumnCount = 1, RowCount = 5 };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 45));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 55));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        var top = new FlowLayoutPanel { Dock = DockStyle.Top, AutoSize = true, FlowDirection = FlowDirection.LeftToRight };
        var staticPreset = new Button { Text = "套用静态页面示例", AutoSize = true };
        var interactivePreset = new Button { Text = "套用留言板示例", AutoSize = true };
        var loadHtml = new Button { Text = "从 HTML 文件读取", AutoSize = true };
        staticPreset.Click += (_, _) => UseStaticPreset();
        interactivePreset.Click += (_, _) => UseInteractivePreset();
        loadHtml.Click += (_, _) => LoadHtmlFile();
        top.Controls.Add(staticPreset);
        top.Controls.Add(interactivePreset);
        top.Controls.Add(loadHtml);
        root.Controls.Add(top, 0, 0);

        var basic = new TableLayoutPanel { Dock = DockStyle.Top, ColumnCount = 4, AutoSize = true };
        basic.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        basic.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));
        basic.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        basic.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));
        AddText(basic, 0, 0, "模板名称：", _name, "模板名称");
        AddText(basic, 0, 2, "链接名：", _slug, "模板链接名");
        AddText(basic, 1, 0, "分类 ID：", _category, "分类 ID，例如 community");
        AddText(basic, 1, 2, "分类显示名：", _categoryLabel, "分类显示名");
        AddText(basic, 2, 0, "一句话简介：", _summary, "一句话简介");
        basic.SetColumnSpan(_summary, 3);
        AddText(basic, 3, 0, "标签：", _tags, "标签，逗号分隔");
        basic.SetColumnSpan(_tags, 3);
        _description.Multiline = true;
        _description.Height = 72;
        AddText(basic, 4, 0, "详细说明：", _description, "详细说明");
        basic.SetColumnSpan(_description, 3);

        _interactiveRequired.Text = "需要互动功能";
        _interactiveRequired.AutoSize = true;
        _analyticsRecommended.Text = "建议开启统计";
        _analyticsRecommended.AutoSize = true;
        _analyticsRecommended.Checked = true;
        var checks = new FlowLayoutPanel { Dock = DockStyle.Top, AutoSize = true };
        checks.Controls.Add(_interactiveRequired);
        checks.Controls.Add(_analyticsRecommended);
        basic.Controls.Add(checks, 1, 5);
        basic.SetColumnSpan(checks, 3);
        root.Controls.Add(basic, 0, 1);

        _name.TextChanged += (_, _) => { if (!_slugTouched) _slug.Text = NormalizeSlug(_name.Text); };
        _slug.TextChanged += (_, _) => _slugTouched = true;

        var jsonSplit = new SplitContainer { Dock = DockStyle.Fill, Orientation = Orientation.Vertical, SplitterDistance = 460 };
        jsonSplit.Panel1.Controls.Add(WrapText("参数声明 JSON", _configFields));
        jsonSplit.Panel2.Controls.Add(WrapText("数据集合声明 JSON", _collections));
        root.Controls.Add(jsonSplit, 0, 2);

        _htmlSource.Multiline = true;
        _htmlSource.ScrollBars = ScrollBars.Both;
        _htmlSource.WordWrap = false;
        _htmlSource.AcceptsTab = true;
        root.Controls.Add(WrapText("模板 HTML 源码", _htmlSource), 0, 3);

        var bottom = new TableLayoutPanel { Dock = DockStyle.Bottom, ColumnCount = 2, AutoSize = true };
        bottom.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        bottom.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        _status.AutoSize = true;
        _status.AccessibleName = "模板投稿状态";
        bottom.Controls.Add(_status, 0, 0);
        var buttons = new FlowLayoutPanel { AutoSize = true, FlowDirection = FlowDirection.RightToLeft };
        var submit = new Button { Text = "提交审核", AutoSize = true };
        var cancel = new Button { Text = "取消", AutoSize = true, DialogResult = DialogResult.Cancel };
        submit.Click += (_, _) => Accept();
        buttons.Controls.Add(submit);
        buttons.Controls.Add(cancel);
        bottom.Controls.Add(buttons, 1, 0);
        root.Controls.Add(bottom, 0, 4);

        Controls.Add(root);
        CancelButton = cancel;
        UseStaticPreset();
    }

    private static void AddText(TableLayoutPanel panel, int row, int col, string label, TextBox box, string accessibleName)
    {
        panel.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        panel.Controls.Add(new Label { Text = label, AutoSize = true, Anchor = AnchorStyles.Left, Margin = new Padding(0, 6, 8, 6) }, col, row);
        box.Anchor = AnchorStyles.Left | AnchorStyles.Right;
        box.AccessibleName = accessibleName;
        panel.Controls.Add(box, col + 1, row);
    }

    private static Control WrapText(string title, TextBox box)
    {
        var group = new GroupBox { Text = title, Dock = DockStyle.Fill };
        box.Dock = DockStyle.Fill;
        box.Multiline = true;
        box.ScrollBars = ScrollBars.Both;
        box.WordWrap = false;
        box.AcceptsTab = true;
        group.Controls.Add(box);
        return group;
    }

    private void LoadHtmlFile()
    {
        using var dialog = new OpenFileDialog { Title = "选择模板 HTML 文件", Filter = "HTML 文件|*.html;*.htm;*.txt|所有文件|*.*" };
        if (dialog.ShowDialog(this) == DialogResult.OK) _htmlSource.Text = System.IO.File.ReadAllText(dialog.FileName);
    }

    private void UseStaticPreset()
    {
        _name.Text = "静态展示页模板";
        _slug.Text = "simple-page-template";
        _category.Text = "community";
        _categoryLabel.Text = "用户投稿";
        _summary.Text = "一个可配置标题、副标题和主题色的静态页面。";
        _description.Text = "适合做个人介绍、班级公告、活动页面等简单展示站点。";
        _tags.Text = "静态页面,展示,入门";
        _interactiveRequired.Checked = false;
        _analyticsRecommended.Checked = true;
        _configFields.Text = "[\r\n  {\"name\":\"siteTitle\",\"label\":\"网站标题\",\"type\":\"string\",\"required\":true,\"default\":\"我的网站\",\"placeholder\":\"例如：我的班级主页\",\"help\":\"会替换 HTML 里的 {{siteTitle}}。\"},\r\n  {\"name\":\"subtitle\",\"label\":\"副标题\",\"type\":\"text\",\"required\":false,\"default\":\"这里写一句介绍。\",\"placeholder\":\"一句话介绍网站\",\"help\":\"会替换 HTML 里的 {{subtitle}}。\"},\r\n  {\"name\":\"themeColor\",\"label\":\"主题色\",\"type\":\"color\",\"required\":false,\"default\":\"#2563eb\",\"placeholder\":\"#2563eb\",\"help\":\"颜色必须是 #RRGGBB 格式。\"}\r\n]";
        _collections.Text = "[]";
        _htmlSource.Text = "<!doctype html>\r\n<html lang=\"zh-CN\">\r\n<head>\r\n  <meta charset=\"utf-8\">\r\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\r\n  <title>{{siteTitle}}</title>\r\n  <style>body{font-family:system-ui,Microsoft YaHei,sans-serif;margin:0;min-height:100vh;display:grid;place-items:center;background:#f8fafc;color:#0f172a}main{max-width:760px;padding:40px}h1{color:{{themeColor}}}</style>\r\n</head>\r\n<body><main><h1>{{siteTitle}}</h1><p>{{subtitle}}</p></main></body>\r\n</html>";
        SetStatus("已套用静态页面示例。");
    }

    private void UseInteractivePreset()
    {
        UseStaticPreset();
        _name.Text = "留言板模板";
        _slug.Text = "message-board-template";
        _summary.Text = "带 messages 数据表声明的留言板模板。";
        _description.Text = "创建作品时会自动创建 messages 数据表。正式发布后，访客留言会保存到当前作品的数据表里。";
        _tags.Text = "留言板,互动,数据表";
        _interactiveRequired.Checked = true;
        _collections.Text = "[\r\n  {\r\n    \"name\": \"messages\",\r\n    \"permissions\": { \"publicRead\": true, \"publicWrite\": true },\r\n    \"fields\": [\r\n      { \"name\": \"nickname\", \"type\": \"string\", \"required\": true, \"isList\": false },\r\n      { \"name\": \"content\", \"type\": \"text\", \"required\": true, \"isList\": false }\r\n    ]\r\n  }\r\n]";
        _htmlSource.Text = "<!doctype html>\r\n<html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>{{siteTitle}}</title></head><body><main><h1>{{siteTitle}}</h1><p>{{subtitle}}</p><section><form id=\"form\"><input id=\"nickname\" placeholder=\"昵称\" required><textarea id=\"content\" placeholder=\"留言\" required></textarea><button>发送</button></form><div id=\"status\" role=\"status\" aria-live=\"polite\"></div><div id=\"list\"></div></section></main><script>const API_BASE='{{PLAYPAGE_API_BASE}}';const PROJECT_KEY='{{PLAYPAGE_PUBLIC_KEY}}';const PREVIEW='{{PLAYPAGE_PREVIEW}}'==='true';const preview=[];async function api(path,opt={}){if(PREVIEW){return path.includes('records')&&(!opt.method||opt.method==='GET')?{items:preview}:{id:'p',data:JSON.parse(opt.body||'{}').data,createdAt:new Date().toISOString()}}const r=await fetch(API_BASE+path,{...opt,headers:{'Content-Type':'application/json','X-Project-Key':PROJECT_KEY,...(opt.headers||{})}});const d=await r.json().catch(()=>({}));if(!r.ok)throw new Error(d.error||'请求失败');return d}async function load(){const d=await api('/collections/messages/records');list.innerHTML=(d.items||[]).map(x=>'<p><b>'+x.data.nickname+'</b>：'+x.data.content+'</p>').join('')||'还没有留言。'}form.onsubmit=async e=>{e.preventDefault();status.textContent='正在发送';const item=await api('/collections/messages/records',{method:'POST',body:JSON.stringify({data:{nickname:nickname.value,content:content.value}})});if(PREVIEW)preview.push(item);content.value='';status.textContent='已发送';load()};load();</script></body></html>";
        SetStatus("已套用留言板示例。");
    }

    private void Accept()
    {
        try
        {
            if (string.IsNullOrWhiteSpace(_name.Text) || string.IsNullOrWhiteSpace(_summary.Text))
            {
                SetStatus("模板名称和简介都要填写。");
                return;
            }
            if (string.IsNullOrWhiteSpace(_htmlSource.Text))
            {
                SetStatus("请提供模板 HTML 源码。");
                return;
            }
            var fields = JsonSerializer.Deserialize<List<TemplateConfigField>>(_configFields.Text.Trim().Length == 0 ? "[]" : _configFields.Text, JsonOptions()) ?? new List<TemplateConfigField>();
            var collections = JsonSerializer.Deserialize<List<TemplateCollectionDefinition>>(_collections.Text.Trim().Length == 0 ? "[]" : _collections.Text, JsonOptions()) ?? new List<TemplateCollectionDefinition>();
            Result = new TemplateSubmissionCreateRequest
            {
                Name = _name.Text.Trim(),
                Slug = NormalizeSlug(_slug.Text),
                Category = string.IsNullOrWhiteSpace(_category.Text) ? "community" : _category.Text.Trim(),
                CategoryLabel = string.IsNullOrWhiteSpace(_categoryLabel.Text) ? "用户投稿" : _categoryLabel.Text.Trim(),
                Summary = _summary.Text.Trim(),
                Description = _description.Text.Trim(),
                Tags = SplitTags(_tags.Text),
                InteractiveRequired = _interactiveRequired.Checked,
                AnalyticsRecommended = _analyticsRecommended.Checked,
                ConfigFields = fields,
                Collections = collections,
                HtmlSource = _htmlSource.Text,
                SourceType = "text"
            };
            DialogResult = DialogResult.OK;
            Close();
        }
        catch (Exception ex)
        {
            SetStatus("JSON 解析失败：" + ex.Message);
        }
    }

    private static JsonSerializerOptions JsonOptions() => new JsonSerializerOptions { PropertyNameCaseInsensitive = true };

    private static List<string> SplitTags(string value)
    {
        var result = new List<string>();
        foreach (var item in value.Split(new[] { ',', '，', '\n', '\r' }, StringSplitOptions.RemoveEmptyEntries))
        {
            var text = item.Trim();
            if (text.Length > 0) result.Add(text);
        }
        return result;
    }

    private void SetStatus(string text)
    {
        _status.Text = text;
        _status.AccessibleName = text;
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
