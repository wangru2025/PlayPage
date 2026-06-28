using Android.App;
using Android.Content;
using Android.OS;
using Android.Widget;
using PlayPage.Core;

namespace PlayPage.Client.Android;

public sealed partial class MainActivity
{
    private async System.Threading.Tasks.Task InitializeSessionAsync()
    {
        SetMainActionsEnabled(false);
        try
        {
            var token = LoadToken();
            if (!string.IsNullOrWhiteSpace(token))
            {
                _client.SetToken(token);
                _currentUser = await _client.GetMeAsync();
                if (!await EnsureProfileAsync()) return;
                UpdateAccountLabel();
                SetMainActionsEnabled(true);
                await LoadProjectsAsync();
                return;
            }
        }
        catch
        {
            SaveToken("");
            _client.SetToken(null);
            _currentUser = null;
        }

        await ShowLoginDialogAsync();
    }

    private string LoadToken() => GetSharedPreferences("playpage", FileCreationMode.Private)?.GetString("accessToken", "") ?? "";

    private void SaveToken(string token)
    {
        var editor = GetSharedPreferences("playpage", FileCreationMode.Private)?.Edit();
        if (editor == null) return;
        if (string.IsNullOrWhiteSpace(token)) editor.Remove("accessToken"); else editor.PutString("accessToken", token.Trim());
        editor.Apply();
    }

    private async System.Threading.Tasks.Task LogoutAsync()
    {
        try
        {
            await _client.LogoutAsync();
        }
        catch
        {
            // 本地退出优先。
        }
        SaveToken("");
        _client.SetToken(null);
        _currentUser = null;
        _projectItems.Clear();
        _adapter?.Clear();
        SetMainActionsEnabled(false);
        UpdateAccountLabel();
        await ShowLoginDialogAsync();
    }

    private async System.Threading.Tasks.Task ShowLoginDialogAsync()
    {
        var layout = new LinearLayout(this) { Orientation = Orientation.Vertical };
        layout.SetPadding(32, 12, 32, 0);

        var email = new EditText(this) { Hint = "邮箱" };
        email.InputType = global::Android.Text.InputTypes.TextVariationEmailAddress;
        email.ContentDescription = "邮箱";
        layout.AddView(email);

        var code = new EditText(this) { Hint = "验证码" };
        code.InputType = global::Android.Text.InputTypes.ClassNumber;
        code.ContentDescription = "验证码";
        layout.AddView(code);

        var sendCode = new Button(this) { Text = "发送验证码" };
        sendCode.ContentDescription = "发送验证码";
        layout.AddView(sendCode);

        var status = new TextView(this) { Text = "请输入邮箱，发送验证码后登录。" };
        status.ContentDescription = "登录状态";
        layout.AddView(status);

        var tcs = new System.Threading.Tasks.TaskCompletionSource<bool>();
        var dialog = new AlertDialog.Builder(this)
            .SetTitle("登录 PlayPage")
            .SetView(layout)
            .SetPositiveButton("登录", (sender, _) => { })
            .SetNegativeButton("退出", (_, _) => tcs.TrySetResult(false))
            .Create();

        var countdown = 0;
        Handler? handler = new Handler(Looper.MainLooper!);
        System.Action? tick = null;
        tick = () =>
        {
            if (countdown <= 0)
            {
                sendCode.Enabled = true;
                sendCode.Text = "发送验证码";
                return;
            }
            sendCode.Enabled = false;
            sendCode.Text = $"重新发送（{countdown} 秒）";
            countdown--;
            handler?.PostDelayed(tick, 1000);
        };

        sendCode.Click += async (_, _) =>
        {
            if (countdown > 0) return;
            var emailText = email.Text?.Trim() ?? "";
            if (emailText.Length == 0)
            {
                status.Text = "请先填写邮箱。";
                return;
            }
            try
            {
                // 发送验证码只禁用发送按钮，邮箱和验证码输入框必须保持可编辑。
                email.Enabled = true;
                code.Enabled = true;
                sendCode.Enabled = false;
                status.Text = "正在发送验证码。";
                var response = await _client.RequestCodeAsync(emailText);
                status.Text = "验证码已经发送到 " + response.Email + "。";
                countdown = 60;
                tick();
            }
            catch (System.Exception ex)
            {
                status.Text = "发送验证码失败：" + ex.Message;
                email.Enabled = true;
                code.Enabled = true;
                sendCode.Enabled = true;
                Toast.MakeText(this, ex.Message, ToastLength.Long)?.Show();
            }
        };

        dialog.SetOnShowListener(new DialogShowListener(() =>
        {
            var loginButton = dialog.GetButton((int)DialogButtonType.Positive);
            loginButton.Click += async (_, _) =>
            {
                var emailText = email.Text?.Trim() ?? "";
                var codeText = code.Text?.Trim() ?? "";
                if (emailText.Length == 0)
                {
                    status.Text = "请先填写邮箱。";
                    return;
                }
                if (codeText.Length == 0)
                {
                    status.Text = "请先填写验证码。";
                    return;
                }
                try
                {
                    loginButton.Enabled = false;
                    status.Text = "正在登录。";
                    var result = await _client.VerifyCodeAsync(emailText, codeText);
                    SaveToken(result.AccessToken);
                    _currentUser = result.User;
                    dialog.Dismiss();
                    handler = null;
                    if (!await EnsureProfileAsync())
                    {
                        tcs.TrySetResult(false);
                        return;
                    }
                    UpdateAccountLabel();
                    SetMainActionsEnabled(true);
                    await LoadProjectsAsync();
                    tcs.TrySetResult(true);
                }
                catch (System.Exception ex)
                {
                    status.Text = "登录失败：" + ex.Message;
                    loginButton.Enabled = true;
                    Toast.MakeText(this, ex.Message, ToastLength.Long)?.Show();
                }
            };
        }));

        dialog.SetCanceledOnTouchOutside(false);
        dialog.Show();
        await tcs.Task;
    }

    private async System.Threading.Tasks.Task<bool> EnsureProfileAsync()
    {
        while (_currentUser != null && string.IsNullOrWhiteSpace(_currentUser.Username))
        {
            var username = await PromptAsync("设置公开名字", "请输入你的公开名字");
            if (string.IsNullOrWhiteSpace(username))
            {
                Toast.MakeText(this, "新账号需要先设置公开名字。", ToastLength.Long)?.Show();
                continue;
            }
            try
            {
                _currentUser = await _client.UpdateProfileAsync(username.Trim());
                SetStatus("公开名字已保存。");
            }
            catch (System.Exception ex)
            {
                ShowError(ex);
            }
        }
        return _currentUser != null && !string.IsNullOrWhiteSpace(_currentUser.Username);
    }

    private sealed class DialogShowListener : Java.Lang.Object, IDialogInterfaceOnShowListener
    {
        private readonly System.Action _action;
        public DialogShowListener(System.Action action) => _action = action;
        public void OnShow(IDialogInterface? dialog) => _action();
    }
}
