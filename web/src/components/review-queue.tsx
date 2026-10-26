"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { Avatar } from "@/components/avatar";
import { CategoryBadge } from "@/components/category-badge";
import { Cooked } from "@/components/cooked";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import { relativeAge } from "@/lib/format";
import type { MessageKey } from "@/lib/i18n";
import { PostAction, type Reviewable, type UserMap } from "@/lib/types";
import { topicUrl, userUrl } from "@/lib/utils";

const statuses: { value: number; label: MessageKey }[] = [
  { value: 0, label: "review.pending" },
  { value: 1, label: "review.approved" },
  { value: 2, label: "review.rejected" },
  { value: 3, label: "review.ignored" },
  { value: 4, label: "review.deleted" },
];

const flagLabels: Record<number, MessageKey> = {
  [PostAction.OffTopic]: "flag.offTopic",
  [PostAction.Inappropriate]: "flag.inappropriate",
  [PostAction.Spam]: "flag.spam",
  [PostAction.NotifyModerators]: "flag.other",
};

// The review queue: flagged posts waiting for staff to agree (hide), disagree, delete or ignore.
export function ReviewQueue() {
  const { t, locale, categories, refreshSite } = useSite();
  const [status, setStatus] = useState(0);
  const [items, setItems] = useState<Reviewable[] | null>(null);
  const [users, setUsers] = useState<UserMap>({});

  const load = useCallback(async () => {
    const res = await apiGet<{ reviewables: Reviewable[]; users: UserMap }>("get-reviewables", { status });
    setItems(res.data.reviewables);
    setUsers(res.data.users);
  }, [status]);

  useEffect(() => {
    let cancelled = false;
    apiGet<{ reviewables: Reviewable[]; users: UserMap }>("get-reviewables", { status })
      .then((res) => {
        if (!cancelled) {
          setItems(res.data.reviewables);
          setUsers(res.data.users);
        }
      })
      .catch(() => !cancelled && setItems([]));
    return () => {
      cancelled = true;
    };
  }, [status]);

  const perform = async (id: number, action: string) => {
    try {
      await apiPost("perform-reviewable", { id, action });
      await load();
      refreshSite().catch(() => undefined);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  return (
    <div>
      <div className="user-secondary-navigation">
        {statuses.map((s) => (
          <button key={s.value} type="button" className={`btn btn-small ${status === s.value ? "btn-primary" : "btn-flat"}`} onClick={() => setStatus(s.value)}>
            {t(s.label)}
          </button>
        ))}
      </div>
      {items === null ? <div className="spinner" /> : null}
      {items?.length === 0 ? <p className="text-[var(--primary-medium)]">{t("review.empty")}</p> : null}
      {items?.map((item) => {
        const author = item.post ? users[item.post.userId] : undefined;
        return (
          <div key={item.id} className="reviewable-item">
            <div className="reviewable-meta">
              {item.topic ? (
                <Link href={topicUrl(item.topic, item.post?.postNumber)} className="font-bold">
                  {item.topic.title}
                </Link>
              ) : null}
              {item.topic ? <CategoryBadge category={categories.find((c) => c.id === item.topic!.categoryId)} categories={categories} /> : null}
              <span className="ml-auto text-[var(--primary-medium)]">{relativeAge(item.createdAt, locale)}</span>
            </div>
            <div className="post-body">
              {author ? (
                <Link href={userUrl(author.username)} className="flex items-center gap-2 font-bold text-[var(--primary-high)]">
                  <Avatar user={author} size={25} />
                  {author.username}
                </Link>
              ) : null}
              {item.post ? <Cooked html={item.post.cooked} /> : null}
            </div>
            <div className="reviewable-scores">
              {t("review.flaggedBy")}:
              <ul className="m-0 mt-1 list-none p-0">
                {item.scores.map((score, i) => (
                  <li key={i} className="flex flex-wrap items-center gap-2 py-0.5">
                    <Avatar user={users[score.userId]} size={20} />
                    <strong>{users[score.userId]?.username}</strong>
                    <span>{flagLabels[score.actionType] ? t(flagLabels[score.actionType]) : score.actionType}</span>
                    {score.message ? <span className="text-[var(--primary-medium)]">“{score.message}”</span> : null}
                  </li>
                ))}
              </ul>
            </div>
            {item.status === 0 ? (
              <div className="reviewable-actions">
                <button type="button" className="btn btn-primary" onClick={() => perform(item.id, "agree")}>
                  <Icon name="check" />
                  {t("review.agree")}
                </button>
                <button type="button" className="btn btn-default" onClick={() => perform(item.id, "disagree")}>
                  <Icon name="xmark" />
                  {t("review.disagree")}
                </button>
                <button type="button" className="btn btn-danger" onClick={() => perform(item.id, "delete")}>
                  <Icon name="trash-can" />
                  {t("review.deletePost")}
                </button>
                <button type="button" className="btn btn-flat" onClick={() => perform(item.id, "ignore")}>
                  {t("review.ignore")}
                </button>
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
