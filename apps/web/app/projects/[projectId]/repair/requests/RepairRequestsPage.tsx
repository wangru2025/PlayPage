"use client";

import type { FormEvent } from "react";
import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";
import {
  aiStatusLabel,
  combinedRepairStatus,
  issueLabel,
  requestStatusLabel,
  type Project,
  type ProjectResponse,
  type RepairAIJob,
  type RepairRequest,
  type StatusTone
} from "../repairTypes";

type Props = { projectId: string };

export function RepairRequestsPage({ projectId }: Props) {
  const [project, setProject] = useState<Project | null>(null);
  const [requests, setRequests] = useState<RepairRequest[]>([]);
  const [jobsByRequest, setJobsByRequest] = useState<Record<string, RepairAIJob | null>>({});
  const [replyDrafts, setReplyDrafts] = useState<Record<string, string>>({});
  const [replyingId, setReplyingId] = useState("");
  const [loading, setLoading] = useState(true);
  const [statusText, setStatusText] = useState("正在读取修复申请。");
  const [statusTone, setStatusTone] = useState<StatusTone>("info");

  useEffect(() => {
    void loadData();
  }, [projectId]);

  async function loadData() {
    setLoading(true);
    try {
      const [projectData, requestData] = await Promise.all([
        getJSON<ProjectResponse>(`/api/v1/projects/${projectId}`),
        getJSON<{ items: RepairRequest[] }>(`/api/v1/projects/${projectId}/repair-requests`)
      ]);
      setProject(projectData.project);
      setRequests(requestData.items);
      setStatusText("修复申请已读取。");
      setStatusTone("success");
      void loadLatestJobs(requestData.items);
    } catch (error) {
      setProject(null);
      setRequests([]);
      setStatusText(error instanceof Error ? error.message : "读取修复申请失败。");
      setStatusTone("error");
    } finally {
      setLoading(false);
    }
  }

  async function loadLatestJobs(items: RepairRequest[]) {
    const pairs = await Promise.all(items.map(async (item) => {
      try {
        const data = await getJSON<{ job: RepairAIJob }>(`/api/v1/projects/${projectId}/repair-requests/${item.id}/ai/latest`);
        return [item.id, data.job] as const;
      } catch {
        return [item.id, null] as const;
      }
    }));
    setJobsByRequest(Object.fromEntries(pairs));
  }

  async function submitReply(event: FormEvent<HTMLFormElement>, item: RepairRequest) {
    event.preventDefault();
    if (replyingId) return;
    const reply = (replyDrafts[item.id] ?? "").trim();
    if (reply === "") {
      setStatusText("请填写要补充的信息。");
      setStatusTone("error");
      return;
    }
    try {
      setReplyingId(item.id);
      setStatusText("正在提交补充信息。");
      setStatusTone("info");
      await postJSON<RepairRequest>(`/api/v1/projects/${projectId}/repair-requests/${item.id}/reply`, { reply });
      setReplyDrafts((current) => ({ ...current, [item.id]: "" }));
      await loadData();
      setStatusText("补充信息已提交。");
      setStatusTone("success");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : "提交补充信息失败。");
      setStatusTone("error");
    } finally {
      setReplyingId("");
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <nav aria-label="修复申请列表导航" style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <a className="button-primary" href={`/projects/${projectId}/repair`}>提交新的修复申请</a>
        </nav>
        <h1 style={{ margin: 0, fontSize: "2.4rem" }}>修复申请</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>{project ? `作品：${project.name}` : "查看这个作品的修复申请。"}</p>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite" role="status">
          {loading ? "正在读取修复申请。" : statusText}
        </div>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 14 }}>
        {requests.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>这个作品还没有修复申请。</p> : null}
        <div className="project-grid">
          {requests.map((item) => {
            const job = jobsByRequest[item.id] ?? null;
            return (
              <article key={item.id} style={{ border: "1px solid var(--line)", borderRadius: 18, padding: 16, display: "grid", gap: 10, background: "rgba(255,255,255,0.72)" }}>
                <div style={{ display: "grid", gap: 4 }}>
                  <strong>状态：{combinedRepairStatus(item, job)}</strong>
                  <span style={{ color: "var(--muted)" }}>管理员状态：{requestStatusLabel(item.status)}</span>
                  <span style={{ color: "var(--muted)" }}>AI 状态：{aiStatusLabel(job?.status ?? "")}</span>
                  <span style={{ color: "var(--muted)" }}>问题类型：{issueLabel(item.issueType)}</span>
                </div>
                <p style={{ margin: 0, whiteSpace: "pre-wrap" }}>{item.description}</p>
                {item.expected ? <p style={{ margin: 0, whiteSpace: "pre-wrap", color: "var(--muted)" }}>期望：{item.expected}</p> : null}
                {item.adminReply ? <div className="status"><strong>管理员回复：</strong>{item.adminReply}</div> : null}
                {item.userReply ? <div className="status"><strong>你的补充：</strong>{item.userReply}</div> : null}
                <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
                  <a className="button-secondary" href={`/projects/${projectId}/repair/${item.id}/roundtable`}>查看圆桌聊天记录</a>
                  <a className="button-secondary" href={`/projects/${projectId}/repair/${item.id}/ai`}>进入 AI 圆桌页面</a>
                </div>
                {item.status === "need_info" && !item.userReply ? (
                  <form onSubmit={(event) => submitReply(event, item)} style={{ display: "grid", gap: 10 }}>
                    <p style={{ margin: 0, color: "var(--muted)" }}>管理员需要你补充信息。只能回复一次，请一次写清楚。</p>
                    <label className="field">
                      <span>要补充的信息</span>
                      <textarea rows={4} value={replyDrafts[item.id] ?? ""} onChange={(event) => setReplyDrafts((current) => ({ ...current, [item.id]: event.target.value }))} />
                    </label>
                    <div>
                      <button className="button-primary" type="submit" disabled={replyingId !== ""} aria-busy={replyingId === item.id}>
                        {replyingId === item.id ? "正在提交" : "提交补充信息"}
                      </button>
                    </div>
                  </form>
                ) : null}
              </article>
            );
          })}
        </div>
      </section>
    </section>
  );
}
