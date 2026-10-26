import type { Metadata } from "next";
import { Directory } from "@/components/directory";
import { serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { User } from "@/lib/types";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("users.title"), alternates: { canonical: "/u" } };
}

export interface DirectoryItem {
  user: User;
  likesReceived: number;
  likesGiven: number;
  topicCount: number;
  postCount: number;
  topicsEntered: number;
  postsRead: number;
  daysVisited: number;
}

export default async function Page() {
  const t = await getT();
  const res = await serverGet<DirectoryItem[], boolean>("get-directory", { order: "likes_received" });
  return (
    <div className="directory">
      <h1 className="page-title">{t("users.title")}</h1>
      <Directory initial={res.data ?? []} initialMore={!!res.data2} />
    </div>
  );
}
