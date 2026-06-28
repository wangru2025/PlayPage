using System;
using System.Drawing;
using System.Threading.Tasks;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

public sealed class LoginForm : Form
{
    private readonly PlayPageApiClient _client;
    private readonly TextBox _email = new TextBox();
    private readonly TextBox _code = new TextBox();
    private readonly Button _sendCode = new Button();
    private readonly Button _login = new Button();
    private readonly Label _status = new Label();
    private readonly Timer _timer = new Timer();
    private int _resendSeconds;

    public AuthResult? Result { get; private set; }

    public LoginForm(PlayPageApiClient client)
    {
        _client = client;
        Text = "登录 PlayPage";
        StartPosition = FormStartPosition.CenterParent;
        FormBorderStyle = FormBorderStyle.FixedDialog;
        MinimizeBox = false;
        MaximizeBox = false;
        ClientSize = new Size(460, 245);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "PlayPage 登录窗口";

        var root = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            Padding = new Padding(16),
            ColumnCount = 2,
            RowCount = 5
        };
        root.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        root.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));

        var emailLabel = new Label { Text = "邮箱(&E)：", AutoSize = true, Anchor = AnchorStyles.Left, Margin = new Padding(0, 6, 8, 6) };
        _email.Anchor = AnchorStyles.Left | AnchorStyles.Right;
        _email.AccessibleName = "邮箱";
        _email.AutoCompleteMode = AutoCompleteMode.SuggestAppend;
        _email.AutoCompleteSource = AutoCompleteSource.RecentlyUsedList;

        var codeLabel = new Label { Text = "验证码(&C)：", AutoSize = true, Anchor = AnchorStyles.Left, Margin = new Padding(0, 6, 8, 6) };
        _code.Anchor = AnchorStyles.Left | AnchorStyles.Right;
        _code.AccessibleName = "验证码";

        _sendCode.Text = "发送验证码(&S)";
        _sendCode.AccessibleName = "发送验证码";
        _sendCode.AutoSize = true;
        _sendCode.Click += async (_, _) => await SendCodeAsync();

        _login.Text = "登录(&L)";
        _login.AccessibleName = "登录";
        _login.AutoSize = true;
        _login.Click += async (_, _) => await LoginAsync();

        var actions = new FlowLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, FlowDirection = FlowDirection.LeftToRight };
        actions.Controls.Add(_sendCode);
        actions.Controls.Add(_login);

        _status.AutoSize = true;
        _status.Dock = DockStyle.Fill;
        _status.AccessibleName = "登录状态";
        _status.Text = "请输入邮箱，发送验证码后登录。";

        root.Controls.Add(emailLabel, 0, 0);
        root.Controls.Add(_email, 1, 0);
        root.Controls.Add(codeLabel, 0, 1);
        root.Controls.Add(_code, 1, 1);
        root.Controls.Add(actions, 1, 2);
        root.Controls.Add(_status, 0, 3);
        root.SetColumnSpan(_status, 2);

        Controls.Add(root);
        AcceptButton = _login;

        _timer.Interval = 1000;
        _timer.Tick += (_, _) => TickCountdown();
    }

    protected override void OnShown(EventArgs e)
    {
        base.OnShown(e);
        _email.Focus();
    }

    private async Task SendCodeAsync()
    {
        if (_resendSeconds > 0) return;
        var email = _email.Text.Trim();
        if (email.Length == 0)
        {
            SetStatus("请先填写邮箱。", true);
            _email.Focus();
            return;
        }

        try
        {
            SetBusy(true, "正在发送验证码。");
            var response = await _client.RequestCodeAsync(email);
            StartCountdown(60);
            SetStatus($"验证码已经发送到 {response.Email}，请查看邮箱。", false);
            _code.Focus();
        }
        catch (Exception ex)
        {
            SetStatus("发送验证码失败：" + ex.Message, true);
            MessageBox.Show(this, ex.Message, "PlayPage", MessageBoxButtons.OK, MessageBoxIcon.Error);
            _sendCode.Enabled = true;
        }
        finally
        {
            _login.Enabled = true;
        }
    }

    private async Task LoginAsync()
    {
        var email = _email.Text.Trim();
        var code = _code.Text.Trim();
        if (email.Length == 0)
        {
            SetStatus("请先填写邮箱。", true);
            _email.Focus();
            return;
        }
        if (code.Length == 0)
        {
            SetStatus("请先填写验证码。", true);
            _code.Focus();
            return;
        }

        try
        {
            SetBusy(true, "正在登录。");
            Result = await _client.VerifyCodeAsync(email, code);
            DialogResult = DialogResult.OK;
            Close();
        }
        catch (Exception ex)
        {
            SetStatus("登录失败：" + ex.Message, true);
            MessageBox.Show(this, ex.Message, "PlayPage", MessageBoxButtons.OK, MessageBoxIcon.Error);
            SetBusy(false, _status.Text);
        }
    }

    private void SetBusy(bool busy, string status)
    {
        _email.Enabled = !busy;
        _code.Enabled = !busy;
        _login.Enabled = !busy;
        _sendCode.Enabled = !busy && _resendSeconds <= 0;
        SetStatus(status, false);
    }

    private void StartCountdown(int seconds)
    {
        _resendSeconds = seconds;
        _sendCode.Enabled = false;
        UpdateSendCodeText();
        _timer.Start();
    }

    private void TickCountdown()
    {
        if (_resendSeconds > 0) _resendSeconds--;
        if (_resendSeconds <= 0)
        {
            _timer.Stop();
            _sendCode.Enabled = true;
        }
        UpdateSendCodeText();
    }

    private void UpdateSendCodeText()
    {
        _sendCode.Text = _resendSeconds > 0 ? $"重新发送（{_resendSeconds} 秒）" : "发送验证码(&S)";
    }

    private void SetStatus(string text, bool assertive)
    {
        _status.Text = text;
        _status.AccessibleName = text;
        if (assertive) System.Media.SystemSounds.Exclamation.Play();
    }
}
