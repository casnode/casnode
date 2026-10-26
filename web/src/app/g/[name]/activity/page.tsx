import { notFound } from "next/navigation";
import { UserStream } from "@/components/user/user-stream";
import { serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { UserMap, UserPost } from "@/lib/types";
import { loadGroup } from "../group-data";

export default async function Page({ params }: { params: Promise<{ name: string }> }) {
  const group = await loadGroup(decodeURIComponent((await params).name));
  if (!group) {
    notFound();
  }
  const t = await getT();
  const res = await serverGet<{ posts: UserPost[]; users: UserMap; more: boolean }>("get-group-activity", { name: group.name });
  if (res.status !== "ok") {
    return <div className="alert alert-error">{res.msg}</div>;
  }
  return (
    <UserStream
      username=""
      filter=""
      initial={res.data.posts ?? []}
      initialUsers={res.data.users ?? {}}
      initialMore={res.data.more}
      empty={t("group.noActivity")}
      source={{ path: "get-group-activity", params: { name: group.name } }}
    />
  );
}
