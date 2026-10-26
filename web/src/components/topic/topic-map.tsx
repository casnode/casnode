"use client";

import Link from "next/link";
import { Avatar } from "@/components/avatar";
import { useSite } from "@/components/site-provider";
import { formatNumber } from "@/lib/format";
import type { Topic, TopicDetails, UserMap } from "@/lib/types";
import { userUrl } from "@/lib/utils";

// The strip under the first post: views, users, likes, the most active posters and the reading time.
export function TopicMap({ topic, details, users, readingTime }: { topic: Topic; details: TopicDetails; users: UserMap; readingTime: number }) {
  const { t } = useSite();
  const participants = details.participants ?? [];
  return (
    <section className="topic-map --op" aria-label="topic map">
      <div className="topic-map__stats">
        <div className="topic-map__stat topic-map__views-trigger">
          <span className="number">{formatNumber(topic.views)}</span>
          <span className="topic-map__stat-label">{t("topic.views")}</span>
        </div>
        {topic.likeCount > 0 ? (
          <div className="topic-map__stat topic-map__likes-trigger">
            <span className="number">{formatNumber(topic.likeCount)}</span>
            <span className="topic-map__stat-label">{t("topic.likes")}</span>
          </div>
        ) : null}
        {participants.length > 1 ? (
          <>
            <div className="topic-map__stat topic-map__users-trigger">
              <span className="number">{formatNumber(topic.participantCount || participants.length)}</span>
              <span className="topic-map__stat-label">{t("topic.users")}</span>
            </div>
            <div className="topic-map__users-list">
              {participants.slice(0, 10).map((p) => {
                const user = users[p.userId];
                if (!user) {
                  return null;
                }
                return (
                  <Link key={p.userId} href={userUrl(user.username)} className="poster" title={user.username}>
                    <Avatar user={user} size={32} />
                    {p.postCount > 1 ? <span className="post-count">{p.postCount}</span> : null}
                  </Link>
                );
              })}
            </div>
          </>
        ) : null}
        <div className="topic-map__stat topic-map__read-time read-time">
          <span className="number">{readingTime} min</span>
          <span className="topic-map__stat-label">{t("topic.read")}</span>
        </div>
      </div>
    </section>
  );
}
