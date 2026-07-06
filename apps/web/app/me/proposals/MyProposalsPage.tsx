"use client";

import { useEffect, useState } from "react";
import { getJSON } from "@/lib/api";

type Proposal = { id:string; sourceProjectId:string; sourceProjectName:string; targetProjectId:string; targetProjectName:string; authorUsername:string; targetOwnerUsername:string; title:string; status:string; createdAt:string; };
type ListResponse = { items: Proposal[] };
type Tone = "info" | "success" | "error";

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

export function MyProposalsPage() {
  const [items, setItems] = useState<Proposal[]>([]);
  const [status, setStatus] = useState("正在读取我的提案。");
  const [tone, setTone] = useState<Tone>("info");
  const [loading, setLoading] = useState(true);

  useEffect(() => { void loadItems(); }, []);

  async function loadItems() {
    try {
      setLoading(true);
      const data = await getJSON<ListResponse>("/api/v1/me/proposals");
      setItems(data.items);
      setStatus(data.items.length ? "我的提案已读取。" : "还没有与你有关的提案。");
      setTone("success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "读取我的提案失败。");
      setTone("error");
    } finally {
      setLoading(false);
    }
  }

  return <section className="panel" style={{ padding: 28, display: "grid", gap: 18 }}>
    <header style={{ display: "grid", gap: 8 }}>
      <p style={{ margin: 0, color: "var(--muted)" }}>个人中心</p>
      <h1 style={{ margin: 0, fontSize: "2rem" }}>我的提案</h1>
      <p style={{ margin: 0, color: "var(--muted)" }}>这里包含你提交给别人的提案，以及别人提交给你作品的提案。</p>
    </header>
    <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}><button className="button-secondary" type="button" onClick={() => void loadItems()} disabled={loading}>刷新</button></div>
    <div className="status" data-tone={tone === "info" ? undefined : tone} aria-live="polite">{loading ? "正在读取我的提案。" : status}</div>
    <section style={{ display: "grid", gap: 12 }}>
      {items.map((item) => <article key={item.id} className="panel" style={{ padding: 18, display: "grid", gap: 8 }}>
        <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}><a href={`/projects/${item.targetProjectId}/proposals/${item.id}`} style={{ fontWeight: 800 }}>{item.title}</a><span className="soft-badge">{statusText(item.status)}</span></div>
        <p style={{ margin: 0, color: "var(--muted)" }}>《{item.sourceProjectName || "改编作品"}》 → 《{item.targetProjectName || "原作品"}》，提交人 @{item.authorUsername || "匿名用户"}，{formatTime(item.createdAt)}</p>
      </article>)}
    </section>
  </section>;
}
