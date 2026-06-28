using System.Drawing;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

public sealed class UpgradePlanForm : Form
{
    private readonly RadioButton _light = new RadioButton();
    private readonly RadioButton _support = new RadioButton();
    private readonly RadioButton _wechat = new RadioButton();
    private readonly RadioButton _alipay = new RadioButton();
    private readonly TextBox _note = new TextBox();

    public UpgradePlanResult? Result { get; private set; }

    public UpgradePlanForm()
    {
        Text = "开通或升级套餐";
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(720, 560);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "开通或升级套餐窗口";

        var root = new TableLayoutPanel { Dock = DockStyle.Fill, ColumnCount = 1, RowCount = 5, Padding = new Padding(16) };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        root.Controls.Add(new Label
        {
            Text = "先看清每个套餐的价格和权益，再选择支付方式继续。",
            AutoSize = true,
            MaximumSize = new Size(660, 0)
        }, 0, 0);

        var planPanel = new TableLayoutPanel { Dock = DockStyle.Fill, ColumnCount = 2, RowCount = 1 };
        planPanel.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));
        planPanel.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));
        _light.Text = "轻享版，3 元 / 月。最多 10 个作品；每作品 30 MB 互动数据；每月 100000 次查询；每月 10000 次写入。";
        _light.Checked = true;
        _light.AutoSize = true;
        _light.MaximumSize = new Size(310, 0);
        _support.Text = "支持版，6 元 / 月。最多 30 个作品；每作品 100 MB 互动数据；每月 500000 次查询；每月 50000 次写入。";
        _support.AutoSize = true;
        _support.MaximumSize = new Size(310, 0);
        planPanel.Controls.Add(WrapOption("轻享版", _light), 0, 0);
        planPanel.Controls.Add(WrapOption("支持版", _support), 1, 0);
        root.Controls.Add(planPanel, 0, 1);

        var payPanel = new FlowLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, FlowDirection = FlowDirection.LeftToRight };
        payPanel.Controls.Add(new Label { Text = "支付方式：", AutoSize = true, Margin = new Padding(0, 6, 12, 6) });
        _wechat.Text = "微信支付";
        _wechat.Checked = true;
        _wechat.AutoSize = true;
        _alipay.Text = "支付宝";
        _alipay.AutoSize = true;
        payPanel.Controls.Add(_wechat);
        payPanel.Controls.Add(_alipay);
        root.Controls.Add(payPanel, 0, 2);

        var notePanel = new TableLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, ColumnCount = 1, RowCount = 2 };
        notePanel.Controls.Add(new Label { Text = "付款备注、转账昵称或其他说明，可留空：", AutoSize = true }, 0, 0);
        _note.Dock = DockStyle.Top;
        _note.AccessibleName = "付款备注";
        notePanel.Controls.Add(_note, 0, 1);
        root.Controls.Add(notePanel, 0, 3);

        var buttons = new FlowLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, FlowDirection = FlowDirection.RightToLeft };
        var cancel = new Button { Text = "取消", AutoSize = true, DialogResult = DialogResult.Cancel };
        var ok = new Button { Text = "提交申请", AutoSize = true };
        ok.Click += (_, _) => Submit();
        buttons.Controls.Add(cancel);
        buttons.Controls.Add(ok);
        root.Controls.Add(buttons, 0, 4);

        Controls.Add(root);
        CancelButton = cancel;
    }

    private static Control WrapOption(string title, RadioButton option)
    {
        var box = new GroupBox { Text = title, Dock = DockStyle.Fill, Padding = new Padding(12) };
        option.Dock = DockStyle.Fill;
        box.Controls.Add(option);
        return box;
    }

    private void Submit()
    {
        Result = new UpgradePlanResult
        {
            TargetPlan = _support.Checked ? "support" : "light",
            PaymentMethod = _alipay.Checked ? "alipay" : "wechat",
            PayerNote = _note.Text.Trim()
        };
        DialogResult = DialogResult.OK;
        Close();
    }
}

public sealed class UpgradePlanResult
{
    public string TargetPlan { get; set; } = "light";
    public string PaymentMethod { get; set; } = "wechat";
    public string PayerNote { get; set; } = "";
}
