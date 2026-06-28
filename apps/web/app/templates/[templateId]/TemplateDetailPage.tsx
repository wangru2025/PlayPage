"use client";

import { useEffect, useState } from "react";
import { getJSON } from "@/lib/api";
import type { ProjectTemplate } from "../templateTypes";

type TemplateDetailResponse = {
  template: ProjectTemplate;
};

const text = {
  loading: "正在读取模板。",
  use: "使用这个模板创建作品",
  back: "返回模板市场",
  configTitle: "可配置参数",
  dataTitle: "模板数据表",
  noCollections: "这个模板暂时不需要自动创建数据表。",
  interactiveRequired: "这个模板需要互动功能。",
  analyticsRecommended: "建议开启访问统计。",
  official: "官方模板",
  author: "作者",
  openPreview: "打开完整预览"
};

export function TemplateDetailPage({ templateId }: { templateId: string }) {
  const [template, setTemplate] = useState<ProjectTemplate | null>(null);
  const [statusText, setStatusText] = useState(text.loading);
  const [statusTone, setStatusTone] = useState<"info" | "success" | "error">("info");

  useEffect(() => {
    async function load() {
      try {
        const payload = await getJSON<TemplateDetailResponse>(`/api/v1/templates/${encodeURIComponent(templateId)}`);
        setTemplate(payload.template);
        setStatusText("模板已读取。");
        setStatusTone("success");
      } catch (error) {
        setTemplate(null);
        setStatusText(error instanceof Error ? error.message : "读取模板失败。");
        setStatusTone("error");
      }
    }
    void load();
  }, [templateId]);

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <nav aria-label="模板详情导航" style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <a className="button-secondary" href="/templates">{text.back}</a>
        </nav>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{template ? template.name : "模板详情"}</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>
          {template ? `${template.categoryLabel} · ${template.source === "community" ? `${text.author}：${template.authorName || "未知作者"}` : text.official}` : "查看模板信息。"}
        </p>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite">
          {statusText}
        </div>
      </header>

      {template ? (
        <>
          <section className="panel" style={{ padding: 24, display: "grid", gap: 14 }}>
            <p style={{ margin: 0, fontSize: "1.1rem" }}>{template.description}</p>
            <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
              {template.tags.map((tag) => <span key={tag} className="soft-badge">{tag}</span>)}
              {template.interactiveRequired ? <span className="soft-badge">{text.interactiveRequired}</span> : null}
              {template.analyticsRecommended ? <span className="soft-badge">{text.analyticsRecommended}</span> : null}
            </div>
            <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
              <a className="button-primary" href={`/projects/new?source=template&templateId=${encodeURIComponent(template.id)}`}>
                {text.use}
              </a>
              <a className="button-secondary" href={`/templates/${encodeURIComponent(template.id)}/preview`} target="_blank" rel="noreferrer">
                {text.openPreview}
              </a>
            </div>
          </section>

          <section className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
            <h2 style={{ margin: 0 }}>{text.configTitle}</h2>
            {template.configFields.map((field) => (
              <div key={field.name} style={{ border: "1px solid var(--line)", borderRadius: 16, padding: 14 }}>
                <strong>{field.label}</strong>
                <p style={{ margin: "4px 0 0", color: "var(--muted)" }}>
                  类型：{field.type}；默认值：{field.default || "无"}{field.required ? "；必填" : ""}
                </p>
              </div>
            ))}
          </section>

          <section className="panel" style={{ padding: 24, display: "grid", gap: 12 }}>
            <h2 style={{ margin: 0 }}>{text.dataTitle}</h2>
            {template.collections.length === 0 ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.noCollections}</p> : null}
            {template.collections.map((collection) => (
              <div key={collection.name} style={{ border: "1px solid var(--line)", borderRadius: 16, padding: 14 }}>
                <strong>{collection.name}</strong>
                <p style={{ margin: "4px 0 0", color: "var(--muted)" }}>
                  字段：{collection.fields.map((field) => field.name).join("、") || "无"}
                </p>
              </div>
            ))}
          </section>
        </>
      ) : null}
    </section>
  );
}
