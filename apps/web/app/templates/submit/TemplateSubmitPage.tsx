"use client";

import { useMemo, useState } from "react";
import { postForm } from "@/lib/api";
import type { TemplateSubmission } from "../templateTypes";

type ParamDraft = {
  name: string;
  label: string;
  type: "string" | "text" | "color" | "select";
  required: boolean;
  defaultValue: string;
  placeholder: string;
  help: string;
  options: string;
};

const minimalTemplateHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{siteTitle}}</title>
  <style>
    :root { --theme: {{themeColor}}; }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      font-family: system-ui, "Microsoft YaHei", sans-serif;
      background: #f8fafc;
      color: #0f172a;
    }
    main {
      width: min(760px, calc(100vw - 32px));
      padding: 40px;
      border-radius: 28px;
      background: white;
      box-shadow: 0 24px 70px rgba(15, 23, 42, .12);
    }
    h1 { color: var(--theme); font-size: clamp(2rem, 7vw, 4rem); margin: 0 0 12px; }
    p { line-height: 1.8; }
  </style>
</head>
<body>
  <main>
    <h1>{{siteTitle}}</h1>
    <p>{{subtitle}}</p>
  </main>
</body>
</html>`;

const interactiveTemplateHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{siteTitle}}</title>
  <style>
    :root { --theme: {{themeColor}}; }
    body { margin: 0; font-family: system-ui, "Microsoft YaHei", sans-serif; background: #fff7ed; color: #211812; }
    main { width: min(860px, calc(100vw - 32px)); margin: 0 auto; padding: 48px 0; }
    h1 { color: var(--theme); }
    .panel { background: #fff; border: 1px solid #ead7c4; border-radius: 24px; padding: 22px; margin-top: 18px; }
    label { display: grid; gap: 6px; margin-bottom: 12px; }
    input, textarea, button { font: inherit; }
    input, textarea { width: 100%; padding: 12px; border: 1px solid #decbb8; border-radius: 14px; box-sizing: border-box; }
    button { border: 0; border-radius: 999px; padding: 12px 18px; background: var(--theme); color: #fff; font-weight: 700; }
    .msg { border-top: 1px solid #f1e1cf; padding: 14px 0; }
    .meta { color: #7c6f64; font-size: .92rem; }
  </style>
</head>
<body>
  <main>
    <h1>{{siteTitle}}</h1>
    <p>{{subtitle}}</p>
    <section class="panel">
      <form id="form">
        <label>昵称<input id="nickname" required maxlength="40"></label>
        <label>留言<textarea id="content" required rows="4" maxlength="1000"></textarea></label>
        <button>发送留言</button>
      </form>
      <div id="status" role="status" aria-live="polite"></div>
    </section>
    <section class="panel">
      <h2>留言</h2>
      <div id="list">正在读取留言……</div>
    </section>
  </main>
  <script>
    const API_BASE = "{{PLAYPAGE_API_BASE}}";
    const PROJECT_KEY = "{{PLAYPAGE_PUBLIC_KEY}}";
    const PREVIEW = "{{PLAYPAGE_PREVIEW}}" === "true";
    const previewMessages = [
      { id: "preview-1", data: { nickname: "模板预览", content: "这是预览数据，不会保存。" }, createdAt: new Date().toISOString() }
    ];

    function esc(value) {
      return String(value || "").replace(/[&<>"]/g, function (char) {
        return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[char];
      });
    }

    async function api(path, options) {
      options = options || {};
      if (PREVIEW) return previewApi(path, options);
      const response = await fetch(API_BASE + path, {
        ...options,
        headers: {
          "Content-Type": "application/json",
          "X-Project-Key": PROJECT_KEY,
          ...(options.headers || {})
        }
      });
      const payload = await response.json().catch(function () { return {}; });
      if (!response.ok) throw new Error(payload.error || "请求失败");
      return payload;
    }

    async function previewApi(path, options) {
      await new Promise(function (resolve) { setTimeout(resolve, 120); });
      if (path === "/collections/messages/records" && (!options.method || options.method === "GET")) {
        return { items: previewMessages };
      }
      if (path === "/collections/messages/records" && options.method === "POST") {
        const body = JSON.parse(options.body || "{}");
        const item = { id: "preview-" + (previewMessages.length + 1), data: body.data || {}, createdAt: new Date().toISOString() };
        previewMessages.push(item);
        return item;
      }
      throw new Error("预览模式不支持这个操作");
    }

    async function loadMessages() {
      try {
        const data = await api("/collections/messages/records");
        const items = data.items || [];
        list.innerHTML = items.length ? items.map(function (item) {
          return '<article class="msg"><strong>' + esc(item.data.nickname) + '</strong><div>' + esc(item.data.content) + '</div><div class="meta">' + new Date(item.createdAt).toLocaleString() + '</div></article>';
        }).join("") : "还没有留言。";
      } catch (error) {
        list.textContent = error.message;
      }
    }

    form.addEventListener("submit", async function (event) {
      event.preventDefault();
      status.textContent = "正在发送……";
      try {
        await api("/collections/messages/records", {
          method: "POST",
          body: JSON.stringify({ data: { nickname: nickname.value.trim(), content: content.value.trim() } })
        });
        content.value = "";
        status.textContent = PREVIEW ? "预览留言已临时显示。" : "已发送。";
        await loadMessages();
      } catch (error) {
        status.textContent = error.message;
      }
    });

    loadMessages();
  </script>
</body>
</html>`;

