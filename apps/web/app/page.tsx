const copy = {
  audience: "\u7ed9\u521a\u5f00\u59cb\u505a\u7f51\u9875\u7684\u4eba",
  brand: "PlayPage",
  titleA: "\u628a AI \u5199\u51fa\u6765\u7684\u5c0f\u7f51\u9875",
  titleB: "\u53d8\u6210\u771f\u7684\u5728\u7ebf\u4f5c\u54c1",
  intro:
    "\u4f60\u4e0d\u7528\u5148\u5b66\u670d\u52a1\u5668\uff0c\u4e5f\u4e0d\u7528\u81ea\u5df1\u6298\u817e\u90e8\u7f72\u3002\u6ce8\u518c\u4e4b\u540e\uff0c\u5148\u521b\u5efa\u4f5c\u54c1\uff0c\u518d\u4e0a\u4f20\u7f51\u9875\u538b\u7f29\u5305\uff0c\u5e73\u53f0\u5c31\u4f1a\u7ed9\u4f60\u4e00\u4e2a\u80fd\u5206\u4eab\u51fa\u53bb\u7684\u7f51\u5740\u3002\u7559\u8a00\u677f\u3001\u8bba\u575b\u548c\u8bc4\u8bba\u533a\uff0c\u4e5f\u7ec8\u4e8e\u80fd\u771f\u7684\u4fdd\u5b58\u5185\u5bb9\u3002",
  start: "\u5f00\u59cb\u521b\u5efa\u4f5c\u54c1",
  signin: "\u767b\u5f55\u6216\u6ce8\u518c",
  square: "\u5148\u770b\u770b\u522b\u4eba\u7684\u4f5c\u54c1",
  abilityTitle: "\u5e73\u53f0\u80fd\u505a\u4ec0\u4e48",
  threeSteps: "\u4e09\u6b65\u5c31\u80fd\u53d1\u51fa\u53bb",
  step1: "\u8ba9 AI \u5e2e\u4f60\u5199\u7f51\u9875",
  step2: "\u4e0a\u4f20 HTML \u6216 ZIP",
  step3: "\u628a\u7f51\u5740\u53d1\u7ed9\u522b\u4eba\u6253\u5f00",
  suitable: "\u9002\u5408\u505a\u8fd9\u4e9b",
  item1: "\u4e2a\u4eba\u4e3b\u9875\u548c\u5c0f\u6e38\u620f\u9875\u9762",
  item2: "\u5c0f\u8bba\u575b\u3001\u7559\u8a00\u677f\u3001\u6811\u6d1e",
  item3: "\u4f5c\u54c1\u5c55\u793a\u9875\u548c\u73ed\u7ea7\u6d3b\u52a8\u9875",
  highlights: "\u5e73\u53f0\u7279\u70b9",
  h1: "\u4e0d\u7528\u61c2\u670d\u52a1\u5668",
  h1body: "\u5148\u628a\u4f5c\u54c1\u505a\u51fa\u6765\u3002\u4e0a\u4f20\u4e4b\u540e\uff0c\u5e73\u53f0\u8d1f\u8d23\u53d1\u5e03\u3001\u4fdd\u5b58\u548c\u8bbf\u95ee\u5730\u5740\u3002",
  h2: "\u652f\u6301\u771f\u5b9e\u4e92\u52a8",
  h2body: "\u4e0d\u518d\u662f\u201c\u672c\u5730\u53d8\u91cf\u81ea\u5df1\u73a9\u201d\u3002\u522b\u4eba\u53d1\u7684\u5185\u5bb9\u80fd\u771f\u6b63\u7559\u5728\u7f51\u4e0a\u3002",
  h3: "\u9002\u5408\u5185\u6d4b\u5206\u4eab",
  h3body: "\u5148\u7528\u4e0d\u516c\u5f00\u94fe\u63a5\u5206\u4eab\u7ed9\u540c\u5b66\u548c\u670b\u53cb\uff0c\u4f5c\u54c1\u6210\u719f\u540e\u518d\u7ee7\u7eed\u6269\u5c55\u3002"
};

export default function HomePage() {
  return (
    <main id="main-content" className="shell" style={{ padding: "36px 0 72px", display: "grid", gap: 20 }}>
      <section className="panel" style={{ padding: 32, display: "grid", gap: 24 }}>
        <div className="hero-grid">
          <div style={{ display: "grid", gap: 18 }}>
            <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
              <span className="soft-badge">{copy.brand}</span>
              <span className="soft-badge">{copy.audience}</span>
            </div>

            <div style={{ display: "grid", gap: 12 }}>
              <h1 style={{ margin: 0, fontSize: "clamp(2.5rem, 7vw, 5rem)", lineHeight: 0.96 }}>
                {copy.titleA}
                <br />
                {copy.titleB}
              </h1>
              <p style={{ margin: 0, maxWidth: 720, fontSize: "1.08rem", color: "var(--muted)" }}>{copy.intro}</p>
            </div>

            <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
              <a className="button-primary" href="/projects">
                {copy.start}
              </a>
              <a className="button-secondary" href="/auth">
                {copy.signin}
              </a>
              <a className="button-secondary" href="/square">
                {copy.square}
              </a>
            </div>
          </div>

          <aside style={{ display: "grid", gap: 14, alignContent: "start" }} aria-label={copy.abilityTitle}>
            <div className="panel" style={{ padding: 20, background: "rgba(255,255,255,0.72)" }}>
              <strong>{copy.threeSteps}</strong>
              <ol style={{ margin: "12px 0 0", paddingLeft: 18, color: "var(--muted)" }}>
                <li>{copy.step1}</li>
                <li>{copy.step2}</li>
                <li>{copy.step3}</li>
              </ol>
            </div>
            <div className="panel" style={{ padding: 20, background: "rgba(255,255,255,0.72)" }}>
              <strong>{copy.suitable}</strong>
              <ul style={{ margin: "12px 0 0", paddingLeft: 18, color: "var(--muted)" }}>
                <li>{copy.item1}</li>
                <li>{copy.item2}</li>
                <li>{copy.item3}</li>
              </ul>
            </div>
          </aside>
        </div>
      </section>

      <section className="card-grid" aria-label={copy.highlights}>
        <article className="panel" style={{ padding: 22 }}>
          <h2 style={{ marginTop: 0 }}>{copy.h1}</h2>
          <p style={{ marginBottom: 0, color: "var(--muted)" }}>{copy.h1body}</p>
        </article>
        <article className="panel" style={{ padding: 22 }}>
          <h2 style={{ marginTop: 0 }}>{copy.h2}</h2>
          <p style={{ marginBottom: 0, color: "var(--muted)" }}>{copy.h2body}</p>
        </article>
        <article className="panel" style={{ padding: 22 }}>
          <h2 style={{ marginTop: 0 }}>{copy.h3}</h2>
          <p style={{ marginBottom: 0, color: "var(--muted)" }}>{copy.h3body}</p>
        </article>
      </section>
    </main>
  );
}
