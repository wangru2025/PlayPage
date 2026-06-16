"use client";

import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { getJSON } from "@/lib/api";

type ProjectDailyStats = {
  date: string;
  pageViews: number;
  apiRequests: number;
  apiSuccesses: number;
  apiFailures: number;
};

type ProjectStatsSummary = {
  projectId: string;
  from: string;
  to: string;
  totalPageViews: number;
  totalApiRequests: number;
  totalApiSuccesses: number;
  totalApiFailures: number;
  apiSuccessRate: number;
  apiFailureRate: number;
  items: ProjectDailyStats[];
};

type ProjectStatsPageProps = {
  projectId: string;
};

type StatusTone = "info" | "success" | "error";

const text = {
  back: "返回作品列表",
  title: "统计数据",
  intro: "这里可以查看这个作品的每日访问量和互动 API 请求统计。",
  from: "开始日期",
  to: "结束日期",
  query: "查询统计数据",
  last7: "最近 7 天",
  last30: "最近 30 天",
  loading: "正在读取统计数据。",
  ready: "统计数据已更新。",
  fail: "读取统计数据失败。",
  empty: "这个日期范围内还没有统计数据。",
  totalPageViews: "总访问量",
  totalApiRequests: "互动 API 请求次数",
  totalApiSuccesses: "成功次数",
  totalApiFailures: "失败次数",
  apiSuccessRate: "成功率",
  apiFailureRate: "失败率",
  tableLabel: "每日统计数据",
  date: "日期",
  pageViews: "访问量",
  apiRequests: "API 请求次数",
  apiSuccesses: "成功次数",
  apiFailures: "失败次数"
};

function formatDate(date: Date): string {
  return date.toISOString().slice(0, 10);
}

function defaultRange(days: number): { from: string; to: string } {
  const to = new Date();
  const from = new Date();
  from.setDate(to.getDate() - (days - 1));
  return { from: formatDate(from), to: formatDate(to) };
}

function formatRate(value: number): string {
  if (!Number.isFinite(value)) {
    return "0%";
  }
  return `${(value * 100).toFixed(1)}%`;
}

function rowSuccessRate(item: ProjectDailyStats): number {
  return item.apiRequests > 0 ? item.apiSuccesses / item.apiRequests : 0;
}

function rowFailureRate(item: ProjectDailyStats): number {
  return item.apiRequests > 0 ? item.apiFailures / item.apiRequests : 0;
}

export function ProjectStatsPage({ projectId }: ProjectStatsPageProps) {
  const initialRange = useMemo(() => defaultRange(30), []);
  const [from, setFrom] = useState(initialRange.from);
  const [to, setTo] = useState(initialRange.to);
  const [stats, setStats] = useState<ProjectStatsSummary | null>(null);
  const [statusText, setStatusText] = useState(text.loading);
  const [statusTone, setStatusTone] = useState<StatusTone>("info");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    void loadStats(initialRange.from, initialRange.to);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  async function loadStats(nextFrom = from, nextTo = to) {
    try {
      setLoading(true);
      setStatusText(text.loading);
      setStatusTone("info");
      const data = await getJSON<ProjectStatsSummary>(`/api/v1/projects/${projectId}/stats?from=${encodeURIComponent(nextFrom)}&to=${encodeURIComponent(nextTo)}`);
      setStats(data);
      setFrom(data.from);
      setTo(data.to);
      setStatusText(text.ready);
      setStatusTone("success");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : text.fail);
      setStatusTone("error");
    } finally {
      setLoading(false);
    }
  }

  function applyQuickRange(days: number) {
    const range = defaultRange(days);
    setFrom(range.from);
    setTo(range.to);
    void loadStats(range.from, range.to);
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 10 }}>
        <div>
          <a className="button-secondary" href="/projects">
            {text.back}
          </a>
        </div>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.title}</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>{text.intro}</p>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite">
          {statusText}
        </div>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }} aria-label="统计日期筛选">
        <div className="card-grid">
          <div className="field">
            <label htmlFor="stats-from">{text.from}</label>
            <input id="stats-from" type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
          </div>
          <div className="field">
            <label htmlFor="stats-to">{text.to}</label>
            <input id="stats-to" type="date" value={to} onChange={(event) => setTo(event.target.value)} />
          </div>
        </div>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <button className="button-primary" type="button" disabled={loading} aria-busy={loading} onClick={() => loadStats()}>
            {text.query}
          </button>
          <button className="button-secondary" type="button" disabled={loading} onClick={() => applyQuickRange(7)}>
            {text.last7}
          </button>
          <button className="button-secondary" type="button" disabled={loading} onClick={() => applyQuickRange(30)}>
            {text.last30}
          </button>
        </div>
      </section>

      {stats ? (
        <>
          <section className="card-grid" aria-label="统计汇总">
            <StatCard label={text.totalPageViews} value={String(stats.totalPageViews)} />
            <StatCard label={text.totalApiRequests} value={String(stats.totalApiRequests)} />
            <StatCard label={text.totalApiSuccesses} value={String(stats.totalApiSuccesses)} />
            <StatCard label={text.totalApiFailures} value={String(stats.totalApiFailures)} />
            <StatCard label={text.apiSuccessRate} value={formatRate(stats.apiSuccessRate)} />
            <StatCard label={text.apiFailureRate} value={formatRate(stats.apiFailureRate)} />
          </section>

          <section className="panel" style={{ padding: 24, display: "grid", gap: 14 }}>
            <h2 style={{ margin: 0 }}>{text.tableLabel}</h2>
            {stats.items.length === 0 ? (
              <p style={{ margin: 0, color: "var(--muted)" }}>{text.empty}</p>
            ) : (
              <div style={{ overflowX: "auto" }}>
                <table style={{ width: "100%", borderCollapse: "collapse" }}>
                  <thead>
                    <tr>
                      <TableHead>{text.date}</TableHead>
                      <TableHead>{text.pageViews}</TableHead>
                      <TableHead>{text.apiRequests}</TableHead>
                      <TableHead>{text.apiSuccesses}</TableHead>
                      <TableHead>{text.apiFailures}</TableHead>
                      <TableHead>{text.apiSuccessRate}</TableHead>
                      <TableHead>{text.apiFailureRate}</TableHead>
                    </tr>
                  </thead>
                  <tbody>
                    {stats.items.map((item) => (
                      <tr key={item.date}>
                        <TableCell>{item.date}</TableCell>
                        <TableCell>{item.pageViews}</TableCell>
                        <TableCell>{item.apiRequests}</TableCell>
                        <TableCell>{item.apiSuccesses}</TableCell>
                        <TableCell>{item.apiFailures}</TableCell>
                        <TableCell>{formatRate(rowSuccessRate(item))}</TableCell>
                        <TableCell>{formatRate(rowFailureRate(item))}</TableCell>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </>
      ) : null}
    </section>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <article className="panel" style={{ padding: 20, display: "grid", gap: 6 }}>
      <p style={{ margin: 0, color: "var(--muted)" }}>{label}</p>
      <strong style={{ fontSize: "1.8rem" }}>{value}</strong>
    </article>
  );
}

function TableHead({ children }: { children: ReactNode }) {
  return <th style={{ textAlign: "left", borderBottom: "1px solid var(--line)", padding: "10px 8px" }}>{children}</th>;
}

function TableCell({ children }: { children: ReactNode }) {
  return <td style={{ borderBottom: "1px solid var(--line)", padding: "10px 8px" }}>{children}</td>;
}
