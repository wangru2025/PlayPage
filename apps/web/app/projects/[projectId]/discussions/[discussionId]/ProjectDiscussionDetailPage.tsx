"use client";

import { FormEvent, useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";

type Discussion = { id: string; projectId: string; projectName: string; projectUrl: string; authorUsername: string; title: string; body: string; status: string; commentsCount: number; closedByUsername?: string; createdAt: string; };
type Comment = { id: string; authorUsername: string; body: string; createdAt: string; };
type DetailResponse = { discussion: Discussion; comments: Comment[] };
type Tone = "info" | "success" | "error";

function formatTime(value: string) { if (!value) return ""; return new Date(value).toLocaleString("zh-CN"); }

export function ProjectDiscussionDetailPage({ projectId, discussionId }: { projectId: string; discussionId: string }) {
  const [discussion, setDiscussion] = useState<Discussion | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [reply, setReply] = useState("");
  const [status, setStatus] = useState("正在读取讨论。");
  const [tone, setTone] = useState<Tone>("info");
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);

  useEffect(() => { void loadDetail(); }, [projectId, discussionId]);

  async function loadDetail() {
    try {
      setLoading(true);
      const data = await getJSON<DetailResponse>(`/api/v1/projects/${projectId}/discussions/${discussionId}`);
      setDiscussion(data.discussion);
      setComments(data.comments);
      setStatus("讨论已读取。");
      setTone("success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "读取讨论失败。");
      setTone("error");
    } finally { setLoading(false); }
  }

  async function submitReply(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (working || !discussion) return;
    if (reply.trim() === "") { setStatus("请填写回复内容。"); setTone("error"); return; }
    try {
      setWorking(true);
      const created = await postJSON<Comment>(`/api/v1/projects/${projectId}/discussions/${discussionId}/comments`, { body: reply });
      setComments((prev) => [...prev, created]);
      setReply("");
      setStatus("回复已发送。");
      setTone("success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "发送回复失败。");
      setTone("error");
    } finally { setWorking(false); }
  }

  async function updateStatus(nextStatus: "open" | "closed") {
    if (working) return;
    try {
      setWorking(true);
      const updated = await postJSON<Discussion>(`/api/v1/projects/${projectId}/discussions/${discussionId}/status`, { status: nextStatus });
      setDiscussion(updated);
      setStatus(nextStatus === "closed" ? "讨论已关闭。" : "讨论已重新打开。");
      setTone("success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "更新讨论状态失败。");
      setTone("error");
    } finally { setWorking(false); }
  }

  return (
    <section className="panel" style={{ padding: 28, display: "grid", gap: 18 }}>
      <header style={{ display: "grid", gap: 8 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>PlayPage 作品讨论区</p>
        <h1 style={{ margin: 0, fontSize: "2rem" }}>{discussion?.title ?? "讨论详情"}</h1>
        {discussion ? <p style={{ margin: 0, color: "var(--muted)" }}>由 {discussion.authorUsername || "匿名用户"} 发起，{formatTime(discussion.createdAt)}</p> : null}
      </header>
      <div className="status" data-tone={tone === "info" ? undefined : tone} aria-live="polite">{loading ? "正在读取讨论。" : status}</div>
      {discussion ? <>
        <article className="panel" style={{ padding: 18, display: "grid", gap: 10 }}>
          <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
            <span className="soft-badge">{discussion.status === "closed" ? "已关闭" : "进行中"}</span>
            {discussion.projectUrl ? <a href={discussion.projectUrl} target="_blank" rel="noreferrer">打开作品</a> : null}
          </div>
          <p style={{ whiteSpace: "pre-wrap", margin: 0 }}>{discussion.body}</p>
        </article>
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
          {discussion.status === "closed" ? <button className="button-secondary" type="button" disabled={working} onClick={() => void updateStatus("open")}>重新打开讨论</button> : <button className="button-secondary" type="button" disabled={working} onClick={() => void updateStatus("closed")}>关闭讨论</button>}
          <button className="button-secondary" type="button" disabled={loading || working} onClick={() => void loadDetail()}>刷新</button>
        </div>
        <section style={{ display: "grid", gap: 12 }} aria-label="回复列表">
          <h2 style={{ margin: 0 }}>回复</h2>
          {comments.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>还没有回复。</p> : null}
          {comments.map((item) => <article key={item.id} className="panel" style={{ padding: 16, display: "grid", gap: 8 }}><strong>{item.authorUsername || "匿名用户"} <span style={{ color: "var(--muted)", fontWeight: 400 }}>{formatTime(item.createdAt)}</span></strong><p style={{ whiteSpace: "pre-wrap", margin: 0 }}>{item.body}</p></article>)}
        </section>
        {discussion.status === "closed" ? <p className="field-note">这个讨论已经关闭，不能继续回复。</p> : <form onSubmit={submitReply} style={{ display: "grid", gap: 12, maxWidth: 760 }}><label className="field"><span>写回复</span><textarea value={reply} onChange={(event) => setReply(event.target.value)} rows={5} disabled={working} /></label><button className="button-primary" type="submit" disabled={working} aria-busy={working}>{working ? "正在发送" : "发送回复"}</button></form>}
      </> : null}
    </section>
  );
}
