package http

import (
	"fmt"
	"net/http"
	"strings"

	"ai-static-host/api/internal/domain"
)

const interactiveDocTemplate = "# {{PROJECT_NAME}} 互动 API 文档\n\n## 文档用途\n这份文档是给 AI 用的。请根据下面的作品信息、认证方式、接口、错误码和现有数据表，为这个作品编写互动功能。\n\n## 重要规则\n- 所有公开互动 API 请求都必须带上 `X-Project-Key` 请求头。\n- API 基础地址固定使用相对路径：`/api/v1/public/projects/{{PROJECT_ID}}`。不要使用 'api.wangru.net'，也不要自己猜测其他 API 域名。\n- 请使用 UTF-8，并在生成的 HTML 中加入 `<meta charset=\"utf-8\">`。\n- 判断接口错误时，请优先使用 HTTP 状态码，不要只判断英文错误文本。平台错误提示是中文，例如“找不到这个作品数据表”。\n- 如果读取单个数据表返回 HTTP `404`，表示这个数据表不存在，可以创建数据表；如果是其他状态码，不要当成数据表不存在。\n- 如果要写入留言、评论、论坛帖子、角色数据、存档等内容，先确认数据表是否存在；不存在时先创建数据表。\n- 创建数据表会受作品当前套餐和用量限制影响；如果接口返回 `429` 或额度相关错误，请用中文提示用户减少数据表、减少操作频率或升级套餐，不要绕过限制。\n- 创建数据表失败时不要反复重试，更不要删除、清空或重建已有数据表。\n- 公开接口不允许删除数据表，也不要生成删除数据表、清空数据表或重建数据表的代码。已有数据表满足需求时必须复用。\n- 权限只使用 `publicRead` 和 `publicWrite`，不要使用登录用户、创作者或所有者专用写入权限。\n- 如果这个数据表需要访客通过网页新增、编辑或删除记录，创建数据表时必须设置 `publicWrite: true`。如果设置为 `false`，公开网页后续将无法写入、修改或删除这个表的记录。\n- 只有纯展示、不需要访客提交或修改内容的数据表，才建议设置 `publicWrite: false`。\n- 新增或更新记录时，业务数据统一放进 `data` 对象里。\n- 不要把 `X-Project-Key` 保存到 `localStorage`。如果必须写进静态网页，只把它作为当前作品的公开互动 key 使用。\n- 用户点击按钮、提交表单等明确操作后才发起写入请求，不要因为页面加载、鼠标悬停或输入框聚焦就写入数据。\n\n## 作品信息\n- 作品 ID：`{{PROJECT_ID}}`\n- 作品名称：{{PROJECT_NAME}}\n- 作品地址：{{PUBLIC_URL}}\n- 互动功能：已开启\n\n## 认证方式\n每个公开互动 API 请求都要带上这个请求头：\n```http\nX-Project-Key: {{PROJECT_KEY}}\n```\n\n## API 基础地址\n在作品页面里调用接口时，请使用这个相对地址作为基础地址：\n```js\nconst API_BASE = '/api/v1/public/projects/{{PROJECT_ID}}';\n```\n不要写成 `https://api.wangru.net/...`。如果必须使用完整地址，请使用当前网站域名加上上面的相对路径。\n\n## 接口列表\n- 读取作品信息：`GET /api/v1/public/projects/{{PROJECT_ID}}`\n- 读取数据表列表：`GET /api/v1/public/projects/{{PROJECT_ID}}/collections`\n- 创建数据表：`POST /api/v1/public/projects/{{PROJECT_ID}}/collections`\n- 读取单个数据表：`GET /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}`\n- 读取记录列表：`GET /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records`\n  - 可选查询参数：`limit`、`offset`、`sort`、`where`、`filter`\n- 创建记录：`POST /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records`\n- 读取单条记录：`GET /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records/{recordId}`\n- 更新记录：`PATCH /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records/{recordId}`\n- 删除记录：`DELETE /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records/{recordId}`\n\n## 接口响应示例\n### 读取作品信息：GET /api/v1/public/projects/{{PROJECT_ID}}\n```json\n{\n  \"project\": {\n    \"id\": \"{{PROJECT_ID}}\",\n    \"name\": \"{{PROJECT_NAME}}\",\n    \"publicUrl\": \"{{PUBLIC_URL}}\",\n    \"interactive\": true\n  }\n}\n```\n\n### 读取数据表列表：GET /api/v1/public/projects/{{PROJECT_ID}}/collections\n```json\n{\n  \"projectId\": \"{{PROJECT_ID}}\",\n  \"items\": [\n    {\n      \"id\": \"collection-id\",\n      \"projectId\": \"{{PROJECT_ID}}\",\n      \"name\": \"posts\",\n      \"permissions\": {\n        \"publicRead\": true,\n        \"publicWrite\": true\n      },\n      \"fields\": [\n        {\n          \"name\": \"title\",\n          \"type\": \"string\",\n          \"required\": true,\n          \"isList\": false\n        },\n        {\n          \"name\": \"content\",\n          \"type\": \"text\",\n          \"required\": true,\n          \"isList\": false\n        }\n      ],\n      \"createdAt\": \"2026-01-01T00:00:00Z\"\n    }\n  ]\n}\n```\n\n### 创建数据表：POST /api/v1/public/projects/{{PROJECT_ID}}/collections\n```json\n{\n  \"id\": \"collection-id\",\n  \"projectId\": \"{{PROJECT_ID}}\",\n  \"name\": \"posts\",\n  \"permissions\": {\n    \"publicRead\": true,\n    \"publicWrite\": true\n  },\n  \"fields\": [\n    {\n      \"name\": \"title\",\n      \"type\": \"string\",\n      \"required\": true,\n      \"isList\": false\n    },\n    {\n      \"name\": \"content\",\n      \"type\": \"text\",\n      \"required\": true,\n      \"isList\": false\n    }\n  ],\n  \"createdAt\": \"2026-01-01T00:00:00Z\"\n}\n```\n\n### 读取单个数据表：GET /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}\n响应格式和创建数据表返回格式相同。如果数据表不存在，会返回 HTTP `404`。\n\n### 读取记录列表：GET /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records\n支持简单查询参数，适合登录、论坛、排行榜、云存档等常见场景：\n- `limit`：返回数量，1 到 200，默认最多 200。\n- `offset`：跳过数量，默认 0。\n- `sort`：排序字段，默认 `-createdAt`。例如 `sort=-createdAt`、`sort=score`、`sort=-score,createdAt`。业务字段可以直接写字段名，也可以写 `data.score`。\n- `where`：推荐的筛选格式，可重复传多个条件，多个条件之间是“并且”。格式为 `字段:操作:值`，例如 `where=username:eq:alice`、`where=score:gte:100`。\n- `filter`：兼容常见 AI 写法，只支持简单条件，例如 `filter=data.username==\"alice\"`。\n\n支持的筛选操作：\n- `eq`：等于。\n- `ne`：不等于。\n- `contains`：包含文本。\n- `gt` / `gte`：大于 / 大于等于。\n- `lt` / `lte`：小于 / 小于等于。\n\n常用例子：\n```js\n// 查找用户名为 alice 的用户，常用于登录/注册前检查\napiFetch('/collections/users/records?where=username:eq:alice&limit=1')\n\n// 读取某个论坛板块的最新帖子\napiFetch('/collections/posts/records?where=board:eq:feedback&sort=-createdAt&limit=20')\n\n// 读取排行榜前 10 名\napiFetch('/collections/scores/records?sort=-score&limit=10')\n```\n\n```json\n{\n  \"projectId\": \"{{PROJECT_ID}}\",\n  \"collectionName\": \"posts\",\n  \"items\": [\n    {\n      \"id\": \"record-id\",\n      \"projectId\": \"{{PROJECT_ID}}\",\n      \"collectionId\": \"collection-id\",\n      \"data\": {\n        \"title\": \"示例标题\",\n        \"content\": \"示例内容\"\n      },\n      \"status\": \"active\",\n      \"createdAt\": \"2026-01-01T00:00:00Z\",\n      \"updatedAt\": \"2026-01-01T00:00:00Z\"\n    }\n  ]\n}\n```\n\n### 创建记录：POST /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records\n### 读取单条记录：GET /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records/{recordId}\n### 更新记录：PATCH /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records/{recordId}\n以上三个接口都会返回单条记录，格式如下：\n```json\n{\n  \"id\": \"record-id\",\n  \"projectId\": \"{{PROJECT_ID}}\",\n  \"collectionId\": \"collection-id\",\n  \"data\": {\n    \"title\": \"示例标题\",\n    \"content\": \"示例内容\"\n  },\n  \"status\": \"active\",\n  \"createdAt\": \"2026-01-01T00:00:00Z\",\n  \"updatedAt\": \"2026-01-01T00:00:00Z\"\n}\n```\n\n### 删除记录：DELETE /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records/{recordId}\n```json\n{\n  \"status\": \"deleted\"\n}\n```\n\n## 错误码和错误响应示例\n公开互动接口出错时通常返回 JSON，里面有 `error` 字段。请把 `error` 字段作为中文提示展示给用户，但业务判断要优先看 HTTP 状态码。\n\n### HTTP 400：请求内容或作品状态不正确\n常见于作品没有启用互动功能、JSON 格式不正确、数据表名为空、记录内容为空等情况。\n```json\n{\n  \"error\": \"请求内容格式不正确\"\n}\n```\n```json\n{\n  \"error\": \"这个作品没有启用互动功能\"\n}\n```\n\n### HTTP 401：作品数据密钥不正确\n常见于没有传 `X-Project-Key`，或传入的 key 不属于这个作品。\n```json\n{\n  \"error\": \"作品数据密钥不正确\"\n}\n```\n\n### HTTP 403：当前公开权限不允许这个操作\n常见于读取 `publicRead: false` 的数据表、写入 `publicWrite: false` 的数据表，或尝试通过公开接口删除数据表。\n```json\n{\n  \"error\": \"这个作品数据表不允许公开读取\"\n}\n```\n```json\n{\n  \"error\": \"这个作品数据表不允许公开写入\"\n}\n```\n```json\n{\n  \"error\": \"公开接口不允许删除作品数据表。\"\n}\n```\n\n### HTTP 404：作品、接口、数据表或记录不存在\n读取单个数据表时如果返回 `404`，可以认为这个数据表不存在，然后创建它。不要只判断英文 `not found`。\n```json\n{\n  \"error\": \"找不到这个作品\"\n}\n```\n```json\n{\n  \"error\": \"没有找到这个接口\"\n}\n```\n```json\n{\n  \"error\": \"找不到这个作品数据表\"\n}\n```\n```json\n{\n  \"error\": \"找不到这条记录\"\n}\n```\n\n### HTTP 429：套餐额度或用量限制\n常见于当前作品达到套餐允许的数据表、记录、请求次数或其他互动用量限制。遇到这种错误时，请用中文提示用户减少操作频率、减少数据量或升级套餐，不要循环重试。\n```json\n{\n  \"error\": \"这个作品本月的互动查询次数已经用完了\"\n}\n```\n```json\n{\n  \"error\": \"这个作品本月的互动写入次数已经用完了\"\n}\n```\n\n### HTTP 500：服务器读取或保存失败\n常见于平台临时故障。请提示用户稍后再试，不要在页面里展示英文技术细节。\n```json\n{\n  \"error\": \"读取作品数据表失败\"\n}\n```\n```json\n{\n  \"error\": \"读取记录失败\"\n}\n```\n\n## 推荐的前端请求封装\n请优先使用下面这种封装。它会保留 HTTP 状态码，方便正确判断 `404`、`401`、`403`、`429` 等错误。\n```js\nconst API_BASE = '/api/v1/public/projects/{{PROJECT_ID}}';\nconst PROJECT_KEY = '{{PROJECT_KEY}}';\n\nasync function apiFetch(path, options = {}) {\n  const headers = {\n    'X-Project-Key': PROJECT_KEY,\n    ...(options.headers || {})\n  };\n\n  if (options.body && !headers['Content-Type']) {\n    headers['Content-Type'] = 'application/json';\n  }\n\n  const response = await fetch(`${API_BASE}${path}`, {\n    ...options,\n    headers\n  });\n\n  let data = null;\n  try {\n    data = await response.json();\n  } catch {\n    data = null;\n  }\n\n  if (!response.ok) {\n    const error = new Error(data?.error || `请求失败，状态码：${response.status}`);\n    error.status = response.status;\n    error.data = data;\n    throw error;\n  }\n\n  return data;\n}\n```\n\n## 推荐的数据表初始化方式\n初始化时先读取数据表；只有 HTTP `404` 才创建数据表；其他错误要提示用户或停止初始化。\n```js\nasync function ensureCollection(name, fields) {\n  try {\n    return await apiFetch(`/collections/${encodeURIComponent(name)}`);\n  } catch (error) {\n    // 数据表不存在时，平台会返回 HTTP 404。请判断 status，不要只判断英文错误文本。\n    if (error.status !== 404) {\n      throw error;\n    }\n  }\n\n  return apiFetch('/collections', {\n    method: 'POST',\n    body: JSON.stringify({\n      name,\n      permissions: { publicRead: true, publicWrite: true },\n      fields\n    })\n  });\n}\n\n// 示例：初始化帖子表。\nawait ensureCollection('posts', [\n  { name: 'title', type: 'string', required: true, isList: false },\n  { name: 'content', type: 'text', required: true, isList: false }\n]);\n```\n\n## 创建数据表示例\n这个例子会创建 `posts` 数据表，用来保存标题和正文。字段名建议使用英文、拼音或数字，界面文案可以是中文。\n```json\n{\n  \"name\": \"posts\",\n  \"permissions\": {\n    \"publicRead\": true,\n    \"publicWrite\": true\n  },\n  \"fields\": [\n    {\n      \"name\": \"title\",\n      \"type\": \"string\",\n      \"required\": true,\n      \"isList\": false\n    },\n    {\n      \"name\": \"content\",\n      \"type\": \"text\",\n      \"required\": true,\n      \"isList\": false\n    }\n  ]\n}\n```\n\n## 创建记录示例\n创建记录时，具体内容放在 `data` 里面。`data` 里的字段要和数据表 `fields` 中定义的字段对应。\n```json\n{\n  \"data\": {\n    \"title\": \"示例标题\",\n    \"content\": \"示例内容\"\n  }\n}\n```\n\n{{COLLECTIONS_SECTION}}## 前端 fetch 示例\n### 读取 posts 记录\n```js\napiFetch('/collections/posts/records')\n```\n\n### 新增 posts 记录\n```js\napiFetch('/collections/posts/records', {\n  method: 'POST',\n  body: JSON.stringify({\n    data: {\n      title: '示例标题',\n      content: '示例内容'\n    }\n  })\n})\n```\n\n### 创建 posts 数据表\n```js\napiFetch('/collections', {\n  method: 'POST',\n  body: JSON.stringify({\n    name: 'posts',\n    permissions: { publicRead: true, publicWrite: true },\n    fields: [\n      { name: 'title', type: 'string', required: true, isList: false }\n    ]\n  })\n})\n```\n\n## 给 AI 的执行要求\n- 先根据用户想要的互动效果设计数据表，例如 `comments`、`messages`、`posts`、`profiles`、`saves`。\n- 如果现有数据表已经满足需求，直接复用，不要重复创建同名数据表。\n- 不要调用或编写删除数据表的逻辑。互动作品里的公开 key 只用于自动创建必要数据表和读写记录，不用于破坏性表管理。\n- 不要根据英文错误文本判断业务逻辑；请使用 `error.status` 或 `response.status`。\n- 写代码时要处理网络失败、接口返回错误和空数据状态，并用中文提示用户。\n- 生成页面时要保证按钮、输入框、文本框有清晰中文标签，键盘也能操作。"

