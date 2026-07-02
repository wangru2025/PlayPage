package http

import "strings"

func renderPersonalHomeTemplate(values map[string]string) string {
	links := []string{}
	for _, key := range []string{"link1", "link2", "link3"} {
		if strings.TrimSpace(values[key]) != "" {
			links = append(links, `<a href="`+values[key]+`" target="_blank" rel="noreferrer">`+values[key]+`</a>`)
		}
	}
	if len(links) == 0 {
		links = append(links, `<span>还没有填写链接</span>`)
	}
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + values["siteTitle"] + `</title><style>
:root{--theme:` + values["themeColor"] + `}*{box-sizing:border-box}body{margin:0;min-height:100vh;font-family:system-ui,"Microsoft YaHei",sans-serif;background:linear-gradient(135deg,#fff7ed,#eef2ff);color:#1f2937}.wrap{width:min(920px,calc(100vw - 32px));margin:0 auto;padding:72px 0}.card{background:rgba(255,255,255,.88);border:1px solid rgba(0,0,0,.08);border-radius:32px;padding:40px;box-shadow:0 24px 70px rgba(31,41,55,.12)}.avatar{width:96px;height:96px;border-radius:28px;background:var(--theme);color:#fff;display:grid;place-items:center;font-size:3rem;font-weight:900}h1{font-size:clamp(2rem,7vw,4rem);margin:22px 0 8px;color:var(--theme)}p{font-size:1.12rem;line-height:1.8}.links{display:grid;gap:12px;margin-top:24px}.links a,.links span{padding:14px 16px;border:1px solid #e5e7eb;border-radius:16px;background:#fff;color:#111827;text-decoration:none;overflow-wrap:anywhere}</style></head><body><main class="wrap"><section class="card"><div class="avatar" aria-hidden="true">` + firstRune(values["nickname"]) + `</div><h1>` + values["nickname"] + `</h1><p>` + values["bio"] + `</p><div class="links">` + strings.Join(links, "") + `</div></section></main></body></html>`
}
