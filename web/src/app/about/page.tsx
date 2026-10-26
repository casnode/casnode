import type { Metadata } from "next";
import Link from "next/link";
import { Avatar } from "@/components/avatar";
import { Cooked } from "@/components/cooked";
import { formatNumber } from "@/lib/format";
import { getSite, serverGet } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";
import type { User } from "@/lib/types";
import { userUrl } from "@/lib/utils";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("about.title"), alternates: { canonical: "/about" } };
}

interface About {
  aboutHtml: string;
  stats: Record<string, number>;
  admins: User[];
  moderators: User[];
}

function UserGrid({ users }: { users: User[] }) {
  return (
    <div className="about-users">
      {users.map((user) => (
        <Link key={user.id} href={userUrl(user.username)} className="user-info text-[var(--primary)]">
          <Avatar user={user} size={45} />
          <span>
            <span className="block font-bold">{user.username}</span>
            {user.name && user.name !== user.username ? <span className="text-[var(--primary-medium)] text-[0.8706em]">{user.name}</span> : null}
          </span>
        </Link>
      ))}
    </div>
  );
}

export default async function Page() {
  const t = await getT();
  const site = await getSite();
  const res = await serverGet<About>("get-about");
  const about = res.data;
  const title = site.settings.title || "Casnode";
  return (
    <div className="about-page">
      <h1 className="page-title">
        {t("about.title")} {title}
      </h1>
      <section>
        {about?.aboutHtml ? <Cooked html={about.aboutHtml} className="p-0" /> : <p>{site.settings.site_description}</p>}
        <p className="flex gap-4">
          <Link href="/faq">{t("about.guidelines")}</Link>
          {site.settings.tos_text ? <Link href="/tos">{t("about.tos")}</Link> : null}
          {site.settings.privacy_text ? <Link href="/privacy">{t("about.privacy")}</Link> : null}
        </p>
      </section>
      {about?.admins?.length ? (
        <section>
          <h2>{t("about.admins")}</h2>
          <UserGrid users={about.admins} />
        </section>
      ) : null}
      {about?.moderators?.length ? (
        <section>
          <h2>{t("about.moderators")}</h2>
          <UserGrid users={about.moderators} />
        </section>
      ) : null}
      {about ? (
        <section>
          <h2>{t("about.stats")}</h2>
          <table className="about-stats">
            <thead>
              <tr>
                <th />
                <th>{t("about.allTime")}</th>
                <th>{t("about.last7")}</th>
                <th>{t("about.last30")}</th>
              </tr>
            </thead>
            <tbody>
              {(["topics", "posts", "users"] as const).map((key) => (
                <tr key={key}>
                  <td>{t(key === "topics" ? "about.topics" : key === "posts" ? "about.posts" : "about.users")}</td>
                  <td>{formatNumber(about.stats[key])}</td>
                  <td>{formatNumber(about.stats[`${key}7`])}</td>
                  <td>{formatNumber(about.stats[`${key}30`])}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}
      {site.settings.contact_email ? (
        <section>
          <h2>{t("about.contact")}</h2>
          <a href={`mailto:${site.settings.contact_email}`}>{site.settings.contact_email}</a>
        </section>
      ) : null}
    </div>
  );
}
