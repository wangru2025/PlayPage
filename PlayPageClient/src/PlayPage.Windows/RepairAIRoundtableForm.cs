using System;
using System.Drawing;
using System.Linq;
using System.Threading.Tasks;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

public sealed class RepairAIRoundtableForm : Form
{
    private readonly PlayPageApiClient _client;
    private readonly ProjectSummary _project;
    private readonly RepairRequestInfo _repair;
    private readonly ListView _messages = new ListView();
    private readonly Label _status = new Label();
    private RepairAIState? _state;

    public RepairAIRoundtableForm(PlayPageApiClient client, ProjectSummary project, RepairRequestInfo repair)
    {
        _client = client;
        _project = project;
        _repair = repair;
        Text = "AI 圆桌 - " + project.Name;
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(1100, 720);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "AI 圆桌窗口";
        Controls.Add(BuildLayout());
        Load += async (_, _) => await RefreshAsync();
    }

    private Control BuildLayout()
    {
        var root = new TableLayoutPanel { Dock = DockStyle.Fill, RowCount = 4, ColumnCount = 1, Padding = new Padding(12) };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        var summary = new Label
        {
            Text = $"修复申请：{PlayPageDisplay.IssueType(_repair.IssueType)}\n{_repair.Description}",
            AutoSize = true,
            MaximumSize = new Size(1020, 0)
        };
        root.Controls.Add(summary, 0, 0);

        var actions = new FlowLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, FlowDirection = FlowDirection.LeftToRight };
        AddButton(actions, "刷新", async () => await RefreshAsync());
        AddButton(actions, "启动 AI 圆桌", async () => await StartAsync());
        AddButton(actions, "叫停 AI 圆桌", async () => await StopAsync());
        AddButton(actions, "预览修复效果", async () => await PreviewAsync());
        AddButton(actions, "满意，发布修复", async () => await PublishAsync());
        AddButton(actions, "还有问题", async () => await FeedbackAsync());
        root.Controls.Add(actions, 0, 1);

        _messages.Dock = DockStyle.Fill;
        _messages.View = View.Details;
        _messages.FullRowSelect = true;
        _messages.HideSelection = false;
        _messages.AccessibleName = "AI 圆桌消息列表";
        _messages.Columns.Add("序号", 70);
        _messages.Columns.Add("角色", 170);
        _messages.Columns.Add("时间", 150);
        _messages.Columns.Add("内容", 760);
        root.Controls.Add(_messages, 0, 2);

        _status.AutoSize = true;
        _status.AccessibleName = "AI 圆桌状态";
        root.Controls.Add(_status, 0, 3);
        return root;
    }

    private static void AddButton(FlowLayoutPanel panel, string text, Func<Task> action)
    {
        var button = new Button { Text = text, AutoSize = true };
        button.Click += async (_, _) => await action();
        panel.Controls.Add(button);
    }

    private async Task RefreshAsync()
    {
        try
        {
            _status.Text = "正在读取 AI 圆桌。";
            _state = await _client.GetLatestRepairAIAsync(_project.Id, _repair.Id);
            RenderState();
        }
        catch (Exception ex)
        {
            _state = null;
            _messages.Items.Clear();
            _status.Text = "还没有启动 AI 圆桌。";
            if (ex is not PlayPageApiException) MessageBox.Show(this, ex.Message, "PlayPage", MessageBoxButtons.OK, MessageBoxIcon.Error);
        }
    }

    private async Task StartAsync()
    {
        try
        {
            _status.Text = "正在启动 AI 圆桌。";
            _state = await _client.StartRepairAIAsync(_project.Id, _repair.Id);
            RenderState();
        }
        catch (Exception ex) { ShowError(ex); }
    }

    private async Task StopAsync()
    {
        if (MessageBox.Show(this, "确认叫停 AI 圆桌吗？叫停后本轮不会继续生成修复结果。", "叫停 AI 圆桌", MessageBoxButtons.YesNo, MessageBoxIcon.Question) != DialogResult.Yes) return;
        try
        {
            _state = await _client.StopRepairAIAsync(_project.Id, _repair.Id);
            RenderState();
        }
        catch (Exception ex) { ShowError(ex); }
    }

    private async Task PreviewAsync()
    {
        try
        {
            _state = await _client.CreateRepairAIPreviewAsync(_project.Id, _repair.Id);
            var url = !string.IsNullOrWhiteSpace(_state.Url) ? _state.Url : _state.Job.PreviewUrl;
            if (!string.IsNullOrWhiteSpace(url))
            {
                System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo { FileName = url, UseShellExecute = true });
            }
            RenderState();
        }
        catch (Exception ex) { ShowError(ex); }
    }

    private async Task PublishAsync()
    {
        if (MessageBox.Show(this, "确认发布 AI 修复版本？", "发布 AI 修复", MessageBoxButtons.YesNo, MessageBoxIcon.Question) != DialogResult.Yes) return;
        try
        {
            _state = await _client.PublishRepairAIAsync(_project.Id, _repair.Id);
            RenderState();
        }
        catch (Exception ex) { ShowError(ex); }
    }

    private async Task FeedbackAsync()
    {
        var text = Prompt.Show(this, "还有问题", "请描述问题还在哪里：");
        if (string.IsNullOrWhiteSpace(text)) return;
        try
        {
            _state = await _client.SendRepairAIFeedbackAsync(_project.Id, _repair.Id, text);
            RenderState();
        }
        catch (Exception ex) { ShowError(ex); }
    }

    private void RenderState()
    {
        _messages.Items.Clear();
        if (_state == null)
        {
            _status.Text = "还没有启动 AI 圆桌。";
            return;
        }
        foreach (var message in _state.Messages.OrderBy(x => x.MessageSeq))
        {
            var row = new ListViewItem(message.MessageSeq.ToString());
            row.SubItems.Add(message.AgentName);
            row.SubItems.Add(message.CreatedAt.LocalDateTime.ToString("HH:mm:ss"));
            row.SubItems.Add(message.Content.Replace("\r", " ").Replace("\n", " "));
            _messages.Items.Add(row);
        }
        var preview = !string.IsNullOrWhiteSpace(_state.Url) ? _state.Url : _state.Job.PreviewUrl;
        _status.Text = $"AI 状态：{PlayPageDisplay.AiStatus(_state.Job.Status)}；第 {_state.Job.Round} 轮" +
            (string.IsNullOrWhiteSpace(preview) ? "" : "；已有预览") +
            (string.IsNullOrWhiteSpace(_state.Job.ErrorMessage) ? "" : "；" + _state.Job.ErrorMessage);
    }

    private void ShowError(Exception ex)
    {
        _status.Text = ex.Message;
        MessageBox.Show(this, ex.Message, "PlayPage", MessageBoxButtons.OK, MessageBoxIcon.Error);
    }
}
