"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { deleteJSON, getJSON, patchJSON, postJSON, reportClientError } from "@/lib/api";

type Project = {
  id: string;
  username: string;
  slug: string;
  name: string;
  interactive: boolean;
  visibility: string;
  publicUrl: string;
};

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

type CollectionInput = {
  name?: string;
  permissions: PermissionSet;
  fields: FieldSchema[];
};

type ProjectResponse = {
  project: Project;
};

type CollectionListResponse = {
  items: Collection[];
};

type InteractiveDocResponse = {
  doc: string;
};

const text = {
  title: "互动功能",
  loading: "正在读取互动功能设置。",
  disabled: "这个作品还没有启用互动功能。",
  empty: "这个作品还没有建立任何作品数据表。",
  docButton: "复制 API 文档",
  exportButton: "导出数据表",
  docDone: "API 文档已经复制。",
  docFail: "复制失败，请从下面的文本框手动复制。",
  sectionData: "作品数据",
  sectionHelp: "给 AI 的说明",
  help:
    "把下面生成的 API 文档复制给 AI，再告诉它你想做什么页面、要存什么数据，它就能按这个接口改写网页。",
  path: "作品地址：",
  publicRead: "允许公开读取",
  publicWrite: "允许公开写入",
  fieldRequired: "必填",
  fieldList: "多值",
  fieldRef: "关联",
  idle: "可以开始管理这个作品的互动功能。",
  loadFail: "读取互动功能失败",
  typeLabel: "类型：",
  advancedTitle: "高级数据表管理",
  advancedHelp: "高级用户可以在这里新建、编辑或删除数据表。删除数据表会同时删除其中的记录，请谨慎操作。",
  createCollection: "新建数据表",
  saveCollection: "保存数据表",
  deleteCollection: "删除数据表",
  configJson: "数据表 JSON 配置",
  createTemplate: "新建数据表 JSON",
  createDone: "数据表已创建。",
  saveDone: "数据表已保存。",
  deleteDone: "数据表已删除。",
  invalidJson: "JSON 格式不正确。",
  confirmDeletePrefix: "确定要删除数据表",
  confirmDeleteSuffix: "吗？这会同时删除该表里的所有记录。"
};

function formatPermissions(permissions: PermissionSet): string[] {
  const items: string[] = [];
  if (permissions.publicRead) {
    items.push(text.publicRead);
  }
  if (permissions.publicWrite) {
    items.push(text.publicWrite);
  }
  return items;
}

function collectionToConfigText(collection: Collection): string {
  return JSON.stringify({
    permissions: collection.permissions,
    fields: collection.fields
  }, null, 2);
}

function defaultCollectionConfig(): string {
  return JSON.stringify({
    name: "messages",
    permissions: { publicRead: true, publicWrite: true },
    fields: [
      { name: "content", type: "text", required: true, isList: false },
      { name: "nickname", type: "string", required: false, isList: false }
    ]
  }, null, 2);
}

async function copyText(text: string): Promise<void> {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text);
    return;
  }

  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "true");
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  textarea.style.pointerEvents = "none";
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  const ok = document.execCommand("copy");
  document.body.removeChild(textarea);
  if (!ok) {
    throw new Error("copy failed");
  }
}

