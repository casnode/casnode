import type { Metadata } from "next";
import { ReviewQueue } from "@/components/review-queue";
import { getSite } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";

export const metadata: Metadata = { title: "Review", robots: { index: false } };

export default async function Page() {
  const site = await getSite();
  const t = await getT();
  const user = site.currentUser;
  if (!user || (!user.admin && !user.moderator)) {
    return <div className="alert alert-info">{t("login.required")}</div>;
  }
  return (
    <div>
      <h1 className="page-title">{t("review.title")}</h1>
      <ReviewQueue />
    </div>
  );
}
