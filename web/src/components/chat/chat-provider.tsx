"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useMessageBus } from "@/components/message-bus";
import { useSite } from "@/components/site-provider";
import { apiGet } from "@/lib/api";
import type { ChatChannel, ChatMessage } from "@/lib/types";

// The user's chat channels and their unread counts, kept up to date from the message bus for the sidebar, the
// header and the chat pages.
interface ChatValue {
  enabled: boolean;
  loaded: boolean;
  channels: ChatChannel[];
  directChannels: ChatChannel[];
  unreadChannels: number;
  unreadDirect: number;
  activeChannelId: number;
  setActiveChannelId: (id: number) => void;
  reload: () => Promise<void>;
  markRead: (channelId: number, messageId: number) => void;
}

const ChatContext = createContext<ChatValue | null>(null);

export function ChatProvider({ children }: { children: ReactNode }) {
  const { currentUser, settings } = useSite();
  const enabled = !!currentUser && settings.chat_enabled !== "false";
  const [channels, setChannels] = useState<ChatChannel[]>([]);
  const [directChannels, setDirectChannels] = useState<ChatChannel[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [activeChannelId, setActiveChannelId] = useState(0);

  const reload = useCallback(async () => {
    if (!enabled) {
      return;
    }
    try {
      const res = await apiGet<{ channels: ChatChannel[]; directChannels: ChatChannel[] }>("get-chat-channels");
      setChannels(res.data.channels ?? []);
      setDirectChannels(res.data.directChannels ?? []);
    } finally {
      setLoaded(true);
    }
  }, [enabled]);

  useEffect(() => {
    reload().catch(() => undefined);
  }, [reload]);

  const update = useCallback((channelId: number, change: (c: ChatChannel) => ChatChannel) => {
    setChannels((list) => list.map((c) => (c.id === channelId ? change(c) : c)));
    setDirectChannels((list) => list.map((c) => (c.id === channelId ? change(c) : c)));
  }, []);

  const markRead = useCallback(
    (channelId: number, messageId: number) => {
      update(channelId, (c) => ({ ...c, unreadCount: 0, membership: c.membership ? { ...c.membership, lastReadMessageId: Math.max(messageId, c.membership.lastReadMessageId) } : c.membership }));
    },
    [update],
  );

  useMessageBus(enabled ? "/chat/" : null, (message) => {
    const data = message.data as { type: string; message?: ChatMessage };
    if (data.type !== "sent" || !data.message) {
      return;
    }
    const sent = data.message;
    const known = channels.some((c) => c.id === sent.channelId) || directChannels.some((c) => c.id === sent.channelId);
    if (!known) {
      // a direct message from someone new, or a channel just joined elsewhere
      if (sent.userId !== currentUser?.id) {
        reload().catch(() => undefined);
      }
      return;
    }
    const reading = activeChannelId === sent.channelId && document.visibilityState === "visible";
    const mine = sent.userId === currentUser?.id;
    update(sent.channelId, (c) => ({
      ...c,
      lastMessageId: sent.id,
      lastMessageAt: sent.createdAt,
      unreadCount: reading || mine || !c.membership?.following ? c.unreadCount : c.unreadCount + 1,
    }));
    setDirectChannels((list) => [...list].sort((a, b) => (b.lastMessageAt > a.lastMessageAt ? 1 : -1)));
  });

  useMessageBus(enabled ? "/chat-user" : null, (message) => {
    const data = message.data as { type: string; channelId?: number; messageId?: number };
    if (data.type === "read" && data.channelId) {
      markRead(data.channelId, data.messageId ?? 0);
    } else {
      reload().catch(() => undefined);
    }
  });

  const value = useMemo<ChatValue>(
    () => ({
      enabled,
      loaded,
      channels,
      directChannels,
      unreadChannels: channels.filter((c) => c.unreadCount > 0 && !c.membership?.muted).length,
      unreadDirect: directChannels.reduce((sum, c) => sum + c.unreadCount, 0),
      activeChannelId,
      setActiveChannelId,
      reload,
      markRead,
    }),
    [enabled, loaded, channels, directChannels, activeChannelId, reload, markRead],
  );
  return <ChatContext.Provider value={value}>{children}</ChatContext.Provider>;
}

export function useChat(): ChatValue {
  const value = useContext(ChatContext);
  if (!value) {
    throw new Error("useChat() outside ChatProvider");
  }
  return value;
}
