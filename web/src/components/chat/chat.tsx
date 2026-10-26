"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Fragment, useCallback, useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { Avatar } from "@/components/avatar";
import { CategoryBadge } from "@/components/category-badge";
import { useChat } from "@/components/chat/chat-provider";
import { CategoryChooser, UserChooser } from "@/components/choosers";
import { Cooked } from "@/components/cooked";
import { Icon } from "@/components/icon";
import { useMessageBus } from "@/components/message-bus";
import { useSite } from "@/components/site-provider";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import { longDate } from "@/lib/format";
import type { MessageKey } from "@/lib/i18n";
import type { ChatChannel, ChatMessage, ChatMessageList, ChatReaction, UserMap } from "@/lib/types";
import { chatChannelTitle, chatChannelUrl, userUrl } from "@/lib/utils";

// Discourse's full-page chat: the channels on the left, the channel on the right.

export function ChatShell({ children }: { children: ReactNode }) {
  const { t } = useSite();
  const pathname = usePathname();
  const listOnly = pathname === "/chat";
  return (
    <div className={`full-page-chat${listOnly ? " list-only" : ""}`}>
      <aside className="chat-channels-list">
        <ChatChannelLinks />
        <div className="chat-channels-footer">
          <Link href="/chat/browse" className="btn btn-flat btn-small">
            <Icon name="list" />
            {t("chat.browse")}
          </Link>
          <Link href="/chat/new-message" className="btn btn-flat btn-small">
            <Icon name="plus" />
            {t("chat.newMessage")}
          </Link>
        </div>
      </aside>
      <main className="chat-main">{children}</main>
    </div>
  );
}

// The channels and direct messages the user follows, as links with their unread counts; shown in the chat page and
// the sidebar.
export function ChatChannelLinks({ compact, onNavigate }: { compact?: boolean; onNavigate?: () => void }) {
  const { t, currentUser } = useSite();
  const { channels, directChannels, activeChannelId, loaded } = useChat();
  const link = (channel: ChatChannel) => {
    const direct = channel.channelType === "direct";
    const others = (channel.directUsers ?? []).filter((u) => u.id !== currentUser?.id);
    return (
      <li key={channel.id} className="sidebar-section-link-wrapper">
        <Link href={chatChannelUrl(channel)} className={`sidebar-section-link chat-channel-link${channel.id === activeChannelId ? " active" : ""}${channel.unreadCount > 0 ? " unread" : ""}`} onClick={onNavigate}>
          <span className="sidebar-section-link-prefix">{direct ? <Avatar user={others[0] ?? channel.directUsers?.[0]} size={20} /> : <Icon name="hashtag" />}</span>
          <span className="sidebar-section-link-content-text">{chatChannelTitle(channel, currentUser)}</span>
          {channel.unreadCount > 0 ? direct ? <span className="sidebar-section-link-content-badge urgent">{channel.unreadCount}</span> : <span className="sidebar-section-link-suffix unread" /> : null}
        </Link>
      </li>
    );
  };
  return (
    <>
      <div className="sidebar-section-wrapper" data-section-name="chat-channels">
        <div className="sidebar-section-header-wrapper">
          <span className="sidebar-section-header">
            <span className="sidebar-section-header-text">{t("chat.channels")}</span>
          </span>
          <Link href="/chat/browse" className="sidebar-section-header-button" title={t("chat.browse")} onClick={onNavigate}>
            <Icon name={compact ? "list" : "plus"} />
          </Link>
        </div>
        <ul className="sidebar-section-content">
          {channels.map(link)}
          {loaded && channels.length === 0 ? (
            <li className="sidebar-section-link-wrapper">
              <Link href="/chat/browse" className="sidebar-section-link secondary" onClick={onNavigate}>
                <span className="sidebar-section-link-prefix">
                  <Icon name="list" />
                </span>
                <span className="sidebar-section-link-content-text">{t("chat.browse")}</span>
              </Link>
            </li>
          ) : null}
        </ul>
      </div>
      <div className="sidebar-section-wrapper" data-section-name="chat-dms">
        <div className="sidebar-section-header-wrapper">
          <span className="sidebar-section-header">
            <span className="sidebar-section-header-text">{t("chat.directMessages")}</span>
          </span>
          <Link href="/chat/new-message" className="sidebar-section-header-button" title={t("chat.newMessage")} onClick={onNavigate}>
            <Icon name="plus" />
          </Link>
        </div>
        <ul className="sidebar-section-content">{directChannels.map(link)}</ul>
      </div>
    </>
  );
}

