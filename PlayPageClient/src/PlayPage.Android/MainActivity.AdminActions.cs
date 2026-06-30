using System.Collections.Generic;
using System.Linq;
using Android.App;
using Android.Content;
using Android.Views;
using Android.Widget;
using PlayPage.Core;

namespace PlayPage.Client.Android;

public sealed partial class MainActivity
{
    private async System.Threading.Tasks.Task ShowTemplatesAsync()
    {
        if (_currentUser == null)
        {
            await ShowLoginDialogAsync();
            return;
        }
        var items = await _client.ListTemplatesAsync();
        if (items.Count == 0)
        {
            ShowMessage("模板市场", "暂无模板。");
            return;
        }
        ShowTemplatePicker(items.OrderBy(t => t.CategoryLabel).ThenBy(t => t.Name).ToList());
    }

    private void ShowTemplatePicker(IReadOnlyList<TemplateInfo> templates)
    {
        var labels = templates.Select(t => $"{t.CategoryLabel}｜{t.Name}\n{t.Summary}").ToArray();
        new AlertDialog.Builder(this)
            .SetTitle("模板市场")
            .SetItems(labels, async (_, args) =>
            {
                try
                {
                    var detail = await _client.GetTemplateAsync(templates[args.Which].Id);
                    ShowTemplateDetail(detail);
                }
                catch (System.Exception ex) { ShowError(ex); }
            })
            .SetNegativeButton("关闭", (_, _) => { })
            .Show();
    }


    private void ShowTemplateDetail(TemplateInfo template)
    {
        var fields = template.ConfigFields.Count == 0
            ? "无"
            : string.Join("\n", template.ConfigFields.Select(f => $"- {f.Label}（{AndroidTemplateFieldType(f.Type)}{(f.Required ? "，必填" : "，可选")}）"));
        var collections = template.Collections.Count == 0
            ? "无"
            : string.Join("\n", template.Collections.Select(c => $"- {c.Name}，{c.Fields.Count} 个字段"));
        var message = $"{template.Summary}\n\n作者：{template.AuthorName}\n分类：{template.CategoryLabel}\n使用次数：{template.UsageCount}\n需要互动功能：{PlayPageDisplay.YesNo(template.InteractiveRequired)}\n建议开启统计：{PlayPageDisplay.YesNo(template.AnalyticsRecommended)}\n\n详细说明：\n{template.Description}\n\n模板参数：\n{fields}\n\n需要的数据集合：\n{collections}\n\n点“使用模板”后进入创建页面。";
        new AlertDialog.Builder(this)
            .SetTitle(template.Name)
            .SetMessage(message)
            .SetPositiveButton("使用模板", async (_, _) =>
            {
                try { await PromptCreateFromTemplateAsync(template); }
                catch (System.Exception ex) { ShowError(ex); }
            })
            .SetNegativeButton("返回模板市场", (_, _) => { })
            .Show();
    }

    private static string AndroidTemplateFieldType(string type) => type switch
    {
        "color" => "颜色",
        "select" => "下拉选择",
        "text" => "多行文本",
        "number" => "数字",
        _ => "文本"
    };

