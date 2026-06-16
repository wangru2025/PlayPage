"use client";

import { useEffect, useId, useState } from "react";
import { buildURL, getCSRFToken, postJSON, reportClientError } from "@/lib/api";

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
  warnings?: string[];
  createdAt: string;
};

type UploadMode = "zip" | "html" | "text";
type Step = "fill" | "publishing" | "done";

const maxContentBytes = 10 * 1024 * 1024;

const text = {
  back: "\u8fd4\u56de\u4f5c\u54c1\u5217\u8868",
  title: "\u521b\u5efa\u4f5c\u54c1",
  intro: "\u5148\u586b\u5199\u4f5c\u54c1\u4fe1\u606f\uff0c\u518d\u9009\u62e9\u4e0a\u4f20\u65b9\u5f0f\u3002",
  updateTitle: "\u4e0a\u4f20\u65b0\u7248\u672c",
  updateIntro: "\u4fdd\u7559\u539f\u6765\u7684\u4f5c\u54c1\u5730\u5740\uff0c\u53ea\u66ff\u6362\u91cc\u9762\u7684\u7f51\u9875\u5185\u5bb9\u3002",
  name: "\u4f5c\u54c1\u540d\u79f0",
  namePlaceholder: "\u4f8b\u5982\uff1a\u6211\u7684\u5c0f\u7ad9",
  slug: "\u4f5c\u54c1\u94fe\u63a5\u540d",
  slugPlaceholder: "\u4f8b\u5982\uff1a\u6211\u7684\u5c0f\u7ad9",
  interactive: "\u542f\u7528\u4e92\u52a8\u529f\u80fd",
  interactiveHint: "\u52fe\u9009\u540e\uff0c\u8fd9\u4e2a\u4f5c\u54c1\u624d\u4f1a\u542f\u7528\u4f5c\u54c1\u6570\u636e\u63a5\u53e3\u548c\u4e92\u52a8\u80fd\u529b\u3002",
  interactiveTitle: "\u4e92\u52a8\u529f\u80fd",
  interactiveSummary:
    "\u542f\u7528\u540e\uff0c\u4f5c\u54c1\u4f1a\u83b7\u5f97\u5728\u7ebf\u6570\u636e\u63a5\u53e3\u3002\u4e0a\u4f20\u5e76\u521b\u5efa\u4f5c\u54c1\u540e\uff0c\u9875\u9762\u4f1a\u81ea\u52a8\u8df3\u8f6c\u5230\u4e92\u52a8\u529f\u80fd\u9875\uff0c\u4f60\u53ef\u4ee5\u5728\u90a3\u91cc\u590d\u5236\u5305\u542b\u771f\u5b9e key \u7684 API \u6587\u6863\u3002",
  interactiveListTitle: "\u5b83\u53ef\u4ee5\u5e2e\u4f60\u505a",
  interactiveList1: "\u8ba9\u8bc4\u8bba\u3001\u7559\u8a00\u3001\u5c0f\u8bba\u575b\u3001\u63d2\u56fe\u5185\u5bb9\u771f\u6b63\u4fdd\u5b58\u5230\u7ebf\u4e0a",
  interactiveList2: "\u8ba9 AI \u76f4\u63a5\u5e2e\u4f60\u521b\u5efa\u4f5c\u54c1\u6570\u636e\u8868\u548c\u8bfb\u5199\u4ee3\u7801",
  interactiveList3: "\u4f60\u4e0d\u9700\u8981\u81ea\u5df1\u7406\u89e3 key\uff0c\u53ea\u9700\u8981\u590d\u5236\u7ed9 AI",
  analytics: "\u542f\u7528\u8bbf\u95ee\u91cf\u7edf\u8ba1",
  analyticsHint: "\u542f\u7528\u540e\uff0cPlayPage \u4f1a\u5728\u4f5c\u54c1\u9875\u9762\u4e2d\u52a0\u5165\u8bbf\u95ee\u7edf\u8ba1\u4ee3\u7801\uff0c\u7528\u6765\u7edf\u8ba1\u6bcf\u65e5\u8bbf\u95ee\u91cf\u548c\u4e92\u52a8 API \u8bf7\u6c42\u60c5\u51b5\u3002",
  analyticsTitle: "\u8bbf\u95ee\u91cf\u7edf\u8ba1\u8bf4\u660e",
  analyticsSummary: "\u542f\u7528\u540e\uff0cPlayPage \u4f1a\u5728\u4f5c\u54c1 HTML \u4e2d\u52a0\u5165\u4e00\u6bb5\u8bbf\u95ee\u7edf\u8ba1\u4ee3\u7801\u3002\u5b83\u4f1a\u8bb0\u5f55\u6bcf\u65e5\u8bbf\u95ee\u91cf\u548c\u4e92\u52a8 API \u8bf7\u6c42\u7edf\u8ba1\uff0c\u5305\u62ec\u8bf7\u6c42\u6b21\u6570\u3001\u6210\u529f\u6b21\u6570\u3001\u5931\u8d25\u6b21\u6570\u3001\u6210\u529f\u7387\u548c\u5931\u8d25\u7387\u3002\u5b83\u4e0d\u4f1a\u8bfb\u53d6\u9875\u9762\u8f93\u5165\u5185\u5bb9\u3001\u5bc6\u7801\u6216\u4e92\u52a8\u6570\u636e\u3002\u4f60\u4e0b\u8f7d\u4f5c\u54c1\u6e90\u7801\u65f6\uff0c\u4e5f\u4f1a\u4e0b\u8f7d\u5305\u542b\u8fd9\u6bb5\u7edf\u8ba1\u4ee3\u7801\u7684\u7248\u672c\u3002",
  uploadType: "\u4e0a\u4f20\u65b9\u5f0f",
  zip: "ZIP \u538b\u7f29\u5305",
  html: "\u5355\u4e2a HTML \u6587\u4ef6",
  text: "\u76f4\u63a5\u7c98\u8d34 HTML \u4ee3\u7801",
  zipHint: "\u4e0a\u4f20\u5185\u5bb9\u4e0d\u80fd\u8d85\u8fc7 10MB\u3002ZIP \u53ea\u652f\u6301\u767d\u540d\u5355\u9759\u6001\u8d44\u6e90\u6587\u4ef6\u3002",
  htmlHint: "\u53ef\u4ee5\u76f4\u63a5\u4e0a\u4f20\u4e00\u4e2a .html\u3001.ht m \u6216 .txt \u6587\u4ef6\uff0c\u6587\u4ef6\u5185\u5bb9\u4e0d\u80fd\u8d85\u8fc7 10MB\u3002".replace(" ", ""),
  textHint:
    "\u9002\u5408\u76f4\u63a5\u628a AI \u751f\u6210\u7684 HTML \u6e90\u7801\u7c98\u8d34\u8fdb\u6765\uff0c\u5185\u5bb9\u4e0d\u80fd\u8d85\u8fc7 10MB\u3002\u8bf7\u786e\u8ba4\u4ece\u7b2c\u4e00\u884c\u5230\u6700\u540e\u4e00\u884c\u90fd\u5df2\u7ecf\u7c98\u8d34\u5b8c\u6574\u3002",
  file: "\u9009\u62e9\u6587\u4ef6",
  code: "HTML \u4ee3\u7801",
  codePlaceholder:
    "<!doctype html>\n<html lang=\"zh-CN\">\n<head>\n  <meta charset=\"utf-8\" />\n  <title>\u6211\u7684\u4f5c\u54c1</title>\n</head>\n<body>\n  <h1>\u4f60\u597d</h1>\n</body>\n</html>",
  start: "\u4e0a\u4f20\u4f5c\u54c1",
  updateStart: "\u4e0a\u4f20\u65b0\u7248\u672c",
  creating: "\u6b63\u5728\u521b\u5efa\u4f5c\u54c1\u5e76\u53d1\u5e03\u3002",
  uploading: "\u6b63\u5728\u4e0a\u4f20\u5185\u5bb9\u3002",
  progressTitle: "\u53d1\u5e03\u8fdb\u5ea6",
  progressDone: "\u53d1\u5e03\u5b8c\u6210",
  progressUnknown: "\u6b63\u5728\u51c6\u5907\u3002",
  success: "\u4f5c\u54c1\u5df2\u7ecf\u53d1\u5e03\u5b8c\u6210\uff0c\u6b63\u5728\u8fd4\u56de\u4f5c\u54c1\u5217\u8868\u3002",
  warningsTitle: "\u53d1\u5e03\u63d0\u9192\uff1a",
  fail: "\u521b\u5efa\u6216\u53d1\u5e03\u5931\u8d25\u3002",
  tooLarge: "\u4e0a\u4f20\u5185\u5bb9\u4e0d\u80fd\u8d85\u8fc7 10MB\u3002",
  fileMissing: "\u8bf7\u5148\u9009\u62e9\u4e0a\u4f20\u6587\u4ef6\u3002",
  codeMissing: "\u8bf7\u5148\u7c98\u8d34 HTML \u4ee3\u7801\u3002",
  requiredName: "\u8bf7\u5148\u586b\u5199\u4f5c\u54c1\u540d\u79f0\u3002",
  requiredSlug: "\u8bf7\u5148\u586b\u5199\u4f5c\u54c1\u94fe\u63a5\u540d\u3002",
  uploadedBytes: "\u5df2\u4e0a\u4f20\uff1a",
  uploadSpeed: "\u5f53\u524d\u901f\u5ea6\uff1a",
  unknownTotal: "\u672a\u77e5",
  calculating: "\u8ba1\u7b97\u4e2d"
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
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugTouched, setSlugTouched] = useState(false);
  const [interactive, setInteractive] = useState(false);
  const [analyticsEnabled, setAnalyticsEnabled] = useState(false);
  const [mode, setMode] = useState<UploadMode>("zip");
  const [file, setFile] = useState<File | null>(null);
  const [htmlText, setHtmlText] = useState("");
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
  }, []);

  useEffect(() => {
    if (slugTouched) {
      return;
    }
    setSlug(normalizePathText(name));
  }, [name, slugTouched]);

  const progressPercent = progressTotal > 0 ? Math.min(Math.round((progressLoaded / progressTotal) * 100), 100) : 0;

  const isUpdateMode = routeProjectId !== "";
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

    try {
      setWorking(true);
      setStep("publishing");
      setStatusText(isUpdateMode ? text.uploading : text.creating);
      setProgressLoaded(0);
      setProgressTotal(0);
      setProgressSpeed(0);
      setReleaseWarnings([]);

      let targetProjectId = routeProjectId;
      let targetInteractive = interactive;
      if (!isUpdateMode) {
        const project = await postJSON<CreateProjectResponse>("/api/v1/projects", {
          name,
          slug: normalizePathText(slug),
          interactive,
          analyticsEnabled
        });
        targetProjectId = project.id;
        targetInteractive = project.interactive;
        setStatusText(text.uploading);
      }

      if (mode === "zip" || mode === "html") {
        const form = new FormData();
        form.append("file", file as File);
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
      } else {
        const release = await postJSON<ReleaseResponse>(`/api/v1/projects/${targetProjectId}/releases?mode=text`, {
          html: htmlText
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
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
          <a className="button-secondary" href="/projects">
            {text.back}
          </a>
        </div>
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
          ) : (
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
          )}

          {statusText ? <div className="status" aria-live="polite">{statusText}</div> : null}

          <div>
            <button className="button-primary" type="button" disabled={working} onClick={submit}>
              {isUpdateMode ? text.updateStart : text.start}
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
