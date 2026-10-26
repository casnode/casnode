"use client";

import Link from "next/link";
import { Icon } from "@/components/icon";
import type { MessageKey, Translate } from "@/lib/i18n";
import { NotificationType, type Notification, type UserMap } from "@/lib/types";
import { badgeUrl, chatChannelUrl, groupUrl, topicUrl } from "@/lib/utils";

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
  [NotificationType.GrantedBadge]: "certificate",
  [NotificationType.MovedPost]: "arrow-right",
  [NotificationType.GroupMentioned]: "users",
  [NotificationType.MembershipAccepted]: "user-plus",
  [NotificationType.MembershipRequest]: "user-plus",
  [NotificationType.ChatMention]: "comment",
  [NotificationType.ChatMessage]: "comment",
};

interface NotificationData {
  badgeId?: number;
  badgeName?: string;
  badgeCode?: string;
  groupName?: string;
  chatChannelId?: number;
  chatMessageId?: number;
  chatChannelName?: string;
  chatChannelSlug?: string;
}

function dataOf(n: Notification): NotificationData {
  try {
    return n.data ? (JSON.parse(n.data) as NotificationData) : {};
  } catch {
    return {};
  }
}

export function notificationUrl(n: Notification): string {
  const data = dataOf(n);
  switch (n.notificationType) {
    case NotificationType.GrantedBadge:
      return data.badgeId ? badgeUrl({ id: data.badgeId, name: data.badgeName ?? "" }) : "/badges";
    case NotificationType.MembershipAccepted:
      return groupUrl(data.groupName ?? "");
    case NotificationType.MembershipRequest:
      return groupUrl(data.groupName ?? "", "requests");
    case NotificationType.ChatMention:
    case NotificationType.ChatMessage:
      return chatChannelUrl({ id: data.chatChannelId ?? 0, slug: data.chatChannelSlug ?? "" }, data.chatMessageId);
    default:
      return topicUrl({ id: n.topicId, slug: n.slug }, n.postNumber);
  }
}

// notificationTitle is what the notification is about: the topic, or the badge, group or chat channel.
export function notificationTitle(n: Notification, t: Translate): string {
  const data = dataOf(n);
  switch (n.notificationType) {
    case NotificationType.GrantedBadge: {
      const key = `badge.${data.badgeCode}.name` as MessageKey;
      const name = data.badgeCode && t(key) !== key ? t(key) : (data.badgeName ?? "");
      return t("menu.earnedBadge", { name });
    }
    case NotificationType.MembershipAccepted:
    case NotificationType.MembershipRequest:
      return data.groupName ?? "";
    case NotificationType.ChatMention:
      return data.chatChannelName ? `#${data.chatChannelName}` : t("chat.directMessage");
    case NotificationType.ChatMessage:
      return t("chat.directMessage");
    default:
      return n.topicTitle;
  }
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
    case NotificationType.GrantedBadge:
      return t("menu.grantedBadge");
    case NotificationType.MovedPost:
      return t("menu.movedPost");
    case NotificationType.GroupMentioned:
      return t("menu.groupMentioned", { group: dataOf(n).groupName ?? "" });
    case NotificationType.MembershipAccepted:
      return t("menu.membershipAccepted");
    case NotificationType.MembershipRequest:
      return t("menu.membershipRequest");
    case NotificationType.ChatMention:
      return t("menu.chatMention");
    case NotificationType.ChatMessage:
      return t("menu.chatMessage");
    default:
      return t("menu.posted");
  }
}

export function NotificationItem({ notification, users, t, onOpen }: { notification: Notification; users: UserMap; t: Translate; onOpen: (n: Notification) => void }) {
  const actor = users[notification.actingUserId];
  const label = notification.notificationType === NotificationType.GrantedBadge ? "" : (actor?.username ?? "");
  return (
    <li className={notification.read ? "" : "unread"}>
      <Link href={notificationUrl(notification)} onClick={() => onOpen(notification)} title={`${label} ${notificationVerb(notification, t)}`}>
        <Icon name={typeIcons[notification.notificationType] ?? "bell"} />
        <span className="item-text min-w-0">
          {label ? <span className="item-label">{label}</span> : null} <span className="item-description">{notificationTitle(notification, t)}</span>
        </span>
      </Link>
    </li>
  );
}
