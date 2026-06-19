"use client";

import type { FormEvent } from "react";
import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";
import { issueOptions, type Project, type ProjectResponse, type RepairRequest, type StatusTone } from "./repairTypes";

type Props = { projectId: string };

export function RepairRequestForm({ projectId }: Props) {
  const [project, setProject] = useState<Project | null>(null);
  const [issueType, setIssueType] = useState("page_broken");
  const [description, setDescription] = useState("");
  const [expected, setExpected] = useState("");
  const [contact, setContact] = useState("");
  const [allowAdminEdit, setAllowAdminEdit] = useState(true);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [statusText, setStatusText] = useState("正在读取作品。");
  const [statusTone, setStatusTone] = useState<StatusTone>("info");

  useEffect(() => {
    void loadProject();
  }, [projectId]);

  async function loadProject() {
    setLoading(true);
    try {
      const data = await getJSON<ProjectResponse>(`/api/v1/projects/${projectId}`);
      setProject(data.project);
      setStatusText(data.project.currentReleaseId ? "请描述你遇到的问题。" : "这个作品还没有上传内容，建议先上传作品。");
      setStatusTone("info");
    } catch (error) {
      setProject(null);
      setStatusText(error instanceof Error ? error.message : "读取作品失败。");
      setStatusTone("error");
    } finally {
      setLoading(false);
    }
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;
    if (description.trim() === "") {
      setStatusText("请填写遇到的问题。");
      setStatusTone("error");
      return;
    }
    if (!allowAdminEdit) {
      setStatusText("请先确认允许 PlayPage 查看并修复这个作品。");
      setStatusTone("error");
      return;
    }
    try {
      setSubmitting(true);
      setStatusText("正在提交修复申请。");
      setStatusTone("info");
      const created = await postJSON<RepairRequest>(`/api/v1/projects/${projectId}/repair-requests`, {
        issueType,
        description,
        expected,
        contact,
        allowAdminEdit
      });
      setDescription("");
      setExpected("");
      setContact("");
      setStatusText("修复申请已提交。");
      setStatusTone("success");
      const startAI = window.confirm("修复申请已提交。要现在抓几个 AI 牛马来修作品，让管理员摸鱼一下吗？");
      if (startAI) {
        window.location.href = `/projects/${projectId}/repair/${created.id}/ai?autoStart=1`;
      } else {
        window.location.href = `/projects/${projectId}/repair/requests`;
      }
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : "提交修复申请失败。");
      setStatusTone("error");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <nav aria-label="修复申请页面导航" style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <a className="button-secondary" href={`/projects/${projectId}/repair/requests`}>查看修复申请</a>
          <a className="button-secondary" href="/projects/repair">查看活动说明</a>
        </nav>
        <h1 style={{ margin: 0, fontSize: "2.4rem" }}>提交修复申请</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 800 }}>
          页面白屏、按钮没反应、互动功能报错、中文乱码，都可以在这里提交。提交后你可以选择立刻启动 AI 圆桌，也可以等管理员处理。
        </p>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite" role="status">
          {loading ? "正在读取作品。" : statusText}
        </div>
      </header>

      {project ? (
        <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
          <div style={{ display: "grid", gap: 6 }}>
            <h2 style={{ margin: 0 }}>作品：{project.name}</h2>
            <p style={{ margin: 0, color: "var(--muted)" }}>作品地址：{project.publicUrl}</p>
            <div>
              <a className="button-secondary" href={project.publicUrl} target="_blank" rel="noreferrer">打开作品</a>
            </div>
          </div>
          <form onSubmit={submit} style={{ display: "grid", gap: 14 }}>
            <label className="field">
              <span>问题类型</span>
              <select value={issueType} onChange={(event) => setIssueType(event.target.value)}>
                {issueOptions.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
              </select>
            </label>
            <label className="field">
              <span>遇到的问题</span>
              <textarea rows={6} value={description} onChange={(event) => setDescription(event.target.value)} />
              <span className="field-note">例如：页面显示数据表不存在、按钮没反应、互动功能连接失败、中文乱码。</span>
            </label>
            <label className="field">
              <span>期望效果</span>
              <textarea rows={4} value={expected} onChange={(event) => setExpected(event.target.value)} />
            </label>
            <label className="field">
              <span>补充联系方式或备注，可选</span>
              <input value={contact} onChange={(event) => setContact(event.target.value)} />
            </label>
            <label style={{ display: "flex", gap: 10, alignItems: "flex-start" }}>
              <input type="checkbox" checked={allowAdminEdit} onChange={(event) => setAllowAdminEdit(event.target.checked)} style={{ marginTop: 6 }} />
              <span>我同意 PlayPage 查看并修复这个作品的 HTML 或相关文件，用于处理本次修复申请。</span>
            </label>
            <div>
              <button className="button-primary" type="submit" disabled={submitting} aria-busy={submitting}>
                {submitting ? "正在提交" : "提交修复申请"}
              </button>
            </div>
          </form>
        </section>
      ) : null}
    </section>
  );
}
