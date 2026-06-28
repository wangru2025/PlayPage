using System;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using System.Threading.Tasks;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

public sealed class RepairRequestsForm : Form
{
    private readonly PlayPageApiClient _client;
    private readonly ProjectSummary _project;
    private readonly ListView _list = new ListView();
    private readonly Label _status = new Label();
    private readonly Button _reply = new Button();
    private readonly Button _roundtable = new Button();

    public RepairRequestsForm(PlayPageApiClient client, ProjectSummary project)
    {
        _client = client;
        _project = project;
        Text = "查看修复申请 - " + project.Name;
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(980, 620);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "修复申请列表窗口";
        Controls.Add(BuildLayout());
        Load += async (_, _) => await RefreshAsync();
    }

    private Control BuildLayout()
    {
        var root = new TableLayoutPanel { Dock = DockStyle.Fill, RowCount = 3, ColumnCount = 1, Padding = new Padding(12) };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        var actions = new FlowLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, FlowDirection = FlowDirection.LeftToRight };
        var refresh = new Button { Text = "刷新", AutoSize = true };
        refresh.Click += async (_, _) => await RefreshAsync();
        _roundtable.Text = "进入 AI 圆桌";
        _roundtable.AutoSize = true;
        _roundtable.Click += (_, _) => OpenRoundtable();
        _reply.Text = "补充信息";
        _reply.AutoSize = true;
        _reply.Click += async (_, _) => await ReplyAsync();
        actions.Controls.Add(refresh);
        actions.Controls.Add(_roundtable);
        actions.Controls.Add(_reply);
        root.Controls.Add(actions, 0, 0);

        _list.Dock = DockStyle.Fill;
        _list.View = View.Details;
        _list.FullRowSelect = true;
        _list.MultiSelect = false;
        _list.HideSelection = false;
        _list.AccessibleName = "修复申请列表";
        _list.Columns.Add("状态", 140);
        _list.Columns.Add("问题类型", 140);
        _list.Columns.Add("提交时间", 160);
        _list.Columns.Add("问题描述", 420);
        _list.Columns.Add("管理员回复", 300);
        _list.DoubleClick += (_, _) => OpenRoundtable();
        root.Controls.Add(_list, 0, 1);

        _status.AutoSize = true;
        _status.AccessibleName = "状态";
        root.Controls.Add(_status, 0, 2);
        return root;
    }

    private async Task RefreshAsync()
    {
        try
        {
            _status.Text = "正在读取修复申请。";
            var items = await _client.ListRepairRequestsAsync(_project.Id);
            _list.Items.Clear();
            foreach (var item in items.OrderByDescending(x => x.CreatedAt))
            {
                var row = new ListViewItem(PlayPageDisplay.Status(item.Status));
                row.SubItems.Add(PlayPageDisplay.IssueType(item.IssueType));
                row.SubItems.Add(item.CreatedAt.LocalDateTime.ToString("yyyy-MM-dd HH:mm"));
                row.SubItems.Add(item.Description);
                row.SubItems.Add(string.IsNullOrWhiteSpace(item.AdminReply) ? "暂无" : item.AdminReply);
                row.Tag = item;
                _list.Items.Add(row);
            }
            _status.Text = items.Count == 0 ? "暂无修复申请。" : $"已读取 {items.Count} 条修复申请。";
        }
        catch (Exception ex) { ShowError(ex); }
    }

    private RepairRequestInfo? SelectedRepair()
    {
        if (_list.SelectedItems.Count == 0) return null;
        return _list.SelectedItems[0].Tag as RepairRequestInfo;
    }

    private void OpenRoundtable()
    {
        var repair = SelectedRepair();
        if (repair == null)
        {
            MessageBox.Show(this, "请先选择一条修复申请。", "AI 圆桌", MessageBoxButtons.OK, MessageBoxIcon.Information);
            return;
        }
        using var form = new RepairAIRoundtableForm(_client, _project, repair);
        form.ShowDialog(this);
    }

    private async Task ReplyAsync()
    {
        var repair = SelectedRepair();
        if (repair == null)
        {
            MessageBox.Show(this, "请先选择一条修复申请。", "补充信息", MessageBoxButtons.OK, MessageBoxIcon.Information);
            return;
        }
        var text = Prompt.Show(this, "补充信息", "请填写要补充给管理员的信息：");
        if (string.IsNullOrWhiteSpace(text)) return;
        await _client.ReplyRepairRequestAsync(_project.Id, repair.Id, text);
        _status.Text = "补充信息已提交。";
        await RefreshAsync();
    }

    private void ShowError(Exception ex)
    {
        _status.Text = ex.Message;
        MessageBox.Show(this, ex.Message, "PlayPage", MessageBoxButtons.OK, MessageBoxIcon.Error);
    }
}
