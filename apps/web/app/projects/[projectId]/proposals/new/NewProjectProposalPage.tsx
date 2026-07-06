"use client";

import { FormEvent, useState } from "react";
import { postJSON } from "@/lib/api";

type Proposal = { id: string; targetProjectId: string; targetProjectName: string; title: string; status: string };
type Tone = "info" | "success" | "error";

export function NewProjectProposalPage({ projectId }: { projectId: string }) {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [status, setStatus] = useState("把这个改编作品当前发布版本提交给原作者。提交后，原作者可以采纳并发布到原作品。");
  const [tone, setTone] = useState<Tone>("info");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    try {
      setSubmitting(true);
      setTone("info");
      setStatus("正在提交提案。");
      const item = await postJSON<Proposal>(`/api/v1/projects/${projectId}/proposals`, { title, body });
      setTone("success");
      setStatus("提案已提交，正在进入提案详情。");
      window.location.href = `/projects/${item.targetProjectId}/proposals/${item.id}`;
    } catch (error) {
      setTone("error");
      setStatus(error instanceof Error ? error.message : "提交提案失败。");
    } finally {
      setSubmitting(false);
    }
  }

  return <section className="panel" style={{ padding: 28, display: "grid", gap: 18 }}>
    <header style={{ display: "grid", gap: 8 }}>
      <p style={{ margin: 0, color: "var(--muted)" }}>提交改进提案</p>
      <h1 style={{ margin: 0, fontSize: "2rem" }}>把改编成果提交给原作者</h1>
      <p style={{ margin: 0, color: "var(--muted)", maxWidth: 760 }}>系统会固定使用这个改编作品的当前发布版本作为提案内容。原作者采纳后，会在原作品更新历史里记录来源。</p>
    </header>
    <form onSubmit={submit} style={{ display: "grid", gap: 14 }}>
      <label className="field"><span>提案标题</span><input value={title} onChange={(event) => setTitle(event.target.value)} maxLength={120} placeholder="例如：修复登录后回复不显示的问题" required /></label>
      <label className="field"><span>改进说明</span><textarea value={body} onChange={(event) => setBody(event.target.value)} rows={10} maxLength={20000} placeholder="说明你改了哪里、解决了什么问题、是否有需要原作者注意的地方。" required /></label>
      <div className="status" data-tone={tone === "info" ? undefined : tone} aria-live="polite">{status}</div>
      <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
        <button className="button-primary" type="submit" disabled={submitting}>{submitting ? "正在提交" : "提交提案"}</button>
      </div>
    </form>
  </section>;
}
