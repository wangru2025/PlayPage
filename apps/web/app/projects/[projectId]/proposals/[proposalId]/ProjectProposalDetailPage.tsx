"use client";

import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";

type Proposal = { id:string; sourceProjectId:string; sourceProjectName:string; sourceProjectUrl:string; targetProjectId:string; targetProjectName:string; targetProjectUrl:string; authorUsername:string; title:string; body:string; status:string; reviewNote:string; mergedReleaseId?:string; createdAt:string; updatedAt:string; };
type DetailResponse = { proposal: Proposal; canReview: boolean };
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

export function ProjectProposalDetailPage({ projectId, proposalId }: { projectId: string; proposalId: string }) {
  const [proposal, setProposal] = useState<Proposal | null>(null);
  const [canReview, setCanReview] = useState(false);
  const [note, setNote] = useState("");
  const [status, setStatus] = useState("正在读取提案。");
  const [tone, setTone] = useState<Tone>("info");
  const [loading, setLoading] = useState(true);
  const [reviewing, setReviewing] = useState(false);

  useEffect(() => { void loadDetail(); }, [projectId, proposalId]);

  async function loadDetail() {
    try {
      setLoading(true);
      const data = await getJSON<DetailResponse>(`/api/v1/projects/${projectId}/proposals/${proposalId}`);
      setProposal(data.proposal);
      setCanReview(data.canReview);
      setStatus("提案已读取。");
      setTone("success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "读取提案失败。");
      setTone("error");
    } finally {
      setLoading(false);
    }
  }

  async function review(nextStatus: "accepted" | "rejected") {
    if (nextStatus === "accepted" && !confirm("确定采纳这个提案，并把改编作品的提案版本发布为原作品的新版本吗？")) return;
    if (nextStatus === "rejected" && !confirm("确定拒绝这个提案吗？")) return;
    try {
      setReviewing(true);
      const updated = await postJSON<Proposal>(`/api/v1/projects/${projectId}/proposals/${proposalId}/review`, { status: nextStatus, note });
      setProposal(updated);
      setStatus(nextStatus === "accepted" ? "已采纳并发布新版本。" : "已拒绝这个提案。");
      setTone("success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "处理提案失败。");
      setTone("error");
    } finally {
      setReviewing(false);
    }
  }

  return <section className="panel" style={{ padding: 28, display: "grid", gap: 18 }}>
    <header style={{ display: "grid", gap: 8 }}>
      <p style={{ margin: 0, color: "var(--muted)" }}>改进提案详情</p>
      <h1 style={{ margin: 0, fontSize: "2rem" }}>{proposal?.title || "改进提案"}</h1>
    </header>
    <div className="status" data-tone={tone === "info" ? undefined : tone} aria-live="polite">{loading ? "正在读取提案。" : status}</div>
    {proposal ? <>
      <section className="panel" style={{ padding: 18, display: "grid", gap: 10 }}>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}><span className="soft-badge">{statusText(proposal.status)}</span><span className="soft-badge">提交时间 {formatTime(proposal.createdAt)}</span></div>
        <p style={{ margin: 0, color: "var(--muted)" }}>由 @{proposal.authorUsername || "匿名用户"} 从改编作品《{proposal.sourceProjectName || "改编作品"}》提交给《{proposal.targetProjectName || "原作品"}》。</p>
        <div style={{ whiteSpace: "pre-wrap", lineHeight: 1.7 }}>{proposal.body}</div>
        {proposal.reviewNote ? <p style={{ margin: 0, color: "var(--muted)", whiteSpace: "pre-wrap" }}>处理备注：{proposal.reviewNote}</p> : null}
      </section>
      <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
        {proposal.sourceProjectUrl ? <a className="button-secondary" href={proposal.sourceProjectUrl} target="_blank" rel="noreferrer">打开改编作品</a> : null}
        {proposal.targetProjectUrl ? <a className="button-secondary" href={proposal.targetProjectUrl} target="_blank" rel="noreferrer">打开原作品</a> : null}
      </div>
      {canReview && proposal.status === "open" ? <section className="panel" style={{ padding: 18, display: "grid", gap: 12 }}>
        <h2 style={{ margin: 0 }}>处理提案</h2>
        <label className="field"><span>处理备注，可选</span><textarea rows={4} value={note} onChange={(event) => setNote(event.target.value)} maxLength={1000} /></label>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <button className="button-primary" type="button" onClick={() => void review("accepted")} disabled={reviewing}>采纳并发布</button>
          <button className="button-secondary" type="button" onClick={() => void review("rejected")} disabled={reviewing}>拒绝提案</button>
        </div>
      </section> : null}
    </> : null}
  </section>;
}
