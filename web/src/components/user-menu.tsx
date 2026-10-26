"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { Icon } from "@/components/icon";
import { NotificationItem } from "@/components/notification-item";
import { useSite } from "@/components/site-provider";
import { apiGet, apiPost } from "@/lib/api";
import type { Notification, UserMap, UserPost } from "@/lib/types";
import { topicUrl, userUrl } from "@/lib/utils";

type Tab = "all" | "replies" | "likes" | "messages" | "bookmarks" | "profile";

const tabs: { id: Tab; icon: string; labelKey: "menu.all" | "menu.replies" | "menu.likes" | "menu.messages" | "menu.bookmarks" | "menu.profile" }[] = [
  { id: "all", icon: "bell", labelKey: "menu.all" },
  { id: "replies", icon: "reply", labelKey: "menu.replies" },
  { id: "likes", icon: "heart", labelKey: "menu.likes" },
  { id: "messages", icon: "envelope", labelKey: "menu.messages" },
  { id: "bookmarks", icon: "bookmark", labelKey: "menu.bookmarks" },
  { id: "profile", icon: "user", labelKey: "menu.profile" },
];

// The panel under the avatar: notifications by kind on the left, the tabs on the right, like Discourse's user menu.
export function UserMenu({ onClose }: { onClose: () => void }) {
  const { currentUser, t, site, setCounts, refreshSite } = useSite();
  const router = useRouter();
  const [tab, setTab] = useState<Tab>("all");
  const [loaded, setLoaded] = useState<{ tab: Tab; items: Notification[] } | null>(null);
  const [users, setUsers] = useState<UserMap>({});
  const [bookmarks, setBookmarks] = useState<UserPost[] | null>(null);

  useEffect(() => {
    let cancelled = false;
    if (tab === "profile") {
      return;
    }
    if (tab === "bookmarks") {
      apiGet<{ posts: UserPost[] }>("get-user-activity", { username: currentUser!.username, filter: "bookmarks" })
        .then((res) => !cancelled && setBookmarks(res.data.posts))
        .catch(() => !cancelled && setBookmarks([]));
      return;
    }
    apiGet<{ notifications: Notification[]; users: UserMap }>("get-notifications", { filter: tab, limit: 30 })
      .then((res) => {
        if (!cancelled) {
          setLoaded({ tab, items: res.data.notifications });
          setUsers(res.data.users);
        }
      })
      .catch(() => !cancelled && setLoaded({ tab, items: [] }));
    return () => {
      cancelled = true;
    };
  }, [tab, currentUser]);

  const open = async (n: Notification) => {
    onClose();
    if (!n.read) {
      await apiPost("read-notification", { id: n.id }).catch(() => undefined);
      refreshSite().catch(() => undefined);
    }
  };

  const dismissAll = async () => {
    await apiPost("read-notification").catch(() => undefined);
    setLoaded((l) => (l ? { ...l, items: l.items.map((n) => ({ ...n, read: true })) } : l));
    setCounts({ notifications: 0, highPriorityNotifications: 0 });
  };

  const logout = async () => {
    await apiPost("signout").catch(() => undefined);
    onClose();
    router.refresh();
  };

  if (!currentUser) {
    return null;
  }
  const unreadMessages = site.counts?.unreadMessages ?? 0;
  const notifications = loaded?.tab === tab ? loaded.items : null;

  return (
    <div className="user-menu menu-panel">
      <div className="menu-tabs-container" role="tablist">
        {tabs.map((item) => (
          <button
            key={item.id}
            type="button"
            role="tab"
            className={`menu-tab${tab === item.id ? " active" : ""}`}
            title={t(item.labelKey)}
            aria-selected={tab === item.id}
            onClick={() => setTab(item.id)}
          >
            <Icon name={item.icon} />
            {item.id === "all" && (site.counts?.notifications ?? 0) > 0 ? <span className="badge-notification">{site.counts.notifications}</span> : null}
            {item.id === "messages" && unreadMessages > 0 ? <span className="badge-notification">{unreadMessages}</span> : null}
          </button>
        ))}
      </div>
      <div className="quick-access-panel">
        {tab === "profile" ? (
          <div className="panel-body">
            <ul>
              <li>
                <Link href={userUrl(currentUser.username, "summary")} onClick={onClose}>
                  <Icon name="user" />
                  {t("user.summary")}
                </Link>
              </li>
              <li>
                <Link href={userUrl(currentUser.username, "activity")} onClick={onClose}>
                  <Icon name="list" />
                  {t("user.activity")}
                </Link>
              </li>
              <li>
                <Link href={userUrl(currentUser.username, "messages")} onClick={onClose}>
                  <Icon name="envelope" />
                  {t("user.messages")}
                </Link>
              </li>
              <li>
                <Link href={userUrl(currentUser.username, "activity/bookmarks")} onClick={onClose}>
                  <Icon name="bookmark" />
                  {t("user.bookmarks")}
                </Link>
              </li>
              <li>
                <Link href={userUrl(currentUser.username, "preferences")} onClick={onClose}>
                  <Icon name="gear" />
                  {t("user.preferences")}
                </Link>
              </li>
              <li>
                <button type="button" onClick={logout}>
                  <Icon name="right-from-bracket" />
                  {t("user.logOut")}
                </button>
              </li>
            </ul>
          </div>
        ) : tab === "bookmarks" ? (
          <>
            <div className="panel-body">
              {bookmarks === null ? <div className="spinner" /> : null}
              {bookmarks?.length === 0 ? <div className="empty-state">{t("user.noBookmarks")}</div> : null}
              <ul>
                {bookmarks?.map((item) =>
                  item.topic ? (
                    <li key={item.post.id}>
                      <Link href={topicUrl(item.topic, item.post.postNumber)} onClick={onClose}>
                        <Icon name="bookmark" />
                        <span className="item-description">{item.topic.title}</span>
                      </Link>
                    </li>
                  ) : null,
                )}
              </ul>
            </div>
            <div className="panel-body-bottom">
              <Link href={userUrl(currentUser.username, "activity/bookmarks")} className="btn btn-default btn-small" onClick={onClose}>
                {t("menu.viewAll")}
              </Link>
            </div>
          </>
        ) : (
          <>
            <div className="panel-body">
              {notifications === null ? <div className="spinner" /> : null}
              {notifications?.length === 0 ? <div className="empty-state">{t("menu.noNotifications")}</div> : null}
              <ul>
                {notifications?.map((n) => (
                  <NotificationItem key={n.id} notification={n} users={users} t={t} onOpen={open} />
                ))}
              </ul>
            </div>
            <div className="panel-body-bottom">
              <Link
                href={userUrl(currentUser.username, tab === "messages" ? "messages" : "notifications")}
                className="btn btn-small"
                onClick={onClose}
              >
                {t("menu.viewAll")}
              </Link>
              <button type="button" className="btn btn-small" onClick={dismissAll} title={t("menu.dismiss")}>
                <Icon name="check" />
                {t("menu.dismiss")}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
