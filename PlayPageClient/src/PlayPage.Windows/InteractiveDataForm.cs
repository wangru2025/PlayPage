using System;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using System.Text.Encodings.Web;
using System.Text.Json;
using System.Threading.Tasks;
using System.Windows.Forms;
using PlayPage.Core;

namespace PlayPage.Windows;

public sealed class InteractiveDataForm : Form
{
    private static readonly JsonSerializerOptions JsonOptions = new JsonSerializerOptions
    {
        WriteIndented = true,
        Encoder = JavaScriptEncoder.UnsafeRelaxedJsonEscaping,
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
        PropertyNameCaseInsensitive = true
    };

    private readonly PlayPageApiClient _client;
    private readonly ProjectSummary _project;
    private readonly ListBox _collections = new ListBox();
    private readonly ListView _records = new ListView();
    private readonly TextBox _schemaJson = new TextBox();
    private readonly TextBox _recordJson = new TextBox();
    private readonly Label _status = new Label();
    private CollectionInfo? _selectedCollection;

    public InteractiveDataForm(PlayPageApiClient client, ProjectSummary project)
    {
        _client = client;
        _project = project;

        Text = "管理互动数据 - " + project.Name;
        StartPosition = FormStartPosition.CenterParent;
        MinimumSize = new Size(1060, 700);
        AutoScaleMode = AutoScaleMode.Font;
        AccessibleName = "互动数据管理窗口";

        Controls.Add(BuildLayout());
        Load += async (_, _) => await RefreshCollectionsAsync();
    }

    private Control BuildLayout()
    {
        var root = new TableLayoutPanel { Dock = DockStyle.Fill, ColumnCount = 2, RowCount = 2, Padding = new Padding(12) };
        root.ColumnStyles.Add(new ColumnStyle(SizeType.Absolute, 280));
        root.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        var left = new TableLayoutPanel { Dock = DockStyle.Fill, RowCount = 4, ColumnCount = 1 };
        left.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        left.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        left.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        left.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        left.Controls.Add(new Label { Text = "数据集合", AutoSize = true }, 0, 0);
        _collections.Dock = DockStyle.Fill;
        _collections.AccessibleName = "数据集合列表";
        _collections.SelectedIndexChanged += async (_, _) => await CollectionSelectionChangedAsync();
        left.Controls.Add(_collections, 0, 1);

        var refresh = new Button { Text = "刷新集合(&R)", AutoSize = true, AccessibleName = "刷新集合" };
        refresh.Click += async (_, _) => await RefreshCollectionsAsync();
        left.Controls.Add(refresh, 0, 2);

        var createHelp = new Label
        {
            Text = "提示：左侧选择集合后，可查看或修改集合结构；右侧可读取和新增记录。",
            AutoSize = true,
            MaximumSize = new Size(260, 0)
        };
        left.Controls.Add(createHelp, 0, 3);

        var tabs = new TabControl { Dock = DockStyle.Fill, AccessibleName = "互动数据操作区" };
        tabs.TabPages.Add(BuildSchemaPage());
        tabs.TabPages.Add(BuildRecordsPage());

        _status.AutoSize = true;
        _status.Dock = DockStyle.Fill;
        _status.AccessibleName = "操作状态";
        _status.Text = "正在读取数据集合。";

        root.Controls.Add(left, 0, 0);
        root.Controls.Add(tabs, 1, 0);
        root.Controls.Add(_status, 0, 1);
        root.SetColumnSpan(_status, 2);
        return root;
    }

