# 客户端 API 文档

本文档面向 Android、Windows 和其他非浏览器客户端。Web 端继续使用 Cookie + CSRF；客户端推荐使用 Bearer Token。

## 基础地址

```text
https://web.wangru.net
```

所有接口路径都以 `/api/` 开头。请求和响应都使用 UTF-8。

## 通用错误响应

接口出错时通常返回 JSON，里面有 `error` 字段。客户端可以直接把它展示给用户。

```json
{
  "error": "中文错误提示"
}
```

## 认证方式

### Web 浏览器

- 登录成功后，服务端写入 HttpOnly Cookie。
- 状态变更请求需要 CSRF 令牌。
- 现有 Web 用户不需要重新登录。

### Android / Windows 客户端

- 登录成功后，从响应里读取 `accessToken`。
- 后续请求添加：

```http
Authorization: Bearer accessToken
```

- 使用 Bearer Token 时不需要 Cookie，也不需要 CSRF 令牌。

## 登录接口

### 1. 请求邮箱验证码

```http
POST /api/v1/auth/request-code
```

请求体：

```json
{
  "email": "user@example.com",
  "username": "wangwang"
}
```

成功响应：HTTP `202`

```json
{
  "status": "code-sent",
  "email": "user@example.com",
  "expiresIn": 600
}
```

### 2. 校验验证码并登录

```http
POST /api/v1/auth/verify-code
```

请求体：

```json
{
  "email": "user@example.com",
  "code": "123456"
}
```

成功响应：HTTP `200`

```json
{
  "status": "signed-in",
  "user": {
    "id": "user-id",
    "username": "wangwang",
    "email": "user@example.com",
    "status": "active",
    "role": "user",
    "planCode": "free",
    "createdAt": "2026-01-01T00:00:00Z"
  },
  "csrfToken": "给 Web 浏览器使用的 CSRF 令牌",
  "accessToken": "给客户端使用的 Bearer Token",
  "tokenType": "Bearer",
  "expiresAt": "2026-02-01T00:00:00Z",
  "expiresInSeconds": 2592000
}
```

客户端请保存 `accessToken`；`csrfToken` 是给 Web Cookie 会话用的，客户端可以忽略。

### 3. 获取当前登录用户

```http
GET /api/v1/me
Authorization: Bearer accessToken
```

### 4. 退出登录

```http
POST /api/v1/auth/logout
Authorization: Bearer accessToken
```

成功响应：

```json
{
  "status": "signed-out"
}
```

## 请求示例

```js
const API_BASE = 'https://web.wangru.net';
let accessToken = '登录后得到的 accessToken';

async function apiFetch(path, options = {}) {
  const headers = { ...(options.headers || {}), Authorization: `Bearer ${accessToken}` };
  if (options.body && !headers['Content-Type']) headers['Content-Type'] = 'application/json';
  const response = await fetch(`${API_BASE}${path}`, { ...options, headers });
  const data = await response.json().catch(() => null);
  if (!response.ok) throw new Error(data?.error || `请求失败，状态码：${response.status}`);
  return data;
}
```

## 已登录后常用接口

```http
GET /api/v1/me
POST /api/v1/me/profile
GET /api/v1/projects
POST /api/v1/projects
GET /api/v1/projects/{projectId}
DELETE /api/v1/projects/{projectId}
GET /api/v1/projects/{projectId}/releases
POST /api/v1/projects/{projectId}/releases?mode=zip
POST /api/v1/projects/{projectId}/releases?mode=html
POST /api/v1/projects/{projectId}/releases?mode=text
GET /api/v1/projects/{projectId}/source
POST /api/v1/projects/{projectId}/visibility
POST /api/v1/projects/{projectId}/path
GET /api/v1/projects/{projectId}/domains
POST /api/v1/projects/{projectId}/domains
GET /api/v1/projects/{projectId}/collections
POST /api/v1/projects/{projectId}/collections
GET /api/v1/projects/{projectId}/collections/{collectionName}
PATCH /api/v1/projects/{projectId}/collections/{collectionName}
DELETE /api/v1/projects/{projectId}/collections/{collectionName}
GET /api/v1/projects/{projectId}/collections/{collectionName}/records
POST /api/v1/projects/{projectId}/collections/{collectionName}/records
GET /api/v1/projects/{projectId}/interactive-doc
```

公开互动 API 不使用 Bearer Token，而是使用作品自己的 `X-Project-Key`。请在作品互动功能页复制后端生成的互动 API 文档。

## 错误处理建议

- HTTP `400`：请求内容格式不正确，或必填字段为空。
- HTTP `401`：未登录、token 不存在、token 过期或 token 错误，请用户重新登录。
- HTTP `403`：没有权限，或 Web 端 CSRF 校验失败。客户端如果使用 Bearer Token，正常不需要处理 CSRF。
- HTTP `404`：资源不存在。
- HTTP `429`：请求过于频繁或超过套餐额度。
- HTTP `500`：服务器临时错误，请稍后重试。
