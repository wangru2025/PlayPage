"use client";

import { FormEvent, useEffect, useId, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";

type Project = {
  id: string;
  username: string;
  slug: string;
  name: string;
  publicUrl: string;
  interactive: boolean;
  analyticsEnabled: boolean;
  showOnProfile: boolean;
  allowForks: boolean;
};

type ProjectResponse = { project: Project };
type ForkResponse = { project: Project };

type Tone = "info" | "success" | "error";

function normalizePathText(value: string): string {
  let normalized = value.trim().toLowerCase().replace(/\s+/g, "-");
  normalized = Array.from(normalized).map((char) => (/[-_.]/u.test(char) || /[\p{L}\p{N}]/u.test(char) ? char : "-")).join("");
  normalized = normalized.replace(/^-+|[-._]+$/g, "");
  while (normalized.includes("--")) normalized = normalized.replaceAll("--", "-");
  return normalized;
}

export function ProjectForkPage() {
  const nameId = useId();
  const slugId = useId();
  const interactiveId = useId();
  const analyticsId = useId();
  const showOnProfileId = useId();
  const allowForksId = useId();
  const [sourceId, setSourceId] = useState("");
  const [source, setSource] = useState<Project | null>(null);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugTouched, setSlugTouched] = useState(false);
  const [interactive, setInteractive] = useState(false);
  const [analyticsEnabled, setAnalyticsEnabled] = useState(false);
  const [showOnProfile, setShowOnProfile] = useState(true);
  const [allowForks, setAllowForks] = useState(true);
  const [status, setStatus] = useState("正在读取原作品。");
  const [tone, setTone] = useState<Tone>("info");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const id = params.get("projectId") ?? "";
    setSourceId(id);
    if (!id) {
      setStatus("缺少原作品编号，无法改编。");
      setTone("error");
      setLoading(false);
      return;
    }
    void loadSource(id);
  }, []);

  useEffect(() => {
    if (!slugTouched) setSlug(normalizePathText(name));
  }, [name, slugTouched]);

  async function loadSource(id: string) {
    try {
      setLoading(true);
      const data = await getJSON<ProjectResponse>(`/api/v1/public/projects/${encodeURIComponent(id)}/profile`);
      setSource(data.project);
      setName(`${data.project.name} 的改编版`);
      setSlug(normalizePathText(`${data.project.slug}-remix`));
      setInteractive(data.project.interactive);
      setAnalyticsEnabled(data.project.analyticsEnabled);
      if (!data.project.allowForks) {
        setStatus("这个作品的作者没有开放改编。");
        setTone("error");
      } else {
        setStatus("可以填写改编后的作品信息。");
        setTone("success");
      }
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "读取原作品失败。");
      setTone("error");
    } finally {
      setLoading(false);
    }
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting || !source || !source.allowForks) return;
    if (name.trim() === "" || slug.trim() === "") {
      setStatus("请填写改编后的作品名称和链接名。");
      setTone("error");
      return;
    }
    try {
      setSubmitting(true);
      setStatus("正在创建改编作品。");
      setTone("info");
      const data = await postJSON<ForkResponse>(`/api/v1/projects/${encodeURIComponent(sourceId)}/fork`, {
        name,
        slug: normalizePathText(slug),
        interactive,
        analyticsEnabled,
        showOnProfile,
        allowForks
      });
      setStatus("改编作品已经创建，正在打开作品设置页。");
      setTone("success");
      window.location.href = `/projects/${data.project.id}/settings`;
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "创建改编作品失败。");
      setTone("error");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px", display: "grid", gap: 18 }}>
      <section className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>PlayPage 改编</p>
        <h1 style={{ margin: 0, fontSize: "2.3rem" }}>改编作品</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 760 }}>
          改编会把原作品当前版本复制到你的作品里继续创作。原作品不会被修改，互动数据不会复制，发布后的作品会显示来源。
        </p>
      </section>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        <div className="status" data-tone={tone === "info" ? undefined : tone} aria-live="polite">{status}</div>
        {source ? (
          <div style={{ display: "grid", gap: 6 }}>
            <strong>原作品：{source.username} 的《{source.name}》</strong>
            <a href={source.publicUrl} target="_blank" rel="noreferrer">打开原作品</a>
          </div>
        ) : null}

        <form onSubmit={submit} style={{ display: "grid", gap: 16, maxWidth: 760 }}>
          <label className="field" htmlFor={nameId}>
            <span>改编后的作品名称</span>
            <input id={nameId} value={name} onChange={(event) => setName(event.target.value)} disabled={loading || submitting || !source?.allowForks} required />
          </label>
          <label className="field" htmlFor={slugId}>
            <span>改编后的链接名</span>
            <input id={slugId} value={slug} onChange={(event) => { setSlugTouched(true); setSlug(event.target.value); }} disabled={loading || submitting || !source?.allowForks} required />
            <span className="field-note">保存后会生成你的独立作品地址。</span>
          </label>
          <label style={{ display: "flex", gap: 10, alignItems: "start" }} htmlFor={interactiveId}>
            <input id={interactiveId} type="checkbox" checked={interactive} onChange={(event) => setInteractive(event.target.checked)} disabled={loading || submitting || !source?.allowForks} style={{ width: "auto", marginTop: 4 }} />
            <span>启用互动功能</span>
          </label>
          <label style={{ display: "flex", gap: 10, alignItems: "start" }} htmlFor={analyticsId}>
            <input id={analyticsId} type="checkbox" checked={analyticsEnabled} onChange={(event) => setAnalyticsEnabled(event.target.checked)} disabled={loading || submitting || !source?.allowForks} style={{ width: "auto", marginTop: 4 }} />
            <span>启用访问量统计</span>
          </label>
          <label style={{ display: "flex", gap: 10, alignItems: "start" }} htmlFor={showOnProfileId}>
            <input id={showOnProfileId} type="checkbox" checked={showOnProfile} onChange={(event) => setShowOnProfile(event.target.checked)} disabled={loading || submitting || !source?.allowForks} style={{ width: "auto", marginTop: 4 }} />
            <span style={{ display: "grid", gap: 4 }}>
              <span>在作者主页展示这个作品</span>
              <span className="field-note">关闭后，别人打开你的作者主页不会看到它，但作品直链不受影响。</span>
            </span>
          </label>

          <label style={{ display: "flex", gap: 10, alignItems: "start" }} htmlFor={allowForksId}>
            <input id={allowForksId} type="checkbox" checked={allowForks} onChange={(event) => setAllowForks(event.target.checked)} disabled={loading || submitting || !source?.allowForks} style={{ width: "auto", marginTop: 4 }} />
            <span style={{ display: "grid", gap: 4 }}>
              <span>允许别人继续改编我的改编作品</span>
              <span className="field-note">如果关闭，别人不能从你的这个改编版继续复制创作。</span>
            </span>
          </label>
          <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
            <button className="button-primary" type="submit" disabled={loading || submitting || !source?.allowForks} aria-busy={submitting}>{submitting ? "正在创建" : "创建改编作品"}</button>
          </div>
        </form>
      </section>
    </main>
  );
}
