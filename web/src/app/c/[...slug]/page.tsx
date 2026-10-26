import type { Metadata } from "next";
import { notFound, permanentRedirect } from "next/navigation";
import { CategoryForm } from "@/components/category-form";
import { firstParam, ListPage, parsePage } from "@/components/list-page";
import { getSite } from "@/lib/server-api";
import type { Category } from "@/lib/types";
import { categoryUrl } from "@/lib/utils";

const filters = ["latest", "new", "unread", "hot", "top"];

// /c/<slug>/<id>, /c/<parent>/<slug>/<id>, then /l/<filter> or /edit
function parse(segments: string[], categories: Category[]) {
  const parts = segments.map((s) => decodeURIComponent(s));
  let filter = "latest";
  let edit = false;
  const lIndex = parts.indexOf("l");
  if (lIndex >= 0) {
    filter = parts[lIndex + 1] ?? "latest";
    parts.splice(lIndex);
  }
  if (parts[parts.length - 1] === "edit") {
    edit = true;
    parts.pop();
  }
  let category: Category | undefined;
  const last = parts[parts.length - 1];
  if (last && /^\d+$/.test(last)) {
    category = categories.find((c) => c.id === Number(last));
  } else if (last) {
    category = categories.find((c) => c.slug.toLowerCase() === last.toLowerCase());
  }
  return { category, filter, edit, hasId: /^\d+$/.test(last ?? "") };
}

type Props = { params: Promise<{ slug: string[] }>; searchParams: Promise<Record<string, string | string[] | undefined>> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const site = await getSite();
  const { category, filter } = parse((await params).slug, site.categories);
  if (!category) {
    return {};
  }
  const url = categoryUrl(category, site.categories);
  return {
    title: category.name,
    description: category.description || undefined,
    alternates: { canonical: filter === "latest" ? url : `${url}/l/${filter}`, types: { "application/rss+xml": `${url}.rss` } },
    openGraph: { title: category.name, description: category.description || undefined },
  };
}

export default async function Page({ params, searchParams }: Props) {
  const site = await getSite();
  const { category, filter, edit, hasId } = parse((await params).slug, site.categories);
  if (!category || !filters.includes(filter)) {
    notFound();
  }
  if (!hasId && !edit) {
    permanentRedirect(categoryUrl(category, site.categories));
  }
  if (edit) {
    return <CategoryForm category={category} />;
  }
  const query = await searchParams;
  return <ListPage filter={filter} category={category} page={parsePage(query.page)} period={firstParam(query.period)} order={firstParam(query.order)} />;
}
