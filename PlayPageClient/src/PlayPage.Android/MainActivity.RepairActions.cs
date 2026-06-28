using System;
using System.Linq;
using Android.App;
using Android.Content;
using Android.Widget;
using PlayPage.Core;

namespace PlayPage.Client.Android;

public sealed partial class MainActivity
{
    private async System.Threading.Tasks.Task CreateRepairAsync(ProjectSummary project)
    {
        var desc = await PromptAsync("申请修复", "请描述遇到的问题");
        if (string.IsNullOrWhiteSpace(desc)) return;
        var expected = await PromptAsync("申请修复", "你期望修成什么样，可留空");
        await _client.CreateRepairRequestAsync(project.Id, new RepairRequestCreateRequest
        {
            IssueType = "other",
            Description = desc,
            Expected = expected,
            AllowAdminEdit = true,
            Contact = _currentUser?.Email ?? ""
        });
        SetStatus("修复申请已提交。可以在“查看修复申请”里查看处理进度。 ");
    }

    private async System.Threading.Tasks.Task ShowRepairRequestsAsync(ProjectSummary project)
    {
        var repairs = (await _client.ListRepairRequestsAsync(project.Id)).OrderByDescending(r => r.CreatedAt).ToList();
        if (repairs.Count == 0)
        {
            ShowMessage("查看修复申请", "暂无修复申请。");
            return;
        }

        var labels = repairs.Select(r => $"{PlayPageDisplay.Status(r.Status)}｜{PlayPageDisplay.IssueType(r.IssueType)}\n{r.CreatedAt.LocalDateTime:yyyy-MM-dd HH:mm}\n{ShortText(r.Description, 80)}").ToArray();
        new AlertDialog.Builder(this)
            .SetTitle("查看修复申请")
            .SetItems(labels, (_, args) => ShowRepairRequestActions(project, repairs[args.Which]))
            .SetNegativeButton("关闭", (_, _) => { })
            .Show();
    }

    private void ShowRepairRequestActions(ProjectSummary project, RepairRequestInfo repair)
    {
        var adminReply = string.IsNullOrWhiteSpace(repair.AdminReply) ? "暂无" : repair.AdminReply;
        var title = $"{PlayPageDisplay.Status(repair.Status)}｜{PlayPageDisplay.IssueType(repair.IssueType)}";
        var message = $"问题描述：\n{repair.Description}\n\n管理员回复：\n{adminReply}";
        var actions = new[] { "进入 AI 圆桌", "补充信息", "查看详情" };
        new AlertDialog.Builder(this)
            .SetTitle(title)
            .SetMessage(message)
            .SetItems(actions, async (_, args) =>
            {
                try
                {
                    if (args.Which == 0) await ShowRepairAIRoundtableAsync(project, repair);
                    else if (args.Which == 1) await ReplyRepairRequestAsync(project, repair);
                    else ShowMessage("修复申请详情", message);
                }
                catch (Exception ex) { ShowError(ex); }
            })
            .SetNegativeButton("返回", (_, _) => { })
            .Show();
    }

    private async System.Threading.Tasks.Task ReplyRepairRequestAsync(ProjectSummary project, RepairRequestInfo repair)
    {
        var text = await PromptAsync("补充信息", "请填写要补充给管理员的信息");
        if (string.IsNullOrWhiteSpace(text)) return;
        await _client.ReplyRepairRequestAsync(project.Id, repair.Id, text);
        SetStatus("补充信息已提交。 ");
    }

    private async System.Threading.Tasks.Task ShowRepairAIRoundtableAsync(ProjectSummary project, RepairRequestInfo repair)
    {
        RepairAIState? state = null;
        try { state = await _client.GetLatestRepairAIAsync(project.Id, repair.Id); }
        catch { }
        ShowRepairAIRoundtableDialog(project, repair, state);
        await System.Threading.Tasks.Task.CompletedTask;
    }

