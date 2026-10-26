"use client";

import Link from "next/link";
import { Icon } from "@/components/icon";
import type { Translate } from "@/lib/i18n";
import { NotificationType, type Notification, type UserMap } from "@/lib/types";
import { topicUrl } from "@/lib/utils";

const typeIcons: Record<number, string> = {
  [NotificationType.Mentioned]: "at",
  [NotificationType.Replied]: "reply",
  [NotificationType.Quoted]: "quote-left",
  [NotificationType.Edited]: "pencil",
  [NotificationType.Liked]: "heart",
  [NotificationType.PrivateMessage]: "envelope",
  [NotificationType.InvitedToPrivateMessage]: "envelope",
  [NotificationType.Posted]: "comment",
  [NotificationType.AdminMessage]: "envelope",
};

export function notificationUrl(n: Notification): string {
  return topicUrl({ id: n.topicId, slug: n.slug }, n.postNumber);
}

export function notificationVerb(n: Notification, t: Translate): string {
  switch (n.notificationType) {
    case NotificationType.Mentioned:
      return t("menu.mentioned");
    case NotificationType.Replied:
      return t("menu.replied");
    case NotificationType.Quoted:
      return t("menu.quoted");
    case NotificationType.Liked:
      return t("menu.liked");
    case NotificationType.PrivateMessage:
      return t("menu.privateMessage");
    case NotificationType.InvitedToPrivateMessage:
      return t("menu.invited");
    default:
      return t("menu.posted");
  }
}

export function NotificationItem({ notification, users, t, onOpen }: { notification: Notification; users: UserMap; t: Translate; onOpen: (n: Notification) => void }) {
  const actor = users[notification.actingUserId];
  return (
    <li className={notification.read ? "" : "unread"}>
      <Link href={notificationUrl(notification)} onClick={() => onOpen(notification)} title={`${actor?.username ?? ""} ${notificationVerb(notification, t)}`}>
        <Icon name={typeIcons[notification.notificationType] ?? "bell"} />
        <span className="item-text min-w-0">
          <span className="item-label">{actor?.username ?? ""}</span> <span className="item-description">{notification.topicTitle}</span>
        </span>
      </Link>
    </li>
  );
}
