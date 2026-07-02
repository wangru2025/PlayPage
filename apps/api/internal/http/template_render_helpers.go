package http

import "strings"

func addTemplatePreviewBanner(body string) string {
	banner := `<div style="position:fixed;left:16px;right:16px;bottom:16px;z-index:99999;padding:10px 14px;border:1px solid rgba(15,23,42,.18);border-radius:14px;background:rgba(255,255,255,.94);box-shadow:0 12px 34px rgba(15,23,42,.16);font:14px/1.5 system-ui,'Microsoft YaHei',sans-serif;color:#0f172a">当前为 PlayPage 模板预览。互动数据仅用于演示，不会保存。</div>`
	if strings.Contains(body, "<body>") {
		return strings.Replace(body, "<body>", "<body>"+banner, 1)
	}
	return strings.Replace(body, "<body", "<body", 1) + banner
}

func renderBlankTemplate(values map[string]string) string {
	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>` + values["siteTitle"] + `</title>
  <style>
    :root { color-scheme: light; --theme: ` + values["themeColor"] + `; }
    body { margin: 0; min-height: 100vh; display: grid; place-items: center; font-family: system-ui, "Microsoft YaHei", sans-serif; background: #fff8ef; color: #241b15; }
    main { width: min(760px, calc(100vw - 32px)); padding: 42px; border: 1px solid #ead8c2; border-radius: 28px; background: #fffdf8; box-shadow: 0 18px 50px rgba(120, 72, 24, .12); }
    h1 { margin: 0 0 12px; color: var(--theme); font-size: clamp(2rem, 7vw, 4rem); }
    p { margin: 0; font-size: 1.12rem; line-height: 1.8; }
  </style>
</head>
<body>
  <main>
    <h1>` + values["siteTitle"] + `</h1>
    <p>` + values["subtitle"] + `</p>
  </main>
</body>
</html>`
}
