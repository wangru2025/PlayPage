"use client";

import { useEffect, useMemo, useState } from "react";
import { getJSON } from "@/lib/api";
import type { ProjectTemplate, TemplateCategory } from "./templateTypes";

type TemplateListResponse = {
  items: ProjectTemplate[];
  categories: TemplateCategory[];
};

const text = {
  title: "模板市场",
  intro: "先把模板选择流程搭起来。后续这里可以放论坛、个人主页、小游戏、工具页等模板。",
  submit: "投稿模板",
  search: "搜索模板",
  searchPlaceholder: "输入模板名称、标签或用途",
  category: "分类",
  allCategories: "全部分类",
  loading: "正在读取模板。",
  loaded: "模板已读取。",
  empty: "没有找到符合条件的模板。",
  detail: "查看模板",
  use: "使用这个模板",
  official: "官方模板",
  author: "作者",
  interactiveRequired: "需要互动功能",
  analyticsRecommended: "建议开启统计"
};

export function TemplateMarketPage() {
  const [items, setItems] = useState<ProjectTemplate[]>([]);
  const [categories, setCategories] = useState<TemplateCategory[]>([]);
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("");
  const [statusText, setStatusText] = useState(text.loading);
  const [statusTone, setStatusTone] = useState<"info" | "success" | "error">("info");

  async function loadTemplates() {
    try {
      setStatusText(text.loading);
      setStatusTone("info");
      const params = new URLSearchParams();
      if (query.trim()) params.set("q", query.trim());
      if (category) params.set("category", category);
      const payload = await getJSON<TemplateListResponse>(`/api/v1/templates${params.toString() ? `?${params}` : ""}`);
      setItems(payload.items);
      setCategories(payload.categories);
      setStatusText(text.loaded);
      setStatusTone("success");
    } catch (error) {
      setItems([]);
      setStatusText(error instanceof Error ? error.message : "读取模板失败。");
      setStatusTone("error");
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadTemplates();
    }, 150);
    return () => window.clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query, category]);

  const categoryOptions = useMemo(() => categories, [categories]);

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.title}</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 820 }}>{text.intro}</p>
        <div>
          <a className="button-primary" href="/templates/submit">{text.submit}</a>
        </div>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite">
          {statusText}
        </div>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        <div className="card-grid">
          <label className="field">
            <span>{text.search}</span>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={text.searchPlaceholder} />
          </label>
          <label className="field">
            <span>{text.category}</span>
            <select value={category} onChange={(event) => setCategory(event.target.value)}>
              <option value="">{text.allCategories}</option>
              {categoryOptions.map((item) => (
                <option key={item.id} value={item.id}>{item.label}</option>
              ))}
            </select>
          </label>
        </div>
      </section>

      <section className="project-grid" aria-label="模板列表">
        {items.length === 0 ? (
          <div className="panel" style={{ padding: 24 }}>
            <p style={{ margin: 0, color: "var(--muted)" }}>{text.empty}</p>
          </div>
        ) : null}
        {items.map((item) => (
          <article key={item.id} className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
            <div style={{ display: "flex", gap: 12, flexWrap: "wrap", alignItems: "center", justifyContent: "space-between" }}>
              <div>
                <h2 style={{ margin: 0 }}>{item.name}</h2>
                <p style={{ margin: "4px 0 0", color: "var(--muted)" }}>
                  {item.categoryLabel} · {item.source === "community" ? `${text.author}：${item.authorName || "未知作者"}` : text.official}
                </p>
              </div>
              <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                {item.interactiveRequired ? <span className="soft-badge">{text.interactiveRequired}</span> : null}
                {item.analyticsRecommended ? <span className="soft-badge">{text.analyticsRecommended}</span> : null}
              </div>
            </div>
            <p style={{ margin: 0 }}>{item.summary}</p>
            <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
              {item.tags.map((tag) => <span key={tag} className="soft-badge">{tag}</span>)}
            </div>
            <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
              <a className="button-secondary" href={`/templates/${encodeURIComponent(item.slug)}`}>{text.detail}</a>
              <a className="button-primary" href={`/projects/new?source=template&templateId=${encodeURIComponent(item.id)}`}>{text.use}</a>
            </div>
          </article>
        ))}
      </section>
    </section>
  );
}
