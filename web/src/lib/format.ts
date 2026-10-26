import { isValidDate } from "@/lib/utils";

// Dates and numbers the way Discourse shows them: "3h", "Oct 26", "Mar 2024", "1.2k".

const monthsEn = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

export function shortDate(date: Date, locale: string, withYear: boolean): string {
  if (locale.startsWith("zh")) {
    return withYear ? `${date.getFullYear()}年${date.getMonth() + 1}月` : `${date.getMonth() + 1}月${date.getDate()}日`;
  }
  return withYear ? `${monthsEn[date.getMonth()]} ${date.getFullYear()}` : `${monthsEn[date.getMonth()]} ${date.getDate()}`;
}

// relativeAge is the tiny age of topic lists and posts: minutes, hours, then days for two weeks, then the date
// ("Oct 26"), with the year when it is another year ("Mar 2024").
export function relativeAge(value: string | undefined, locale: string, now = Date.now()): string {
  if (!isValidDate(value)) {
    return "";
  }
  const date = new Date(value!);
  const minutes = Math.max(1, Math.round(Math.abs(now - date.getTime()) / 60000));
  const zh = locale.startsWith("zh");
  if (minutes <= 44) {
    return zh ? `${minutes} 分钟` : `${minutes}m`;
  }
  if (minutes <= 1409) {
    const hours = Math.max(1, Math.round(minutes / 60));
    return zh ? `${hours} 小时` : `${hours}h`;
  }
  if (minutes <= 14 * 1440) {
    const days = Math.max(1, Math.round(minutes / 1440));
    return zh ? `${days} 天` : `${days}d`;
  }
  return shortDate(date, locale, new Date(now).getFullYear() !== date.getFullYear());
}

// longAge is the "3 hours ago" of tooltips and profiles.
export function longDate(value: string | undefined, locale: string): string {
  if (!isValidDate(value)) {
    return "";
  }
  return new Date(value!).toLocaleString(locale.startsWith("zh") ? "zh-CN" : "en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

export function longAgo(value: string | undefined, locale: string): string {
  if (!isValidDate(value)) {
    return "";
  }
  const seconds = Math.max(0, (Date.now() - new Date(value!).getTime()) / 1000);
  const zh = locale.startsWith("zh");
  const units: [number, string, string][] = [
    [60, "minute", "分钟"],
    [3600, "hour", "小时"],
    [86400, "day", "天"],
    [2592000, "month", "个月"],
    [31536000, "year", "年"],
  ];
  if (seconds < 60) {
    return zh ? "刚刚" : "just now";
  }
  let res = "";
  for (let i = 0; i < units.length; i++) {
    const [size, en, cn] = units[i];
    const next = units[i + 1]?.[0] ?? Infinity;
    if (seconds < next) {
      const n = Math.floor(seconds / size);
      res = zh ? `${n} ${cn}前` : `${n} ${en}${n === 1 ? "" : "s"} ago`;
      break;
    }
  }
  return res;
}

export function formatNumber(n: number | undefined): string {
  const value = n ?? 0;
  if (value >= 1000000) {
    return `${(value / 1000000).toFixed(1).replace(/\.0$/, "")}m`;
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1).replace(/\.0$/, "")}k`;
  }
  return String(value);
}

// heatClass colors the views and replies numbers of busy topics, like Discourse's "heatmap".
export function viewsHeat(views: number): string {
  if (views >= 5000) {
    return "heatmap-high";
  }
  if (views >= 2000) {
    return "heatmap-med";
  }
  if (views >= 1000) {
    return "heatmap-low";
  }
  return "";
}

export function likesHeat(likes: number, posts: number): string {
  if (posts < 10) {
    return "";
  }
  const ratio = likes / posts;
  if (ratio > 2) {
    return "heatmap-high";
  }
  if (ratio > 1) {
    return "heatmap-med";
  }
  if (ratio > 0.5) {
    return "heatmap-low";
  }
  return "";
}

export function activityHeat(createdAt: string, bumpedAt: string): string {
  if (!isValidDate(createdAt) || !isValidDate(bumpedAt)) {
    return "";
  }
  const days = (new Date(bumpedAt).getTime() - new Date(createdAt).getTime()) / 86400000;
  if (days > 120) {
    return "coldmap-high";
  }
  if (days > 60) {
    return "coldmap-med";
  }
  if (days > 14) {
    return "coldmap-low";
  }
  return "";
}