const interactiveDocUpdateSemanticsSection = `## 更新记录的重要规则
` + "公开接口的 `PATCH` 更新记录会替换整份 `data` 对象，不会自动合并字段。只提交 `{ data: { likes: 1 } }` 会让原来的 `title`、`content`、`board` 等字段消失。论坛点赞、编辑资料、更新存档、修改回复等局部更新都必须先读取原记录，再合并字段，最后提交完整 `data`。\n\n" + `推荐写法：
` + "```js\nasync function patchRecordData(collectionName, recordId, updates) {\n  const record = await apiFetch(`/collections/${encodeURIComponent(collectionName)}/records/${encodeURIComponent(recordId)}`);\n  const merged = { ...(record.data || {}), ...updates };\n\n  return apiFetch(`/collections/${encodeURIComponent(collectionName)}/records/${encodeURIComponent(recordId)}`, {\n    method: 'PATCH',\n    body: JSON.stringify({ data: merged })\n  });\n}\n\n// 示例：论坛点赞时只改 likes 和 likedBy，但提交前必须保留原帖子的 title/content/board 等字段。\nawait patchRecordData('posts', postId, { likes: nextLikes, likedBy: nextLikedBy });\n```\n\n" + `禁止写法：
` + "```js\n// 错误：这会把这条帖子 data 替换成只有 likes 字段，帖子会像被删除一样从板块列表消失。\napiFetch(`/collections/posts/records/${postId}`, {\n  method: 'PATCH',\n  body: JSON.stringify({ data: { likes: nextLikes } })\n});\n```\n\n"

