package http

import (
	"encoding/json"
	"html"
	"strings"
)

type recordListField struct {
	Name      string `json:"name"`
	Label     string `json:"label"`
	Required  bool   `json:"required"`
	Multiline bool   `json:"multiline"`
}

type recordListTemplateOptions struct {
	Collection     string
	FormTitle      string
	ListTitle      string
	SubmitText     string
	EmptyText      string
	SortNumber     string
	Fields         []recordListField
	HiddenDefaults map[string]string
	Preview        []map[string]string
}

func renderRecordListTemplate(values map[string]string, projectID, publicKey string, preview bool, opts recordListTemplateOptions) string {
	cfg := templateRuntimeConfig(projectID, publicKey, preview)
	fieldsJSON := mustTemplateJSON(opts.Fields)
	previewJSON := mustTemplateJSON(opts.Preview)
	hiddenJSON := mustTemplateJSON(opts.HiddenDefaults)
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + values["siteTitle"] + `</title><style>
:root{--theme:` + values["themeColor"] + `}*{box-sizing:border-box}body{margin:0;font-family:system-ui,"Microsoft YaHei",sans-serif;background:linear-gradient(135deg,#f8fafc,#fff7ed);color:#0f172a}.wrap{width:min(980px,calc(100vw - 32px));margin:0 auto;padding:44px 0}header{margin-bottom:18px}h1{margin:0;color:var(--theme);font-size:clamp(2rem,7vw,3.6rem)}p{line-height:1.75}.muted{color:#64748b}.panel,.item{background:rgba(255,255,255,.94);border:1px solid #e2e8f0;border-radius:22px;padding:20px;margin-top:16px;box-shadow:0 12px 34px rgba(15,23,42,.08)}label{display:grid;gap:6px;margin:0 0 12px;font-weight:700}input,textarea,button{font:inherit}input,textarea{width:100%;padding:12px;border:1px solid #cbd5e1;border-radius:14px;background:#fff}button{border:0;border-radius:999px;padding:11px 18px;background:var(--theme);color:#fff;font-weight:800}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:14px}.field{margin:8px 0}.field strong{display:block;color:#334155}.field span,.field a{overflow-wrap:anywhere}.status{margin-top:10px;color:#475569}.item h3{margin:.2rem 0 .5rem;color:var(--theme)}.meta{color:#64748b;font-size:.92rem}</style></head><body><main class="wrap"><header><h1>` + values["siteTitle"] + `</h1><p class="muted">` + values["subtitle"] + `</p></header><section class="panel"><h2>` + html.EscapeString(opts.FormTitle) + `</h2><form id="form"></form><div id="status" class="status" role="status" aria-live="polite"></div></section><section class="panel"><h2>` + html.EscapeString(opts.ListTitle) + `</h2><div id="list">正在读取……</div></section></main><script>
const CFG=` + cfg + `,COLLECTION=` + mustTemplateJSON(opts.Collection) + `,FIELDS=` + fieldsJSON + `,PREVIEW_DATA=` + previewJSON + `,HIDDEN_DEFAULTS=` + hiddenJSON + `,SORT_NUMBER=` + mustTemplateJSON(opts.SortNumber) + `,EMPTY_TEXT=` + mustTemplateJSON(opts.EmptyText) + `,SUBMIT_TEXT=` + mustTemplateJSON(opts.SubmitText) + `;
const api=p=>CFG.apiBase+p;const headers={'Content-Type':'application/json','X-Project-Key':CFG.publicKey};const previewRecords=PREVIEW_DATA.map((data,i)=>({id:'preview-'+(i+1),data,createdAt:new Date(Date.now()-(i+1)*3600000).toISOString()}));
function esc(s){return String(s||'').replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]))}
function fieldControl(f){const id='f_'+f.name;const required=f.required?' required':'';const max=f.multiline?' maxlength="2000"':' maxlength="200"';const input=f.multiline?'<textarea id="'+id+'" rows="4"'+required+max+'></textarea>':'<input id="'+id+'"'+required+max+'>';return '<label for="'+id+'">'+esc(f.label)+input+'</label>'}
form.innerHTML=FIELDS.map(fieldControl).join('')+'<button type="submit">'+esc(SUBMIT_TEXT)+'</button>';
async function request(path,opt={}){if(CFG.preview)return previewRequest(path,opt);const r=await fetch(api(path),{...opt,headers:{...headers,...opt.headers}});const j=await r.json().catch(()=>({}));if(!r.ok)throw new Error(j.error||'请求失败');return j}
async function previewRequest(path,opt={}){await new Promise(r=>setTimeout(r,120));const method=opt.method||'GET';const target='/collections/'+COLLECTION+'/records';if(path===target&&method==='GET')return{items:previewRecords};if(path===target&&method==='POST'){const body=JSON.parse(opt.body||'{}');const item={id:'preview-'+(previewRecords.length+1),data:body.data||{},createdAt:new Date().toISOString()};previewRecords.push(item);return item}throw new Error('预览模式不支持这个操作')}
function valueHTML(name,value){value=String(value||'');if(/url|link|地址/i.test(name)&&/^https?:\/\//.test(value)){return '<a href="'+esc(value)+'" target="_blank" rel="noreferrer">'+esc(value)+'</a>'}return '<span>'+esc(value)+'</span>'}
function renderItem(x,index){const title=x.data.title||x.data.player||x.data.nickname||x.data.name||('第 '+(index+1)+' 条');const body=FIELDS.map(f=>'<div class="field"><strong>'+esc(f.label)+'</strong>'+valueHTML(f.name,x.data[f.name])+'</div>').join('');const hidden=Object.keys(x.data||{}).filter(k=>!FIELDS.some(f=>f.name===k)&&x.data[k]).map(k=>'<div class="field"><strong>'+esc(k)+'</strong>'+valueHTML(k,x.data[k])+'</div>').join('');return '<article class="item"><h3>'+esc(title)+'</h3>'+body+hidden+'<div class="meta">'+new Date(x.createdAt).toLocaleString()+'</div></article>'}
async function load(){try{const data=await request('/collections/'+COLLECTION+'/records');let items=(data.items||[]).slice();if(SORT_NUMBER){items.sort((a,b)=>(Number(b.data[SORT_NUMBER])||0)-(Number(a.data[SORT_NUMBER])||0))}else{items.reverse()}list.innerHTML=items.length?items.map(renderItem).join(''):esc(EMPTY_TEXT)}catch(e){list.textContent=e.message}}
form.addEventListener('submit',async e=>{e.preventDefault();status.textContent='正在提交……';const data={...HIDDEN_DEFAULTS};for(const f of FIELDS){const el=document.getElementById('f_'+f.name);data[f.name]=(el&&el.value||'').trim();if(f.required&&!data[f.name]){status.textContent='请填写：'+f.label;return}}try{await request('/collections/'+COLLECTION+'/records',{method:'POST',body:JSON.stringify({data})});for(const f of FIELDS){const el=document.getElementById('f_'+f.name);if(el)el.value=''}status.textContent=CFG.preview?'预览数据已临时显示，刷新后会恢复默认演示数据。':'已提交。';await load()}catch(err){status.textContent=err.message}});
load();</script></body></html>`
}

func mustTemplateJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(raw)
}

func templateRuntimeConfig(projectID, publicKey string, preview bool) string {
	payload := map[string]any{
		"apiBase":   "/api/v1/public/projects/" + projectID,
		"publicKey": publicKey,
		"preview":   preview,
	}
	raw, _ := json.Marshal(payload)
	return string(raw)
}

func firstRune(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "P"
	}
	for _, r := range value {
		return html.EscapeString(string(r))
	}
	return "P"
}
