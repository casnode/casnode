import Link from "next/link";
import { formatNumber } from "@/lib/format";
import { serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";

export default async function Page() {
  const t = await getT();
  const res = await serverGet<Record<string, number>>("get-dashboard");
  if (res.status !== "ok") {
    return <div className="alert alert-error">{res.msg}</div>;
  }
  const d = res.data;
  const stat = (label: string, key: string) => (
    <div className="dashboard-stat">
      <div className="label">{label}</div>
      <div className="value">{formatNumber(d[key])}</div>
      <div className="sub">
        <span>
          {t("adminPage.today")}: {formatNumber(d[`${key}Today`])}
        </span>
        <span>
          {t("adminPage.week")}: {formatNumber(d[`${key}Week`])}
        </span>
        <span>
          {t("adminPage.month")}: {formatNumber(d[`${key}Month`])}
        </span>
      </div>
    </div>
  );
  return (
    <div>
      <h1 className="page-title">{t("adminPage.dashboard")}</h1>
      {d.reviewables > 0 ? (
        <div className="alert alert-info">
          <Link href="/review">{t("adminPage.pendingReview", { count: d.reviewables })}</Link>
        </div>
      ) : null}
      <div className="dashboard-stats">
        {stat(t("adminPage.topics"), "topics")}
        {stat(t("adminPage.posts"), "posts")}
        <div className="dashboard-stat">
          <div className="label">{t("adminPage.signups")}</div>
          <div className="value">{formatNumber(d.users)}</div>
        </div>
      </div>
    </div>
  );
}
