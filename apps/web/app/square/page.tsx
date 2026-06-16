import { getJSON } from "@/lib/api";

export const dynamic = "force-dynamic";

type Project = {
  id: string;
  username: string;
  slug: string;
  name: string;
  visibility: string;
  publicUrl: string;
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
                  {project.username}
                </p>
                <p style={{ margin: 0, color: "var(--muted)" }}>{project.publicUrl}</p>
              </div>
              <div>
                <a className="button-secondary" href={project.publicUrl} target="_blank" rel="noreferrer">
                  {"打开作品"}
                </a>
              </div>
            </article>
          ))}
        </section>
      )}
    </main>
  );
}
