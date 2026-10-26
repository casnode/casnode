import type { Metadata } from "next";
import Link from "next/link";
import { Avatar } from "@/components/avatar";
import { CategoryBadge } from "@/components/category-badge";
import { Icon } from "@/components/icon";
import { TagList } from "@/components/tag-list";
import { relativeAge } from "@/lib/format";
import { getSite, serverGet } from "@/lib/server-api";
import { getLocale, getT } from "@/lib/server-i18n";
import type { SearchResult } from "@/lib/types";
import { topicUrl, userUrl } from "@/lib/utils";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("search.title"), robots: { index: false } };
}

function highlight(text: string, q: string) {
  const terms = q
    .split(/\s+/)
    .filter((w) => w && !/^[@#]|:/.test(w))
    .map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
  if (terms.length === 0) {
    return text;
  }
  const parts = text.split(new RegExp(`(${terms.join("|")})`, "ig"));
  return parts.map((part, i) => (i % 2 === 1 ? <span key={i} className="search-highlight">{part}</span> : part));
}

export default async function Page({ searchParams }: { searchParams: Promise<{ q?: string; page?: string }> }) {
  const { q = "", page = "0" } = await searchParams;
  const t = await getT();
  const locale = await getLocale();
  const site = await getSite();
  const pageNumber = Math.max(0, Number(page) || 0);
  const res = q.trim() ? await serverGet<SearchResult>("search", { q, page: pageNumber }) : null;
  const result = res?.status === "ok" ? res.data : null;

  return (
    <div className="search-page">
      <h1 className="page-title">{t("search.title")}</h1>
      <form className="search-bar" action="/search" method="get">
        <input className="d-input" name="q" defaultValue={q} placeholder={t("search.placeholder")} autoFocus />
        <button type="submit" className="btn btn-primary">
          <Icon name="search" />
          {t("search.button")}
        </button>
      </form>
      <p className="text-[var(--primary-medium)] text-[0.8706em] mb-4">{t("search.tips")}</p>
      {result ? (
        <>
          <div className="result-count">
            {result.posts.length === 0 ? t("search.noResults") : t("search.results", { count: `${result.posts.length}${result.more ? "+" : ""}`, term: `"${q}"` })}
          </div>
          {result.matchUsers.length > 0 && pageNumber === 0 ? (
            <div className="mb-4 flex flex-wrap gap-4">
              {result.matchUsers.map((user) => (
                <Link key={user.id} href={userUrl(user.username)} className="flex items-center gap-2 text-[var(--primary)]">
                  <Avatar user={user} size={25} />
                  {user.username}
                </Link>
              ))}
            </div>
          ) : null}
          {result.posts.map((post) => {
            const topic = result.topics[post.topicId];
            const user = result.users[post.userId];
            if (!topic) {
              return null;
            }
            return (
              <div key={post.id} className="fps-result">
                <div className="author">
                  <Link href={user ? userUrl(user.username) : "#"}>
                    <Avatar user={user} size={45} />
                  </Link>
                </div>
                <div className="topic">
                  <Link href={topicUrl(topic, post.postNumber)} className="search-link">
                    <span className="topic-title">{highlight(topic.title, q)}</span>
                  </Link>
                  <div className="search-category flex flex-wrap items-center gap-2">
                    <CategoryBadge category={site.categories.find((c) => c.id === topic.categoryId)} categories={site.categories} />
                    <TagList tags={topic.tags} />
                  </div>
                  <div className="blurb">
                    <span className="date">{relativeAge(post.createdAt, locale)} - </span>
                    {highlight(post.blurb, q)}
                  </div>
                  {post.likeCount > 0 ? (
                    <div className="mt-1 text-[var(--primary-medium)] text-[0.8706em]">
                      <Icon name="heart" /> {post.likeCount}
                    </div>
                  ) : null}
                </div>
              </div>
            );
          })}
          <div className="mt-4 flex gap-2">
            {pageNumber > 0 ? (
              <Link href={`/search?q=${encodeURIComponent(q)}&page=${pageNumber - 1}`} className="btn btn-default">
                {t("common.previous")}
              </Link>
            ) : null}
            {result.more ? (
              <Link href={`/search?q=${encodeURIComponent(q)}&page=${pageNumber + 1}`} className="btn btn-default">
                {t("common.next")}
              </Link>
            ) : null}
          </div>
        </>
      ) : res ? (
        <div className="alert alert-error">{res.msg}</div>
      ) : null}
    </div>
  );
}