    private void ShowRepairAIRoundtableDialog(ProjectSummary project, RepairRequestInfo repair, RepairAIState? state)
    {
        var title = "AI 圆桌";
        var body = state == null ? "还没有启动 AI 圆桌。" : FormatAIState(state);
        var actions = new[] { "刷新", "启动 AI 圆桌", "叫停 AI 圆桌", "预览修复效果", "满意，发布修复", "还有问题" };
        new AlertDialog.Builder(this)
            .SetTitle(title)
            .SetMessage(body)
            .SetItems(actions, async (_, args) =>
            {
                try
                {
                    RepairAIState next;
                    switch (args.Which)
                    {
                        case 0:
                            next = await _client.GetLatestRepairAIAsync(project.Id, repair.Id);
                            ShowRepairAIRoundtableDialog(project, repair, next);
                            break;
                        case 1:
                            next = await _client.StartRepairAIAsync(project.Id, repair.Id);
                            ShowRepairAIRoundtableDialog(project, repair, next);
                            break;
                        case 2:
                            next = await _client.StopRepairAIAsync(project.Id, repair.Id);
                            ShowRepairAIRoundtableDialog(project, repair, next);
                            break;
                        case 3:
                            next = await _client.CreateRepairAIPreviewAsync(project.Id, repair.Id);
                            var url = !string.IsNullOrWhiteSpace(next.Url) ? next.Url : next.Job.PreviewUrl;
                            if (!string.IsNullOrWhiteSpace(url)) StartActivity(new Intent(Intent.ActionView, global::Android.Net.Uri.Parse(url)));
                            ShowRepairAIRoundtableDialog(project, repair, next);
                            break;
                        case 4:
                            new AlertDialog.Builder(this)
                                .SetTitle("发布 AI 修复")
                                .SetMessage("确认发布 AI 修复版本？")
                                .SetPositiveButton("发布", async (_, _) =>
                                {
                                    try
                                    {
                                        var published = await _client.PublishRepairAIAsync(project.Id, repair.Id);
                                        SetStatus("AI 修复版本已发布。 ");
                                        ShowRepairAIRoundtableDialog(project, repair, published);
                                        await LoadProjectsAsync();
                                    }
                                    catch (Exception ex) { ShowError(ex); }
                                })
                                .SetNegativeButton("取消", (_, _) => { })
                                .Show();
                            break;
                        case 5:
                            await SendRepairAIFeedbackAsync(project, repair);
                            break;
                    }
                }
                catch (Exception ex) { ShowError(ex); }
            })
            .SetNegativeButton("关闭", (_, _) => { })
            .Show();
    }

    private async System.Threading.Tasks.Task SendRepairAIFeedbackAsync(ProjectSummary project, RepairRequestInfo repair)
    {
        var text = await PromptAsync("还有问题", "请描述问题还在哪里");
        if (string.IsNullOrWhiteSpace(text)) return;
        var state = await _client.SendRepairAIFeedbackAsync(project.Id, repair.Id, text);
        ShowRepairAIRoundtableDialog(project, repair, state);
    }

    private static string FormatAIState(RepairAIState state)
    {
        var lines = new System.Collections.Generic.List<string>
        {
            $"AI 状态：{PlayPageDisplay.AiStatus(state.Job.Status)}；第 {state.Job.Round} 轮"
        };
        if (!string.IsNullOrWhiteSpace(state.Job.ErrorMessage)) lines.Add("错误：" + state.Job.ErrorMessage);
        foreach (var message in state.Messages.OrderBy(m => m.MessageSeq).TakeLast(30))
        {
            lines.Add("");
            lines.Add($"{message.MessageSeq}. {message.AgentName}");
            lines.Add(message.Content);
        }
        return string.Join("\n", lines);
    }

    private static string ShortText(string value, int max)
    {
        value = (value ?? "").Replace("\r", " ").Replace("\n", " ").Trim();
        return value.Length <= max ? value : value[..max] + "……";
    }

    private async System.Threading.Tasks.Task CreateDomainAsync(ProjectSummary project)
    {
        var subdomain = await PromptAsync("申请独立网址", "子域名，例如 my-game");
        if (string.IsNullOrWhiteSpace(subdomain)) return;
        var created = await _client.CreateDomainRequestAsync(project.Id, subdomain);
        SetStatus("独立网址申请已提交：" + created.Domain);
    }

    private async System.Threading.Tasks.Task ShowDomainsAsync(ProjectSummary project)
    {
        var domains = await _client.ListDomainsAsync(project.Id);
        var deleteRequests = await _client.ListDomainDeleteRequestsAsync(project.Id);
        if (domains.Count == 0 && deleteRequests.Count == 0)
        {
            ShowMessage("独立网址", "暂无独立网址申请。");
            return;
        }

        var actions = domains.Select(d => $"申请删除：{d.Domain}（{PlayPageDisplay.Status(d.Status)}）")
            .Concat(deleteRequests.Select(r => $"删除申请：{r.Domain}（{PlayPageDisplay.Status(r.Status)}）"))
            .ToArray();
        new AlertDialog.Builder(this)
            .SetTitle("独立网址")
            .SetItems(actions, async (_, args) =>
            {
                try
                {
                    if (args.Which >= domains.Count) return;
                    var domain = domains[args.Which];
                    var reason = await PromptAsync("申请删除独立网址", "删除原因，可留空");
                    await _client.CreateDomainDeleteRequestAsync(project.Id, domain.Id, reason ?? "");
                    SetStatus("已提交删除独立网址申请：" + domain.Domain);
                }
                catch (System.Exception ex) { ShowError(ex); }
            })
            .SetNegativeButton("关闭", (_, _) => { })
            .Show();
    }
}
