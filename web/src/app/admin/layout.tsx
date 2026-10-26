import type { Metadata } from "next";
import { AdminNav } from "@/components/admin/admin-nav";
import { getSite } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";

export const metadata: Metadata = { title: "Admin", robots: { index: false } };

export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  const site = await getSite();
  const t = await getT();
  const user = site.currentUser;
  if (!user || (!user.admin && !user.moderator)) {
    return <div className="alert alert-info">{t("login.required")}</div>;
  }
  return (
    <div className="admin-contents">
      <AdminNav />
      {children}
    </div>
  );
}