const interactiveDocEmailCodeSection = "## 作品邮箱验证码接口\n" +
	"如果作品需要登录、注册、重置密码、绑定邮箱等邮箱验证码能力，请使用 PlayPage 官方验证码接口。平台会发送固定格式的验证码邮件，作品不能自定义邮件正文，避免垃圾邮件和钓鱼风险。\n\n" +
	"账号级每日额度：免费版 20 封/天，轻享版 50 封/天，支持版 100 封/天。这里的额度按作品所有者账号累计，不是按单个作品累计。同一作品、同一邮箱、同一用途 60 秒内只能发送一次。\n\n" +
	"### 发送验证码\n" +
	"```js\nawait apiFetch('/auth/email-code/send', {\n  method: 'POST',\n  body: JSON.stringify({\n    email: 'user@example.com',\n    purpose: 'login'\n  })\n});\n```\n\n" +
	"`purpose` 只能是：`login`、`register`、`reset`、`bind`、`custom`。不传时默认为 `login`。\n\n" +
	"成功响应通常是 HTTP `202`：\n" +
	"```json\n{\n  \"ok\": true,\n  \"status\": \"code-sent\",\n  \"message\": \"验证码已发送，请查看邮箱。\",\n  \"email\": \"user@example.com\",\n  \"purpose\": \"login\",\n  \"expiresIn\": 600,\n  \"quota\": {\n    \"used\": 1,\n    \"limit\": 20\n  }\n}\n```\n\n" +
	"### 验证验证码\n" +
	"```js\nawait apiFetch('/auth/email-code/verify', {\n  method: 'POST',\n  body: JSON.stringify({\n    email: 'user@example.com',\n    purpose: 'login',\n    code: '123456'\n  })\n});\n```\n\n" +
	"成功响应：\n" +
	"```json\n{\n  \"ok\": true,\n  \"verified\": true,\n  \"email\": \"user@example.com\",\n  \"purpose\": \"login\"\n}\n```\n\n" +
	"兼容路径：`POST /auth/send-code` 和 `POST /auth/verify-code` 也可用，但新代码优先使用 `/auth/email-code/send` 和 `/auth/email-code/verify`。\n\n" +
	"验证码验证成功后会立即失效。验证失败会返回 HTTP `401`。额度用完或发送过快会返回 HTTP `429`。请把返回的中文 `error` 展示给用户，不要循环重试。\n\n"

