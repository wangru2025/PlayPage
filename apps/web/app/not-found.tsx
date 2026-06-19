export default function NotFound() {
  return (
    <main id="main-content" className="shell" style={{ padding: "48px 0 72px" }}>
      <section className="panel" style={{ padding: 28, display: "grid", gap: 16 }}>
        <p className="soft-badge" style={{ margin: 0 }}>
          404
        </p>
        <h1 style={{ margin: 0, fontSize: "2.4rem" }}>没有找到这个页面</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>
          这个地址可能写错了，也可能页面已经被移动或删除。
        </p>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <a className="button-primary" href="/projects">
            回到我的作品
          </a>
          <a className="button-secondary" href="/square">
            去作品广场
          </a>
        </div>
      </section>
    </main>
  );
}