export function ChatIndex() {
  const { t } = useSite();
  const { channels, directChannels, loaded } = useChat();
  const router = useRouter();
  useEffect(() => {
    if (!loaded || window.innerWidth < 768) {
      return;
    }
    const first = channels[0] ?? directChannels[0];
    router.replace(first ? chatChannelUrl(first) : "/chat/browse");
  }, [loaded, channels, directChannels, router]);
  return <div className="chat-empty">{loaded ? t("chat.chooseChannel") : <div className="spinner" />}</div>;
}

const quickReactions = ["👍", "❤️", "😄", "🎉", "😮", "🙏"];
const moreReactions = ["😂", "😢", "🤔", "👀", "🔥", "💯", "✅", "👏", "🚀", "😅", "👎", "❓"];

function sameDay(a: string, b: string) {
  return new Date(a).toDateString() === new Date(b).toDateString();
}

function formatTime(value: string, locale: string) {
  return new Date(value).toLocaleTimeString(locale === "zh" ? "zh-CN" : "en-US", { hour: "2-digit", minute: "2-digit" });
}

function formatDay(value: string, locale: string, t: (key: MessageKey) => string) {
  const date = new Date(value);
  const today = new Date();
  const yesterday = new Date(Date.now() - 86400000);
  if (date.toDateString() === today.toDateString()) {
    return t("chat.today");
  }
  if (date.toDateString() === yesterday.toDateString()) {
    return t("chat.yesterday");
  }
  return date.toLocaleDateString(locale === "zh" ? "zh-CN" : "en-US", { year: "numeric", month: "long", day: "numeric", weekday: "long" });
}

