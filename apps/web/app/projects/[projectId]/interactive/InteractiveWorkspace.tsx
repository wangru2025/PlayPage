"use client";

import { useEffect, useState } from "react";
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
  back: "\u8fd4\u56de\u4f5c\u54c1\u5217\u8868",
  title: "\u4e92\u52a8\u529f\u80fd",
  loading: "\u6b63\u5728\u8bfb\u53d6\u4e92\u52a8\u529f\u80fd\u8bbe\u7f6e\u3002",
  disabled: "\u8fd9\u4e2a\u4f5c\u54c1\u8fd8\u6ca1\u6709\u542f\u7528\u4e92\u52a8\u529f\u80fd\u3002",
  empty: "\u8fd9\u4e2a\u4f5c\u54c1\u8fd8\u6ca1\u6709\u5efa\u7acb\u4efb\u4f55\u4f5c\u54c1\u6570\u636e\u8868\u3002",
  docButton: "\u590d\u5236 API \u6587\u6863",
  docDone: "API \u6587\u6863\u5df2\u7ecf\u590d\u5236\u3002",
  docFail: "\u590d\u5236\u5931\u8d25\uff0c\u8bf7\u4ece\u4e0b\u9762\u7684\u6587\u672c\u6846\u624b\u52a8\u590d\u5236\u3002",
  sectionData: "\u4f5c\u54c1\u6570\u636e",
  sectionHelp: "\u7ed9 AI \u7684\u8bf4\u660e",
  help:
    "\u628a\u4e0b\u9762\u751f\u6210\u7684 API \u6587\u6863\u590d\u5236\u7ed9 AI\uff0c\u518d\u544a\u8bc9\u5b83\u4f60\u60f3\u505a\u4ec0\u4e48\u9875\u9762\u3001\u8981\u5b58\u4ec0\u4e48\u6570\u636e\uff0c\u5b83\u5c31\u80fd\u6309\u8fd9\u4e2a\u63a5\u53e3\u6539\u5199\u7f51\u9875\u3002",
  path: "\u4f5c\u54c1\u5730\u5740\uff1a",
  publicRead: "\u5141\u8bb8\u516c\u5f00\u8bfb\u53d6",
  publicWrite: "\u5141\u8bb8\u516c\u5f00\u5199\u5165",
  fieldRequired: "\u5fc5\u586b",
  fieldList: "\u591a\u503c",
  fieldRef: "\u5173\u8054",
  idle: "\u53ef\u4ee5\u5f00\u59cb\u7ba1\u7406\u8fd9\u4e2a\u4f5c\u54c1\u7684\u4e92\u52a8\u529f\u80fd\u3002",
  loadFail: "\u8bfb\u53d6\u4e92\u52a8\u529f\u80fd\u5931\u8d25",
  typeLabel: "\u7c7b\u578b\uff1a",
  advancedTitle: "\u9ad8\u7ea7\u6570\u636e\u8868\u7ba1\u7406",
  advancedHelp: "\u9ad8\u7ea7\u7528\u6237\u53ef\u4ee5\u5728\u8fd9\u91cc\u65b0\u5efa\u3001\u7f16\u8f91\u6216\u5220\u9664\u6570\u636e\u8868\u3002\u5220\u9664\u6570\u636e\u8868\u4f1a\u540c\u65f6\u5220\u9664\u5176\u4e2d\u7684\u8bb0\u5f55\uff0c\u8bf7\u8c28\u614e\u64cd\u4f5c\u3002",
  createCollection: "\u65b0\u5efa\u6570\u636e\u8868",
  saveCollection: "\u4fdd\u5b58\u6570\u636e\u8868",
  deleteCollection: "\u5220\u9664\u6570\u636e\u8868",
  configJson: "\u6570\u636e\u8868 JSON \u914d\u7f6e",
  createTemplate: "\u65b0\u5efa\u6570\u636e\u8868 JSON",
  createDone: "\u6570\u636e\u8868\u5df2\u521b\u5efa\u3002",
  saveDone: "\u6570\u636e\u8868\u5df2\u4fdd\u5b58\u3002",
  deleteDone: "\u6570\u636e\u8868\u5df2\u5220\u9664\u3002",
  invalidJson: "JSON \u683c\u5f0f\u4e0d\u6b63\u786e\u3002",
  confirmDeletePrefix: "\u786e\u5b9a\u8981\u5220\u9664\u6570\u636e\u8868",
  confirmDeleteSuffix: "\u5417\uff1f\u8fd9\u4f1a\u540c\u65f6\u5220\u9664\u8be5\u8868\u91cc\u7684\u6240\u6709\u8bb0\u5f55\u3002"
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
        setStatusText("\u6570\u636e\u8868\u540d\u5b57\u4e0d\u80fd\u4e3a\u7a7a\u3002");
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
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
          <a className="button-secondary" href="/projects">
            {text.back}
          </a>
        </div>
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
                <button className="button-primary" type="button" onClick={copyDoc} disabled={!apiDoc}>
                  {text.docButton}
                </button>
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
