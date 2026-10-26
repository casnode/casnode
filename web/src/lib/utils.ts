import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { Category, Topic, User } from "@/lib/types";

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
