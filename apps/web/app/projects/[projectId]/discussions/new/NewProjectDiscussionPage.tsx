"use client";

import { FormEvent, useState } from "react";
import { postJSON } from "@/lib/api";

type Discussion = { id: string; title: string; };
type Tone = "info" | "success" | "error";

export function NewProjectDiscussionPage({ projectId }: { projectId: string }) {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [status, setStatus] = useState("请填写讨论标题和内容。");
  const [tone, setTone] = useState<Tone>("info");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;
    if (title.trim() === "" || body.trim() === "") {
      setStatus("请填写讨论标题和内容。");
      setTone("error");
      return;
    }
    try {
      setSubmitting(true);
      setStatus("正在发布讨论。");
      setTone("info");
      const created = await postJSON<Discussion>(`/api/v1/projects/${projectId}/discussions`, { title, body });
      setStatus("讨论已发布，正在打开详情页。");
      setTone("success");
      window.location.href = `/projects/${projectId}/discussions/${created.id}`;
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "发布讨论失败。");
      setTone("error");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="panel" style={{ padding: 28, display: "grid", gap: 18 }}>
      <header style={{ display: "grid", gap: 8 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>PlayPage 作品讨论区</p>
        <h1 style={{ margin: 0, fontSize: "2rem" }}>发起讨论</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 760 }}>可以反馈问题、提出建议，或者和作者交流创作想法。</p>
      </header>

      <div className="status" data-tone={tone === "info" ? undefined : tone} aria-live="polite">{status}</div>

      <form onSubmit={submit} style={{ display: "grid", gap: 14, maxWidth: 800 }}>
        <label className="field">
          <span>讨论标题</span>
          <input value={title} onChange={(event) => setTitle(event.target.value)} disabled={submitting} maxLength={120} required placeholder="一句话说明想讨论什么" />
          <span className="field-note">最多 120 个字。</span>
        </label>
        <label className="field">
          <span>讨论内容</span>
          <textarea value={body} onChange={(event) => setBody(event.target.value)} disabled={submitting} rows={10} required placeholder="请具体说明问题、建议或想法。" />
        </label>
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
          <button className="button-primary" type="submit" disabled={submitting} aria-busy={submitting}>{submitting ? "正在发布" : "发布讨论"}</button>
        </div>
      </form>
    </section>
  );
}
