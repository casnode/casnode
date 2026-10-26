import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { firstParam, ListPage, parsePage } from "@/components/list-page";
import { tagUrl } from "@/lib/utils";

const filters = ["latest", "new", "unread", "hot", "top"];

type Props = { params: Promise<{ parts: string[] }>; searchParams: Promise<Record<string, string | string[] | undefined>> };

function parse(parts: string[]) {
  const tag = decodeURIComponent(parts[0] ?? "");
  const filter = parts[1] === "l" ? (parts[2] ?? "latest") : "latest";
  return { tag, filter };
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { tag, filter } = parse((await params).parts);
  return {
    title: tag,
    alternates: { canonical: filter === "latest" ? tagUrl(tag) : `${tagUrl(tag)}/l/${filter}`, types: { "application/rss+xml": `${tagUrl(tag)}.rss` } },
  };
}

export default async function Page({ params, searchParams }: Props) {
  const { tag, filter } = parse((await params).parts);
  if (!tag || !filters.includes(filter)) {
    notFound();
  }
  const query = await searchParams;
  return <ListPage filter={filter} tag={tag} page={parsePage(query.page)} period={firstParam(query.period)} />;
}
