"use client";

import { useEffect, useId, useState, useTransition } from "react";
import { getJSON, postJSON } from "@/lib/api";

type User = {
  id: string;
  username: string;
  email: string;
  role: string;
  planCode: string;
};

type UpgradeRequest = {
  id: string;
  userId: string;
  userEmail: string;
  username: string;
  currentPlan: string;
  targetPlan: string;
  paymentMethod: string;
  payerNote: string;
  systemNote: string;
  status: string;
  adminNote: string;
  reviewedBy: string;
  createdAt: string;
};

type StatusTone = "info" | "success" | "error";

const text = {
  title: "个人中心",
  loading: "正在读取你的账号信息……",
  needLogin: "请先登录后再进入个人中心。",
  login: "去登录或注册",
  currentEmail: "当前邮箱",
  username: "公开名字",
  usernameHint: "会同步更新你的作品地址前缀。",
  usernamePlaceholder: "例如：小雨",
  plan: "当前套餐",
  role: "账号身份",
  openAdmin: "打开管理后台",
  openUpgrade: "开通或升级套餐",
  latestRequest: "最近一次开通请求",
  latestStatus: "当前状态",
  latestPlan: "申请套餐",
  latestMethod: "支付方式",
  latestUserNote: "你的付款备注",
  latestSystemNote: "系统备注",
  noRequest: "你还没有提交过套餐开通请求。",
  planFree: "免费版",
  planLight: "轻享版",
  planSupport: "支持版",
  planAdmin: "最高管理员",
  roleUser: "普通用户",
  roleAdmin: "管理员",
  roleSuperAdmin: "最高管理员",
  save: "保存",
  saveOk: "已保存。",
  saveFail: "保存失败。",
  signOut: "退出登录",
  signOutOk: "你已经退出登录。",
  signOutFail: "退出登录失败。"
};

function planLabel(planCode: string): string {
  switch (planCode) {
    case "light":
      return text.planLight;
    case "support":
      return text.planSupport;
    case "admin":
      return text.planAdmin;
    default:
      return text.planFree;
  }
}

function roleLabel(role: string): string {
  switch (role) {
    case "admin":
      return text.roleAdmin;
    case "super_admin":
      return text.roleSuperAdmin;
    default:
      return text.roleUser;
  }
}

export function PersonalCenter() {
  const [user, setUser] = useState<User | null>(null);
  const [username, setUsername] = useState("");
  const [latestRequest, setLatestRequest] = useState<UpgradeRequest | null>(null);
  const [statusText, setStatusText] = useState(text.loading);
  const [statusTone, setStatusTone] = useState<StatusTone>("info");
  const [loading, setLoading] = useState(true);
  const [isPending, startTransition] = useTransition();
  const usernameId = useId();

  useEffect(() => {
    void loadMe();
  }, []);

  async function loadMe() {
    setLoading(true);
    try {
      const me = await getJSON<User>("/api/v1/me");
      const requestData = await getJSON<{ items: UpgradeRequest[] }>("/api/v1/me/upgrade-requests");
      setUser(me);
      setUsername(me.username);
      setLatestRequest(requestData.items[0] ?? null);
      setStatus("", "info");
    } catch {
      setUser(null);
      setStatus(text.needLogin, "error");
    } finally {
      setLoading(false);
    }
  }

  function setStatus(message: string, tone: StatusTone) {
    setStatusText(message);
    setStatusTone(tone);
  }

  async function saveProfile() {
    try {
      const updated = await postJSON<User>("/api/v1/me/profile", { username });
      setUser(updated);
      setUsername(updated.username);
      setStatus(text.saveOk, "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : text.saveFail, "error");
    }
  }

  async function logout() {
    try {
      await postJSON<{ status: string }>("/api/v1/auth/logout", {});
      setUser(null);
      setStatus(text.signOutOk, "info");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : text.signOutFail, "error");
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 10 }}>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.title}</h1>
        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite">
          {loading ? text.loading : statusText}
        </div>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 18 }}>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
          {user ? (
            <button className="button-ghost" type="button" disabled={isPending} onClick={() => startTransition(logout)}>
              {text.signOut}
            </button>
          ) : null}
        </div>

        {user ? (
          <>
            <div className="field">
              <label>{text.currentEmail}</label>
              <input type="text" value={user.email} readOnly aria-readonly="true" />
            </div>

            <div className="card-grid">
              <div className="field">
                <label>{text.plan}</label>
                <input type="text" value={planLabel(user.planCode)} readOnly aria-readonly="true" />
              </div>
              <div className="field">
                <label>{text.role}</label>
                <input type="text" value={roleLabel(user.role)} readOnly aria-readonly="true" />
              </div>
            </div>

            <div className="field">
              <label htmlFor={usernameId}>{text.username}</label>
              <input
                id={usernameId}
                type="text"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                placeholder={text.usernamePlaceholder}
              />
              <p className="field-note">{text.usernameHint}</p>
            </div>

            <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
              <button className="button-primary" type="button" disabled={isPending} onClick={() => startTransition(saveProfile)}>
                {text.save}
              </button>
              {user.role === "user" ? (
                <a className="button-secondary" href="/me/upgrade">
                  {text.openUpgrade}
                </a>
              ) : null}
              {user.role === "admin" || user.role === "super_admin" ? (
                <a className="button-secondary" href="/admin">
                  {text.openAdmin}
                </a>
              ) : null}
            </div>
          </>
        ) : (
          <div style={{ display: "grid", gap: 12 }}>
            <p style={{ margin: 0, color: "var(--muted)" }}>{text.needLogin}</p>
            <div>
              <a className="button-primary" href="/auth">
                {text.login}
              </a>
            </div>
          </div>
        )}
      </section>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        <h2 style={{ margin: 0 }}>{text.latestRequest}</h2>
        {latestRequest ? (
          <div className="status" style={{ display: "grid", gap: 6 }}>
            <span>{text.latestPlan}：{planLabel(latestRequest.targetPlan)}</span>
            <span>{text.latestMethod}：{latestRequest.paymentMethod === "wechat" ? "微信支付" : "支付宝"}</span>
            <span>{text.latestStatus}：{latestRequest.status}</span>
            <span>{text.latestUserNote}：{latestRequest.payerNote || "未填写"}</span>
            <span>{text.latestSystemNote}：{latestRequest.systemNote || "无"}</span>
          </div>
        ) : (
          <p style={{ margin: 0, color: "var(--muted)" }}>{text.noRequest}</p>
        )}
      </section>
    </section>
  );
}
