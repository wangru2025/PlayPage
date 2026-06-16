"use client";

import type { FormEvent } from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import { buildWebSocketURL, getJSON, postJSON } from "@/lib/api";
import {
  aiStatusLabel,
  combinedRepairStatus,
  isFinalAIStatus,
  type Project,
  type ProjectResponse,
  type RepairAIJob,
  type RepairAIMessage,
  type RepairRequest,
  type StatusTone
} from "../../repairTypes";

type Props = { projectId: string; requestId: string; autoStart?: boolean };

export function AIroundtablePage({ projectId, requestId, autoStart }: Props) {
  const [project, setProject] = useState<Project | null>(null);
  const [request, setRequest] = useState<RepairRequest | null>(null);
  const [job, setJob] = useState<RepairAIJob | null>(null);
  const [messages, setMessages] = useState<RepairAIMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);
  const [feedbackText, setFeedbackText] = useState("");
  const [showFeedback, setShowFeedback] = useState(false);
  const [statusText, setStatusText] = useState("正在读取 AI 圆桌。");
  const [statusTone, setStatusTone] = useState<StatusTone>("info");
  const wsRef = useRef<WebSocket | null>(null);
  const pollRef = useRef<number | null>(null);

  const canPreview = job?.status === "ready_preview" || job?.status === "published";
  const requestStatus = useMemo(() => request ? combinedRepairStatus(request, job) : "", [request, job]);

  useEffect(() => {
    void loadData();
    return () => {
      wsRef.current?.close();
      stopPolling();
    };
  }, [projectId, requestId]);

  async function loadData() {
    setLoading(true);
    try {
      const [projectData, requestData] = await Promise.all([
        getJSON<ProjectResponse>(`/api/v1/projects/${projectId}`),
        getJSON<{ items: RepairRequest[] }>(`/api/v1/projects/${projectId}/repair-requests`)
      ]);
      setProject(projectData.project);
      const currentRequest = requestData.items.find((item) => item.id === requestId) ?? null;
      setRequest(currentRequest);
      try {
        const data = await getJSON<{ job: RepairAIJob; messages: RepairAIMessage[] }>(`/api/v1/projects/${projectId}/repair-requests/${requestId}/ai/latest`);
        setJob(data.job);
        setMessages(data.messages ?? []);
        updateStatusFromJob(data.job);
        connectWS(data.job.id);
      } catch {
        setJob(null);
        setMessages([]);
        if (autoStart && currentRequest) {
          await startAIFor(currentRequest);
        } else {
          setStatusText("还没有启动 AI 圆桌。");
          setStatusTone("info");
        }
      }
    } catch (error) {
      setProject(null);
      setRequest(null);
      setJob(null);
      setMessages([]);
      setStatusText(error instanceof Error ? error.message : "读取 AI 圆桌失败。");
      setStatusTone("error");
    } finally {
      setLoading(false);
    }
  }

  function setStatus(message: string, tone: StatusTone) {
    setStatusText(message);
    setStatusTone(tone);
  }

  function stopPolling() {
    if (pollRef.current !== null) {
      window.clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }

  function startPolling() {
    stopPolling();
    pollRef.current = window.setInterval(() => {
      void refreshLatest(false);
    }, 5000);
  }

  async function refreshLatest(reconnect: boolean) {
    try {
      const data = await getJSON<{ job: RepairAIJob; messages: RepairAIMessage[] }>(`/api/v1/projects/${projectId}/repair-requests/${requestId}/ai/latest`);
      setJob(data.job);
      setMessages(data.messages ?? []);
      updateStatusFromJob(data.job);
      if (isFinalAIStatus(data.job.status)) stopPolling();
      if (reconnect) connectWS(data.job.id);
    } catch {}
  }

  function updateStatusFromJob(nextJob: RepairAIJob) {
    if (nextJob.status === "failed") {
      setStatus(nextJob.errorMessage || "AI 圆桌失败，请等待管理员处理。", "error");
    } else if (nextJob.status === "ready_preview") {
      setStatus("AI 圆桌已生成修复版本，请先预览效果。", "success");
    } else if (nextJob.status === "published") {
      setStatus("修复版本已发布。", "success");
    } else if (nextJob.status === "needs_admin") {
      setStatus("这条申请已经交给管理员处理。", "info");
    } else if (nextJob.status === "canceled") {
      setStatus("AI 圆桌已叫停。", "info");
    } else if (nextJob.status === "running") {
      setStatus("AI 圆桌正在抢救，消息会陆续显示。", "info");
    }
  }

  function connectWS(jobId: string) {
    wsRef.current?.close();
    const ws = new WebSocket(buildWebSocketURL(`/api/v1/projects/${projectId}/repair-requests/${requestId}/ai/ws?jobId=${encodeURIComponent(jobId)}`));
    wsRef.current = ws;
    startPolling();
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === "snapshot") {
          setJob(data.job);
          setMessages(data.messages ?? []);
          if (data.job) updateStatusFromJob(data.job);
          if (data.job && isFinalAIStatus(data.job.status)) stopPolling();
        }
        if (data.type === "message") {
          setMessages((current) => current.some((m) => m.id === data.message.id) ? current : [...current, data.message]);
        }
        if (data.type === "job" && data.job) {
          setJob(data.job);
          updateStatusFromJob(data.job);
          if (isFinalAIStatus(data.job.status)) stopPolling();
        }
      } catch {}
    };
    ws.onerror = () => setStatus("实时连接暂时异常，页面会继续自动刷新最新进度。", "error");
  }

  async function startAI() {
    if (!request || working) return;
    await startAIFor(request);
  }

  async function startAIFor(targetRequest: RepairRequest) {
    if (working) return;
    try {
      setWorking(true);
      setStatus("正在启动 AI 急救圆桌。", "info");
      const data = await postJSON<{ job: RepairAIJob }>(`/api/v1/projects/${projectId}/repair-requests/${targetRequest.id}/ai/start`, {});
      setJob(data.job);
      setMessages([]);
      connectWS(data.job.id);
      setStatus("AI 急救圆桌已经开工。", "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "启动 AI 圆桌失败。", "error");
    } finally {
      setWorking(false);
    }
  }

  async function previewAI() {
    if (!request || !job) return;
    try {
      setWorking(true);
      const data = await postJSON<{ url: string; job: RepairAIJob }>(`/api/v1/projects/${projectId}/repair-requests/${request.id}/ai/preview`, {});
      setJob(data.job);
      window.open(data.url, "_blank", "noopener,noreferrer");
      setStatus("预览页面已打开。", "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "生成预览失败。", "error");
    } finally {
      setWorking(false);
    }
  }

  async function stopAI() {
    if (!request || !job || job.status !== "running" || working) return;
    if (!confirm("确认叫停 AI 圆桌吗？叫停后本轮不会继续生成修复结果。")) return;
    try {
      setWorking(true);
      setStatus("正在叫停 AI 圆桌。", "info");
      const data = await postJSON<{ job: RepairAIJob; messages: RepairAIMessage[] }>(`/api/v1/projects/${projectId}/repair-requests/${request.id}/ai/stop`, {});
      setJob(data.job);
      setMessages(data.messages ?? []);
      updateStatusFromJob(data.job);
      stopPolling();
      wsRef.current?.close();
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "叫停 AI 圆桌失败。", "error");
    } finally {
      setWorking(false);
    }
  }

  async function publishAI() {
    if (!request || !confirm("确认发布 AI 修复后的版本吗？")) return;
    try {
      setWorking(true);
      const data = await postJSON<{ message: string; job: RepairAIJob }>(`/api/v1/projects/${projectId}/repair-requests/${request.id}/ai/publish`, {});
      setJob(data.job);
      setStatus(data.message || "修复版本已发布。", "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "发布失败。", "error");
    } finally {
      setWorking(false);
    }
  }

  async function reportStillBroken(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!request || !job) return;
    if (job.round >= 2) {
      alert("真不好意思，这几个牛马太傻了，已经把此申请交给管理员处理。");
      await postJSON(`/api/v1/projects/${projectId}/repair-requests/${request.id}/ai/feedback`, { feedback: "第二轮后用户仍反馈有问题" }).catch(() => undefined);
      return;
    }
    if (feedbackText.trim() === "") {
      setStatus("请填写问题还存在什么情况。", "error");
      return;
    }
    try {
      setWorking(true);
      const data = await postJSON<{ job: RepairAIJob }>(`/api/v1/projects/${projectId}/repair-requests/${request.id}/ai/feedback`, { feedback: feedbackText });
      setJob(data.job);
      setMessages([]);
      setFeedbackText("");
      setShowFeedback(false);
      connectWS(data.job.id);
      setStatus("第二轮 AI 圆桌已经开始。", "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "提交反馈失败。", "error");
    } finally {
      setWorking(false);
    }
  }

  if (loading) {
    return (
      <section style={{ display: "grid", gap: 18 }}>
        <header className="panel" style={{ padding: 28 }}>
          <h1 style={{ margin: 0 }}>AI 圆桌</h1>
          <div className="status">正在读取 AI 圆桌。</div>
        </header>
      </section>
    );
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <nav aria-label="AI 圆桌页面导航" style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <a className="button-secondary" href={`/projects/${projectId}/repair/requests`}>返回修复申请</a>
          <a className="button-secondary" href={`/projects/${projectId}/repair/${requestId}/roundtable`}>查看圆桌聊天记录</a>
        </nav>
        <h1 style={{ margin: 0, fontSize: "2.4rem" }}>AI 圆桌</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 800 }}>
          这里会实时显示 AI 牛马们的讨论过程。修好后先预览，确认没问题再发布。
        </p>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite" role="status">{statusText}</div>
      </header>

      {project && request ? (
        <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
          <div style={{ display: "grid", gap: 6 }}>
            <h2 style={{ margin: 0 }}>作品：{project.name}</h2>
            <p style={{ margin: 0, color: "var(--muted)" }}>申请状态：{requestStatus}</p>
          </div>

          <div aria-live="polite" aria-label="AI 圆桌消息" style={{ display: "grid", gap: 10 }}>
            {messages.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>还没有圆桌消息。</p> : null}
            {messages.map((message) => (
              <article key={message.id} style={{ border: "1px solid var(--line)", borderRadius: 14, padding: 12, background: "rgba(255,255,255,0.72)" }}>
                <strong>{message.messageSeq}. {message.agentName}</strong>
                <p style={{ margin: "6px 0 0", whiteSpace: "pre-wrap" }}>{message.content}</p>
              </article>
            ))}
          </div>

          {job?.status === "ready_preview" || job?.status === "published" ? (
            <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
              <button className="button-secondary" type="button" onClick={previewAI} disabled={working}>预览修复效果</button>
              <button className="button-primary" type="button" onClick={publishAI} disabled={working || job.status === "published"}>没问题，发布</button>
              <button className="button-secondary" type="button" onClick={() => setShowFeedback(true)} disabled={working || job.status === "published"}>还有问题</button>
            </div>
          ) : null}

          {job?.status === "running" ? (
            <div style={{ display: "grid", gap: 10 }}>
              <div className="status" aria-live="polite">圆桌还在进行中，等它写完再看结果。补丁如果没打上，会自动继续重试。</div>
              <div>
                <button className="button-secondary" type="button" onClick={stopAI} disabled={working}>
                  {working ? "正在处理" : "叫停 AI 圆桌"}
                </button>
              </div>
            </div>
          ) : null}

          {!job ? (
            <div>
              <button className="button-primary" type="button" onClick={startAI} disabled={working}>
                {working ? "正在启动" : "启动 AI 圆桌"}
              </button>
            </div>
          ) : null}

          {showFeedback ? (
            <form onSubmit={reportStillBroken} style={{ display: "grid", gap: 12, borderTop: "1px solid var(--line)", paddingTop: 14 }}>
              <label className="field">
                <span>问题还存在什么情况</span>
                <textarea rows={4} value={feedbackText} onChange={(event) => setFeedbackText(event.target.value)} />
              </label>
              <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
                <button className="button-primary" type="submit" disabled={working}>发送反馈</button>
                <button className="button-secondary" type="button" onClick={() => setShowFeedback(false)}>取消</button>
              </div>
            </form>
          ) : null}

          {job?.status === "failed" ? <div className="status" data-tone="error" role="alert">{job.errorMessage || "AI 圆桌失败，请等待管理员处理。"}</div> : null}
        </section>
      ) : null}
    </section>
  );
}
