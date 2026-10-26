"use client";

import { useEffect, useRef } from "react";

// A post's HTML as the server rendered and sanitized it; code blocks get highlighted once in the browser.
export function Cooked({ html, className }: { html: string; className?: string }) {
  const ref = useRef<HTMLDivElement>(null);

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

  return <div ref={ref} className={`cooked${className ? ` ${className}` : ""}`} dangerouslySetInnerHTML={{ __html: html }} />;
}
