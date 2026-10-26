import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { cache } from "react";
import { BadgeDetail } from "@/components/badge/badges";
import { badgeDescription, badgeName } from "@/lib/badges";
import { serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { Badge, BadgeGrant } from "@/lib/types";
import { badgeUrl } from "@/lib/utils";

type Props = { params: Promise<{ id: string }>; searchParams: Promise<{ username?: string }> };

const loadBadge = cache(async (id: string, username?: string) => serverGet<Badge, { grants: BadgeGrant[]; more: boolean }>("get-badge", { id, username }));

export async function generateMetadata({ params, searchParams }: Props): Promise<Metadata> {
  const res = await loadBadge((await params).id, (await searchParams).username);
  if (res.status !== "ok") {
    return {};
  }
  const t = await getT();
  return { title: badgeName(res.data, t), description: badgeDescription(res.data, t), alternates: { canonical: badgeUrl(res.data) } };
}

export default async function Page({ params, searchParams }: Props) {
  const { username } = await searchParams;
  const res = await loadBadge((await params).id, username);
  if (res.status !== "ok") {
    notFound();
  }
  return <BadgeDetail badge={res.data} initial={res.data2.grants ?? []} initialMore={res.data2.more} username={username} />;
}
