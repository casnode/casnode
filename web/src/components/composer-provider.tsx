"use client";

import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import type { Post, Topic, User } from "@/lib/types";

export type ComposerAction = "createTopic" | "reply" | "edit" | "privateMessage";

export interface ComposerOptions {
  action: ComposerAction;
  topic?: Pick<Topic, "id" | "title" | "slug" | "categoryId" | "archetype" | "tags">;
  replyToPost?: Pick<Post, "postNumber" | "userId"> & { username?: string };
  post?: Post;
  categoryId?: number;
  recipients?: string[];
  title?: string;
  raw?: string;
  author?: User;
}

export interface ComposerState extends ComposerOptions {
  key: number;
  minimized: boolean;
}

interface ComposerContextValue {
  composer: ComposerState | null;
  open: (options: ComposerOptions) => void;
  close: () => void;
  setMinimized: (minimized: boolean) => void;
  // text to insert at the caret of an open composer (a quote), or to start a reply with
  pendingInsert: string | null;
  insert: (text: string) => void;
  consumeInsert: () => void;
}

const ComposerContext = createContext<ComposerContextValue | null>(null);

export function ComposerProvider({ children }: { children: ReactNode }) {
  const [composer, setComposer] = useState<ComposerState | null>(null);
  const [pendingInsert, setPendingInsert] = useState<string | null>(null);

  const open = useCallback((options: ComposerOptions) => {
    setComposer((current) => {
      // asking again for the composer already open just shows it
      if (current && current.action === options.action && current.topic?.id === options.topic?.id && current.post?.id === options.post?.id && current.replyToPost?.postNumber === options.replyToPost?.postNumber) {
        return { ...current, minimized: false };
      }
      return { ...options, key: Date.now(), minimized: false };
    });
  }, []);

  const close = useCallback(() => setComposer(null), []);
  const setMinimized = useCallback((minimized: boolean) => setComposer((c) => (c ? { ...c, minimized } : c)), []);
  const insert = useCallback((text: string) => setPendingInsert(text), []);
  const consumeInsert = useCallback(() => setPendingInsert(null), []);

  const value = useMemo(
    () => ({ composer, open, close, setMinimized, pendingInsert, insert, consumeInsert }),
    [composer, open, close, setMinimized, pendingInsert, insert, consumeInsert],
  );
  return <ComposerContext.Provider value={value}>{children}</ComposerContext.Provider>;
}

export function useComposer(): ComposerContextValue {
  const value = useContext(ComposerContext);
  if (!value) {
    throw new Error("useComposer() outside ComposerProvider");
  }
  return value;
}

// Pages that show posts listen for this to show a post the composer just saved without reloading.
export const postSavedEvent = "casnode:post-saved";
export interface PostSavedDetail {
  topicId: number;
  post: Post;
  edited: boolean;
}
