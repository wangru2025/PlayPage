"use client";

import { useEffect, useState } from "react";
import { postJSON, deleteJSON, getJSON } from "@/lib/api";

export function FavoriteProjectButton({ projectId, initialFavorited, initialCount }: { projectId: string; initialFavorited?: boolean; initialCount: number }) {
  const [favorited, setFavorited] = useState(initialFavorited ?? false);
  const [count, setCount] = useState(initialCount);
  const [working, setWorking] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => {
    let canceled = false;
    async function loadFavoriteState() {
      try {
        const data = await getJSON<{ favorited: boolean; favoritesCount: number }>(`/api/v1/projects/${projectId}/favorite`);
        if (canceled) return;
        setFavorited(data.favorited);
        setCount(data.favoritesCount);
      } catch {
        // 未登录时保持公开计数，只在用户点击时提示登录。
      }
    }
    void loadFavoriteState();
    return () => {
      canceled = true;
    };
  }, [projectId]);

  async function toggle() {
    if (working) return;
    try {
      setWorking(true);
      setMessage("");
      if (favorited) {
        await deleteJSON<{ status: string }>(`/api/v1/projects/${projectId}/favorite`);
        setFavorited(false);
        setCount((current) => Math.max(0, current - 1));
      } else {
        await postJSON<{ status: string }>(`/api/v1/projects/${projectId}/favorite`, {});
        setFavorited(true);
        setCount((current) => current + 1);
      }
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "操作失败，请先确认已经登录。");
    } finally {
      setWorking(false);
    }
  }

  return (
    <span style={{ display: "inline-flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
      <button className="button-secondary" type="button" disabled={working} aria-busy={working} onClick={toggle}>
        {favorited ? "取消收藏" : "收藏作品"}（{count}）
      </button>
      {message ? <span className="field-note" role="status">{message}</span> : null}
    </span>
  );
}
