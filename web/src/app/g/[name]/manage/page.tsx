import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { GroupManage } from "@/components/group/group";
import { getSite } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import { loadGroup } from "../group-data";

export const metadata: Metadata = { robots: { index: false } };

export default async function Page({ params }: { params: Promise<{ name: string }> }) {
  const group = await loadGroup(decodeURIComponent((await params).name));
  if (!group) {
    notFound();
  }
  const t = await getT();
  const site = await getSite();
  if (!group.canManage && !(group.automatic && site.currentUser?.admin)) {
    return <div className="alert alert-info">{t("login.required")}</div>;
  }
  return <GroupManage group={group} />;
}
