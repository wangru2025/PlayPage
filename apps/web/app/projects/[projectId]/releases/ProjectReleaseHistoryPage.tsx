"use client";

import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";

type Project = {
  id: string;
  name: string;
  publicUrl: string;
  currentReleaseId?: string;
};

type Release = {
  id: string;
  projectId: string;
  status: string;
  archivePath: string;
  publicPath: string;
  entryFile: string;
  changeNote: string;
  createdAt: string;
};

type StatusTone = "info" | "success" | "error";

type Props = { projectId: string };

const text = {
  title: "作品历史版本",
  intro: "这里显示当前版本之前的所有发布版本。选择一个历史版本后，可以把作品回滚到当时的内容。",
  loading: "正在读取历史版本。",
  ready: "历史版本已读取。",
  empty: "还没有可回滚的历史版本。至少上传过两个版本后，这里才会出现旧版本。",
  rollback: "回滚到这个版本",
  rollbackAsk: "确认要回滚到这个历史版本吗？当前版本不会删除，只是把线上作品切换到旧版本。",
  rollbackWorking: "正在回滚版本。",
  rollbackOk: "已经回滚到所选历史版本。",
  rollbackFail: "回滚失败。",
  current: "当前版本",
  noteEmpty: "没有填写更新内容",
  openProject: "打开作品"
};

function formatDate(value: string): string {
  if (!value) return "未知时间";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", { hour12: false });
}

export function ProjectReleaseHistoryPage({ projectId }: Props) {
  const [project, setProject] = useState<Project | null>(null);
  const [releases, setReleases] = useState<Release[]>([]);
  const [statusText, setStatusText] = useState(text.loading);
  const [statusTone, setStatusTone] = useState<StatusTone>("info");
  const [workingId, setWorkingId] = useState("");

  useEffect(() => {
    void loadData();
  }, [projectId]);

  async function loadData() {
    try {
      const [projectData, releaseData] = await Promise.all([
        getJSON<{ project: Project }>(`/api/v1/projects/${projectId}`),
        getJSON<{ items: Release[] }>(`/api/v1/projects/${projectId}/releases`)
      ]);
      setProject(projectData.project);
      setReleases(releaseData.items);
      setStatusText(text.ready);
      setStatusTone("success");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : "读取历史版本失败。");
      setStatusTone("error");
    }
  }

  async function rollback(release: Release) {
    if (workingId !== "") return;
    if (!window.confirm(text.rollbackAsk)) return;
    try {
      setWorkingId(release.id);
      setStatusText(text.rollbackWorking);
      setStatusTone("info");
      await postJSON(`/api/v1/projects/${projectId}/releases/${release.id}/rollback`, {});
      await loadData();
      setStatusText(text.rollbackOk);
      setStatusTone("success");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : text.rollbackFail);
      setStatusTone("error");
    } finally {
      setWorkingId("");
    }
  }

  const history = [...releases]
    .filter((release) => release.id !== project?.currentReleaseId)
    .sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime());

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <h1 style={{ margin: 0, fontSize: "2.4rem" }}>{text.title}</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 820 }}>{text.intro}</p>
        {project ? (
          <div style={{ display: "flex", gap: 10, flexWrap: "wrap", alignItems: "center" }}>
            <span className="soft-badge">{project.name}</span>
            <a className="button-secondary" href={project.publicUrl} target="_blank" rel="noreferrer">{text.openProject}</a>
          </div>
        ) : null}
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite" role="status">
          {statusText}
        </div>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 14 }}>
        {history.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.empty}</p> : null}
        {history.map((release, index) => (
          <article key={release.id} style={{ display: "grid", gap: 10, padding: 18, border: "1px solid var(--line)", borderRadius: 18, background: "rgba(255,255,255,0.72)" }}>
            <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap", alignItems: "start" }}>
              <div style={{ display: "grid", gap: 6 }}>
                <h2 style={{ margin: 0, fontSize: "1.2rem" }}>第 {index + 1} 次发布</h2>
                <p style={{ margin: 0, color: "var(--muted)" }}>{formatDate(release.createdAt)}</p>
                <p style={{ margin: 0, whiteSpace: "pre-wrap" }}>{release.changeNote?.trim() || text.noteEmpty}</p>
              </div>
              <button className="button-primary" type="button" disabled={workingId !== ""} aria-busy={workingId === release.id} onClick={() => rollback(release)}>
                {workingId === release.id ? text.rollbackWorking : text.rollback}
              </button>
            </div>
          </article>
        ))}
      </section>
    </section>
  );
}
