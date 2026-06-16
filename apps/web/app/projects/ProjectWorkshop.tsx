"use client";

import { useEffect, useState } from "react";
import { buildURL, deleteJSON, getJSON, postJSON } from "@/lib/api";

type User = {
  id: string;
  username: string;
  email: string;
};

type Project = {
  id: string;
  username: string;
  slug: string;
  name: string;
  interactive: boolean;
  analyticsEnabled: boolean;
  visibility: string;
  publicUrl: string;
  currentReleaseId?: string;
};

type ProjectListResponse = {
  items: Project[];
};

type ProjectDomain = {
  id: string;
  projectId: string;
  subdomain: string;
  domain: string;
  status: string;
  rejectReason: string;
  adminNote: string;
  createdAt: string;
};

type StatusTone = "info" | "success" | "error";

const text = {
  initial: "\u6b63\u5728\u8bfb\u53d6\u4f60\u7684\u4f5c\u54c1\u3002",
  ready: "\u53ef\u4ee5\u7ee7\u7eed\u7ba1\u7406\u4f60\u7684\u4f5c\u54c1\u4e86\u3002",
  invite: "\u8bf7\u5148\u767b\u5f55\u540e\u518d\u67e5\u770b\u4f60\u7684\u4f5c\u54c1\u3002",
  login: "\u53bb\u767b\u5f55\u6216\u6ce8\u518c",
  needProfile: "\u8bf7\u5148\u8bbe\u7f6e\u516c\u5f00\u540d\u5b57\uff0c\u518d\u5f00\u59cb\u521b\u5efa\u548c\u53d1\u5e03\u4f5c\u54c1\u3002",
  openProfileSetup: "\u53bb\u8bbe\u7f6e\u516c\u5f00\u540d\u5b57",
  logoutOk: "\u4f60\u5df2\u7ecf\u9000\u51fa\u767b\u5f55\u3002",
  logoutFail: "\u9000\u51fa\u767b\u5f55\u5931\u8d25\u3002",
  visibilityPublic: "\u4f5c\u54c1\u5df2\u7ecf\u516c\u5f00\u5230\u5e7f\u573a\u3002",
  visibilityUnlisted: "\u4f5c\u54c1\u5df2\u7ecf\u4ece\u5e7f\u573a\u9690\u85cf\u3002",
  visibilityFail: "\u5207\u6362\u516c\u5f00\u72b6\u6001\u5931\u8d25\u3002",
  visibilityWorking: "\u6b63\u5728\u4fdd\u5b58\u516c\u5f00\u72b6\u6001\u3002",
  deleteAsk: "\u786e\u8ba4\u8981\u5220\u9664\u8fd9\u4e2a\u4f5c\u54c1\u5417\uff1f\u5220\u9664\u4e4b\u540e\u5c31\u4e0d\u80fd\u6062\u590d\u4e86\u3002",
  deleteOk: "\u4f5c\u54c1\u5df2\u7ecf\u5220\u9664\u3002",
  deleteFail: "\u5220\u9664\u4f5c\u54c1\u5931\u8d25\u3002",
  deleteWorking: "\u6b63\u5728\u5220\u9664\u4f5c\u54c1\u3002",
  house: "\u4f5c\u54c1\u5c0f\u5c4b",
  heading: "\u6211\u7684\u4f5c\u54c1",
  summary:
    "\u8fd9\u91cc\u4e13\u95e8\u653e\u4f60\u5df2\u7ecf\u521b\u5efa\u597d\u7684\u4f5c\u54c1\u3002\u521b\u5efa\u3001\u4e0a\u4f20\u3001\u4e92\u52a8\u529f\u80fd\u548c\u516c\u5f00\u8bbe\u7f6e\u90fd\u62c6\u5f00\u4e86\uff0c\u9875\u9762\u4f1a\u66f4\u6e05\u695a\u3002",
  create: "\u521b\u5efa\u4f5c\u54c1",
  current: "\u4e2a\u4eba\u4e2d\u5fc3",
  logout: "\u9000\u51fa\u767b\u5f55",
  openSquare: "\u53bb\u4f5c\u54c1\u5e7f\u573a",
  checking: "\u6b63\u5728\u68c0\u67e5\u767b\u5f55\u72b6\u6001\u3002",
  emptyTitle: "\u8fd8\u6ca1\u6709\u4f5c\u54c1",
  emptyHint: "\u5148\u53bb\u521b\u5efa\u4e00\u4e2a\u4f5c\u54c1\uff0c\u7136\u540e\u518d\u56de\u6765\u7ba1\u7406\u5b83\u3002",
  url: "\u4f5c\u54c1\u5730\u5740\uff1a",
  published: "\u5df2\u7ecf\u4e0a\u4f20\u7f51\u9875",
  noUpload: "\u8fd8\u6ca1\u6709\u4e0a\u4f20\u5185\u5bb9",
  publicInSquare: "\u6b63\u5728\u5e7f\u573a\u5c55\u793a",
  onlyLink: "\u53ea\u6709\u77e5\u9053\u94fe\u63a5\u7684\u4eba\u80fd\u770b",
  interactiveOn: "\u5df2\u542f\u7528\u4e92\u52a8\u529f\u80fd",
  interactiveOff: "\u672a\u542f\u7528\u4e92\u52a8\u529f\u80fd",
  analyticsOn: "\u5df2\u542f\u7528\u8bbf\u95ee\u91cf\u7edf\u8ba1",
  openStats: "\u7edf\u8ba1\u6570\u636e",
  uploadNew: "\u4e0a\u4f20\u65b0\u7248\u672c",
  downloadSource: "\u4e0b\u8f7d\u4f5c\u54c1\u6e90\u6587\u4ef6",
  deleteProject: "\u5220\u9664\u4f5c\u54c1",
  open: "\u6253\u5f00\u4f5c\u54c1",
  makePublic: "\u516c\u5f00\u5230\u5e7f\u573a",
  makeUnlisted: "\u4ece\u5e7f\u573a\u9690\u85cf",
  openInteractive: "\u7ba1\u7406\u4e92\u52a8\u529f\u80fd",
  listLabel: "\u4f60\u7684\u4f5c\u54c1\u5217\u8868",
  visibilitySaving: "\u6b63\u5728\u4fdd\u5b58",
  openDisabled: "\u8bf7\u5148\u4e0a\u4f20\u4f5c\u54c1\u5185\u5bb9",
  sourceDisabled: "\u8bf7\u5148\u4e0a\u4f20\u4f5c\u54c1\u5185\u5bb9",
  deleteDisabled: "\u8bf7\u5148\u4ece\u5e7f\u573a\u9690\u85cf\u540e\u518d\u5220\u9664",
  domainTitle: "\u72ec\u7acb\u4f5c\u54c1\u7f51\u5740",
  domainHint: "\u4f60\u53ef\u4ee5\u7533\u8bf7\u4e00\u4e2a\u7c7b\u4f3c https://my-game.wangru.net \u7684\u77ed\u7f51\u5740\u3002\u7533\u8bf7\u540e\u9700\u8981\u7ba1\u7406\u5458\u4eba\u5de5\u5ba1\u6838\u3002",
  domainManage: "\u7533\u8bf7\u72ec\u7acb\u7f51\u5740",
  domainPending: "\u5ba1\u6838\u4e2d",
  domainActive: "\u5df2\u901a\u8fc7",
  domainRejected: "\u5df2\u9a73\u56de",
  domainLoadingFail: "\u8bfb\u53d6\u72ec\u7acb\u7f51\u5740\u7533\u8bf7\u5931\u8d25\u3002",
  repairBannerTitle: "AI \u7f51\u9875\u6025\u6551\u7ad9\u6d3b\u52a8",
  repairBannerText: "AI \u5199\u574f\u4e86\u3001\u9875\u9762\u767d\u5c4f\u3001\u6309\u94ae\u6ca1\u53cd\u5e94\u3001\u4e92\u52a8\u529f\u80fd\u62a5\u9519\uff1f\u53ef\u4ee5\u63d0\u4ea4\u4fee\u590d\u7533\u8bf7\uff0cPlayPage \u4f1a\u5c3d\u91cf\u5e2e\u4f60\u770b\u770b\u3002",
  repairBannerLink: "\u67e5\u770b\u6d3b\u52a8\u8bf4\u660e",
  moreActions: "\u66f4\u591a\u64cd\u4f5c",
  collapseActions: "\u6536\u8d77\u64cd\u4f5c",
  actionsFor: "\u7684\u66f4\u591a\u64cd\u4f5c",
  requestRepair: "\u7533\u8bf7\u4fee\u590d",
  viewRepairRequests: "\u67e5\u770b\u4fee\u590d\u7533\u8bf7"
};

