"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Avatar } from "@/components/avatar";
import { Icon } from "@/components/icon";
import { notificationTitle, notificationUrl, notificationVerb } from "@/components/notification-item";
import { useSite } from "@/components/site-provider";
import { apiGet, apiPost } from "@/lib/api";
import { longDate, relativeAge } from "@/lib/format";
import type { Notification, UserMap } from "@/lib/types";

const filters = ["all", "unread", "replies", "likes", "messages", "chat", "badges"] as const;

export function NotificationsList() {
  const { t, locale, refreshSite } = useSite();
  const [filter, setFilter] = useState<(typeof filters)[number]>("all");
  const [items, setItems] = useState<Notification[] | null>(null);
  const [users, setUsers] = useState<UserMap>({});
  const [page, setPage] = useState(0);
  const [more, setMore] = useState(false);

  useEffect(() => {
    let cancelled = false;
    apiGet<{ notifications: Notification[]; users: UserMap; more: boolean }>("get-notifications", { filter, page: 0 }).then((res) => {
      if (!cancelled) {
        setItems(res.data.notifications);
        setUsers(res.data.users);
        setMore(res.data.more);
        setPage(0);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [filter]);

  const loadMore = async () => {
    const res = await apiGet<{ notifications: Notification[]; users: UserMap; more: boolean }>("get-notifications", { filter, page: page + 1 });
    setItems((list) => [...(list ?? []), ...res.data.notifications]);
    setUsers((u) => ({ ...u, ...res.data.users }));
    setMore(res.data.more);
    setPage(page + 1);
  };

  const markAll = async () => {
    await apiPost("read-notification");
    setItems((list) => list?.map((n) => ({ ...n, read: true })) ?? null);
    refreshSite().catch(() => undefined);
  };

  const labels: Record<string, string> = { all: t("menu.all"), unread: "Unread", replies: t("menu.replies"), likes: t("menu.likes"), messages: t("menu.messages"), chat: t("chat.title"), badges: t("badge.badges") };

  return (
    <div>
      <div className="user-secondary-navigation">
        {filters.map((f) => (
          <button key={f} type="button" className={`btn btn-small ${f === filter ? "btn-primary" : "btn-flat"}`} onClick={() => setFilter(f)}>
            {f === "unread" ? t("nav.unread") : labels[f]}
          </button>
        ))}
        <button type="button" className="btn btn-small btn-default ml-auto" onClick={markAll}>
          <Icon name="check" />
          {t("menu.dismiss")}
        </button>
      </div>
      {items === null ? <div className="spinner" /> : null}
      {items?.length === 0 ? <p className="text-[var(--primary-medium)]">{t("user.noNotifications")}</p> : null}
      <div className="user-stream">
        {items?.map((n) => {
          const actor = users[n.actingUserId];
          return (
            <div key={n.id} className={`user-stream-item${n.read ? "" : " bg-[var(--tertiary-very-low)]"}`}>
              <Avatar user={actor} size={45} />
              <div className="stream-content">
                <div className="item-header">
                  <Link
                    href={notificationUrl(n)}
                    className="title"
                    onClick={() => {
                      if (!n.read) {
                        apiPost("read-notification", { id: n.id }).catch(() => undefined);
                      }
                    }}
                  >
                    {notificationTitle(n, t)}
                  </Link>
                  <span className="time" title={longDate(n.createdAt, locale)} suppressHydrationWarning>
                    {relativeAge(n.createdAt, locale)}
                  </span>
                </div>
                <div className="mt-1 text-[var(--primary-high)]">
                  <strong>{actor?.username}</strong> {notificationVerb(n, t)}
                </div>
                {n.excerpt ? <div className="excerpt text-[var(--primary-medium)]">{n.excerpt}</div> : null}
              </div>
            </div>
          );
        })}
      </div>
      {more ? (
        <button type="button" className="btn btn-default mt-4" onClick={loadMore}>
          {t("common.loadMore")}
        </button>
      ) : null}
    </div>
  );
}