    private TabPage BuildSchemaPage()
    {
        var page = new TabPage("集合结构");
        var root = new TableLayoutPanel { Dock = DockStyle.Fill, RowCount = 3, ColumnCount = 1, Padding = new Padding(8) };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));

        root.Controls.Add(new Label { Text = "集合 JSON。新建时需要 name、permissions、fields；更新时使用 permissions、fields。", AutoSize = true }, 0, 0);
        _schemaJson.Multiline = true;
        _schemaJson.ScrollBars = ScrollBars.Both;
        _schemaJson.AcceptsReturn = true;
        _schemaJson.AcceptsTab = true;
        _schemaJson.WordWrap = false;
        _schemaJson.Dock = DockStyle.Fill;
        _schemaJson.Font = new Font(FontFamily.GenericMonospace, 10);
        _schemaJson.AccessibleName = "集合结构 JSON";
        root.Controls.Add(_schemaJson, 0, 1);

        var actions = new FlowLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, FlowDirection = FlowDirection.LeftToRight };
        var preset = new Button { Text = "填入示例(&P)", AutoSize = true };
        preset.Click += (_, _) => FillCollectionPreset();
        var create = new Button { Text = "新建集合(&N)", AutoSize = true };
        create.Click += async (_, _) => await CreateCollectionAsync();
        var update = new Button { Text = "保存集合结构(&S)", AutoSize = true };
        update.Click += async (_, _) => await UpdateCollectionAsync();
        actions.Controls.Add(preset);
        actions.Controls.Add(create);
        actions.Controls.Add(update);
        root.Controls.Add(actions, 0, 2);

        page.Controls.Add(root);
        return page;
    }

    private TabPage BuildRecordsPage()
    {
        var page = new TabPage("记录");
        var root = new TableLayoutPanel { Dock = DockStyle.Fill, RowCount = 4, ColumnCount = 1, Padding = new Padding(8) };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 55));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 45));

        var topActions = new FlowLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, FlowDirection = FlowDirection.LeftToRight };
        var refresh = new Button { Text = "刷新记录(&R)", AutoSize = true };
        refresh.Click += async (_, _) => await RefreshRecordsAsync();
        topActions.Controls.Add(refresh);
        root.Controls.Add(topActions, 0, 0);

        _records.Dock = DockStyle.Fill;
        _records.View = View.Details;
        _records.FullRowSelect = true;
        _records.MultiSelect = false;
        _records.AccessibleName = "记录列表";
        _records.Columns.Add("ID", 220);
        _records.Columns.Add("状态", 80);
        _records.Columns.Add("创建时间", 160);
        _records.Columns.Add("数据", 520);
        root.Controls.Add(_records, 0, 1);

        root.Controls.Add(new Label { Text = "新增记录 data JSON。只填写 data 对象，不要外包一层 data。", AutoSize = true }, 0, 2);
        var recordPanel = new TableLayoutPanel { Dock = DockStyle.Fill, RowCount = 2, ColumnCount = 1 };
        recordPanel.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        recordPanel.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        _recordJson.Multiline = true;
        _recordJson.ScrollBars = ScrollBars.Both;
        _recordJson.AcceptsReturn = true;
        _recordJson.AcceptsTab = true;
        _recordJson.WordWrap = false;
        _recordJson.Dock = DockStyle.Fill;
        _recordJson.Font = new Font(FontFamily.GenericMonospace, 10);
        _recordJson.AccessibleName = "新增记录 JSON";
        recordPanel.Controls.Add(_recordJson, 0, 0);
        var createRecord = new Button { Text = "新增记录(&A)", AutoSize = true, Anchor = AnchorStyles.Left };
        createRecord.Click += async (_, _) => await CreateRecordAsync();
        recordPanel.Controls.Add(createRecord, 0, 1);
        root.Controls.Add(recordPanel, 0, 3);

        page.Controls.Add(root);
        return page;
    }

    private async Task RefreshCollectionsAsync()
    {
        try
        {
            SetStatus("正在读取数据集合。", false);
            var items = await _client.ListCollectionsAsync(_project.Id);
            _collections.Items.Clear();
            foreach (var item in items.OrderBy(x => x.Name)) _collections.Items.Add(new CollectionListItem(item));
            if (_collections.Items.Count > 0) _collections.SelectedIndex = 0;
            else
            {
                _selectedCollection = null;
                _schemaJson.Text = "";
                _records.Items.Clear();
                FillCollectionPreset();
            }
            SetStatus($"已读取 {items.Count} 个数据集合。", false);
        }
        catch (Exception ex) { ShowError(ex); }
    }

    private async Task CollectionSelectionChangedAsync()
    {
        if (_collections.SelectedItem is not CollectionListItem item) return;
        _selectedCollection = item.Value;
        _schemaJson.Text = JsonSerializer.Serialize(new CollectionUpdateRequest
        {
            Permissions = _selectedCollection.Permissions,
            Fields = _selectedCollection.Fields
        }, JsonOptions);
        await RefreshRecordsAsync();
    }

    private async Task RefreshRecordsAsync()
    {
        if (_selectedCollection == null)
        {
            SetStatus("请先选择一个数据集合。", true);
            return;
        }
        try
        {
            SetStatus("正在读取记录。", false);
            var records = await _client.ListRecordsAsync(_project.Id, _selectedCollection.Name);
            _records.Items.Clear();
            foreach (var record in records.OrderByDescending(x => x.CreatedAt))
            {
                var row = new ListViewItem(record.Id);
                row.SubItems.Add(record.Status);
                row.SubItems.Add(record.CreatedAt.LocalDateTime.ToString("yyyy-MM-dd HH:mm:ss"));
                row.SubItems.Add(JsonSerializer.Serialize(record.Data, JsonOptions).Replace("\r", "").Replace("\n", " "));
                _records.Items.Add(row);
            }
            SetStatus($"集合 {_selectedCollection.Name} 已读取 {records.Count} 条记录。", false);
        }
        catch (Exception ex) { ShowError(ex); }
    }

    private async Task CreateCollectionAsync()
    {
        try
        {
            var request = JsonSerializer.Deserialize<CollectionCreateRequest>(_schemaJson.Text, JsonOptions);
            if (request == null || string.IsNullOrWhiteSpace(request.Name))
            {
                SetStatus("新建集合需要填写 name。", true);
                return;
            }
            await _client.CreateCollectionAsync(_project.Id, request);
            SetStatus("数据集合已创建。", false);
            await RefreshCollectionsAsync();
        }
        catch (Exception ex) { ShowError(ex); }
    }

    private async Task UpdateCollectionAsync()
    {
        if (_selectedCollection == null)
        {
            SetStatus("请先选择要更新的集合。", true);
            return;
        }
        try
        {
            var request = JsonSerializer.Deserialize<CollectionUpdateRequest>(_schemaJson.Text, JsonOptions);
            if (request == null)
            {
                SetStatus("集合结构 JSON 无效。", true);
                return;
            }
            await _client.UpdateCollectionAsync(_project.Id, _selectedCollection.Name, request);
            SetStatus("集合结构已保存。", false);
            await RefreshCollectionsAsync();
        }
        catch (Exception ex) { ShowError(ex); }
    }

    private async Task CreateRecordAsync()
    {
        if (_selectedCollection == null)
        {
            SetStatus("请先选择一个数据集合。", true);
            return;
        }
        try
        {
            var data = JsonSerializer.Deserialize<Dictionary<string, object?>>(_recordJson.Text, JsonOptions);
            if (data == null)
            {
                SetStatus("记录 JSON 无效。", true);
                return;
            }
            await _client.CreateRecordAsync(_project.Id, _selectedCollection.Name, data);
            _recordJson.Text = "";
            SetStatus("记录已新增。", false);
            await RefreshRecordsAsync();
        }
        catch (Exception ex) { ShowError(ex); }
    }

    private void FillCollectionPreset()
    {
        _schemaJson.Text = "{\r\n  \"name\": \"messages\",\r\n  \"permissions\": { \"publicRead\": true, \"publicWrite\": true },\r\n  \"fields\": [\r\n    { \"name\": \"nickname\", \"type\": \"string\", \"required\": true, \"isList\": false },\r\n    { \"name\": \"content\", \"type\": \"text\", \"required\": true, \"isList\": false }\r\n  ]\r\n}";
        _recordJson.Text = "{\r\n  \"nickname\": \"访客\",\r\n  \"content\": \"你好，PlayPage！\"\r\n}";
    }

    private void SetStatus(string text, bool assertive)
    {
        _status.Text = text;
        _status.AccessibleName = text;
        if (assertive) System.Media.SystemSounds.Exclamation.Play();
    }

    private void ShowError(Exception ex)
    {
        SetStatus(ex.Message, true);
        MessageBox.Show(this, ex.Message, "PlayPage", MessageBoxButtons.OK, MessageBoxIcon.Error);
    }

    private sealed class CollectionListItem
    {
        public CollectionListItem(CollectionInfo value) => Value = value;
        public CollectionInfo Value { get; }
        public override string ToString() => $"{Value.Name}（{Value.Fields.Count} 个字段）";
    }
}
