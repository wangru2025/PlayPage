"use client";

import { FormEvent, useEffect, useId, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";

type Project = {
  id: string;
  slug: string;
  name: string;
  interactive: boolean;
  analyticsEnabled: boolean;
  publicUrl: string;
};

type ProjectResponse = {
  project: Project;
};

type StatusTone = "info" | "success" | "error";

const text = {
  loading: "正在读取作品设置。",
  ready: "可以修改作品设置了。",
  loadFail: "读取作品设置失败。",
  saving: "正在保存作品设置。",
  saved: "作品设置已经保存。",
  saveFail: "保存作品设置失败。",
  title: "作品设置",
  summary: "这里可以修改作品名字、链接名，以及互动功能和访问量统计开关。",
  name: "作品名字",
  slug: "作品链接名",
  slugHint: "只能使用中文、英文、数字、短横线等适合作为网址路径的字符。保存后作品地址会变化。",
  interactive: "启用互动功能",
  interactiveHint: "关闭后，公开互动 API 会停止读写这个作品的数据表；已有数据不会删除。",
  analytics: "启用访问量统计",
  analyticsHint: "开启后，互动 API 请求统计会立即生效；页面访问量统计代码会在下次上传或发布作品时加入 HTML。",
  publicUrl: "当前作品地址",
  save: "保存设置",
  openProject: "打开作品",
  nameRequired: "请填写作品名字。",
  slugRequired: "请填写作品链接名。"
};

export function ProjectSettingsPage({ projectId }: { projectId: string }) {
  const nameId = useId();
  const slugId = useId();
  const interactiveId = useId();
  const analyticsId = useId();
  const [project, setProject] = useState<Project | null>(null);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [interactive, setInteractive] = useState(false);
  const [analyticsEnabled, setAnalyticsEnabled] = useState(false);
  const [statusText, setStatusText] = useState(text.loading);
  const [statusTone, setStatusTone] = useState<StatusTone>("info");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    void loadProject();
  }, [projectId]);

  async function loadProject() {
    try {
      setLoading(true);
      const data = await getJSON<ProjectResponse>(`/api/v1/projects/${projectId}`);
      setProject(data.project);
      setName(data.project.name);
      setSlug(data.project.slug);
      setInteractive(data.project.interactive);
      setAnalyticsEnabled(data.project.analyticsEnabled);
      setStatus(text.ready, "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : text.loadFail, "error");
    } finally {
      setLoading(false);
    }
  }

  function setStatus(message: string, tone: StatusTone) {
    setStatusText(message);
    setStatusTone(tone);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (saving) {
      return;
    }
    if (name.trim() === "") {
      setStatus(text.nameRequired, "error");
      return;
    }
    if (slug.trim() === "") {
      setStatus(text.slugRequired, "error");
      return;
    }

    try {
      setSaving(true);
      setStatus(text.saving, "info");
      const updated = await postJSON<Project>(`/api/v1/projects/${projectId}/settings`, {
        name,
        slug,
        interactive,
        analyticsEnabled
      });
      setProject(updated);
      setName(updated.name);
      setSlug(updated.slug);
      setInteractive(updated.interactive);
      setAnalyticsEnabled(updated.analyticsEnabled);
      setStatus(text.saved, "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : text.saveFail, "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="panel" style={{ padding: 28, display: "grid", gap: 18 }}>
      <header style={{ display: "grid", gap: 8 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>PlayPage</p>
        <h1 style={{ margin: 0, fontSize: "2rem" }}>{text.title}</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 760 }}>{text.summary}</p>
      </header>

      <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite">
        {loading ? text.loading : statusText}
      </div>

      <form onSubmit={submit} style={{ display: "grid", gap: 16, maxWidth: 760 }}>
        <label style={{ display: "grid", gap: 8 }} htmlFor={nameId}>
          <span>{text.name}</span>
          <input id={nameId} value={name} onChange={(event) => setName(event.target.value)} disabled={loading || saving} required />
        </label>

        <label style={{ display: "grid", gap: 8 }} htmlFor={slugId}>
          <span>{text.slug}</span>
          <input id={slugId} value={slug} onChange={(event) => setSlug(event.target.value)} disabled={loading || saving} required />
          <span className="field-note">{text.slugHint}</span>
        </label>

        <label style={{ display: "flex", gap: 10, alignItems: "start" }} htmlFor={interactiveId}>
          <input id={interactiveId} type="checkbox" checked={interactive} onChange={(event) => setInteractive(event.target.checked)} disabled={loading || saving} style={{ width: "auto", marginTop: 4 }} />
          <span style={{ display: "grid", gap: 4 }}>
            <span>{text.interactive}</span>
            <span className="field-note">{text.interactiveHint}</span>
          </span>
        </label>

        <label style={{ display: "flex", gap: 10, alignItems: "start" }} htmlFor={analyticsId}>
          <input id={analyticsId} type="checkbox" checked={analyticsEnabled} onChange={(event) => setAnalyticsEnabled(event.target.checked)} disabled={loading || saving} style={{ width: "auto", marginTop: 4 }} />
          <span style={{ display: "grid", gap: 4 }}>
            <span>{text.analytics}</span>
            <span className="field-note">{text.analyticsHint}</span>
          </span>
        </label>

        {project ? (
          <p style={{ margin: 0, color: "var(--muted)" }}>
            {text.publicUrl}：{project.publicUrl}
          </p>
        ) : null}

        <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
          <button className="button-primary" type="submit" disabled={loading || saving} aria-busy={saving}>
            {saving ? text.saving : text.save}
          </button>
          {project ? (
            <a className="button-secondary" href={project.publicUrl} target="_blank" rel="noreferrer">
              {text.openProject}
            </a>
          ) : null}
        </div>
      </form>
    </section>
  );
}
