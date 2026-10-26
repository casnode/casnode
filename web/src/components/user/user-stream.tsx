"use client";

import Link from "next/link";
import { useState } from "react";
import { Avatar } from "@/components/avatar";
import { CategoryBadge } from "@/components/category-badge";
import { Cooked } from "@/components/cooked";
import { useSite } from "@/components/site-provider";
import { apiGet } from "@/lib/api";
import { longDate, relativeAge } from "@/lib/format";
import type { UserMap, UserPost } from "@/lib/types";
import { topicUrl, userUrl } from "@/lib/utils";

export function UserStream({ username, filter, initial, initialUsers, initialMore, empty }: { username: string; filter: string; initial: UserPost[]; initialUsers: UserMap; initialMore: boolean; empty: string }) {
  const { locale, categories, t } = useSite();
  const [items, setItems] = useState(initial);
  const [users, setUsers] = useState(initialUsers);
  const [more, setMore] = useState(initialMore);
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(false);

  const loadMore = async () => {
    setLoading(true);
    try {
      const res = await apiGet<{ posts: UserPost[]; users: UserMap; more: boolean }>("get-user-activity", { username, filter, page: page + 1 });
      setItems((list) => [...list, ...res.data.posts]);
      setUsers((u) => ({ ...u, ...res.data.users }));
      setMore(res.data.more);
      setPage(page + 1);
    } finally {
      setLoading(false);
    }
  };

  if (items.length === 0) {
    return <p className="text-[var(--primary-medium)]">{empty}</p>;
  }

  return (
    <div className="user-stream">
      {items.map(({ post, topic }) => {
        if (!topic) {
          return null;
        }
        const author = users[post.userId];
        return (
          <div key={post.id} className="user-stream-item">
            <Link href={author ? userUrl(author.username) : "#"}>
              <Avatar user={author} size={45} />
            </Link>
            <div className="stream-content">
              <div className="item-header">
                <Link href={topicUrl(topic, post.postNumber)} className="title">
                  {topic.title}
                </Link>
                <span className="time" title={longDate(post.createdAt, locale)} suppressHydrationWarning>
                  {relativeAge(post.createdAt, locale)}
                </span>
              </div>
              <div className="category">
                <CategoryBadge category={categories.find((c) => c.id === topic.categoryId)} categories={categories} />
              </div>
              <Cooked html={post.cooked} className="excerpt" />
            </div>
          </div>
        );
      })}
      {more ? (
        <div className="py-4">
          <button type="button" className="btn btn-default" disabled={loading} onClick={loadMore}>
            {t("common.loadMore")}
          </button>
        </div>
      ) : null}
    </div>
  );
}
