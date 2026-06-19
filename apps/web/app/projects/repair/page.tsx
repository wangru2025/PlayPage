const text = {
  title: "AI 网页急救站",
  subtitle: "AI 写坏了？页面白屏、按钮失灵、互动功能报错，都可以提交给 PlayPage 帮你看看。",
  cta: "去选择要修复的作品",
  scopeTitle: "可以申请修复的问题",
  ruleTitle: "活动规则",
  processTitle: "处理流程",
  canFix: [
    "页面打不开、白屏或控制台报错。",
    "按钮点了没反应，表单无法提交。",
    "互动功能连接失败、数据表不存在、字段写错。",
    "中文乱码、路径 404、AI 写错 API 地址。",
    "简单样式错乱或移动端明显显示异常。"
  ],
  rules: [
    "这是修 bug 活动，不是免费代写大型项目。",
    "每次申请请只写一个主要问题，描述越清楚越容易处理。",
    "提交时需要确认允许管理员查看并修改该作品代码。",
    "同一个作品已有待处理申请时，不能重复提交。",
    "涉及诈骗、恶意软件、盗版、违法违规内容的作品不会处理。"
  ],
  steps: [
    "在我的作品里点击申请修复。",
    "填写问题、期望效果，并确认授权。",
    "管理员在后台查看作品和问题描述。",
    "修好或需要补充信息时，会在申请记录里回复你。"
  ]
};

export default function RepairActivityPage() {
  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <section style={{ display: "grid", gap: 18 }}>
        <header className="panel" style={{ padding: 28, display: "grid", gap: 14 }}>
          <p className="soft-badge" style={{ justifySelf: "start", margin: 0 }}>PlayPage 活动</p>
          <h1 style={{ margin: 0, fontSize: "2.6rem" }}>{text.title}</h1>
          <p style={{ margin: 0, color: "var(--muted)", maxWidth: 780 }}>{text.subtitle}</p>
          <div>
            <a className="button-primary" href="/projects">{text.cta}</a>
          </div>
        </header>

        <section className="card-grid" aria-label="活动说明">
          <article className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
            <h2 style={{ margin: 0 }}>{text.scopeTitle}</h2>
            <ul style={{ margin: 0, paddingLeft: 22 }}>
              {text.canFix.map((item) => <li key={item}>{item}</li>)}
            </ul>
          </article>
          <article className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
            <h2 style={{ margin: 0 }}>{text.ruleTitle}</h2>
            <ul style={{ margin: 0, paddingLeft: 22 }}>
              {text.rules.map((item) => <li key={item}>{item}</li>)}
            </ul>
          </article>
          <article className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
            <h2 style={{ margin: 0 }}>{text.processTitle}</h2>
            <ol style={{ margin: 0, paddingLeft: 22 }}>
              {text.steps.map((item) => <li key={item}>{item}</li>)}
            </ol>
          </article>
        </section>
      </section>
    </main>
  );
}
