"use client";

import { FormEvent, useEffect, useState } from "react";
import { getJSON } from "@/lib/api";

type Discussion = {
  id: string;
  projectId: string;
  projectName: string;
  projectUrl: string;
  authorUsername: string;
  title: string;
  body: string;
  status: string;
  commentsCount: number;
  lastCommentedAt: string;
  createdAt: string;
};

type ListResponse = { items: Discussion[] };
type Tone = "info" | "success" | "error";
type FilterStatus = "" | "open" | "closed";

function formatTime(value: string) {
  if (!value) return "";
  return new Date(value).toLocaleString("zh-CN");
}

function statusText(value: string) {
  return value === "closed" ? "已关闭" : "进行中";
}

export function ProjectDiscussionsPage({ projectId }: { projectId: string }) {
  const [items, setItems] = useState<Discussion[]>([]);
  const [filterStatus, setFilterStatus] = useState<FilterStatus>("");
  const [keyword, setKeyword] = useState("");
  const [appliedKeyword, setAppliedKeyword] = useState("");
  const [status, setStatus] = useState("正在读取讨论区。");
  const [tone, setTone] = useState<Tone>("info");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const initialStatus = params.get("status") ?? "";
    const initialKeyword = params.get("q") ?? "";
    const safeStatus: FilterStatus = initialStatus === "open" || initialStatus === "closed" ? initialStatus : "";
    setFilterStatus(safeStatus);
    setKeyword(initialKeyword);
    setAppliedKeyword(initialKeyword);
    void loadItems(safeStatus, initialKeyword, false);
  }, [projectId]);

  function syncURL(nextStatus: FilterStatus, nextKeyword: string) {
    const params = new URLSearchParams();
    if (nextStatus) params.set("status", nextStatus);
    if (nextKeyword.trim()) params.set("q", nextKeyword.trim());
    const query = params.toString();
    window.history.replaceState(null, "", query ? `?${query}` : window.location.pathname);
  }

  async function loadItems(nextStatus = filterStatus, nextKeyword = appliedKeyword, updateURL = true) {
    try {
      setLoading(true);
      const params = new URLSearchParams();
      if (nextStatus) params.set("status", nextStatus);
      if (nextKeyword.trim()) params.set("q", nextKeyword.trim());
      if (updateURL) syncURL(nextStatus, nextKeyword);
      const query = params.toString();
      const data = await getJSON<ListResponse>(`/api/v1/projects/${projectId}/discussions${query ? `?${query}` : ""}`);
      setItems(data.items);
      setStatus(data.items.length === 0 ? "没有找到符合条件的讨论。" : "讨论区已读取。");
      setTone("success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "读取讨论区失败。");
      setTone("error");
    } finally {
      setLoading(false);
    }
  }

  function applyStatus(nextStatus: FilterStatus) {
    setFilterStatus(nextStatus);
    void loadItems(nextStatus, appliedKeyword);
  }

  function search(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextKeyword = keyword.trim();
    setAppliedKeyword(nextKeyword);
    void loadItems(filterStatus, nextKeyword);
  }

  function clearSearch() {
    setKeyword("");
    setAppliedKeyword("");
    void loadItems(filterStatus, "");
  }

  return (
    <section className="panel" style={{ padding: 28, display: "grid", gap: 18 }}>
      <header style={{ display: "grid", gap: 8 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>PlayPage 作品讨论区</p>
        <h1 style={{ margin: 0, fontSize: "2rem" }}>讨论区</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 760 }}>这里用于收集建议、反馈问题和交流创作想法。</p>
      </header>

      <div style={{ display: "flex", gap: 12, flexWrap: "wrap", alignItems: "center" }}>
        <a className="button-primary" href={`/projects/${projectId}/discussions/new`}>发起讨论</a>
        <button className="button-secondary" type="button" onClick={() => void loadItems()} disabled={loading}>刷新列表</button>
      </div>

      <form onSubmit={search} className="panel" style={{ padding: 16, display: "grid", gap: 12 }} aria-label="筛选讨论">
        <label className="field">
          <span>搜索讨论</span>
          <input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索标题、内容或发起人" maxLength={100} />
        </label>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap", alignItems: "center" }}>
          <button className="button-secondary" type="submit" disabled={loading}>搜索</button>
          {appliedKeyword ? <button className="button-ghost" type="button" onClick={clearSearch} disabled={loading}>清除搜索</button> : null}
          <span className="field-note">状态筛选：</span>
          <button className={filterStatus === "" ? "button-primary" : "button-secondary"} type="button" onClick={() => applyStatus("")} disabled={loading}>全部</button>
          <button className={filterStatus === "open" ? "button-primary" : "button-secondary"} type="button" onClick={() => applyStatus("open")} disabled={loading}>进行中</button>
          <button className={filterStatus === "closed" ? "button-primary" : "button-secondary"} type="button" onClick={() => applyStatus("closed")} disabled={loading}>已关闭</button>
        </div>
      </form>

      <div className="status" data-tone={tone === "info" ? undefined : tone} aria-live="polite">{loading ? "正在读取讨论区。" : status}</div>

      <section style={{ display: "grid", gap: 12 }} aria-label="讨论列表">
        {items.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>还没有符合条件的讨论。</p> : null}
        {items.map((item) => (
          <article key={item.id} className="panel" style={{ padding: 18, display: "grid", gap: 8 }}>
            <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
              <a href={`/projects/${projectId}/discussions/${item.id}`} style={{ fontWeight: 800, fontSize: "1.1rem" }}>{item.title}</a>
              <span className="soft-badge">{statusText(item.status)}</span>
            </div>
            <p style={{ margin: 0, color: "var(--muted)" }}>由 {item.authorUsername || "匿名用户"} 发起，{formatTime(item.createdAt)}，回复 {item.commentsCount}，最近活动 {formatTime(item.lastCommentedAt)}</p>
          </article>
        ))}
      </section>
    </section>
  );
}
