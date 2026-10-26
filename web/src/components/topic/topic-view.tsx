"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { Fragment, useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { Avatar } from "@/components/avatar";
import { CategoryBadge } from "@/components/category-badge";
import { CategoryChooser, TagChooser } from "@/components/choosers";
import { postSavedEvent, useComposer, type PostSavedDetail } from "@/components/composer-provider";
import { Icon } from "@/components/icon";
import { useLayoutState } from "@/components/layout-state";
import { useMessageBus } from "@/components/message-bus";
import { useSite } from "@/components/site-provider";
import { TagList } from "@/components/tag-list";
import { TopicList, TopicStatuses } from "@/components/topic-list";
import { Post } from "@/components/topic/post";
import { MovePostsModal } from "@/components/topic/move-posts";
import { Timeline } from "@/components/topic/timeline";
import { NotificationsButton, TopicAdminMenu } from "@/components/topic/topic-controls";
import { TopicMap } from "@/components/topic/topic-map";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import type { Post as PostData, TopicView as TopicViewData, UserMap } from "@/lib/types";
import { categoryUrl, groupUrl, topicUrl, userUrl } from "@/lib/utils";

const chunk = 20;

export function TopicView({ initial, targetPostNumber }: { initial: TopicViewData; targetPostNumber: number }) {
  const { t, currentUser, categories, login, refreshSite } = useSite();
  const { open: openComposer, composer, insert } = useComposer();
  const { setHeaderTopic } = useLayoutState();
  const router = useRouter();

  const [prevInitial, setPrevInitial] = useState(initial);
  const [topic, setTopic] = useState(initial.topic);
  const [details, setDetails] = useState(initial.details);
  const [stream, setStream] = useState(initial.stream ?? []);
  const [posts, setPosts] = useState(initial.posts ?? []);
  const [users, setUsers] = useState<UserMap>(initial.users ?? {});
  const [current, setCurrent] = useState(0);
  const [highlighted, setHighlighted] = useState<number>(0);
  const [loading, setLoading] = useState<"" | "before" | "after">("");
  const [editingTitle, setEditingTitle] = useState(false);
  const [quote, setQuote] = useState<{ text: string; postNumber: number; username: string; x: number; y: number } | null>(null);
  const [selecting, setSelecting] = useState(false);
  const [selected, setSelected] = useState<number[]>([]);
  const [moving, setMoving] = useState(false);

  if (prevInitial !== initial) {
    setPrevInitial(initial);
    setTopic(initial.topic);
    setDetails(initial.details);
    setStream(initial.stream ?? []);
    setPosts(initial.posts ?? []);
    setUsers((u) => ({ ...u, ...initial.users }));
  }

  const titleRef = useRef<HTMLDivElement>(null);
  const topSentinel = useRef<HTMLDivElement>(null);
  const bottomSentinel = useRef<HTMLDivElement>(null);
  const prependAnchor = useRef<{ number: number; top: number } | null>(null);
  const scrollTarget = useRef<number>(targetPostNumber > 1 ? targetPostNumber : 0);
  const maxRead = useRef(0);
  const sentRead = useRef(0);

  const firstLoaded = posts[0]?.postNumber ?? 0;
  const lastLoaded = posts[posts.length - 1]?.postNumber ?? 0;
  const firstIndex = stream.indexOf(firstLoaded);
  const lastIndex = stream.indexOf(lastLoaded);
  const category = categories.find((c) => c.id === topic.categoryId) ?? initial.category ?? undefined;
  const url = topicUrl(topic);

  const addUsers = useCallback((more: UserMap) => setUsers((u) => ({ ...u, ...more })), []);

  const fetchPosts = useCallback(
    async (numbers: number[]) => {
      if (numbers.length === 0) {
        return [];
      }
      const res = await apiGet<PostData[], UserMap>("get-posts", { topicId: topic.id, postNumbers: numbers.join(",") });
      addUsers(res.data2 ?? {});
      return res.data ?? [];
    },
    [topic.id, addUsers],
  );

  const loadAfter = useCallback(async () => {
    if (loading || lastIndex < 0 || lastIndex >= stream.length - 1) {
      return;
    }
    setLoading("after");
    try {
      const more = await fetchPosts(stream.slice(lastIndex + 1, lastIndex + 1 + chunk));
      setPosts((list) => [...list, ...more.filter((p) => !list.some((x) => x.id === p.id))]);
    } finally {
      setLoading("");
    }
  }, [loading, lastIndex, stream, fetchPosts]);

  const loadBefore = useCallback(async () => {
    if (loading || firstIndex <= 0) {
      return;
    }
    setLoading("before");
    try {
      const more = await fetchPosts(stream.slice(Math.max(0, firstIndex - chunk), firstIndex));
      const anchor = document.getElementById(`post_${firstLoaded}`);
      prependAnchor.current = anchor ? { number: firstLoaded, top: anchor.getBoundingClientRect().top } : null;
      setPosts((list) => [...more.filter((p) => !list.some((x) => x.id === p.id)), ...list]);
    } finally {
      setLoading("");
    }
  }, [loading, firstIndex, firstLoaded, stream, fetchPosts]);

  // keep the post that was at the top in place when earlier posts are added above it
  useLayoutEffect(() => {
    const anchor = prependAnchor.current;
    if (!anchor) {
      return;
    }
    prependAnchor.current = null;
    const el = document.getElementById(`post_${anchor.number}`);
    if (el) {
      window.scrollBy(0, el.getBoundingClientRect().top - anchor.top);
    }
  }, [posts]);

  const scrollToPost = useCallback((postNumber: number, highlight: boolean) => {
    const el = document.getElementById(`post_${postNumber}`);
    if (!el) {
      return false;
    }
    const offset = postNumber === 1 ? 0 : el.getBoundingClientRect().top + window.scrollY - 80;
    window.scrollTo({ top: Math.max(0, offset) });
    if (highlight) {
      setHighlighted(postNumber);
      setTimeout(() => setHighlighted(0), 2500);
    }
    return true;
  }, []);

  const jumpTo = useCallback(
    async (index: number) => {
      const target = stream[Math.max(0, Math.min(index, stream.length - 1))];
      if (target === undefined) {
        return;
      }
      if (posts.some((p) => p.postNumber === target)) {
        scrollToPost(target, index > 0);
        return;
      }
      const start = Math.max(0, Math.min(index - 5, stream.length - chunk));
      const loaded = await fetchPosts(stream.slice(start, start + chunk));
      scrollTarget.current = target;
      setPosts(loaded);
    },
    [stream, posts, fetchPosts, scrollToPost],
  );

  // after the first render, or after jumping to posts that had to be loaded, go to the post asked for
  useEffect(() => {
    if (scrollTarget.current && scrollToPost(scrollTarget.current, true)) {
      scrollTarget.current = 0;
    }
  }, [posts, scrollToPost]);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) {
            continue;
          }
          if (entry.target === bottomSentinel.current) {
            loadAfter();
          } else if (entry.target === topSentinel.current && !scrollTarget.current && window.scrollY > 0) {
            loadBefore();
          }
        }
      },
      { rootMargin: "600px 0px" },
    );
    if (bottomSentinel.current) {
      observer.observe(bottomSentinel.current);
    }
    if (topSentinel.current) {
      observer.observe(topSentinel.current);
    }
    return () => observer.disconnect();
  }, [loadAfter, loadBefore]);

  // where the reader is: for the timeline, the address bar and what has been read
  useEffect(() => {
    let frame = 0;
    const onScroll = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => {
        const elements = Array.from(document.querySelectorAll<HTMLElement>(".post-stream [data-post-number]"));
        const viewportBottom = window.innerHeight;
        let currentNumber = 0;
        for (const el of elements) {
          const rect = el.getBoundingClientRect();
          const n = Number(el.dataset.postNumber);
          if (rect.top < viewportBottom - 50 && rect.bottom > 0) {
            if (!currentNumber && rect.bottom > 120) {
              currentNumber = n;
            }
            if (n > maxRead.current && rect.top < viewportBottom - 50) {
              maxRead.current = n;
            }
          }
        }
        // the address follows the post at the top; the timeline shows the last post once the page is scrolled to the end
        let timelineNumber = currentNumber;
        if (window.scrollY > 0 && window.innerHeight + window.scrollY >= document.body.scrollHeight - 300 && lastIndex === stream.length - 1) {
          maxRead.current = Math.max(maxRead.current, stream[stream.length - 1] ?? 0);
          timelineNumber = stream[stream.length - 1] ?? currentNumber;
        }
        if (window.innerHeight >= document.body.scrollHeight - 300 && lastIndex === stream.length - 1) {
          maxRead.current = Math.max(maxRead.current, stream[stream.length - 1] ?? 0);
        }
        if (timelineNumber) {
          const index = stream.indexOf(timelineNumber);
          if (index >= 0) {
            setCurrent(index);
          }
        }
        if (currentNumber) {
          const nextUrl = currentNumber > 1 ? `${url}/${currentNumber}` : url;
          if (window.location.pathname !== nextUrl && !scrollTarget.current) {
            window.history.replaceState(window.history.state, "", nextUrl + window.location.search);
          }
        }
        if (titleRef.current) {
          const titleVisible = titleRef.current.getBoundingClientRect().bottom > 64;
          setHeaderTopic(titleVisible ? null : { id: topic.id, title: topic.title, url, categoryId: topic.categoryId });
        }
      });
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    onScroll();
    return () => {
      window.removeEventListener("scroll", onScroll);
      cancelAnimationFrame(frame);
    };
  }, [stream, lastIndex, url, topic.id, topic.title, topic.categoryId, setHeaderTopic]);

  useEffect(() => () => setHeaderTopic(null), [setHeaderTopic]);

  // tell the server how far the user has read, a couple of seconds after it changes
  useEffect(() => {
    if (!currentUser) {
      return;
    }
    const timer = setInterval(() => {
      if (maxRead.current > sentRead.current) {
        const lastRead = maxRead.current;
        sentRead.current = lastRead;
        apiPost("record-topic-timing", { id: topic.id, lastRead })
          .then(() => refreshSite())
          .catch(() => undefined);
      }
    }, 3000);
    return () => clearInterval(timer);
  }, [currentUser, topic.id, refreshSite]);

  // a reply or edit saved in the composer shows up without reloading the page
  useEffect(() => {
    const onSaved = async (e: Event) => {
      const detail = (e as CustomEvent<PostSavedDetail>).detail;
      if (detail.topicId !== topic.id) {
        return;
      }
      const [fresh] = await fetchPosts([detail.post.postNumber]);
      if (!fresh) {
        return;
      }
      if (detail.edited) {
        setPosts((list) => list.map((p) => (p.id === fresh.id ? fresh : p)));
        return;
      }
      setStream((s) => (s.includes(fresh.postNumber) ? s : [...s, fresh.postNumber]));
      // posting in a topic starts tracking it, as the server has just recorded
      setTopic((tp) => ({ ...countNewPost(tp, fresh), notificationLevel: Math.max(tp.notificationLevel, 2) }));
      if (lastIndex === stream.length - 1) {
        setPosts((list) => appendPost(list, fresh));
        scrollTarget.current = fresh.postNumber;
      } else {
        const loaded = await fetchPosts([...stream.slice(-chunk + 1), fresh.postNumber]);
        scrollTarget.current = fresh.postNumber;
        setPosts(loaded);
      }
      maxRead.current = Math.max(maxRead.current, fresh.postNumber);
    };
    window.addEventListener(postSavedEvent, onSaved);
    return () => window.removeEventListener(postSavedEvent, onSaved);
  }, [topic.id, fetchPosts, lastIndex, stream]);

  // what others do in the topic shows up while it is open
  useMessageBus(`/topic/${topic.id}`, async (message) => {
    const data = message.data as { type: string; postId?: number; postNumber?: number; userId?: number };
    const number = data.postNumber ?? 0;
    const loaded = posts.some((p) => p.postNumber === number);
    switch (data.type) {
      case "created": {
        if (stream.includes(number)) {
          return;
        }
        const atEnd = lastIndex === stream.length - 1;
        setStream((s) => (s.includes(number) ? s : [...s, number]));
        if (!atEnd) {
          setTopic((tp) => ({ ...tp, highestPostNumber: Math.max(tp.highestPostNumber, number), postsCount: tp.highestPostNumber >= number ? tp.postsCount : tp.postsCount + 1 }));
          return;
        }
        const [fresh] = await fetchPosts([number]);
        if (fresh) {
          setTopic((tp) => countNewPost(tp, fresh));
          setPosts((list) => appendPost(list, fresh));
        }
        return;
      }
      case "revised":
      case "rebaked":
      case "liked":
      case "poll":
      case "recovered":
      case "deleted": {
        if (!loaded) {
          return;
        }
        const [fresh] = await fetchPosts([number]);
        if (fresh) {
          setPosts((list) => list.map((p) => (p.postNumber === number ? fresh : p)));
        } else {
          setPosts((list) => list.filter((p) => p.postNumber !== number));
          setStream((s) => s.filter((n) => n !== number));
        }
        return;
      }
      case "status":
      case "moved":
        router.refresh();
        return;
    }
  });

  // selecting text in a post offers to quote it
  useEffect(() => {
    const onMouseUp = () => {
      setTimeout(() => {
        const selection = window.getSelection();
        const text = selection?.toString().trim();
        if (!selection || !text || selection.rangeCount === 0) {
          setQuote(null);
          return;
        }
        const range = selection.getRangeAt(0);
        const postEl = (range.commonAncestorContainer instanceof Element ? range.commonAncestorContainer : range.commonAncestorContainer.parentElement)?.closest<HTMLElement>(".topic-post");
        if (!postEl || !postEl.querySelector(".cooked")?.contains(range.commonAncestorContainer)) {
          setQuote(null);
          return;
        }
        const postNumber = Number(postEl.dataset.postNumber);
        const post = posts.find((p) => p.postNumber === postNumber);
        const rect = range.getBoundingClientRect();
        setQuote({ text, postNumber, username: users[post?.userId ?? 0]?.username ?? "", x: rect.left + window.scrollX, y: rect.top + window.scrollY - 44 });
      }, 0);
    };
    document.addEventListener("mouseup", onMouseUp);
    return () => document.removeEventListener("mouseup", onMouseUp);
  }, [posts, users]);

  const quoteSelection = () => {
    if (!quote) {
      return;
    }
    if (!currentUser) {
      login();
      return;
    }
    const markdown = `[quote="${quote.username}, post:${quote.postNumber}, topic:${topic.id}"]\n${quote.text}\n[/quote]\n\n`;
    if (!composer || composer.topic?.id !== topic.id) {
      openComposer({ action: "reply", topic });
    }
    insert(markdown);
    setQuote(null);
    window.getSelection()?.removeAllRanges();
  };

  const replyToTopic = () => {
    if (!currentUser) {
      login();
      return;
    }
    openComposer({ action: "reply", topic });
  };

  const onPostChange = (post: PostData) => setPosts((list) => list.map((p) => (p.id === post.id ? post : p)));
  const onPostDeleted = (post: PostData) => {
    if (post.postNumber === 1) {
      router.refresh();
      return;
    }
    if (currentUser?.admin || currentUser?.moderator) {
      setPosts((list) => list.map((p) => (p.id === post.id ? { ...p, deleted: true, canDelete: false, canRecover: true } : p)));
    } else {
      setPosts((list) => list.filter((p) => p.id !== post.id));
      setStream((s) => s.filter((n) => n !== post.postNumber));
    }
  };

  const refresh = () => router.refresh();
  const toggleSelected = (post: PostData) => setSelected((list) => (list.includes(post.id) ? list.filter((id) => id !== post.id) : [...list, post.id]));
  const stopSelecting = () => {
    setSelecting(false);
    setSelected([]);
  };
  const lastPost = posts[posts.length - 1];
  const currentPost = posts.find((p) => p.postNumber === stream[current]);
  const isMessage = topic.archetype === "private_message";
  const firstPost = posts.find((p) => p.postNumber === 1);

  return (
    <div className="topic-view">
      <div id="topic-title" ref={titleRef}>
        {editingTitle ? (
          <TitleEditor
            topic={topic}
            onCancel={() => setEditingTitle(false)}
            onSaved={(next) => {
              setTopic({ ...topic, ...next });
              setEditingTitle(false);
              router.refresh();
            }}
          />
        ) : (
          <div className="title-wrapper">
            <h1 data-topic-id={topic.id}>
              <TopicStatuses topic={topic} t={t} />
              <Link href={url} className="fancy-title">
                {topic.title}
              </Link>
              {details.canEdit ? (
                <button type="button" className="edit-topic" title={t("topic.edit")} onClick={() => setEditingTitle(true)}>
                  <Icon name="pencil" />
                </button>
              ) : null}
            </h1>
            {!isMessage ? (
              <div className="topic-category">
                <CategoryBadge category={category} categories={categories} />
                <TagList tags={topic.tags} />
              </div>
            ) : (
              <div className="topic-category flex-wrap">
                <span className="text-[var(--primary-medium)]">{t("topic.participants")}:</span>
                {(details.allowedUserIds ?? []).map((id) => {
                  const user = users[id];
                  return user ? (
                    <Link key={id} href={userUrl(user.username)} className="inline-flex items-center gap-1 text-[var(--primary-high)]">
                      <Avatar user={user} size={20} />
                      {user.username}
                    </Link>
                  ) : null;
                })}
                {(details.allowedGroups ?? []).map((group) => (
                  <Link key={`g${group.id}`} href={groupUrl(group.name)} className="inline-flex items-center gap-1 text-[var(--primary-high)]">
                    <Icon name="users" />
                    {group.name}
                  </Link>
                ))}
                <InviteButton topicId={topic.id} onInvited={refresh} />
              </div>
            )}
          </div>
        )}
      </div>

      {topic.deleted ? <div className="alert alert-error">{t("topic.deletedNotice")}</div> : null}

      <div className="container posts">
        <div className="topic-area">
          <div className="posts-wrapper">
            <div ref={topSentinel} />
            {loading === "before" ? <div className="spinner" /> : null}
            <div className="post-stream">
              {posts.map((post, i) => (
                <Fragment key={post.id}>
                  <Post
                    post={post}
                    topic={topic}
                    users={users}
                    previousPostNumber={posts[i - 1]?.postNumber ?? (i === 0 ? post.postNumber - 1 : undefined)}
                    nextPost={posts[i + 1]}
                    canReply={details.canReply || !currentUser}
                    highlighted={highlighted === post.postNumber}
                    onChange={onPostChange}
                    onDeleted={onPostDeleted}
                    onAddUsers={addUsers}
                    selectable={selecting}
                    selected={selected.includes(post.id)}
                    onToggleSelect={toggleSelected}
                  />
                  {post.postNumber === 1 ? <TopicMap topic={topic} details={details} users={users} readingTime={initial.readingTime} /> : null}
                </Fragment>
              ))}
            </div>
            <div ref={bottomSentinel} />
            {loading === "after" ? <div className="spinner" /> : null}
          </div>

          {lastIndex === stream.length - 1 ? (
            <>
              {topic.closed ? (
                <div className="topic-notice">
                  <Icon name="lock" />
                  {t("topic.closedBy")}
                </div>
              ) : null}
              {topic.archived ? (
                <div className="topic-notice">
                  <Icon name="box-archive" />
                  {t("topic.archivedNotice")}
                </div>
              ) : null}
              <div id="topic-footer-buttons">
                <div className="topic-footer-main-buttons">
                  <TopicAdminMenu topic={topic} details={details} onChange={refresh} onSelectPosts={() => setSelecting(true)} />
                  <button
                    type="button"
                    className="btn btn-default share"
                    title={t("topic.share")}
                    onClick={async () => {
                      try {
                        await navigator.clipboard.writeText(`${window.location.origin}${url}`);
                        toast.success(t("topic.linkCopied"));
                      } catch {
                        window.prompt(t("share.title"), `${window.location.origin}${url}`);
                      }
                    }}
                  >
                    <Icon name="link" />
                    {t("topic.share")}
                  </button>
                  {currentUser && firstPost ? (
                    <button
                      type="button"
                      className="btn btn-default bookmark"
                      onClick={async () => {
                        try {
                          await apiPost("bookmark-post", { id: firstPost.id, undo: firstPost.bookmarked ? "true" : undefined });
                          onPostChange({ ...firstPost, bookmarked: !firstPost.bookmarked });
                        } catch (err) {
                          toast.error(errorMessage(err));
                        }
                      }}
                    >
                      <Icon name={firstPost.bookmarked ? "bookmark" : "far-bookmark"} />
                      {t("topic.bookmark")}
                    </button>
                  ) : null}
                  {(details.canReply || !currentUser) && !topic.deleted ? (
                    <button type="button" className="btn btn-primary create" title={t("topic.replyToTopic")} onClick={replyToTopic}>
                      <Icon name="reply" />
                      {t("topic.reply")}
                    </button>
                  ) : null}
                </div>
                {currentUser ? <NotificationsButton topic={topic} showReason onChange={(level) => setTopic({ ...topic, notificationLevel: level })} /> : null}
              </div>
              {!currentUser ? (
                <div className="signup-cta">
                  <h3>{t("topic.signupCtaTitle")}</h3>
                  <p className="m-0 text-[var(--primary-high)]">{t("topic.signupCta")}</p>
                  <div className="buttons">
                    <button type="button" className="btn btn-primary" onClick={() => login(true)}>
                      {t("header.signup")}
                    </button>
                    <button type="button" className="btn btn-default" onClick={() => login(false)}>
                      {t("header.login")}
                    </button>
                  </div>
                </div>
              ) : null}
              <SuggestedTopics view={initial} isMessage={isMessage} categoryName={category?.name} categoryHref={category ? categoryUrl(category, categories) : undefined} />
            </>
          ) : null}
        </div>

        <Timeline
          total={stream.length}
          current={current}
          currentDate={currentPost?.createdAt ?? lastPost?.createdAt}
          firstDate={topic.createdAt}
          lastDate={topic.lastPostedAt}
          onJump={jumpTo}
        >
          {currentUser ? (
            <>
              {details.canReply ? (
                <button type="button" className="btn btn-primary create" onClick={replyToTopic}>
                  <Icon name="reply" />
                  {t("topic.reply")}
                </button>
              ) : null}
              <TopicAdminMenu topic={topic} details={details} onChange={refresh} onSelectPosts={() => setSelecting(true)} />
            </>
          ) : null}
        </Timeline>
      </div>

      {selecting ? (
        <div className="selected-posts">
          <p className="m-0">{t("move.selectedCount", { count: selected.length })}</p>
          <button
            type="button"
            className="btn btn-default btn-small"
            onClick={async () => {
              // every post of the topic, loaded or not
              const all = await fetchPosts(stream.slice(0, 100));
              setSelected(all.filter((p) => p.postType === 1).map((p) => p.id));
            }}
          >
            {t("move.selectAll")}
          </button>
          <button type="button" className="btn btn-default btn-small" onClick={() => setSelected([])}>
            {t("move.deselectAll")}
          </button>
          <button type="button" className="btn btn-primary btn-small" disabled={selected.length === 0} onClick={() => setMoving(true)}>
            <Icon name="arrow-right" />
            {t("move.moveTo")}
          </button>
          <button type="button" className="btn btn-flat btn-small" onClick={stopSelecting}>
            {t("common.cancel")}
          </button>
        </div>
      ) : null}
      {moving ? (
        <MovePostsModal
          topic={topic}
          postIds={selected}
          onClose={() => setMoving(false)}
          onMoved={(destination) => {
            setMoving(false);
            stopSelecting();
            router.push(topicUrl(destination));
            router.refresh();
          }}
        />
      ) : null}
      {quote ? (
        <button type="button" className="btn btn-primary btn-small fixed-quote" style={{ position: "absolute", left: quote.x, top: quote.y, zIndex: 900 }} onMouseDown={(e) => e.preventDefault()} onClick={quoteSelection}>
          <Icon name="quote-left" />
          {t("topic.quote")}
        </button>
      ) : null}
    </div>
  );
}