export function ChatChannelView({ channelId, targetMessageId }: { channelId: number; targetMessageId: number }) {
  const { t, locale, currentUser, categories } = useSite();
  const chat = useChat();
  const router = useRouter();
  const [channel, setChannel] = useState<ChatChannel | null>(null);
  const [error, setError] = useState("");
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [users, setUsers] = useState<UserMap>({});
  const [morePast, setMorePast] = useState(false);
  const [moreFuture, setMoreFuture] = useState(false);
  const [loading, setLoading] = useState(false);
  const [replyTo, setReplyTo] = useState<ChatMessage | null>(null);
  const [editing, setEditing] = useState<ChatMessage | null>(null);
  const [highlighted, setHighlighted] = useState(targetMessageId);
  const [editingChannel, setEditingChannel] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const anchor = useRef<{ id: number; top: number } | null>(null);
  const stickToBottom = useRef(true);
  const lastMarked = useRef(0);
  const { setActiveChannelId, markRead } = chat;

  const loadChannel = useCallback(async () => {
    try {
      const res = await apiGet<ChatChannel>("get-chat-channel", { id: channelId });
      setChannel(res.data);
    } catch (err) {
      setError(errorMessage(err));
    }
  }, [channelId]);

  useEffect(() => {
    setActiveChannelId(channelId);
    return () => setActiveChannelId(0);
  }, [channelId, setActiveChannelId]);

  useEffect(() => {
    let cancelled = false;
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadChannel();
    apiGet<ChatMessageList>("get-chat-messages", { channelId, around: targetMessageId || undefined })
      .then((res) => {
        if (cancelled) {
          return;
        }
        setMessages(res.data.messages ?? []);
        setUsers(res.data.users ?? {});
        setMorePast(res.data.morePast);
        setMoreFuture(res.data.moreFuture);
        stickToBottom.current = !targetMessageId;
      })
      .catch((err) => setError(errorMessage(err)));
    return () => {
      cancelled = true;
    };
  }, [channelId, targetMessageId, loadChannel]);

  // keep the view at the bottom as messages come, at the message asked for, or still when older ones are added above
  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (!el) {
      return;
    }
    if (anchor.current) {
      const node = el.querySelector<HTMLElement>(`[data-message-id="${anchor.current.id}"]`);
      if (node) {
        el.scrollTop += node.getBoundingClientRect().top - anchor.current.top;
      }
      anchor.current = null;
      return;
    }
    if (highlighted && messages.some((m) => m.id === highlighted)) {
      el.querySelector<HTMLElement>(`[data-message-id="${highlighted}"]`)?.scrollIntoView({ block: "center" });
      const timer = setTimeout(() => setHighlighted(0), 3000);
      return () => clearTimeout(timer);
    }
    if (stickToBottom.current) {
      el.scrollTop = el.scrollHeight;
    }
  }, [messages, highlighted]);

  const markLatestRead = useCallback(() => {
    const last = messages[messages.length - 1];
    if (!last || moreFuture || last.id <= lastMarked.current || document.visibilityState !== "visible" || !channel?.membership) {
      return;
    }
    lastMarked.current = last.id;
    markRead(channelId, last.id);
    apiPost("mark-chat-read", { channelId, messageId: last.id }).catch(() => undefined);
  }, [messages, moreFuture, channel, channelId, markRead]);

  useEffect(() => {
    if (stickToBottom.current) {
      markLatestRead();
    }
  }, [markLatestRead]);

  useEffect(() => {
    const onVisible = () => stickToBottom.current && markLatestRead();
    document.addEventListener("visibilitychange", onVisible);
    return () => document.removeEventListener("visibilitychange", onVisible);
  }, [markLatestRead]);

  const loadPast = async () => {
    if (loading || !morePast || messages.length === 0) {
      return;
    }
    setLoading(true);
    try {
      const first = messages[0];
      const res = await apiGet<ChatMessageList>("get-chat-messages", { channelId, before: first.id });
      const node = scrollRef.current?.querySelector<HTMLElement>(`[data-message-id="${first.id}"]`);
      anchor.current = node ? { id: first.id, top: node.getBoundingClientRect().top } : null;
      setUsers((u) => ({ ...u, ...res.data.users }));
      setMessages((list) => [...(res.data.messages ?? []).filter((m) => !list.some((x) => x.id === m.id)), ...list]);
      setMorePast(res.data.morePast);
    } finally {
      setLoading(false);
    }
  };

  const loadFuture = async () => {
    if (loading || !moreFuture || messages.length === 0) {
      return;
    }
    setLoading(true);
    try {
      const res = await apiGet<ChatMessageList>("get-chat-messages", { channelId, after: messages[messages.length - 1].id });
      setUsers((u) => ({ ...u, ...res.data.users }));
      setMessages((list) => [...list, ...(res.data.messages ?? []).filter((m) => !list.some((x) => x.id === m.id))]);
      setMoreFuture(res.data.moreFuture);
    } finally {
      setLoading(false);
    }
  };

  const onScroll = () => {
    const el = scrollRef.current;
    if (!el) {
      return;
    }
    stickToBottom.current = !moreFuture && el.scrollHeight - el.scrollTop - el.clientHeight < 60;
    if (el.scrollTop < 200) {
      loadPast();
    }
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 200) {
      loadFuture();
    }
    if (stickToBottom.current) {
      markLatestRead();
    }
  };

  useMessageBus(`/chat/${channelId}`, (message) => {
    const data = message.data as { type: string; message?: ChatMessage; users?: UserMap; messageId?: number; reactions?: ChatReaction[] };
    switch (data.type) {
      case "sent":
        if (!data.message || moreFuture) {
          return;
        }
        setUsers((u) => ({ ...u, ...data.users }));
        setMessages((list) => (list.some((m) => m.id === data.message!.id) ? list : [...list, data.message!]));
        return;
      case "edit":
      case "restore":
        if (data.message) {
          setUsers((u) => ({ ...u, ...data.users }));
          setMessages((list) => list.map((m) => (m.id === data.message!.id ? { ...data.message!, reactions: m.reactions } : m)));
        }
        return;
      case "delete":
        setMessages((list) =>
          currentUser?.admin || currentUser?.moderator ? list.map((m) => (m.id === data.messageId ? { ...m, deleted: true, cooked: "" } : m)) : list.filter((m) => m.id !== data.messageId),
        );
        return;
      case "reaction":
        setMessages((list) =>
          list.map((m) =>
            m.id === data.messageId ? { ...m, reactions: (data.reactions ?? []).map((r) => ({ ...r, reacted: !!currentUser && r.usernames.includes(currentUser.username) })) } : m,
          ),
        );
        return;
      case "channel":
        loadChannel();
        return;
    }
  });

  const send = async (raw: string) => {
    if (editing) {
      const res = await apiPost<ChatMessage>("edit-chat-message", undefined, { id: editing.id, raw });
      setMessages((list) => list.map((m) => (m.id === res.data.id ? { ...res.data, reactions: m.reactions } : m)));
      setEditing(null);
      return;
    }
    const res = await apiPost<ChatMessage>("send-chat-message", undefined, { channelId, raw, inReplyToId: replyTo?.id ?? 0 });
    setReplyTo(null);
    stickToBottom.current = true;
    if (currentUser) {
      setUsers((u) => ({ ...u, [currentUser.id]: currentUser }));
    }
    if (moreFuture) {
      router.replace(chatChannelUrl(channel ?? { id: channelId, slug: "" }));
      return;
    }
    const sent = { ...res.data, reactions: res.data.reactions ?? [], inReplyTo: replyTo ? { id: replyTo.id, userId: replyTo.userId, excerpt: replyTo.excerpt } : undefined };
    setMessages((list) => (list.some((m) => m.id === sent.id) ? list : [...list, sent]));
    if (!channel?.membership?.following) {
      loadChannel();
      chat.reload();
    }
  };

  const startEdit = async (message: ChatMessage) => {
    try {
      const res = await apiGet<ChatMessage>("get-chat-message", { id: message.id });
      setReplyTo(null);
      setEditing(res.data);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const react = async (message: ChatMessage, emoji: string) => {
    const existing = message.reactions.find((r) => r.emoji === emoji);
    try {
      const res = await apiPost<ChatReaction[]>("react-chat-message", { id: message.id, emoji, undo: existing?.reacted ? "true" : undefined });
      setMessages((list) => list.map((m) => (m.id === message.id ? { ...m, reactions: res.data } : m)));
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const remove = async (message: ChatMessage) => {
    if (!window.confirm(t("chat.confirmDelete"))) {
      return;
    }
    try {
      await apiPost("delete-chat-message", { id: message.id });
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const restore = async (message: ChatMessage) => {
    try {
      await apiPost("delete-chat-message", { id: message.id, restore: "true" });
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const follow = async (value: boolean) => {
    try {
      await apiPost("follow-chat-channel", { id: channelId, follow: String(value) });
      await loadChannel();
      await chat.reload();
      if (!value) {
        router.push("/chat");
      }
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  if (error) {
    return <div className="chat-empty">{error}</div>;
  }
  if (!channel) {
    return <div className="chat-empty">{<div className="spinner" />}</div>;
  }

  const direct = channel.channelType === "direct";
  const category = categories.find((c) => c.id === channel.categoryId);
  const following = !!channel.membership?.following;
  const lastOwn = [...messages].reverse().find((m) => m.userId === currentUser?.id && !m.deleted);

  return (
    <div className="chat-channel">
      <div className="chat-channel-header">
        <Link href="/chat" className="btn btn-flat btn-icon chat-back" aria-label={t("chat.channels")}>
          <Icon name="arrow-left" />
        </Link>
        <div className="chat-channel-title">
          {direct ? <Icon name="envelope" /> : <Icon name="hashtag" />}
          <h1>{chatChannelTitle(channel, currentUser)}</h1>
          {!direct && category ? <CategoryBadge category={category} categories={categories} /> : null}
          {channel.status !== "open" ? <span className="chat-channel-status">{t(`chat.status.${channel.status}` as MessageKey)}</span> : null}
        </div>
        {channel.description ? <div className="chat-channel-description">{channel.description}</div> : null}
        <div className="chat-channel-actions">
          {!direct ? (
            <span className="text-[var(--primary-medium)]" title={t("chat.members")}>
              <Icon name="users" /> {channel.userCount}
            </span>
          ) : null}
          {!direct && following ? (
            <button type="button" className="btn btn-flat btn-small" onClick={() => follow(false)}>
              {t("chat.leave")}
            </button>
          ) : null}
          {direct && following ? (
            <button type="button" className="btn btn-flat btn-small" title={t("chat.closeDirect")} onClick={() => follow(false)}>
              <Icon name="xmark" />
            </button>
          ) : null}
          {channel.canModerate ? (
            <button type="button" className="btn btn-flat btn-small" title={t("chat.editChannel")} onClick={() => setEditingChannel(true)}>
              <Icon name="gear" />
            </button>
          ) : null}
        </div>
      </div>
      <div className="chat-messages-scroll" ref={scrollRef} onScroll={onScroll}>
        {morePast ? <div className="chat-loading-more">{loading ? <div className="spinner small" /> : null}</div> : <div className="chat-channel-start">{t("chat.channelStart")}</div>}
        {messages.map((message, i) => {
          const previous = messages[i - 1];
          const newDay = !previous || !sameDay(previous.createdAt, message.createdAt);
          const grouped =
            !newDay && previous && previous.userId === message.userId && !message.inReplyTo && new Date(message.createdAt).getTime() - new Date(previous.createdAt).getTime() < 5 * 60000;
          return (
            <Fragment key={message.id}>
              {newDay ? (
                <div className="chat-day-separator">
                  <span suppressHydrationWarning>{formatDay(message.createdAt, locale, t)}</span>
                </div>
              ) : null}
              <ChatMessageItem
                message={message}
                users={users}
                grouped={!!grouped}
                highlighted={highlighted === message.id}
                canReact={channel.canSend}
                onReply={() => {
                  setEditing(null);
                  setReplyTo(message);
                }}
                onEdit={() => startEdit(message)}
                onDelete={() => remove(message)}
                onRestore={() => restore(message)}
                onReact={(emoji) => react(message, emoji)}
                onJump={(id) => {
                  if (messages.some((m) => m.id === id)) {
                    setHighlighted(id);
                  } else {
                    router.push(chatChannelUrl(channel, id));
                  }
                }}
                link={chatChannelUrl(channel, message.id)}
              />
            </Fragment>
          );
        })}
        {moreFuture ? <div className="chat-loading-more">{loading ? <div className="spinner small" /> : null}</div> : null}
      </div>
      {channel.canSend ? (
        <ChatComposer
          key={channel.id}
          channelName={chatChannelTitle(channel, currentUser)}
          replyTo={replyTo}
          replyUser={replyTo ? users[replyTo.userId]?.username : undefined}
          editing={editing}
          onCancel={() => {
            setReplyTo(null);
            setEditing(null);
          }}
          onSend={send}
          onEditLast={() => lastOwn && startEdit(lastOwn)}
        />
      ) : (
        <div className="chat-composer-disabled">{channel.status === "open" ? t("chat.cantSend") : t(`chat.status.${channel.status}` as MessageKey)}</div>
      )}
      {!following && !direct && channel.canSend ? (
        <div className="chat-join-bar">
          <span>{t("chat.previewing")}</span>
          <button type="button" className="btn btn-primary btn-small" onClick={() => follow(true)}>
            {t("chat.join")}
          </button>
        </div>
      ) : null}
      {editingChannel ? <ChannelFormModal channel={channel} onClose={() => setEditingChannel(false)} onSaved={() => { setEditingChannel(false); loadChannel(); chat.reload(); }} /> : null}
    </div>
  );
}

function ChatMessageItem({
  message,
  users,
  grouped,
  highlighted,
  canReact,
  onReply,
  onEdit,
  onDelete,
  onRestore,
  onReact,
  onJump,
  link,
}: {
  message: ChatMessage;
  users: UserMap;
  grouped: boolean;
  highlighted: boolean;
  canReact: boolean;
  onReply: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onRestore: () => void;
  onReact: (emoji: string) => void;
  onJump: (id: number) => void;
  link: string;
}) {
  const { t, locale, currentUser } = useSite();
  const [picker, setPicker] = useState(false);
  const author = users[message.userId];
  const mine = message.userId === currentUser?.id;
  const staff = !!currentUser && (currentUser.admin || currentUser.moderator);
  const parentAuthor = message.inReplyTo ? users[message.inReplyTo.userId] : undefined;

  const copyLink = async () => {
    const full = `${window.location.origin}${link}`;
    try {
      await navigator.clipboard.writeText(full);
      toast.success(t("topic.linkCopied"));
    } catch {
      window.prompt(t("share.title"), full);
    }
  };

  return (
    <div className={`chat-message-container${grouped ? " is-grouped" : ""}${highlighted ? " highlighted" : ""}${message.deleted ? " deleted" : ""}`} data-message-id={message.id}>
      {message.inReplyTo ? (
        <button type="button" className="chat-reply" onClick={() => onJump(message.inReplyTo!.id)}>
          <Icon name="share-nodes" className="rotate-180 scale-y-[-1]" />
          <Avatar user={parentAuthor} size={16} />
          <span className="chat-reply-excerpt">{message.inReplyTo.excerpt || t("chat.deletedMessage")}</span>
        </button>
      ) : null}
      <div className="chat-message">
        <div className="chat-message-left">
          {grouped ? (
            <span className="chat-time-hover" suppressHydrationWarning>
              {formatTime(message.createdAt, locale)}
            </span>
          ) : (
            <Link href={author ? userUrl(author.username) : "#"} data-user-card={author?.username}>
              <Avatar user={author} size={32} />
            </Link>
          )}
        </div>
        <div className="chat-message-content">
          {!grouped ? (
            <div className="chat-message-info">
              <Link href={author ? userUrl(author.username) : "#"} data-user-card={author?.username} className="chat-message-username">
                {author?.username}
              </Link>
              <Link href={link} className="chat-message-time" title={longDate(message.createdAt, locale)} suppressHydrationWarning>
                {formatTime(message.createdAt, locale)}
              </Link>
            </div>
          ) : null}
          {message.deleted ? (
            <div className="chat-message-deleted">{t("chat.deletedMessage")}</div>
          ) : (
            <div className="chat-message-text">
              <Cooked html={message.cooked} />
              {message.edited ? <span className="chat-message-edited">{t("chat.edited")}</span> : null}
            </div>
          )}
          {message.reactions.length > 0 ? (
            <div className="chat-message-reactions">
              {message.reactions.map((reaction) => (
                <button
                  key={reaction.emoji}
                  type="button"
                  className={`chat-message-reaction${reaction.reacted ? " reacted" : ""}`}
                  title={reaction.usernames.join(", ")}
                  disabled={!canReact}
                  onClick={() => onReact(reaction.emoji)}
                >
                  <span className="emoji">{reaction.emoji}</span>
                  <span className="count">{reaction.count}</span>
                </button>
              ))}
            </div>
          ) : null}
        </div>
        {!message.deleted || staff ? (
          <div className="chat-message-actions">
            {!message.deleted && canReact ? (
              <>
                {quickReactions.slice(0, 3).map((emoji) => (
                  <button key={emoji} type="button" className="btn btn-flat" onClick={() => onReact(emoji)}>
                    {emoji}
                  </button>
                ))}
                <span className="relative">
                  <button type="button" className="btn btn-flat" title={t("chat.react")} onClick={() => setPicker(!picker)}>
                    <Icon name="face-smile" />
                  </button>
                  {picker ? (
                    <div className="chat-emoji-picker" onMouseLeave={() => setPicker(false)}>
                      {[...quickReactions, ...moreReactions].map((emoji) => (
                        <button
                          key={emoji}
                          type="button"
                          onClick={() => {
                            setPicker(false);
                            onReact(emoji);
                          }}
                        >
                          {emoji}
                        </button>
                      ))}
                    </div>
                  ) : null}
                </span>
                <button type="button" className="btn btn-flat" title={t("chat.reply")} onClick={onReply}>
                  <Icon name="reply" />
                </button>
              </>
            ) : null}
            {!message.deleted ? (
              <button type="button" className="btn btn-flat" title={t("topic.copyLink")} onClick={copyLink}>
                <Icon name="link" />
              </button>
            ) : null}
            {mine && !message.deleted ? (
              <button type="button" className="btn btn-flat" title={t("chat.edit")} onClick={onEdit}>
                <Icon name="pencil" />
              </button>
            ) : null}
            {(mine || staff) && !message.deleted ? (
              <button type="button" className="btn btn-flat" title={t("chat.delete")} onClick={onDelete}>
                <Icon name="trash-can" />
              </button>
            ) : null}
            {staff && message.deleted ? (
              <button type="button" className="btn btn-flat" title={t("chat.restore")} onClick={onRestore}>
                <Icon name="rotate-left" />
              </button>
            ) : null}
          </div>
        ) : null}
      </div>
    </div>
  );
}

function ChatComposer({
  channelName,
  replyTo,
  replyUser,
  editing,
  onCancel,
  onSend,
  onEditLast,
}: {
  channelName: string;
  replyTo: ChatMessage | null;
  replyUser?: string;
  editing: ChatMessage | null;
  onCancel: () => void;
  onSend: (raw: string) => Promise<void>;
  onEditLast: () => void;
}) {
  const { t, settings } = useSite();
  const [raw, setRaw] = useState("");
  const [busy, setBusy] = useState(false);
  const [prevEditing, setPrevEditing] = useState(editing);
  const ref = useRef<HTMLTextAreaElement>(null);
  const draftKey = `casnode-chat-draft-${channelName}`;

  if (prevEditing !== editing) {
    setPrevEditing(editing);
    setRaw(editing?.raw ?? "");
  }

  useEffect(() => {
    try {
      const draft = localStorage.getItem(draftKey);
      if (draft) {
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setRaw(draft);
      }
    } catch {
      // no drafts in private windows
    }
  }, [draftKey]);

  useEffect(() => {
    if (editing) {
      return;
    }
    try {
      if (raw) {
        localStorage.setItem(draftKey, raw);
      } else {
        localStorage.removeItem(draftKey);
      }
    } catch {
      // no drafts in private windows
    }
  }, [raw, draftKey, editing]);

  useEffect(() => {
    ref.current?.focus();
  }, [replyTo, editing]);

  useLayoutEffect(() => {
    const el = ref.current;
    if (el) {
      el.style.height = "auto";
      el.style.height = `${Math.min(el.scrollHeight, 200)}px`;
    }
  }, [raw]);

  const submit = async () => {
    const text = raw.trim();
    if (!text || busy) {
      return;
    }
    setBusy(true);
    try {
      await onSend(text);
      setRaw("");
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
      ref.current?.focus();
    }
  };

  const max = Number(settings.chat_max_message_length || 6000);
  return (
    <div className="chat-composer-container">
      {replyTo || editing ? (
        <div className="chat-composer-message-details">
          <Icon name={editing ? "pencil" : "reply"} />
          <span className="truncate">{editing ? t("chat.editing") : `${replyUser ?? ""}: ${replyTo?.excerpt ?? ""}`}</span>
          <button type="button" className="btn btn-flat btn-icon ml-auto" aria-label={t("common.cancel")} onClick={onCancel}>
            <Icon name="xmark" />
          </button>
        </div>
      ) : null}
      <div className="chat-composer">
        <textarea
          ref={ref}
          className="chat-composer-input"
          rows={1}
          value={raw}
          maxLength={max}
          placeholder={t("chat.placeholder", { channel: channelName })}
          onChange={(e) => setRaw(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
              e.preventDefault();
              submit();
            } else if (e.key === "Escape" && (replyTo || editing)) {
              e.preventDefault();
              onCancel();
              setRaw("");
            } else if (e.key === "ArrowUp" && !raw) {
              e.preventDefault();
              onEditLast();
            }
          }}
        />
        <button type="button" className="btn btn-primary btn-icon chat-composer-send" disabled={!raw.trim() || busy} aria-label={t("chat.send")} onClick={submit}>
          <Icon name="paper-plane" />
        </button>
      </div>
    </div>
  );
}

function ChannelFormModal({ channel, onClose, onSaved }: { channel?: ChatChannel; onClose: () => void; onSaved: (channel: ChatChannel) => void }) {
  const { t, currentUser } = useSite();
  const [name, setName] = useState(channel?.name ?? "");
  const [description, setDescription] = useState(channel?.description ?? "");
  const [categoryId, setCategoryId] = useState(channel?.categoryId ?? 0);
  const [status, setStatus] = useState(channel?.status ?? "open");
  const [busy, setBusy] = useState(false);
  const router = useRouter();

  const save = async () => {
    setBusy(true);
    try {
      const res = await apiPost<ChatChannel>(channel ? "update-chat-channel" : "create-chat-channel", undefined, { id: channel?.id ?? 0, name, description, categoryId, status });
      onSaved(res.data);
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    if (!channel || !window.confirm(t("chat.confirmDeleteChannel"))) {
      return;
    }
    try {
      await apiPost("delete-chat-channel", { id: channel.id });
      onClose();
      router.push("/chat/browse");
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        title={channel ? t("chat.editChannel") : t("chat.newChannel")}
        footer={
          <>
            <button type="button" className="btn btn-primary" disabled={busy || !categoryId} onClick={save}>
              {channel ? t("user.save") : t("chat.createChannel")}
            </button>
            {channel && currentUser?.admin ? (
              <button type="button" className="btn btn-danger ml-auto" onClick={remove}>
                {t("chat.deleteChannel")}
              </button>
            ) : null}
          </>
        }
      >
        <div className="control-group">
          <label className="form-label">{t("chat.category")}</label>
          <CategoryChooser value={categoryId} onChange={setCategoryId} placeholder={t("composer.chooseCategory")} />
          <div className="instructions">{t("chat.categoryHelp")}</div>
        </div>
        <div className="control-group">
          <label className="form-label">{t("chat.channelName")}</label>
          <input className="d-input" value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="control-group">
          <label className="form-label">{t("chat.channelDescription")}</label>
          <textarea className="d-input" rows={2} value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        {channel ? (
          <div className="control-group">
            <label className="form-label">{t("chat.channelStatus")}</label>
            <select className="d-select" value={status} onChange={(e) => setStatus(e.target.value as ChatChannel["status"])}>
              {(["open", "read_only", "closed", "archived"] as const).map((s) => (
                <option key={s} value={s}>
                  {t(`chat.status.${s}` as MessageKey)}
                </option>
              ))}
            </select>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

export function ChatBrowse() {
  const { t, currentUser, categories } = useSite();
  const chat = useChat();
  const router = useRouter();
  const [status, setStatus] = useState("open");
  const [term, setTerm] = useState("");
  const [channels, setChannels] = useState<ChatChannel[] | null>(null);
  const [creating, setCreating] = useState(false);
  const [version, setVersion] = useState(0);
  const staff = !!currentUser && (currentUser.admin || currentUser.moderator);

  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(() => {
      apiGet<ChatChannel[]>("browse-chat-channels", { status, term: term.trim() })
        .then((res) => !cancelled && setChannels(res.data ?? []))
        .catch((err) => toast.error(errorMessage(err)));
    }, 150);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [status, term, version]);

  const follow = async (channel: ChatChannel, value: boolean) => {
    try {
      await apiPost("follow-chat-channel", { id: channel.id, follow: String(value) });
      setVersion((v) => v + 1);
      chat.reload();
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  return (
    <div className="chat-browse">
      <div className="chat-browse-header">
        <h1>{t("chat.allChannels")}</h1>
        {staff ? (
          <button type="button" className="btn btn-primary ml-auto" onClick={() => setCreating(true)}>
            <Icon name="plus" />
            {t("chat.newChannel")}
          </button>
        ) : null}
      </div>
      <div className="chat-browse-filters">
        {(["all", "open", "closed", "archived"] as const).map((s) => (
          <button key={s} type="button" className={`btn btn-small ${status === s ? "btn-primary" : "btn-flat"}`} onClick={() => setStatus(s)}>
            {t(`chat.filter.${s}` as MessageKey)}
          </button>
        ))}
        <input className="d-input max-w-[16em] ml-auto" value={term} placeholder={t("chat.filterChannels")} onChange={(e) => setTerm(e.target.value)} />
      </div>
      {channels === null ? <div className="spinner" /> : null}
      {channels && channels.length === 0 ? <p className="text-[var(--primary-medium)]">{t("chat.noChannels")}</p> : null}
      <div className="chat-browse-list">
        {(channels ?? []).map((channel) => {
          const category = categories.find((c) => c.id === channel.categoryId);
          const following = !!channel.membership?.following;
          return (
            <div key={channel.id} className="chat-channel-card">
              <Link href={chatChannelUrl(channel)} className="chat-channel-card-title">
                <Icon name="hashtag" />
                {channel.name}
              </Link>
              {category ? <CategoryBadge category={category} categories={categories} /> : null}
              {channel.description ? <p>{channel.description}</p> : null}
              <div className="chat-channel-card-footer">
                <span className="text-[var(--primary-medium)]">
                  <Icon name="users" /> {t("chat.memberCount", { count: channel.userCount })}
                </span>
                {channel.status !== "open" ? <span className="chat-channel-status">{t(`chat.status.${channel.status}` as MessageKey)}</span> : null}
                {currentUser ? (
                  following ? (
                    <button type="button" className="btn btn-default btn-small ml-auto" onClick={() => follow(channel, false)}>
                      {t("chat.leave")}
                    </button>
                  ) : (
                    <button type="button" className="btn btn-primary btn-small ml-auto" onClick={() => follow(channel, true)}>
                      {t("chat.join")}
                    </button>
                  )
                ) : null}
              </div>
            </div>
          );
        })}
      </div>
      {creating ? (
        <ChannelFormModal
          onClose={() => setCreating(false)}
          onSaved={(channel) => {
            setCreating(false);
            chat.reload();
            router.push(chatChannelUrl(channel));
          }}
        />
      ) : null}
    </div>
  );
}

export function NewDirectMessage() {
  const { t } = useSite();
  const router = useRouter();
  const [usernames, setUsernames] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const start = async () => {
    setBusy(true);
    try {
      const res = await apiPost<ChatChannel>("create-direct-channel", undefined, { usernames });
      router.push(chatChannelUrl(res.data));
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="chat-new-message">
      <h1>{t("chat.newMessage")}</h1>
      <p className="text-[var(--primary-high)]">{t("chat.newMessageHelp")}</p>
      <UserChooser values={usernames} onChange={setUsernames} placeholder={t("composer.recipients")} />
      <button type="button" className="btn btn-primary mt-4" disabled={busy || usernames.length === 0} onClick={start}>
        <Icon name="comment" />
        {t("chat.startConversation")}
      </button>
    </div>
  );
}
