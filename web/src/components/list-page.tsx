import Link from "next/link";
import { notFound } from "next/navigation";
import { NavigationBar } from "@/components/navigation-bar";
import { TopicList } from "@/components/topic-list";
import { getSite, serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { Category, TopicList as TopicListData } from "@/lib/types";
import { categoryUrl } from "@/lib/utils";

export interface ListPageProps {
  filter: string;
  category?: Category;
  tag?: string;
  period?: string;
  page?: number;
  order?: string;
  ascending?: boolean;
}

// The pages of topic lists: latest, new, top..., in a category or with a tag. The first page is rendered here for
// search engines and the rest loads while scrolling; ?page=N renders any page, with links to the next one.
export async function ListPage({ filter, category, tag, period, page = 0, order, ascending }: ListPageProps) {
  const site = await getSite();
  const t = await getT();
  if ((filter === "new" || filter === "unread") && !site.currentUser) {
    return (
      <div className="contents-wrap">
        <NavigationBar filter={filter} category={category} tag={tag} />
        <div className="alert alert-info">{t("login.required")}</div>
      </div>
    );
  }

  const res = await serverGet<TopicListData>("get-topics", {
    filter,
    categoryId: category?.id,
    tag,
    period,
    page,
    order,
    ascending: ascending ? "true" : undefined,
  });
  if (res.status !== "ok") {
    notFound();
  }
  const list = res.data;
  if (page > 0 && list.topics.length === 0) {
    notFound();
  }
  const empty = filter === "new" ? t("list.noNewTopics") : filter === "unread" ? t("list.noUnreadTopics") : undefined;
  const base = category ? categoryUrl(category, site.categories) : "";

  return (
    <div className="list-container">
      {category ? (
        <div className="category-heading sr-only">
          <h1>{category.name}</h1>
          <p>{category.description}</p>
        </div>
      ) : null}
      {tag ? <h1 className="sr-only">{tag}</h1> : null}
      <NavigationBar filter={filter} category={category} tag={tag} period={period} />
      <TopicList
        initial={list}
        query={{ filter, categoryId: category?.id, tag, period, order, ascending }}
        showCategory={!category || (category.subcategoryIds?.length ?? 0) > 0}
        emptyMessage={empty}
        footer={
          <p>
            {filter !== "latest" ? <Link href={base ? `${base}/l/latest` : "/latest"}>{t("list.viewLatest")}</Link> : null}
            {filter !== "latest" ? " · " : null}
            <Link href="/categories">{t("list.browseCategories")}</Link>
          </p>
        }
      />
      <nav className="sr-only" aria-label="pagination">
        {page > 0 ? (
          <a href={`?page=${page - 1}`} rel="prev">
            {t("common.previous")}
          </a>
        ) : null}
        {list.more ? (
          <a href={`?page=${page + 1}`} rel="next">
            {t("common.next")}
          </a>
        ) : null}
      </nav>
    </div>
  );
}

export function parsePage(value?: string | string[]): number {
  const n = Number(Array.isArray(value) ? value[0] : value);
  return Number.isFinite(n) && n > 0 ? Math.floor(n) : 0;
}

export function firstParam(value?: string | string[]): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}
