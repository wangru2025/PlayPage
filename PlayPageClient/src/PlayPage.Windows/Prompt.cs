using System.Drawing;
using System.Windows.Forms;

namespace PlayPage.Windows;

internal static class Prompt
{
    public static string Show(IWin32Window owner, string title, string label, string defaultValue = "")
    {
        using var form = new Form
        {
            Text = title,
            StartPosition = FormStartPosition.CenterParent,
            Width = 520,
            Height = 160,
            MinimizeBox = false,
            MaximizeBox = false,
            FormBorderStyle = FormBorderStyle.FixedDialog
        };
        var textLabel = new Label { Text = label, Left = 12, Top = 12, Width = 480, AutoSize = true };
        var input = new TextBox { Left = 12, Top = 40, Width = 480, Text = defaultValue, AccessibleName = label };
        var ok = new Button { Text = "确定", Left = 320, Width = 80, Top = 76, DialogResult = DialogResult.OK };
        var cancel = new Button { Text = "取消", Left = 412, Width = 80, Top = 76, DialogResult = DialogResult.Cancel };
        form.Controls.AddRange(new Control[] { textLabel, input, ok, cancel });
        form.AcceptButton = ok;
        form.CancelButton = cancel;
        input.SelectAll();
        return form.ShowDialog(owner) == DialogResult.OK ? input.Text.Trim() : "";
    }
}
