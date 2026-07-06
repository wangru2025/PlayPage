"use client";

import { useEffect, useMemo, useState } from "react";
import { usePathname } from "next/navigation";

type BackTarget = {
  fromHref: string;
  fromLabel: string;
  toHref: string;
};

const STORAGE_KEY = "playpage:lastRouteBackTarget";

const routeLabels: Array<[RegExp, string]> = [
  [/^\/$/, "首页"],
  [/^\/@[^/]+$/, "作者主页"],
  [/^\/projects$/, "我的作品"],
  [/^\/projects\/new$/, "创建作品"],
  [/^\/projects\/fork$/, "改编作品"],
  [/^\/projects\/repair$/, "AI 网页急救站活动"],
  [/^\/projects\/[^/]+\/app-builds$/, "导出安装包"],
  [/^\/projects\/[^/]+\/discussions$/, "讨论区"],
  [/^\/projects\/[^/]+\/discussions\/new$/, "发起讨论"],
  [/^\/projects\/[^/]+\/discussions\/[^/]+$/, "讨论详情"],
  [/^\/projects\/[^/]+\/proposals$/, "改进提案"],
  [/^\/projects\/[^/]+\/proposals\/new$/, "新增提案"],
  [/^\/projects\/[^/]+\/proposals\/[^/]+$/, "提案详情"],
  [/^\/projects\/[^/]+\/releases$/, "历史版本"],
  [/^\/projects\/[^/]+\/repair$/, "提交修复申请"],
  [/^\/projects\/[^/]+\/repair\/requests$/, "修复申请"],
  [/^\/projects\/[^/]+\/repair\/[^/]+\/ai$/, "AI 圆桌"],
  [/^\/projects\/[^/]+\/repair\/[^/]+\/roundtable$/, "圆桌聊天记录"],
  [/^\/projects\/[^/]+\/domains$/, "申请独立网址"],
  [/^\/projects\/[^/]+\/interactive$/, "互动功能"],
  [/^\/projects\/[^/]+\/interactive\/export$/, "导出互动数据"],
  [/^\/projects\/[^/]+\/settings$/, "作品设置"],
  [/^\/projects\/[^/]+\/stats$/, "统计数据"],
  [/^\/contest$/, "创作比赛"],
  [/^\/contest\/submit$/, "提交参赛作品"],
  [/^\/templates$/, "模板市场"],
  [/^\/templates\/submit$/, "投稿模板"],
  [/^\/templates\/[^/]+$/, "模板详情"],
  [/^\/me$/, "个人中心"],
  [/^\/me\/profile$/, "个人资料"],
  [/^\/me\/favorites$/, "我的收藏"],
  [/^\/me\/following$/, "我的关注"],
  [/^\/me\/proposals$/, "我的提案"],
  [/^\/me\/upgrade$/, "升级套餐"],
  [/^\/me\/upgrade\/pay$/, "套餐支付"],
  [/^\/square$/, "作品广场"],
  [/^\/auth$/, "登录或注册"],
  [/^\/auth\/profile$/, "设置公开名字"],
  [/^\/admin$/, "管理后台"],
  [/^\/admin\/contest$/, "创作比赛管理"]
];

function normalizePath(path: string): string {
  if (!path) {
    return "/";
  }
  try {
    const url = new URL(path, window.location.origin);
    return url.pathname.replace(/\/+$/, "") || "/";
  } catch {
    return path.split(/[?#]/, 1)[0].replace(/\/+$/, "") || "/";
  }
}

function labelForPath(path: string): string {
  const normalized = normalizePath(path);
  for (const [pattern, label] of routeLabels) {
    if (pattern.test(normalized)) {
      return label;
    }
  }
  return "上一页";
}

function isAuthFlowPath(path: string): boolean {
  const normalized = normalizePath(path);
  return normalized === "/auth" || normalized === "/auth/session" || normalized === "/auth/profile";
}

function isErrorFallbackPage(): boolean {
  return document.querySelector("[data-route-back-source='error']") !== null;
}

function isUsefulBackTarget(target: BackTarget | null, currentPath: string): target is BackTarget {
  if (!target) {
    return false;
  }
  const fromPath = normalizePath(target.fromHref);
  const toPath = normalizePath(target.toHref);
  const nowPath = normalizePath(currentPath);
  if (fromPath === nowPath || toPath !== nowPath) {
    return false;
  }
  if (isAuthFlowPath(fromPath) || isAuthFlowPath(nowPath)) {
    return false;
  }
  return true;
}

function readTarget(): BackTarget | null {
  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as BackTarget;
    if (!parsed.fromHref || !parsed.toHref || !parsed.fromLabel) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

export function RouteBackProvider({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [target, setTarget] = useState<BackTarget | null>(null);

  useEffect(() => {
    setTarget(readTarget());
  }, [pathname]);

  useEffect(() => {
    function onClick(event: MouseEvent) {
      if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
        return;
      }
      const link = (event.target as Element | null)?.closest("a[href]");
      if (!(link instanceof HTMLAnchorElement)) {
        return;
      }
      const href = link.getAttribute("href") || "";
      if (isErrorFallbackPage()) {
        window.sessionStorage.removeItem(STORAGE_KEY);
        setTarget(null);
        return;
      }
      if (link.dataset.routeBackIgnore === "true") {
        return;
      }
      if (href === "" || href.startsWith("#") || href.startsWith("mailto:") || href.startsWith("tel:") || link.target) {
        return;
      }
      const url = new URL(link.href, window.location.origin);
      if (url.origin !== window.location.origin) {
        return;
      }
      const fromHref = normalizePath(window.location.pathname);
      const toHref = normalizePath(url.pathname);
      if (fromHref === toHref) {
        return;
      }
      if (isAuthFlowPath(fromHref) || isAuthFlowPath(toHref)) {
        window.sessionStorage.removeItem(STORAGE_KEY);
        setTarget(null);
        return;
      }
      const nextTarget: BackTarget = {
        fromHref,
        fromLabel: link.dataset.backLabel || labelForPath(fromHref),
        toHref
      };
      window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(nextTarget));
      setTarget(nextTarget);
    }

    document.addEventListener("click", onClick, true);
    return () => document.removeEventListener("click", onClick, true);
  }, []);

  const visibleTarget = useMemo(() => (isUsefulBackTarget(target, pathname) ? target : null), [pathname, target]);

  return (
    <>
      {visibleTarget ? (
        <div className="route-back shell" aria-label="页面返回导航">
          <a
            className="button-secondary route-back-link"
            href={visibleTarget.fromHref}
            data-route-back-ignore="true"
            onClick={() => {
              window.sessionStorage.removeItem(STORAGE_KEY);
              setTarget(null);
            }}
          >
            返回{visibleTarget.fromLabel}
          </a>
        </div>
      ) : null}
      {children}
    </>
  );
}
