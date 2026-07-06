"use client";

import { useEffect, useState } from "react";
import { getJSON } from "@/lib/api";

type User = { username: string; email: string; role: string; planCode: string };

function planLabel(planCode: string): string {
  switch (planCode) {
    case "light": return "轻享版";
    case "support": return "支持版";
    case "admin": return "管理员版";
    default: return "免费版";
  }
}

export function MeDashboard() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getJSON<User>("/api/v1/me").then(setUser).catch(() => setUser(null)).finally(() => setLoading(false));
  }, []);

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px", display: "grid", gap: 18 }}>
      <section className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>个人中心</p>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{loading ? "正在读取账号信息" : user ? `你好，${user.username || user.email}` : "请先登录"}</h1>
        {user ? <p style={{ margin: 0, color: "var(--muted)" }}>当前套餐：{planLabel(user.planCode)}</p> : null}
      </section>
      {user ? (
        <section className="card-grid" aria-label="个人中心功能">
          <a className="panel" href="/me/profile" style={{ padding: 24, display: "grid", gap: 8, textDecoration: "none", color: "inherit" }}>
            <h2 style={{ margin: 0 }}>个人资料</h2><p style={{ margin: 0, color: "var(--muted)" }}>修改公开名字、查看邮箱和账号身份。</p>
          </a>
          <a className="panel" href="/me/favorites" style={{ padding: 24, display: "grid", gap: 8, textDecoration: "none", color: "inherit" }}>
            <h2 style={{ margin: 0 }}>我的收藏</h2><p style={{ margin: 0, color: "var(--muted)" }}>查看你收藏过的公开作品。</p>
          </a>
          <a className="panel" href="/me/following" style={{ padding: 24, display: "grid", gap: 8, textDecoration: "none", color: "inherit" }}>
            <h2 style={{ margin: 0 }}>我的关注</h2><p style={{ margin: 0, color: "var(--muted)" }}>查看你关注的作者，后续作者发布新作品或更新作品时可收到通知。</p>
          </a>
          <a className="panel" href="/me/proposals" style={{ padding: 24, display: "grid", gap: 8, textDecoration: "none", color: "inherit" }}>
            <h2 style={{ margin: 0 }}>我的提案</h2><p style={{ margin: 0, color: "var(--muted)" }}>查看你提交的改进提案，以及别人给你作品提交的提案。</p>
          </a>
          <a className="panel" href="/me/upgrade" style={{ padding: 24, display: "grid", gap: 8, textDecoration: "none", color: "inherit" }}>
            <h2 style={{ margin: 0 }}>套餐与开通</h2><p style={{ margin: 0, color: "var(--muted)" }}>查看套餐，提交或查看开通申请。</p>
          </a>
        </section>
      ) : loading ? null : (
        <section className="panel" style={{ padding: 24 }}><a className="button-primary" href="/auth">去登录或注册</a></section>
      )}
    </main>
  );
}
