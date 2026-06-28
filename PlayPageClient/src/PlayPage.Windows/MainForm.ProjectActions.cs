using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

public sealed partial class MainForm
{
    private async Task ShowProjectDetailsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var detail = await _client.GetProjectAsync(p.Id);
        MessageBox.Show(this, $"作品名：{detail.Project.Name}\n地址：{detail.Project.PublicUrl}\n互动功能：{PlayPageDisplay.YesNo(detail.Project.Interactive)}\n统计功能：{PlayPageDisplay.YesNo(detail.Project.AnalyticsEnabled)}\n广场显示：{PlayPageDisplay.Visibility(detail.Project.Visibility)}", "作品详情");
    }

    private async Task ToggleVisibilityAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var next = p.Visibility == "public" ? "unlisted" : "public";
        await _client.UpdateProjectVisibilityAsync(p.Id, next);
        await LoadProjectsAsync();
    }

    private async Task ShowRepairRequestsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        using var form = new RepairRequestsForm(_client, p);
        form.ShowDialog(this);
        await Task.CompletedTask;
    }

    private async Task ShowDomainsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var items = await _client.ListDomainsAsync(p.Id);
        var deleteRequests = await _client.ListDomainDeleteRequestsAsync(p.Id);
        if (items.Count == 0 && deleteRequests.Count == 0)
        {
            MessageBox.Show(this, "暂无独立网址申请。", "独立网址");
            return;
        }
        using var form = new DomainManagementForm(items, deleteRequests);
        if (form.ShowDialog(this) != DialogResult.OK || form.SelectedDomain == null) return;
        var domain = form.SelectedDomain;
        var reason = Prompt.Show(this, "申请删除独立网址", "删除原因，可留空：") ?? "";
        await _client.CreateDomainDeleteRequestAsync(p.Id, domain.Id, reason);
        SetStatus($"已提交删除独立网址申请：{domain.Domain}", false);
    }

    private async Task ShowReleasesAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var items = await _client.ListReleasesAsync(p.Id);
        if (items.Count == 0)
        {
            MessageBox.Show(this, "暂无历史版本。", "历史版本");
            return;
        }
        using var form = new ReleaseHistoryForm(items);
        if (form.ShowDialog(this) != DialogResult.OK || form.SelectedRelease == null) return;
        var selected = form.SelectedRelease;
        var note = string.IsNullOrWhiteSpace(selected.ChangeNote) ? selected.CreatedAt.LocalDateTime.ToString("yyyy-MM-dd HH:mm:ss") : selected.ChangeNote;
        if (MessageBox.Show(this, $"确认回滚到这个版本吗？\n{note}", "确认回滚", MessageBoxButtons.YesNo, MessageBoxIcon.Question) != DialogResult.Yes) return;
        await _client.RollbackReleaseAsync(p.Id, selected.Id);
        SetStatus("作品已经回滚到所选版本。", false);
        await LoadProjectsAsync();
    }

    private async Task CreateProjectAsync()
    {
        if (_currentUser == null)
        {
            await ShowLoginDialogAsync();
            return;
        }

        using var form = new CreateProjectForm();
        if (form.ShowDialog(this) != DialogResult.OK || form.Result == null) return;
        var input = form.Result;

        SetStatus("正在创建作品。", false);
        var project = await _client.CreateProjectAsync(new ProjectCreateRequest
        {
            Name = input.Name,
            Slug = input.Slug,
            Interactive = input.Interactive,
            AnalyticsEnabled = input.AnalyticsEnabled
        });

        if (input.UploadMode == CreateProjectUploadMode.File)
        {
            var ext = System.IO.Path.GetExtension(input.FilePath).ToLowerInvariant();
            var mode = ext == ".zip" ? "zip" : "html";
            SetStatus("正在上传作品文件。", false);
            await _client.UploadReleaseFileAsync(project.Id, input.FilePath, mode, input.ChangeNote);
        }
        else if (input.UploadMode == CreateProjectUploadMode.HtmlText)
        {
            SetStatus("正在发布粘贴的 HTML。", false);
            await _client.UploadReleaseHtmlTextAsync(project.Id, input.HtmlText, input.ChangeNote);
        }

        SetStatus("作品已创建。", false);
        await LoadProjectsAsync();
    }

    private async Task UploadReleaseAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        using var dialog = new OpenFileDialog { Title = "选择 HTML 或 ZIP 文件", Filter = "网页文件|*.html;*.htm;*.zip|所有文件|*.*" };
        if (dialog.ShowDialog(this) != DialogResult.OK) return;
        var ext = System.IO.Path.GetExtension(dialog.FileName).ToLowerInvariant();
        var mode = ext == ".zip" ? "zip" : "html";
        var note = Prompt.Show(this, "上传新版本", "更新内容，可留空：");
        await _client.UploadReleaseFileAsync(p.Id, dialog.FileName, mode, note);
        SetStatus("新版本已上传。", false);
        await LoadProjectsAsync();
    }

    private async Task DownloadSourceAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var file = await _client.DownloadProjectSourceAsync(p.Id);
        using var dialog = new SaveFileDialog { Title = "保存作品源码", FileName = file.FileName };
        if (dialog.ShowDialog(this) != DialogResult.OK) return;
        await System.IO.File.WriteAllBytesAsync(dialog.FileName, file.Content);
        SetStatus("作品源码已保存。", false);
    }

    private async Task EditSettingsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        using var form = new ProjectSettingsForm(p);
        if (form.ShowDialog(this) != DialogResult.OK || form.Result == null) return;
        var input = form.Result;
        await _client.UpdateProjectSettingsAsync(p.Id, new ProjectSettingsRequest
        {
            Name = input.Name,
            Slug = input.Slug,
            Interactive = input.Interactive,
            AnalyticsEnabled = input.AnalyticsEnabled
        });
        SetStatus("作品设置已保存。", false);
        await LoadProjectsAsync();
    }

    private async Task CreateRepairAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var desc = Prompt.Show(this, "申请修复", "请描述遇到的问题：");
        if (string.IsNullOrWhiteSpace(desc)) return;
        var expected = Prompt.Show(this, "申请修复", "你期望修成什么样，可留空：");
        await _client.CreateRepairRequestAsync(p.Id, new RepairRequestCreateRequest
        {
            IssueType = "other",
            Description = desc,
            Expected = expected,
            AllowAdminEdit = true,
            Contact = _currentUser?.Email ?? ""
        });
        SetStatus("修复申请已提交。", false);
    }

    private async Task CreateDomainAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var subdomain = Prompt.Show(this, "申请独立网址", "请输入子域名，例如 my-game：");
        if (string.IsNullOrWhiteSpace(subdomain)) return;
        var created = await _client.CreateDomainRequestAsync(p.Id, subdomain);
        SetStatus($"独立网址申请已提交：{created.Domain}", false);
    }

    private async Task DeleteProjectAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var domains = await _client.ListDomainsAsync(p.Id);
        var activeDomains = domains.Where(x => x.Status == "active").ToList();
        var message = $"确认删除作品“{p.Name}”吗？此操作会删除作品本身。";
        if (activeDomains.Count > 0)
        {
            message += $"\n\n这个作品有 {activeDomains.Count} 个已通过的独立网址。删除作品前会先提交独立网址删除申请，管理员处理前这些网址会进入待清理状态。";
        }
        if (MessageBox.Show(this, message, "删除作品", MessageBoxButtons.YesNo, MessageBoxIcon.Warning) != DialogResult.Yes) return;

        foreach (var domain in activeDomains)
        {
            try
            {
                await _client.CreateDomainDeleteRequestAsync(p.Id, domain.Id, "删除作品时自动申请删除独立网址");
            }
            catch
            {
                // 如果已有重复申请或接口拒绝，继续删除作品；用户仍可在独立网址页面查看状态。
            }
        }

        await _client.DeleteProjectAsync(p.Id);
        SetStatus("作品已删除。", false);
        await LoadProjectsAsync();
    }

    private async Task ShowStatsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var stats = await _client.GetProjectStatsAsync(p.Id);
        MessageBox.Show(this, $"访问量：{stats.TotalPageViews}\nAPI 请求：{stats.TotalApiRequests}\n成功：{stats.TotalApiSuccesses}\n失败：{stats.TotalApiFailures}\n成功率：{stats.ApiSuccessRate:P2}", "统计数据");
    }

    private async Task ManageInteractiveDataAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        if (!p.Interactive)
        {
            MessageBox.Show(this, "这个作品没有开启互动功能。可以先在作品设置里开启互动功能。", "管理互动数据", MessageBoxButtons.OK, MessageBoxIcon.Information);
            return;
        }
        using var form = new InteractiveDataForm(_client, p);
        form.ShowDialog(this);
        await Task.CompletedTask;
    }

    private async Task ExportDataAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var collections = await _client.ListCollectionsAsync(p.Id);
        if (collections.Count == 0)
        {
            MessageBox.Show(this, "这个作品还没有数据表。", "导出数据表");
            return;
        }
        var names = collections.Select(item => item.Name).ToList();
        var format = MessageBox.Show(this, "是否导出为 Word 表格？点“否”则导出 JSON。", "导出数据表", MessageBoxButtons.YesNo) == DialogResult.Yes ? "word" : "json";
        var file = await _client.ExportProjectDataAsync(p.Id, names, format);
        using var dialog = new SaveFileDialog { Title = "保存数据表导出", FileName = file.FileName };
        if (dialog.ShowDialog(this) != DialogResult.OK) return;
        await System.IO.File.WriteAllBytesAsync(dialog.FileName, file.Content);
        SetStatus("数据表已导出。", false);
    }


}
