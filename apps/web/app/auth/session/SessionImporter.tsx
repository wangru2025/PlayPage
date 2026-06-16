"use client";

import { useEffect, useState } from "react";

const text = {
  title: "正在导入登录状态",
  working: "正在把测试账号的登录状态写入浏览器。",
  done: "已经写入完成，正在跳转。",
  fail: "缺少登录参数，无法继续。"
};

export function SessionImporter() {
  const [message, setMessage] = useState(text.working);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get("token") ?? "";
    const next = params.get("next") || "/projects";
    if (token.trim() === "") {
      setMessage(text.fail);
      return;
    }

    document.cookie = `web_wangru_session=${token}; Path=/; Max-Age=2592000; Secure; SameSite=Lax`;
    setMessage(text.done);
    window.setTimeout(() => {
      window.location.href = next;
    }, 300);
  }, []);

  return (
    <section className="panel" style={{ padding: 28, display: "grid", gap: 10 }}>
      <h1 style={{ margin: 0, fontSize: "2rem" }}>{text.title}</h1>
      <p style={{ margin: 0, color: "var(--muted)" }}>{message}</p>
    </section>
  );
}
