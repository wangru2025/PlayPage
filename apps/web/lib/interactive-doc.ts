export type InteractiveProject = {
  id: string;
  name: string;
  publicUrl: string;
};

export type InteractiveFieldSchema = {
  name: string;
  type: string;
  required: boolean;
  isList: boolean;
  reference?: string;
};

export type InteractivePermissionSet = {
  publicRead: boolean;
  publicWrite: boolean;
};

export type InteractiveCollection = {
  id: string;
  name: string;
  permissions: InteractivePermissionSet;
  fields: InteractiveFieldSchema[];
};

type BuildInteractiveDocOptions = {
  project: InteractiveProject;
  projectKey: string;
  collections?: InteractiveCollection[];
};

function codeBlock(language: string, content: string): string {
  return "```" + language + "\n" + content + "\n```";
}

function yesNo(value: boolean): string {
  return value ? "是" : "否";
}

function jsonExample(value: unknown): string {
  return codeBlock("json", JSON.stringify(value, null, 2));
}

function buildStandardFetchExample(apiBase: string, projectKey: string): string {
  return [
    `const API_BASE = '${apiBase}';`,
    `const PROJECT_KEY = '${projectKey}';`,
    "",
    "async function apiFetch(path, options = {}) {",
    "  const headers = {",
    "    'X-Project-Key': PROJECT_KEY,",
    "    ...(options.headers || {})",
    "  };",
    "",
    "  if (options.body && !headers['Content-Type']) {",
    "    headers['Content-Type'] = 'application/json';",
    "  }",
    "",
    "  const response = await fetch(`${API_BASE}${path}`, {",
    "    ...options,",
    "    headers",
    "  });",
    "",
    "  let data = null;",
    "  try {",
    "    data = await response.json();",
    "  } catch {",
    "    data = null;",
    "  }",
    "",
    "  if (!response.ok) {",
    "    const error = new Error(data?.error || `请求失败，状态码：${response.status}`);",
    "    error.status = response.status;",
    "    error.data = data;",
    "    throw error;",
    "  }",
    "",
    "  return data;",
    "}",
  ].join("\n");
}

function buildEnsureCollectionExample(): string {
  return [
    "async function ensureCollection(name, fields) {",
    "  try {",
    "    return await apiFetch(`/collections/${encodeURIComponent(name)}`);",
    "  } catch (error) {",
    "    // 数据表不存在时，平台会返回 HTTP 404。请判断 status，不要只判断英文错误文本。",
    "    if (error.status !== 404) {",
    "      throw error;",
    "    }",
    "  }",
    "",
    "  return apiFetch('/collections', {",
    "    method: 'POST',",
    "    body: JSON.stringify({",
    "      name,",
    "      permissions: { publicRead: true, publicWrite: true },",
    "      fields",
    "    })",
    "  });",
    "}",
    "",
    "// 示例：初始化帖子表。",
    "await ensureCollection('posts', [",
    "  { name: 'title', type: 'string', required: true, isList: false },",
    "  { name: 'content', type: 'text', required: true, isList: false }",
    "]);",
  ].join("\n");
}

