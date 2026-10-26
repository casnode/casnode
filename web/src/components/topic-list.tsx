"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { Avatar } from "@/components/avatar";
import { CategoryBadge } from "@/components/category-badge";
import { Icon } from "@/components/icon";
import { useMessageBus } from "@/components/message-bus";
import { useSite } from "@/components/site-provider";
import { TagList } from "@/components/tag-list";
import { apiGet } from "@/lib/api";
import { activityHeat, formatNumber, likesHeat, longDate, relativeAge, viewsHeat } from "@/lib/format";
import type { Translate } from "@/lib/i18n";
import type { Topic, TopicList as TopicListData, UserMap } from "@/lib/types";
import { isPinned, topicLinkUrl, topicUrl } from "@/lib/utils";

export function TopicStatuses({ topic, t }: { topic: Topic; t: Translate }) {
  return (
    <span className="topic-statuses">
      {topic.archetype === "private_message" ? (
        <span className="topic-status" title="message">
          <Icon name="envelope" />
        </span>
      ) : null}
      {topic.closed && !topic.archived ? (
        <span className="topic-status closed" title={t("list.closed")}>
          <Icon name="lock" />
        </span>
      ) : null}
      {topic.archived ? (
        <span className="topic-status archived" title={t("list.archived")}>
          <Icon name="box-archive" />
        </span>
      ) : null}
      {isPinned(topic) ? (
        <span className="topic-status pinned" title={t("list.pinned")}>
          <Icon name="thumbtack" />
        </span>
      ) : null}
      {!topic.visible ? (
        <span className="topic-status unlisted" title={t("list.unlisted")}>
          <Icon name="eye-slash" />
        </span>
      ) : null}
    </span>
  );
}

function Posters({ topic, users, t }: { topic: Topic; users: UserMap; t: Translate }) {
  const ids = topic.posterIds ?? [topic.userId];
  return (
    <td className="posters">
      {ids.slice(0, 5).map((id, i) => {
        const user = users[id];
        if (!user) {
          return null;
        }
        const latest = i === ids.length - 1 && ids.length > 1;
        const role = i === 0 ? t("list.originalPoster") : latest ? t("list.mostRecentPoster") : t("list.frequentPoster");
        return (
          <a key={`${id}-${i}`} href={`/u/${encodeURIComponent(user.username)}`} className={latest ? "latest" : i === 0 ? "op" : ""} data-user-card={user.username}>
            <Avatar user={user} size={24} title={`${user.username} - ${role}`} />
          </a>
        );
      })}
    </td>
  );
}

function TopicListItem({ topic, users, showPosters, showCategory, showExcerpt }: { topic: Topic; users: UserMap; showPosters: boolean; showCategory: boolean; showExcerpt: boolean }) {
  const { t, locale, categories } = useSite();
  const category = categories.find((c) => c.id === topic.categoryId);
  const visited = topic.lastReadPostNumber > 0 && topic.lastReadPostNumber >= topic.highestPostNumber;
  const pinned = isPinned(topic);
  const lastPostUrl = topicUrl(topic, topic.highestPostNumber);
  const activityTitle = `${t("list.firstPost")}: ${longDate(topic.createdAt, locale)}\n${t("list.lastPost")}: ${longDate(topic.bumpedAt, locale)}`;

  return (
    <tr className={`topic-list-item${visited ? " visited" : ""}${pinned ? " pinned" : ""}${topic.unseen ? " unseen-topic" : ""}${topic.unreadPosts ? " unread-posts" : ""}`} data-topic-id={topic.id}>
      <td className="main-link">
        <span className="link-top-line">
          <TopicStatuses topic={topic} t={t} />
          <Link href={topicLinkUrl(topic)} className="title raw-link raw-topic-link" data-topic-id={topic.id}>
            {topic.title}
          </Link>
          {topic.unreadPosts > 0 ? (
            <Link href={topicLinkUrl(topic)} className="badge-notification unread-posts" title={t("list.unreadPosts", { count: topic.unreadPosts })}>
              {topic.unreadPosts}
            </Link>
          ) : topic.unseen ? (
            <span className="badge-notification new-topic" title={t("list.newTopic")} />
          ) : null}
        </span>
        <div className="link-bottom-line">
          {showCategory ? <CategoryBadge category={category} categories={categories} /> : null}
          <TagList tags={topic.tags} />
        </div>
        {showExcerpt && pinned && topic.excerpt ? (
          <div className="topic-excerpt">
            {topic.excerpt}{" "}
            <Link href={topicUrl(topic)} className="read-more">
              {t("list.readMore")}
            </Link>
          </div>
        ) : null}
      </td>
      {showPosters ? <Posters topic={topic} users={users} t={t} /> : null}
      <td className={`num posts-map posts ${likesHeat(topic.likeCount, topic.postsCount)}`}>
        <Link href={lastPostUrl} className="badge-posts">
          {formatNumber(topic.replyCount)}
        </Link>
      </td>
      <td className={`num views ${viewsHeat(topic.views)}`}>
        <span className="number" title={String(topic.views)}>
          {formatNumber(topic.views)}
        </span>
      </td>
      <td className={`num activity ${activityHeat(topic.createdAt, topic.bumpedAt)}`} title={activityTitle}>
        <Link href={lastPostUrl} className="post-activity">
          <span className="relative-date" suppressHydrationWarning>
            {relativeAge(topic.bumpedAt, locale)}
          </span>
        </Link>
      </td>
    </tr>
  );
}

