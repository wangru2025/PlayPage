"use client";

import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";

type DomainDeleteRequest = {
  id: string;
  projectName?: string;
  projectPublicUrl?: string;
  ownerEmail?: string;
  username?: string;
  domain: string;
  reason: string;
  status: string;
  adminNote: string;
  createdAt: string;
};

const tableWrapStyle = { overflowX: "auto" as const };
const tableStyle = { width: "100%", borderCollapse: "collapse" as const, minWidth: 820 };
const cellStyle = { borderBottom: "1px solid var(--line)", padding: "10px 12px", textAlign: "left" as const, verticalAlign: "top" as const };

function statusLabel(status: string): string {
  if (status === "completed") return "已删除配置";
  if (status === "rejected") return "已驳回";
  return "待处理";
}

function formatDate(value: string): string {
  if (!value) return "未知";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", { hour12: false });
}

export function AdminDomainDeleteRequests() {
  const [items, setItems] = useState<DomainDeleteRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusText, setStatusText] = useState("正在读取独立网址删除申请。");
  const [showFinished, setShowFinished] = useState(false);

  useEffect(() => {
    void loadData();
  }, []);

  async function loadData() {
    try {
      const data = await getJSON<{ items: DomainDeleteRequest[] }>("/api/v1/admin/project-domain-delete-requests");
      setItems(data.items);
      setStatusText("独立网址删除申请已读取。");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : "读取独立网址删除申请失败。");
    } finally {
      setLoading(false);
    }
  }

  async function reviewRequest(item: DomainDeleteRequest, status: "completed" | "rejected") {
    let adminNote = "";
    if (status === "completed") {
      const ok = window.confirm(`请确认已经在服务器运行 ./pz.sh d 并删除配置：${item.domain}`);
      if (!ok) return;
      adminNote = "已运行 ./pz.sh d 删除独立网址配置";
    } else {
      const note = window.prompt("请输入驳回原因。");
      if (!note || note.trim() === "") return;
      adminNote = note.trim();
    }
    setStatusText("正在保存处理结果。");
    try {
      await postJSON(`/api/v1/admin/project-domain-delete-requests/${item.id}/review`, { status, adminNote });
      await loadData();
      setStatusText("处理结果已保存。");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : "保存处理结果失败。");
    }
  }

  const visibleItems = showFinished ? items : items.filter((item) => item.status === "pending");

  return (
    <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-labelledby="admin-domain-delete-title">
      <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap", alignItems: "center" }}>
        <h2 id="admin-domain-delete-title" style={{ margin: 0 }}>删除独立网址申请</h2>
        <button className="button-secondary" type="button" onClick={() => setShowFinished((value) => !value)}>
          {showFinished ? "隐藏已完成项目" : "显示已完成项目"}
        </button>
      </div>
      <p style={{ margin: 0, color: "var(--muted)" }}>
        先在服务器执行 ./pz.sh d 删除对应域名配置并重载服务，再在这里标记已删除配置。
      </p>
      <div className="status" aria-live="polite">{loading ? "正在读取独立网址删除申请。" : statusText}</div>
      {visibleItems.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>当前没有需要处理的删除申请。</p> : null}
      <div style={tableWrapStyle} role="region" aria-label="删除独立网址申请表格">
        <table style={tableStyle}>
          <thead>
            <tr>
              <th style={cellStyle}>域名</th>
              <th style={cellStyle}>用户</th>
              <th style={cellStyle}>作品</th>
              <th style={cellStyle}>申请原因</th>
              <th style={cellStyle}>状态</th>
              <th style={cellStyle}>管理员备注</th>
              <th style={cellStyle}>创建时间</th>
              <th style={cellStyle}>操作</th>
            </tr>
          </thead>
          <tbody>
            {visibleItems.map((item) => (
              <tr key={item.id}>
                <td style={cellStyle}>{item.domain}</td>
                <td style={cellStyle}>{item.ownerEmail || item.username || "未知用户"}</td>
                <td style={cellStyle}>
                  {item.projectPublicUrl ? <a href={item.projectPublicUrl} target="_blank" rel="noreferrer">{item.projectName || item.projectPublicUrl}</a> : (item.projectName || "作品可能已删除")}
                </td>
                <td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{item.reason || "-"}</td>
                <td style={cellStyle}>{statusLabel(item.status)}</td>
                <td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{item.adminNote || "-"}</td>
                <td style={cellStyle}>{formatDate(item.createdAt)}</td>
                <td style={cellStyle}>
                  {item.status === "pending" ? (
                    <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                      <button className="button-primary" type="button" onClick={() => reviewRequest(item, "completed")}>标记已删除配置</button>
                      <button className="button-ghost" type="button" onClick={() => reviewRequest(item, "rejected")}>驳回</button>
                    </div>
                  ) : "已处理"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