export function buildInteractiveApiDoc({ project, projectKey, collections = [] }: BuildInteractiveDocOptions): string {
  const apiBase = `/api/v1/public/projects/${project.id}`;
  const collectionPath = `${apiBase}/collections/{collectionName}`;
  const recordPath = `${collectionPath}/records/{recordId}`;
  const lines: string[] = [];

  lines.push(`# ${project.name} 互动 API 文档`);
  lines.push("");
  lines.push("## 文档用途");
  lines.push("这份文档是给 AI 用的。请根据下面的作品信息、认证方式、接口、错误码和现有数据表，为这个作品编写互动功能。");
  lines.push("");

  lines.push("## 重要规则");
  lines.push("- 所有公开互动 API 请求都必须带上 `X-Project-Key` 请求头。");
  lines.push(`- API 基础地址固定使用相对路径：\`${apiBase}\`。不要使用 \'api.wangru.net\'，也不要自己猜测其他 API 域名。`);
  lines.push("- 请使用 UTF-8，并在生成的 HTML 中加入 `<meta charset=\"utf-8\">`。");
  lines.push("- 判断接口错误时，请优先使用 HTTP 状态码，不要只判断英文错误文本。平台错误提示是中文，例如“找不到这个作品数据表”。");
  lines.push("- 如果读取单个数据表返回 HTTP `404`，表示这个数据表不存在，可以创建数据表；如果是其他状态码，不要当成数据表不存在。");
  lines.push("- 如果要写入留言、评论、论坛帖子、角色数据、存档等内容，先确认数据表是否存在；不存在时先创建数据表。");
  lines.push("- 创建数据表会受作品当前套餐和用量限制影响；如果接口返回 `429` 或额度相关错误，请用中文提示用户减少数据表、减少操作频率或升级套餐，不要绕过限制。");
  lines.push("- 创建数据表失败时不要反复重试，更不要删除、清空或重建已有数据表。");
  lines.push("- 公开接口不允许删除数据表，也不要生成删除数据表、清空数据表或重建数据表的代码。已有数据表满足需求时必须复用。");
  lines.push("- 权限只使用 `publicRead` 和 `publicWrite`，不要使用登录用户、创作者或所有者专用写入权限。");
  lines.push("- 如果这个数据表需要访客通过网页新增、编辑或删除记录，创建数据表时必须设置 `publicWrite: true`。如果设置为 `false`，公开网页后续将无法写入、修改或删除这个表的记录。");
  lines.push("- 只有纯展示、不需要访客提交或修改内容的数据表，才建议设置 `publicWrite: false`。");
  lines.push("- 新增或更新记录时，业务数据统一放进 `data` 对象里。");
  lines.push("- 不要把 `X-Project-Key` 保存到 `localStorage`。如果必须写进静态网页，只把它作为当前作品的公开互动 key 使用。");
  lines.push("- 用户点击按钮、提交表单等明确操作后才发起写入请求，不要因为页面加载、鼠标悬停或输入框聚焦就写入数据。");
  lines.push("");

  lines.push("## 作品信息");
  lines.push(`- 作品 ID：\`${project.id}\``);
  lines.push(`- 作品名称：${project.name}`);
  lines.push(`- 作品地址：${project.publicUrl}`);
  lines.push("- 互动功能：已开启");
  lines.push("");

  lines.push("## 认证方式");
  lines.push("每个公开互动 API 请求都要带上这个请求头：");
  lines.push(codeBlock("http", `X-Project-Key: ${projectKey}`));
  lines.push("");

  lines.push("## API 基础地址");
  lines.push("在作品页面里调用接口时，请使用这个相对地址作为基础地址：");
  lines.push(codeBlock("js", `const API_BASE = '${apiBase}';`));
  lines.push("不要写成 `https://api.wangru.net/...`。如果必须使用完整地址，请使用当前网站域名加上上面的相对路径。");
  lines.push("");

  lines.push("## 接口列表");
  lines.push(`- 读取作品信息：\`GET ${apiBase}\``);
  lines.push(`- 读取数据表列表：\`GET ${apiBase}/collections\``);
  lines.push(`- 创建数据表：\`POST ${apiBase}/collections\``);
  lines.push(`- 读取单个数据表：\`GET ${collectionPath}\``);
  lines.push(`- 读取记录列表：\`GET ${collectionPath}/records\``);
  lines.push(`- 创建记录：\`POST ${collectionPath}/records\``);
  lines.push(`- 读取单条记录：\`GET ${recordPath}\``);
  lines.push(`- 更新记录：\`PATCH ${recordPath}\``);
  lines.push(`- 删除记录：\`DELETE ${recordPath}\``);
  lines.push("");

  lines.push("## 接口响应示例");
  lines.push("### 读取作品信息：GET " + apiBase);
  lines.push(jsonExample({
    project: {
      id: project.id,
      name: project.name,
      publicUrl: project.publicUrl,
      interactive: true
    }
  }));
  lines.push("");

  lines.push("### 读取数据表列表：GET " + apiBase + "/collections");
  lines.push(jsonExample({
    projectId: project.id,
    items: [
      {
        id: "collection-id",
        projectId: project.id,
        name: "posts",
        permissions: { publicRead: true, publicWrite: true },
        fields: [
          { name: "title", type: "string", required: true, isList: false },
          { name: "content", type: "text", required: true, isList: false }
        ],
        createdAt: "2026-01-01T00:00:00Z"
      }
    ]
  }));
  lines.push("");

  lines.push("### 创建数据表：POST " + apiBase + "/collections");
  lines.push(jsonExample({
    id: "collection-id",
    projectId: project.id,
    name: "posts",
    permissions: { publicRead: true, publicWrite: true },
    fields: [
      { name: "title", type: "string", required: true, isList: false },
      { name: "content", type: "text", required: true, isList: false }
    ],
    createdAt: "2026-01-01T00:00:00Z"
  }));
  lines.push("");

  lines.push("### 读取单个数据表：GET " + collectionPath);
  lines.push("响应格式和创建数据表返回格式相同。如果数据表不存在，会返回 HTTP `404`。");
  lines.push("");

  lines.push("### 读取记录列表：GET " + collectionPath + "/records");
  lines.push(jsonExample({
    projectId: project.id,
    collectionName: "posts",
    items: [
      {
        id: "record-id",
        projectId: project.id,
        collectionId: "collection-id",
        data: { title: "示例标题", content: "示例内容" },
        status: "active",
        createdAt: "2026-01-01T00:00:00Z",
        updatedAt: "2026-01-01T00:00:00Z"
      }
    ]
  }));
  lines.push("");

  lines.push("### 创建记录：POST " + collectionPath + "/records");
  lines.push("### 读取单条记录：GET " + recordPath);
  lines.push("### 更新记录：PATCH " + recordPath);
  lines.push("以上三个接口都会返回单条记录，格式如下：");
  lines.push(jsonExample({
    id: "record-id",
    projectId: project.id,
    collectionId: "collection-id",
    data: { title: "示例标题", content: "示例内容" },
    status: "active",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z"
  }));
  lines.push("");

  lines.push("### 删除记录：DELETE " + recordPath);
  lines.push(jsonExample({ status: "deleted" }));
  lines.push("");

  lines.push("## 错误码和错误响应示例");
  lines.push("公开互动接口出错时通常返回 JSON，里面有 `error` 字段。请把 `error` 字段作为中文提示展示给用户，但业务判断要优先看 HTTP 状态码。");
  lines.push("");
  lines.push("### HTTP 400：请求内容或作品状态不正确");
  lines.push("常见于作品没有启用互动功能、JSON 格式不正确、数据表名为空、记录内容为空等情况。");
  lines.push(jsonExample({ error: "请求内容格式不正确" }));
  lines.push(jsonExample({ error: "这个作品没有启用互动功能" }));
  lines.push("");
  lines.push("### HTTP 401：作品数据密钥不正确");
  lines.push("常见于没有传 `X-Project-Key`，或传入的 key 不属于这个作品。");
  lines.push(jsonExample({ error: "作品数据密钥不正确" }));
  lines.push("");
  lines.push("### HTTP 403：当前公开权限不允许这个操作");
  lines.push("常见于读取 `publicRead: false` 的数据表、写入 `publicWrite: false` 的数据表，或尝试通过公开接口删除数据表。");
  lines.push(jsonExample({ error: "这个作品数据表不允许公开读取" }));
  lines.push(jsonExample({ error: "这个作品数据表不允许公开写入" }));
  lines.push(jsonExample({ error: "公开接口不允许删除作品数据表。" }));
  lines.push("");
  lines.push("### HTTP 404：作品、接口、数据表或记录不存在");
  lines.push("读取单个数据表时如果返回 `404`，可以认为这个数据表不存在，然后创建它。不要只判断英文 `not found`。");
  lines.push(jsonExample({ error: "找不到这个作品" }));
  lines.push(jsonExample({ error: "没有找到这个接口" }));
  lines.push(jsonExample({ error: "找不到这个作品数据表" }));
  lines.push(jsonExample({ error: "找不到这条记录" }));
  lines.push("");
  lines.push("### HTTP 429：套餐额度或用量限制");
  lines.push("常见于当前作品达到套餐允许的数据表、记录、请求次数或其他互动用量限制。遇到这种错误时，请用中文提示用户减少操作频率、减少数据量或升级套餐，不要循环重试。");
  lines.push(jsonExample({ error: "这个作品本月的互动查询次数已经用完了" }));
  lines.push(jsonExample({ error: "这个作品本月的互动写入次数已经用完了" }));
  lines.push("");
  lines.push("### HTTP 500：服务器读取或保存失败");
  lines.push("常见于平台临时故障。请提示用户稍后再试，不要在页面里展示英文技术细节。");
  lines.push(jsonExample({ error: "读取作品数据表失败" }));
  lines.push(jsonExample({ error: "读取记录失败" }));
  lines.push("");

  lines.push("## 推荐的前端请求封装");
  lines.push("请优先使用下面这种封装。它会保留 HTTP 状态码，方便正确判断 `404`、`401`、`403`、`429` 等错误。");
  lines.push(codeBlock("js", buildStandardFetchExample(apiBase, projectKey)));
  lines.push("");

  lines.push("## 推荐的数据表初始化方式");
  lines.push("初始化时先读取数据表；只有 HTTP `404` 才创建数据表；其他错误要提示用户或停止初始化。");
  lines.push(codeBlock("js", buildEnsureCollectionExample()));
  lines.push("");

  lines.push("## 创建数据表示例");
  lines.push("这个例子会创建 `posts` 数据表，用来保存标题和正文。字段名建议使用英文、拼音或数字，界面文案可以是中文。");
  lines.push(jsonExample({
    name: "posts",
    permissions: { publicRead: true, publicWrite: true },
    fields: [
      { name: "title", type: "string", required: true, isList: false },
      { name: "content", type: "text", required: true, isList: false }
    ]
  }));
  lines.push("");

  lines.push("## 创建记录示例");
  lines.push("创建记录时，具体内容放在 `data` 里面。`data` 里的字段要和数据表 `fields` 中定义的字段对应。");
  lines.push(jsonExample({ data: { title: "示例标题", content: "示例内容" } }));
  lines.push("");

  lines.push("## 现有数据表");
  if (collections.length === 0) {
    lines.push("暂时没有数据表。如果需要保存数据，请先调用创建数据表接口。");
  } else {
    for (const collection of collections) {
      lines.push(`### ${collection.name}`);
      lines.push(`- publicRead：${yesNo(collection.permissions.publicRead)}`);
      lines.push(`- publicWrite：${yesNo(collection.permissions.publicWrite)}`);
      if (collection.fields.length === 0) {
        lines.push("- 字段：暂无");
      } else {
        lines.push("- 字段：");
        for (const field of collection.fields) {
          const reference = field.reference ? `，关联数据表：${field.reference}` : "";
          lines.push(`  - ${field.name}：类型 ${field.type}，必填 ${yesNo(field.required)}，多值 ${yesNo(field.isList)}${reference}`);
        }
      }
      lines.push("");
    }
  }
  lines.push("");

  lines.push("## 前端 fetch 示例");
  lines.push("### 读取 posts 记录");
  lines.push(codeBlock("js", `apiFetch('/collections/posts/records')`));
  lines.push("");
  lines.push("### 新增 posts 记录");
  lines.push(codeBlock("js", `apiFetch('/collections/posts/records', {
  method: 'POST',
  body: JSON.stringify({
    data: {
      title: '示例标题',
      content: '示例内容'
    }
  })
})`));
  lines.push("");
  lines.push("### 创建 posts 数据表");
  lines.push(codeBlock("js", `apiFetch('/collections', {
  method: 'POST',
  body: JSON.stringify({
    name: 'posts',
    permissions: { publicRead: true, publicWrite: true },
    fields: [
      { name: 'title', type: 'string', required: true, isList: false }
    ]
  })
})`));
  lines.push("");

  lines.push("## 给 AI 的执行要求");
  lines.push("- 先根据用户想要的互动效果设计数据表，例如 `comments`、`messages`、`posts`、`profiles`、`saves`。");
  lines.push("- 如果现有数据表已经满足需求，直接复用，不要重复创建同名数据表。");
  lines.push("- 不要调用或编写删除数据表的逻辑。互动作品里的公开 key 只用于自动创建必要数据表和读写记录，不用于破坏性表管理。");
  lines.push("- 不要根据英文错误文本判断业务逻辑；请使用 `error.status` 或 `response.status`。");
  lines.push("- 写代码时要处理网络失败、接口返回错误和空数据状态，并用中文提示用户。");
  lines.push("- 生成页面时要保证按钮、输入框、文本框有清晰中文标签，键盘也能操作。");

  return lines.join("\n");
}
