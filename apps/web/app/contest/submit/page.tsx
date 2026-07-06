"use client";

import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";

type Project = {
  id: string;
  name: string;
  publicUrl: string;
  currentReleaseId?: string;
};

type ProjectResponse = {
  project: Project;
};

type SubmissionStatusResponse = {
  submitted: boolean;
  submission?: { id: string; track: string; status: string };
};

export default function ContestSubmitPage() {
  const [projectId, setProjectId] = useState("");
  const [project, setProject] = useState<Project | null>(null);
  const [intro, setIntro] = useState("");
  const [story, setStory] = useState("");
  const [allowShowcase, setAllowShowcase] = useState(true);
  const [statusText, setStatusText] = useState("正在准备提交页面。");
  const [statusTone, setStatusTone] = useState<"info" | "success" | "error">("info");
  const [working, setWorking] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const id = params.get("projectId") ?? "";
    setProjectId(id);
    if (!id) {
      setStatus("请先从某个作品的更多操作里进入参赛提交页面。", "error");
      return;
    }
    void loadProject(id);
  }, []);

  function setStatus(message: string, tone: "info" | "success" | "error") {
    setStatusText(message);
    setStatusTone(tone);
  }

  async function loadProject(id: string) {
    try {
      const payload = await getJSON<ProjectResponse>(`/api/v1/projects/${id}`);
      setProject(payload.project);
      const state = await getJSON<SubmissionStatusResponse>(`/api/v1/projects/${id}/contest-submission`);
      if (state.submitted) {
        setSubmitted(true);
        setStatus("这个作品已经提交过比赛。", "success");
      } else {
        setStatus("请填写参赛信息。", "info");
      }
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "读取作品失败。", "error");
    }
  }

  async function submit() {
    if (!projectId || !project) return;
    if (!intro.trim()) {
      setStatus("请填写作品介绍。", "error");
      return;
    }
    try {
      setWorking(true);
      setStatus("正在提交参赛作品。", "info");
      await postJSON(`/api/v1/projects/${projectId}/contest-submission`, {
        intro,
        story,
        allowShowcase
      });
      setSubmitted(true);
      setStatus("提交成功。你的作品已经参加 PlayPage 作品创作比赛。", "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "提交失败。", "error");
    } finally {
      setWorking(false);
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>PlayPage 作品创作比赛</p>
        <h1 style={{ margin: 0, fontSize: "2.3rem" }}>提交参赛作品</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>
          从你的作品里选择一个参加比赛。请简单说明作品做什么，以及你为什么做它。
        </p>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite">
          {statusText}
        </div>
      </header>

      {project ? (
        <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
          <div style={{ display: "grid", gap: 6 }}>
            <h2 style={{ margin: 0 }}>{project.name}</h2>
            <a href={project.publicUrl} target="_blank" rel="noreferrer">{project.publicUrl}</a>
            {!project.currentReleaseId ? <p style={{ margin: 0, color: "var(--muted)" }}>这个作品还没有上传网页，不能参赛。</p> : null}
          </div>

          <p className="field-note" style={{ margin: 0 }}>
            不需要选择赛道。活动结束后，管理员会根据作品特点统一评选“最有创意、最好玩、最实用、最佳互动、新人潜力”等奖项。
          </p>

          <label className="field">
            <span>作品介绍 *</span>
            <textarea rows={5} value={intro} maxLength={600} onChange={(event) => setIntro(event.target.value)} disabled={submitted || working} placeholder="这个作品是做什么的？怎么玩？有什么亮点？" />
          </label>

          <label className="field">
            <span>创作故事，可选</span>
            <textarea rows={8} value={story} maxLength={100000} onChange={(event) => setStory(event.target.value)} disabled={submitted || working} placeholder="你为什么想做这个作品？用了 AI 吗？有没有遇到什么问题？可以写长一点。" />
            <span className="field-note">最多 100000 字，正常创作故事基本不会写满。</span>
          </label>

          <label>
            <input type="checkbox" checked={allowShowcase} onChange={(event) => setAllowShowcase(event.target.checked)} disabled={submitted || working} />
            {" "}允许 PlayPage 在活动页、广场精选或文章中展示这个作品
          </label>

          <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
            <button className="button-primary" type="button" onClick={submit} disabled={submitted || working || !project.currentReleaseId}>
              {submitted ? "已提交" : "提交参赛"}
            </button>
            <a className="button-secondary" href="/contest">查看活动说明</a>
          </div>
        </section>
      ) : null}
    </section>
  );
}