func (rt *Router) handleProjectInteractiveDoc(w http.ResponseWriter, r *http.Request, projectID string) {
	_, project, ok := rt.requireOwnedProject(w, r, projectID)
	if !ok {
		return
	}
	if !project.Interactive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "这个作品还没有启用互动功能"})
		return
	}

	_ = rt.ensureProjectLiveLink(r.Context(), project)

	projectKey := rt.lookupProjectPublicKey(r.Context(), projectID)
	if strings.TrimSpace(projectKey) == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "生成互动 API 文档失败"})
		return
	}

	collections, err := rt.store.ListCollections(r.Context(), project.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取作品数据表失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"doc": buildInteractiveAPIDoc(project, projectKey, collections),
	})
}

func buildInteractiveAPIDoc(project domain.Project, projectKey string, collections []domain.Collection) string {
	doc := interactiveDocTemplate
	doc = strings.ReplaceAll(doc, "{{PROJECT_ID}}", project.ID)
	doc = strings.ReplaceAll(doc, "{{PROJECT_NAME}}", project.Name)
	doc = strings.ReplaceAll(doc, "{{PUBLIC_URL}}", project.PublicURL)
	doc = strings.ReplaceAll(doc, "{{PROJECT_KEY}}", projectKey)
	doc = strings.ReplaceAll(doc, "{{COLLECTIONS_SECTION}}", buildInteractiveCollectionsSection(collections))
	doc = addInteractiveDocUpdateSemantics(doc)
	return doc
}

