using System.Linq;
using System.Threading.Tasks;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

public sealed partial class MainForm
{
    private async Task ShowTemplatesAsync()
    {
        if (_currentUser == null)
        {
            await ShowLoginDialogAsync();
            return;
        }
        var items = await _client.ListTemplatesAsync();
        if (items.Count == 0)
        {
            MessageBox.Show(this, "暂无模板。", "模板市场");
            return;
        }
        using var market = new TemplateMarketForm(items);
        if (market.ShowDialog(this) != DialogResult.OK || market.SelectedTemplate == null) return;
        var template = await _client.GetTemplateAsync(market.SelectedTemplate.Id);

        using var detail = new TemplateDetailForm(template);
        if (detail.ShowDialog(this) != DialogResult.OK || !detail.UseTemplate) return;

        using var create = new TemplateCreateProjectForm(template);
        if (create.ShowDialog(this) != DialogResult.OK || create.Result == null) return;
        var input = create.Result;

        SetStatus("正在创建模板作品。", false);
        var project = await _client.CreateProjectAsync(new ProjectCreateRequest
        {
            Name = input.Name,
            Slug = input.Slug,
            Interactive = input.Interactive,
            AnalyticsEnabled = input.AnalyticsEnabled
        });
        SetStatus("正在用模板发布作品。", false);
        await _client.CreateReleaseFromTemplateAsync(project.Id, new TemplateCreateReleaseRequest
        {
            TemplateId = input.Template.Id,
            Params = input.Params,
            ChangeNote = input.ChangeNote
        });
        SetStatus("模板作品已创建。", false);
        await LoadProjectsAsync();
    }

    private async Task SubmitTemplateAsync()
    {
        if (_currentUser == null)
        {
            await ShowLoginDialogAsync();
            return;
        }
        using var form = new TemplateSubmissionForm();
        if (form.ShowDialog(this) != DialogResult.OK || form.Result == null) return;
        await _client.CreateTemplateSubmissionAsync(form.Result);
        SetStatus("模板投稿已提交，等待管理员审核。", false);
    }

    private async Task ShowAdminSummaryAsync()
    {
        var users = await _client.AdminListUsersAsync();
        var projects = await _client.AdminListProjectsAsync();
        var upgrades = await _client.AdminListUpgradeRequestsAsync();
        var repairs = await _client.AdminListRepairRequestsAsync();
        MessageBox.Show(this, $"用户：{users.Count}\n作品：{projects.Count}\n升级申请：{upgrades.Count}\n修复申请：{repairs.Count}", "管理摘要");
    }

    private async Task ShowAdminUpgradeRequestsAsync()
    {
        var items = await _client.AdminListUpgradeRequestsAsync();
        var pending = items.Where(x => x.Status == "pending").ToList();
        if (pending.Count == 0)
        {
            MessageBox.Show(this, "没有待处理升级申请。", "升级申请");
            return;
        }
        var first = pending[0];
        var approve = MessageBox.Show(this, $"处理第一条待审核申请？\n用户：{first.UserEmail}\n目标套餐：{PlayPageDisplay.Plan(first.TargetPlan)}\n备注：{first.PayerNote}\n\n点是通过，点否取消。", "升级申请", MessageBoxButtons.YesNo) == DialogResult.Yes;
        if (!approve) return;
        await _client.AdminReviewUpgradeRequestAsync(first.Id, new UpgradeRequestReviewRequest { Status = "approved", TargetPlan = first.TargetPlan, AdminNote = "客户端审核通过" });
        SetStatus("升级申请已通过。", false);
    }

    private async Task ShowAdminDomainRequestsAsync()
    {
        var items = await _client.AdminListProjectDomainsAsync("pending");
        if (items.Count == 0)
        {
            MessageBox.Show(this, "没有待处理独立网址申请。", "独立网址申请");
            return;
        }
        MessageBox.Show(this, string.Join("\n", items.Select(x => $"{x.Domain}｜{x.OwnerEmail}｜{x.ProjectName}")), "独立网址申请");
    }

    private async Task ShowAdminDomainDeleteRequestsAsync()
    {
        var items = await _client.AdminListProjectDomainDeleteRequestsAsync("pending");
        if (items.Count == 0)
        {
            MessageBox.Show(this, "没有待处理独立网址删除申请。", "独立网址删除申请");
            return;
        }
        var first = items[0];
        var approve = MessageBox.Show(this, $"处理第一条删除申请？\n域名：{first.Domain}\n原因：{first.Reason}\n\n点是标记已完成，点否取消。", "独立网址删除申请", MessageBoxButtons.YesNo, MessageBoxIcon.Question) == DialogResult.Yes;
        if (!approve) return;
        var note = Prompt.Show(this, "独立网址删除申请", "管理员备注，可留空：") ?? "";
        await _client.AdminReviewProjectDomainDeleteAsync(first.Id, new ProjectDomainDeleteReviewRequest { Status = "completed", AdminNote = note });
        SetStatus("独立网址删除申请已标记完成。", false);
    }

    private async Task ShowAdminRepairRequestsAsync()
    {
        var items = await _client.AdminListRepairRequestsAsync("pending");
        if (items.Count == 0)
        {
            MessageBox.Show(this, "没有待处理修复申请。", "修复申请");
            return;
        }
        MessageBox.Show(this, string.Join("\n\n", items.Select(x => $"{x.ProjectName}\n{x.OwnerEmail}\n{x.Description}")), "修复申请");
    }

    private async Task ShowAdminTemplateSubmissionsAsync()
    {
        var items = await _client.AdminListTemplateSubmissionsAsync("pending");
        if (items.Count == 0)
        {
            MessageBox.Show(this, "没有待审核模板投稿。", "模板投稿");
            return;
        }
        var first = items[0];
        var publish = MessageBox.Show(this, $"发布第一条模板投稿？\n{first.Name}\n作者：{first.AuthorName}\n{first.Summary}", "模板投稿", MessageBoxButtons.YesNo) == DialogResult.Yes;
        if (!publish) return;
        await _client.AdminReviewTemplateSubmissionAsync(first.Id, new TemplateSubmissionReviewRequest { Status = "published", AdminNote = "客户端审核发布" });
        SetStatus("模板投稿已发布。", false);
    }
}
