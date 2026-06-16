const copy = {
  audience: "给刚开始做网页的人",
  brand: "PlayPage",
  titleA: "把 AI 写出来的小网页",
  titleB: "变成真的在线作品",
  intro:
    "你不用先学服务器，也不用自己折腾部署。注册之后，先创建作品，再上传网页压缩包，平台就会给你一个能分享出去的网址。留言板、论坛和评论区，也终于能真的保存内容。",
  start: "开始创建作品",
  signin: "登录或注册",
  square: "先看看别人的作品",
  abilityTitle: "平台能做什么",
  threeSteps: "三步就能发出去",
  step1: "让 AI 帮你写网页",
  step2: "上传 HTML 或 ZIP",
  step3: "把网址发给别人打开",
  suitable: "适合做这些",
  item1: "个人主页和小游戏页面",
  item2: "小论坛、留言板、树洞",
  item3: "作品展示页和班级活动页",
  highlights: "平台特点",
  h1: "不用懂服务器",
  h1body: "先把作品做出来。上传之后，平台负责发布、保存和访问地址。",
  h2: "支持真实互动",
  h2body: "不再是“本地变量自己玩”。别人发的内容能真正留在网上。",
  h3: "适合内测分享",
  h3body: "先用不公开链接分享给同学和朋友，作品成熟后再继续扩展。"
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