func addInteractiveDocUpdateSemantics(doc string) string {
	doc = strings.ReplaceAll(doc,
		"- 新增或更新记录时，业务数据统一放进 `data` 对象里。",
		"- 新增或更新记录时，业务数据统一放进 `data` 对象里。\n- 重要：`PATCH` 更新记录会替换整份 `data` 对象，不会自动合并字段；局部更新必须先读取原记录、合并字段，再提交完整 `data`。\n- 如果作品需要邮箱验证码，请使用“作品邮箱验证码接口”；不要自己假设其它邮件接口，也不要在前端伪造验证码。",
	)
	doc = strings.ReplaceAll(doc,
		"- 更新记录：`PATCH /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records/{recordId}`",
		"- 更新记录：`PATCH /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records/{recordId}`（替换整份 `data`，不是字段合并）",
	)
	doc = strings.ReplaceAll(doc,
		"- 删除记录：`DELETE /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records/{recordId}`",
		"- 删除记录：`DELETE /api/v1/public/projects/{{PROJECT_ID}}/collections/{collectionName}/records/{recordId}`\n- 发送作品邮箱验证码：`POST /api/v1/public/projects/{{PROJECT_ID}}/auth/email-code/send`\n- 验证作品邮箱验证码：`POST /api/v1/public/projects/{{PROJECT_ID}}/auth/email-code/verify`\n  - 兼容路径：`POST /auth/send-code` 和 `POST /auth/verify-code` 也可用，但新代码优先使用 `/auth/email-code/send` 和 `/auth/email-code/verify`。",
	)
	doc = strings.Replace(doc, "## 错误码和错误响应示例", interactiveDocEmailCodeSection+"## 错误码和错误响应示例", 1)
	return strings.Replace(doc, "## 创建数据表示例", interactiveDocUpdateSemanticsSection+"## 创建数据表示例", 1)
}

