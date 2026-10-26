"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useComposer } from "@/components/composer-provider";
import { useSite } from "@/components/site-provider";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import type { MessageKey } from "@/lib/i18n";
import { userUrl } from "@/lib/utils";

// Discourse's keyboard shortcuts: g then a letter to go somewhere, j / k to move through topics and posts, and
// single letters for what to do with the selected one. ? shows them all.
const sections: { title: MessageKey; items: [string, MessageKey][] }[] = [
  {
    title: "shortcuts.jumpTo",
    items: [
      ["g h", "shortcuts.home"],
      ["g l", "shortcuts.latest"],
      ["g n", "shortcuts.new"],
      ["g u", "shortcuts.unread"],
      ["g c", "shortcuts.categories"],
      ["g t", "shortcuts.top"],
      ["g b", "shortcuts.bookmarks"],
      ["g p", "shortcuts.profile"],
      ["g m", "shortcuts.messages"],
      ["-", "shortcuts.chat"],
    ],
  },
  {
    title: "shortcuts.navigation",
    items: [
      ["j / k", "shortcuts.nextPrevious"],
      ["o / Enter", "shortcuts.open"],
      ["u", "shortcuts.back"],
      [".", "shortcuts.showNew"],
      ["shift j / shift k", "shortcuts.nextPreviousSection"],
    ],
  },
  {
    title: "shortcuts.application",
    items: [
      ["c", "shortcuts.createTopic"],
      ["/", "shortcuts.search"],
      ["=", "shortcuts.sidebar"],
      ["p", "shortcuts.profileMenu"],
      ["?", "shortcuts.help"],
      ["Esc", "shortcuts.close"],
    ],
  },
  {
    title: "shortcuts.actions",
    items: [
      ["l", "shortcuts.like"],
      ["r", "shortcuts.reply"],
      ["shift r", "shortcuts.replyTopic"],
      ["e", "shortcuts.edit"],
      ["b", "shortcuts.bookmark"],
      ["s", "shortcuts.share"],
      ["shift a", "shortcuts.adminMenu"],
      ["ctrl Enter", "shortcuts.submit"],
    ],
  },
];

function isTyping(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  const el = target;
  return el.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName) || !!el.closest("[role=dialog] input, [role=dialog] textarea");
}

function click(selector: string, root: ParentNode = document): boolean {
  const el = root.querySelector<HTMLElement>(selector);
  if (!el) {
    return false;
  }
  el.click();
  return true;
}

// clicks a post action that may be behind the "..." button
function clickPostAction(post: HTMLElement, selector: string) {
  if (click(selector, post)) {
    return;
  }
  if (click(".show-more-actions", post)) {
    setTimeout(() => click(selector, post), 0);
  }
}