export function InteractiveWorkspace({ projectId }: { projectId: string }) {
  const [project, setProject] = useState<Project | null>(null);
  const [apiDoc, setApiDoc] = useState("");
  const [collections, setCollections] = useState<Collection[]>([]);
  const [collectionDrafts, setCollectionDrafts] = useState<Record<string, string>>({});
  const [newCollectionDraft, setNewCollectionDraft] = useState(defaultCollectionConfig);
  const [loading, setLoading] = useState(true);
  const [statusText, setStatusText] = useState("");

  useEffect(() => {
    void loadData();
  }, [projectId]);

  async function loadData() {
    setLoading(true);
    try {
      const [projectData, collectionData, docData] = await Promise.all([
        getJSON<ProjectResponse>(`/api/v1/projects/${projectId}`),
        getJSON<CollectionListResponse>(`/api/v1/projects/${projectId}/collections`),
        getJSON<InteractiveDocResponse>(`/api/v1/projects/${projectId}/interactive-doc`)
      ]);
      setProject(projectData.project);
      setApiDoc(docData.doc || "");
      setCollections(collectionData.items);
      setCollectionDrafts(Object.fromEntries(collectionData.items.map((item) => [item.name, collectionToConfigText(item)])));
      setStatusText("");
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : text.loadFail);
    } finally {
      setLoading(false);
    }
  }

  async function copyDoc() {
    try {
      await copyText(apiDoc);
      setStatusText(text.docDone);
    } catch {
      void reportClientError({
        source: "interactive-copy-doc",
        projectId,
        message: "copy api doc failed"
      });
      setStatusText(text.docFail);
    }
  }

  async function createCollectionFromDraft() {
    try {
      const input = JSON.parse(newCollectionDraft) as CollectionInput;
      if (!input.name || !input.name.trim()) {
        setStatusText("数据表名字不能为空。");
        return;
      }
      await postJSON<Collection>(`/api/v1/projects/${projectId}/collections`, input);
      setStatusText(text.createDone);
      setNewCollectionDraft(defaultCollectionConfig());
      await loadData();
    } catch (error) {
      setStatusText(error instanceof SyntaxError ? text.invalidJson : error instanceof Error ? error.message : text.invalidJson);
    }
  }

  async function saveCollection(collection: Collection) {
    try {
      const input = JSON.parse(collectionDrafts[collection.name] || collectionToConfigText(collection)) as CollectionInput;
      await patchJSON<Collection>(`/api/v1/projects/${projectId}/collections/${encodeURIComponent(collection.name)}`, {
        permissions: input.permissions,
        fields: input.fields
      });
      setStatusText(text.saveDone);
      await loadData();
    } catch (error) {
      setStatusText(error instanceof SyntaxError ? text.invalidJson : error instanceof Error ? error.message : text.invalidJson);
    }
  }

  async function deleteCollection(collection: Collection) {
    if (!window.confirm(`${text.confirmDeletePrefix} ${collection.name} ${text.confirmDeleteSuffix}`)) {
      return;
    }
    try {
      await deleteJSON<{ status: string }>(`/api/v1/projects/${projectId}/collections/${encodeURIComponent(collection.name)}`);
      setStatusText(text.deleteDone);
      await loadData();
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : text.loadFail);
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 10 }}>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.title}</h1>
        {project ? <p style={{ margin: 0, color: "var(--muted)" }}>{text.path}{project.publicUrl}</p> : null}
      </header>

      <div className="status" aria-live="polite">
        {loading ? text.loading : statusText || text.idle}
      </div>

      {!loading && project && !project.interactive ? (
        <section className="panel" style={{ padding: 24 }}>
          <p style={{ margin: 0, color: "var(--muted)" }}>{text.disabled}</p>
        </section>
      ) : null}

      {!loading && project && project.interactive ? (
        <>
          <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
            <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
              <div style={{ display: "grid", gap: 8 }}>
                <h2 style={{ margin: 0 }}>{text.sectionHelp}</h2>
                <p style={{ margin: 0, color: "var(--muted)" }}>{text.help}</p>
              </div>
              <div>
                <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
                  <Link className="button-secondary" href={`/projects/${projectId}/interactive/export`}>
                    {text.exportButton}
                  </Link>
                  <button className="button-primary" type="button" onClick={copyDoc} disabled={!apiDoc}>
                    {text.docButton}
                  </button>
                </div>
              </div>
            </div>
            <label className="field" style={{ display: "grid", gap: 8 }}>
              <span>API 文档文本</span>
              <textarea
                readOnly
                rows={18}
                value={apiDoc}
                onFocus={(event) => event.currentTarget.select()}
                style={{ fontFamily: "monospace", whiteSpace: "pre", overflow: "auto" }}
              />
            </label>
          </section>

          <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
            <div style={{ display: "grid", gap: 8 }}>
              <h2 style={{ margin: 0 }}>{text.advancedTitle}</h2>
              <p style={{ margin: 0, color: "var(--muted)" }}>{text.advancedHelp}</p>
            </div>
            <label className="field" style={{ display: "grid", gap: 8 }}>
              <span>{text.createTemplate}</span>
              <textarea
                rows={10}
                value={newCollectionDraft}
                onChange={(event) => setNewCollectionDraft(event.target.value)}
                style={{ fontFamily: "monospace", whiteSpace: "pre", overflow: "auto" }}
              />
            </label>
            <div>
              <button className="button-primary" type="button" onClick={createCollectionFromDraft}>
                {text.createCollection}
              </button>
            </div>
            {collections.length > 0 ? (
              <div style={{ display: "grid", gap: 14 }}>
                {collections.map((collection) => (
                  <article key={`manage-${collection.id}`} style={{ display: "grid", gap: 10 }}>
                    <label className="field" style={{ display: "grid", gap: 8 }}>
                      <span>{collection.name} {text.configJson}</span>
                      <textarea
                        rows={10}
                        value={collectionDrafts[collection.name] ?? collectionToConfigText(collection)}
                        onChange={(event) => setCollectionDrafts((current) => ({ ...current, [collection.name]: event.target.value }))}
                        style={{ fontFamily: "monospace", whiteSpace: "pre", overflow: "auto" }}
                      />
                    </label>
                    <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
                      <button className="button-secondary" type="button" onClick={() => saveCollection(collection)}>
                        {text.saveCollection}
                      </button>
                      <button className="button-secondary" type="button" onClick={() => deleteCollection(collection)}>
                        {text.deleteCollection}
                      </button>
                    </div>
                  </article>
                ))}
              </div>
            ) : null}
          </section>

          <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
            <h2 style={{ margin: 0 }}>{text.sectionData}</h2>
            {collections.length === 0 ? (
              <p style={{ margin: 0, color: "var(--muted)" }}>{text.empty}</p>
            ) : (
              <div className="project-grid">
                {collections.map((collection) => (
                  <article
                    key={collection.id}
                    style={{
                      display: "grid",
                      gap: 14,
                      padding: 18,
                      borderRadius: 20,
                      border: "1px solid var(--line)",
                      background: "rgba(255,255,255,0.78)"
                    }}
                  >
                    <div style={{ display: "grid", gap: 8 }}>
                      <h3 style={{ margin: 0 }}>{collection.name}</h3>
                      <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
                        {formatPermissions(collection.permissions).map((item) => (
                          <span key={item} className="soft-badge">
                            {item}
                          </span>
                        ))}
                      </div>
                    </div>

                    <div className="card-grid">
                      {collection.fields.map((field) => (
                        <div
                          key={`${collection.id}-${field.name}`}
                          style={{
                            display: "grid",
                            gap: 6,
                            padding: 14,
                            borderRadius: 16,
                            border: "1px solid var(--line)",
                            background: "#fff"
                          }}
                        >
                          <strong>{field.name}</strong>
                          <span style={{ color: "var(--muted)" }}>
                            {text.typeLabel}
                            {field.type}
                          </span>
                          <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
                            {field.required ? <span className="soft-badge">{text.fieldRequired}</span> : null}
                            {field.isList ? <span className="soft-badge">{text.fieldList}</span> : null}
                            {field.reference ? (
                              <span className="soft-badge">
                                {text.fieldRef}
                                {"："}
                                {field.reference}
                              </span>
                            ) : null}
                          </div>
                        </div>
                      ))}
                    </div>
                  </article>
                ))}
              </div>
            )}
          </section>
        </>
      ) : null}
    </section>
  );
}