func buildInteractiveCollectionsSection(collections []domain.Collection) string {
	var lines []string
	lines = append(lines, "## 现有数据表")
	if len(collections) == 0 {
		lines = append(lines, "暂时没有数据表。如果需要保存数据，请先调用创建数据表接口。", "")
		return strings.Join(lines, "\n") + "\n"
	}

	for _, collection := range collections {
		lines = append(lines, "### "+collection.Name)
		lines = append(lines, fmt.Sprintf("- publicRead：%s", interactiveDocYesNo(collection.Permissions.PublicRead)))
		lines = append(lines, fmt.Sprintf("- publicWrite：%s", interactiveDocYesNo(collection.Permissions.PublicWrite)))
		if len(collection.Fields) == 0 {
			lines = append(lines, "- 字段：暂无")
		} else {
			lines = append(lines, "- 字段：")
			for _, field := range collection.Fields {
				reference := ""
				if field.Reference != "" {
					reference = "，关联数据表：" + field.Reference
				}
				lines = append(lines, fmt.Sprintf("  - %s：类型 %s，必填 %s，多值 %s%s", field.Name, field.Type, interactiveDocYesNo(field.Required), interactiveDocYesNo(field.IsList), reference))
			}
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n") + "\n"
}

func interactiveDocYesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}
