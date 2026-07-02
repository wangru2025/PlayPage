package http

import "strings"

func renderLinkDirectoryTemplate(values map[string]string) string {
	var cards []string
	for i := 1; i <= 3; i++ {
		title := values["link"+string(rune('0'+i))+"Title"]
		url := values["link"+string(rune('0'+i))+"Url"]
		if strings.TrimSpace(title) != "" && strings.TrimSpace(url) != "" {
			cards = append(cards, `<a class="card" href="`+url+`" target="_blank" rel="noreferrer"><strong>`+title+`</strong><span>`+url+`</span></a>`)
		}
	}
	if len(cards) == 0 {
		cards = append(cards, `<div class="card"><strong>还没有链接</strong><span>请编辑 HTML 添加更多链接。</span></div>`)
	}
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + values["siteTitle"] + `</title><style>
:root{--theme:` + values["themeColor"] + `}body{margin:0;font-family:system-ui,"Microsoft YaHei",sans-serif;background:#f8fafc;color:#0f172a}.wrap{width:min(980px,calc(100vw - 32px));margin:0 auto;padding:56px 0}header{margin-bottom:28px}h1{font-size:clamp(2rem,7vw,4rem);margin:0;color:var(--theme)}p{color:#475569;font-size:1.1rem}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:16px}.card{display:grid;gap:8px;padding:22px;border:1px solid #e2e8f0;border-radius:22px;background:#fff;text-decoration:none;color:inherit;box-shadow:0 14px 40px rgba(15,23,42,.08)}.card strong{font-size:1.25rem}.card span{color:#64748b;overflow-wrap:anywhere}</style></head><body><main class="wrap"><header><h1>` + values["siteTitle"] + `</h1><p>` + values["subtitle"] + `</p></header><section class="grid">` + strings.Join(cards, "") + `</section></main></body></html>`
}
