"use client";

import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";
import type { TemplateSubmission } from "@/app/templates/templateTypes";

const tableWrapStyle = { overflowX: "auto" as const };
const tableStyle = { width: "100%", borderCollapse: "collapse" as const, minWidth: 860 };
const cellStyle = { borderBottom: "1px solid var(--line)", padding: "10px 12px", textAlign: "left" as const, verticalAlign: "top" as const };

function statusLabel(status: string): string {
  switch (status) {
    case "pending": return "待审核";
    case "published": return "已发布";
    case "rejected": return "已驳回";
    default: return status;
  }
}

function formatDate(value: string): string {
  if (!value) return "未知";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", { hour12: false });
}

export function AdminTemplateSubmissions() {
  const [items, setItems] = useState<TemplateSubmission[]>([]);
  const [showFinished, setShowFinished] = useState(false);
  const [statusText, setStatusText] = useState("正在读取模板投稿……");
  const [workingId, setWorkingId] = useState("");

  async function load() {
    try {
      const payload = await getJSON<{ items: TemplateSubmission[] }>("/api/v1/admin/template-submissions");
      setItems(payload.items);
      setStatusText("模板投稿已读取。");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : "读取模板投稿失败。");
    }
  }

  useEffect(() => { void load(); }, []);

  async function review(item: TemplateSubmission, status: "published" | "rejected") {
    const adminNote = status === "rejected" ? window.prompt("请输入驳回原因。") ?? "" : window.prompt("审核备注，可留空。") ?? "";
    if (status === "rejected" && !adminNote.trim()) return;
    try {
      setWorkingId(item.id);
      setStatusText("正在保存审核结果……");
      await postJSON(`/api/v1/admin/template-submissions/${item.id}/review`, { status, adminNote });
      await load();
      setStatusText("审核结果已保存。");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : "保存审核结果失败。");
    } finally {
      setWorkingId("");
    }
  }

  const visibleItems = showFinished ? items : items.filter((item) => item.status === "pending");

  return (
    <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-labelledby="admin-template-submissions-title">
      <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap", alignItems: "center" }}>
        <h2 id="admin-template-submissions-title" style={{ margin: 0 }}>模板投稿审核</h2>
        <button className="button-secondary" type="button" onClick={() => setShowFinished((value) => !value)}>
          {showFinished ? "隐藏已完成项目" : "显示已完成项目"}
        </button>
      </div>
      <div className="status" aria-live="polite">{statusText}</div>
      {visibleItems.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>当前没有需要处理的模板投稿。</p> : null}
      <div style={tableWrapStyle} role="region" aria-label="模板投稿审核表格">
        <table style={tableStyle}>
          <thead>
            <tr>
              <th style={cellStyle}>模板</th>
              <th style={cellStyle}>作者</th>
              <th style={cellStyle}>分类</th>
              <th style={cellStyle}>声明</th>
              <th style={cellStyle}>状态</th>
              <th style={cellStyle}>创建时间</th>
              <th style={cellStyle}>操作</th>
            </tr>
          </thead>
          <tbody>
            {visibleItems.map((item) => (
              <tr key={item.id}>
                <td style={cellStyle}>
                  <strong>{item.name}</strong>
                  <div style={{ color: "var(--muted)" }}>{item.slug}</div>
                  <div>{item.summary}</div>
                </td>
                <td style={cellStyle}>{item.authorName || item.authorEmail || "未知作者"}<br />{item.authorEmail || ""}</td>
                <td style={cellStyle}>{item.categoryLabel}<br /><span style={{ color: "var(--muted)" }}>{item.category}</span></td>
                <td style={cellStyle}>
                  {item.interactiveRequired ? "需要互动功能" : "不强制互动"}<br />
                  参数 {item.configFields.length} 个，数据表 {item.collections.length} 个
                </td>
                <td style={cellStyle}>{statusLabel(item.status)}{item.adminNote ? <div style={{ color: "var(--muted)" }}>{item.adminNote}</div> : null}</td>
                <td style={cellStyle}>{formatDate(item.createdAt)}</td>
                <td style={cellStyle}>
                  <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                    <a className="button-secondary" href={`/api/v1/admin/template-submissions/${item.id}/preview`} target="_blank" rel="noreferrer">预览</a>
                    {item.status === "pending" ? (
                      <>
                        <button className="button-primary" type="button" disabled={workingId === item.id} onClick={() => review(item, "published")}>发布模板</button>
                        <button className="button-ghost" type="button" disabled={workingId === item.id} onClick={() => review(item, "rejected")}>驳回</button>
                      </>
                    ) : <span style={{ color: "var(--muted)" }}>已处理</span>}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
