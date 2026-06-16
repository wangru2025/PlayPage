"use client";

import { useState } from "react";

const plans = [
  {
    code: "light",
    name: "轻享版",
    price: "3 元 / 月",
    rights: ["最多 10 个作品", "每作品 30 MB 互动数据", "每月 100000 次查询", "每月 10000 次写入"]
  },
  {
    code: "support",
    name: "支持版",
    price: "6 元 / 月",
    rights: ["最多 30 个作品", "每作品 100 MB 互动数据", "每月 500000 次查询", "每月 50000 次写入"]
  }
] as const;

const text = {
  title: "开通或升级套餐",
  intro: "先看清每个套餐的价格和权益，再选择支付方式继续付款。",
  back: "回到个人中心",
  chooseMethod: "选择支付方式",
  next: "确定支付"
};

export function UpgradePlans() {
  const [targetPlan, setTargetPlan] = useState("light");
  const [paymentMethod, setPaymentMethod] = useState("wechat");

  return (
    <section style={{ display: "grid", gap: 18 }}>
      <header className="panel" style={{ padding: 28, display: "grid", gap: 10 }}>
        <div>
          <a className="button-secondary" href="/me">
            {text.back}
          </a>
        </div>
        <h1 style={{ margin: 0, fontSize: "2.5rem" }}>{text.title}</h1>
        <p style={{ margin: 0, color: "var(--muted)" }}>{text.intro}</p>
      </header>

      <section className="card-grid">
        {plans.map((plan) => (
          <article key={plan.code} className="panel" style={{ padding: 24, display: "grid", gap: 14 }}>
            <label style={{ display: "grid", gap: 12, cursor: "pointer" }}>
              <div style={{ display: "flex", justifyContent: "space-between", gap: 12, alignItems: "center" }}>
                <strong style={{ fontSize: "1.2rem" }}>{plan.name}</strong>
                <input type="radio" name="target-plan" checked={targetPlan === plan.code} onChange={() => setTargetPlan(plan.code)} />
              </div>
              <span style={{ color: "var(--muted)" }}>{plan.price}</span>
              <ul style={{ margin: 0, paddingLeft: 18, color: "var(--muted)" }}>
                {plan.rights.map((item) => <li key={item}>{item}</li>)}
              </ul>
            </label>
          </article>
        ))}
      </section>

      <section className="panel" style={{ padding: 24, display: "grid", gap: 16 }}>
        <strong>{text.chooseMethod}</strong>
        <label><input type="radio" name="payment-method" checked={paymentMethod === "wechat"} onChange={() => setPaymentMethod("wechat")} /> 微信支付</label>
        <label><input type="radio" name="payment-method" checked={paymentMethod === "alipay"} onChange={() => setPaymentMethod("alipay")} /> 支付宝</label>
        <div>
          <a className="button-primary" href={`/me/upgrade/pay?plan=${targetPlan}&method=${paymentMethod}`}>
            {text.next}
          </a>
        </div>
      </section>
    </section>
  );
}
