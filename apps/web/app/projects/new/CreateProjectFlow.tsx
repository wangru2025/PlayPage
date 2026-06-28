"use client";

import { useEffect, useId, useState } from "react";
import { buildURL, getCSRFToken, getJSON, postJSON, reportClientError } from "@/lib/api";
import type { ProjectTemplate, TemplateConfigField } from "@/app/templates/templateTypes";

type CreateProjectResponse = {
  id: string;
  username: string;
  slug: string;
  name: string;
  interactive: boolean;
  analyticsEnabled: boolean;
  visibility: string;
  publicUrl: string;
  currentReleaseId?: string;
};

type ReleaseResponse = {
  id: string;
  projectId: string;
  status: string;
  archivePath: string;
  publicPath: string;
  entryFile: string;
  changeNote: string;
  warnings?: string[];
  createdAt: string;
};

type TemplateDetailResponse = {
  template: ProjectTemplate;
};

type UploadMode = "zip" | "html" | "text" | "template";
type Step = "fill" | "publishing" | "done";

const maxContentBytes = 10 * 1024 * 1024;

const text = {
  title: "创建作品",
  intro: "先填写作品信息，再选择上传方式。",
  updateTitle: "上传新版本",
  updateIntro: "保留原来的作品地址，只替换里面的网页内容。",
  name: "作品名称",
  namePlaceholder: "例如：我的小站",
  slug: "作品链接名",
  slugPlaceholder: "例如：我的小站",
  interactive: "启用互动功能",
  interactiveHint: "勾选后，这个作品才会启用作品数据接口和互动能力。",
  interactiveTitle: "互动功能",
  interactiveSummary:
    "启用后，作品会获得在线数据接口。上传并创建作品后，页面会自动跳转到互动功能页，你可以在那里复制包含真实 key 的 API 文档。",
  interactiveListTitle: "它可以帮你做",
  interactiveList1: "让评论、留言、小论坛、插图内容真正保存到线上",
  interactiveList2: "让 AI 直接帮你创建作品数据表和读写代码",
  interactiveList3: "你不需要自己理解 key，只需要复制给 AI",
  analytics: "启用访问量统计",
  analyticsHint: "启用后，PlayPage 会在作品页面中加入访问统计代码，用来统计每日访问量和互动 API 请求情况。",
  analyticsTitle: "访问量统计说明",
  analyticsSummary: "启用后，PlayPage 会在作品 HTML 中加入一段访问统计代码。它会记录每日访问量和互动 API 请求统计，包括请求次数、成功次数、失败次数、成功率和失败率。它不会读取页面输入内容、密码或互动数据。你下载作品源码时，也会下载包含这段统计代码的版本。",
  uploadType: "上传方式",
  changeNote: "更新内容（可选）",
  changeNotePlaceholder: "例如：修复按钮无反应、增加排行榜、调整页面样式",
  zip: "ZIP 压缩包",
  html: "单个 HTML 文件",
  text: "直接粘贴 HTML 代码",
  template: "使用模板",
  templateHint: "先去模板市场选择模板，再回到这里填写模板参数并创建作品。",
  chooseTemplate: "前往模板市场选择模板",
  changeTemplate: "更换模板",
  selectedTemplate: "已选择模板：",
  templateParams: "模板参数",
  templateMissing: "请先选择模板。",
  templateLoading: "正在读取模板。",
  templateLoaded: "模板已读取，请填写参数。",
  zipHint: "上传内容不能超过 10MB。ZIP 只支持白名单静态资源文件。",
  htmlHint: "可以直接上传一个 .html、.ht m 或 .txt 文件，文件内容不能超过 10MB。".replace(" ", ""),
  textHint:
    "适合直接把 AI 生成的 HTML 源码粘贴进来，内容不能超过 10MB。请确认从第一行到最后一行都已经粘贴完整。",
  file: "选择文件",
  code: "HTML 代码",
  codePlaceholder:
    "<!doctype html>\n<html lang=\"zh-CN\">\n<head>\n  <meta charset=\"utf-8\" />\n  <title>我的作品</title>\n</head>\n<body>\n  <h1>你好</h1>\n</body>\n</html>",
  start: "上传作品",
  templateStart: "创建模板作品",
  updateStart: "上传新版本",
  creating: "正在创建作品并发布。",
  uploading: "正在上传内容。",
  progressTitle: "发布进度",
  progressDone: "发布完成",
  progressUnknown: "正在准备。",
  success: "作品已经发布完成，正在返回作品列表。",
  warningsTitle: "发布提醒：",
  fail: "创建或发布失败。",
  tooLarge: "上传内容不能超过 10MB。",
  fileMissing: "请先选择上传文件。",
  codeMissing: "请先粘贴 HTML 代码。",
  templateParamMissing: "请填写模板参数：",
  templateColorInvalid: "颜色参数必须是 #RRGGBB 格式：",
  requiredName: "请先填写作品名称。",
  requiredSlug: "请先填写作品链接名。",
  uploadedBytes: "已上传：",
  uploadSpeed: "当前速度：",
  unknownTotal: "未知",
  calculating: "计算中"
};