    private System.Threading.Tasks.Task<bool> PromptCreateFromTemplateAsync(TemplateInfo template)
    {
        var tcs = new System.Threading.Tasks.TaskCompletionSource<bool>();
        var scroll = new ScrollView(this);
        var layout = new LinearLayout(this) { Orientation = Orientation.Vertical };
        layout.SetPadding(32, 12, 32, 0);
        scroll.AddView(layout);

        layout.AddView(new TextView(this) { Text = $"{template.Name}\n{template.Summary}\n{template.Description}" });

        var name = new EditText(this) { Hint = "作品名称", Text = template.Name };
        name.ContentDescription = "作品名称";
        layout.AddView(name);

        var slug = new EditText(this) { Hint = "作品链接名", Text = NormalizeSlug(template.Name) };
        slug.ContentDescription = "作品链接名";
        layout.AddView(slug);

        var interactive = new CheckBox(this) { Text = "启用互动功能", Checked = template.InteractiveRequired };
        interactive.Enabled = !template.InteractiveRequired;
        layout.AddView(interactive);

        var analytics = new CheckBox(this) { Text = "启用访问量统计", Checked = template.AnalyticsRecommended };
        layout.AddView(analytics);

        var changeNote = new EditText(this) { Hint = "更新内容，可留空" };
        layout.AddView(changeNote);

        var paramControls = new Dictionary<string, View>();
        if (template.ConfigFields.Count > 0)
        {
            layout.AddView(new TextView(this) { Text = "模板参数" });
        }
        foreach (var field in template.ConfigFields)
        {
            layout.AddView(new TextView(this) { Text = field.Label + (field.Required ? " *" : "") });
            View control;
            if (field.Type == "select")
            {
                var spinner = new Spinner(this);
                var adapter = new ArrayAdapter<string>(this, global::Android.Resource.Layout.SimpleSpinnerItem, field.Options.Count == 0 ? new[] { field.Default } : field.Options.ToArray());
                adapter.SetDropDownViewResource(global::Android.Resource.Layout.SimpleSpinnerDropDownItem);
                spinner.Adapter = adapter;
                var index = field.Options.FindIndex(x => x == field.Default);
                if (index >= 0) spinner.SetSelection(index);
                control = spinner;
            }
            else
            {
                var edit = new EditText(this) { Hint = string.IsNullOrWhiteSpace(field.Placeholder) ? field.Label : field.Placeholder, Text = field.Default };
                edit.ContentDescription = field.Label;
                if (field.Type == "text")
                {
                    edit.SetSingleLine(false);
                    edit.SetMinLines(3);
                }
                control = edit;
            }
            layout.AddView(control);
            if (!string.IsNullOrWhiteSpace(field.Help)) layout.AddView(new TextView(this) { Text = field.Help });
            paramControls[field.Name] = control;
        }

        var status = new TextView(this) { Text = "填写参数后点击创建。" };
        layout.AddView(status);

        var dialog = new AlertDialog.Builder(this)
            .SetTitle("使用模板创建作品")
            .SetView(scroll)
            .SetPositiveButton("创建", (sender, _) => { })
            .SetNegativeButton("取消", (_, _) => tcs.TrySetResult(false))
            .Create();

        dialog.SetOnShowListener(new DialogShowListener(() =>
        {
            var ok = dialog.GetButton((int)DialogButtonType.Positive);
            ok.Click += async (_, _) =>
            {
                var projectName = name.Text?.Trim() ?? "";
                var projectSlug = NormalizeSlug(slug.Text ?? "");
                if (projectName.Length == 0)
                {
                    status.Text = "请填写作品名称。";
                    return;
                }
                if (projectSlug.Length == 0)
                {
                    status.Text = "请填写作品链接名。";
                    return;
                }
                var values = new Dictionary<string, string>();
                foreach (var field in template.ConfigFields)
                {
                    var value = ReadTemplateValue(paramControls[field.Name]);
                    if (field.Required && string.IsNullOrWhiteSpace(value))
                    {
                        status.Text = "请填写模板参数：" + field.Label;
                        return;
                    }
                    if (field.Type == "color" && !string.IsNullOrWhiteSpace(value) && !System.Text.RegularExpressions.Regex.IsMatch(value.Trim(), "^#[0-9a-fA-F]{6}$"))
                    {
                        status.Text = "颜色参数必须是 #RRGGBB 格式：" + field.Label;
                        return;
                    }
                    values[field.Name] = value;
                }

                try
                {
                    ok.Enabled = false;
                    status.Text = "正在创建模板作品。";
                    var project = await _client.CreateProjectAsync(new ProjectCreateRequest
                    {
                        Name = projectName,
                        Slug = projectSlug,
                        Interactive = template.InteractiveRequired || interactive.Checked,
                        AnalyticsEnabled = analytics.Checked
                    });
                    await _client.CreateReleaseFromTemplateAsync(project.Id, new TemplateCreateReleaseRequest
                    {
                        TemplateId = template.Id,
                        Params = values,
                        ChangeNote = changeNote.Text?.Trim() ?? ""
                    });
                    dialog.Dismiss();
                    SetStatus("模板作品已创建。");
                    await LoadProjectsAsync();
                    tcs.TrySetResult(true);
                }
                catch (System.Exception ex)
                {
                    ok.Enabled = true;
                    status.Text = "创建失败：" + ex.Message;
                    Toast.MakeText(this, ex.Message, ToastLength.Long)?.Show();
                }
            };
        }));
        dialog.Show();
        return tcs.Task;
    }

    private static string ReadTemplateValue(View view)
    {
        if (view is EditText edit) return edit.Text?.Trim() ?? "";
        if (view is Spinner spinner) return spinner.SelectedItem?.ToString()?.Trim() ?? "";
        return "";
    }


