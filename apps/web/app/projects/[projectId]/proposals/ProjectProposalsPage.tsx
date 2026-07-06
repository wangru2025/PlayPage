"use client";

import { useEffect, useState } from "react";
import { getJSON } from "@/lib/api";

type Proposal = { id:string; sourceProjectId:string; sourceProjectName:string; sourceProjectUrl:string; targetProjectId:string; targetProjectName:string; authorUsername:string; title:string; body:string; status:string; createdAt:string; updatedAt:string; };
type Project = { id:string; name:string; publicUrl:string };
type ListResponse = { project: Project; items: Proposal[] };
type Tone = "info" | "success" | "error";
type FilterStatus = "" | "open" | "accepted" | "rejected" | "closed";

function statusText(status: string) {
  switch (status) {
    case "open": return "等待作者处理";
    case "accepted": return "已采纳";
    case "rejected": return "已拒绝";
    case "closed": return "已关闭";
    default: return status;
  }
}
function formatTime(value: string) { return value ? new Date(value).toLocaleString("zh-CN") : ""; }

export function ProjectProposalsPage({ projectId }: { projectId: string }) {
  const [project, setProject] = useState<Project | null>(null);
  const [items, setItems] = useState<Proposal[]>([]);
  const [filterStatus, setFilterStatus] = useState<FilterStatus>("");
  const [status, setStatus] = useState("正在读取改进提案。");
  const [tone, setTone] = useState<Tone>("info");
  const [loading, setLoading] = useState(true);

  useEffect(() => { void loadItems(""); }, [projectId]);

  async function loadItems(nextStatus: FilterStatus = filterStatus) {
    try {
      setLoading(true);
      const query = nextStatus ? `?status=${nextStatus}` : "";
      const data = await getJSON<ListResponse>(`/api/v1/projects/${projectId}/proposals${query}`);
      setProject(data.project);
      setItems(data.items);
      setFilterStatus(nextStatus);
      setStatus(data.items.length ? "改进提案已读取。" : "还没有符合条件的改进提案。");
      setTone("success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "读取改进提案失败。");
      setTone("error");
    } finally {
      setLoading(false);
    }
  }

  return <section className="panel" style={{ padding: 28, display: "grid", gap: 18 }}>
    <header style={{ display: "grid", gap: 8 }}>
      <p style={{ margin: 0, color: "var(--muted)" }}>PlayPage 改进提案</p>
      <h1 style={{ margin: 0, fontSize: "2rem" }}>{project ? `《${project.name}》的改进提案` : "改进提案"}</h1>
      <p style={{ margin: 0, color: "var(--muted)" }}>改编作者可以把自己的改进提交给原作者，原作者采纳后会发布为原作品的新版本。</p>
    </header>
    <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
      <button className={filterStatus === "" ? "button-primary" : "button-secondary"} type="button" onClick={() => void loadItems("")} disabled={loading}>全部</button>
      <button className={filterStatus === "open" ? "button-primary" : "button-secondary"} type="button" onClick={() => void loadItems("open")} disabled={loading}>等待处理</button>
      <button className={filterStatus === "accepted" ? "button-primary" : "button-secondary"} type="button" onClick={() => void loadItems("accepted")} disabled={loading}>已采纳</button>
      <button className={filterStatus === "rejected" ? "button-primary" : "button-secondary"} type="button" onClick={() => void loadItems("rejected")} disabled={loading}>已拒绝</button>
    </div>
    <div className="status" data-tone={tone === "info" ? undefined : tone} aria-live="polite">{loading ? "正在读取改进提案。" : status}</div>
    <section style={{ display: "grid", gap: 12 }} aria-label="提案列表">
      {items.map((item) => <article key={item.id} className="panel" style={{ padding: 18, display: "grid", gap: 8 }}>
        <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
          <a href={`/projects/${projectId}/proposals/${item.id}`} style={{ fontWeight: 800, fontSize: "1.1rem" }}>{item.title}</a>
          <span className="soft-badge">{statusText(item.status)}</span>
        </div>
        <p style={{ margin: 0, color: "var(--muted)" }}>来自 @{item.authorUsername || "匿名用户"} 的《{item.sourceProjectName || "改编作品"}》，{formatTime(item.createdAt)}</p>
      </article>)}
    </section>
  </section>;
}
