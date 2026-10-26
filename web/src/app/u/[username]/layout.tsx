import Link from "next/link";
import { notFound } from "next/navigation";
import { Avatar } from "@/components/avatar";
import { Cooked } from "@/components/cooked";
import { Icon } from "@/components/icon";
import { MessageButton } from "@/components/user/message-button";
import { UserNav } from "@/components/user/user-nav";
import { longAgo, longDate } from "@/lib/format";
import { getSite } from "@/lib/server-api";
import { getLocale, getT } from "@/lib/server-i18n";
import { isValidDate, userUrl } from "@/lib/utils";
import { loadUser } from "./user-data";

// The top of every profile page: avatar, names, about me, and the tabs.
export default async function UserLayout({ children, params }: { children: React.ReactNode; params: Promise<{ username: string }> }) {
  const { username } = await params;
  const user = await loadUser(decodeURIComponent(username));
  if (!user) {
    notFound();
  }
  const site = await getSite();
  const t = await getT();
  const locale = await getLocale();
  const self = site.currentUser?.id === user.id;
  const suspended = isValidDate(user.suspendedTill) && new Date(user.suspendedTill!) > new Date();

  return (
    <div className="user-main">
      <section className="user-profile-header">
        <div className="user-profile-avatar">
          <Avatar user={user} size={120} />
        </div>
        <div className="primary-textual">
          <h1 className="username">
            {user.username}
            {user.admin || user.moderator ? <Icon name="shield-halved" title={user.admin ? t("user.admin") : t("user.moderator")} /> : null}
          </h1>
          {user.name && user.name !== user.username ? <h2 className="full-name">{user.name}</h2> : null}
          {user.title ? <h3 className="user-title">{user.title}</h3> : null}
          {user.location || user.website ? (
            <div className="location-and-website">
              {user.location ? (
                <span>
                  <Icon name="location-dot" />
                  {user.location}
                </span>
              ) : null}
              {user.website ? (
                <a href={/^https?:\/\//.test(user.website) ? user.website : `https://${user.website}`} rel="nofollow ugc noopener" target="_blank">
                  <Icon name="globe" />
                  {user.website.replace(/^https?:\/\//, "")}
                </a>
              ) : null}
            </div>
          ) : null}
          {suspended ? <div className="alert alert-error mt-2">{t("user.suspended", { date: longDate(user.suspendedTill, locale) })}</div> : null}
          {user.bioCooked ? (
            <div className="bio">
              <Cooked html={user.bioCooked} />
            </div>
          ) : null}
        </div>
        <div className="profile-controls">
          {site.currentUser && !self ? <MessageButton username={user.username} /> : null}
          {self ? (
            <Link href={userUrl(user.username, "preferences")} className="btn btn-default">
              <Icon name="gear" />
              {t("user.preferences")}
            </Link>
          ) : null}
        </div>
      </section>
      <dl className="user-profile-secondary">
        <div>
          <dt>{t("user.joined")}</dt>
          <dd title={longDate(user.createdAt, locale)}>{longAgo(user.createdAt, locale)}</dd>
        </div>
        {isValidDate(user.lastPostedAt) ? (
          <div>
            <dt>{t("user.lastPost")}</dt>
            <dd title={longDate(user.lastPostedAt, locale)}>{longAgo(user.lastPostedAt, locale)}</dd>
          </div>
        ) : null}
        {isValidDate(user.lastSeenAt) ? (
          <div>
            <dt>{t("user.seen")}</dt>
            <dd title={longDate(user.lastSeenAt, locale)}>{longAgo(user.lastSeenAt, locale)}</dd>
          </div>
        ) : null}
        <div>
          <dt>{t("user.trustLevel")}</dt>
          <dd>{user.trustLevel}</dd>
        </div>
        {(self || site.currentUser?.admin) && user.email ? (
          <div>
            <dt>{t("user.email")}</dt>
            <dd>{user.email}</dd>
          </div>
        ) : null}
      </dl>
      <UserNav username={user.username} />
      <section className="user-content">{children}</section>
    </div>
  );
}