export interface TopicListQuery {
  filter: string;
  categoryId?: number;
  tag?: string;
  period?: string;
  order?: string;
  ascending?: boolean;
  username?: string;
}

// The table of topics; more pages load while scrolling down, like Discourse's endless list.
export function TopicList({
  initial,
  query,
  showPosters = true,
  showCategory = true,
  emptyMessage,
  footer,
}: {
  initial: TopicListData;
  query: TopicListQuery;
  showPosters?: boolean;
  showCategory?: boolean;
  emptyMessage?: string;
  footer?: React.ReactNode;
}) {
  const { t, categories } = useSite();
  const router = useRouter();
  const [incoming, setIncoming] = useState<number[]>([]);
  const [topics, setTopics] = useState(initial.topics ?? []);
  const [users, setUsers] = useState(initial.users ?? {});
  const [page, setPage] = useState(initial.page ?? 0);
  const [more, setMore] = useState(initial.more);
  const [loading, setLoading] = useState(false);
  const [prevInitial, setPrevInitial] = useState(initial);
  const sentinel = useRef<HTMLDivElement>(null);

  if (prevInitial !== initial) {
    setPrevInitial(initial);
    setIncoming([]);
    setTopics(initial.topics ?? []);
    setUsers(initial.users ?? {});
    setPage(initial.page ?? 0);
    setMore(initial.more);
  }

  // new topics and replies while the list is open, offered like Discourse's "See 3 new or updated topics"
  const live = ["latest", "new", "unread", "hot"].includes(query.filter) && !query.username && !query.order;
  useMessageBus(live ? "/latest" : null, (message) => {
    const data = message.data as { type: string; topicId: number; categoryId: number; archetype: string; tags: string[] | null };
    if (data.archetype === "private_message" || (query.filter === "new" && data.type !== "created")) {
      return;
    }
    if (query.categoryId) {
      const category = categories.find((c) => c.id === query.categoryId);
      if (data.categoryId !== query.categoryId && !(category?.subcategoryIds ?? []).includes(data.categoryId)) {
        return;
      }
    }
    if (query.tag && !(data.tags ?? []).includes(query.tag)) {
      return;
    }
    // a reply to the topic already first in the list changes nothing worth showing
    if (data.type === "bumped" && topics[0]?.id === data.topicId) {
      return;
    }
    setIncoming((list) => (list.includes(data.topicId) ? list : [...list, data.topicId]));
  });

  useEffect(() => {
    const el = sentinel.current;
    if (!el || !more) {
      return;
    }
    const observer = new IntersectionObserver(
      async (entries) => {
        if (!entries[0].isIntersecting || loading) {
          return;
        }
        setLoading(true);
        try {
          const res = await apiGet<TopicListData>("get-topics", {
            filter: query.filter,
            categoryId: query.categoryId,
            tag: query.tag,
            period: query.period,
            order: query.order,
            ascending: query.ascending ? "true" : undefined,
            username: query.username,
            page: page + 1,
          });
          setTopics((list) => {
            const seen = new Set(list.map((x) => x.id));
            return [...list, ...res.data.topics.filter((x) => !seen.has(x.id))];
          });
          setUsers((u) => ({ ...u, ...res.data.users }));
          setPage(page + 1);
          setMore(res.data.more);
        } catch {
          setMore(false);
        } finally {
          setLoading(false);
        }
      },
      { rootMargin: "400px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [more, page, loading, query]);

  const banner =
    incoming.length > 0 ? (
      <div className="show-more">
        <button
          type="button"
          className="alert alert-info clickable show-more-topics"
          onClick={() => {
            setIncoming([]);
            window.scrollTo({ top: 0 });
            router.refresh();
          }}
        >
          {t("list.newTopicsAvailable", { count: incoming.length })}
        </button>
      </div>
    ) : null;

  if (topics.length === 0) {
    return (
      <>
        {banner}
        <div className="topic-list-bottom">
          <div className="footer-message">
            <h3>{emptyMessage ?? t("list.noTopicsYet")}</h3>
            {footer}
          </div>
        </div>
      </>
    );
  }

  const showExcerpt = query.filter === "latest" && !query.order;

  return (
    <>
      {banner}
      <table className="topic-list">
        <thead className="topic-list-header">
          <tr>
            <th className="default" scope="col">
              {t("list.topic")}
            </th>
            {showPosters ? <th className="posters" scope="col" /> : null}
            <th className="num posts" scope="col">
              {t("list.replies")}
            </th>
            <th className="num views" scope="col">
              {t("list.views")}
            </th>
            <th className="num activity" scope="col">
              {t("list.activity")}
            </th>
          </tr>
        </thead>
        <tbody className="topic-list-body">
          {topics.map((topic) => (
            <TopicListItem key={topic.id} topic={topic} users={users} showPosters={showPosters} showCategory={showCategory} showExcerpt={showExcerpt} />
          ))}
        </tbody>
      </table>
      <div ref={sentinel} />
      {loading ? <div className="spinner" /> : null}
      {!more && footer !== null ? (
        <div className="topic-list-bottom">
          <div className="footer-message">
            <h3>{t("list.noTopics")}</h3>
            {footer}
          </div>
        </div>
      ) : null}
    </>
  );
}