function TitleEditor({ topic, onCancel, onSaved }: { topic: TopicViewData["topic"]; onCancel: () => void; onSaved: (topic: Partial<TopicViewData["topic"]>) => void }) {
  const { t, settings } = useSite();
  const [title, setTitle] = useState(topic.title);
  const [categoryId, setCategoryId] = useState(topic.categoryId);
  const [tags, setTags] = useState<string[]>(topic.tags ?? []);
  const [busy, setBusy] = useState(false);

  const save = async () => {
    setBusy(true);
    try {
      const res = await apiPost<TopicViewData["topic"]>("update-topic", undefined, { id: topic.id, title, categoryId, tags });
      onSaved(res.data);
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="edit-title-form">
      <input className="d-input text-[1.3195em]" value={title} autoFocus onChange={(e) => setTitle(e.target.value)} onKeyDown={(e) => e.key === "Enter" && save()} />
      {topic.archetype !== "private_message" ? (
        <div className="row">
          <div className="min-w-[16em]">
            <CategoryChooser value={categoryId} onChange={setCategoryId} placeholder={t("composer.chooseCategory")} />
          </div>
          {settings.tagging_enabled !== "false" ? (
            <div className="min-w-[20em] flex-1">
              <TagChooser values={tags} onChange={setTags} placeholder={t("composer.tags")} max={Number(settings.max_tags_per_topic || 5)} />
            </div>
          ) : null}
        </div>
      ) : null}
      <div className="row">
        <button type="button" className="btn btn-primary" disabled={busy} onClick={save}>
          <Icon name="check" />
        </button>
        <button type="button" className="btn btn-default" onClick={onCancel}>
          <Icon name="xmark" />
        </button>
      </div>
    </div>
  );
}

function InviteButton({ topicId, onInvited }: { topicId: number; onInvited: () => void }) {
  const { t } = useSite();
  const invite = async () => {
    const username = window.prompt(t("topic.inviteUser"));
    if (!username) {
      return;
    }
    try {
      await apiPost("invite-to-message", { id: topicId, username: username.replace(/^@/, "") });
      onInvited();
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };
  return (
    <button type="button" className="btn btn-default btn-small" onClick={invite}>
      <Icon name="user-plus" />
      {t("topic.invite")}
    </button>
  );
}

function SuggestedTopics({ view, isMessage, categoryName, categoryHref }: { view: TopicViewData; isMessage: boolean; categoryName?: string; categoryHref?: string }) {
  const { t, currentUser } = useSite();
  if (!view.suggested || view.suggested.length === 0) {
    return null;
  }
  const hasUnread = view.suggested.some((s) => s.unseen || s.unreadPosts > 0);
  const heading = isMessage ? t("topic.related") : currentUser && hasUnread ? t("topic.newAndUnread") : t("topic.suggested");
  return (
    <div className="more-topics__container">
      <h3 role="heading" id="suggested-topics-title">
        {heading}
      </h3>
      <TopicList
        initial={{ topics: view.suggested, users: view.suggestedUsers ?? {}, more: false, page: 0 }}
        query={{ filter: "suggested" }}
        showPosters={false}
        footer={null}
      />
      {!isMessage ? (
        <p className="more-topics__browse-more">
          {categoryName && categoryHref ? (
            <>
              {t("topic.wantToRead", { category: "\u0000", latest: "\u0001" })
                .split(/(\u0000|\u0001)/)
                .map((part, i) =>
                  part === "\u0000" ? (
                    <Link key={i} href={categoryHref}>
                      {categoryName}
                    </Link>
                  ) : part === "\u0001" ? (
                    <Link key={i} href="/latest">
                      {t("list.viewLatest")}
                    </Link>
                  ) : (
                    <Fragment key={i}>{part}</Fragment>
                  ),
                )}
            </>
          ) : null}
        </p>
      ) : null}
    </div>
  );
}

// countNewPost adds a post that just came in to the topic's numbers, once however many times it is reported.
function countNewPost(topic: TopicViewData["topic"], post: PostData): TopicViewData["topic"] {
  if (topic.highestPostNumber >= post.postNumber) {
    return topic;
  }
  return {
    ...topic,
    postsCount: topic.postsCount + 1,
    replyCount: topic.replyCount + 1,
    highestPostNumber: post.postNumber,
    lastPostedAt: post.createdAt,
  };
}

function appendPost(list: PostData[], post: PostData): PostData[] {
  if (list.some((p) => p.id === post.id)) {
    return list;
  }
  const updated = list.map((p) => (p.postNumber === post.replyToPostNumber ? { ...p, replyCount: p.replyCount + 1 } : p));
  return [...updated, post];
}
