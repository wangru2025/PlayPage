"use client";

import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";

type RepairRequest = {
  id: string;
  projectId: string;
  projectName: string;
  projectPublicUrl: string;
  ownerEmail: string;
  username: string;
  issueType: string;
  description: string;
  expected: string;
  allowAdminEdit: boolean;
  contact: string;
  status: string;
  adminReply: string;
  userReply: string;
  userRepliedAt: string;
  createdAt: string;
};

const text = {
  title: "网页修复申请",
  loading: "正在加载修复申请……",
  empty: "当前没有需要处理的修复申请。",
  loadFail: "修复申请加载失败。",
  saving: "正在保存处理结果……",
  saveFail: "处理结果保存失败。",
  saveOk: "处理结果已经保存。",
  user: "用户",
  project: "作品",
  issue: "问题类型",
  expected: "期望效果",
  description: "问题描述",
  allow: "允许管理员修改",
  noAllow: "未允许管理员修改",
  noExpected: "未填写",
  userReply: "用户补充信息",
  noUserReply: "用户暂未补充。",
  adminReply: "管理员回复",
  noAdminReply: "暂无回复",
  replyPrompt: "请输入给用户的回复。",
  process: "标记处理中",
  needInfo: "需要用户补充信息",
  fixed: "标记已修复",
  reject: "标记无法处理",
  close: "关闭申请",
  openProject: "打开作品",
  showFinished: "显示已完成申请",
  hideFinished: "隐藏已完成申请",
  status: "状态",
  actions: "操作",
  tableRegion: "网页修复申请表格"
};

const issueLabels: Record<string, string> = {
  page_broken: "页面打不开或显示异常",
  button_broken: "按钮不能用",
  interactive_error: "互动功能出错",
  data_error: "数据保存或读取异常",
  encoding_error: "乱码问题",
  style_error: "样式问题",
  ai_code_error: "AI 生成的代码跑不通",
  other: "其他问题"
};

const cellStyle = { borderBottom: "1px solid var(--line)", padding: "10px 12px", textAlign: "left" as const, verticalAlign: "top" as const };

function statusLabel(status: string): string {
  switch (status) {
    case "pending": return "待处理";
    case "processing": return "处理中";
    case "need_info": return "等待用户补充";
    case "fixed": return "已修复";
    case "ai_fixed": return "AI 已修复";
    case "rejected": return "无法处理";
    case "closed": return "已关闭";
    default: return status;
  }
}

export function AdminRepairRequests() {
  const [items, setItems] = useState<RepairRequest[]>([]);
  const [statusText, setStatusText] = useState(text.loading);
  const [workingId, setWorkingId] = useState("");
  const [showFinished, setShowFinished] = useState(false);

  useEffect(() => { void loadItems(); }, []);

  async function loadItems() {
    try {
      const data = await getJSON<{ items: RepairRequest[] }>("/api/v1/admin/repair-requests");
      setItems(data.items);
      setStatusText(data.items.length === 0 ? text.empty : "");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : text.loadFail);
    }
  }

  async function updateRequest(item: RepairRequest, status: string, requireReply: boolean) {
    let adminReply = item.adminReply || "";
    if (requireReply) {
      const input = window.prompt(text.replyPrompt, adminReply);
      if (input === null || input.trim() === "") return;
      adminReply = input.trim();
    }
    try {
      setWorkingId(item.id);
      setStatusText(text.saving);
      await postJSON(`/api/v1/admin/repair-requests/${item.id}/review`, { status, adminReply });
      await loadItems();
      setStatusText(text.saveOk);
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : text.saveFail);
    } finally {
      setWorkingId("");
    }
  }

  const visibleItems = showFinished ? items : items.filter((item) => item.status === "pending" || item.status === "processing" || item.status === "need_info");

  return (
    <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-labelledby="admin-repairs-title">
      <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap", alignItems: "center" }}>
        <h2 id="admin-repairs-title" style={{ margin: 0 }}>{text.title}</h2>
        <button className="button-secondary" type="button" onClick={() => setShowFinished((value) => !value)}>{showFinished ? text.hideFinished : text.showFinished}</button>
      </div>
      <div className="status" aria-live="polite">{statusText || `当前显示 ${visibleItems.length} 条修复申请。`}</div>
      {visibleItems.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.empty}</p> : null}
      <div style={{ overflowX: "auto" }} role="region" aria-label={text.tableRegion}>
        <table style={{ width: "100%", minWidth: 980, borderCollapse: "collapse" }}>
          <thead><tr><th style={cellStyle}>{text.project}</th><th style={cellStyle}>{text.user}</th><th style={cellStyle}>{text.issue}</th><th style={cellStyle}>{text.description}</th><th style={cellStyle}>{text.expected}</th><th style={cellStyle}>{text.userReply}</th><th style={cellStyle}>{text.adminReply}</th><th style={cellStyle}>{text.status}</th><th style={cellStyle}>{text.actions}</th></tr></thead>
          <tbody>
            {visibleItems.map((item) => (
              <tr key={item.id}>
                <td style={cellStyle}><strong>{item.projectName}</strong><div><a href={item.projectPublicUrl} target="_blank" rel="noreferrer">{text.openProject}</a></div><div style={{ color: "var(--muted)", fontSize: "0.9rem" }}>{item.allowAdminEdit ? text.allow : text.noAllow}</div></td>
                <td style={cellStyle}>{item.ownerEmail || item.username}</td>
                <td style={cellStyle}>{issueLabels[item.issueType] ?? item.issueType}</td>
                <td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{item.description}</td>
                <td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{item.expected || text.noExpected}</td>
                <td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{item.userReply || text.noUserReply}</td>
                <td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{item.adminReply || text.noAdminReply}</td>
                <td style={cellStyle}>{statusLabel(item.status)}</td>
                <td style={cellStyle}><div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}><button className="button-secondary" type="button" disabled={workingId !== ""} onClick={() => updateRequest(item, "processing", false)}>{text.process}</button><button className="button-secondary" type="button" disabled={workingId !== ""} onClick={() => updateRequest(item, "need_info", true)}>{text.needInfo}</button><button className="button-primary" type="button" disabled={workingId !== ""} onClick={() => updateRequest(item, "fixed", true)}>{text.fixed}</button><button className="button-ghost" type="button" disabled={workingId !== ""} onClick={() => updateRequest(item, "rejected", true)}>{text.reject}</button><button className="button-ghost" type="button" disabled={workingId !== ""} onClick={() => updateRequest(item, "closed", false)}>{text.close}</button></div></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