export function ProjectWorkshop() {
  const [user, setUser] = useState<User | null>(null);
  const [projects, setProjects] = useState<Project[]>([]);
  const [statusText, setStatusText] = useState(text.initial);
  const [statusTone, setStatusTone] = useState<StatusTone>("info");
  const [loading, setLoading] = useState(true);
  const [loggingOut, setLoggingOut] = useState(false);
  const [visibilityProjectId, setVisibilityProjectId] = useState("");
  const [deletingProjectId, setDeletingProjectId] = useState("");
  const [domainsByProject, setDomainsByProject] = useState<Record<string, ProjectDomain[]>>({});
  const [expandedProjectId, setExpandedProjectId] = useState("");

  useEffect(() => {
    void loadDashboard();
  }, []);

  async function loadDashboard() {
    setLoading(true);
    try {
      const me = await getJSON<User>("/api/v1/me");
      setUser(me);
      if (me.username.trim() === "") {
        setProjects([]);
        setStatus(text.needProfile, "info");
        return;
      }
      await refreshProjects();
      setStatus(text.ready, "success");
    } catch {
      setUser(null);
      setProjects([]);
      setStatus(text.invite, "info");
    } finally {
      setLoading(false);
    }
  }

  async function refreshProjects() {
    const data = await getJSON<ProjectListResponse>("/api/v1/projects");
    setProjects(data.items);
    void refreshProjectDomains(data.items);
    return data.items;
  }

  async function refreshProjectDomains(items: Project[]) {
    try {
      const pairs = await Promise.all(
        items.map(async (project) => {
          const data = await getJSON<{ items: ProjectDomain[] }>(`/api/v1/projects/${project.id}/domains`);
          return [project.id, data.items] as const;
        })
      );
      setDomainsByProject(Object.fromEntries(pairs));
    } catch (error) {
      setStatus(error instanceof Error ? error.message : text.domainLoadingFail, "error");
    }
  }

  function setStatus(message: string, tone: StatusTone) {
    setStatusText(message);
    setStatusTone(tone);
  }

  async function logout() {
    try {
      setLoggingOut(true);
      await postJSON<{ status: string }>("/api/v1/auth/logout", {});
      setUser(null);
      setProjects([]);
      setStatus(text.logoutOk, "info");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : text.logoutFail, "error");
    } finally {
      setLoggingOut(false);
    }
  }

  async function updateVisibility(projectId: string, visibility: "public" | "unlisted") {
    if (visibilityProjectId !== "" || deletingProjectId !== "") {
      return;
    }
    try {
      setVisibilityProjectId(projectId);
      setStatus(text.visibilityWorking, "info");
      await postJSON<Project>(`/api/v1/projects/${projectId}/visibility`, {
        visibility
      });
      const items = await refreshProjects();
      const updated = items.find((project) => project.id === projectId);
      const finalVisibility = updated?.visibility ?? visibility;
      setStatus(finalVisibility === "public" ? text.visibilityPublic : text.visibilityUnlisted, "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : text.visibilityFail, "error");
    } finally {
      setVisibilityProjectId("");
    }
  }

  async function deleteProject(project: Project) {
    if (visibilityProjectId !== "" || deletingProjectId !== "") {
      return;
    }
    if (!window.confirm(text.deleteAsk)) {
      return;
    }

    try {
      setDeletingProjectId(project.id);
      setStatus(text.deleteWorking, "info");
      await deleteJSON<{ status: string }>(`/api/v1/projects/${project.id}`);
      await refreshProjects();
      setStatus(text.deleteOk, "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : text.deleteFail, "error");
    } finally {
      setDeletingProjectId("");
    }
  }

  function domainStatusLabel(status: string): string {
    if (status === "active") return text.domainActive;
    if (status === "rejected") return text.domainRejected;
    return text.domainPending;
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 14 }}>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            gap: 16,
            flexWrap: "wrap",
            alignItems: "start"
          }}
        >
          <div style={{ display: "grid", gap: 8 }}>
            <p style={{ margin: 0, color: "var(--muted)" }}>{text.house}</p>
            <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.heading}</h1>
            <p style={{ margin: 0, color: "var(--muted)", maxWidth: 760 }}>{text.summary}</p>
          </div>
          <nav aria-label="\u4f5c\u54c1\u9875\u9762\u5feb\u6377\u5165\u53e3" style={{ display: "grid", gap: 10, justifyItems: "end" }}>
            <a className="soft-badge" href="/me">
              {text.current}
            </a>
            <div style={{ display: "flex", gap: 10, flexWrap: "wrap", justifyContent: "end" }}>
              <a className="button-primary" href="/projects/new">
                {text.create}
              </a>
              <a className="button-secondary" href="/square">
                {text.openSquare}
              </a>
              {user ? (
                <button className="button-ghost" type="button" onClick={logout} disabled={loggingOut || visibilityProjectId !== "" || deletingProjectId !== ""}>
                  {text.logout}
                </button>
              ) : null}
            </div>
          </nav>
        </div>

        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite">
          {loading ? text.checking : statusText}
        </div>
      </header>

      <section className="panel" style={{ padding: 22, display: "flex", gap: 16, flexWrap: "wrap", justifyContent: "space-between", alignItems: "center", background: "linear-gradient(135deg, rgba(214,101,47,0.12), rgba(255,255,255,0.86))" }}>
        <div style={{ display: "grid", gap: 6 }}>
          <h2 style={{ margin: 0, fontSize: "1.35rem" }}>{text.repairBannerTitle}</h2>
          <p style={{ margin: 0, color: "var(--muted)", maxWidth: 760 }}>{text.repairBannerText}</p>
        </div>
        <a className="button-primary" href="/projects/repair">{text.repairBannerLink}</a>
      </section>

      {!user ? (
        <section className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
          <p style={{ margin: 0, color: "var(--muted)" }}>{text.invite}</p>
          <div>
            <a className="button-primary" href="/auth">
              {text.login}
            </a>
          </div>
        </section>
      ) : user.username.trim() === "" ? (
        <section className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
          <p style={{ margin: 0, color: "var(--muted)" }}>{text.needProfile}</p>
          <div>
            <a className="button-primary" href="/auth/profile?next=/projects">
              {text.openProfileSetup}
            </a>
          </div>
        </section>
      ) : projects.length === 0 ? (
        <section className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
          <h2 style={{ margin: 0 }}>{text.emptyTitle}</h2>
          <p style={{ margin: 0, color: "var(--muted)" }}>{text.emptyHint}</p>
          <div>
            <a className="button-primary" href="/projects/new">
              {text.create}
            </a>
          </div>
        </section>
      ) : (
        <section className="project-grid" aria-label={text.listLabel}>
          {projects.map((project) => (
            <article key={project.id} className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
              <header style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap", alignItems: "start" }}>
                <div style={{ display: "grid", gap: 8 }}>
                  <h2 style={{ margin: 0, fontSize: "1.35rem" }}>{project.name}</h2>
                  <p style={{ margin: 0, color: "var(--muted)" }}>
                    {text.url}
                    {project.publicUrl}
                  </p>
                  <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                    <span className="soft-badge">{project.currentReleaseId ? text.published : text.noUpload}</span>
                    <span className="soft-badge">{project.visibility === "public" ? text.publicInSquare : text.onlyLink}</span>
                    {project.interactive ? <span className="soft-badge">{text.interactiveOn}</span> : <span className="soft-badge">{text.interactiveOff}</span>}
                    {project.analyticsEnabled ? <span className="soft-badge">{text.analyticsOn}</span> : null}
                  </div>
                </div>
                <div style={{ display: "flex", gap: 10, flexWrap: "wrap", justifyContent: "end" }}>
                  <a className="button-primary" href={project.publicUrl} target="_blank" rel="noreferrer">
                    {text.open}
                  </a>
                  <button
                    className="button-secondary"
                    type="button"
                    aria-expanded={expandedProjectId === project.id}
                    aria-controls={`project-actions-${project.id}`}
                    onClick={() => setExpandedProjectId((current) => current === project.id ? "" : project.id)}
                  >
                    {expandedProjectId === project.id ? text.collapseActions : text.moreActions}
                  </button>
                </div>
              </header>

              {(domainsByProject[project.id] ?? []).length > 0 ? (
                <section style={{ border: "1px solid var(--line)", borderRadius: 16, padding: 14, display: "grid", gap: 10, background: "rgba(255,255,255,0.55)" }}>
                  <h3 style={{ margin: 0, fontSize: "1.05rem" }}>{text.domainTitle}</h3>
                  {(domainsByProject[project.id] ?? []).map((item) => (
                    <div key={item.id} className="soft-badge" style={{ justifySelf: "start" }}>
                      {domainStatusLabel(item.status)}：{item.domain}
                      {item.rejectReason ? `，原因：${item.rejectReason}` : ""}
                    </div>
                  ))}
                </section>
              ) : null}

              {expandedProjectId === project.id ? (
                <div
                  id={`project-actions-${project.id}`}
                  style={{ display: "flex", gap: 12, flexWrap: "wrap", borderTop: "1px solid var(--line)", paddingTop: 14 }}
                  aria-label={`${project.name}${text.actionsFor}`}
                >
                  <a className="button-primary" href={`/projects/new?projectId=${project.id}`}>
                    {text.uploadNew}
                  </a>
                  <button
                    className="button-secondary"
                    type="button"
                    disabled={!project.currentReleaseId || visibilityProjectId !== "" || deletingProjectId !== ""}
                    title={!project.currentReleaseId ? text.sourceDisabled : undefined}
                    onClick={() => {
                      window.location.href = buildURL(`/api/v1/projects/${project.id}/source`);
                    }}
                  >
                    {text.downloadSource}
                  </button>
                  {project.interactive ? (
                    <a className="button-secondary" href={`/projects/${project.id}/interactive`}>
                      {text.openInteractive}
                    </a>
                  ) : null}
                  {project.analyticsEnabled ? (
                    <a className="button-secondary" href={`/projects/${project.id}/stats`}>
                      {text.openStats}
                    </a>
                  ) : null}
                  <button
                    className="button-secondary"
                    type="button"
                    disabled={!project.currentReleaseId || visibilityProjectId !== "" || deletingProjectId !== ""}
                    aria-busy={visibilityProjectId === project.id}
                    title={!project.currentReleaseId ? text.openDisabled : undefined}
                    onClick={() => updateVisibility(project.id, project.visibility === "public" ? "unlisted" : "public")}
                  >
                    {visibilityProjectId === project.id
                      ? text.visibilitySaving
                      : project.visibility === "public"
                        ? text.makeUnlisted
                        : text.makePublic}
                  </button>
                  <a className="button-secondary" href={`/projects/${project.id}/repair`}>
                    {text.requestRepair}
                  </a>
                  <a className="button-secondary" href={`/projects/${project.id}/repair/requests`}>
                    {text.viewRepairRequests}
                  </a>
                  <a className="button-secondary" href={`/projects/${project.id}/domains`}>
                    {text.domainManage}
                  </a>
                  <button
                    className="button-ghost"
                    type="button"
                    disabled={project.visibility === "public" || visibilityProjectId !== "" || deletingProjectId !== ""}
                    aria-busy={deletingProjectId === project.id}
                    title={project.visibility === "public" ? text.deleteDisabled : undefined}
                    onClick={() => deleteProject(project)}
                  >
                    {deletingProjectId === project.id ? text.deleteWorking : text.deleteProject}
                  </button>
                </div>
              ) : null}
            </article>
          ))}
        </section>
      )}
    </section>
  );
}
