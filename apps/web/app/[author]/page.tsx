import { notFound } from "next/navigation";
import { getJSON } from "@/lib/api";
import { FavoriteProjectButton } from "../square/FavoriteProjectButton";
import { FollowAuthorButton } from "./FollowAuthorButton";

export const dynamic = "force-dynamic";

type Project = {
  id: string;
  username: string;
  slug: string;
  name: string;
  publicUrl: string;
  allowForks: boolean;
  favoritesCount: number;
  forksCount: number;
  forkedFromProjectId?: string;
  forkedFromUsername?: string;
  forkedFromProjectName?: string;
  forkedFromProjectUrl?: string;
};

type AuthorProfile = {
  username: string;
  displayName: string;
  joinedAt: string;
  projects: Project[];
  projectCount: number;
  favoritesCount: number;
  forksCount: number;
  followersCount: number;
  followingCount: number;
};

function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "未知时间";
  return date.toLocaleDateString("zh-CN");
}

export default async function AuthorPage({ params }: { params: Promise<{ author: string }> }) {
  const { author } = await params;
  const decoded = decodeURIComponent(author);
  if (!decoded.startsWith("@") || decoded.length <= 1) {
    notFound();
  }
  const username = decoded.slice(1);
  let profile: AuthorProfile;
  try {
    profile = await getJSON<AuthorProfile>(`/api/v1/authors/${encodeURIComponent(username)}`);
  } catch {
    notFound();
  }

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px", display: "grid", gap: 18 }}>
      <section className="panel" style={{ padding: 28, display: "grid", gap: 14 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>作者主页</p>
        <h1 style={{ margin: 0, fontSize: "2.6rem" }}>@{profile.displayName || profile.username}</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>加入时间：{formatDate(profile.joinedAt)}</p>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <span className="soft-badge">主页作品 {profile.projectCount}</span>
          <span className="soft-badge">收到收藏 {profile.favoritesCount}</span>
          <span className="soft-badge">被改编 {profile.forksCount}</span>
          <span className="soft-badge">关注了 {profile.followingCount}</span>
        </div>
        <div>
          <FollowAuthorButton username={profile.username} initialFollowersCount={profile.followersCount} />
        </div>
      </section>

      {profile.projects.length === 0 ? (
        <section className="panel" style={{ padding: 24 }}>
          <p style={{ margin: 0, color: "var(--muted)" }}>这个作者还没有在主页展示作品。</p>
        </section>
      ) : (
        <section className="card-grid" aria-label="作者主页作品">
          {profile.projects.map((project) => (
            <article key={project.id} className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
              <div style={{ display: "grid", gap: 8 }}>
                <h2 style={{ margin: 0 }}>{project.name}</h2>
                {project.forkedFromProjectId ? (
                  <p style={{ margin: 0, color: "var(--muted)" }}>
                    改编自 {project.forkedFromUsername || "原作者"} 的
                    {project.forkedFromProjectUrl ? (
                      <a href={project.forkedFromProjectUrl}>《{project.forkedFromProjectName || "原作品"}》</a>
                    ) : (
                      <>《{project.forkedFromProjectName || "原作品"}》</>
                    )}
                  </p>
                ) : null}
                <p style={{ margin: 0, color: "var(--muted)" }}>{project.publicUrl}</p>
                <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                  <span className="soft-badge">收藏 {project.favoritesCount}</span>
                  <span className="soft-badge">被改编 {project.forksCount}</span>
                </div>
              </div>
              <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
                <a className="button-primary" href={project.publicUrl} target="_blank" rel="noreferrer">打开作品</a>
                <FavoriteProjectButton projectId={project.id} initialFavorited={false} initialCount={project.favoritesCount} />
                <a className="button-secondary" href={`/projects/${project.id}/discussions`}>讨论区</a>
                <a className="button-secondary" href={`/projects/${project.id}/proposals`}>改进提案</a>
                {project.allowForks ? <a className="button-secondary" href={`/projects/fork?projectId=${project.id}`}>改编这个作品</a> : null}
              </div>
            </article>
          ))}
        </section>
      )}
    </main>
  );
}
