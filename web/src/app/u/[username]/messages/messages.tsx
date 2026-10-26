import Link from "next/link";
import { TopicList } from "@/components/topic-list";
import { MessageButton } from "@/components/user/message-button";
import { getSite, serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { TopicList as TopicListData } from "@/lib/types";
import { userUrl } from "@/lib/utils";

export async function MessagesPage({ username, box }: { username: string; box: "inbox" | "sent" }) {
  const site = await getSite();
  const t = await getT();
  const filter = box === "sent" ? "private_messages_sent" : "private_messages";
  const res = await serverGet<TopicListData>("get-topics", { filter, username });
  if (res.status !== "ok") {
    return <div className="alert alert-error">{res.msg}</div>;
  }
  const base = userUrl(username, "messages");
  return (
    <div>
      <div className="user-secondary-navigation">
        <Link href={base} className={`btn btn-small ${box === "inbox" ? "btn-primary" : "btn-flat"}`}>
          {t("sidebar.inbox")}
        </Link>
        <Link href={`${base}/sent`} className={`btn btn-small ${box === "sent" ? "btn-primary" : "btn-flat"}`}>
          {t("sidebar.sent")}
        </Link>
        <span className="ml-auto">{site.currentUser?.username === username ? <MessageButton /> : null}</span>
      </div>
      <TopicList initial={res.data} query={{ filter, username }} showCategory={false} emptyMessage={t("user.noMessages")} footer={null} />
    </div>
  );
}
