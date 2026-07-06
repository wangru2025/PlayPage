"use client";

import { useEffect, useState } from "react";
import { getJSON } from "@/lib/api";

type AuthorSummary = {
  userId: string;
  username: string;
  displayName: string;
  joinedAt: string;
  projectCount: number;
  followersCount: number;
};

export function FollowingPage() {
  const [items, setItems] = useState<AuthorSummary[]>([]);
  const [status, setStatus] = useState("正在读取关注列表。");
  const [loading, setLoading] = useState(true);

  useEffect(() => { void load(); }, []);

  async function load() {
    try {
      setLoading(true);
      const data = await getJSON<{ items: AuthorSummary[] }>("/api/v1/me/following");
      setItems(data.items);
      setStatus(data.items.length ? "关注列表已读取。" : "你还没有关注作者。");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "读取关注列表失败，请先登录。");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <section className="panel" style={{ padding: 28, display: "grid", gap: 10 }}>
        <p style={{ margin: 0, color: "var(--muted)" }}>个人中心</p>
        <h1 style={{ margin: 0, fontSize: "2.2rem" }}>我的关注</h1>
        <div className="status" aria-live="polite">{loading ? "正在读取关注列表。" : status}</div>
      </section>

      {items.length ? (
        <section className="card-grid" aria-label="关注作者列表">
          {items.map((item) => (
            <article key={item.userId} className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
              <h2 style={{ margin: 0 }}>@{item.displayName || item.username}</h2>
              <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                <span className="soft-badge">公开作品 {item.projectCount}</span>
                <span className="soft-badge">关注者 {item.followersCount}</span>
              </div>
              <div>
                <a className="button-primary" href={`/@${encodeURIComponent(item.username)}`}>查看作者主页</a>
              </div>
            </article>
          ))}
        </section>
      ) : null}
    </section>
  );
}
