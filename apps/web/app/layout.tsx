import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "PlayPage",
  description: "\u628a AI \u5199\u7684\u7f51\u7ad9\uff0c\u771f\u6b63\u53d1\u5230\u7f51\u4e0a\u3002"
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="zh-CN">
      <body>
        <a className="skip-link" href="#main-content">
          {"\u8df3\u5230\u4e3b\u8981\u5185\u5bb9"}
        </a>
        {children}
      </body>
    </html>
  );
}
