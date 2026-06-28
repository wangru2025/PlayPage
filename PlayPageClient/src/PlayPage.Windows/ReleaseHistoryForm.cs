using System;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

internal sealed class ReleaseHistoryForm : Form
{
    private readonly ListBox _list = new ListBox();
    private readonly IReadOnlyList<ReleaseInfo> _releases;

    public ReleaseInfo? SelectedRelease { get; private set; }

    public ReleaseHistoryForm(IReadOnlyList<ReleaseInfo> releases)
    {
        _releases = releases;
        Text = "作品历史版本";
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(720, 420);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "作品历史版本窗口";

        var root = new TableLayoutPanel { Dock = DockStyle.Fill, Padding = new Padding(14), ColumnCount = 1, RowCount = 3 };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        root.Controls.Add(new Label { Text = "选择一个历史版本后可以回滚。回滚会把当前作品内容恢复到所选版本。", AutoSize = true }, 0, 0);
        _list.Dock = DockStyle.Fill;
        _list.AccessibleName = "历史版本列表";
        foreach (var item in releases.OrderByDescending(x => x.CreatedAt))
        {
            _list.Items.Add(new ReleaseListItem(item));
        }
        if (_list.Items.Count > 0) _list.SelectedIndex = 0;
        root.Controls.Add(_list, 0, 1);

        var buttons = new FlowLayoutPanel { Dock = DockStyle.Bottom, AutoSize = true, FlowDirection = FlowDirection.RightToLeft };
        var rollback = new Button { Text = "回滚到此版本", AutoSize = true };
        var close = new Button { Text = "关闭", AutoSize = true, DialogResult = DialogResult.Cancel };
        rollback.Click += (_, _) => Accept();
        buttons.Controls.Add(rollback);
        buttons.Controls.Add(close);
        root.Controls.Add(buttons, 0, 2);

        Controls.Add(root);
        AcceptButton = rollback;
        CancelButton = close;
    }

    private void Accept()
    {
        if (_list.SelectedItem is not ReleaseListItem item) return;
        SelectedRelease = item.Release;
        DialogResult = DialogResult.OK;
        Close();
    }

    private sealed class ReleaseListItem
    {
        public ReleaseInfo Release { get; }
        public ReleaseListItem(ReleaseInfo release) => Release = release;
        public override string ToString()
        {
            var note = string.IsNullOrWhiteSpace(Release.ChangeNote) ? "无更新说明" : Release.ChangeNote;
            var time = Release.CreatedAt.LocalDateTime.ToString("yyyy-MM-dd HH:mm:ss");
            return $"{time}｜{note}｜入口：{Release.EntryFile}";
        }
    }
}
