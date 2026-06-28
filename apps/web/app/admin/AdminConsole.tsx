"use client";

import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";
import { AdminRepairRequests } from "./AdminRepairRequests";
import { AdminDomainDeleteRequests } from "./AdminDomainDeleteRequests";
import { AdminTemplateSubmissions } from "./AdminTemplateSubmissions";

type AdminUser = { id: string; email: string; username: string; status: string; role: string; planCode: string; projectCount: number; createdAt: string; };
type UserUpdatePayload = { role: string; planCode: string; };
type AdminProject = { id: string; name: string; slug: string; username: string; ownerUserId: string; ownerEmail: string; interactive: boolean; visibility: string; currentReleaseId: string; publicUrl: string; createdAt: string; };
type UpgradeRequest = { id: string; userId: string; userEmail: string; currentPlan: string; targetPlan: string; paymentMethod: string; payerNote: string; status: string; adminNote: string; reviewedBy: string; createdAt: string; };
type ProjectDomainRequest = { id: string; projectId: string; projectName: string; projectPublicUrl: string; ownerUserId: string; ownerEmail: string; username: string; subdomain: string; domain: string; status: string; rejectReason: string; adminNote: string; reviewedBy: string; createdAt: string; };
type AdminSection = "repairs" | "templates" | "upgrades" | "domains" | "domainDeletes" | "users" | "projects";

const text = {
  title: "管理后台",
  intro: "在这里处理网页修复、套餐升级、独立网址、用户和作品。默认隐藏已经完成的项目。",
  loading: "正在加载管理数据……",
  needAdmin: "需要管理员权限或请稍后重试。",
  users: "用户管理",
  projects: "作品管理",
  upgrades: "升级申请",
  repairs: "网页修复申请",
  templates: "模板投稿审核",
  role: "角色",
  plan: "套餐",
  owner: "作者",
  visibility: "公开状态",
  interactive: "互动功能",
  published: "已发布",
  noRelease: "未发布",
  yes: "是",
  no: "否",
  approve: "通过升级",
  reject: "驳回",
  reviewing: "正在处理申请……",
  saveUser: "保存用户设置",
  savingUser: "正在保存用户设置……",
  domains: "独立网址审核",
  domainDeletes: "删除独立网址申请",
  domainReviewing: "正在处理独立网址申请……",
  domainApprove: "标记为已通过",
  domainReject: "驳回申请",
  domainManualSteps: "通过后请按申请内容完成独立网址配置，并在这里记录审核状态。",
  rejectReasonAsk: "请输入驳回原因。",
  noDomainRequests: "当前没有需要处理的独立网址申请。",
  noUpgradeRequests: "当前没有需要处理的升级申请。",
  noUsers: "当前没有用户。",
  noProjects: "当前没有作品。",
  showFinished: "显示已完成项目",
  hideFinished: "隐藏已完成项目",
  actions: "操作",
  status: "状态",
  createdAt: "创建时间",
  paymentMethod: "支付方式",
  payerNote: "付款备注",
  sourcePlan: "当前套餐",
  targetPlan: "目标套餐",
  openProject: "打开作品",
  tableRegion: "表格"
};

const sections: Array<{ id: AdminSection; label: string }> = [
  { id: "repairs", label: text.repairs }, { id: "templates", label: text.templates }, { id: "upgrades", label: text.upgrades }, { id: "domains", label: text.domains }, { id: "domainDeletes", label: text.domainDeletes }, { id: "users", label: text.users }, { id: "projects", label: text.projects }
];

const tableWrapStyle = { overflowX: "auto" as const };
const tableStyle = { width: "100%", borderCollapse: "collapse" as const, minWidth: 760 };
const cellStyle = { borderBottom: "1px solid var(--line)", padding: "10px 12px", textAlign: "left" as const, verticalAlign: "top" as const };

