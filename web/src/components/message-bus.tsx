"use client";

import { createContext, useCallback, useContext, useEffect, useRef, type ReactNode } from "react";
import { useSite } from "@/components/site-provider";
import type { BusMessage } from "@/lib/types";

// The pages' end of the message bus (object/message_bus.go): one long poll for every channel the page listens to.
// A component subscribes with useMessageBus("/topic/12", callback); "/chat/" takes every chat channel.
type Callback = (message: BusMessage) => void;

interface MessageBusValue {
  subscribe: (channel: string, callback: Callback) => () => void;
}

const MessageBusContext = createContext<MessageBusValue | null>(null);

export function MessageBusProvider({ children }: { children: ReactNode }) {
  const { currentUser, refreshSite } = useSite();
  const subscribers = useRef(new Map<string, Set<Callback>>());
  const restart = useRef<() => void>(() => undefined);

  const subscribe = useCallback((channel: string, callback: Callback) => {
    let set = subscribers.current.get(channel);
    const isNew = !set;
    if (!set) {
      set = new Set();
      subscribers.current.set(channel, set);
    }
    set.add(callback);
    if (isNew) {
      restart.current();
    }
    return () => {
      const current = subscribers.current.get(channel);
      current?.delete(callback);
      if (current && current.size === 0) {
        subscribers.current.delete(channel);
      }
    };
  }, []);

  useEffect(() => {
    let stopped = false;
    let since = -1;
    let controller: AbortController | null = null;
    let failures = 0;

    const deliver = (message: BusMessage) => {
      for (const [channel, set] of subscribers.current) {
        if (channel === message.channel || (channel.endsWith("/") && message.channel.startsWith(channel))) {
          for (const callback of set) {
            try {
              callback(message);
            } catch (err) {
              console.error(err);
            }
          }
        }
      }
    };

    const loop = async () => {
      while (!stopped) {
        const channels = Array.from(subscribers.current.keys());
        controller = new AbortController();
        try {
          const params = new URLSearchParams({ since: String(since), channels: channels.join(",") });
          const res = await fetch(`/api/message-bus-poll?${params.toString()}`, { credentials: "same-origin", signal: controller.signal, cache: "no-store" });
          const json = await res.json();
          if (json.status !== "ok") {
            throw new Error(json.msg);
          }
          failures = 0;
          since = json.data.lastId;
          for (const message of json.data.messages as BusMessage[]) {
            since = Math.max(since, message.id);
            deliver(message);
          }
        } catch (err) {
          if (stopped) {
            return;
          }
          if ((err as Error).name !== "AbortError") {
            failures++;
            await new Promise((resolve) => setTimeout(resolve, Math.min(30000, 2000 * failures)));
          }
        }
        // a hidden tab asks less often
        if (document.hidden && !stopped) {
          await new Promise((resolve) => setTimeout(resolve, 5000));
        }
      }
    };

    restart.current = () => controller?.abort();
    loop();
    return () => {
      stopped = true;
      restart.current = () => undefined;
      controller?.abort();
    };
  }, []);

  // the notification badge and the New / Unread counts follow what happens
  useEffect(() => {
    if (!currentUser) {
      return;
    }
    let timer: ReturnType<typeof setTimeout> | undefined;
    const refresh = () => {
      clearTimeout(timer);
      timer = setTimeout(() => refreshSite().catch(() => undefined), 1500);
    };
    const offNotification = subscribe("/notification", refresh);
    const offLatest = subscribe("/latest", refresh);
    return () => {
      clearTimeout(timer);
      offNotification();
      offLatest();
    };
  }, [currentUser, subscribe, refreshSite]);

  return <MessageBusContext.Provider value={{ subscribe }}>{children}</MessageBusContext.Provider>;
}

export function useMessageBusContext(): MessageBusValue {
  const value = useContext(MessageBusContext);
  if (!value) {
    throw new Error("useMessageBus() outside MessageBusProvider");
  }
  return value;
}

// useMessageBus calls callback with each message of the channel while the component is shown.
export function useMessageBus(channel: string | null, callback: Callback) {
  const { subscribe } = useMessageBusContext();
  const ref = useRef(callback);
  useEffect(() => {
    ref.current = callback;
  });
  useEffect(() => {
    if (!channel) {
      return;
    }
    return subscribe(channel, (message) => ref.current(message));
  }, [channel, subscribe]);
}
