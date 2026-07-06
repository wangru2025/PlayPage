"use client";

import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";

type ContestSubmission = {
  id: string;
  userEmail: string;
  username: string;
  projectName: string;
  projectUrl: string;
  intro: string;
  story: string;
  allowShowcase: boolean;
  status: string;
  adminNote: string;
  createdAt: string;
};

type ContestResponse = {
  items: ContestSubmission[];
};

const statusText: Record<string, string> = {
  pending: "待评选",
  shortlisted: "已入围",
  winner: "已获奖",
  rejected: "不入选"
};

const tableWrapStyle = { overflowX: "auto" as const };
const tableStyle = { width: "100%", borderCollapse: "collapse" as const, minWidth: 980 };
const cellStyle = { borderBottom: "1px solid var(--line)", padding: "10px 12px", textAlign: "left" as const, verticalAlign: "top" as const };

export default function AdminContestPage() {
  const [items, setItems] = useState<ContestSubmission[]>([]);
  const [showRejected, setShowRejected] = useState(false);
  const [notes, setNotes] = useState<Record<string, string>>({});
  const [statusTextValue, setStatusTextValue] = useState("正在读取参赛作品。");
  const [statusTone, setStatusTone] = useState<"info" | "success" | "error">("info");
  const [workingId, setWorkingId] = useState("");

  useEffect(() => {
    void load();
  }, []);

  function setStatus(message: string, tone: "info" | "success" | "error") {
    setStatusTextValue(message);
    setStatusTone(tone);
  }

  async function load() {
    try {
      const payload = await getJSON<ContestResponse>("/api/v1/admin/contest-submissions");
      setItems(payload.items);
      const nextNotes: Record<string, string> = {};
      for (const item of payload.items) nextNotes[item.id] = item.adminNote || "";
      setNotes(nextNotes);
      setStatus("参赛作品已读取。", "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "读取参赛作品失败。", "error");
    }
  }

  async function updateStatus(item: ContestSubmission, status: string) {
    let adminNote = notes[item.id] ?? "";
    if (status === "rejected" && adminNote.trim() === "") {
      const reason = window.prompt("请输入不入选原因，这段内容会通过邮件发给用户。");
      if (!reason || reason.trim() === "") {
        setStatus("不入选时需要填写原因。", "error");
        return;
      }
      adminNote = reason.trim();
      setNotes((current) => ({ ...current, [item.id]: adminNote }));
    }
    try {
      setWorkingId(item.id);
      setStatus("正在保存参赛状态。", "info");
      await postJSON(`/api/v1/admin/contest-submissions/${item.id}/review`, {
        status,
        adminNote
      });
      await load();
      setStatus("参赛状态已保存。", "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "保存失败。", "error");
    } finally {
      setWorkingId("");
    }
  }

  const visibleItems = showRejected ? items : items.filter((item) => item.status !== "rejected");

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>管理员后台</p>
        <h1 style={{ margin: 0, fontSize: "2.3rem" }}>创作比赛管理</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>查看用户提交的参赛作品，标记入围、获奖或不入选。</p>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <button className="button-secondary" type="button" onClick={() => setShowRejected((value) => !value)}>
            {showRejected ? "隐藏不入选项目" : "显示不入选项目"}
          </button>
          <button className="button-secondary" type="button" onClick={load}>刷新</button>
        </div>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite">
          {statusTextValue}
        </div>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        {visibleItems.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>暂无参赛作品。</p> : null}
        <div style={tableWrapStyle} role="region" aria-label="参赛作品表格">
          <table style={tableStyle}>
            <thead>
              <tr>
                <th style={cellStyle}>作品</th>
                <th style={cellStyle}>用户</th>
                <th style={cellStyle}>介绍</th>
                <th style={cellStyle}>创作故事</th>
                <th style={cellStyle}>状态</th>
                <th style={cellStyle}>管理员备注</th>
                <th style={cellStyle}>操作</th>
              </tr>
            </thead>
            <tbody>
              {visibleItems.map((item) => (
                <tr key={item.id}>
                  <td style={cellStyle}>
                    <strong>{item.projectName}</strong>
                    <br />
                    <a href={item.projectUrl} target="_blank" rel="noreferrer">打开作品</a>
                    <br />
                    <span style={{ color: "var(--muted)" }}>{new Date(item.createdAt).toLocaleString()}</span>
                  </td>
                  <td style={cellStyle}>
                    {item.username || "未设置"}
                    <br />
                    <span style={{ color: "var(--muted)" }}>{item.userEmail}</span>
                    <br />
                    <span style={{ color: "var(--muted)" }}>{item.allowShowcase ? "允许展示" : "不允许展示"}</span>
                  </td>
                  <td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{item.intro}</td>
                  <td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{item.story || "未填写"}</td>
                  <td style={cellStyle}>{statusText[item.status] || item.status}</td>
                  <td style={cellStyle}>
                    <textarea
                      rows={4}
                      value={notes[item.id] ?? ""}
                      onChange={(event) => setNotes((current) => ({ ...current, [item.id]: event.target.value }))}
                      aria-label={`${item.projectName} 的管理员备注`}
                    />
                  </td>
                  <td style={cellStyle}>
                    <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                      <button className="button-secondary" type="button" disabled={workingId === item.id} onClick={() => updateStatus(item, "pending")}>待定</button>
                      <button className="button-secondary" type="button" disabled={workingId === item.id} onClick={() => updateStatus(item, "shortlisted")}>入围</button>
                      <button className="button-primary" type="button" disabled={workingId === item.id} onClick={() => updateStatus(item, "winner")}>获奖</button>
                      <button className="button-ghost" type="button" disabled={workingId === item.id} onClick={() => updateStatus(item, "rejected")}>不入选</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </section>
  );
}
