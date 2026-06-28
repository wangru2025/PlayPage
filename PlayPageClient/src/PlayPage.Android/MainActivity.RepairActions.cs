using System.Linq;
using Android.App;
using Android.Widget;
using PlayPage.Core;

namespace PlayPage.Client.Android;

public sealed partial class MainActivity
{
    private async System.Threading.Tasks.Task CreateRepairAsync(ProjectSummary project)
    {
        var desc = await PromptAsync("申请修复", "请描述遇到的问题");
        if (string.IsNullOrWhiteSpace(desc)) return;
        await _client.CreateRepairRequestAsync(project.Id, new RepairRequestCreateRequest { IssueType = "other", Description = desc, AllowAdminEdit = true, Contact = _currentUser?.Email ?? "" });
        SetStatus("修复申请已提交。");
    }

    private async System.Threading.Tasks.Task ShowRepairRequestsAsync(ProjectSummary project)
    {
        var repairs = await _client.ListRepairRequestsAsync(project.Id);
        ShowMessage("修复申请", repairs.Count == 0 ? "暂无修复申请。" : string.Join("\n\n", repairs.Select(r => $"{r.Status}\n{r.Description}")));
    }

    private async System.Threading.Tasks.Task StartOrShowRepairAIAsync(ProjectSummary project)
    {
        var repairs = await _client.ListRepairRequestsAsync(project.Id);
        if (repairs.Count == 0) { ShowMessage("AI 圆桌", "这个作品还没有修复申请。"); return; }
        try
        {
            var state = await _client.GetLatestRepairAIAsync(project.Id, repairs[0].Id);
            ShowAIState(state);
        }
        catch
        {
            var state = await _client.StartRepairAIAsync(project.Id, repairs[0].Id);
            ShowAIState(state);
        }
    }

    private void ShowAIState(RepairAIState state)
    {
        var lines = new System.Collections.Generic.List<string> { $"状态：{state.Job.Status}；第 {state.Job.Round} 轮" };
        foreach (var message in state.Messages)
        {
            lines.Add($"{message.MessageSeq}. {message.AgentName}");
            lines.Add(message.Content);
        }
        ShowMessage("AI 圆桌", string.Join("\n", lines));
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

        var actions = domains.Select(d => $"申请删除：{d.Domain}（{d.Status}）")
            .Concat(deleteRequests.Select(r => $"删除申请：{r.Domain}（{r.Status}）"))
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
