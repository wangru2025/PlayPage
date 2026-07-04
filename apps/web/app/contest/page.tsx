const tracks = [
  ["最有创意奖", "脑洞、整活、奇怪但有趣的网页。"],
  ["最好玩奖", "小游戏、点击器、经营、答题、解谜等。"],
  ["最实用奖", "工具、学习辅助、音频/文本处理等。"],
  ["最佳互动作品奖", "使用留言、论坛、排行榜、云存档等互动功能。"],
  ["新人潜力奖", "不要求成熟，重点看想法和继续创作的潜力。"]
];

const rewards = [
  ["最有创意奖", "1～5 人", "获奖徽章、广场精选推荐 7 天、支持版 1 个月。第一名额外获得首页展示位 7 天。"],
  ["最好玩奖", "1～5 人", "获奖徽章、广场精选推荐 7 天、支持版 1 个月。适合游戏类作品。"],
  ["最实用奖", "1～5 人", "获奖徽章、广场精选推荐 7 天、支持版 1 个月。适合工具、学习、效率类作品。"],
  ["最佳互动作品奖", "1～5 人", "获奖徽章、广场精选推荐 7 天、支持版 1 个月，并额外赠送互动用量加油包一次。"],
  ["新人潜力奖", "1～5 人", "获奖徽章、广场精选推荐 3 天、轻享版 1 个月。重点鼓励第一次认真创作的用户。"]
];

export default function ContestPage() {
  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 32, display: "grid", gap: 14 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>PlayPage 作品创作比赛</p>
        <h1 style={{ margin: 0, fontSize: "2.6rem" }}>把你的脑洞变成一个能打开的网址</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 860 }}>
          不要求你写出世界级网站，只要求你把想法做成能打开、能体验、能分享的 PlayPage 作品。AI 辅助、自写代码、从模板创建都可以。
        </p>
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
          <a className="button-primary" href="/projects">去我的作品提交参赛</a>
          <a className="button-secondary" href="/templates">先逛模板市场</a>
        </div>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
        <h2 style={{ margin: 0 }}>活动时间</h2>
        <ul style={{ margin: 0, paddingLeft: 22, lineHeight: 1.8 }}>
          <li>投稿时间：2026 年 7 月 5 日至 2026 年 7 月 26 日。</li>
          <li>评选时间：2026 年 7 月 27 日至 2026 年 7 月 31 日。</li>
          <li>结果公布：预计 2026 年 8 月 1 日公布获奖名单。</li>
          <li>奖励发放：结果公布后 7 天内陆续发放。</li>
        </ul>
      </section>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
        <h2 style={{ margin: 0 }}>参赛规则</h2>
        <ul style={{ margin: 0, paddingLeft: 22, lineHeight: 1.8 }}>
          <li>作品必须托管在 PlayPage，并且能正常打开。</li>
          <li>可以使用 AI 生成，也可以自己写代码或从模板创建。</li>
          <li>每个账号可以提交多个作品，但同一个作品只能提交一次。</li>
          <li>禁止违法违规、恶意代码、诈骗、色情、严重侵权内容。</li>
          <li>访问量只作为参考，不直接决定排名。</li>
        </ul>
      </section>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        <h2 style={{ margin: 0 }}>奖项赛道</h2>
        <div className="card-grid">
          {tracks.map(([name, desc]) => (
            <article key={name} style={{ border: "1px solid var(--line)", borderRadius: 18, padding: 18 }}>
              <h3 style={{ margin: 0 }}>{name}</h3>
              <p style={{ margin: "8px 0 0", color: "var(--muted)" }}>{desc}</p>
            </article>
          ))}
        </div>
      </section>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
        <h2 style={{ margin: 0 }}>奖励设计</h2>
        <p style={{ margin: 0, color: "var(--muted)" }}>
          每个赛道根据参赛作品数量和质量评选 1～5 人。人少时可以少评，人多且质量好时可以多评。最终名单由管理员统一评定。
        </p>
        <div className="card-grid">
          {rewards.map(([name, count, reward]) => (
            <article key={name} style={{ border: "1px solid var(--line)", borderRadius: 18, padding: 18 }}>
              <h3 style={{ margin: 0 }}>{name}</h3>
              <p style={{ margin: "8px 0 0", color: "var(--muted)" }}>名额：{count}</p>
              <p style={{ margin: "8px 0 0" }}>{reward}</p>
            </article>
          ))}
        </div>
      </section>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
        <h2 style={{ margin: 0 }}>评审维度</h2>
        <p style={{ margin: 0, color: "var(--muted)" }}>
          创意、完成度、体验、PlayPage 互动能力使用、分享传播都会参考。我们会优先鼓励真正有想法、愿意继续完善的作品。
        </p>
      </section>
    </section>
  );
}
