"use client";

import { useEffect, useId, useState } from "react";
import { getJSON, postJSON } from "@/lib/api";

type User = {
  id: string;
  username: string;
  email: string;
};

type StatusTone = "info" | "success" | "error";

const text = {
  title: "设置公开名字",
  intro: "先设置一个公开名字，之后再继续。",
  username: "公开名字",
  usernamePlaceholder: "例如：小雨",
  usernameHint: "这个名字会出现在你的作品地址前缀里。",
  save: "继续",
  loading: "正在读取你的账号信息。",
  needLogin: "请先登录，再设置公开名字。",
  needUsername: "请先填写公开名字。",
  saveOk: "已经设置好，正在继续。",
  saveFail: "保存失败。",
  backAuth: "返回登录页"
};

export function CompleteProfile() {
  const [username, setUsername] = useState("");
  const [working, setWorking] = useState(false);
  const [loading, setLoading] = useState(true);
  const [statusText, setStatusText] = useState(text.loading);
  const [statusTone, setStatusTone] = useState<StatusTone>("info");
  const usernameId = useId();

  function setStatus(message: string, tone: StatusTone) {
    setStatusText(message);
    setStatusTone(tone);
  }

  function getNextURL(): string {
    const params = new URLSearchParams(window.location.search);
    const next = params.get("next") ?? "/projects";
    return next.startsWith("/") ? next : "/projects";
  }

  useEffect(() => {
    async function loadMe() {
      try {
        const me = await getJSON<User>("/api/v1/me");
        if (me.username.trim() !== "") {
          window.location.href = getNextURL();
          return;
        }
        setStatus("", "info");
      } catch {
        setStatus(text.needLogin, "error");
      } finally {
        setLoading(false);
      }
    }

    void loadMe();
  }, []);

  async function submit() {
    if (working) {
      return;
    }
    if (username.trim() === "") {
      setStatus(text.needUsername, "error");
      return;
    }

    try {
      setWorking(true);
      await postJSON<User>("/api/v1/me/profile", { username });
      setStatus(text.saveOk, "success");
      window.setTimeout(() => {
        window.location.href = getNextURL();
      }, 400);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : text.saveFail, "error");
    } finally {
      setWorking(false);
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 10 }}>
        <div>
          <a className="button-secondary" href="/auth">
            {text.backAuth}
          </a>
        </div>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.title}</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>{text.intro}</p>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 18, maxWidth: 680 }}>
        <div className="field">
          <label htmlFor={usernameId}>{text.username}</label>
          <input
            id={usernameId}
            type="text"
            autoComplete="nickname"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            placeholder={text.usernamePlaceholder}
            disabled={loading}
          />
          <p className="field-note">{text.usernameHint}</p>
        </div>

        <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
          <button className="button-primary" type="button" disabled={working || loading} onClick={submit}>
            {text.save}
          </button>
        </div>

        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite">
          {loading ? text.loading : statusText}
        </div>
      </section>
    </section>
  );
}
