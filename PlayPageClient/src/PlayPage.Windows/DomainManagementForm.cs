using System;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

public sealed class DomainManagementForm : Form
{
    private readonly ListBox _domains = new ListBox();
    private readonly ListBox _deleteRequests = new ListBox();

    public ProjectDomainInfo? SelectedDomain { get; private set; }

    public DomainManagementForm(IReadOnlyList<ProjectDomainInfo> domains, IReadOnlyList<ProjectDomainDeleteRequestInfo> deleteRequests)
    {
        Text = "独立网址";
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(760, 480);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "独立网址管理窗口";

        var root = new TableLayoutPanel { Dock = DockStyle.Fill, ColumnCount = 2, RowCount = 2, Padding = new Padding(12) };
        root.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));
        root.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        var domainPanel = new TableLayoutPanel { Dock = DockStyle.Fill, RowCount = 2, ColumnCount = 1 };
        domainPanel.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        domainPanel.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        domainPanel.Controls.Add(new Label { Text = "当前独立网址", AutoSize = true }, 0, 0);
        _domains.Dock = DockStyle.Fill;
        _domains.AccessibleName = "当前独立网址列表";
        foreach (var domain in domains.OrderBy(x => x.Domain)) _domains.Items.Add(new DomainItem(domain));
        domainPanel.Controls.Add(_domains, 0, 1);

        var requestPanel = new TableLayoutPanel { Dock = DockStyle.Fill, RowCount = 2, ColumnCount = 1 };
        requestPanel.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        requestPanel.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        requestPanel.Controls.Add(new Label { Text = "删除申请", AutoSize = true }, 0, 0);
        _deleteRequests.Dock = DockStyle.Fill;
        _deleteRequests.AccessibleName = "独立网址删除申请列表";
        foreach (var request in deleteRequests.OrderByDescending(x => x.CreatedAt)) _deleteRequests.Items.Add(new DeleteRequestItem(request));
        requestPanel.Controls.Add(_deleteRequests, 0, 1);

        var buttons = new FlowLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, FlowDirection = FlowDirection.RightToLeft };
        var close = new Button { Text = "关闭", AutoSize = true, DialogResult = DialogResult.Cancel };
        var delete = new Button { Text = "申请删除所选网址", AutoSize = true };
        delete.Click += (_, _) =>
        {
            if (_domains.SelectedItem is not DomainItem item)
            {
                MessageBox.Show(this, "请先选择一个独立网址。", "独立网址", MessageBoxButtons.OK, MessageBoxIcon.Information);
                return;
            }
            SelectedDomain = item.Value;
            DialogResult = DialogResult.OK;
            Close();
        };
        buttons.Controls.Add(close);
        buttons.Controls.Add(delete);

        root.Controls.Add(domainPanel, 0, 0);
        root.Controls.Add(requestPanel, 1, 0);
        root.Controls.Add(buttons, 0, 1);
        root.SetColumnSpan(buttons, 2);
        Controls.Add(root);
    }

    private sealed class DomainItem
    {
        public DomainItem(ProjectDomainInfo value) => Value = value;
        public ProjectDomainInfo Value { get; }
        public override string ToString() => $"{Value.Domain}｜{PlayPageDisplay.Status(Value.Status)}";
    }

    private sealed class DeleteRequestItem
    {
        public DeleteRequestItem(ProjectDomainDeleteRequestInfo value) => Value = value;
        public ProjectDomainDeleteRequestInfo Value { get; }
        public override string ToString() => $"{Value.Domain}｜{PlayPageDisplay.Status(Value.Status)}｜{Value.CreatedAt.LocalDateTime:yyyy-MM-dd HH:mm}";
    }
}
