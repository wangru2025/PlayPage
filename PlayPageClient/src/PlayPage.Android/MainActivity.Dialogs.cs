using Android.App;
using Android.Views;
using Android.Widget;

namespace PlayPage.Client.Android;

public sealed partial class MainActivity
{
    private System.Threading.Tasks.Task<string> PromptAsync(string title, string hint)
    {
        var tcs = new System.Threading.Tasks.TaskCompletionSource<string>();
        var input = new EditText(this) { Hint = hint };
        input.ContentDescription = hint;
        new AlertDialog.Builder(this)
            .SetTitle(title)
            .SetView(input)
            .SetPositiveButton("确定", (_, _) => tcs.TrySetResult(input.Text ?? ""))
            .SetNegativeButton("取消", (_, _) => tcs.TrySetResult(""))
            .Show();
        return tcs.Task;
    }

    private void ShowMessage(string title, string message) => new AlertDialog.Builder(this).SetTitle(title).SetMessage(message).SetPositiveButton("确定", (_, _) => { }).Show();

    private void ShowError(System.Exception ex)
    {
        SetStatus("操作失败：" + ex.Message);
        Toast.MakeText(this, ex.Message, ToastLength.Long)?.Show();
    }

    private void SetStatus(string text)
    {
        if (_status == null) return;
        _status.Text = text;
        _status.ContentDescription = "状态：" + text;
        _status.SendAccessibilityEvent(global::Android.Views.Accessibility.EventTypes.Announcement);
    }
}
