import Link from "next/link";
import { Avatar } from "@/components/avatar";
import { CategoryBadge } from "@/components/category-badge";
import { Icon } from "@/components/icon";
import { NavigationBar } from "@/components/navigation-bar";
import { formatNumber, relativeAge } from "@/lib/format";
import { getSite, serverGet } from "@/lib/server-api";
import { getLocale, getT } from "@/lib/server-i18n";
import type { Category, TopicList } from "@/lib/types";
import { categoryUrl, topicLinkUrl, topicUrl } from "@/lib/utils";

// /categories: the categories with their activity, beside the latest topics (Discourse's
// "categories_and_latest_topics"), or alone, or as boxes, as desktop_category_page_style says.
export async function CategoriesPage() {
  const site = await getSite();
  const t = await getT();
  const locale = await getLocale();
  const res = await serverGet<Category[], TopicList>("get-categories");
  const categories = res.status === "ok" ? res.data : [];
  const latest = res.status === "ok" ? res.data2 : null;
  const style = site.settings.desktop_category_page_style || "categories_and_latest_topics";
  const all = site.categories;
  const uncategorizedId = Number(site.settings.uncategorized_category_id || 0);
  const top = categories.filter((c) => c.parentId === 0 && !(c.id === uncategorizedId && c.topicCount === 0));

  const categoryTable = (
    <table className="category-list">
      <thead>
        <tr>
          <th className="category">{t("categories.category")}</th>
          <th className="topics">{t("categories.topics")}</th>
        </tr>
      </thead>
      <tbody>
        {top.map((category) => {
          const subs = categories.filter((c) => c.parentId === category.id);
          const recent = category.topicsWeek > 0 ? t("categories.perWeek", { count: category.topicsWeek }) : category.topicsMonth > 0 ? t("categories.perMonth", { count: category.topicsMonth }) : "";
          return (
            <tr key={category.id} data-category-id={category.id}>
              <td className="category" style={{ ["--category-badge-color" as string]: `#${category.color}` }}>
                <h3 className="m-0">
                  <Link href={categoryUrl(category, all)} className="category-name-link">
                    {category.readRestricted ? <Icon name="lock" className="text-[var(--primary-medium)] text-[0.6em]" /> : null}
                    {category.name}
                  </Link>
                </h3>
                {category.description ? <div className="category-description">{category.description}</div> : null}
                {subs.length > 0 ? (
                  <div className="subcategories">
                    {subs.map((sub) => (
                      <CategoryBadge key={sub.id} category={sub} categories={all} hideParent />
                    ))}
                  </div>
                ) : null}
              </td>
              <td className="topics">
                {recent ? (
                  <div title={`${category.topicCount}`}>
                    <span className="value">{recent.split(" ")[0]}</span> <span className="unit">{recent.split(" ").slice(1).join(" ")}</span>
                  </div>
                ) : (
                  <span className="value">{formatNumber(category.topicCount)}</span>
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );

  return (
    <div className="list-container">
      <h1 className="sr-only">{t("nav.categories")}</h1>
      <NavigationBar filter="categories" />
      {categories.length === 0 ? <p>{t("categories.empty")}</p> : null}
      {style === "categories_boxes" ? (
        <div className="category-boxes">
          {top.map((category) => (
            <Link key={category.id} href={categoryUrl(category, all)} className="category-box" style={{ ["--category-badge-color" as string]: `#${category.color}` }}>
              <span className="category-box-heading">
                <span className="inline-block h-[0.75em] w-[0.75em] rounded-[2px]" style={{ background: `#${category.color}` }} />
                {category.name}
              </span>
              {category.description ? <span className="description">{category.description}</span> : null}
            </Link>
          ))}
        </div>
      ) : style === "categories_only" || !latest ? (
        categoryTable
      ) : (
        <div className="categories-and-latest">
          <div className="column categories">{categoryTable}</div>
          <div className="column latest-topic-list">
            <div className="table-heading">{t("categories.latest")}</div>
            {latest.topics.map((topic) => {
              const poster = latest.users[topic.lastPostUserId] ?? latest.users[topic.userId];
              const visited = topic.lastReadPostNumber > 0 && topic.lastReadPostNumber >= topic.highestPostNumber;
              return (
                <div key={topic.id} className={`latest-topic-list-item${visited ? " visited" : ""}`}>
                  <div className="topic-poster">
                    {poster ? (
                      <Link href={`/u/${encodeURIComponent(poster.username)}`}>
                        <Avatar user={poster} size={45} />
                      </Link>
                    ) : null}
                  </div>
                  <div className="main-link">
                    <div className="top-row">
                      <Link href={topicLinkUrl(topic)} className="title">
                        {topic.title}
                      </Link>
                      {topic.unreadPosts > 0 ? <span className="badge-notification unread-posts">{topic.unreadPosts}</span> : topic.unseen ? <span className="badge-notification new-topic" /> : null}
                    </div>
                    <div className="bottom-row">
                      <CategoryBadge category={all.find((c) => c.id === topic.categoryId)} categories={all} />
                    </div>
                  </div>
                  <div className="topic-stats">
                    <div className="posts">
                      <Link href={topicUrl(topic, topic.highestPostNumber)} className="text-[var(--primary-medium)]">
                        {formatNumber(topic.replyCount)}
                      </Link>
                    </div>
                    <div className="activity" suppressHydrationWarning>
                      {relativeAge(topic.bumpedAt, locale)}
                    </div>
                  </div>
                </div>
              );
            })}
            <Link href="/latest" className="more-topics-link btn btn-default">
              {t("categories.moreLatest")}
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
