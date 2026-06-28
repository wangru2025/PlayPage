using Android.App;
using Android.OS;
using Android.Views;
using Android.Widget;
using PlayPage.Core;

namespace PlayPage.Android;

[Activity(Label = "PlayPage", MainLauncher = true, Exported = true)]
public sealed class MainActivity : Activity
{
    private readonly PlayPageApiClient _client = new PlayPageApiClient(PlayPageOptions.Production);
    private EditText? _email;
    private EditText? _password;
    private Button? _login;
    private TextView? _status;
    private ListView? _projects;
    private ArrayAdapter<string>? _adapter;

    protected override void OnCreate(Bundle? savedInstanceState)
    {
        base.OnCreate(savedInstanceState);
        Title = "PlayPage 客户端";

        var root = new LinearLayout(this) { Orientation = Orientation.Vertical };
        root.SetPadding(32, 32, 32, 32);

        _email = new EditText(this) { Hint = "邮箱" };
        _email.InputType = Android.Text.InputTypes.TextVariationEmailAddress;
        _email.ContentDescription = "邮箱";
        root.AddView(_email, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _password = new EditText(this) { Hint = "密码" };
        _password.InputType = Android.Text.InputTypes.ClassText | Android.Text.InputTypes.TextVariationPassword;
        _password.ContentDescription = "密码";
        root.AddView(_password, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _login = new Button(this) { Text = "登录" };
        _login.ContentDescription = "登录 PlayPage 并读取我的作品列表";
        _login.Click += async (_, _) => await LoginAsync();
        root.AddView(_login, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _status = new TextView(this) { Text = "请输入账号密码登录。" };
        _status.ContentDescription = "状态：请输入账号密码登录。";
        root.AddView(_status, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, ViewGroup.LayoutParams.WrapContent));

        _projects = new ListView(this);
        _projects.ContentDescription = "我的作品列表";
        _adapter = new ArrayAdapter<string>(this, global::Android.Resource.Layout.SimpleListItem1);
        _projects.Adapter = _adapter;
        root.AddView(_projects, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MatchParent, 0, 1));

        SetContentView(root);
    }

    private async System.Threading.Tasks.Task LoginAsync()
    {
        try
        {
            if (_login != null) _login.Enabled = false;
            SetStatus("正在登录。");
            await _client.LoginAsync(_email?.Text ?? "", _password?.Text ?? "");
            SetStatus("登录成功，正在读取作品。");
            var projects = await _client.GetProjectsAsync();
            _adapter?.Clear();
            foreach (var project in projects) _adapter?.Add($"{project.Name}（{project.Slug}）");
            SetStatus($"已读取 {projects.Count} 个作品。");
        }
        catch (System.Exception ex)
        {
            SetStatus("操作失败：" + ex.Message);
            Toast.MakeText(this, ex.Message, ToastLength.Long)?.Show();
        }
        finally
        {
            if (_login != null) _login.Enabled = true;
        }
    }

    private void SetStatus(string text)
    {
        if (_status == null) return;
        _status.Text = text;
        _status.ContentDescription = "状态：" + text;
        _status.SendAccessibilityEvent(Android.Views.Accessibility.EventTypes.Announcement);
    }
}