export function KeyboardShortcuts() {
  const { currentUser, login } = useSite();
  const { open: openComposer } = useComposer();
  const router = useRouter();
  const [help, setHelp] = useState(false);
  const pending = useRef<{ key: string; at: number } | null>(null);

  useEffect(() => {
    const items = () => {
      const posts = Array.from(document.querySelectorAll<HTMLElement>(".post-stream .topic-post"));
      if (posts.length > 0) {
        return posts;
      }
      return Array.from(document.querySelectorAll<HTMLElement>(".topic-list .topic-list-item"));
    };
    const selectedItem = () => document.querySelector<HTMLElement>(".topic-post.selected, .topic-list-item.selected");

    const move = (delta: number) => {
      const list = items();
      if (list.length === 0) {
        return;
      }
      const current = selectedItem();
      let index = current ? list.indexOf(current) : -1;
      if (index < 0) {
        // start from what is on screen
        index = list.findIndex((el) => el.getBoundingClientRect().bottom > 80) - (delta > 0 ? 1 : 0);
      }
      const next = list[Math.max(0, Math.min(list.length - 1, index + delta))];
      current?.classList.remove("selected");
      next.classList.add("selected");
      const rect = next.getBoundingClientRect();
      if (rect.top < 70 || rect.bottom > window.innerHeight) {
        window.scrollTo({ top: rect.top + window.scrollY - 80 });
      }
    };

    const go = (path: string, needsUser = false) => {
      if (needsUser && !currentUser) {
        login();
        return;
      }
      router.push(path);
    };

    const onKey = (e: KeyboardEvent) => {
      if (e.ctrlKey || e.metaKey || e.altKey || e.defaultPrevented || isTyping(e.target) || document.querySelector(".d-modal__container")) {
        return;
      }
      const key = e.key;
      const now = Date.now();
      const first = pending.current && now - pending.current.at < 1500 ? pending.current.key : "";
      pending.current = null;

      if (first === "g") {
        const me = currentUser?.username ?? "";
        const targets: Record<string, [string, boolean]> = {
          h: ["/", false],
          l: ["/latest", false],
          n: ["/new", true],
          u: ["/unread", true],
          c: ["/categories", false],
          t: ["/top", false],
          b: [userUrl(me, "activity/bookmarks"), true],
          p: [userUrl(me, "activity"), true],
          m: [userUrl(me, "messages"), true],
        };
        const target = targets[key];
        if (target) {
          e.preventDefault();
          go(target[0], target[1]);
        }
        return;
      }

      const selected = selectedItem();
      const post = selected?.classList.contains("topic-post") ? selected : null;
      const handled = () => e.preventDefault();
      switch (key) {
        case "g":
          pending.current = { key, at: now };
          return;
        case "?":
          handled();
          setHelp(true);
          return;
        case "j":
        case "k":
          handled();
          move(key === "j" ? 1 : -1);
          return;
        case "J":
        case "K":
          handled();
          move(key === "J" ? 10 : -10);
          return;
        case "o":
        case "Enter":
          if (selected && !post) {
            handled();
            click("a.title", selected);
          }
          return;
        case "u":
          handled();
          router.back();
          return;
        case ".":
          if (click(".show-more-topics")) {
            handled();
          }
          return;
        case "c":
          handled();
          if (!currentUser) {
            login();
          } else {
            openComposer({ action: "createTopic" });
          }
          return;
        case "/":
          handled();
          click(".search-dropdown");
          return;
        case "=":
          handled();
          click(".header-sidebar-toggle button");
          return;
        case "p":
          if (!post) {
            handled();
            click(".current-user");
          }
          return;
        case "-":
          if (currentUser) {
            handled();
            go("/chat");
          }
          return;
        case "R":
          handled();
          click("#topic-footer-buttons .create");
          return;
        case "A":
          handled();
          click(".toggle-admin-menu");
          return;
      }
      if (!post) {
        return;
      }
      const actions: Record<string, string> = {
        l: ".toggle-like",
        r: "button.reply",
        e: "button.edit",
        b: "button.bookmark",
        s: ".post-action-menu__copy-link",
      };
      if (actions[key]) {
        handled();
        clickPostAction(post, actions[key]);
      }
    };

    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [currentUser, login, openComposer, router]);

  return <KeyboardHelp open={help} onClose={() => setHelp(false)} />;
}

function KeyboardHelp({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useSite();
  if (!open) {
    return null;
  }
  return (
    <Dialog open onOpenChange={(value) => !value && onClose()}>
      <DialogContent title={t("shortcuts.title")} className="keyboard-shortcuts-modal">
        <div className="keyboard-shortcuts">
          {sections.map((section) => (
            <section key={section.title}>
              <h4>{t(section.title)}</h4>
              <ul>
                {section.items.map(([keys, label]) => (
                  <li key={keys}>
                    <span className="keys">
                      {keys.split(" / ").map((combo, i) => (
                        <span key={combo}>
                          {i > 0 ? " / " : null}
                          {combo.split(" ").map((k) => (
                            <kbd key={k}>{k}</kbd>
                          ))}
                        </span>
                      ))}
                    </span>
                    <span>{t(label)}</span>
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
