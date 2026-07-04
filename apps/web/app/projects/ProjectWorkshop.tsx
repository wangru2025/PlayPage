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
  initial: "正在读取你的作品。",
  ready: "可以继续管理你的作品了。",
  invite: "请先登录后再查看你的作品。",
  login: "去登录或注册",
  needProfile: "请先设置公开名字，再开始创建和发布作品。",
  openProfileSetup: "去设置公开名字",
  logoutOk: "你已经退出登录。",
  logoutFail: "退出登录失败。",
  visibilityPublic: "作品已经公开到广场。",
  visibilityUnlisted: "作品已经从广场隐藏。",
  visibilityFail: "切换公开状态失败。",
  visibilityWorking: "正在保存公开状态。",
  deleteAsk: "确认要删除这个作品吗？删除之后就不能恢复了。",
  deleteOk: "作品已经删除。",
  deleteFail: "删除作品失败。",
  deleteWorking: "正在删除作品。",
  house: "作品小屋",
  heading: "我的作品",
  summary:
    "这里专门放你已经创建好的作品。创建、上传、互动功能和公开设置都拆开了，页面会更清楚。",
  create: "创建作品",
  current: "个人中心",
  logout: "退出登录",
  openSquare: "去作品广场",
  checking: "正在检查登录状态。",
  emptyTitle: "还没有作品",
  emptyHint: "先去创建一个作品，然后再回来管理它。",
  url: "作品地址：",
  published: "已经上传网页",
  noUpload: "还没有上传内容",
  publicInSquare: "正在广场展示",
  onlyLink: "只有知道链接的人能看",
  interactiveOn: "已启用互动功能",
  interactiveOff: "未启用互动功能",
  analyticsOn: "已启用访问量统计",
  openStats: "统计数据",
  projectSettings: "作品设置",
  releaseHistory: "作品历史版本",
  uploadNew: "上传新版本",
  downloadSource: "下载作品源文件",
  deleteProject: "删除作品",
  open: "打开作品",
  makePublic: "公开到广场",
  makeUnlisted: "从广场隐藏",
  openInteractive: "管理互动功能",
  listLabel: "你的作品列表",
  visibilitySaving: "正在保存",
  openDisabled: "请先上传作品内容",
  sourceDisabled: "请先上传作品内容",
  deleteDisabled: "请先从广场隐藏后再删除",
  domainTitle: "独立作品网址",
  domainHint: "你可以申请一个类似 https://my-game.wangru.net 的短网址。申请后需要管理员人工审核。",
  domainManage: "申请独立网址",
  domainPending: "审核中",
  domainActive: "已通过",
  domainRejected: "已驳回",
  domainLoadingFail: "读取独立网址申请失败。",
  contestBannerTitle: "PlayPage 作品创作比赛",
  contestBannerText: "把你的脑洞变成一个能打开的网址。小游戏、工具、论坛、整活网页都可以参加。",
  contestBannerLink: "查看比赛说明",
  moreActions: "更多操作",
  collapseActions: "收起操作",
  actionsFor: "的更多操作",
  requestRepair: "申请修复",
  viewRepairRequests: "查看修复申请",
  joinContest: "参加创作比赛"
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
          <nav aria-label="作品页面快捷入口" style={{ display: "grid", gap: 10, justifyItems: "end" }}>
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

      <section className="panel" style={{ padding: 22, display: "flex", gap: 16, flexWrap: "wrap", justifyContent: "space-between", alignItems: "center", background: "linear-gradient(135deg, rgba(124,58,237,0.13), rgba(255,255,255,0.9))" }}>
        <div style={{ display: "grid", gap: 6 }}>
          <h2 style={{ margin: 0, fontSize: "1.35rem" }}>{text.contestBannerTitle}</h2>
          <p style={{ margin: 0, color: "var(--muted)", maxWidth: 760 }}>{text.contestBannerText}</p>
        </div>
        <a className="button-primary" href="/contest">{text.contestBannerLink}</a>
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
                  <a className="button-secondary" href={`/projects/${project.id}/settings`}>
                    {text.projectSettings}
                  </a>
                  <a className="button-secondary" href={`/projects/${project.id}/releases`}>
                    {text.releaseHistory}
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
                  <a className="button-secondary" href={`/contest/submit?projectId=${project.id}`}>
                    {text.joinContest}
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
