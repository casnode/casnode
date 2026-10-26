import type { Metadata } from "next";
import Link from "next/link";
import { serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { Tag } from "@/lib/types";
import { tagUrl } from "@/lib/utils";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("tags.title"), alternates: { canonical: "/tags" } };
}

export default async function Page({ searchParams }: { searchParams: Promise<{ sort?: string }> }) {
  const t = await getT();
  const { sort } = await searchParams;
  const res = await serverGet<Tag[]>("get-tags");
  const tags = [...(res.data ?? [])];
  if (sort === "name") {
    tags.sort((a, b) => a.name.localeCompare(b.name));
  }
  return (
    <div className="tags-index">
      <h1 className="page-title">{t("tags.title")}</h1>
      <div className="mb-4 flex items-center gap-2 text-[var(--primary-medium)]">
        {t("tags.sortBy")}
        <Link href="/tags" className={`btn btn-small ${sort !== "name" ? "btn-primary" : "btn-flat"}`}>
          {t("tags.sortCount")}
        </Link>
        <Link href="/tags?sort=name" className={`btn btn-small ${sort === "name" ? "btn-primary" : "btn-flat"}`}>
          {t("tags.sortName")}
        </Link>
      </div>
      {tags.length === 0 ? <p>{t("tags.none")}</p> : null}
      <div className="tag-box-list">
        {tags.map((tag) => (
          <div key={tag.id} className="tag-box">
            <Link href={tagUrl(tag.name)} className="discourse-tag">
              {tag.name}
            </Link>
            <span className="tag-count">× {tag.topicCount}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
