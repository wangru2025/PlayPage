"use client";

import { useEffect, useId, useState } from "react";
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
  status: string;
  targetPlan: string;
};

const text = {
  loading: "正在准备付款页面。",
  title: "确认付款",
  intro: "请扫码付款，付款成功后点击“我已付款”发送开通请求。预计 24 小时内完成人工审核。",
  note: "你的备注",
  noteHint: "这里是你自己填写给管理员看的备注，可以写付款账号或其他说明。",
  notePlaceholder: "例如：我是今天中午付款的",
  payDone: "我已付款",
  submitOk: "开通请求已发送，等待人工审核。",
  submitFail: "提交开通请求失败。",
  needNote: "请先填写备注。",
  back: "回到套餐介绍",
  submitting: "正在提交开通请求。",
  pendingExists: "你已经有一个待审核的升级申请，请不要重复提交。"
};

export function UpgradePayment() {
  const [loading, setLoading] = useState(true);
  const [user, setUser] = useState<User | null>(null);
  const [targetPlan, setTargetPlan] = useState("");
  const [paymentMethod, setPaymentMethod] = useState("wechat");
  const [payerNote, setPayerNote] = useState("");
  const [statusText, setStatusText] = useState(text.loading);
  const [submitting, setSubmitting] = useState(false);
  const [hasPendingRequest, setHasPendingRequest] = useState(false);
  const noteId = useId();

  useEffect(() => {
    async function load() {
      try {
        const params = new URLSearchParams(window.location.search);
        setTargetPlan(params.get("plan") ?? "light");
        setPaymentMethod(params.get("method") ?? "wechat");
        const me = await getJSON<User>("/api/v1/me");
        setUser(me);
        const requestData = await getJSON<{ items: UpgradeRequest[] }>("/api/v1/me/upgrade-requests");
        const pending = requestData.items.some((item) => item.status === "pending");
        setHasPendingRequest(pending);
        setStatusText(pending ? text.pendingExists : "");
      } catch (error) {
        setStatusText(error instanceof Error ? error.message : text.loading);
      } finally {
        setLoading(false);
      }
    }
    void load();
  }, []);

  async function submit() {
    if (submitting || hasPendingRequest) {
      setStatusText(text.pendingExists);
      return;
    }
    if (payerNote.trim() === "") {
      setStatusText(text.needNote);
      return;
    }
    try {
      setSubmitting(true);
      setStatusText(text.submitting);
      await postJSON("/api/v1/me/upgrade-requests", {
        targetPlan,
        paymentMethod,
        payerNote
      });
      setHasPendingRequest(true);
      setStatusText(text.submitOk);
    } catch (error) {
      setStatusText(error instanceof Error ? error.message : text.submitFail);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section
      className="panel"
      style={{
        minHeight: "calc(100vh - 48px)",
        display: "grid",
        gridTemplateRows: "1fr auto",
        overflow: "hidden"
      }}
    >
      <div style={{ padding: 28, display: "grid", gap: 18, overflowY: "auto" }}>
        <div>
          <a className="button-secondary" href="/me/upgrade">
            {text.back}
          </a>
        </div>
        <div style={{ display: "grid", gap: 10, justifyItems: "center", textAlign: "center" }}>
          <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.title}</h1>
          <p style={{ margin: 0, color: "var(--muted)", maxWidth: 680 }}>{text.intro}</p>
        </div>

        <div className="panel" style={{ padding: 18, justifySelf: "center", width: "100%", maxWidth: 420 }}>
          <img
            src={paymentMethod === "wechat" ? "/qrcode?name=wei.jpg" : "/qrcode?name=zhifubao.jpg"}
            alt={paymentMethod === "wechat" ? "微信收款二维码" : "支付宝收款二维码"}
            style={{ width: "100%", maxHeight: "52vh", objectFit: "contain", borderRadius: 20, display: "block" }}
          />
        </div>

        <div className="field" style={{ maxWidth: 680, justifySelf: "center", width: "100%" }}>
          <label htmlFor={noteId}>{text.note}</label>
          <input
            id={noteId}
            type="text"
            value={payerNote}
            onChange={(event) => setPayerNote(event.target.value)}
            placeholder={text.notePlaceholder}
          />
          <p className="field-note">{text.noteHint}</p>
        </div>
      </div>

      <div
        style={{
          padding: "18px 28px 28px",
          borderTop: "1px solid var(--line)",
          background: "rgba(255,255,255,0.92)",
          display: "grid",
          gap: 12
        }}
      >
        <div style={{ justifySelf: "center" }}>
          <button className="button-primary" type="button" disabled={loading || !user || submitting || hasPendingRequest} onClick={submit}>
            {submitting ? text.submitting : text.payDone}
          </button>
        </div>

        <div className="status" aria-live="polite">
          {statusText}
        </div>
      </div>
    </section>
  );
}
