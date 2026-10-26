import { notFound } from "next/navigation";
import { TopicList } from "@/components/topic-list";
import { ActivityNav } from "@/components/user/user-nav";
import { UserStream } from "@/components/user/user-stream";
import { serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { TopicList as TopicListData, UserMap, UserPost } from "@/lib/types";

const filters: Record<string, string> = { "": "all", replies: "replies", "likes-given": "likes", bookmarks: "bookmarks" };

export async function ActivityPage({ username, filter }: { username: string; filter: string }) {
  const t = await getT();
  if (filter === "topics") {
    const res = await serverGet<TopicListData>("get-topics", { filter: "created", username, order: "created" });
    if (res.status !== "ok") {
      notFound();
    }
    return (
      <>
        <ActivityNav username={username} />
        <TopicList initial={res.data} query={{ filter: "created", username, order: "created" }} emptyMessage={t("user.noTopics")} footer={null} />
      </>
    );
  }

  const apiFilter = filters[filter];
  if (!apiFilter) {
    notFound();
  }
  const res = await serverGet<{ posts: UserPost[]; users: UserMap; more: boolean }>("get-user-activity", { username, filter: apiFilter });
  const empty = apiFilter === "likes" ? t("user.noLikes") : apiFilter === "bookmarks" ? t("user.noBookmarks") : apiFilter === "replies" ? t("user.noReplies") : t("user.noActivity");
  return (
    <>
      <ActivityNav username={username} />
      {res.status === "ok" ? (
        <UserStream username={username} filter={apiFilter} initial={res.data.posts ?? []} initialUsers={res.data.users ?? {}} initialMore={res.data.more} empty={empty} />
      ) : (
        <div className="alert alert-error">{res.msg}</div>
      )}
    </>
  );
}
