"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { Avatar } from "@/components/avatar";
import { BadgeChip } from "@/components/badge-chip";
import { useComposer } from "@/components/composer-provider";
import { Cooked } from "@/components/cooked";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import { longAgo, longDate } from "@/lib/format";
import type { ChatChannel, UserCard as UserCardData } from "@/lib/types";
import { chatChannelUrl, groupUrl, isValidDate, userUrl } from "@/lib/utils";

const cardWidth = 580;

// Clicking a user's avatar or name (or an @mention) shows their card instead of leaving the page, like Discourse.
// Links opt in with data-user-card="name", the class trigger-user-card, or the class mention of cooked posts.
export function UserCardProvider({ children }: { children: ReactNode }) {
  const [card, setCard] = useState<{ username: string; top: number; left: number } | null>(null);
  const pathname = usePathname();
  const [prevPath, setPrevPath] = useState(pathname);
  if (prevPath !== pathname) {
    setPrevPath(pathname);
    setCard(null);
  }

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) {
        return;
      }
      const target = e.target as Element | null;
      const link = target?.closest<HTMLAnchorElement>("a[data-user-card], a.trigger-user-card, .cooked a.mention, .chat-message a.mention");
      if (!link || link.closest(".user-card")) {
        return;
      }
      let username = link.dataset.userCard ?? "";
      if (!username) {
        const match = /^\/u\/([^/?#]+)/.exec(link.getAttribute("href") ?? "");
        username = match ? decodeURIComponent(match[1]) : "";
      }
      if (!username) {
        return;
      }
      e.preventDefault();
      const rect = link.getBoundingClientRect();
      const left = Math.max(8, Math.min(rect.left + window.scrollX, window.scrollX + document.documentElement.clientWidth - Math.min(cardWidth, window.innerWidth - 16) - 8));
      setCard({ username, top: rect.bottom + window.scrollY + 4, left });
    };
    document.addEventListener("click", onClick);
    return () => document.removeEventListener("click", onClick);
  }, []);

  return (
    <>
      {children}
      {card ? <UserCard key={card.username} username={card.username} top={card.top} left={card.left} onClose={() => setCard(null)} /> : null}
    </>
  );
}

function UserCard({ username, top, left, onClose }: { username: string; top: number; left: number; onClose: () => void }) {
  const { t, locale, currentUser } = useSite();
  const { open: openComposer } = useComposer();
  const router = useRouter();
  const [data, setData] = useState<UserCardData | null>(null);
  const [error, setError] = useState("");
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    apiGet<UserCardData>("get-user-card", { username })
      .then((res) => !cancelled && setData(res.data))
      .catch((err) => !cancelled && setError(errorMessage(err)));
    return () => {
      cancelled = true;
    };
  }, [username]);

  useEffect(() => {
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose();
      }
    };
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    // the click that opened the card is still going on
    const timer = setTimeout(() => document.addEventListener("mousedown", onDown));
    document.addEventListener("keydown", onKey);
    return () => {
      clearTimeout(timer);
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [onClose]);

  const chat = useCallback(async () => {
    try {
      const res = await apiPost<ChatChannel>("create-direct-channel", undefined, { usernames: [username] });
      onClose();
      router.push(chatChannelUrl(res.data));
    } catch (err) {
      toast.error(errorMessage(err));
    }
  }, [username, onClose, router]);

  const user = data?.user;
  const staff = user && (user.admin || user.moderator);
  return (
    <div ref={ref} className="user-card" role="dialog" aria-label={username} style={{ top, left, width: `min(${cardWidth}px, calc(100vw - 16px))` }}>
      {!data && !error ? <div className="spinner" /> : null}
      {error ? <div className="p-4">{error}</div> : null}
      {user && data ? (
        <>
          <div className="card-row first-row">
            <Link href={userUrl(user.username)} className="user-card-avatar" onClick={onClose}>
              <Avatar user={user} size={120} />
            </Link>
            <div className="names">
              <h1 className={staff ? "staff" : ""}>
                <Link href={userUrl(user.username)} onClick={onClose}>
                  {user.username}
                </Link>
                {staff ? <Icon name="shield-halved" title={user.admin ? t("user.admin") : t("user.moderator")} /> : null}
              </h1>
              {user.name && user.name !== user.username ? <h2 className="full-name">{user.name}</h2> : null}
              {user.title ? <h3 className="user-title">{user.title}</h3> : null}
            </div>
            <ul className="usercard-controls">
              {data.canMessage ? (
                <li>
                  <button
                    type="button"
                    className="btn btn-primary"
                    onClick={() => {
                      onClose();
                      openComposer({ action: "privateMessage", recipients: [user.username] });
                    }}
                  >
                    <Icon name="envelope" />
                    {t("user.message")}
                  </button>
                </li>
              ) : null}
              {data.canChat ? (
                <li>
                  <button type="button" className="btn btn-default" onClick={chat}>
                    <Icon name="comment" />
                    {t("chat.title")}
                  </button>
                </li>
              ) : null}
              {currentUser?.id === user.id ? (
                <li>
                  <Link href={userUrl(user.username, "activity")} className="btn btn-default" onClick={onClose}>
                    <Icon name="list" />
                    {t("user.activity")}
                  </Link>
                </li>
              ) : null}
            </ul>
          </div>
          {user.location || user.website ? (
            <div className="card-row location-and-website">
              {user.location ? (
                <span>
                  <Icon name="location-dot" />
                  {user.location}
                </span>
              ) : null}
              {user.website ? (
                <a href={/^https?:\/\//.test(user.website) ? user.website : `https://${user.website}`} rel="nofollow ugc noopener" target="_blank">
                  <Icon name="globe" />
                  {user.website.replace(/^https?:\/\//, "")}
                </a>
              ) : null}
            </div>
          ) : null}
          {user.bioCooked ? (
            <div className="card-row bio">
              <Cooked html={user.bioCooked} />
            </div>
          ) : null}
          <div className="card-row metadata">
            <span>
              <span className="desc">{t("user.joined")}</span> <span title={longDate(user.createdAt, locale)}>{longAgo(user.createdAt, locale)}</span>
            </span>
            {isValidDate(user.lastPostedAt) ? (
              <span>
                <span className="desc">{t("user.lastPost")}</span> <span title={longDate(user.lastPostedAt, locale)}>{longAgo(user.lastPostedAt, locale)}</span>
              </span>
            ) : null}
            {isValidDate(user.lastSeenAt) ? (
              <span>
                <span className="desc">{t("user.seen")}</span> <span title={longDate(user.lastSeenAt, locale)}>{longAgo(user.lastSeenAt, locale)}</span>
              </span>
            ) : null}
          </div>
          {data.groups.length > 0 ? (
            <div className="card-row groups">
              <span className="desc">{t("group.groups")}</span>
              {data.groups.map((group) => (
                <Link key={group.id} href={groupUrl(group.name)} onClick={onClose}>
                  {group.name}
                </Link>
              ))}
            </div>
          ) : null}
          {data.badges.length > 0 ? (
            <div className="card-row badges" onClick={onClose}>
              {data.badges.map((item) => (
                <BadgeChip key={item.badge.id} badge={item.badge} count={item.count} />
              ))}
              {data.badgeCount > data.badges.length ? (
                <Link href={userUrl(user.username, "badges")} className="more-user-badges">
                  {t("badge.more", { count: data.badgeCount - data.badges.length })}
                </Link>
              ) : null}
            </div>
          ) : null}
        </>
      ) : null}
    </div>
  );
}
