"use client";

import { useEffect, useState } from "react";
import { deleteJSON, getJSON, postJSON } from "@/lib/api";

type AuthorProfile = {
  username: string;
  followersCount: number;
  followingByMe?: boolean;
};

type User = { username: string; email: string };

export function FollowAuthorButton({ username, initialFollowersCount }: { username: string; initialFollowersCount: number }) {
  const [loggedIn, setLoggedIn] = useState(false);
  const [isSelf, setIsSelf] = useState(false);
  const [following, setFollowing] = useState(false);
  const [followersCount, setFollowersCount] = useState(initialFollowersCount);
  const [working, setWorking] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => {
    let canceled = false;
    async function load() {
      try {
        const me = await getJSON<User>("/api/v1/me");
        if (canceled) return;
        setLoggedIn(true);
        setIsSelf(me.username === username);
        const profile = await getJSON<AuthorProfile>(`/api/v1/authors/${encodeURIComponent(username)}`);
        if (canceled) return;
        setFollowing(profile.followingByMe ?? false);
        setFollowersCount(profile.followersCount);
      } catch {
        if (!canceled) setLoggedIn(false);
      }
    }
    void load();
    return () => { canceled = true; };
  }, [username]);

  async function reloadProfile() {
    const profile = await getJSON<AuthorProfile>(`/api/v1/authors/${encodeURIComponent(username)}`);
    setFollowing(profile.followingByMe ?? false);
    setFollowersCount(profile.followersCount);
  }

  if (!loggedIn || isSelf) {
    return <span className="soft-badge">关注者 {followersCount}</span>;
  }

  async function toggle() {
    if (working) return;
    try {
      setWorking(true);
      setMessage("");
      if (following) {
        await deleteJSON<{ status: string }>(`/api/v1/authors/${encodeURIComponent(username)}/follow`);
      } else {
        await postJSON<{ status: string }>(`/api/v1/authors/${encodeURIComponent(username)}/follow`, {});
      }
      await reloadProfile();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "操作失败，请稍后再试。");
    } finally {
      setWorking(false);
    }
  }

  return (
    <span style={{ display: "inline-flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
      <button className="button-secondary" type="button" disabled={working} aria-busy={working} onClick={toggle}>
        {following ? "取消关注" : "关注作者"}（{followersCount}）
      </button>
      {message ? <span className="field-note" role="status">{message}</span> : null}
    </span>
  );
}
