"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import type { MessageKey } from "@/lib/i18n";

const items: { href: string; key: MessageKey; icon: string; adminOnly?: boolean }[] = [
  { href: "/admin", key: "adminPage.dashboard", icon: "gauge" },
  { href: "/admin/site_settings", key: "adminPage.settings", icon: "gear", adminOnly: true },
  { href: "/admin/users", key: "adminPage.users", icon: "users" },
  { href: "/categories", key: "adminPage.categories", icon: "layer-group", adminOnly: true },
  { href: "/tags", key: "adminPage.tags", icon: "tag" },
  { href: "/g", key: "adminPage.groups", icon: "user-group" },
  { href: "/admin/badges", key: "adminPage.badges", icon: "certificate", adminOnly: true },
  { href: "/admin/email", key: "adminPage.email", icon: "envelope", adminOnly: true },
  { href: "/admin/logins", key: "adminPage.logins", icon: "key", adminOnly: true },
];

export function AdminNav() {
  const { t, currentUser } = useSite();
  const pathname = usePathname();
  return (
    <nav className="admin-nav">
      {items
        .filter((item) => !item.adminOnly || currentUser?.admin)
        .map((item) => (
          <Link key={item.href} href={item.href} className={`btn ${pathname === item.href ? "btn-primary" : "btn-flat"}`}>
            <Icon name={item.icon} />
            {t(item.key)}
          </Link>
        ))}
    </nav>
  );
}