function roleLabel(role: string): string { switch (role) { case "super_admin": return "超级管理员"; case "admin": return "管理员"; default: return "普通用户"; } }
function planLabel(planCode: string): string { switch (planCode) { case "light": return "轻量版"; case "support": return "支持者"; case "admin": return "管理员套餐"; default: return "免费版"; } }
function requestStatusLabel(status: string): string { switch (status) { case "pending": return "待处理"; case "approved": return "已通过"; case "rejected": return "已驳回"; default: return status; } }
function domainStatusLabel(status: string): string { switch (status) { case "pending": return "待处理"; case "active": return "已通过"; case "rejected": return "已驳回"; default: return status; } }
function formatDate(value: string): string { if (!value) return "未知"; const date = new Date(value); if (Number.isNaN(date.getTime())) return value; return date.toLocaleString("zh-CN", { hour12: false }); }

export function AdminConsole() {
  const [activeSection, setActiveSection] = useState<AdminSection>("repairs");
  const [loading, setLoading] = useState(true);
  const [statusText, setStatusText] = useState(text.loading);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [projects, setProjects] = useState<AdminProject[]>([]);
  const [requests, setRequests] = useState<UpgradeRequest[]>([]);
  const [domainRequests, setDomainRequests] = useState<ProjectDomainRequest[]>([]);
  const [editingUsers, setEditingUsers] = useState<Record<string, UserUpdatePayload>>({});
  const [showFinishedRequests, setShowFinishedRequests] = useState(false);
  const [showFinishedDomains, setShowFinishedDomains] = useState(false);

  useEffect(() => { async function loadData() { try { const [userData, projectData, requestData, domainData] = await Promise.all([ getJSON<{ items: AdminUser[] }>("/api/v1/admin/users"), getJSON<{ items: AdminProject[] }>("/api/v1/admin/projects"), getJSON<{ items: UpgradeRequest[] }>("/api/v1/admin/upgrade-requests"), getJSON<{ items: ProjectDomainRequest[] }>("/api/v1/admin/project-domains") ]); setUsers(userData.items); setProjects(projectData.items); setRequests(requestData.items); setDomainRequests(domainData.items); setStatusText("管理数据加载完成。"); } catch (error) { setStatusText(error instanceof Error ? error.message : text.needAdmin); } finally { setLoading(false); } } void loadData(); }, []);

  async function reviewRequest(requestId: string, status: "approved" | "rejected") { setStatusText(text.reviewing); try { await postJSON(`/api/v1/admin/upgrade-requests/${requestId}/review`, { status }); const latest = await getJSON<{ items: UpgradeRequest[] }>("/api/v1/admin/upgrade-requests"); setRequests(latest.items); setStatusText("申请处理完成。"); } catch (error) { setStatusText(error instanceof Error ? error.message : text.needAdmin); } }
  async function reviewProjectDomain(requestId: string, status: "active" | "rejected") { const rejectReason = status === "rejected" ? window.prompt(text.rejectReasonAsk) : ""; if (status === "rejected" && (!rejectReason || rejectReason.trim() === "")) return; setStatusText(text.domainReviewing); try { await postJSON(`/api/v1/admin/project-domains/${requestId}/review`, { status, rejectReason: rejectReason ?? "", adminNote: status === "active" ? "审核已通过" : "" }); const latest = await getJSON<{ items: ProjectDomainRequest[] }>("/api/v1/admin/project-domains"); setDomainRequests(latest.items); setStatusText("申请处理完成。"); } catch (error) { setStatusText(error instanceof Error ? error.message : text.needAdmin); } }
  async function saveUserSettings(userId: string) { const payload = editingUsers[userId]; if (!payload) return; setStatusText(text.savingUser); try { await postJSON(`/api/v1/admin/users/${userId}`, payload); const latest = await getJSON<{ items: AdminUser[] }>("/api/v1/admin/users"); setUsers(latest.items); setStatusText("用户设置已保存。"); } catch (error) { setStatusText(error instanceof Error ? error.message : text.needAdmin); } }

  const visibleRequests = showFinishedRequests ? requests : requests.filter((request) => request.status === "pending");
  const visibleDomainRequests = showFinishedDomains ? domainRequests : domainRequests.filter((request) => request.status === "pending");

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 14 }}>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.title}</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>{text.intro}</p>
        <nav aria-label="管理后台页面" style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          {sections.map((section) => (<button key={section.id} className={activeSection === section.id ? "button-primary" : "button-secondary"} type="button" aria-current={activeSection === section.id ? "page" : undefined} onClick={() => setActiveSection(section.id)}>{section.label}</button>))}
        </nav>
        <div className="status" aria-live="polite">{loading ? text.loading : statusText}</div>
      </header>

      {activeSection === "repairs" ? <AdminRepairRequests /> : null}

      {activeSection === "templates" ? <AdminTemplateSubmissions /> : null}

      {activeSection === "domainDeletes" ? <AdminDomainDeleteRequests /> : null}

      {activeSection === "upgrades" ? (<section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-labelledby="admin-upgrades-title"><div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap", alignItems: "center" }}><h2 id="admin-upgrades-title" style={{ margin: 0 }}>{text.upgrades}</h2><button className="button-secondary" type="button" onClick={() => setShowFinishedRequests((value) => !value)}>{showFinishedRequests ? text.hideFinished : text.showFinished}</button></div>{visibleRequests.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.noUpgradeRequests}</p> : null}<div style={tableWrapStyle} role="region" aria-label={`${text.upgrades}${text.tableRegion}`}><table style={tableStyle}><thead><tr><th style={cellStyle}>用户</th><th style={cellStyle}>{text.sourcePlan}</th><th style={cellStyle}>{text.targetPlan}</th><th style={cellStyle}>{text.paymentMethod}</th><th style={cellStyle}>{text.payerNote}</th><th style={cellStyle}>{text.status}</th><th style={cellStyle}>{text.actions}</th></tr></thead><tbody>{visibleRequests.map((request) => (<tr key={request.id}><td style={cellStyle}>{request.userEmail}</td><td style={cellStyle}>{planLabel(request.currentPlan)}</td><td style={cellStyle}>{planLabel(request.targetPlan)}</td><td style={cellStyle}>{request.paymentMethod === "wechat" ? "微信" : "其他"}</td><td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{request.payerNote || "未填写"}</td><td style={cellStyle}>{requestStatusLabel(request.status)}</td><td style={cellStyle}>{request.status === "pending" ? (<div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}><button className="button-primary" type="button" onClick={() => reviewRequest(request.id, "approved")}>{text.approve}</button><button className="button-ghost" type="button" onClick={() => reviewRequest(request.id, "rejected")}>{text.reject}</button></div>) : "已处理"}</td></tr>))}</tbody></table></div></section>) : null}

      {activeSection === "domains" ? (<section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-labelledby="admin-domains-title"><div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap", alignItems: "center" }}><h2 id="admin-domains-title" style={{ margin: 0 }}>{text.domains}</h2><button className="button-secondary" type="button" onClick={() => setShowFinishedDomains((value) => !value)}>{showFinishedDomains ? text.hideFinished : text.showFinished}</button></div><p style={{ margin: 0, color: "var(--muted)" }}>{text.domainManualSteps}</p>{visibleDomainRequests.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.noDomainRequests}</p> : null}<div style={tableWrapStyle} role="region" aria-label={`${text.domains}${text.tableRegion}`}><table style={tableStyle}><thead><tr><th style={cellStyle}>申请网址</th><th style={cellStyle}>用户</th><th style={cellStyle}>作品</th><th style={cellStyle}>原作品地址</th><th style={cellStyle}>{text.status}</th><th style={cellStyle}>驳回原因</th><th style={cellStyle}>{text.actions}</th></tr></thead><tbody>{visibleDomainRequests.map((item) => (<tr key={item.id}><td style={cellStyle}>{item.domain}</td><td style={cellStyle}>{item.ownerEmail || item.username}</td><td style={cellStyle}>{item.projectName}</td><td style={cellStyle}><a href={item.projectPublicUrl} target="_blank" rel="noreferrer">{item.projectPublicUrl}</a></td><td style={cellStyle}>{domainStatusLabel(item.status)}</td><td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{item.rejectReason || "-"}</td><td style={cellStyle}>{item.status === "pending" ? (<div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}><button className="button-primary" type="button" onClick={() => reviewProjectDomain(item.id, "active")}>{text.domainApprove}</button><button className="button-ghost" type="button" onClick={() => reviewProjectDomain(item.id, "rejected")}>{text.domainReject}</button></div>) : "已处理"}</td></tr>))}</tbody></table></div></section>) : null}

      {activeSection === "users" ? (<section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-labelledby="admin-users-title"><h2 id="admin-users-title" style={{ margin: 0 }}>{text.users}</h2>{users.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.noUsers}</p> : null}<div style={tableWrapStyle} role="region" aria-label={`${text.users}${text.tableRegion}`}><table style={tableStyle}><thead><tr><th style={cellStyle}>用户名</th><th style={cellStyle}>邮箱</th><th style={cellStyle}>{text.role}</th><th style={cellStyle}>{text.plan}</th><th style={cellStyle}>作品数</th><th style={cellStyle}>{text.createdAt}</th><th style={cellStyle}>{text.actions}</th></tr></thead><tbody>{users.map((user) => (<tr key={user.id}><td style={cellStyle}>{user.username || "未设置"}</td><td style={cellStyle}>{user.email}</td><td style={cellStyle}><label className="field" style={{ gap: 4 }}><span className="sr-only">{user.email} 的角色</span><select value={editingUsers[user.id]?.role ?? user.role} onChange={(event) => setEditingUsers((current) => ({ ...current, [user.id]: { role: event.target.value, planCode: current[user.id]?.planCode ?? user.planCode } }))}><option value="user">普通用户</option><option value="admin">管理员</option><option value="super_admin">超级管理员</option></select></label><span style={{ color: "var(--muted)", fontSize: "0.9rem" }}>当前：{roleLabel(user.role)}</span></td><td style={cellStyle}><label className="field" style={{ gap: 4 }}><span className="sr-only">{user.email} 的套餐</span><select value={editingUsers[user.id]?.planCode ?? user.planCode} onChange={(event) => setEditingUsers((current) => ({ ...current, [user.id]: { role: current[user.id]?.role ?? user.role, planCode: event.target.value } }))}><option value="free">免费版</option><option value="light">轻量版</option><option value="support">支持者</option><option value="admin">管理员套餐</option></select></label><span style={{ color: "var(--muted)", fontSize: "0.9rem" }}>当前：{planLabel(user.planCode)}</span></td><td style={cellStyle}>{user.projectCount}</td><td style={cellStyle}>{formatDate(user.createdAt)}</td><td style={cellStyle}><button className="button-secondary" type="button" onClick={() => saveUserSettings(user.id)}>{text.saveUser}</button></td></tr>))}</tbody></table></div></section>) : null}

      {activeSection === "projects" ? (<section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-labelledby="admin-projects-title"><h2 id="admin-projects-title" style={{ margin: 0 }}>{text.projects}</h2>{projects.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.noProjects}</p> : null}<div style={tableWrapStyle} role="region" aria-label={`${text.projects}${text.tableRegion}`}><table style={tableStyle}><thead><tr><th style={cellStyle}>作品</th><th style={cellStyle}>{text.owner}</th><th style={cellStyle}>地址</th><th style={cellStyle}>{text.visibility}</th><th style={cellStyle}>{text.interactive}</th><th style={cellStyle}>发布状态</th></tr></thead><tbody>{projects.map((project) => (<tr key={project.id}><td style={cellStyle}>{project.name}</td><td style={cellStyle}>{project.ownerEmail}</td><td style={cellStyle}><a href={project.publicUrl} target="_blank" rel="noreferrer">{project.publicUrl}</a></td><td style={cellStyle}>{project.visibility === "public" ? "公开" : "不公开"}</td><td style={cellStyle}>{project.interactive ? text.yes : text.no}</td><td style={cellStyle}>{project.currentReleaseId ? text.published : text.noRelease}</td></tr>))}</tbody></table></div></section>) : null}
    </section>
  );
}
