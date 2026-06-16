## Web Deployment Notes

`Next.js` 使用 `output: "standalone"` 时，运行入口是 `.next/standalone/server.js`，但浏览器端脚本和样式仍然需要 `.next/static`。

部署时至少要保证下面两处同时存在：

- `/opt/ai-static-host/web/.next/static`
- `/opt/ai-static-host/web/.next/standalone/.next/static`

如果只复制了 `standalone`，公网访问时 `/_next/static/...` 会返回 `404`，页面就会在浏览器里报 `Application error: a client-side exception has occurred`。

推荐直接使用：

```bash
bash /opt/ai-static-host/src/deploy/deploy-web.sh
```
