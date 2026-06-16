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
  eyebrow: "\u4f5c\u54c1\u5e7f\u573a",
  title: "\u770b\u770b\u522b\u4eba\u505a\u4e86\u4ec0\u4e48",
  intro:
    "\u8fd9\u91cc\u53ea\u5c55\u793a\u4f5c\u8005\u4e3b\u52a8\u516c\u5f00\u7684\u4f5c\u54c1\u3002\u4f60\u53ef\u4ee5\u5148\u53bb\u901b\u901b\uff0c\u518d\u56de\u5230\u81ea\u5df1\u7684\u4f5c\u54c1\u5217\u8868\u7ee7\u7eed\u5236\u4f5c\u3002",
  start: "\u6211\u4e5f\u8981\u505a\u4e00\u4e2a",
  empty: "\u8fd8\u6ca1\u6709\u516c\u5f00\u4f5c\u54c1\u3002\u7b49\u7b2c\u4e00\u6279\u5185\u6d4b\u7528\u6237\u516c\u5f00\u4e4b\u540e\uff0c\u8fd9\u91cc\u5c31\u4f1a\u70ed\u95f9\u8d77\u6765\u3002",
  author: "\u4f5c\u8005\uff1a@"
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
                  {"\u6253\u5f00\u4f5c\u54c1"}
                </a>
              </div>
            </article>
          ))}
        </section>
      )}
    </main>
  );
}
