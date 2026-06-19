"use client";

import { useEffect, useId, useState } from "react";
import { postJSON } from "@/lib/api";

type AuthCodeResponse = {
  email: string;
  expiresIn: number;
};

type SignedInResponse = {
  status: string;
  user: {
    id: string;
    username: string;
    email: string;
  };
};

type StatusTone = "info" | "success" | "error";

const text = {
  title: "登录或注册",
  email: "邮箱",
  emailPlaceholder: "例如：name@example.com",
  requestCode: "发送验证码",
  resendCode: "重新发送",
  resendCountdown: "重新发送（{seconds} 秒）",
  code: "验证码",
  codePlaceholder: "6 位验证码",
  verifyCode: "进入 PlayPage",
  sentOk: "验证码已经发送到你的邮箱。",
  sentFail: "发送验证码失败。",
  verifyOk: "登录成功，正在进入作品页。",
  verifyFail: "登录失败。",
  needEmail: "请先填写邮箱。",
  needUsername: "请先填写公开名字。",
  needCode: "请先填写验证码。"
};

export function AuthGate() {
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [working, setWorking] = useState(false);
  const [resendSeconds, setResendSeconds] = useState(0);
  const [statusText, setStatusText] = useState("");
  const [statusTone, setStatusTone] = useState<StatusTone>("info");
  const emailId = useId();
  const codeId = useId();

  function setStatus(message: string, tone: StatusTone) {
    setStatusText(message);
    setStatusTone(tone);
  }

  useEffect(() => {
    if (resendSeconds <= 0) {
      return undefined;
    }
    const timer = window.setTimeout(() => {
      setResendSeconds((seconds) => Math.max(0, seconds - 1));
    }, 1000);
    return () => window.clearTimeout(timer);
  }, [resendSeconds]);

  async function requestCode() {
    if (working || resendSeconds > 0) {
      return;
    }
    if (email.trim() === "") {
      setStatus(text.needEmail, "error");
      return;
    }

    try {
      setWorking(true);
      await postJSON<AuthCodeResponse>("/api/v1/auth/request-code", {
        email
      });
      setCodeSent(true);
      setResendSeconds(60);
      setStatus(text.sentOk, "success");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : text.sentFail, "error");
    } finally {
      setWorking(false);
    }
  }

  const requestCodeText = resendSeconds > 0
    ? text.resendCountdown.replace("{seconds}", String(resendSeconds))
    : codeSent
      ? text.resendCode
      : text.requestCode;

  async function verifyCode() {
    if (working) {
      return;
    }
    if (email.trim() === "") {
      setStatus(text.needEmail, "error");
      return;
    }
    if (code.trim() === "") {
      setStatus(text.needCode, "error");
      return;
    }

    try {
      setWorking(true);
      const response = await postJSON<SignedInResponse>("/api/v1/auth/verify-code", {
        email,
        code
      });
      setStatus(text.verifyOk, "success");
      window.setTimeout(() => {
        window.location.href = response.user.username.trim() === ""
          ? "/auth/profile?next=/projects"
          : "/projects";
      }, 500);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : text.verifyFail, "error");
    } finally {
      setWorking(false);
    }
  }

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 10 }}>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.title}</h1>
      </header>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 18, maxWidth: 680 }}>
        <div className="field">
          <label htmlFor={emailId}>{text.email}</label>
          <input
            id={emailId}
            type="email"
            autoComplete="email"
            inputMode="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            placeholder={text.emailPlaceholder}
          />
        </div>

        <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
          <button className="button-primary" type="button" disabled={working || resendSeconds > 0} onClick={requestCode}>
            {requestCodeText}
          </button>
        </div>

        {codeSent ? (
          <>
            <div className="field">
              <label htmlFor={codeId}>{text.code}</label>
              <input
                id={codeId}
                type="text"
                inputMode="numeric"
                autoComplete="one-time-code"
                value={code}
                onChange={(event) => setCode(event.target.value)}
                placeholder={text.codePlaceholder}
              />
            </div>

            <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
              <button className="button-primary" type="button" disabled={working} onClick={verifyCode}>
                {text.verifyCode}
              </button>
            </div>
          </>
        ) : null}

        <div className="status" data-tone={statusTone === "info" ? undefined : statusTone} aria-live="polite">
          {statusText}
        </div>
      </section>
    </section>
  );
}
