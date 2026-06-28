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
        MessageBox.Show(this, $"作品名：{detail.Project.Name}\n地址：{detail.Project.PublicUrl}\n互动功能：{(detail.Project.Interactive ? "开" : "关")}\n统计功能：{(detail.Project.AnalyticsEnabled ? "开" : "关")}\n可见性：{detail.Project.Visibility}", "作品详情");
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
        var items = await _client.ListRepairRequestsAsync(p.Id);
        MessageBox.Show(this, items.Count == 0 ? "暂无修复申请。" : string.Join("\n\n", items), "修复申请");
    }

    private async Task ShowDomainsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var items = await _client.ListDomainsAsync(p.Id);
        MessageBox.Show(this, items.Count == 0 ? "暂无独立网址申请。" : string.Join("\n", items), "独立网址");
    }

    private async Task ShowReleasesAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var items = await _client.ListReleasesAsync(p.Id);
        MessageBox.Show(this, items.Count == 0 ? "暂无历史版本。" : string.Join("\n", items), "历史版本");
    }

    private async Task CreateProjectAsync()
    {
        if (_currentUser == null)
        {
            await ShowLoginDialogAsync();
            return;
        }
        var name = Prompt.Show(this, "新建作品", "作品名称：");
        if (string.IsNullOrWhiteSpace(name)) return;
        var slug = Prompt.Show(this, "新建作品", "作品链接名，只能用字母数字短横线，留空则自动生成：");
        var interactive = MessageBox.Show(this, "是否开启互动功能？", "新建作品", MessageBoxButtons.YesNo) == DialogResult.Yes;
        var analytics = MessageBox.Show(this, "是否开启访问量统计？", "新建作品", MessageBoxButtons.YesNo) == DialogResult.Yes;
        await _client.CreateProjectAsync(new ProjectCreateRequest { Name = name, Slug = slug, Interactive = interactive, AnalyticsEnabled = analytics });
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
        var name = Prompt.Show(this, "作品设置", "作品名称：", p.Name);
        if (string.IsNullOrWhiteSpace(name)) return;
        var slug = Prompt.Show(this, "作品设置", "链接名：", p.Slug);
        if (string.IsNullOrWhiteSpace(slug)) return;
        var interactive = MessageBox.Show(this, "是否开启互动功能？", "作品设置", MessageBoxButtons.YesNo) == DialogResult.Yes;
        var analytics = MessageBox.Show(this, "是否开启统计功能？", "作品设置", MessageBoxButtons.YesNo) == DialogResult.Yes;
        await _client.UpdateProjectSettingsAsync(p.Id, new ProjectSettingsRequest { Name = name, Slug = slug, Interactive = interactive, AnalyticsEnabled = analytics });
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

    private async Task ShowStatsAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var stats = await _client.GetProjectStatsAsync(p.Id);
        MessageBox.Show(this, $"访问量：{stats.TotalPageViews}\nAPI 请求：{stats.TotalApiRequests}\n成功：{stats.TotalApiSuccesses}\n失败：{stats.TotalApiFailures}\n成功率：{stats.ApiSuccessRate:P2}", "统计数据");
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

    private async Task<RepairRequestInfo?> FirstRepairRequestAsync(ProjectSummary project)
    {
        var repairs = await _client.ListRepairRequestsAsync(project.Id);
        if (repairs.Count == 0)
        {
            MessageBox.Show(this, "这个作品还没有修复申请。", "AI 圆桌");
            return null;
        }
        return repairs[0];
    }

    private async Task StartOrShowRepairAIAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var repair = await FirstRepairRequestAsync(p);
        if (repair == null) return;
        try
        {
            var latest = await _client.GetLatestRepairAIAsync(p.Id, repair.Id);
            ShowAIState(latest);
        }
        catch
        {
            if (MessageBox.Show(this, "还没有 AI 圆桌记录，是否立即启动？", "AI 圆桌", MessageBoxButtons.YesNo) != DialogResult.Yes) return;
            var started = await _client.StartRepairAIAsync(p.Id, repair.Id);
            ShowAIState(started);
        }
    }

    private async Task PreviewRepairAIAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var repair = await FirstRepairRequestAsync(p);
        if (repair == null) return;
        var state = await _client.CreateRepairAIPreviewAsync(p.Id, repair.Id);
        if (!string.IsNullOrWhiteSpace(state.Url))
        {
            System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo { FileName = state.Url, UseShellExecute = true });
        }
        ShowAIState(state);
    }

    private async Task PublishRepairAIAsync()
    {
        var p = SelectedProject();
        if (p == null) return;
        var repair = await FirstRepairRequestAsync(p);
        if (repair == null) return;
        if (MessageBox.Show(this, "确认发布 AI 修复版本？", "发布 AI 修复", MessageBoxButtons.YesNo) != DialogResult.Yes) return;
        var state = await _client.PublishRepairAIAsync(p.Id, repair.Id);
        ShowAIState(state);
        await LoadProjectsAsync();
    }

    private void ShowAIState(RepairAIState state)
    {
        var lines = new List<string>();
        lines.Add($"状态：{state.Job.Status}；第 {state.Job.Round} 轮");
        if (!string.IsNullOrWhiteSpace(state.Job.ErrorMessage)) lines.Add("错误：" + state.Job.ErrorMessage);
        if (!string.IsNullOrWhiteSpace(state.Job.PreviewUrl)) lines.Add("预览：" + state.Job.PreviewUrl);
        if (!string.IsNullOrWhiteSpace(state.Message)) lines.Add(state.Message);
        lines.Add("");
        foreach (var message in state.Messages.TakeLast(20))
        {
            lines.Add($"{message.MessageSeq}. {message.AgentName}");
            lines.Add(message.Content);
            lines.Add("");
        }
        MessageBox.Show(this, string.Join("\n", lines), "AI 圆桌");
    }
}
