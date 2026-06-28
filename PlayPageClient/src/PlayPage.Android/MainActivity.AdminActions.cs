using System.Linq;
using Android.App;
using PlayPage.Core;

namespace PlayPage.Client.Android;

public sealed partial class MainActivity
{
    private async System.Threading.Tasks.Task ShowTemplatesAsync()
    {
        var items = await _client.ListTemplatesAsync();
        var text = items.Count == 0 ? "暂无模板。" : string.Join("\n", items.Select(t => $"{t.Name} - {t.Summary}"));
        ShowMessage("模板市场", text);
    }

    private async System.Threading.Tasks.Task ShowAdminSummaryAsync()
    {
        var users = await _client.AdminListUsersAsync();
        var projects = await _client.AdminListProjectsAsync();
        var repairs = await _client.AdminListRepairRequestsAsync();
        ShowMessage("管理摘要", $"用户：{users.Count}\n作品：{projects.Count}\n修复申请：{repairs.Count}");
    }
}
