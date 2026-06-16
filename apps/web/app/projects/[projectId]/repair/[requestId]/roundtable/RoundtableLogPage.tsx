"use client";

import { useEffect, useState } from "react";
import { getJSON } from "@/lib/api";
import { aiStatusLabel, type RepairAIJob, type RepairAIMessage, type StatusTone } from "../../repairTypes";

type Props = { projectId: string; requestId: string };

export function RoundtableLogPage({ projectId, requestId }: Props) {
  const [job, setJob] = useState<RepairAIJob | null>(null);
  const [messages, setMessages] = useState<RepairAIMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusText, setStatusText] = useState("正在读取圆桌聊天记录。");
  const [statusTone, setStatusTone] = useState<StatusTone>("info");

  useEffect(() => {
    void loadLog();
  }, [projectId, requestId]);

  async function loadLog() {
    setLoading(true);
    try {
      const data = await getJSON<{ job: RepairAIJob; messages: RepairAIMessage[] }>(`/api/v1/projects/${projectId}/repair-requests/${requestId}/ai/latest`);
      setJob(data.job);
      setMessages(data.messages ?? []);
      setStatusText("圆桌聊天记录已读取。");
      setStatusTone("success");
    } catch (error) {
      setJob(null);
      setMessages([]);
      setStatusText(error instanceof Error ? error.message : "还没有圆桌聊天记录。");
      setStatusTone("error");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <nav aria-label="圆桌聊天记录导航" style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <a className="button-secondary" href={`/projects/${projectId}/repair/requests`}>返回修复申请</a>
          <a className="button-secondary" href={`/projects/${projectId}/repair/${requestId}/ai`}>进入 AI 圆桌页面</a>
        </nav>
        <h1 style={{ margin: 0, fontSize: "2.4rem" }}>圆桌聊天记录</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>这里只展示 AI 们的聊天过程，不在这里发布或重新修复。</p>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite" role="status">
          {loading ? "正在读取圆桌聊天记录。" : statusText}
        </div>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 14 }}>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap", alignItems: "center", justifyContent: "space-between" }}>
          <p style={{ margin: 0, color: "var(--muted)" }}>AI 状态：{aiStatusLabel(job?.status ?? "")}{job ? `；第 ${job.round} 轮` : ""}</p>
          <button className="button-secondary" type="button" onClick={loadLog}>刷新记录</button>
        </div>
        <div aria-live="polite" aria-label="AI 圆桌聊天记录" style={{ display: "grid", gap: 10 }}>
          {messages.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>还没有圆桌消息。</p> : null}
          {messages.map((message) => (
            <article key={message.id} style={{ border: "1px solid var(--line)", borderRadius: 14, padding: 12, background: "rgba(255,255,255,0.72)" }}>
              <strong>{message.messageSeq}. {message.agentName}</strong>
              <p style={{ margin: "6px 0 0", whiteSpace: "pre-wrap" }}>{message.content}</p>
            </article>
          ))}
        </div>
      </section>
    </section>
  );
}
