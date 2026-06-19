"use client";

export default function ErrorPage({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <main id="main-content" className="shell" data-route-back-source="error" style={{ padding: "48px 0 72px" }}>
      <section className="panel" style={{ padding: 28, display: "grid", gap: 16 }}>
        <p className="soft-badge" style={{ margin: 0 }}>
          页面出错
        </p>
        <h1 style={{ margin: 0, fontSize: "2.4rem" }}>这个页面暂时打不开</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>
          可能是网络波动或页面代码临时出错。你可以重试，或者先回到我的作品。
        </p>
        {error.digest ? (
          <p className="field-note" style={{ margin: 0 }}>
            错误编号：{error.digest}
          </p>
        ) : null}
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <button className="button-primary" type="button" onClick={reset}>
            重新加载
          </button>
          <a className="button-secondary" href="/projects" data-route-back-ignore="true">
            回到我的作品
          </a>
        </div>
      </section>
    </main>
  );
}
