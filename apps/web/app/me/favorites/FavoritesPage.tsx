"use client";

import { useEffect, useState } from "react";
import { getJSON } from "@/lib/api";

type Project = {
  id: string;
  username: string;
  name: string;
  publicUrl: string;
  allowForks: boolean;
  favoritesCount: number;
  forksCount: number;
  forkedFromProjectId?: string;
  forkedFromUsername?: string;
  forkedFromProjectName?: string;
  forkedFromProjectUrl?: string;
};

export function FavoritesPage() {
  const [items, setItems] = useState<Project[]>([]);
  const [status, setStatus] = useState("正在读取收藏作品。");
  const [loading, setLoading] = useState(true);

  useEffect(() => { void load(); }, []);

  async function load() {
    try {
      setLoading(true);
      const data = await getJSON<{ items: Project[] }>("/api/v1/me/favorites");
      setItems(data.items);
      setStatus(data.items.length ? "收藏作品已读取。" : "你还没有收藏作品。");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "读取收藏作品失败，请先登录。");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <section className="panel" style={{ padding: 28, display: "grid", gap: 10 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>个人中心</p>
        <h1 style={{ margin: 0, fontSize: "2.2rem" }}>我的收藏</h1>
        <div className="status" aria-live="polite">{loading ? "正在读取收藏作品。" : status}</div>
      </section>

      {items.length ? (
        <section className="card-grid" aria-label="收藏作品列表">
          {items.map((project) => (
            <article key={project.id} className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
              <h2 style={{ margin: 0 }}>{project.name}</h2>
              <p style={{ margin: 0, color: "var(--muted)" }}>作者：<a href={`/@${encodeURIComponent(project.username)}`}>@{project.username}</a></p>
              {project.forkedFromProjectId ? <p style={{ margin: 0, color: "var(--muted)" }}>改编自 {project.forkedFromUsername || "原作者"} 的《{project.forkedFromProjectName || "原作品"}》</p> : null}
              <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                <span className="soft-badge">收藏 {project.favoritesCount}</span>
                <span className="soft-badge">被改编 {project.forksCount}</span>
              </div>
              <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
                <a className="button-primary" href={project.publicUrl} target="_blank" rel="noreferrer">打开作品</a>
                {project.allowForks ? <a className="button-secondary" href={`/projects/fork?projectId=${project.id}`}>改编这个作品</a> : null}
              </div>
            </article>
          ))}
        </section>
      ) : null}
    </section>
  );
}