const initialParams: ParamDraft[] = [
  { name: "siteTitle", label: "网站标题", type: "string", required: true, defaultValue: "我的网站", placeholder: "例如：我的班级主页", help: "会替换 HTML 里的 {{siteTitle}}。", options: "" },
  { name: "subtitle", label: "副标题", type: "text", required: false, defaultValue: "这里写一句介绍。", placeholder: "一句话介绍网站", help: "会替换 HTML 里的 {{subtitle}}。", options: "" },
  { name: "themeColor", label: "主题色", type: "color", required: false, defaultValue: "#2563eb", placeholder: "#2563eb", help: "颜色必须是 #RRGGBB 格式。", options: "" }
];

const messageCollections = `[
  {
    "name": "messages",
    "permissions": { "publicRead": true, "publicWrite": true },
    "fields": [
      { "name": "nickname", "type": "string", "required": true, "isList": false },
      { "name": "content", "type": "text", "required": true, "isList": false }
    ]
  }
]`;

const forumCollectionsExample = `[
  {
    "name": "posts",
    "permissions": { "publicRead": true, "publicWrite": true },
    "fields": [
      { "name": "title", "type": "string", "required": true, "isList": false },
      { "name": "content", "type": "text", "required": true, "isList": false },
      { "name": "author", "type": "string", "required": true, "isList": false }
    ]
  },
  {
    "name": "comments",
    "permissions": { "publicRead": true, "publicWrite": true },
    "fields": [
      { "name": "postId", "type": "string", "required": true, "isList": false },
      { "name": "content", "type": "text", "required": true, "isList": false },
      { "name": "author", "type": "string", "required": true, "isList": false }
    ]
  }
]`;

function paramsToJSON(params: ParamDraft[]): string {
  return JSON.stringify(params.map((item) => ({
    name: item.name.trim(),
    label: item.label.trim(),
    type: item.type,
    required: item.required,
    default: item.defaultValue,
    placeholder: item.placeholder || undefined,
    help: item.help || undefined,
    options: item.type === "select" ? item.options.split(/[,，\n]/).map((option) => option.trim()).filter(Boolean) : undefined
  })), null, 2);
}

function safeColor(value: string): string {
  const trimmed = value.trim();
  if (/^#[0-9a-fA-F]{6}$/.test(trimmed)) {
    return trimmed;
  }
  return "#2563eb";
}

