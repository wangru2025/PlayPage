import type { Metadata } from "next";
import { RouteBackProvider } from "./RouteBackProvider";
import "./globals.css";

export const metadata: Metadata = {
  title: "PlayPage",
  description: "把 AI 写的网站，真正发到网上。"
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="zh-CN">
      <body>
        <a className="skip-link" href="#main-content">
          {"跳到主要内容"}
        </a>
        <RouteBackProvider>{children}</RouteBackProvider>
      </body>
    </html>
  );
}
