import type { Metadata } from "next";
import { GroupsIndex } from "@/components/group/group";
import { serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { Group } from "@/lib/types";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("group.groups"), alternates: { canonical: "/g" } };
}

export default async function Page() {
  const res = await serverGet<Group[]>("get-groups");
  return <GroupsIndex initial={res.data ?? []} />;
}