export function TemplateSubmitPage() {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [category, setCategory] = useState("community");
  const [categoryLabel, setCategoryLabel] = useState("用户投稿");
  const [summary, setSummary] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState("");
  const [interactiveRequired, setInteractiveRequired] = useState(false);
  const [analyticsRecommended, setAnalyticsRecommended] = useState(true);
  const [paramDrafts, setParamDrafts] = useState<ParamDraft[]>(initialParams);
  const [configFields, setConfigFields] = useState(paramsToJSON(initialParams));
  const [collections, setCollections] = useState("[]");
  const [mode, setMode] = useState<"paste" | "file" | "zip">("paste");
  const [htmlSource, setHtmlSource] = useState(minimalTemplateHTML);
  const [file, setFile] = useState<File | null>(null);
  const [statusText, setStatusText] = useState("建议先点“套用静态页面示例”或“套用留言板示例”，再按需要修改。");
  const [statusTone, setStatusTone] = useState<"info" | "success" | "error">("info");
  const [working, setWorking] = useState(false);

  const accept = useMemo(() => (mode === "zip" ? ".zip" : ".html,.htm,.txt"), [mode]);

  function updateParam(index: number, patch: Partial<ParamDraft>) {
    setParamDrafts((current) => {
      const next = current.map((item, itemIndex) => itemIndex === index ? { ...item, ...patch } : item);
      setConfigFields(paramsToJSON(next));
      return next;
    });
  }

  function addParam() {
    setParamDrafts((current) => {
      const newParam: ParamDraft = { name: "newParam", label: "新参数", type: "string", required: false, defaultValue: "", placeholder: "", help: "", options: "" };
      const next = [...current, newParam];
      setConfigFields(paramsToJSON(next));
      return next;
    });
  }

  function removeParam(index: number) {
    setParamDrafts((current) => {
      const next = current.filter((_, itemIndex) => itemIndex !== index);
      setConfigFields(paramsToJSON(next));
      return next;
    });
  }

  function useStaticPreset() {
    setName("静态展示页模板");
    setSlug("simple-page-template");
    setSummary("一个可配置标题、副标题和主题色的静态页面。");
    setDescription("适合做个人介绍、班级公告、活动页面等简单展示站点。");
    setTags("静态页面, 展示, 入门");
    setInteractiveRequired(false);
    setCollections("[]");
    setMode("paste");
    setHtmlSource(minimalTemplateHTML);
    setParamDrafts(initialParams);
    setConfigFields(paramsToJSON(initialParams));
    setStatusText("已套用静态页面示例，可以直接提交测试。");
    setStatusTone("success");
  }

  function useInteractivePreset() {
    setName("留言板模板");
    setSlug("message-board-template");
    setSummary("带 messages 数据表声明的留言板模板。");
    setDescription("创建作品时会自动创建 messages 数据表。正式发布后，访客留言会保存到当前作品的数据表里。");
    setTags("留言板, 互动, 数据表");
    setInteractiveRequired(true);
    setCollections(messageCollections);
    setMode("paste");
    setHtmlSource(interactiveTemplateHTML);
    setParamDrafts(initialParams);
    setConfigFields(paramsToJSON(initialParams));
    setStatusText("已套用留言板示例。这个模板需要互动功能。");
    setStatusTone("success");
  }

  async function submit() {
    try {
      JSON.parse(configFields || "[]");
      JSON.parse(collections || "[]");
    } catch {
      setStatusText("参数声明或数据表声明不是合法 JSON。");
      setStatusTone("error");
      return;
    }
    if (!name.trim() || !summary.trim()) {
      setStatusText("模板名称和简介都要填写。");
      setStatusTone("error");
      return;
    }
    if (mode === "paste" && !htmlSource.trim()) {
      setStatusText("请粘贴模板 HTML。");
      setStatusTone("error");
      return;
    }
    if (mode !== "paste" && !file) {
      setStatusText("请选择要上传的模板文件。");
      setStatusTone("error");
      return;
    }

    const form = new FormData();
    form.append("name", name);
    form.append("slug", slug);
    form.append("category", category);
    form.append("categoryLabel", categoryLabel);
    form.append("summary", summary);
    form.append("description", description);
    form.append("tags", tags);
    form.append("interactiveRequired", String(interactiveRequired));
    form.append("analyticsRecommended", String(analyticsRecommended));
    form.append("configFields", configFields);
    form.append("collections", collections);
    form.append("sourceType", mode === "zip" ? "zip" : mode === "file" ? "file" : "text");
    if (mode === "paste") {
      form.append("htmlSource", htmlSource);
    } else if (file) {
      form.append("file", file);
    }

    try {
      setWorking(true);
      setStatusText("正在提交模板审核……");
      setStatusTone("info");
      await postForm<TemplateSubmission>("/api/v1/template-submissions", form);
      setStatusText("模板已提交审核。管理员发布后会出现在模板市场。");
      setStatusTone("success");
      setFile(null);
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : "提交模板失败。");
      setStatusTone("error");
    } finally {
      setWorking(false);
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>投稿模板</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>
          模板本质上是一个 HTML 文件，加上一份参数声明和可选的数据表声明。用户从模板创建作品时，PlayPage 会把 HTML 里的占位符替换为用户填写的参数。
        </p>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <button className="button-secondary" type="button" onClick={useStaticPreset}>套用静态页面示例</button>
          <button className="button-secondary" type="button" onClick={useInteractivePreset}>套用留言板示例</button>
        </div>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite">{statusText}</div>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        <h2 style={{ margin: 0 }}>1. 模板基本信息</h2>
        <div className="card-grid">
          <label className="field"><span>模板名称</span><input value={name} onChange={(event) => setName(event.target.value)} placeholder="例如：班级留言墙" /></label>
          <label className="field"><span>链接名</span><input value={slug} onChange={(event) => setSlug(event.target.value)} placeholder="例如：class-message-wall" /><span className="field-note">只用于模板详情页地址，不是用户作品地址。</span></label>
          <label className="field"><span>分类 ID</span><input value={category} onChange={(event) => setCategory(event.target.value)} placeholder="community" /><span className="field-note">英文或拼音，例如 site、tool、game、community。</span></label>
          <label className="field"><span>分类显示名</span><input value={categoryLabel} onChange={(event) => setCategoryLabel(event.target.value)} placeholder="用户投稿" /></label>
        </div>
        <label className="field"><span>一句话简介</span><input value={summary} onChange={(event) => setSummary(event.target.value)} maxLength={200} placeholder="告诉用户这个模板能做什么" /></label>
        <label className="field"><span>详细说明</span><textarea rows={4} value={description} onChange={(event) => setDescription(event.target.value)} placeholder="适用场景、创建后效果、是否依赖互动功能等。" /></label>
        <label className="field"><span>标签</span><input value={tags} onChange={(event) => setTags(event.target.value)} placeholder="论坛, 留言, 班级" /></label>
        <div style={{ display: "flex", gap: 14, flexWrap: "wrap" }}>
          <label><input type="checkbox" checked={interactiveRequired} onChange={(event) => setInteractiveRequired(event.target.checked)} /> 需要互动功能</label>
          <label><input type="checkbox" checked={analyticsRecommended} onChange={(event) => setAnalyticsRecommended(event.target.checked)} /> 建议开启统计</label>
        </div>
      </section>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        <h2 style={{ margin: 0 }}>2. 参数声明</h2>
        <p style={{ margin: 0, color: "var(--muted)" }}>
          参数会出现在创建作品页面。比如声明了 <code>themeColor</code>，HTML 里写 <code>{"{{themeColor}}"}</code>，创建时就会被替换成用户选择的颜色。
        </p>
        {paramDrafts.map((param, index) => (
          <fieldset key={`${param.name}-${index}`} style={{ border: "1px solid var(--line)", borderRadius: 18, padding: 16, display: "grid", gap: 12 }}>
            <legend>参数 {index + 1}</legend>
            <div className="card-grid">
              <label className="field"><span>参数名</span><input value={param.name} onChange={(event) => updateParam(index, { name: event.target.value })} placeholder="siteTitle" /></label>
              <label className="field"><span>显示标签</span><input value={param.label} onChange={(event) => updateParam(index, { label: event.target.value })} placeholder="网站标题" /></label>
              <label className="field"><span>类型</span><select value={param.type} onChange={(event) => updateParam(index, { type: event.target.value as ParamDraft["type"] })}><option value="string">单行文本</option><option value="text">多行文本</option><option value="color">颜色选择器</option><option value="select">下拉选择</option></select></label>
              <label className="field"><span>默认值</span><input type={param.type === "color" ? "color" : "text"} value={param.type === "color" ? safeColor(param.defaultValue) : param.defaultValue} onChange={(event) => updateParam(index, { defaultValue: event.target.value })} placeholder={param.type === "color" ? "#2563eb" : ""} /></label>
            </div>
            <label className="field"><span>提示文字</span><input value={param.placeholder} onChange={(event) => updateParam(index, { placeholder: event.target.value })} /></label>
            <label className="field"><span>帮助说明</span><input value={param.help} onChange={(event) => updateParam(index, { help: event.target.value })} /></label>
            {param.type === "select" ? <label className="field"><span>下拉选项</span><input value={param.options} onChange={(event) => updateParam(index, { options: event.target.value })} placeholder="选项一, 选项二, 选项三" /></label> : null}
            <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
              <label><input type="checkbox" checked={param.required} onChange={(event) => updateParam(index, { required: event.target.checked })} /> 必填</label>
              <button className="button-ghost" type="button" onClick={() => removeParam(index)}>删除这个参数</button>
            </div>
          </fieldset>
        ))}
        <div><button className="button-secondary" type="button" onClick={addParam}>新增参数</button></div>
        <details>
          <summary>高级：直接编辑参数声明 JSON</summary>
          <label className="field" style={{ marginTop: 12 }}>
            <span>参数声明 JSON</span>
            <textarea rows={10} value={configFields} onChange={(event) => setConfigFields(event.target.value)} />
          </label>
        </details>
      </section>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        <h2 style={{ margin: 0 }}>3. 数据表声明</h2>
        <p style={{ margin: 0, color: "var(--muted)" }}>
          只有留言板、论坛、排行榜、云存档这类需要在线保存数据的模板才需要声明数据表。静态页面保持空数组 <code>[]</code> 即可。
        </p>
        <div className="status" aria-live="polite">
          数据表声明不是“示例数据”，而是告诉 PlayPage：用户从这个模板创建作品时，要自动为这个作品创建哪些数据集合。
        </div>
        <label className="field"><span>数据表声明 JSON</span><textarea rows={8} value={collections} onChange={(event) => setCollections(event.target.value)} /></label>
        <details>
          <summary>数据集合声明怎么写</summary>
          <div style={{ display: "grid", gap: 12, marginTop: 12 }}>
            <p style={{ margin: 0 }}>
              整体必须是数组。数组里的每一项代表一个数据集合。集合创建后，模板 HTML 里的 JS 可以通过互动 API 读写它。
            </p>
            <ul>
              <li><code>name</code>：集合名，只建议用英文、数字、下划线，例如 <code>messages</code>、<code>posts</code>、<code>scores</code>。</li>
              <li><code>permissions.publicRead</code>：访客是否可以读取。留言板、论坛、排行榜通常设为 <code>true</code>。</li>
              <li><code>permissions.publicWrite</code>：访客是否可以新增或修改。允许发帖、评论、留言时必须设为 <code>true</code>。</li>
              <li><code>fields</code>：字段声明数组。每个字段描述一项业务数据，比如标题、正文、作者、分数。</li>
            </ul>
            <p style={{ margin: 0 }}>
              字段支持常用类型：<code>string</code> 短文本、<code>text</code> 长文本、<code>number</code> 数字、<code>boolean</code> 布尔值。<code>required</code> 表示写入时必须提供，<code>isList</code> 表示这个字段是否是数组，普通字段一般写 <code>false</code>。
            </p>
            <p style={{ margin: 0 }}>
              例子：论坛通常需要两个集合。<code>posts</code> 保存帖子，<code>comments</code> 保存回复；回复通过 <code>postId</code> 关联到帖子 ID。
            </p>
            <pre style={{ whiteSpace: "pre-wrap" }}>{forumCollectionsExample}</pre>
          </div>
        </details>
      </section>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        <h2 style={{ margin: 0 }}>4. 模板源码</h2>
        <fieldset style={{ border: "1px solid var(--line)", borderRadius: 18, padding: 16 }}>
          <legend>源码方式</legend>
          <div style={{ display: "flex", gap: 12, flexWrap: "wrap", marginBottom: 12 }}>
            <label><input type="radio" checked={mode === "paste"} onChange={() => setMode("paste")} /> 粘贴 HTML</label>
            <label><input type="radio" checked={mode === "file"} onChange={() => setMode("file")} /> 上传 HTML</label>
            <label><input type="radio" checked={mode === "zip"} onChange={() => setMode("zip")} /> 上传 ZIP</label>
          </div>
          {mode === "paste" ? (
            <label className="field"><span>HTML 代码</span><textarea rows={18} value={htmlSource} onChange={(event) => setHtmlSource(event.target.value)} placeholder="请粘贴完整 HTML，包含 <!doctype html> 和 <html>。" /></label>
          ) : (
            <label className="field"><span>模板文件</span><input type="file" accept={accept} onChange={(event) => setFile(event.target.files?.[0] ?? null)} /><span className="field-note">ZIP 会优先读取 index.html；没有 index.html 时读取第一个 HTML 文件。</span></label>
          )}
        </fieldset>
        <div><button className="button-primary" type="button" disabled={working} onClick={submit}>{working ? "正在提交……" : "提交审核"}</button></div>
      </section>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 14 }}>
        <h2 style={{ margin: 0 }}>模板开发规范</h2>
        <h3 style={{ margin: 0 }}>必须知道的规则</h3>
        <ol>
          <li>模板必须是完整 HTML 页面，建议包含 <code>&lt;!doctype html&gt;</code>、<code>&lt;html lang="zh-CN"&gt;</code>、<code>&lt;meta charset="utf-8"&gt;</code>。</li>
          <li>参数占位符格式固定是 <code>{"{{参数名}}"}</code>，参数名要和参数声明里的 <code>name</code> 完全一致。</li>
          <li>颜色参数请使用 <code>type: "color"</code>，默认值必须是 <code>#RRGGBB</code>，例如 <code>#2563eb</code>。</li>
          <li>需要互动数据时，勾选“需要互动功能”，并声明要创建的数据表。</li>
          <li>投稿不会执行任何后端代码；模板里的 JS 只会在用户作品页面的浏览器中运行。</li>
        </ol>
        <h3 style={{ margin: 0 }}>系统占位符</h3>
        <ul>
          <li><code>{"{{PLAYPAGE_API_BASE}}"}</code>：当前作品互动 API 基础地址，例如 <code>/api/v1/public/projects/作品ID</code>。</li>
          <li><code>{"{{PLAYPAGE_PUBLIC_KEY}}"}</code>：当前作品公开互动密钥，用在 <code>X-Project-Key</code> 请求头。</li>
          <li><code>{"{{PLAYPAGE_PREVIEW}}"}</code>：预览时是 <code>true</code>，正式发布时是 <code>false</code>。</li>
          <li><code>{"{{PROJECT_ID}}"}</code>、<code>{"{{PROJECT_NAME}}"}</code>：当前作品 ID 和名称。</li>
        </ul>
        <h3 style={{ margin: 0 }}>预览机制</h3>
        <ul>
          <li>模板详情页的“打开完整预览”会用参数默认值渲染 HTML，不会创建真实作品，也不会创建真实数据集合。</li>
          <li>预览时 <code>{"{{PLAYPAGE_PREVIEW}}"}</code> 会被替换成 <code>true</code>；用户真正创建作品后会替换成 <code>false</code>。</li>
          <li>预览时 <code>{"{{PLAYPAGE_API_BASE}}"}</code> 和 <code>{"{{PLAYPAGE_PUBLIC_KEY}}"}</code> 会替换成演示值，不能当成真实数据接口使用。</li>
          <li>互动模板必须自己判断预览模式：预览时使用浏览器内存里的假数据；正式发布时才调用互动 API。</li>
          <li>如果模板预览直接请求真实 API，通常会失败；正确做法是像下方示例一样封装 <code>api</code> 和 <code>previewApi</code>。</li>
        </ul>
        <h3 style={{ margin: 0 }}>互动请求最小写法</h3>
        <pre style={{ whiteSpace: "pre-wrap" }}>{`const API_BASE = "{{PLAYPAGE_API_BASE}}";
const PROJECT_KEY = "{{PLAYPAGE_PUBLIC_KEY}}";
const PREVIEW = "{{PLAYPAGE_PREVIEW}}" === "true";

async function api(path, options = {}) {
  if (PREVIEW) {
    return previewApi(path, options);
  }

  const response = await fetch(API_BASE + path, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      "X-Project-Key": PROJECT_KEY,
      ...(options.headers || {})
    }
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data.error || "请求失败");
  return data;
}

const previewMessages = [
  { id: "preview-1", data: { nickname: "预览用户", content: "这是预览数据，不会保存。" }, createdAt: new Date().toISOString() }
];

async function previewApi(path, options = {}) {
  if (path === "/collections/messages/records" && (!options.method || options.method === "GET")) {
    return { items: previewMessages };
  }
  if (path === "/collections/messages/records" && options.method === "POST") {
    const body = JSON.parse(options.body || "{}");
    const item = { id: "preview-" + (previewMessages.length + 1), data: body.data || {}, createdAt: new Date().toISOString() };
    previewMessages.push(item);
    return item;
  }
  throw new Error("预览模式不支持这个操作");
}`}</pre>
      </section>
    </section>
  );
}
