"use client";

import { useEffect, useRef, useState } from "react";
import { getJSON, postBlob } from "@/lib/api";

type Project = {
  id: string;
  name: string;
  publicUrl: string;
};

type ProjectResponse = { project: Project };

type FieldSchema = {
  name: string;
  type: string;
  required: boolean;
  isList: boolean;
  reference?: string;
};

type PermissionSet = {
  publicRead: boolean;
  publicWrite: boolean;
};

type Collection = {
  id: string;
  name: string;
  permissions: PermissionSet;
  fields: FieldSchema[];
};

type CollectionListResponse = { items: Collection[] };

type StatusTone = "info" | "success" | "error";
type ExportFormat = "json" | "word";

type Props = { projectId: string };

const pageText = {
  title: "导出作品数据表",
  intro: "选择要导出的数据表/集合。导出的内容只包含当前作品互动数据表和记录，不包含平台登录会话或其他作品数据。",
  loading: "正在读取数据表。",
  ready: "请选择要导出的数据表。",
  empty: "这个作品还没有数据表可以导出。",
  exportSelected: "导出所选数据",
  selectAll: "全选",
  selectNone: "全不选",
  chooseFormat: "选择导出格式",
  chooseFormatHelp: "JSON 适合迁移和备份；Word 表格适合人工查看和整理。",
  json: "导出 JSON",
  word: "导出 Word 表格",
  cancel: "取消",
  noSelection: "请至少选择一个数据表。",
  exporting: "正在导出数据，请稍候。",
  done: "导出文件已生成。",
  failed: "导出数据失败。"
};

function downloadBlob(filename: string, blob: Blob) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

export function InteractiveDataExportPage({ projectId }: Props) {
  const [project, setProject] = useState<Project | null>(null);
  const [collections, setCollections] = useState<Collection[]>([]);
  const [selected, setSelected] = useState<Record<string, boolean>>({});
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);
  const [statusText, setStatusText] = useState(pageText.loading);
  const [statusTone, setStatusTone] = useState<StatusTone>("info");
  const dialogRef = useRef<HTMLDialogElement | null>(null);

  const selectedCollections = collections.filter((item) => selected[item.name]);

  useEffect(() => {
    void loadData();
  }, [projectId]);

  async function loadData() {
    setLoading(true);
    try {
      const [projectData, collectionData] = await Promise.all([
        getJSON<ProjectResponse>(`/api/v1/projects/${projectId}`),
        getJSON<CollectionListResponse>(`/api/v1/projects/${projectId}/collections`)
      ]);
      setProject(projectData.project);
      setCollections(collectionData.items);
      setSelected(Object.fromEntries(collectionData.items.map((item) => [item.name, true])));
      setStatusText(collectionData.items.length === 0 ? pageText.empty : pageText.ready);
      setStatusTone(collectionData.items.length === 0 ? "info" : "success");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : pageText.failed);
      setStatusTone("error");
    } finally {
      setLoading(false);
    }
  }

  function openFormatDialog() {
    if (working) return;
    if (selectedCollections.length === 0) {
      setStatusText(pageText.noSelection);
      setStatusTone("error");
      return;
    }
    dialogRef.current?.showModal();
  }

  async function exportData(format: ExportFormat) {
    if (!project || working) return;
    dialogRef.current?.close();
    setWorking(true);
    setStatusText(pageText.exporting);
    setStatusTone("info");
    try {
      const response = await postBlob(`/api/v1/projects/${projectId}/data-export`, {
        format,
        collections: selectedCollections.map((collection) => collection.name)
      });
      downloadBlob(response.filename, response.blob);
      setStatusText(pageText.done);
      setStatusTone("success");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : pageText.failed);
      setStatusTone("error");
    } finally {
      setWorking(false);
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 12 }}>
        <h1 style={{ margin: 0, fontSize: "2.4rem" }}>{pageText.title}</h1>
        <p style={{ margin: 0, color: "var(--muted)", maxWidth: 820 }}>{pageText.intro}</p>
        {project ? <p style={{ margin: 0, color: "var(--muted)" }}>作品：{project.name}</p> : null}
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite" role="status">
          {loading ? pageText.loading : statusText}
        </div>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          <button className="button-secondary" type="button" onClick={() => setSelected(Object.fromEntries(collections.map((item) => [item.name, true])))} disabled={loading || collections.length === 0 || working}>
            {pageText.selectAll}
          </button>
          <button className="button-secondary" type="button" onClick={() => setSelected({})} disabled={loading || collections.length === 0 || working}>
            {pageText.selectNone}
          </button>
          <button className="button-primary" type="button" onClick={openFormatDialog} disabled={loading || collections.length === 0 || working} aria-busy={working}>
            {working ? pageText.exporting : pageText.exportSelected}
          </button>
        </div>

        {collections.length === 0 && !loading ? <p style={{ margin: 0, color: "var(--muted)" }}>{pageText.empty}</p> : null}
        <div style={{ display: "grid", gap: 10 }}>
          {collections.map((collection) => (
            <label key={collection.id} className="panel" style={{ padding: 16, display: "grid", gap: 8, cursor: "pointer" }}>
              <span style={{ display: "flex", gap: 10, alignItems: "center", flexWrap: "wrap" }}>
                <input
                  type="checkbox"
                  checked={Boolean(selected[collection.name])}
                  onChange={(event) => setSelected((current) => ({ ...current, [collection.name]: event.target.checked }))}
                  disabled={working}
                />
                <strong>{collection.name}</strong>
                <span className="soft-badge">字段 {collection.fields.length}</span>
              </span>
              <span style={{ color: "var(--muted)" }}>
                字段：{collection.fields.map((field) => field.name).join("、") || "无字段"}
              </span>
            </label>
          ))}
        </div>
      </section>

      <dialog ref={dialogRef} style={{ border: "1px solid var(--line)", borderRadius: 18, padding: 24, maxWidth: 460 }} aria-labelledby="export-format-title">
        <form method="dialog" style={{ display: "grid", gap: 14 }}>
          <h2 id="export-format-title" style={{ margin: 0 }}>{pageText.chooseFormat}</h2>
          <p style={{ margin: 0, color: "var(--muted)" }}>{pageText.chooseFormatHelp}</p>
          <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
            <button className="button-primary" type="button" onClick={() => exportData("json")}>{pageText.json}</button>
            <button className="button-secondary" type="button" onClick={() => exportData("word")}>{pageText.word}</button>
            <button className="button-ghost" value="cancel">{pageText.cancel}</button>
          </div>
        </form>
      </dialog>
    </section>
  );
}
