import type { Metadata } from "next";
import { PreferencesForm } from "@/components/user/preferences-form";
import { getSite } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import { loadUser } from "../user-data";

export const metadata: Metadata = { robots: { index: false } };

export default async function Page({ params }: { params: Promise<{ username: string }> }) {
  const username = decodeURIComponent((await params).username);
  const site = await getSite();
  const t = await getT();
  const user = await loadUser(username);
  const me = site.currentUser;
  if (!user || !me || (me.id !== user.id && !me.admin && !me.moderator)) {
    return <div className="alert alert-info">{t("login.required")}</div>;
  }
  const accountUrl = site.casdoor.serverUrl ? `${site.casdoor.serverUrl}/account` : "";
  return <PreferencesForm user={user} accountUrl={accountUrl} />;
}
