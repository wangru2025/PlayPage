import { getJSON } from "@/lib/api";
import { FavoriteProjectButton } from "./FavoriteProjectButton";

export const dynamic = "force-dynamic";

type Project = {
  id: string;
  username: string;
  slug: string;
  name: string;
  visibility: string;
  publicUrl: string;
  allowForks: boolean;
  favoritesCount: number;
  forksCount: number;
  forkedFromProjectId?: string;
  forkedFromUsername?: string;
  forkedFromProjectName?: string;
  forkedFromProjectUrl?: string;
  favoritedByMe?: boolean;
};

type SquareResponse = {
  items: Project[];
};

const text = {
  eyebrow: "作品广场",
  title: "看看别人做了什么",
  intro:
    "这里只展示作者主动公开的作品。你可以先去逛逛，再回到自己的作品列表继续制作。",
  start: "我也要做一个",
  empty: "还没有公开作品。等第一批内测用户公开之后，这里就会热闹起来。",
  author: "作者：@"
};

export default async function SquarePage() {
  let projects: Project[] = [];

  try {
    const data = await getJSON<SquareResponse>("/api/v1/square");
    projects = data.items;
  } catch {
    projects = [];
  }

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px", display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>{text.eyebrow}</p>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.title}</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 760 }}>{text.intro}</p>
        <div>
          <a className="button-primary" href="/projects">
            {text.start}
          </a>
        </div>
      </header>

      {projects.length === 0 ? (
        <section className="panel" style={{ padding: 24 }}>
          <p style={{ margin: 0, color: "var(--muted)" }}>{text.empty}</p>
        </section>
      ) : (
        <section className="card-grid" aria-label={text.eyebrow}>
          {projects.map((project) => (
            <article key={project.id} className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
              <div style={{ display: "grid", gap: 8 }}>
                <h2 style={{ margin: 0 }}>{project.name}</h2>
                <p style={{ margin: 0, color: "var(--muted)" }}>
                  {text.author}
                  <a href={`/@${encodeURIComponent(project.username)}`}>{project.username}</a>
                </p>
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
                <a className="button-primary" href={project.publicUrl} target="_blank" rel="noreferrer">
                  打开作品
                </a>
                <FavoriteProjectButton projectId={project.id} initialFavorited={project.favoritedByMe} initialCount={project.favoritesCount} />
                {project.allowForks ? (
                  <a className="button-secondary" href={`/projects/fork?projectId=${project.id}`}>
                    改编这个作品
                  </a>
                ) : null}
              </div>
            </article>
          ))}
        </section>
      )}
    </main>
  );
}
