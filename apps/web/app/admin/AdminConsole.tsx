"use client";

import { useEffect, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";
import { AdminRepairRequests } from "./AdminRepairRequests";

type AdminUser = { id: string; email: string; username: string; status: string; role: string; planCode: string; projectCount: number; createdAt: string; };
type UserUpdatePayload = { role: string; planCode: string; };
type AdminProject = { id: string; name: string; slug: string; username: string; ownerUserId: string; ownerEmail: string; interactive: boolean; visibility: string; currentReleaseId: string; publicUrl: string; createdAt: string; };
type UpgradeRequest = { id: string; userId: string; userEmail: string; currentPlan: string; targetPlan: string; paymentMethod: string; payerNote: string; status: string; adminNote: string; reviewedBy: string; createdAt: string; };
type ProjectDomainRequest = { id: string; projectId: string; projectName: string; projectPublicUrl: string; ownerUserId: string; ownerEmail: string; username: string; subdomain: string; domain: string; status: string; rejectReason: string; adminNote: string; reviewedBy: string; createdAt: string; };
type AdminSection = "repairs" | "upgrades" | "domains" | "users" | "projects";

const text = {
  title: "\u7ba1\u7406\u540e\u53f0",
  intro: "\u5728\u8fd9\u91cc\u5904\u7406\u7f51\u9875\u4fee\u590d\u3001\u5957\u9910\u5347\u7ea7\u3001\u72ec\u7acb\u7f51\u5740\u3001\u7528\u6237\u548c\u4f5c\u54c1\u3002\u9ed8\u8ba4\u9690\u85cf\u5df2\u7ecf\u5b8c\u6210\u7684\u9879\u76ee\u3002",
  loading: "\u6b63\u5728\u52a0\u8f7d\u7ba1\u7406\u6570\u636e\u2026\u2026",
  needAdmin: "\u9700\u8981\u7ba1\u7406\u5458\u6743\u9650\u6216\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002",
  users: "\u7528\u6237\u7ba1\u7406",
  projects: "\u4f5c\u54c1\u7ba1\u7406",
  upgrades: "\u5347\u7ea7\u7533\u8bf7",
  repairs: "\u7f51\u9875\u4fee\u590d\u7533\u8bf7",
  back: "\u8fd4\u56de\u4e2a\u4eba\u4e2d\u5fc3",
  role: "\u89d2\u8272",
  plan: "\u5957\u9910",
  owner: "\u4f5c\u8005",
  visibility: "\u516c\u5f00\u72b6\u6001",
  interactive: "\u4e92\u52a8\u529f\u80fd",
  published: "\u5df2\u53d1\u5e03",
  noRelease: "\u672a\u53d1\u5e03",
  yes: "\u662f",
  no: "\u5426",
  approve: "\u901a\u8fc7\u5347\u7ea7",
  reject: "\u9a73\u56de",
  reviewing: "\u6b63\u5728\u5904\u7406\u7533\u8bf7\u2026\u2026",
  saveUser: "\u4fdd\u5b58\u7528\u6237\u8bbe\u7f6e",
  savingUser: "\u6b63\u5728\u4fdd\u5b58\u7528\u6237\u8bbe\u7f6e\u2026\u2026",
  domains: "\u72ec\u7acb\u7f51\u5740\u5ba1\u6838",
  domainReviewing: "\u6b63\u5728\u5904\u7406\u72ec\u7acb\u7f51\u5740\u7533\u8bf7\u2026\u2026",
  domainApprove: "\u6807\u8bb0\u4e3a\u5df2\u901a\u8fc7",
  domainReject: "\u9a73\u56de\u7533\u8bf7",
  domainManualSteps: "\u901a\u8fc7\u540e\u8bf7\u6309\u7533\u8bf7\u5185\u5bb9\u5b8c\u6210\u72ec\u7acb\u7f51\u5740\u914d\u7f6e\uff0c\u5e76\u5728\u8fd9\u91cc\u8bb0\u5f55\u5ba1\u6838\u72b6\u6001\u3002",
  rejectReasonAsk: "\u8bf7\u8f93\u5165\u9a73\u56de\u539f\u56e0\u3002",
  noDomainRequests: "\u5f53\u524d\u6ca1\u6709\u9700\u8981\u5904\u7406\u7684\u72ec\u7acb\u7f51\u5740\u7533\u8bf7\u3002",
  noUpgradeRequests: "\u5f53\u524d\u6ca1\u6709\u9700\u8981\u5904\u7406\u7684\u5347\u7ea7\u7533\u8bf7\u3002",
  noUsers: "\u5f53\u524d\u6ca1\u6709\u7528\u6237\u3002",
  noProjects: "\u5f53\u524d\u6ca1\u6709\u4f5c\u54c1\u3002",
  showFinished: "\u663e\u793a\u5df2\u5b8c\u6210\u9879\u76ee",
  hideFinished: "\u9690\u85cf\u5df2\u5b8c\u6210\u9879\u76ee",
  actions: "\u64cd\u4f5c",
  status: "\u72b6\u6001",
  createdAt: "\u521b\u5efa\u65f6\u95f4",
  paymentMethod: "\u652f\u4ed8\u65b9\u5f0f",
  payerNote: "\u4ed8\u6b3e\u5907\u6ce8",
  sourcePlan: "\u5f53\u524d\u5957\u9910",
  targetPlan: "\u76ee\u6807\u5957\u9910",
  openProject: "\u6253\u5f00\u4f5c\u54c1",
  tableRegion: "\u8868\u683c"
};

const sections: Array<{ id: AdminSection; label: string }> = [
  { id: "repairs", label: text.repairs }, { id: "upgrades", label: text.upgrades }, { id: "domains", label: text.domains }, { id: "users", label: text.users }, { id: "projects", label: text.projects }
];

const tableWrapStyle = { overflowX: "auto" as const };
const tableStyle = { width: "100%", borderCollapse: "collapse" as const, minWidth: 760 };
const cellStyle = { borderBottom: "1px solid var(--line)", padding: "10px 12px", textAlign: "left" as const, verticalAlign: "top" as const };

function roleLabel(role: string): string { switch (role) { case "super_admin": return "\u8d85\u7ea7\u7ba1\u7406\u5458"; case "admin": return "\u7ba1\u7406\u5458"; default: return "\u666e\u901a\u7528\u6237"; } }
function planLabel(planCode: string): string { switch (planCode) { case "light": return "\u8f7b\u91cf\u7248"; case "support": return "\u652f\u6301\u8005"; case "admin": return "\u7ba1\u7406\u5458\u5957\u9910"; default: return "\u514d\u8d39\u7248"; } }
function requestStatusLabel(status: string): string { switch (status) { case "pending": return "\u5f85\u5904\u7406"; case "approved": return "\u5df2\u901a\u8fc7"; case "rejected": return "\u5df2\u9a73\u56de"; default: return status; } }
function domainStatusLabel(status: string): string { switch (status) { case "pending": return "\u5f85\u5904\u7406"; case "active": return "\u5df2\u901a\u8fc7"; case "rejected": return "\u5df2\u9a73\u56de"; default: return status; } }
function formatDate(value: string): string { if (!value) return "\u672a\u77e5"; const date = new Date(value); if (Number.isNaN(date.getTime())) return value; return date.toLocaleString("zh-CN", { hour12: false }); }

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

  useEffect(() => { async function loadData() { try { const [userData, projectData, requestData, domainData] = await Promise.all([ getJSON<{ items: AdminUser[] }>("/api/v1/admin/users"), getJSON<{ items: AdminProject[] }>("/api/v1/admin/projects"), getJSON<{ items: UpgradeRequest[] }>("/api/v1/admin/upgrade-requests"), getJSON<{ items: ProjectDomainRequest[] }>("/api/v1/admin/project-domains") ]); setUsers(userData.items); setProjects(projectData.items); setRequests(requestData.items); setDomainRequests(domainData.items); setStatusText("\u7ba1\u7406\u6570\u636e\u52a0\u8f7d\u5b8c\u6210\u3002"); } catch (error) { setStatusText(error instanceof Error ? error.message : text.needAdmin); } finally { setLoading(false); } } void loadData(); }, []);

  async function reviewRequest(requestId: string, status: "approved" | "rejected") { setStatusText(text.reviewing); try { await postJSON(`/api/v1/admin/upgrade-requests/${requestId}/review`, { status }); const latest = await getJSON<{ items: UpgradeRequest[] }>("/api/v1/admin/upgrade-requests"); setRequests(latest.items); setStatusText("\u7533\u8bf7\u5904\u7406\u5b8c\u6210\u3002"); } catch (error) { setStatusText(error instanceof Error ? error.message : text.needAdmin); } }
  async function reviewProjectDomain(requestId: string, status: "active" | "rejected") { const rejectReason = status === "rejected" ? window.prompt(text.rejectReasonAsk) : ""; if (status === "rejected" && (!rejectReason || rejectReason.trim() === "")) return; setStatusText(text.domainReviewing); try { await postJSON(`/api/v1/admin/project-domains/${requestId}/review`, { status, rejectReason: rejectReason ?? "", adminNote: status === "active" ? "\u5ba1\u6838\u5df2\u901a\u8fc7" : "" }); const latest = await getJSON<{ items: ProjectDomainRequest[] }>("/api/v1/admin/project-domains"); setDomainRequests(latest.items); setStatusText("\u7533\u8bf7\u5904\u7406\u5b8c\u6210\u3002"); } catch (error) { setStatusText(error instanceof Error ? error.message : text.needAdmin); } }
  async function saveUserSettings(userId: string) { const payload = editingUsers[userId]; if (!payload) return; setStatusText(text.savingUser); try { await postJSON(`/api/v1/admin/users/${userId}`, payload); const latest = await getJSON<{ items: AdminUser[] }>("/api/v1/admin/users"); setUsers(latest.items); setStatusText("\u7528\u6237\u8bbe\u7f6e\u5df2\u4fdd\u5b58\u3002"); } catch (error) { setStatusText(error instanceof Error ? error.message : text.needAdmin); } }

  const visibleRequests = showFinishedRequests ? requests : requests.filter((request) => request.status === "pending");
  const visibleDomainRequests = showFinishedDomains ? domainRequests : domainRequests.filter((request) => request.status === "pending");

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 14 }}>
        <div><a className="button-secondary" href="/me">{text.back}</a></div>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.title}</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>{text.intro}</p>
        <nav aria-label="\u7ba1\u7406\u540e\u53f0\u9875\u9762" style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          {sections.map((section) => (<button key={section.id} className={activeSection === section.id ? "button-primary" : "button-secondary"} type="button" aria-current={activeSection === section.id ? "page" : undefined} onClick={() => setActiveSection(section.id)}>{section.label}</button>))}
        </nav>
        <div className="status" aria-live="polite">{loading ? text.loading : statusText}</div>
      </header>

      {activeSection === "repairs" ? <AdminRepairRequests /> : null}

      {activeSection === "upgrades" ? (<section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-labelledby="admin-upgrades-title"><div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap", alignItems: "center" }}><h2 id="admin-upgrades-title" style={{ margin: 0 }}>{text.upgrades}</h2><button className="button-secondary" type="button" onClick={() => setShowFinishedRequests((value) => !value)}>{showFinishedRequests ? text.hideFinished : text.showFinished}</button></div>{visibleRequests.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.noUpgradeRequests}</p> : null}<div style={tableWrapStyle} role="region" aria-label={`${text.upgrades}${text.tableRegion}`}><table style={tableStyle}><thead><tr><th style={cellStyle}>\u7528\u6237</th><th style={cellStyle}>{text.sourcePlan}</th><th style={cellStyle}>{text.targetPlan}</th><th style={cellStyle}>{text.paymentMethod}</th><th style={cellStyle}>{text.payerNote}</th><th style={cellStyle}>{text.status}</th><th style={cellStyle}>{text.actions}</th></tr></thead><tbody>{visibleRequests.map((request) => (<tr key={request.id}><td style={cellStyle}>{request.userEmail}</td><td style={cellStyle}>{planLabel(request.currentPlan)}</td><td style={cellStyle}>{planLabel(request.targetPlan)}</td><td style={cellStyle}>{request.paymentMethod === "wechat" ? "\u5fae\u4fe1" : "\u5176\u4ed6"}</td><td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{request.payerNote || "\u672a\u586b\u5199"}</td><td style={cellStyle}>{requestStatusLabel(request.status)}</td><td style={cellStyle}>{request.status === "pending" ? (<div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}><button className="button-primary" type="button" onClick={() => reviewRequest(request.id, "approved")}>{text.approve}</button><button className="button-ghost" type="button" onClick={() => reviewRequest(request.id, "rejected")}>{text.reject}</button></div>) : "\u5df2\u5904\u7406"}</td></tr>))}</tbody></table></div></section>) : null}

      {activeSection === "domains" ? (<section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-labelledby="admin-domains-title"><div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap", alignItems: "center" }}><h2 id="admin-domains-title" style={{ margin: 0 }}>{text.domains}</h2><button className="button-secondary" type="button" onClick={() => setShowFinishedDomains((value) => !value)}>{showFinishedDomains ? text.hideFinished : text.showFinished}</button></div><p style={{ margin: 0, color: "var(--muted)" }}>{text.domainManualSteps}</p>{visibleDomainRequests.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.noDomainRequests}</p> : null}<div style={tableWrapStyle} role="region" aria-label={`${text.domains}${text.tableRegion}`}><table style={tableStyle}><thead><tr><th style={cellStyle}>\u7533\u8bf7\u7f51\u5740</th><th style={cellStyle}>\u7528\u6237</th><th style={cellStyle}>\u4f5c\u54c1</th><th style={cellStyle}>\u539f\u4f5c\u54c1\u5730\u5740</th><th style={cellStyle}>{text.status}</th><th style={cellStyle}>\u9a73\u56de\u539f\u56e0</th><th style={cellStyle}>{text.actions}</th></tr></thead><tbody>{visibleDomainRequests.map((item) => (<tr key={item.id}><td style={cellStyle}>{item.domain}</td><td style={cellStyle}>{item.ownerEmail || item.username}</td><td style={cellStyle}>{item.projectName}</td><td style={cellStyle}><a href={item.projectPublicUrl} target="_blank" rel="noreferrer">{item.projectPublicUrl}</a></td><td style={cellStyle}>{domainStatusLabel(item.status)}</td><td style={{ ...cellStyle, whiteSpace: "pre-wrap" }}>{item.rejectReason || "-"}</td><td style={cellStyle}>{item.status === "pending" ? (<div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}><button className="button-primary" type="button" onClick={() => reviewProjectDomain(item.id, "active")}>{text.domainApprove}</button><button className="button-ghost" type="button" onClick={() => reviewProjectDomain(item.id, "rejected")}>{text.domainReject}</button></div>) : "\u5df2\u5904\u7406"}</td></tr>))}</tbody></table></div></section>) : null}

      {activeSection === "users" ? (<section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-labelledby="admin-users-title"><h2 id="admin-users-title" style={{ margin: 0 }}>{text.users}</h2>{users.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.noUsers}</p> : null}<div style={tableWrapStyle} role="region" aria-label={`${text.users}${text.tableRegion}`}><table style={tableStyle}><thead><tr><th style={cellStyle}>\u7528\u6237\u540d</th><th style={cellStyle}>\u90ae\u7bb1</th><th style={cellStyle}>{text.role}</th><th style={cellStyle}>{text.plan}</th><th style={cellStyle}>\u4f5c\u54c1\u6570</th><th style={cellStyle}>{text.createdAt}</th><th style={cellStyle}>{text.actions}</th></tr></thead><tbody>{users.map((user) => (<tr key={user.id}><td style={cellStyle}>{user.username || "\u672a\u8bbe\u7f6e"}</td><td style={cellStyle}>{user.email}</td><td style={cellStyle}><label className="field" style={{ gap: 4 }}><span className="sr-only">{user.email} \u7684\u89d2\u8272</span><select value={editingUsers[user.id]?.role ?? user.role} onChange={(event) => setEditingUsers((current) => ({ ...current, [user.id]: { role: event.target.value, planCode: current[user.id]?.planCode ?? user.planCode } }))}><option value="user">\u666e\u901a\u7528\u6237</option><option value="admin">\u7ba1\u7406\u5458</option><option value="super_admin">\u8d85\u7ea7\u7ba1\u7406\u5458</option></select></label><span style={{ color: "var(--muted)", fontSize: "0.9rem" }}>\u5f53\u524d\uff1a{roleLabel(user.role)}</span></td><td style={cellStyle}><label className="field" style={{ gap: 4 }}><span className="sr-only">{user.email} \u7684\u5957\u9910</span><select value={editingUsers[user.id]?.planCode ?? user.planCode} onChange={(event) => setEditingUsers((current) => ({ ...current, [user.id]: { role: current[user.id]?.role ?? user.role, planCode: event.target.value } }))}><option value="free">\u514d\u8d39\u7248</option><option value="light">\u8f7b\u91cf\u7248</option><option value="support">\u652f\u6301\u8005</option><option value="admin">\u7ba1\u7406\u5458\u5957\u9910</option></select></label><span style={{ color: "var(--muted)", fontSize: "0.9rem" }}>\u5f53\u524d\uff1a{planLabel(user.planCode)}</span></td><td style={cellStyle}>{user.projectCount}</td><td style={cellStyle}>{formatDate(user.createdAt)}</td><td style={cellStyle}><button className="button-secondary" type="button" onClick={() => saveUserSettings(user.id)}>{text.saveUser}</button></td></tr>))}</tbody></table></div></section>) : null}

      {activeSection === "projects" ? (<section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-labelledby="admin-projects-title"><h2 id="admin-projects-title" style={{ margin: 0 }}>{text.projects}</h2>{projects.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.noProjects}</p> : null}<div style={tableWrapStyle} role="region" aria-label={`${text.projects}${text.tableRegion}`}><table style={tableStyle}><thead><tr><th style={cellStyle}>\u4f5c\u54c1</th><th style={cellStyle}>{text.owner}</th><th style={cellStyle}>\u5730\u5740</th><th style={cellStyle}>{text.visibility}</th><th style={cellStyle}>{text.interactive}</th><th style={cellStyle}>\u53d1\u5e03\u72b6\u6001</th></tr></thead><tbody>{projects.map((project) => (<tr key={project.id}><td style={cellStyle}>{project.name}</td><td style={cellStyle}>{project.ownerEmail}</td><td style={cellStyle}><a href={project.publicUrl} target="_blank" rel="noreferrer">{project.publicUrl}</a></td><td style={cellStyle}>{project.visibility === "public" ? "\u516c\u5f00" : "\u4e0d\u516c\u5f00"}</td><td style={cellStyle}>{project.interactive ? text.yes : text.no}</td><td style={cellStyle}>{project.currentReleaseId ? text.published : text.noRelease}</td></tr>))}</tbody></table></div></section>) : null}
    </section>
  );
}
