import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { Badge, Category, ChatChannel, Topic, User } from "@/lib/types";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function topicUrl(topic: Pick<Topic, "id" | "slug">, postNumber?: number): string {
  const base = `/t/${encodeURIComponent(topic.slug || "topic")}/${topic.id}`;
  return postNumber && postNumber > 1 ? `${base}/${postNumber}` : base;
}

// topicLinkUrl goes to the first unread post of a topic the user has started reading.
export function topicLinkUrl(topic: Topic): string {
  if (topic.lastReadPostNumber > 0 && topic.lastReadPostNumber < topic.highestPostNumber) {
    return topicUrl(topic, topic.lastReadPostNumber + 1);
  }
  return topicUrl(topic);
}

export function categoryUrl(category: Pick<Category, "id" | "slug" | "parentId">, categories?: Category[]): string {
  const parent = category.parentId ? categories?.find((c) => c.id === category.parentId) : undefined;
  const prefix = parent ? `${encodeURIComponent(parent.slug)}/` : "";
  return `/c/${prefix}${encodeURIComponent(category.slug)}/${category.id}`;
}

export function userUrl(username: string, tab?: string): string {
  return `/u/${encodeURIComponent(username)}${tab ? `/${tab}` : ""}`;
}

export function tagUrl(name: string): string {
  return `/tag/${encodeURIComponent(name)}`;
}

const namedEntities: Record<string, string> = { amp: "&", lt: "<", gt: ">", quot: '"', apos: "'", nbsp: " " };

// htmlToText is the plain text of a post's rendered HTML, for meta tags and structured data.
export function htmlToText(html: string): string {
  return html
    .replace(/<[^>]+>/g, " ")
    .replace(/&(#x[0-9a-f]+|#\d+|[a-z]+);/gi, (entity, code: string) => {
      if (code[0] === "#") {
        const n = code[1] === "x" || code[1] === "X" ? parseInt(code.slice(2), 16) : parseInt(code.slice(1), 10);
        return n > 0 && n <= 0x10ffff ? String.fromCodePoint(n) : entity;
      }
      return namedEntities[code.toLowerCase()] ?? entity;
    })
    .replace(/\s+/g, " ")
    .trim();
}

// summarize shortens text for a meta description, cutting at a word boundary.
export function summarize(text: string, max = 160): string {
  text = text.replace(/…$/, "").trim();
  if ([...text].length <= max) {
    return text;
  }
  let cut = [...text].slice(0, max).join("");
  const space = cut.lastIndexOf(" ");
  if (space > max * 0.6) {
    cut = cut.slice(0, space);
  }
  return cut.replace(/[\s,.;:!?，。；：！？、-]+$/, "") + "…";
}

export function isValidDate(value?: string | null): boolean {
  if (!value) {
    return false;
  }
  const date = new Date(value);
  return !Number.isNaN(date.getTime()) && date.getFullYear() > 1970;
}

export function isPinned(topic: Topic): boolean {
  return isValidDate(topic.pinnedAt);
}

export function isStaff(user?: User | null): boolean {
  return !!user && (user.admin || user.moderator);
}

export function displayName(user?: User | null): string {
  return user ? user.username : "";
}

export function groupUrl(name: string, tab?: string): string {
  return `/g/${encodeURIComponent(name)}${tab ? `/${tab}` : ""}`;
}

export function badgeUrl(badge: Pick<Badge, "id" | "name">): string {
  const slug = badge.name.toLowerCase().replace(/[^\p{L}\p{N}]+/gu, "-").replace(/^-+|-+$/g, "") || "badge";
  return `/badges/${badge.id}/${encodeURIComponent(slug)}`;
}

export function chatChannelUrl(channel: Pick<ChatChannel, "id" | "slug">, messageId?: number): string {
  const base = `/chat/c/${encodeURIComponent(channel.slug || "dm")}/${channel.id}`;
  return messageId ? `${base}/${messageId}` : base;
}

// A direct channel is named after the other people in it.
export function chatChannelTitle(channel: ChatChannel, currentUser?: User | null): string {
  if (channel.channelType !== "direct") {
    return channel.name;
  }
  const others = (channel.directUsers ?? []).filter((u) => u.id !== currentUser?.id);
  return (others.length > 0 ? others : channel.directUsers ?? []).map((u) => u.username).join(", ");
}
