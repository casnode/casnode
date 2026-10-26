import Link from "next/link";
import { Avatar } from "@/components/avatar";
import { CategoryBadge } from "@/components/category-badge";
import { Icon } from "@/components/icon";
import { formatNumber, relativeAge } from "@/lib/format";
import { getSite } from "@/lib/server-api";
import { getLocale, getT } from "@/lib/server-i18n";
import type { UserMap } from "@/lib/types";
import { topicUrl, userUrl } from "@/lib/utils";
import { loadSummary, loadUser } from "../user-data";

function UserCounts({ title, items, users, icon }: { title: string; items: { userId: number; count: number }[]; users: UserMap; icon: string }) {
  if (items.length === 0) {
    return null;
  }
  return (
    <div>
      <h3>{title}</h3>
      <div className="summary-users">
        {items.map((item) => {
          const user = users[item.userId];
          if (!user) {
            return null;
          }
          return (
            <div key={item.userId} className="user-info">
              <Link href={userUrl(user.username)}>
                <Avatar user={user} size={45} />
              </Link>
              <div>
                <Link href={userUrl(user.username)} className="font-bold text-[var(--primary)]">
                  {user.username}
                </Link>
                <div className="count">
                  <Icon name={icon} />
                  {item.count}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export async function SummaryPage({ username }: { username: string }) {
  const user = await loadUser(username);
  const summary = await loadSummary(username);
  const site = await getSite();
  const t = await getT();
  const locale = await getLocale();
  if (!user || !summary) {
    return null;
  }
  const stats: [number, string][] = [
    [user.daysVisited ?? 0, t("user.daysVisited")],
    [user.topicsEntered ?? 0, t("user.topicsViewed")],
    [user.postsReadCount ?? 0, t("user.postsRead")],
    [user.likesGiven ?? 0, `${t("user.likesGiven")} ♥`],
    [user.likesReceived ?? 0, `${t("user.likesReceived")} ♥`],
    [user.topicCount ?? 0, t("user.topicsCreated")],
    [user.postCount ?? 0, t("user.postsCreated")],
  ];
  return (
    <div className="user-summary-page">
      <section className="stats-section">
        <h3>{t("user.stats")}</h3>
        <ul>
          {stats.map(([value, label]) => (
            <li key={label}>
              <span className="value">{formatNumber(value)}</span>
              <span className="label">{label}</span>
            </li>
          ))}
        </ul>
      </section>
      <div className="top-section">
        <div>
          <h3>{t("user.topReplies")}</h3>
          {summary.topReplies.length === 0 ? <p className="text-[var(--primary-medium)]">{t("user.noReplies")}</p> : null}
          <ul className="summary-list">
            {summary.topReplies.map(({ post, topic }) =>
              topic ? (
                <li key={post.id}>
                  <span className="date">{relativeAge(post.createdAt, locale)}</span>
                  <span className="like-count">
                    <Icon name="heart" />
                    {post.likeCount}
                  </span>
                  <Link href={topicUrl(topic, post.postNumber)} className="topic-link">
                    {topic.title}
                  </Link>
                </li>
              ) : null,
            )}
          </ul>
        </div>
        <div>
          <h3>{t("user.topTopics")}</h3>
          {summary.topTopics.length === 0 ? <p className="text-[var(--primary-medium)]">{t("user.noTopics")}</p> : null}
          <ul className="summary-list">
            {summary.topTopics.map((topic) => (
              <li key={topic.id}>
                <span className="date">{relativeAge(topic.createdAt, locale)}</span>
                <span className="like-count">
                  <Icon name="heart" />
                  {topic.likeCount}
                </span>
                <Link href={topicUrl(topic)} className="topic-link">
                  {topic.title}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </div>
      <div className="top-section">
        <UserCounts title={t("user.mostLikedBy")} items={summary.mostLikedBy} users={summary.users} icon="heart" />
        <UserCounts title={t("user.mostLiked")} items={summary.mostLiked} users={summary.users} icon="heart" />
        <UserCounts title={t("user.mostRepliedTo")} items={summary.mostRepliedTo} users={summary.users} icon="reply" />
        {summary.topCategories.length > 0 ? (
          <div>
            <h3>{t("user.topCategories")}</h3>
            <table className="w-full">
              <tbody>
                {summary.topCategories.map((item) => (
                  <tr key={item.categoryId} className="border-b border-[var(--primary-low)]">
                    <td className="py-2">
                      <CategoryBadge category={site.categories.find((c) => c.id === item.categoryId)} categories={site.categories} />
                    </td>
                    <td className="py-2 text-right text-[var(--primary-medium)]">
                      {item.topicCount} {t("user.topics")} · {item.postCount} {t("user.replies")}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </div>
    </div>
  );
}
