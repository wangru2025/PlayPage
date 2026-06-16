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
  title: "\u7f51\u9875\u4fee\u590d\u7533\u8bf7",
  loading: "\u6b63\u5728\u52a0\u8f7d\u4fee\u590d\u7533\u8bf7\u2026\u2026",
  empty: "\u5f53\u524d\u6ca1\u6709\u9700\u8981\u5904\u7406\u7684\u4fee\u590d\u7533\u8bf7\u3002",
  loadFail: "\u4fee\u590d\u7533\u8bf7\u52a0\u8f7d\u5931\u8d25\u3002",
  saving: "\u6b63\u5728\u4fdd\u5b58\u5904\u7406\u7ed3\u679c\u2026\u2026",
  saveFail: "\u5904\u7406\u7ed3\u679c\u4fdd\u5b58\u5931\u8d25\u3002",
  saveOk: "\u5904\u7406\u7ed3\u679c\u5df2\u7ecf\u4fdd\u5b58\u3002",
  user: "\u7528\u6237",
  project: "\u4f5c\u54c1",
  issue: "\u95ee\u9898\u7c7b\u578b",
  expected: "\u671f\u671b\u6548\u679c",
  description: "\u95ee\u9898\u63cf\u8ff0",
  allow: "\u5141\u8bb8\u7ba1\u7406\u5458\u4fee\u6539",
  noAllow: "\u672a\u5141\u8bb8\u7ba1\u7406\u5458\u4fee\u6539",
  noExpected: "\u672a\u586b\u5199",
  userReply: "\u7528\u6237\u8865\u5145\u4fe1\u606f",
  noUserReply: "\u7528\u6237\u6682\u672a\u8865\u5145\u3002",
  adminReply: "\u7ba1\u7406\u5458\u56de\u590d",
  noAdminReply: "\u6682\u65e0\u56de\u590d",
  replyPrompt: "\u8bf7\u8f93\u5165\u7ed9\u7528\u6237\u7684\u56de\u590d\u3002",
  process: "\u6807\u8bb0\u5904\u7406\u4e2d",
  needInfo: "\u9700\u8981\u7528\u6237\u8865\u5145\u4fe1\u606f",
  fixed: "\u6807\u8bb0\u5df2\u4fee\u590d",
  reject: "\u6807\u8bb0\u65e0\u6cd5\u5904\u7406",
  close: "\u5173\u95ed\u7533\u8bf7",
  openProject: "\u6253\u5f00\u4f5c\u54c1",
  showFinished: "\u663e\u793a\u5df2\u5b8c\u6210\u7533\u8bf7",
  hideFinished: "\u9690\u85cf\u5df2\u5b8c\u6210\u7533\u8bf7",
  status: "\u72b6\u6001",
  actions: "\u64cd\u4f5c",
  tableRegion: "\u7f51\u9875\u4fee\u590d\u7533\u8bf7\u8868\u683c"
};

const issueLabels: Record<string, string> = {
  page_broken: "\u9875\u9762\u6253\u4e0d\u5f00\u6216\u663e\u793a\u5f02\u5e38",
  button_broken: "\u6309\u94ae\u4e0d\u80fd\u7528",
  interactive_error: "\u4e92\u52a8\u529f\u80fd\u51fa\u9519",
  data_error: "\u6570\u636e\u4fdd\u5b58\u6216\u8bfb\u53d6\u5f02\u5e38",
  encoding_error: "\u4e71\u7801\u95ee\u9898",
  style_error: "\u6837\u5f0f\u95ee\u9898",
  ai_code_error: "AI \u751f\u6210\u7684\u4ee3\u7801\u8dd1\u4e0d\u901a",
  other: "\u5176\u4ed6\u95ee\u9898"
};

const cellStyle = { borderBottom: "1px solid var(--line)", padding: "10px 12px", textAlign: "left" as const, verticalAlign: "top" as const };

function statusLabel(status: string): string {
  switch (status) {
    case "pending": return "\u5f85\u5904\u7406";
    case "processing": return "\u5904\u7406\u4e2d";
    case "need_info": return "\u7b49\u5f85\u7528\u6237\u8865\u5145";
    case "fixed": return "\u5df2\u4fee\u590d";
    case "rejected": return "\u65e0\u6cd5\u5904\u7406";
    case "closed": return "\u5df2\u5173\u95ed";
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
      <div className="status" aria-live="polite">{statusText || `\u5f53\u524d\u663e\u793a ${visibleItems.length} \u6761\u4fee\u590d\u7533\u8bf7\u3002`}</div>
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