    private System.Threading.Tasks.Task<bool> SubmitTemplateAsync()
    {
        var tcs = new System.Threading.Tasks.TaskCompletionSource<bool>();
        var scroll = new ScrollView(this);
        var layout = new LinearLayout(this) { Orientation = Orientation.Vertical };
        layout.SetPadding(32, 12, 32, 0);
        scroll.AddView(layout);

        var name = new EditText(this) { Hint = "模板名称" };
        var slug = new EditText(this) { Hint = "模板链接名" };
        var category = new EditText(this) { Hint = "分类 ID，例如 community", Text = "community" };
        var categoryLabel = new EditText(this) { Hint = "分类显示名", Text = "用户投稿" };
        var summary = new EditText(this) { Hint = "一句话简介" };
        var description = new EditText(this) { Hint = "详细说明" };
        description.SetSingleLine(false); description.SetMinLines(3);
        var tags = new EditText(this) { Hint = "标签，逗号分隔" };
        var interactiveRequired = new CheckBox(this) { Text = "需要互动功能" };
        var analyticsRecommended = new CheckBox(this) { Text = "建议开启统计", Checked = true };
        var configFields = new EditText(this) { Hint = "参数声明 JSON", Text = "[]" };
        configFields.SetSingleLine(false); configFields.SetMinLines(5);
        var collections = new EditText(this) { Hint = "数据集合声明 JSON", Text = "[]" };
        collections.SetSingleLine(false); collections.SetMinLines(5);
        var html = new EditText(this) { Hint = "模板 HTML 源码" };
        html.SetSingleLine(false); html.SetMinLines(10);
        html.Text = "<!doctype html>\n<html lang=\"zh-CN\">\n<head><meta charset=\"utf-8\"><title>{{siteTitle}}</title></head>\n<body><h1>{{siteTitle}}</h1></body>\n</html>";
        foreach (var v in new View[] { name, slug, category, categoryLabel, summary, description, tags, interactiveRequired, analyticsRecommended, configFields, collections, html }) layout.AddView(v);
        name.TextChanged += (_, _) => { if (string.IsNullOrWhiteSpace(slug.Text)) slug.Text = NormalizeSlug(name.Text ?? ""); };
        var status = new TextView(this) { Text = "填写模板信息后提交审核。" };
        layout.AddView(status);

        var dialog = new AlertDialog.Builder(this)
            .SetTitle("投稿模板")
            .SetView(scroll)
            .SetPositiveButton("提交审核", (sender, _) => { })
            .SetNegativeButton("取消", (_, _) => tcs.TrySetResult(false))
            .Create();
        dialog.SetOnShowListener(new DialogShowListener(() =>
        {
            var ok = dialog.GetButton((int)DialogButtonType.Positive);
            ok.Click += async (_, _) =>
            {
                var n = name.Text?.Trim() ?? "";
                var sum = summary.Text?.Trim() ?? "";
                if (n.Length == 0 || sum.Length == 0) { status.Text = "模板名称和简介都要填写。"; return; }
                if (string.IsNullOrWhiteSpace(html.Text)) { status.Text = "请提供模板 HTML 源码。"; return; }
                try
                {
                    var fields = System.Text.Json.JsonSerializer.Deserialize<System.Collections.Generic.List<TemplateConfigField>>(string.IsNullOrWhiteSpace(configFields.Text) ? "[]" : configFields.Text!, new System.Text.Json.JsonSerializerOptions { PropertyNameCaseInsensitive = true }) ?? new System.Collections.Generic.List<TemplateConfigField>();
                    var cols = System.Text.Json.JsonSerializer.Deserialize<System.Collections.Generic.List<TemplateCollectionDefinition>>(string.IsNullOrWhiteSpace(collections.Text) ? "[]" : collections.Text!, new System.Text.Json.JsonSerializerOptions { PropertyNameCaseInsensitive = true }) ?? new System.Collections.Generic.List<TemplateCollectionDefinition>();
                    ok.Enabled = false;
                    status.Text = "正在提交模板审核。";
                    await _client.CreateTemplateSubmissionAsync(new TemplateSubmissionCreateRequest
                    {
                        Name = n,
                        Slug = NormalizeSlug(slug.Text ?? n),
                        Category = string.IsNullOrWhiteSpace(category.Text) ? "community" : category.Text!.Trim(),
                        CategoryLabel = string.IsNullOrWhiteSpace(categoryLabel.Text) ? "用户投稿" : categoryLabel.Text!.Trim(),
                        Summary = sum,
                        Description = description.Text?.Trim() ?? "",
                        Tags = SplitTags(tags.Text ?? ""),
                        InteractiveRequired = interactiveRequired.Checked,
                        AnalyticsRecommended = analyticsRecommended.Checked,
                        ConfigFields = fields,
                        Collections = cols,
                        HtmlSource = html.Text ?? "",
                        SourceType = "text"
                    });
                    dialog.Dismiss();
                    SetStatus("模板投稿已提交，等待管理员审核。");
                    tcs.TrySetResult(true);
                }
                catch (System.Exception ex)
                {
                    ok.Enabled = true;
                    status.Text = "提交失败：" + ex.Message;
                    Toast.MakeText(this, ex.Message, ToastLength.Long)?.Show();
                }
            };
        }));
        dialog.Show();
        return tcs.Task;
    }

    private static System.Collections.Generic.List<string> SplitTags(string value)
    {
        var result = new System.Collections.Generic.List<string>();
        foreach (var item in value.Split(new[] { ',', '，', '\n', '\r' }, System.StringSplitOptions.RemoveEmptyEntries))
        {
            var text = item.Trim();
            if (text.Length > 0) result.Add(text);
        }
        return result;
    }

    private async System.Threading.Tasks.Task ShowAdminSummaryAsync()
    {
        var users = await _client.AdminListUsersAsync();
        var projects = await _client.AdminListProjectsAsync();
        var repairs = await _client.AdminListRepairRequestsAsync();
        ShowMessage("管理摘要", $"用户：{users.Count}\n作品：{projects.Count}\n修复申请：{repairs.Count}");
    }
}
