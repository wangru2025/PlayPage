using System.Linq;
using System.Threading.Tasks;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

public sealed partial class MainForm
{
    private async Task ShowTemplatesAsync()
    {
        var items = await _client.ListTemplatesAsync();
        MessageBox.Show(this, items.Count == 0 ? "暂无模板。" : string.Join("\n", items.Select(t => $"{t.Name} - {t.Summary}")), "模板市场");
    }

    private async Task SubmitTemplateAsync()
    {
        var name = Prompt.Show(this, "投稿模板", "模板名称：");
        if (string.IsNullOrWhiteSpace(name)) return;
        var slug = Prompt.Show(this, "投稿模板", "模板链接名：");
        if (string.IsNullOrWhiteSpace(slug)) return;
        var summary = Prompt.Show(this, "投稿模板", "一句话简介：");
        var description = Prompt.Show(this, "投稿模板", "详细说明：");
        using var dialog = new OpenFileDialog { Title = "选择模板 HTML 文件", Filter = "HTML 文件|*.html;*.htm|所有文件|*.*" };
        if (dialog.ShowDialog(this) != DialogResult.OK) return;
        var html = await System.IO.File.ReadAllTextAsync(dialog.FileName);
        await _client.CreateTemplateSubmissionAsync(new TemplateSubmissionCreateRequest
        {
            Name = name,
            Slug = slug,
            Summary = summary,
            Description = description,
            HtmlSource = html,
            SourceType = "html",
            Category = "community",
            CategoryLabel = "社区投稿"
        });
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
        var approve = MessageBox.Show(this, $"处理第一条待审核申请？\n用户：{first.UserEmail}\n目标套餐：{first.TargetPlan}\n备注：{first.PayerNote}\n\n点是通过，点否取消。", "升级申请", MessageBoxButtons.YesNo) == DialogResult.Yes;
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
        MessageBox.Show(this, string.Join("\n", items.Select(x => $"{x.Domain} - {x.OwnerEmail} - {x.ProjectName}")), "独立网址申请");
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
