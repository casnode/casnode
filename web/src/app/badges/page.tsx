import type { Metadata } from "next";
import { BadgesIndex } from "@/components/badge/badges";
import { serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { Badge } from "@/lib/types";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("badge.badges"), alternates: { canonical: "/badges" } };
}

export default async function Page() {
  const res = await serverGet<Badge[]>("get-badges");
  return <BadgesIndex badges={res.data ?? []} />;
}
