"use client";

import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Poll } from "@/components/poll";
import type { Poll as PollData } from "@/lib/types";

// A post's HTML as the server rendered and sanitized it; code blocks get highlighted once in the browser, and the
// polls' option lists are replaced by the polls.
export function Cooked({
  html,
  className,
  polls,
  postId,
  onPollChange,
}: {
  html: string;
  className?: string;
  polls?: PollData[];
  postId?: number;
  onPollChange?: (poll: PollData) => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [mounts, setMounts] = useState<{ name: string; el: HTMLElement }[]>([]);
  const hasPolls = (polls?.length ?? 0) > 0;

  useEffect(() => {
    const el = ref.current;
    if (!el) {
      return;
    }
    const blocks = Array.from(el.querySelectorAll<HTMLElement>("pre code")).filter((block) => !block.dataset.highlighted);
    if (blocks.length === 0) {
      return;
    }
    let cancelled = false;
    import("highlight.js/lib/common").then(({ default: hljs }) => {
      if (cancelled) {
        return;
      }
      for (const block of blocks) {
        const lang = Array.from(block.classList).find((c) => c.startsWith("language-"))?.slice(9);
        if (lang && !hljs.getLanguage(lang)) {
          block.classList.remove(`language-${lang}`);
        }
        hljs.highlightElement(block);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [html]);

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el || !hasPolls) {
      setMounts([]);
      return;
    }
    const found: { name: string; el: HTMLElement }[] = [];
    for (const div of Array.from(el.querySelectorAll<HTMLElement>("div.poll[data-poll-name]"))) {
      let holder = div.querySelector<HTMLElement>(":scope > .poll-widget");
      if (!holder) {
        holder = document.createElement("div");
        holder.className = "poll-widget";
        div.appendChild(holder);
      }
      div.classList.add("poll-active");
      found.push({ name: div.dataset.pollName ?? "", el: holder });
    }
    setMounts(found);
  }, [html, hasPolls]);

  return (
    <>
      <div ref={ref} className={`cooked${className ? ` ${className}` : ""}`} dangerouslySetInnerHTML={{ __html: html }} />
      {mounts.map((mount) => {
        const poll = polls?.find((p) => p.name === mount.name);
        return poll && postId ? createPortal(<Poll poll={poll} postId={postId} onChange={(next) => onPollChange?.(next)} />, mount.el, mount.name) : null;
      })}
    </>
  );
}
