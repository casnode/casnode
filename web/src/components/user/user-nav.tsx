"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import type { MessageKey } from "@/lib/i18n";
import { userUrl } from "@/lib/utils";

export function UserNav({ username }: { username: string }) {
  const { t, currentUser } = useSite();
  const pathname = usePathname();
  const self = currentUser?.username.toLowerCase() === username.toLowerCase();
  const staff = !!currentUser && (currentUser.admin || currentUser.moderator);
  const base = userUrl(username);
  const tabs: { href: string; key: MessageKey; icon: string; show: boolean; match: (p: string) => boolean }[] = [
    { href: `${base}/summary`, key: "user.summary", icon: "user", show: true, match: (p) => p === base || p.startsWith(`${base}/summary`) },
    { href: `${base}/activity`, key: "user.activity", icon: "list", show: true, match: (p) => p.startsWith(`${base}/activity`) },
    { href: `${base}/notifications`, key: "user.notifications", icon: "bell", show: self, match: (p) => p.startsWith(`${base}/notifications`) },
    { href: `${base}/messages`, key: "user.messages", icon: "envelope", show: self || !!currentUser?.admin, match: (p) => p.startsWith(`${base}/messages`) },
    { href: `${base}/preferences`, key: "user.preferences", icon: "gear", show: self || staff, match: (p) => p.startsWith(`${base}/preferences`) },
  ];
  const decoded = decodeURIComponent(pathname);
  return (
    <nav className="user-navigation">
      <ul className="user-navigation-primary">
        {tabs
          .filter((tab) => tab.show)
          .map((tab) => (
            <li key={tab.href} className="user-nav__item">
              <Link href={tab.href} className={tab.match(pathname) || tab.match(decoded) ? "active" : ""}>
                <Icon name={tab.icon} />
                {t(tab.key)}
              </Link>
            </li>
          ))}
      </ul>
    </nav>
  );
}

export function ActivityNav({ username }: { username: string }) {
  const { t, currentUser } = useSite();
  const pathname = decodeURIComponent(usePathname());
  const self = currentUser?.username.toLowerCase() === username.toLowerCase();
  const base = `${userUrl(username)}/activity`;
  const items: { href: string; key: MessageKey; show: boolean }[] = [
    { href: base, key: "user.all", show: true },
    { href: `${base}/topics`, key: "user.topics", show: true },
    { href: `${base}/replies`, key: "user.replies", show: true },
    { href: `${base}/likes-given`, key: "user.likes", show: true },
    { href: `${base}/bookmarks`, key: "user.bookmarks", show: self },
  ];
  return (
    <div className="user-secondary-navigation">
      {items
        .filter((i) => i.show)
        .map((item) => (
          <Link key={item.href} href={item.href} className={`btn btn-small ${decodeURIComponent(item.href) === pathname ? "btn-primary" : "btn-flat"}`}>
            {t(item.key)}
          </Link>
        ))}
    </div>
  );
}
