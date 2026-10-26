import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { TopicList } from "@/components/topic-list";
import { serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { TopicList as TopicListData } from "@/lib/types";
import { loadGroup } from "../group-data";

export const metadata: Metadata = { robots: { index: false } };

export default async function Page({ params }: { params: Promise<{ name: string }> }) {
  const group = await loadGroup(decodeURIComponent((await params).name));
  if (!group) {
    notFound();
  }
  const t = await getT();
  const res = await serverGet<TopicListData>("get-group-messages", { name: group.name });
  if (res.status !== "ok") {
    return <div className="alert alert-info">{res.msg}</div>;
  }
  return <TopicList initial={{ ...res.data, more: false }} query={{ filter: "private_messages_group" }} showCategory={false} emptyMessage={t("group.noMessages")} footer={null} />;
}
