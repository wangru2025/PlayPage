using System.Collections.Generic;
using System.Linq;
using Android.App;
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
                    await PromptCreateFromTemplateAsync(detail);
                }
                catch (System.Exception ex) { ShowError(ex); }
            })
            .SetNegativeButton("关闭", (_, _) => { })
            .Show();
    }

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

    private async System.Threading.Tasks.Task ShowAdminSummaryAsync()
    {
        var users = await _client.AdminListUsersAsync();
        var projects = await _client.AdminListProjectsAsync();
        var repairs = await _client.AdminListRepairRequestsAsync();
        ShowMessage("管理摘要", $"用户：{users.Count}\n作品：{projects.Count}\n修复申请：{repairs.Count}");
    }
}
