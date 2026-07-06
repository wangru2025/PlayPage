"use client";

import { useEffect, useState } from "react";
import { buildWebSocketURL, getJSON, postJSON } from "@/lib/api";

type Settings = {
  appName: string;
  androidEnabled: boolean;
  windowsEnabled: boolean;
  autoUpdate: boolean;
  androidPackageName: string;
  windowsPackageName: string;
};

const emptySettings: Settings = {
  appName: "",
  androidEnabled: false,
  windowsEnabled: false,
  autoUpdate: false,
  androidPackageName: "",
  windowsPackageName: ""
};

type Job = {
  id: string;
  platform: "android" | "windows";
  appName: string;
  packageName: string;
  versionCode: number;
  versionName: string;
  autoUpdate: boolean;
  status: string;
  artifactPath: string;
  errorMessage: string;
  createdAt: string;
  updatedAt: string;
};

const statusLabel: Record<string, string> = {
  pending: "等待构建",
  building: "正在构建",
  succeeded: "构建成功",
  failed: "构建失败",
  skipped: "已跳过"
};

export function AppBuildsPage({ projectId }: { projectId: string }) {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [message, setMessage] = useState("正在读取安装包设置。");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let closed = false;
    let socket: WebSocket | null = null;
    let fallbackTimer: ReturnType<typeof setInterval> | null = null;

    const refresh = () => {
      if (!closed) {
        void load();
      }
    };

    refresh();

    try {
      socket = new WebSocket(buildWebSocketURL(`/api/v1/projects/${projectId}/app-builds/ws`));
      socket.onopen = () => {
        if (fallbackTimer) {
          clearInterval(fallbackTimer);
          fallbackTimer = null;
        }
      };
      socket.onmessage = () => {
        refresh();
      };
      socket.onclose = () => {
        if (!closed && !fallbackTimer) {
          fallbackTimer = setInterval(refresh, 5000);
        }
      };
      socket.onerror = () => {
        socket?.close();
      };
    } catch {
      fallbackTimer = setInterval(refresh, 5000);
    }

    return () => {
      closed = true;
      if (fallbackTimer) {
        clearInterval(fallbackTimer);
      }
      socket?.close();
    };
  }, [projectId]);

  async function load() {
    try {
      const settingsData = await getJSON<{ settings: Settings }>(`/api/v1/projects/${projectId}/app-build-settings`);
      const jobsData = await getJSON<{ items: Job[] }>(`/api/v1/projects/${projectId}/app-builds`);
      setSettings({ ...emptySettings, ...(settingsData.settings ?? {}) });
      setJobs(Array.isArray(jobsData.items) ? jobsData.items : []);
      setMessage("安装包设置已读取。每个账号每天 Android 1 次、Windows 1 次构建额度。");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "读取安装包设置失败。");
    }
  }

  async function saveSettings() {
    if (!settings) return;
    try {
      setBusy(true);
      const data = await postJSON<{ settings: Settings }>(`/api/v1/projects/${projectId}/app-build-settings`, settings);
      setSettings({ ...emptySettings, ...(data.settings ?? {}) });
      setMessage("安装包设置已保存。");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存安装包设置失败。");
    } finally {
      setBusy(false);
    }
  }

  async function createBuild(platform: "android" | "windows") {
    try {
      setBusy(true);
      await postJSON<{ job: Job }>(`/api/v1/projects/${projectId}/app-builds`, {
        platform,
        autoUpdate: settings?.autoUpdate ?? false
      });
      await load();
      setMessage(`${platform === "android" ? "Android" : "Windows"} 安装包构建任务已提交。`);
    } catch (error) {
      await load().catch(() => undefined);
      setMessage(error instanceof Error ? error.message : "提交构建任务失败。");
    } finally {
      setBusy(false);
    }
  }

  if (!settings) {
    return (
      <main className="shell narrow-page">
        <section className="panel"><p>{message}</p></section>
      </main>
    );
  }

  return (
    <main className="shell narrow-page">
      <section className="panel stack-lg">
        <div>
          <p className="eyebrow">离线安装包</p>
          <h1>导出 Android / Windows 安装包</h1>
          <p className="muted">安装包会内置当前作品网页资源。作品里的互动 API 仍会联网访问 PlayPage。自动更新只会构建你勾选的平台。</p>
        </div>
        <p className="status-line">{message}</p>

        <div className="form-grid">
          <label>
            应用名称
            <input value={settings.appName} onChange={(e) => setSettings({ ...settings, appName: e.target.value })} />
          </label>
          <label>
            Android 包名
            <input value={settings.androidPackageName} onChange={(e) => setSettings({ ...settings, androidPackageName: e.target.value })} />
          </label>
          <label>
            Windows 包标识
            <input value={settings.windowsPackageName} onChange={(e) => setSettings({ ...settings, windowsPackageName: e.target.value })} />
          </label>
        </div>

        <div className="stack-sm">
          <label className="check-row"><input type="checkbox" checked={settings.androidEnabled} onChange={(e) => setSettings({ ...settings, androidEnabled: e.target.checked })} /> 需要 Android 安装包</label>
          <label className="check-row"><input type="checkbox" checked={settings.windowsEnabled} onChange={(e) => setSettings({ ...settings, windowsEnabled: e.target.checked })} /> 需要 Windows 安装包</label>
          <label className="check-row"><input type="checkbox" checked={settings.autoUpdate} onChange={(e) => setSettings({ ...settings, autoUpdate: e.target.checked })} /> 作品更新后自动更新已勾选平台的安装包</label>
        </div>

        <div className="actions-row">
          <button className="button-primary" type="button" disabled={busy} onClick={saveSettings}>保存设置</button>
          <button className="button-secondary" type="button" disabled={busy || !settings.androidEnabled} onClick={() => createBuild("android")}>构建 Android</button>
          <button className="button-secondary" type="button" disabled={busy || !settings.windowsEnabled} onClick={() => createBuild("windows")}>构建 Windows</button>
        </div>
      </section>

      <section className="panel stack-md">
        <h2>构建记录</h2>
        {jobs.length === 0 ? <p className="muted">还没有安装包构建记录。</p> : (
          <div className="table-wrap">
            <table className="data-table">
              <thead><tr><th>平台</th><th>版本</th><th>状态</th><th>时间</th><th>下载</th><th>说明</th></tr></thead>
              <tbody>
                {jobs.map((job) => (
                  <tr key={job.id}>
                    <td>{job.platform === "android" ? "Android" : "Windows"}</td>
                    <td>{job.versionName}</td>
                    <td>{statusLabel[job.status] ?? job.status}</td>
                    <td>{formatTime(job.createdAt)}</td>
                    <td>{job.status === "succeeded" && job.artifactPath ? <a href={job.artifactPath}>下载</a> : "—"}</td>
                    <td>{job.errorMessage || (job.autoUpdate ? "自动更新任务" : "手动构建任务")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </main>
  );
}

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "—";
  }
  return date.toLocaleString();
}