function normalizePathText(value: string): string {
  let normalized = value.trim().toLowerCase().replace(/\s+/g, "-");
  normalized = Array.from(normalized).map((char) => {
    if (/[-_.]/u.test(char) || /[\p{L}\p{N}]/u.test(char)) {
      return char;
    }
    return "-";
  }).join("");
  normalized = normalized.replace(/^-+|[-._]+$/g, "");
  while (normalized.includes("--")) {
    normalized = normalized.replaceAll("--", "-");
  }
  return normalized;
}

function normalizeColorValue(value: string, fallback = "#2563eb"): string {
  const trimmed = value.trim();
  if (/^#[0-9a-fA-F]{6}$/.test(trimmed)) {
    return trimmed.toLowerCase();
  }
  if (/^[0-9a-fA-F]{6}$/.test(trimmed)) {
    return `#${trimmed.toLowerCase()}`;
  }
  return fallback;
}

function isHexColor(value: string): boolean {
  return /^#[0-9a-fA-F]{6}$/.test(value.trim());
}

function uploadWithProgress<T>(
  url: string,
  body: FormData,
  onProgress: (loaded: number, total: number, speedBytes: number) => void
): Promise<T> {
  return new Promise((resolve, reject) => {
    const request = new XMLHttpRequest();
    const startedAt = Date.now();

    request.open("POST", buildURL(url), true);
    request.withCredentials = true;
    const csrfToken = getCSRFToken();
    if (csrfToken) {
      request.setRequestHeader("X-CSRF-Token", csrfToken);
    }

    request.upload.onprogress = (event) => {
      if (!event.lengthComputable) {
        onProgress(event.loaded, 0, 0);
        return;
      }

      const elapsed = Math.max((Date.now() - startedAt) / 1000, 0.001);
      onProgress(event.loaded, event.total, event.loaded / elapsed);
    };

    request.onreadystatechange = () => {
      if (request.readyState !== XMLHttpRequest.DONE) {
        return;
      }

      if (request.status >= 200 && request.status < 300) {
        resolve(JSON.parse(request.responseText) as T);
        return;
      }

      try {
        const payload = JSON.parse(request.responseText) as { error?: string };
        reject(new Error(payload.error ?? `请求失败：${request.status}`));
      } catch {
        reject(new Error(`请求失败：${request.status}`));
      }
    };

    request.onerror = () => reject(new Error("上传失败"));
    request.send(body);
  });
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) {
    return "0 B";
  }

  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(value >= 10 || index === 0 ? 0 : 1)} ${units[index]}`;
}

export function CreateProjectFlow() {
  const [routeProjectId, setRouteProjectId] = useState("");
  const [templateId, setTemplateId] = useState("");
  const [selectedTemplate, setSelectedTemplate] = useState<ProjectTemplate | null>(null);
  const [templateParams, setTemplateParams] = useState<Record<string, string>>({});
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugTouched, setSlugTouched] = useState(false);
  const [interactive, setInteractive] = useState(false);
  const [analyticsEnabled, setAnalyticsEnabled] = useState(false);
  const [mode, setMode] = useState<UploadMode>("zip");
  const [file, setFile] = useState<File | null>(null);
  const [htmlText, setHtmlText] = useState("");
  const [changeNote, setChangeNote] = useState("");
  const [step, setStep] = useState<Step>("fill");
  const [statusText, setStatusText] = useState("");
  const [progressLoaded, setProgressLoaded] = useState(0);
  const [progressTotal, setProgressTotal] = useState(0);
  const [progressSpeed, setProgressSpeed] = useState(0);
  const [releaseWarnings, setReleaseWarnings] = useState<string[]>([]);
  const [working, setWorking] = useState(false);
  const nameId = useId();
  const slugId = useId();
  const fileId = useId();
  const htmlId = useId();

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setRouteProjectId(params.get("projectId") ?? "");
    const source = params.get("source") ?? "";
    const nextTemplateId = params.get("templateId") ?? "";
    if (source === "template" && nextTemplateId) {
      setMode("template");
      setTemplateId(nextTemplateId);
    }
  }, []);

  useEffect(() => {
    if (!templateId) {
      setSelectedTemplate(null);
      setTemplateParams({});
      return;
    }
    let canceled = false;
    async function loadTemplate() {
      try {
        setStatusText(text.templateLoading);
        const payload = await getJSON<TemplateDetailResponse>(`/api/v1/templates/${encodeURIComponent(templateId)}`);
        if (canceled) return;
        setSelectedTemplate(payload.template);
        setTemplateParams((current) => {
          const next = { ...current };
          for (const field of payload.template.configFields) {
            if (next[field.name] === undefined) {
              next[field.name] = field.default ?? "";
            }
          }
          return next;
        });
        if (payload.template.interactiveRequired) {
          setInteractive(true);
        }
        if (payload.template.analyticsRecommended) {
          setAnalyticsEnabled(true);
        }
        setStatusText(text.templateLoaded);
      } catch (error) {
        if (canceled) return;
        setSelectedTemplate(null);
        setStatusText(error instanceof Error ? error.message : "读取模板失败。");
      }
    }
    void loadTemplate();
    return () => {
      canceled = true;
    };
  }, [templateId]);

  useEffect(() => {
    if (slugTouched) {
      return;
    }
    setSlug(normalizePathText(name));
  }, [name, slugTouched]);

  const progressPercent = progressTotal > 0 ? Math.min(Math.round((progressLoaded / progressTotal) * 100), 100) : 0;

  const isUpdateMode = routeProjectId !== "";
  const effectiveInteractive = interactive || (mode === "template" && selectedTemplate?.interactiveRequired === true);

  function updateTemplateParam(name: string, value: string) {
    setTemplateParams((current) => ({ ...current, [name]: value }));
  }

  function renderTemplateField(field: TemplateConfigField) {
    const value = templateParams[field.name] ?? field.default ?? "";
    const id = `template-param-${field.name}`;
    if (field.type === "text") {
      return (
        <label key={field.name} className="field" htmlFor={id}>
          <span>{field.label}{field.required ? " *" : ""}</span>
          <textarea
            id={id}
            rows={4}
            value={value}
            onChange={(event) => updateTemplateParam(field.name, event.target.value)}
            placeholder={field.placeholder}
          />
          {field.help ? <span className="field-note">{field.help}</span> : null}
        </label>
      );
    }
    if (field.type === "select") {
      return (
        <label key={field.name} className="field" htmlFor={id}>
          <span>{field.label}{field.required ? " *" : ""}</span>
          <select id={id} value={value} onChange={(event) => updateTemplateParam(field.name, event.target.value)}>
            {(field.options ?? []).map((option) => <option key={option} value={option}>{option}</option>)}
          </select>
          {field.help ? <span className="field-note">{field.help}</span> : null}
        </label>
      );
    }
    if (field.type === "color") {
      const colorValue = normalizeColorValue(value, normalizeColorValue(field.default || "#2563eb"));
      return (
        <div key={field.name} className="field">
          <label htmlFor={id}>
            <span>{field.label}{field.required ? " *" : ""}</span>
          </label>
          <div style={{ display: "grid", gridTemplateColumns: "72px minmax(160px, 1fr)", gap: 10, alignItems: "center" }}>
            <input
              id={id}
              type="color"
              value={colorValue}
              aria-label={`${field.label}颜色选择器`}
              onChange={(event) => updateTemplateParam(field.name, event.target.value)}
              style={{ width: 72, minHeight: 44, padding: 4 }}
            />
            <input
              type="text"
              value={value}
              onChange={(event) => updateTemplateParam(field.name, normalizeColorValue(event.target.value, event.target.value))}
              onBlur={(event) => updateTemplateParam(field.name, normalizeColorValue(event.target.value, colorValue))}
              placeholder={field.placeholder || "#2563eb"}
              aria-label={`${field.label}十六进制颜色值`}
            />
          </div>
          <span className="field-note">
            {field.help || "可以直接点左侧选择颜色，也可以输入 #RRGGBB 格式，例如 #2563eb。"}
          </span>
        </div>
      );
    }
    return (
      <label key={field.name} className="field" htmlFor={id}>
        <span>{field.label}{field.required ? " *" : ""}</span>
        <input
          id={id}
          type={field.type === "color" ? "color" : "text"}
          value={value}
          onChange={(event) => updateTemplateParam(field.name, event.target.value)}
          placeholder={field.placeholder}
        />
        {field.help ? <span className="field-note">{field.help}</span> : null}
      </label>
    );
  }

  async function submit() {
    if (working) {
      return;
    }
    if (!isUpdateMode && name.trim() === "") {
      setStatusText(text.requiredName);
      return;
    }
    if (!isUpdateMode && slug.trim() === "") {
      setStatusText(text.requiredSlug);
      return;
    }
    if ((mode === "zip" || mode === "html") && !file) {
      setStatusText(text.fileMissing);
      return;
    }
    if ((mode === "zip" || mode === "html") && file && file.size > maxContentBytes) {
      setStatusText(text.tooLarge);
      return;
    }
    if (mode === "text" && htmlText.trim() === "") {
      setStatusText(text.codeMissing);
      return;
    }
    if (mode === "text" && new Blob([htmlText]).size > maxContentBytes) {
      setStatusText(text.tooLarge);
      return;
    }
    if (mode === "template") {
      if (!selectedTemplate) {
        setStatusText(text.templateMissing);
        return;
      }
      for (const field of selectedTemplate.configFields) {
        if (field.required && (templateParams[field.name] ?? "").trim() === "") {
          setStatusText(`${text.templateParamMissing}${field.label}`);
          return;
        }
        if (field.type === "color") {
          const colorValue = (templateParams[field.name] ?? field.default ?? "").trim();
          if (colorValue && !isHexColor(colorValue)) {
            setStatusText(`${text.templateColorInvalid}${field.label}`);
            return;
          }
        }
      }
    }

    try {
      setWorking(true);
      setStep("publishing");
      setStatusText(isUpdateMode ? text.uploading : text.creating);
      setProgressLoaded(0);
      setProgressTotal(0);
      setProgressSpeed(0);
      setReleaseWarnings([]);

      let targetProjectId = routeProjectId;
      let targetInteractive = effectiveInteractive;
      if (!isUpdateMode) {
        const project = await postJSON<CreateProjectResponse>("/api/v1/projects", {
          name,
          slug: normalizePathText(slug),
          interactive: effectiveInteractive,
          analyticsEnabled
        });
        targetProjectId = project.id;
        targetInteractive = project.interactive;
        setStatusText(text.uploading);
      }

      if (mode === "zip" || mode === "html") {
        const form = new FormData();
        form.append("file", file as File);
        form.append("changeNote", changeNote);
        const release = await uploadWithProgress<ReleaseResponse>(
          `/api/v1/projects/${targetProjectId}/releases?mode=${mode}`,
          form,
          (loaded, total, speedBytes) => {
            setProgressLoaded(loaded);
            setProgressTotal(total);
            setProgressSpeed(speedBytes);
          }
        );
        setReleaseWarnings(release.warnings ?? []);
      } else if (mode === "text") {
        const release = await postJSON<ReleaseResponse>(`/api/v1/projects/${targetProjectId}/releases?mode=text`, {
          html: htmlText,
          changeNote
        });
        setProgressLoaded(1);
        setProgressTotal(1);
        setReleaseWarnings(release.warnings ?? []);
      } else {
        const release = await postJSON<ReleaseResponse>(`/api/v1/projects/${targetProjectId}/releases?mode=template`, {
          templateId: selectedTemplate?.id ?? templateId,
          params: templateParams,
          changeNote
        });
        setProgressLoaded(1);
        setProgressTotal(1);
        setReleaseWarnings(release.warnings ?? []);
      }

      setStep("done");
      setStatusText(text.success);
      setTimeout(() => {
        window.location.href = !isUpdateMode && targetInteractive
          ? `/projects/${targetProjectId}/interactive`
          : "/projects";
      }, 1200);
    } catch (error) {
      void reportClientError({
        source: "projects-new-submit",
        mode,
        routeProjectId,
        fileName: file?.name ?? "",
        fileSize: file?.size ?? 0,
        projectName: name,
        slug,
        message: error instanceof Error ? error.message : text.fail
      });
      setStep("fill");
      setStatusText(error instanceof Error ? error.message : text.fail);
    } finally {
      setWorking(false);
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 10 }}>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{isUpdateMode ? text.updateTitle : text.title}</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>{isUpdateMode ? text.updateIntro : text.intro}</p>
      </header>

      {step === "fill" ? (
        <section className="panel" style={{ padding: 24, display: "grid", gap: 18 }}>
          {!isUpdateMode ? (
            <>
              <div className="card-grid">
                <div className="field">
                  <label htmlFor={nameId}>{text.name}</label>
                  <input
                    id={nameId}
                    type="text"
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    placeholder={text.namePlaceholder}
                  />
                </div>
                <div className="field">
                  <label htmlFor={slugId}>{text.slug}</label>
                  <input
                    id={slugId}
                    type="text"
                    value={slug}
                    onChange={(event) => {
                      setSlugTouched(true);
                      setSlug(event.target.value);
                    }}
                    placeholder={text.slugPlaceholder}
                  />
                </div>
              </div>

              <div className="field">
                <label>
                  <input
                    type="checkbox"
                    checked={interactive}
                    onChange={(event) => setInteractive(event.target.checked)}
                  />
                  {" "}
                  {text.interactive}
                </label>
                <p className="field-note">{text.interactiveHint}</p>
              </div>

              <div className="field">
                <label>
                  <input
                    type="checkbox"
                    checked={analyticsEnabled}
                    onChange={(event) => setAnalyticsEnabled(event.target.checked)}
                  />
                  {" "}
                  {text.analytics}
                </label>
                <p className="field-note">{text.analyticsHint}</p>
              </div>

              {analyticsEnabled ? (
                <section
                  aria-label={text.analyticsTitle}
                  style={{
                    display: "grid",
                    gap: 12,
                    padding: 18,
                    borderRadius: 20,
                    border: "1px solid var(--line)",
                    background: "rgba(255,255,255,0.72)"
                  }}
                >
                  <h2 style={{ margin: 0, fontSize: "1.2rem" }}>{text.analyticsTitle}</h2>
                  <p style={{ margin: 0, color: "var(--muted)" }}>{text.analyticsSummary}</p>
                </section>
              ) : null}

              {interactive ? (
                <section
                  aria-label={text.interactiveTitle}
                  style={{
                    display: "grid",
                    gap: 12,
                    padding: 18,
                    borderRadius: 20,
                    border: "1px solid var(--line)",
                    background: "rgba(255,255,255,0.72)"
                  }}
                >
                  <h2 style={{ margin: 0, fontSize: "1.2rem" }}>{text.interactiveTitle}</h2>
                  <p style={{ margin: 0, color: "var(--muted)" }}>{text.interactiveSummary}</p>
                  <div>
                    <strong>{text.interactiveListTitle}</strong>
                    <ul style={{ margin: "8px 0 0", paddingLeft: 20 }}>
                      <li>{text.interactiveList1}</li>
                      <li>{text.interactiveList2}</li>
                      <li>{text.interactiveList3}</li>
                    </ul>
                  </div>
                </section>
              ) : null}
            </>
          ) : null}

          <fieldset className="field" style={{ border: 0, padding: 0, margin: 0 }}>
            <legend>{text.uploadType}</legend>
            <label>
              <input type="radio" name="upload-mode" checked={mode === "zip"} onChange={() => setMode("zip")} />
              {" "}
              {text.zip}
            </label>
            <label>
              <input type="radio" name="upload-mode" checked={mode === "html"} onChange={() => setMode("html")} />
              {" "}
              {text.html}
            </label>
            <label>
              <input type="radio" name="upload-mode" checked={mode === "text"} onChange={() => setMode("text")} />
              {" "}
              {text.text}
            </label>
            {!isUpdateMode ? (
              <label>
                <input type="radio" name="upload-mode" checked={mode === "template"} onChange={() => setMode("template")} />
                {" "}
                {text.template}
              </label>
            ) : null}
          </fieldset>

          {mode === "zip" || mode === "html" ? (
            <div className="field">
              <label htmlFor={fileId}>{text.file}</label>
              <input
                id={fileId}
                type="file"
                accept={mode === "zip" ? ".zip" : ".html,.htm,.txt"}
                onChange={(event) => setFile(event.target.files?.[0] ?? null)}
              />
              <p className="field-note">{mode === "zip" ? text.zipHint : text.htmlHint}</p>
            </div>
          ) : mode === "text" ? (
            <div className="field">
              <label htmlFor={htmlId}>{text.code}</label>
              <textarea
                id={htmlId}
                rows={16}
                value={htmlText}
                onChange={(event) => setHtmlText(event.target.value)}
                placeholder={text.codePlaceholder}
              />
              <p className="field-note">{text.textHint}</p>
            </div>
          ) : (
            <section
              aria-label={text.template}
              style={{
                display: "grid",
                gap: 16,
                padding: 18,
                borderRadius: 20,
                border: "1px solid var(--line)",
                background: "rgba(255,255,255,0.72)"
              }}
            >
              <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap", alignItems: "center" }}>
                <div>
                  <h2 style={{ margin: 0, fontSize: "1.2rem" }}>{text.template}</h2>
                  <p style={{ margin: "4px 0 0", color: "var(--muted)" }}>{text.templateHint}</p>
                </div>
                <a className="button-secondary" href="/templates">
                  {selectedTemplate ? text.changeTemplate : text.chooseTemplate}
                </a>
              </div>

              {selectedTemplate ? (
                <>
                  <div className="status" data-tone="success" aria-live="polite">
                    {text.selectedTemplate}
                    <strong>{selectedTemplate.name}</strong>
                  </div>
                  {selectedTemplate.interactiveRequired ? (
                    <p className="field-note" style={{ margin: 0 }}>
                      这个模板需要互动功能，创建时会自动开启互动功能。
                    </p>
                  ) : null}
                  <div style={{ display: "grid", gap: 14 }}>
                    <h3 style={{ margin: 0 }}>{text.templateParams}</h3>
                    {selectedTemplate.configFields.map((field) => renderTemplateField(field))}
                  </div>
                </>
              ) : (
                <div className="status" aria-live="polite">{text.templateMissing}</div>
              )}
            </section>
          )}

          <div className="field">
            <label htmlFor="change-note">{text.changeNote}</label>
            <input
              id="change-note"
              type="text"
              value={changeNote}
              onChange={(event) => setChangeNote(event.target.value)}
              placeholder={text.changeNotePlaceholder}
              maxLength={500}
            />
          </div>

          {statusText ? <div className="status" aria-live="polite">{statusText}</div> : null}

          <div>
            <button className="button-primary" type="button" disabled={working} onClick={submit}>
              {isUpdateMode ? text.updateStart : mode === "template" ? text.templateStart : text.start}
            </button>
          </div>
        </section>
      ) : (
        <section className="panel" style={{ padding: 24, display: "grid", gap: 18 }}>
          <h2 style={{ margin: 0 }}>{text.progressTitle}</h2>
          <div className="status" aria-live="polite">
            {step === "publishing" ? statusText : text.progressDone}
          </div>
          <div style={{ display: "grid", gap: 8 }}>
            <div
              aria-hidden="true"
              style={{
                width: "100%",
                height: 14,
                borderRadius: 999,
                background: "rgba(0,0,0,0.08)",
                overflow: "hidden"
              }}
            >
              <div
                style={{
                  width: `${progressPercent}%`,
                  height: "100%",
                  background: "linear-gradient(90deg, #d97706, #f59e0b)"
                }}
              />
            </div>
            <p style={{ margin: 0, color: "var(--muted)" }}>{statusText || text.progressUnknown}</p>
            <p style={{ margin: 0, color: "var(--muted)" }}>
              {text.uploadedBytes}
              {formatBytes(progressLoaded)} / {progressTotal > 0 ? formatBytes(progressTotal) : text.unknownTotal}
            </p>
            <p style={{ margin: 0, color: "var(--muted)" }}>
              {text.uploadSpeed}
              {progressSpeed > 0 ? `${formatBytes(progressSpeed)}/s` : text.calculating}
            </p>
            {releaseWarnings.length > 0 ? (
              <div className="status" style={{ borderColor: "#d97706", color: "#92400e", background: "rgba(251,191,36,0.14)" }}>
                <strong>{text.warningsTitle}</strong>
                <div>{releaseWarnings.join(" ")}</div>
              </div>
            ) : null}
          </div>
        </section>
      )}
    </section>
  );
}
