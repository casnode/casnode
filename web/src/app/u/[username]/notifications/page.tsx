import type { Metadata } from "next";
import { NotificationsList } from "@/components/user/notifications-list";
import { getSite } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";

export const metadata: Metadata = { robots: { index: false } };

export default async function Page({ params }: { params: Promise<{ username: string }> }) {
  const username = decodeURIComponent((await params).username);
  const site = await getSite();
  const t = await getT();
  if (site.currentUser?.username.toLowerCase() !== username.toLowerCase()) {
    return <div className="alert alert-info">{t("login.required")}</div>;
  }
  return <NotificationsList />;
}
