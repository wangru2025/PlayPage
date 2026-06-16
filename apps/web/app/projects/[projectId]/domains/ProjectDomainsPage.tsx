"use client";

import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";

type Project = { id: string; name: string; publicUrl: string };
type ProjectResponse = { project: Project };
type ProjectDomain = {
  id: string;
  projectId: string;
  subdomain: string;
  domain: string;
  status: string;
  rejectReason: string;
  adminNote: string;
  createdAt: string;
};
type StatusTone = "info" | "success" | "error";

type Props = { projectId: string };

function domainStatusLabel(status: string): string {
  if (status === "active") return "已通过";
  if (status === "rejected") return "已驳回";
  return "审核中";
}

export function ProjectDomainsPage({ projectId }: Props) {
  const [project, setProject] = useState<Project | null>(null);
  const [items, setItems] = useState<ProjectDomain[]>([]);
  const [subdomain, setSubdomain] = useState("");
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);
  const [statusText, setStatusText] = useState("正在读取独立网址申请。");
  const [statusTone, setStatusTone] = useState<StatusTone>("info");

  useEffect(() => {
    void loadData();
  }, [projectId]);

  async function loadData() {
    setLoading(true);
    try {
      const [projectData, domainData] = await Promise.all([
        getJSON<ProjectResponse>(`/api/v1/projects/${projectId}`),
        getJSON<{ items: ProjectDomain[] }>(`/api/v1/projects/${projectId}/domains`)
      ]);
      setProject(projectData.project);
      setItems(domainData.items);
      setStatusText("独立网址申请已读取。");
      setStatusTone("success");
    } catch (error) {
      setProject(null);
      setItems([]);
      setStatusText(error instanceof Error ? error.message : "读取独立网址申请失败。");
      setStatusTone("error");
    } finally {
      setLoading(false);
    }
  }

  async function submitDomain() {
    if (working) return;
    try {
      setWorking(true);
      setStatusText("正在提交独立网址申请。");
      setStatusTone("info");
      await postJSON<ProjectDomain>(`/api/v1/projects/${projectId}/domains`, { subdomain: subdomain.trim().toLowerCase() });
      setSubdomain("");
      await loadData();
      setStatusText("独立网址申请已提交，请等待管理员审核。");
      setStatusTone("success");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : "提交独立网址申请失败。");
      setStatusTone("error");
    } finally {
      setWorking(false);
    }
  }

  const hasPendingOrActive = items.some((item) => item.status === "pending" || item.status === "active");

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <nav aria-label="独立网址页面导航" style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <a className="button-secondary" href="/projects">回到我的作品</a>
        </nav>
        <h1 style={{ margin: 0, fontSize: "2.4rem" }}>申请独立网址</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 760 }}>
          你可以为作品申请一个类似 https://my-game.wangru.net 的短网址。申请后需要管理员人工审核。
        </p>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite" role="status">
          {loading ? "正在读取独立网址申请。" : statusText}
        </div>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        {project ? (
          <div style={{ display: "grid", gap: 6 }}>
            <h2 style={{ margin: 0 }}>作品：{project.name}</h2>
            <p style={{ margin: 0, color: "var(--muted)" }}>作品地址：{project.publicUrl}</p>
          </div>
        ) : null}

        <div style={{ display: "grid", gap: 10 }}>
          <h3 style={{ margin: 0 }}>已有申请</h3>
          {items.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>还没有独立网址申请。</p> : null}
          {items.map((item) => (
            <div key={item.id} className="soft-badge" style={{ justifySelf: "start" }}>
              {domainStatusLabel(item.status)}：{item.domain}
              {item.rejectReason ? `，原因：${item.rejectReason}` : ""}
            </div>
          ))}
        </div>

        {!hasPendingOrActive ? (
          <div style={{ display: "grid", gap: 10 }}>
            <label className="field">
              <span>想要的子域名</span>
              <input value={subdomain} onChange={(event) => setSubdomain(event.target.value)} placeholder="例如 my-game" />
            </label>
            <p style={{ margin: 0, color: "var(--muted)" }}>最终网址会是：{subdomain.trim() ? subdomain.trim().toLowerCase() : "my-game"}.wangru.net</p>
            <div>
              <button className="button-primary" type="button" onClick={submitDomain} disabled={working} aria-busy={working}>
                {working ? "正在提交" : "提交独立网址申请"}
              </button>
            </div>
          </div>
        ) : (
          <p style={{ margin: 0, color: "var(--muted)" }}>这个作品已经有审核中或已通过的独立网址申请，不能重复提交。</p>
        )}
      </section>
    </section>
  );
}
