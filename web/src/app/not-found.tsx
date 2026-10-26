import Link from "next/link";
import { CategoryBadge } from "@/components/category-badge";
import { Icon } from "@/components/icon";
import { getSite, serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { TopicList } from "@/lib/types";
import { topicUrl } from "@/lib/utils";

// Discourse's 404: an apology, a search box, and the popular and recent topics to go on with.
export default async function NotFound() {
  const t = await getT();
  const site = await getSite();
  const [top, latest] = await Promise.all([
    serverGet<TopicList>("get-topics", { filter: "top", period: "monthly" }),
    serverGet<TopicList>("get-topics", { filter: "latest", order: "activity" }),
  ]);
  const column = (title: string, list?: TopicList | null) => (
    <div>
      <h2>{title}</h2>
      {(list?.topics ?? []).slice(0, 8).map((topic) => (
        <div key={topic.id} className="not-found-topic">
          <Link href={topicUrl(topic)}>{topic.title}</Link>
          <div>
            <CategoryBadge category={site.categories.find((c) => c.id === topic.categoryId)} categories={site.categories} />
          </div>
        </div>
      ))}
    </div>
  );
  return (
    <div className="page-not-found">
      <h1>{t("notFound.title")}</h1>
      <Link href="/" className="btn btn-primary">
        <Icon name="house" />
        {t("notFound.home")}
      </Link>
      <div className="page-not-found-search">
        <h2>{t("notFound.search")}</h2>
        <form action="/search" method="get">
          <input className="d-input" name="q" placeholder={t("search.placeholder")} />
          <button type="submit" className="btn btn-primary">
            {t("search.button")}
          </button>
        </form>
      </div>
      <div className="page-not-found-topics">
        {column(t("notFound.popular"), top.status === "ok" ? top.data : null)}
        {column(t("notFound.recent"), latest.status === "ok" ? latest.data : null)}
      </div>
    </div>
  );
}
